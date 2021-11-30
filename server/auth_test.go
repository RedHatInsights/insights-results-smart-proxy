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
	"testing"

	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	types "github.com/RedHatInsights/insights-results-types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCase struct {
	name             string
	identity         string
	expectedError    string
	expectedIdentity types.Identity
	expectedUserID   types.UserID
}

var (
	validIdentity = types.Identity{
		AccountNumber: types.UserID("a user"),
		Internal:      types.Internal{OrgID: 1},
	}
)

func TestGetAuthToken(t *testing.T) {
	testCases := []testCase{
		{
			name:             "valid token",
			identity:         "valid",
			expectedError:    "",
			expectedIdentity: validIdentity,
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

			identity, err := testServer.GetAuthToken(req)
			if tc.expectedError == "" {
				require.NoError(t, err)
				assert.Equal(t, &tc.expectedIdentity, identity)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			}
		})
	}
}

func TestGetCurrentUserID(t *testing.T) {
	testCases := []testCase{
		{
			name:           "valid token",
			identity:       "valid",
			expectedError:  "",
			expectedUserID: validIdentity.AccountNumber,
		},
		{
			name:          "no token",
			identity:      "empty",
			expectedError: "user id is not provided",
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

func getRequest(t *testing.T, identity string) *http.Request {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, "an url", http.NoBody)
	assert.NoError(t, err)

	if identity == "valid" {
		ctx := context.WithValue(req.Context(), types.ContextKeyUser, validIdentity)
		req = req.WithContext(ctx)
	}

	if identity == "bad" {
		ctx := context.WithValue(req.Context(), types.ContextKeyUser, "not an identity")
		req = req.WithContext(ctx)
	}

	return req
}
