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

package helpers

import (
	"testing"
	"time"

	"github.com/RedHatInsights/insights-content-service/groups"
	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"

	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"
)

// APIRequest represents APIRequest
type APIRequest = helpers.APIRequest

// APIResponse represents APIResponse
type APIResponse = helpers.APIResponse

var (
	// ExecuteRequest executes request
	ExecuteRequest = helpers.ExecuteRequest
	// CheckResponseBodyJSON checks response body
	CheckResponseBodyJSON = helpers.CheckResponseBodyJSON
	// AssertReportResponsesEqual fails if report responses aren't equal
	AssertReportResponsesEqual = helpers.AssertReportResponsesEqual
	// NewGockAPIEndpointMatcher creates a matcher for a given endpoint for gock
	NewGockAPIEndpointMatcher = helpers.NewGockAPIEndpointMatcher
	// GockExpectAPIRequest makes gock expect the request with the baseURL and sends back the response
	GockExpectAPIRequest = helpers.GockExpectAPIRequest
	// CleanAfterGock cleans after gock library and prints all unmatched requests
	CleanAfterGock = helpers.CleanAfterGock
	// MustGobSerialize serializes an object using gob or panics
	MustGobSerialize = helpers.MustGobSerialize
)

var (
	// DefaultServerConfig is a default server config
	DefaultServerConfig = server.Configuration{
		Address:     ":8081",
		APIPrefix:   "/api/v1/",
		APISpecFile: "openapi.json",
		Debug:       true,
		Auth:        false,
		AuthType:    "",
		UseHTTPS:    false,
		EnableCORS:  false,
	}

	// DefaultServicesConfig is a default services config
	DefaultServicesConfig = services.Configuration{
		AggregatorBaseEndpoint: "http://localhost:8080/",
		ContentBaseEndpoint:    "http://localhost:8082/",
		GroupsPollingTime:      1 * time.Minute,
	}
)

// AssertAPIRequest creates new server with provided
// serverConfig, servicesConfig (you can leave them nil to use the default ones),
// groupsChannel and contentChannel(can be nil)
// sends api request and checks api response (see docs for APIRequest and APIResponse)
func AssertAPIRequest(
	t testing.TB,
	serverConfig *server.Configuration,
	servicesConfig *services.Configuration,
	groupsChannel chan []groups.Group,
	request *helpers.APIRequest,
	expectedResponse *helpers.APIResponse,
) {
	if serverConfig == nil {
		serverConfig = &DefaultServerConfig
	}
	if servicesConfig == nil {
		servicesConfig = &DefaultServicesConfig
	}

	testServer := server.New(
		*serverConfig,
		*servicesConfig,
		groupsChannel,
	)

	helpers.AssertAPIRequest(t, testServer, serverConfig.APIPrefix, request, expectedResponse)
}
