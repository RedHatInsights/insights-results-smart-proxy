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

package server_test

import (
	"net/http"
	"testing"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
)

func TestMakeURLToEndpointWithValidValue(t *testing.T) {
	apiPrefixdbg := "api/dbg/"
	apiPrefixv1 := "api/v1/"
	apiPrefixv2 := "api/v2/"
	endpoint := "some_valid_endpoint"

	retvaldbg := httputils.MakeURLToEndpoint(apiPrefixdbg, endpoint)
	retvalv1 := httputils.MakeURLToEndpoint(apiPrefixv1, endpoint)
	retvalv2 := httputils.MakeURLToEndpoint(apiPrefixv2, endpoint)

	assert.Equal(t, "api/dbg/some_valid_endpoint", retvaldbg)
	assert.Equal(t, "api/v1/some_valid_endpoint", retvalv1)
	assert.Equal(t, "api/v2/some_valid_endpoint", retvalv2)
}

func TestHTTPServer_ProxyTo_VoteEndpointsExtractUserID(t *testing.T) {
	testCases := []struct {
		name        string
		method      string
		endpoint    string
		newEndpoint string
	}{
		{"like", http.MethodPut, server.LikeRuleEndpoint, ira_server.LikeRuleEndpoint},
		{"dislike", http.MethodPut, server.DislikeRuleEndpoint, ira_server.DislikeRuleEndpoint},
		{"reset_vote", http.MethodPut, server.ResetVoteOnRuleEndpoint, ira_server.ResetVoteOnRuleEndpoint},
		{"get_vote", http.MethodGet, server.GetVoteOnRuleEndpoint, ira_server.GetVoteOnRuleEndpoint},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			helpers.RunTestWithTimeout(t, func(t testing.TB) {
				defer helpers.CleanAfterGock(t)

				helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
					Method:       testCase.method,
					Endpoint:     testCase.newEndpoint,
					EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.UserID},
				}, &helpers.APIResponse{
					StatusCode: http.StatusOK,
					Body:       `{"status": "ok"}`,
				})

				helpers.AssertAPIRequest(t, nil, nil, nil, &helpers.APIRequest{
					Method:       testCase.method,
					Endpoint:     testCase.endpoint,
					EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1},
					UserID:       testdata.UserID,
					OrgID:        testdata.OrgID,
				}, &helpers.APIResponse{
					StatusCode: http.StatusOK,
					Body:       `{"status": "ok"}`,
				})
			}, testTimeout)
		})
	}
}

// TODO: test that proxying is done correctly including request / response modifiers for all endpoints

func TestHTTPServer_ProxyTo_VoteEndpointBadCharacter(t *testing.T) {
	badClusterName := "00000000000000000000000000000000000%1F"
	helpers.AssertAPIRequest(t, &helpers.DefaultServerConfig, &helpers.DefaultServicesConfig, nil, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.LikeRuleEndpoint,
		EndpointArgs: []interface{}{badClusterName, testdata.Rule1ID, testdata.ErrorKey1},
		UserID:       testdata.UserID,
		OrgID:        testdata.OrgID,
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body:       `{"status":"the parameters contains invalid characters and cannot be used"}`,
	})
}
