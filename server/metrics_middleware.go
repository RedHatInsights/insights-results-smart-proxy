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

package server

import (
	"net/http"
	"strings"

	"github.com/RedHatInsights/insights-results-smart-proxy/metrics"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

const unknownUserAgent = "unknown"

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code for metrics
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// normalizeUserAgent normalizes the user agent string to reduce cardinality
// and prevent potential high cardinality issues in Prometheus
func normalizeUserAgent(userAgent string) string {
	if userAgent == "" {
		return unknownUserAgent
	}

	// Convert to lowercase for consistency
	userAgent = strings.ToLower(userAgent)

	// Check for known user agents and normalize them
	switch {
	case strings.Contains(userAgent, "insights-operator/mcp-server"):
		return "insights-operator/mcp-server"
	case strings.Contains(userAgent, "insights-operator"):
		return "insights-operator"
	case strings.Contains(userAgent, "acm-operator"):
		return "acm-operator"
	case strings.Contains(userAgent, "mozilla"):
		return "browser"
	case strings.Contains(userAgent, "curl"):
		return "curl"
	case strings.Contains(userAgent, "wget"):
		return "wget"
	case strings.Contains(userAgent, "python"):
		return "python-client"
	case strings.Contains(userAgent, "go-http-client"):
		return "go-client"
	case strings.Contains(userAgent, "okhttp"):
		return "okhttp-client"
	default:
		return unknownUserAgent
	}
}

// getEndpointFromRequest extracts the endpoint pattern from the request
func getEndpointFromRequest(r *http.Request) string {
	// Try to get the route from mux
	route := mux.CurrentRoute(r)
	if route == nil {
		log.Error().Msg("router is nil")
		return ""
	}

	endpoint, err := route.GetPathTemplate()
	if err != nil {
		log.Error().Err(err).Msg("not valid endpoint template found")
		return ""
	}

	return endpoint
}

// MetricsMiddleware creates a middleware that captures user-agent information in metrics
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip metrics endpoint itself to avoid recursion
		if strings.HasSuffix(r.URL.Path, "/metrics") {
			next.ServeHTTP(w, r)
			return
		}

		// Extract and normalize user agent
		userAgent := r.Header.Get("User-Agent")
		normalizedUserAgent := normalizeUserAgent(userAgent)

		// Get endpoint pattern
		endpoint := getEndpointFromRequest(r)

		// Wrap the response writer to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Increment request counter
		metrics.APIEndpointsRequestsWithUserAgent.WithLabelValues(endpoint, normalizedUserAgent).Inc()

		// Call the next handler
		next.ServeHTTP(rw, r)
	})
}
