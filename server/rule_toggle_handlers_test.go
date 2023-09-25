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
	"net/http"
	"testing"

	ctypes "github.com/RedHatInsights/insights-results-types"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
)

func TestEnableEndpoint(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)
		expectedBody := `{"status": "ok"}`
		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPut,
				Endpoint:     ira_server.EnableRuleForClusterEndpoint,
				EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedBody,
			},
		)

		helpers.AssertAPIRequest(
			t,
			&helpers.DefaultServerConfig,
			&helpers.DefaultServicesConfig,
			nil,
			nil,
			nil,
			&helpers.APIRequest{
				Method:       http.MethodPut,
				Endpoint:     server.EnableRuleForClusterEndpoint,
				EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1},
				XRHIdentity:  goodXRHAuthToken,
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

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)
		expectedBody := `{"status": "ok"}`
		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPut,
				Endpoint:     ira_server.DisableRuleForClusterEndpoint,
				EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedBody,
			},
		)

		helpers.AssertAPIRequest(
			t,
			&helpers.DefaultServerConfig,
			&helpers.DefaultServicesConfig,
			nil,
			nil,
			nil,
			&helpers.APIRequest{
				Method:       http.MethodPut,
				Endpoint:     server.DisableRuleForClusterEndpoint,
				EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1},
				XRHIdentity:  goodXRHAuthToken,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedBody,
			},
		)
	}, testTimeout)
}

func TestEnableEndpointBadErrorKey(t *testing.T) {
	err := loadMockRuleContentDir(&ctypes.RuleContentDirectory{})
	assert.Nil(t, err)
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		expectedBody := fmt.Sprintf(
			`{"status":"Item with ID %s/%s was not found in the storage"}`,
			testdata.Rule1ID,
			testdata.ErrorKey1,
		)
		helpers.AssertAPIRequest(
			t,
			&helpers.DefaultServerConfig,
			&helpers.DefaultServicesConfig,
			nil,
			nil,
			nil,
			&helpers.APIRequest{
				Method:       http.MethodPut,
				Endpoint:     server.EnableRuleForClusterEndpoint,
				EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1},
				XRHIdentity:  goodXRHAuthToken,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusNotFound,
				Body:       expectedBody,
			},
		)
	}, testTimeout)
}

func TestDisableEndpointBadErrorKey(t *testing.T) {
	err := loadMockRuleContentDir(&ctypes.RuleContentDirectory{})
	assert.Nil(t, err)
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		expectedBody := fmt.Sprintf(
			`{"status":"Item with ID %s/%s was not found in the storage"}`,
			testdata.Rule1ID,
			testdata.ErrorKey1,
		)
		helpers.AssertAPIRequest(
			t,
			&helpers.DefaultServerConfig,
			&helpers.DefaultServicesConfig,
			nil,
			nil,
			nil,
			&helpers.APIRequest{
				Method:       http.MethodPut,
				Endpoint:     server.DisableRuleForClusterEndpoint,
				EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1},
				XRHIdentity:  goodXRHAuthToken,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusNotFound,
				Body:       expectedBody,
			},
		)
	}, testTimeout)
}
