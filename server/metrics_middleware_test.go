// Copyright 2025 Red Hat, Inc
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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-results-smart-proxy/metrics"
	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

type Endpoint struct {
	Path    string
	Func    func(http.ResponseWriter, *http.Request)
	Methods []string
}

// TestNormalizeUserAgent tests the user agent normalization functionality
func TestNormalizeUserAgent(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"", "unknown"},
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36", "browser"},
		{"insights-operator/mcp-server/4.10.0", "insights-operator/mcp-server"},
		{"insights-operator/4.10.0", "insights-operator"},
		{"acm-operator/2.5.0", "acm-operator"},
		{"curl/7.68.0", "curl"},
		{"wget/1.20.3", "wget"},
		{"python-requests/2.25.1", "python-client"},
		{"Go-http-client/1.1", "go-client"},
		{"okhttp/4.9.0", "okhttp-client"},
		{"CustomClient/1.0", "unknown"},
		{"/only_version", "unknown"},
	}

	endpoints := []Endpoint{
		{
			Path: "/test",
			Func: func(w http.ResponseWriter, r *http.Request) {
				err := responses.Send(http.StatusOK, w, responses.BuildOkResponse())
				helpers.FailOnError(t, err)
			},
			Methods: []string{http.MethodGet},
		},
	}

	// Create test router
	router := createTestRouter(endpoints)

	// We need to test the internal function, but it's not exported.
	// For now, we'll test the middleware behavior indirectly through HTTP requests
	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			// Create test request with specific user agent
			req := httptest.NewRequest("GET", "/test", nil)
			if tc.input != "" {
				req.Header.Set("User-Agent", tc.input)
			}

			rr := httptest.NewRecorder()
			initValueUserAgentMetric := int64(getCounterVecValue(t, metrics.APIEndpointsRequestsWithUserAgent, "/test", tc.expected))

			router.ServeHTTP(rr, req)

			// Verify the request was processed
			assert.Equal(t, http.StatusOK, rr.Code)
			assertCounterVecValue(t, 1, metrics.APIEndpointsRequestsWithUserAgent, initValueUserAgentMetric, "/test", tc.expected)
		})
	}
}

// TestMetricsMiddleware tests the metrics middleware functionality
func TestMetricsMiddleware(t *testing.T) {
	// Define test endpoints
	endpoints := []Endpoint{
		{
			Path: "/api/v1/clusters",
			Func: func(w http.ResponseWriter, r *http.Request) {
				err := responses.Send(http.StatusOK, w, responses.BuildOkResponse())
				helpers.FailOnError(t, err)
			},
			Methods: []string{http.MethodGet},
		},
		{
			Path: "/api/v2/organizations/{orgId}/clusters/{clusterId}/reports",
			Func: func(w http.ResponseWriter, r *http.Request) {
				err := responses.Send(http.StatusOK, w, responses.BuildOkResponse())
				helpers.FailOnError(t, err)
			},
			Methods: []string{http.MethodGet},
		},
		{
			Path: "/api/v1/info",
			Func: func(w http.ResponseWriter, r *http.Request) {
				err := responses.Send(http.StatusOK, w, responses.BuildOkResponse())
				helpers.FailOnError(t, err)
			},
			Methods: []string{http.MethodGet},
		},
		{
			Path: "/api/v1/org_overview",
			Func: func(w http.ResponseWriter, r *http.Request) {
				err := responses.Send(http.StatusOK, w, responses.BuildOkResponse())
				helpers.FailOnError(t, err)
			},
			Methods: []string{http.MethodGet},
		},
	}

	// Create test router
	router := createTestRouter(endpoints)

	// Test different scenarios
	testCases := []struct {
		name              string
		path              string
		userAgent         string
		expectedUserAgent string
		expectedEndpoint  string
		method            string
	}{
		{
			name:              "Basic request with curl",
			path:              "/api/v1/clusters",
			userAgent:         "curl/7.68.0",
			expectedUserAgent: "curl",
			expectedEndpoint:  "/api/v1/clusters",
			method:            "GET",
		},
		{
			name:              "Insights operator request",
			path:              "/api/v2/organizations/123/clusters/abc/reports",
			userAgent:         "insights-operator/4.10.0",
			expectedUserAgent: "insights-operator",
			expectedEndpoint:  "/api/v2/organizations/{orgId}/clusters/{clusterId}/reports",
			method:            "GET",
		},
		{
			name:              "Browser request",
			path:              "/api/v1/info",
			userAgent:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			expectedUserAgent: "browser",
			expectedEndpoint:  "/api/v1/info",
			method:            "GET",
		},
		{
			name:              "Request without user agent",
			path:              "/api/v1/info",
			userAgent:         "",
			expectedUserAgent: "unknown",
			expectedEndpoint:  "/api/v1/info",
			method:            "GET",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.userAgent != "" {
				req.Header.Set("User-Agent", tc.userAgent)
			}
			rr := httptest.NewRecorder()
			initValueUserAgentMetric := int64(getCounterVecValue(t, metrics.APIEndpointsRequestsWithUserAgent, tc.expectedEndpoint, tc.expectedUserAgent))

			// Execute request using the test router
			router.ServeHTTP(rr, req)

			// Verify the request was processed successfully
			assert.Equal(t, http.StatusOK, rr.Code)
			assertCounterVecValue(t, 1, metrics.APIEndpointsRequestsWithUserAgent, initValueUserAgentMetric, tc.expectedEndpoint, tc.expectedUserAgent)
		})
	}
}

// TestMetricsMiddlewareSkipsMetricsEndpoint tests that the middleware skips the /metrics endpoint
func TestMetricsMiddlewareSkipsMetricsEndpoint(t *testing.T) {
	// Define test endpoints
	endpoints := []Endpoint{
		{
			Path: "/metrics",
			Func: func(w http.ResponseWriter, r *http.Request) {
				err := responses.Send(http.StatusOK, w, responses.BuildOkResponse())
				helpers.FailOnError(t, err)
			},
			Methods: []string{http.MethodGet},
		},
	}

	// Create test router
	router := createTestRouter(endpoints)

	req := httptest.NewRequest("GET", "/metrics", nil)
	req.Header.Set("User-Agent", "curl/7.68.0")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assertCounterVecValue(t, 0, metrics.APIEndpointsRequestsWithUserAgent, 0, "/metrics", "curl")
}

// createTestRouter creates a lightweight test router with the specified endpoints
func createTestRouter(endpoints []Endpoint) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	router.Use(server.MetricsMiddleware)

	for _, endpoint := range endpoints {
		router.HandleFunc(endpoint.Path, endpoint.Func).Methods(endpoint.Methods...)
	}

	return router
}
