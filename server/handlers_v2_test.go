// Copyright 2021 Red Hat, Inc
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
	"fmt"
	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
)

func TestHTTPServer_SetRating(t *testing.T) {
	defer helpers.CleanAfterGock(t)

	rating := `{"rule": "rule_module|error_key","rating":-1}`
	aggregatorResponse := fmt.Sprintf(`{"status":"ok", "ratings":%s}`, rating)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodPost,
			Endpoint:     ira_server.Rating,
			EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
			Body:         rating,
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       aggregatorResponse,
		},
	)

	helpers.AssertAPIv2Request(
		t,
		&serverConfigJWT,
		nil,
		nil,
		nil,
		nil,
		&helpers.APIRequest{
			Method:             http.MethodPost,
			Endpoint:           server.Rating,
			Body:               rating,
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       rating,
		},
	)
}

// TestHTTPServer_ClustersDetailEndpointAggregatorResponseOk verifies that
// the 200 OK and the response body from aggregator are correctly
// forwarded to the client
func TestHTTPServer_ClustersDetailEndpointAggregatorResponseOk(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	aggregatorResponse := `
	{
		"data":[{"cluster":"5d5892d3-1f74-4ccf-91af-548dfc9767bb"}],
		"meta":{
			"count":1,
			"rule_id":"ccx_rules_ocp.external.rules.container_max_root_partition_size|ek1"
		},
		"status":"ok"
	}
	`

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.RuleClusterDetailEndpoint,
			EndpointArgs: []interface{}{testdata.Rule1CompositeID, testdata.OrgID, userIDOnGoodJWTAuthBearer},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       aggregatorResponse,
		},
	)

	helpers.AssertAPIv2Request(
		t,
		&serverConfigJWT,
		nil,
		nil,
		nil,
		nil,
		&helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.ClustersDetail,
			EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       aggregatorResponse,
		},
	)
}

// TestHTTPServer_ClustersDetailEndpointAggregatorResponse400 verifies that
// the 400 Bad Request and the response body from aggregator are correctly
// forwarded to the client
func TestHTTPServer_ClustersDetailEndpointAggregatorResponse400(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	aggregatorResponse := `{"status":"Error during parsing param 'rule_selector' with value X"}`

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.RuleClusterDetailEndpoint,
			EndpointArgs: []interface{}{testdata.Rule1CompositeID, testdata.OrgID, userIDOnGoodJWTAuthBearer},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusBadRequest,
			Body:       aggregatorResponse,
		},
	)

	helpers.AssertAPIv2Request(
		t,
		&serverConfigJWT,
		nil,
		nil,
		nil,
		nil,
		&helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.ClustersDetail,
			EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode: http.StatusBadRequest,
			Body:       aggregatorResponse,
		},
	)
}

// TestHTTPServer_ClustersDetailEndpointAggregatorResponse404 verifies that
// the 404 Not Found and the response body from aggregator are correctly
// forwarded to the client
func TestHTTPServer_ClustersDetailEndpointAggregatorResponse404(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	aggregatorResponse := `{"status":"Item with ID plugin.1|EK_1 was not found in the storage"}`
	proxyResponse := `
	{
		"data":[],
		"meta":{
			"count":0,
			"rule_id":"ccx_rules_ocp.external.rules.node_installer_degraded|ek1"
		},
		"status":"ok"
	}
	`

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.RuleClusterDetailEndpoint,
			EndpointArgs: []interface{}{testdata.Rule1CompositeID, testdata.OrgID, userIDOnGoodJWTAuthBearer},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusNotFound,
			Body:       aggregatorResponse,
		},
	)

	helpers.AssertAPIv2Request(
		t,
		&serverConfigJWT,
		nil,
		nil,
		nil,
		nil,
		&helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.ClustersDetail,
			EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       proxyResponse,
		},
	)
}

// TestHTTPServer_ClustersDetailEndpointAggregatorResponse500 verifies that
// the 500 Internal Error and the response body from aggregator are correctly
// forwarded to the client
func TestHTTPServer_ClustersDetailEndpointAggregatorResponse500(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	aggregatorResponse := `{"status": "Internal Server Error"}`
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.RuleClusterDetailEndpoint,
			EndpointArgs: []interface{}{testdata.Rule1CompositeID, testdata.OrgID, userIDOnGoodJWTAuthBearer},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       aggregatorResponse,
		},
	)

	helpers.AssertAPIv2Request(
		t,
		&serverConfigJWT,
		nil,
		nil,
		nil,
		nil,
		&helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.ClustersDetail,
			EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       aggregatorResponse,
		},
	)
}
