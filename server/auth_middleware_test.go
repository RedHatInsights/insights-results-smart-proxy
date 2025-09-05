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
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/insights-results-smart-proxy/auth"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/RedHatInsights/insights-results-smart-proxy/metrics"
	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	types "github.com/RedHatInsights/insights-results-types"
	prommodels "github.com/prometheus/client_model/go"
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

// Tests for authorization middleware
// MockRBACClient is a mock implementation of the RBAC client for testing
type MockRBACClient struct {
	authorized bool
	enforcing  bool
}

func (m *MockRBACClient) IsAuthorized(token string) bool {
	return m.authorized
}

func (m *MockRBACClient) IsEnforcing() bool {
	return m.enforcing
}

func TestAuthorizationMiddleware(t *testing.T) {
	testCases := []struct {
		name           string
		xrhHeader      types.Token
		rbacClient     *MockRBACClient
		expectedStatus int
	}{
		{
			name: "valid service account with permissions",
			xrhHeader: types.Token{
				Identity: types.Identity{
					AccountNumber: types.UserID("1"),
					OrgID:         1,
					User: types.User{
						UserID: types.UserID("service-account-id"),
					},
					Type: "ServiceAccount",
				}},
			rbacClient:     &MockRBACClient{authorized: true, enforcing: true},
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid service account without permissions",
			xrhHeader: types.Token{
				Identity: types.Identity{
					AccountNumber: types.UserID("1"),
					OrgID:         1,
					User: types.User{
						UserID: types.UserID("service-account-id"),
					},
					Type: "ServiceAccount",
				}},
			rbacClient:     &MockRBACClient{authorized: false, enforcing: true},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "unknown identity type",
			xrhHeader: types.Token{
				Identity: types.Identity{
					AccountNumber: types.UserID("1"),
					OrgID:         1,
					User: types.User{
						UserID: types.UserID("user-id"),
					},
					Type: "UnknownType",
				}},
			rbacClient:     &MockRBACClient{authorized: false, enforcing: true},
			expectedStatus: http.StatusOK, // RBAC is only enforced on ServiceAccounts
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tc.xrhHeader)
			assert.NoError(t, err)

			token := base64.StdEncoding.EncodeToString(jsonData)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			// Set the authorization header, anything but an empty string is enough for these UTs
			req.Header.Set("x-rh-identity", token)

			// Set the context with the identity
			ctx := context.WithValue(req.Context(), types.ContextKeyUser, tc.xrhHeader)
			req = req.WithContext(ctx)

			recorder := httptest.NewRecorder()
			testServer := server.HTTPServer{
				Config: server.Configuration{
					AuthType: "xrh",
					UseRBAC:  true,
				},
			}
			testServer.SetRBACClient(tc.rbacClient)
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Placeholder to wrap with Authorization handler
				w.WriteHeader(http.StatusOK)
			})

			initValueRBACIdentityType := int64(getCounterVecValue(t, metrics.RBACIdentityType, tc.xrhHeader.Identity.Type))
			initValueRBACServiceAccountRejected := int64(getCounterValue(t, metrics.RBACServiceAccountRejected))
			authHandler := testServer.Authorization(handler, nil)

			authHandler.ServeHTTP(recorder, req)

			assert.Equal(t, tc.expectedStatus, recorder.Code)

			assertCounterVecValue(t, 1, metrics.RBACIdentityType, initValueRBACIdentityType, tc.xrhHeader.Identity.Type)
			if tc.expectedStatus == http.StatusForbidden {
				assertCounterValue(t, 1, metrics.RBACServiceAccountRejected, initValueRBACServiceAccountRejected)
			} else {
				assertCounterValue(t, 0, metrics.RBACServiceAccountRejected, initValueRBACServiceAccountRejected)
			}
		})
	}
}

func TestAuthorization_NoAuthURLs(t *testing.T) {
	// Setup the server and the request
	rbacClient := &MockRBACClient{authorized: false, enforcing: true}
	noAuthURLs := []string{"/public-endpoint"}
	testServer := server.HTTPServer{
		Config: server.Configuration{
			Auth:     true,
			AuthType: "xrh",
			UseRBAC:  true,
		},
	}
	testServer.SetRBACClient(rbacClient)

	// Create a request to a noAuthURL
	req := httptest.NewRequest(http.MethodGet, "/public-endpoint", nil)
	rr := httptest.NewRecorder()

	// Call the authorization middleware
	handler := testServer.Authorization(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), noAuthURLs)

	handler.ServeHTTP(rr, req)

	// Verify that the response is OK and RBAC was not used
	assert.Equal(t, http.StatusOK, rr.Code)
}

func assertCounterVecValue(tb testing.TB, expected int64, counterVec *prometheus.CounterVec, initValue int64, labels ...string) {
	assert.Equal(tb, float64(initValue+expected), getCounterVecValue(tb, counterVec, labels...))
}

func assertCounterValue(tb testing.TB, expected int64, counter prometheus.Counter, initValue int64) {
	assert.Equal(tb, float64(expected+initValue), getCounterValue(tb, counter))
}

func getCounterVecValue(tb testing.TB, counterVec *prometheus.CounterVec, labels ...string) float64 {
	counter, err := counterVec.GetMetricWithLabelValues(labels...)
	if err != nil {
		tb.Errorf("Unable to get counter from counterVec %v", err)
	}
	return getCounterValue(tb, counter)
}

func getCounterValue(tb testing.TB, counter prometheus.Counter) float64 {
	pb := &prommodels.Metric{}
	err := counter.Write(pb)
	if err != nil {
		tb.Errorf("Unable to get counter from counter %v", err)
	}

	return pb.GetCounter().GetValue()
}
