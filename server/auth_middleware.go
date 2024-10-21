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

	"github.com/RedHatInsights/insights-operator-utils/collections"
	"github.com/RedHatInsights/insights-results-smart-proxy/auth"
	types "github.com/RedHatInsights/insights-results-types"
	"github.com/rs/zerolog/log"
)

// Authentication middleware for checking auth rights
func (server *HTTPServer) Authentication(next http.Handler, noAuthURLs []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// for specific URLs it is ok to not use auth. mechanisms at all
		// this is specific to OpenAPI JSON response and for all OPTION HTTP methods
		if collections.StringInSlice(r.RequestURI, noAuthURLs) || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// try to read auth. header from HTTP request (if provided by client)
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
