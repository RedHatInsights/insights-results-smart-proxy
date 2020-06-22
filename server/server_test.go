/*
Copyright © 2020 Red Hat, Inc.

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

package server_test

import (
	"net/http"
	"testing"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"
)

var config = server.Configuration{
	Address:     ":8080",
	APIPrefix:   "/api/test/",
	APISpecFile: "openapi.json",
	Debug:       true,
	Auth:        false,
	UseHTTPS:    false,
	EnableCORS:  true,
}

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

func TestServerStartError(t *testing.T) {
	testServer := server.New(server.Configuration{
		Address:   "localhost:99999",
		APIPrefix: "",
	}, services.Configuration{
		AggregatorBaseEndpoint: "http://localhost:8081/api/v1/",
		ContentBaseEndpoint:    "http://localhost:8082/api/v1/",
	},
		nil,
		nil,
	)

	err := testServer.Start()
	assert.EqualError(t, err, "listen tcp: address 99999: invalid port")
}

func TestAddCORSHeaders(t *testing.T) {
	testServer := server.New(
		config,
		services.Configuration{
			AggregatorBaseEndpoint: "http://localhost:8081/api/v1/",
			ContentBaseEndpoint:    "http://localhost:8082/api/v1/",
		},
		nil,
		nil,
	)

	helpers.AssertAPIRequest(t, testServer, config.APIPrefix, &helpers.APIRequest{
		Method:   http.MethodOptions,
		Endpoint: server.RuleGroupsEndpoint,
		ExtraHeaders: http.Header{
			"Origin":                         []string{"http://example.com"},
			"Access-Control-Request-Method":  []string{http.MethodOptions},
			"Access-Control-Request-Headers": []string{"X-Csrf-Token,Content-Type,Content-Length"},
		},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Access-Control-Allow-Origin":      "*",
			"Access-Control-Allow-Credentials": "true",
			"Access-Control-Allow-Methods":     http.MethodOptions,
			"Access-Control-Allow-Headers":     "X-Csrf-Token,Content-Type,Content-Length",
		},
	})
}
