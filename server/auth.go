// Auth implementation based on JWT

/*
Copyright © 2019, 2020, 2022 Red Hat, Inc.

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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/RedHatInsights/insights-operator-utils/collections"
	types "github.com/RedHatInsights/insights-results-types"
	"github.com/rs/zerolog/log"
)

const (
	// #nosec G101
	malformedTokenMessage = "Malformed authentication token"
	invalidTokenMessage   = "Invalid/Malformed auth token"
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
		token, isTokenValid := server.getAuthTokenHeader(w, r)
		if !isTokenValid {
			// everything has been handled already
			return
		}

		if server.Config.LogAuthToken {
			log.Info().Msgf("Authentication token: %s", token)
		}

		// decode auth. token to JSON string
		decoded, err := base64.StdEncoding.DecodeString(token)

		// if token is malformed return HTTP code 403 to client
		if err != nil {
			// malformed token, returns with HTTP code 403 as usual
			log.Error().Err(err).Msg(malformedTokenMessage)
			handleServerError(w, &AuthenticationError{errString: malformedTokenMessage})
			return
		}

		tk := &types.Token{}
		tkV2 := &types.TokenV2{}

		// if we took JWT token, it has different structure than x-rh-identity
		// JWT isn't/can't used in any real environment
		if server.Config.AuthType == "jwt" {
			jwtPayload := &types.JWTPayload{}
			err = json.Unmarshal(decoded, jwtPayload)
			if err != nil {
				// malformed token, returns with HTTP code 403 as usual
				log.Error().Err(err).Msg(malformedTokenMessage)
				handleServerError(w, &AuthenticationError{errString: malformedTokenMessage})
				return
			}
			// Map JWT token to inner token
			tk.Identity = types.Identity{
				AccountNumber: jwtPayload.AccountNumber,
				Internal: types.Internal{
					OrgID: jwtPayload.OrgID,
				},
				User: types.User{
					UserID: jwtPayload.UserID,
				},
			}
		} else {
			// auth type is xrh (x-rh-identity header)

			// unmarshal new token structure (org_id on top level)
			err = json.Unmarshal(decoded, tkV2)
			if err != nil {
				// malformed token, returns with HTTP code 403 as usual
				log.Error().Err(err).Msg(malformedTokenMessage)
				handleServerError(w, &AuthenticationError{errString: malformedTokenMessage})
				return
			}

			// unmarshal old token structure (org_id nested) too
			err = json.Unmarshal(decoded, tk)
			if err != nil {
				// malformed token, returns with HTTP code 403 as usual
				log.Error().Err(err).Msg(malformedTokenMessage)
				handleServerError(w, &AuthenticationError{errString: malformedTokenMessage})
				return
			}
		}

		if tkV2.IdentityV2.OrgID != 0 {
			log.Info().Msg("org_id found on top level in token structure (new format)")
			// fill in old types.Token because many places in smart-proxy and aggregator rely on it.
			tk.Identity.Internal.OrgID = tkV2.IdentityV2.OrgID
		} else {
			log.Error().Msg("org_id not found on top level in token structure (old format)")
		}

		if tk.Identity.AccountNumber == "" || tk.Identity.AccountNumber == "0" {
			log.Info().Msgf("anemic tenant found! org_id %v, user data [%+v]",
				tk.Identity.Internal.OrgID, tk.Identity.User,
			)
		}

		if tk.Identity.Internal.OrgID == 0 {
			msg := fmt.Sprintf("error retrieving requester org_id from token. account_number [%v], user data [%+v]",
				tk.Identity.AccountNumber,
				tk.Identity.User,
			)
			log.Error().Msg(msg)
			handleServerError(w, &AuthenticationError{errString: msg})
			return
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
	identity, err := server.GetAuthToken(request)
	if err != nil {
		return types.UserID(""), err
	}

	if identity.User.UserID == "" {
		log.Info().Msgf("empty userID for username [%v] email [%v]", identity.User.Username, identity.User.Email)
		return types.UserID("0"), nil
	}

	return identity.User.UserID, nil
}

// GetCurrentOrgID retrieves the ID of the organization the user belongs to
func (server *HTTPServer) GetCurrentOrgID(request *http.Request) (types.OrgID, error) {
	identity, err := server.GetAuthToken(request)
	if err != nil {
		return types.OrgID(0), err
	}

	return identity.Internal.OrgID, nil
}

// GetCurrentOrgIDUserID retrieves the ID of the organization the user belongs to and
// the ID of the specific user
func (server *HTTPServer) GetCurrentOrgIDUserID(request *http.Request) (
	types.OrgID, types.UserID, error,
) {
	identity, err := server.GetAuthToken(request)
	if err != nil {
		log.Err(err).Msg("error retrieving identity from token")
		return types.OrgID(0), types.UserID("0"), err
	}

	return identity.Internal.OrgID, identity.User.UserID, nil
}

// GetAuthToken returns current authentication token
func (server *HTTPServer) GetAuthToken(request *http.Request) (*types.Identity, error) {
	i := request.Context().Value(types.ContextKeyUser)

	if i == nil {
		return nil, &AuthenticationError{errString: "token is not provided"}
	}

	identity, ok := i.(types.Identity)
	if !ok {
		return nil, &AuthenticationError{errString: "contextKeyUser has wrong type"}
	}

	return &identity, nil
}

func (server *HTTPServer) getAuthTokenHeader(w http.ResponseWriter, r *http.Request) (string, bool) {
	var tokenHeader string
	// In case of testing on local machine we don't take x-rh-identity
	// header, but instead Authorization with JWT token in it
	if server.Config.AuthType == "jwt" {
		log.Info().Msg("Retrieving jwt token")

		// Grab the token from the header
		tokenHeader = r.Header.Get("Authorization")

		// The token normally comes in format `Bearer {token-body}`, we
		// check if the retrieved token matched this requirement
		splitted := strings.Split(tokenHeader, " ")
		if len(splitted) != 2 {
			log.Error().Msg(invalidTokenMessage)
			handleServerError(w, &AuthenticationError{errString: invalidTokenMessage})
			return "", false
		}

		// Here we take JWT token which include 3 parts, we need only
		// second one
		splitted = strings.Split(splitted[1], ".")
		if len(splitted) < 1 {
			return "", false
		}
		tokenHeader = splitted[1]
	} else {
		log.Info().Msg("Retrieving x-rh-identity token")
		// Grab the token from the header
		tokenHeader = r.Header.Get("x-rh-identity")
	}

	log.Info().Int("Length", len(tokenHeader)).Msg("Token retrieved")

	if tokenHeader == "" {
		const message = "Missing auth token"
		log.Error().Msg(message)
		handleServerError(w, &AuthenticationError{errString: message})
		return "", false
	}

	return tokenHeader, true
}
