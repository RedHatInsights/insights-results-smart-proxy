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

package auth

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"
)

// RBACConfig holds the configuration for RBAC settings.
type RBACConfig struct {
	URL         string `mapstructure:"url" toml:"url"`
	EnforceAuth bool   `mapstructure:"enforce" toml:"enforce"`
}

// constructRBACURL constructs the RBAC URL with the query parameters for checking
// access to Advisor OCP resources. It takes an RBACConfig and returns the
// constructed base URL, host, and any error encountered.
func constructRBACURL(config *RBACConfig) (string, string, error) {
	if !strings.HasPrefix(config.URL, "http://") && !strings.HasPrefix(config.URL, "https://") {
		config.URL = "https://" + config.URL // Default to HTTPS if no scheme is provided
	}

	// Parse the URL to handle both the Host and the Endpoint correctly
	baseURL, err := url.Parse(config.URL)
	if err != nil {
		return "", "", fmt.Errorf("invalid host format: %v", err)
	}

	// Add the /access/ endpoint and ocp-advisor parameter
	baseURL.Path += "/access/"

	params := url.Values{}
	params.Add("application", "ocp-advisor")
	params.Add("limit", "100") // We don't know what customers might have configured, so let's limit responses' size

	uri := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	log.Info().Str("RBAC URL:", uri).Send()

	return uri, baseURL.Host, nil
}
