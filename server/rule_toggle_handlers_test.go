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
	"net/http"
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
)

func TestEnableEndpoint(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)
		defer content.ResetContent()
		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)
		expectedBody := `{"status": "ok"}`
		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPut,
				Endpoint:     ira_server.EnableRuleForClusterEndpoint,
				EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID, userIDOnGoodJWTAuthBearer},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedBody,
			},
		)

		helpers.AssertAPIRequest(
			t,
			&serverConfigJWT,
			&helpers.DefaultServicesConfig,
			nil,
			nil,
			nil,
			&helpers.APIRequest{
				Method:             http.MethodPut,
				Endpoint:           server.EnableRuleForClusterEndpoint,
				EndpointArgs:       []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1},
				AuthorizationToken: goodJWTAuthBearer,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedBody,
			},
		)

	}, testTimeout)
}

func TestDisableEndpoint(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)
		defer content.ResetContent()
		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)
		expectedBody := `{"status": "ok"}`
		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPut,
				Endpoint:     ira_server.DisableRuleForClusterEndpoint,
				EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID, userIDOnGoodJWTAuthBearer},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedBody,
			},
		)

		helpers.AssertAPIRequest(
			t,
			&serverConfigJWT,
			&helpers.DefaultServicesConfig,
			nil,
			nil,
			nil,
			&helpers.APIRequest{
				Method:             http.MethodPut,
				Endpoint:           server.DisableRuleForClusterEndpoint,
				EndpointArgs:       []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1},
				AuthorizationToken: goodJWTAuthBearer,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedBody,
			},
		)

	}, testTimeout)
}
