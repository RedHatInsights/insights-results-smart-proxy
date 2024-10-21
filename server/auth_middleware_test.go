// Copyright 2020 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/insights-results-smart-proxy/auth"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"

	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	types "github.com/RedHatInsights/insights-results-types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCase struct {
	name           string
	identity       string
	expectedError  string
	expectedUserID types.UserID
	expectedOrgID  types.OrgID
}

var (
	validIdentityXRH = types.Token{
		Identity: types.Identity{
			AccountNumber: types.UserID("1"),
			OrgID:         1,
			User: types.User{
				UserID: types.UserID("1"),
			},
			Type: "ServiceAccount",
		},
	}
)

func TestGetCurrentUserID(t *testing.T) {
	testCases := []testCase{
		{
			name:           "valid token",
			identity:       "valid",
			expectedError:  "",
			expectedUserID: validIdentityXRH.Identity.User.UserID,
		},
		{
			name:          "no token",
			identity:      "empty",
			expectedError: "token is not provided",
		},
		{
			name:          "invalid token",
			identity:      "bad",
			expectedError: "contextKeyUser has wrong type",
		},
	}

	testServer := server.HTTPServer{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := getRequest(t, tc.identity)

			userID, err := testServer.GetCurrentUserID(req)
			if tc.expectedError == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedUserID, userID)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			}
		})
	}
}

func TestGetCurrentOrgID(t *testing.T) {
	testCases := []testCase{
		{
			name:          "valid token",
			identity:      "valid",
			expectedError: "",
			expectedOrgID: validIdentityXRH.Identity.OrgID,
		},
		{
			name:          "no token",
			identity:      "empty",
			expectedError: "token is not provided",
		},
		{
			name:          "invalid token",
			identity:      "bad",
			expectedError: "contextKeyUser has wrong type",
		},
	}

	testServer := server.HTTPServer{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			req := getRequest(t, tc.identity)

			orgID, err := testServer.GetCurrentOrgID(req)
			if tc.expectedError == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedOrgID, orgID)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			}
		})
	}
}

func TestGetCurrentOrgIDUserIDFromToken(t *testing.T) {
	testCases := []testCase{
		{
			name:           "valid token",
			identity:       "valid",
			expectedError:  "",
			expectedOrgID:  validIdentityXRH.Identity.OrgID,
			expectedUserID: validIdentityXRH.Identity.User.UserID,
		},
		{
			name:          "no token",
			identity:      "empty",
			expectedError: "token is not provided",
		},
		{
			name:          "invalid token",
			identity:      "bad",
			expectedError: "contextKeyUser has wrong type",
		},
	}

	testServer := server.HTTPServer{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := getRequest(t, tc.identity)

			userID, err := testServer.GetCurrentUserID(req)
			if tc.expectedError == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedUserID, userID)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			}

			orgID, err := testServer.GetCurrentOrgID(req)
			if tc.expectedError == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedOrgID, orgID)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			}
		})
	}
}

func TestGetAuthTokenXRHHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth.GetAuthTokenHeader(r)
	})

	request, err := http.NewRequest(http.MethodGet, "an url", http.NoBody)
	assert.NoError(t, err)
	request.Header.Set(auth.XRHAuthTokenHeader, "token")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	// if xrh auth type is used, contents of header are checked in different function, thus only calling GetAuthTokenHeader is successful
	assert.Equal(t, http.StatusOK, recorder.Code)
}

// TestUnsupportedAuthType checks how that only "xrh" auth type is supported
func TestUnsupportedAuthType(t *testing.T) {
	unsupportedAuthConfig := helpers.DefaultServerConfig
	unsupportedAuthConfig.AuthType = "jwt"

	helpers.AssertAPIRequest(t, &unsupportedAuthConfig, nil, nil, nil, nil, &helpers.APIRequest{
		Method:      http.MethodGet,
		Endpoint:    server.RuleIDs, // any endpoint that requires auth
		XRHIdentity: goodXRHAuthToken,
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
	})
}

func getRequest(t *testing.T, identity string) *http.Request {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, "an url", http.NoBody)
	assert.NoError(t, err)

	if identity == "valid" {
		ctx := context.WithValue(req.Context(), types.ContextKeyUser, validIdentityXRH.Identity)
		req = req.WithContext(ctx)
	}

	if identity == "bad" {
		ctx := context.WithValue(req.Context(), types.ContextKeyUser, "not an identity")
		req = req.WithContext(ctx)
	}

	return req
}
