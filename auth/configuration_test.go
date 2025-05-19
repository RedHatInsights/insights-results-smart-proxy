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

package auth_test

import (
	"testing"

	"github.com/RedHatInsights/insights-results-smart-proxy/auth"
)

func TestConstructRBACURL(t *testing.T) {
	tests := []struct {
		config       *auth.RBACConfig
		expectedURL  string
		expectedHost string
		expectError  bool
	}{
		{
			config:       &auth.RBACConfig{URL: "example.com"},
			expectedURL:  "https://example.com/access/?application=ocp-advisor&limit=100",
			expectedHost: "example.com",
			expectError:  false,
		},
		{
			config:       &auth.RBACConfig{URL: "http://example.com"},
			expectedURL:  "http://example.com/access/?application=ocp-advisor&limit=100",
			expectedHost: "example.com",
			expectError:  false,
		},
		{
			// ephemeral case:
			config:       &auth.RBACConfig{URL: "http://rbac-service:8000/api/rbac/v1"},
			expectedURL:  "http://rbac-service:8000/api/rbac/v1/access/?application=ocp-advisor&limit=100",
			expectedHost: "rbac-service:8000",
			expectError:  false,
		},
		{
			config:       &auth.RBACConfig{URL: "wrong-schema:/invalid"},
			expectedURL:  "https://wrong-schema:/invalid/access/?application=ocp-advisor&limit=100",
			expectedHost: "wrong-schema:",
			expectError:  false, //url.Parse and url.ParseURI can't catch bad URLs, careful!
		},
	}

	for _, tt := range tests {
		base, host, err := auth.ConstructRBACURL(tt.config)

		// Check for expected error
		if (err != nil) != tt.expectError {
			t.Errorf("expected error: %v, got: %v", tt.expectError, err)
		}

		// If no error is expected, check the returned values
		if !tt.expectError {
			if base != tt.expectedURL {
				t.Errorf("expected base URL: %s, got: %s", tt.expectedURL, base)
			}
			if host != tt.expectedHost {
				t.Errorf("expected host: %s, got: %s", tt.expectedHost, host)
			}
		}
	}
}
