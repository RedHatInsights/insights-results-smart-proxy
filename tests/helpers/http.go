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

	"github.com/RedHatInsights/insights-results-smart-proxy/amsclient"
	"github.com/RedHatInsights/insights-results-smart-proxy/auth"
	"github.com/RedHatInsights/insights-results-smart-proxy/content"

	"github.com/RedHatInsights/insights-content-service/groups"
	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"

	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"
)

// APIRequest data type represents APIRequest
type APIRequest = helpers.APIRequest

// APIResponse data type represents APIResponse
type APIResponse = helpers.APIResponse

var (
	// ExecuteRequest function executes specified HTTP request
	ExecuteRequest = helpers.ExecuteRequest

	// CheckResponseBodyJSON function checks response body. It is supposed
	// that the body is represented in JSON format
	CheckResponseBodyJSON = helpers.CheckResponseBodyJSON

	// AssertReportResponsesEqual function fails if report responses aren't
	// equal to each other
	AssertReportResponsesEqual = helpers.AssertReportResponsesEqual

	// NewGockAPIEndpointMatcher function creates a matcher for a given
	// endpoint for gock
	NewGockAPIEndpointMatcher = helpers.NewGockAPIEndpointMatcher

	// GockExpectAPIRequest function makes gock expect the request with the
	// baseURL and sends back the response
	GockExpectAPIRequest = helpers.GockExpectAPIRequest

	// CleanAfterGock function cleans after gock library and prints all
	// unmatched requests
	CleanAfterGock = helpers.CleanAfterGock

	// MustGobSerialize function serializes an object using gob or panics
	// if serialize oparation fails for any reason
	MustGobSerialize = helpers.MustGobSerialize
)

var (
	// DefaultServerConfig is data structure that represents default HTTP
	// server configuration (with CORS disabled). with XRH auth type. XRH type is used in all pre-prod/prod environments.
	DefaultServerConfig = server.Configuration{
		Address:                          ":8081",
		APIdbgPrefix:                     "/api/dbg/",
		APIv1Prefix:                      "/api/v1/",
		APIv2Prefix:                      "/api/v2/",
		APIv1SpecFile:                    "server/api/v1/openapi.json",
		APIv2SpecFile:                    "server/api/v2/openapi.json",
		Debug:                            true,
		Auth:                             true,
		AuthType:                         "xrh",
		UseHTTPS:                         false,
		EnableCORS:                       false,
		EnableInternalRulesOrganizations: false,
	}

	// DefaultServerConfigCORS is data structure that represents default
	// server configuration with CORS enabled
	DefaultServerConfigCORS = server.Configuration{
		Address:       ":8081",
		APIdbgPrefix:  "/api/dbg/",
		APIv1Prefix:   "/api/v1/",
		APIv2Prefix:   "/api/v2/",
		APIv1SpecFile: "server/api/v1/openapi.json",
		APIv2SpecFile: "server/api/v2/openapi.json",
		Debug:         true,
		Auth:          true,
		AuthType:      "xrh",
		UseHTTPS:      false,
		EnableCORS:    true,
	}

	// DefaultServicesConfig is data structure that represents default
	// services configuration
	DefaultServicesConfig = services.Configuration{
		AggregatorBaseEndpoint:  "http://localhost:8080/",
		ContentBaseEndpoint:     "http://localhost:8082/",
		GroupsPollingTime:       1 * time.Minute,
		ContentDirectoryTimeout: 100 * time.Millisecond,
	}
)

// AssertAPIRequest function creates new server with provided
// serverConfig, servicesConfig (you can leave them nil to use the default ones),
// groupsChannel and errorChannel (can be set to nil as well)
// sends api request and checks api response (see docs for APIRequest and APIResponse)
func AssertAPIRequest(
	t testing.TB,
	serverConfig *server.Configuration,
	servicesConfig *services.Configuration,
	groupsChannel chan []groups.Group,
	errorFoundChannel chan bool,
	errorChannel chan error,
	request *helpers.APIRequest,
	expectedResponse *helpers.APIResponse,
) {
	// if custom server configuration is not provided, use default one
	if serverConfig == nil {
		serverConfig = &DefaultServerConfig
	}

	assertAPIRequest(
		t,
		serverConfig,
		servicesConfig,
		groupsChannel,
		errorFoundChannel,
		errorChannel,
		serverConfig.APIv1Prefix,
		request,
		expectedResponse,
	)
}

// AssertAPIv2Request function is exactly the same as AssertAPIResponse, but intended to use
// with API v2 endpoints
func AssertAPIv2Request(
	t testing.TB,
	serverConfig *server.Configuration,
	servicesConfig *services.Configuration,
	groupsChannel chan []groups.Group,
	errorFoundChannel chan bool,
	errorChannel chan error,
	request *helpers.APIRequest,
	expectedResponse *helpers.APIResponse,
) {
	// if custom server configuration is not provided, use default one
	if serverConfig == nil {
		serverConfig = &DefaultServerConfig
	}

	assertAPIRequest(
		t,
		serverConfig,
		servicesConfig,
		groupsChannel,
		errorFoundChannel,
		errorChannel,
		serverConfig.APIv2Prefix,
		request,
		expectedResponse,
	)
}

func assertAPIRequest(
	t testing.TB,
	serverConfig *server.Configuration,
	servicesConfig *services.Configuration,
	groupsChannel chan []groups.Group,
	errorFoundChannel chan bool,
	errorChannel chan error,
	APIPrefix string,
	request *helpers.APIRequest,
	expectedResponse *helpers.APIResponse,
) {
	// create an instance of new REST API server with provided or default
	// configuration
	testServer := CreateHTTPServer(
		serverConfig,
		servicesConfig,
		nil, // AMS client
		nil, // Redis client
		groupsChannel,
		errorFoundChannel,
		errorChannel,
		nil, // RBAC client
	)

	// send the request to newly created REST API server and check its
	// response (if it matches the provided one)
	helpers.AssertAPIRequest(t, testServer, APIPrefix, request, expectedResponse)
}

// CreateHTTPServer creates an instance of the REST API server with provided or
// default configuration
func CreateHTTPServer(
	serverConfig *server.Configuration,
	servicesConfig *services.Configuration,
	amsClient amsclient.AMSClient,
	redis services.RedisInterface,
	groupsChannel chan []groups.Group,
	errorFoundChannel chan bool,
	errorChannel chan error,
	rbacClient auth.RBACClient,
) *server.HTTPServer {
	// if custom server configuration is not provided, use default one
	if serverConfig == nil {
		serverConfig = &DefaultServerConfig
	}

	// if custom services configuration is not provided, use default one
	if servicesConfig == nil {
		servicesConfig = &DefaultServicesConfig
	}

	content.SetContentDirectoryTimeout(servicesConfig.ContentDirectoryTimeout)

	// create an instance of new REST API server with provided or default
	// configuration
	return server.New(
		*serverConfig,
		*servicesConfig,
		amsClient,
		redis,
		groupsChannel,
		errorFoundChannel,
		errorChannel,
		rbacClient,
	)
}
