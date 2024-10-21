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
			config:       &auth.RBACConfig{Host: "example.com", Port: 443},
			expectedURL:  "https://example.com:443/access/?application=ocp-advisor&limit=100",
			expectedHost: "example.com:443",
			expectError:  false,
		},
		{
			config:       &auth.RBACConfig{Host: "http://example.com", Port: 0},
			expectedURL:  "http://example.com/access/?application=ocp-advisor&limit=100",
			expectedHost: "example.com",
			expectError:  false,
		},
		{
			config:       &auth.RBACConfig{Host: "invalid", Port: 0},
			expectedURL:  "https://invalid/access/?application=ocp-advisor&limit=100",
			expectedHost: "invalid",
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
