/*
Copyright © 2019, 2020, 2022, 2023 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/RedHatInsights/insights-operator-utils/collections"
	"github.com/RedHatInsights/insights-results-smart-proxy/auth"
	"github.com/RedHatInsights/insights-results-smart-proxy/metrics"
	types "github.com/RedHatInsights/insights-results-types"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

const (
	accountType          = "Account Type"
	accountNotAuthorized = "Account does not have the required permissions"
)

// setupAuthMiddleware sets up the authentication and authorization middlewares
// for the given router.
func (server *HTTPServer) setupAuthMiddleware(router *mux.Router) {
	apiPrefix := server.Config.APIv1Prefix

	metricsURL := apiPrefix + MetricsEndpoint
	openAPIv1URL := apiPrefix + filepath.Base(server.Config.APIv1SpecFile)
	openAPIv2URL := server.Config.APIv2Prefix + filepath.Base(server.Config.APIv2SpecFile)
	infoV1URL := apiPrefix + InfoEndpoint
	infoV2URL := server.Config.APIv2Prefix + InfoEndpoint

	// Define noAuthURLs for use in authentication and authorization middleware
	noAuthURLs := []string{
		metricsURL,
		openAPIv1URL,
		openAPIv2URL,
		infoV1URL,
		infoV2URL,
		metricsURL + "?",   // to be able to test using Frisby
		openAPIv1URL + "?", // to be able to test using Frisby
		openAPIv2URL + "?", // to be able to test using Frisby
	}

	if server.Config.Auth {
		router.Use(func(next http.Handler) http.Handler {
			return server.Authentication(next, noAuthURLs)
		})
	}

	if server.Config.UseRBAC {
		router.Use(func(next http.Handler) http.Handler {
			return server.Authorization(next, noAuthURLs)
		})
	}
}

// Authentication middleware for checking auth rights
func (server *HTTPServer) Authentication(next http.Handler, noAuthURLs []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For specific URLs it is ok to not use auth. mechanisms at all
		// this is specific to OpenAPI JSON response and for all OPTION HTTP methods
		if collections.StringInSlice(r.RequestURI, noAuthURLs) || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		// Try to read auth. header from HTTP request (if provided by client)
		tk, err := auth.DecodeTokenFromHeader(w, r, server.Config.AuthType)
		if err != nil {
			handleServerError(w, err)
			return
		}

		if tk.Identity.AccountNumber == "" || tk.Identity.AccountNumber == "0" {
			log.Info().Msgf("anemic tenant found! org_id %v, user data [%+v]",
				tk.Identity.OrgID, tk.Identity.User,
			)
		}

		if tk.Identity.OrgID == 0 {
			msg := fmt.Sprintf("error retrieving requester org_id from token. account_number [%v], user data [%+v]",
				tk.Identity.AccountNumber,
				tk.Identity.User,
			)
			log.Error().Msg(msg)
			handleServerError(w, &auth.AuthenticationError{ErrString: msg})
			return
		}

		if tk.Identity.User.UserID == "" {
			tk.Identity.User.UserID = "0"
		}

		// Everything went well, proceed with the request and set the
		// caller to the user retrieved from the parsed token
		ctx := context.WithValue(r.Context(), types.ContextKeyUser, tk.Identity)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// Authorization middleware for checking permissions
func (server *HTTPServer) Authorization(next http.Handler, noAuthURLs []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// for specific URLs it is ok to not use auth. mechanisms at all
		// this is specific to OpenAPI JSON response and for all OPTION HTTP methods
		if collections.StringInSlice(r.RequestURI, noAuthURLs) || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		token, err := auth.DecodeTokenFromHeader(w, r, server.Config.AuthType)
		if err != nil {
			handleServerError(w, err)
			return
		}

		metrics.RBACIdentityType.WithLabelValues(token.Identity.Type).Inc()

		// For now we will only log authorization and only handle service accounts. This logic should
		// be for all users, but let's first make sure we won't disturb existing users by only
		// logging unauthorized service accounts
		if token.Identity.Type == "ServiceAccount" {
			log.Debug().Str("client ID", token.Identity.ServiceAccount.ClientID).Msg("Received a request from a service account")
			// Check permissions for service accounts
			if !server.rbacClient.IsAuthorized(auth.GetAuthTokenHeader(r)) {
				log.Warn().Str(accountType, token.Identity.Type).Msg(accountNotAuthorized)
				metrics.RBACServiceAccountRejected.Inc()
				if server.rbacClient.IsEnforcing() {
					handleServerError(w, &auth.AuthorizationError{ErrString: accountNotAuthorized})
					return
				}
			}
		} else {
			log.Debug().Str(accountType, token.Identity.Type).Msg("RBAC is only used with service accounts for now")
			// handleServerError(w, &auth.AuthorizationError{ErrString: "unknown identity type"})
			// We don't use return because RBAC is not mandatory for users yet
			// return
		}

		// Access is authorized, proceed with the request
		next.ServeHTTP(w, r)
	})
}

// GetCurrentUserID retrieves current user's id from request
func (server *HTTPServer) GetCurrentUserID(request *http.Request) (types.UserID, error) {
	identity, err := auth.GetAuthToken(request)
	if err != nil {
		return types.UserID(""), err
	}

	return identity.User.UserID, nil
}

// GetCurrentOrgID retrieves the ID of the organization the user belongs to
func (server *HTTPServer) GetCurrentOrgID(request *http.Request) (types.OrgID, error) {
	identity, err := auth.GetAuthToken(request)
	if err != nil {
		return types.OrgID(0), err
	}

	return identity.OrgID, nil
}

// GetCurrentOrgIDUserIDFromToken retrieves the ID of the organization the user belongs to and
// the ID of the specific user
func (server *HTTPServer) GetCurrentOrgIDUserIDFromToken(request *http.Request) (
	types.OrgID, types.UserID, error,
) {
	identity, err := auth.GetAuthToken(request)
	if err != nil {
		log.Err(err).Msg("error retrieving identity from token")
		return types.OrgID(0), types.UserID("0"), err
	}

	return identity.OrgID, identity.User.UserID, nil
}

// SetRBACClient sets the server’s RBAC client.
func (server *HTTPServer) SetRBACClient(client auth.RBACClient) {
	server.rbacClient = client
}
