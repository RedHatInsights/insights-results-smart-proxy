/*
Copyright © 2024 Red Hat, Inc.

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

// Package auth handles authentication by decoding the Base64-encoded
// 'x-rh-identity' token from HTTP requests into a user identity object.
package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"

	types "github.com/RedHatInsights/insights-results-types"
	"github.com/rs/zerolog/log"
)

const (
	// XRHAuthTokenHeader represents the name of the header used in XRH authorization type
	// #nosec G101
	XRHAuthTokenHeader = "x-rh-identity"
	// invalidTokenMessage is the message returned when the authentication token is malformed.
	// #nosec G101
	invalidTokenMessage = "Malformed authentication token"
	// missingTokenMessage is the message returned when the authentication token is missing.
	// #nosec G101
	missingTokenMessage = "Missing auth token"
)

// DecodeTokenFromHeader decodes the authentication token from the HTTP request header.
// It returns a pointer to a Token and an error if the token is missing or malformed.
func DecodeTokenFromHeader(_ http.ResponseWriter, r *http.Request, authType string) (*types.Token, error) {
	// Try to read the authentication header from the HTTP request (if provided by the client).
	token := GetAuthTokenHeader(r)
	if token == "" {
		return nil, &AuthenticationError{ErrString: missingTokenMessage}
	}

	// Decode the authentication token to a JSON string.
	decoded, err := base64.StdEncoding.DecodeString(token)

	// If the token is malformed, return HTTP code 403 to the client.
	if err != nil {
		log.Error().Err(err).Msg(invalidTokenMessage)
		return nil, &AuthenticationError{ErrString: invalidTokenMessage}
	}

	tk := &types.Token{}

	if authType == "xrh" {
		// If the authentication type is xrh (x-rh-identity header).
		err = json.Unmarshal(decoded, tk)
		if err != nil {
			log.Error().Err(err).Msg(invalidTokenMessage)
			return nil, &AuthenticationError{ErrString: invalidTokenMessage}
		}
	} else {
		err := errors.New("unknown auth type")
		log.Error().Err(err).Send()
		return nil, err
	}
	return tk, nil
}

// GetAuthToken retrieves the authentication token from the request context.
// It returns a pointer to an Identity and an error if the token is not
// provided or has the wrong type.
func GetAuthToken(request *http.Request) (*types.Identity, error) {
	i := request.Context().Value(types.ContextKeyUser)

	if i == nil {
		return nil, &AuthenticationError{ErrString: "token is not provided"}
	}

	identity, ok := i.(types.Identity)
	if !ok {
		return nil, &AuthenticationError{ErrString: "contextKeyUser has wrong type"}
	}

	return &identity, nil
}

// GetAuthTokenHeader retrieves the authentication token from the HTTP request header.
// It returns the token as a string, or an empty string if the token is missing.
func GetAuthTokenHeader(r *http.Request) string {
	var tokenHeader string

	log.Debug().Msg("Retrieving x-rh-identity token")
	// Grab the token from the header
	tokenHeader = r.Header.Get(XRHAuthTokenHeader)

	log.Debug().Int("Length", len(tokenHeader)).Msg("Token retrieved")

	if tokenHeader == "" {
		log.Error().Msg(missingTokenMessage)
		return ""
	}

	return tokenHeader
}
