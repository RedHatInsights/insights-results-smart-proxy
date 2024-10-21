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

type RBACConfig struct {
	Host        string `mapstructure:"host" toml:"host"`
	Port        uint16 `mapstructure:"port" toml:"port,omitempty"`
	Enabled     bool   `mapstructure:"enabled" toml:"enabled"`
	EnforceAuth bool   `mapstructure:"enforce" toml:"enforce"`
}

// IsEnabled returns true if RBAC is enabled in the configuration
func IsEnabled(config *RBACConfig) bool {
	return config.Enabled
}

// constructRBACURL constructs the RBAC URL with the query parameters for
// checking access to Advisor OCP resources
func constructRBACURL(config *RBACConfig) (string, string, error) {
	if !strings.HasPrefix(config.Host, "http://") && !strings.HasPrefix(config.Host, "https://") {
		config.Host = "https://" + config.Host // Default to HTTPS if no scheme is provided
	}

	// Parse the URL to handle both the Host and the Endpoint correctly
	baseURL, err := url.Parse(config.Host)
	if err != nil {
		return "", "", fmt.Errorf("invalid host format: %v", err)
	}

	// Check if a port is provided and append it to the host
	if config.Port != 0 {
		baseURL.Host = fmt.Sprintf("%s:%d", baseURL.Hostname(), config.Port)
	}

	// Add the /access/ endpoint and ocp-advisor parameter. This could be configured via env-var,
	// but I don't think there's a need for complicating things further

	baseURL.Path = "/access/"

	params := url.Values{}
	params.Add("application", "ocp-advisor")
	params.Add("limit", "100") // We don't know what customers might have configured, so let's limit responses' size

	uri := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	log.Info().Str("RBAC URL:", uri).Send()

	return uri, baseURL.Host, nil
}
