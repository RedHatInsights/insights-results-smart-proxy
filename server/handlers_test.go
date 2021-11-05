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
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-content-service/groups"
	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"

	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
	data "github.com/RedHatInsights/insights-results-smart-proxy/tests/testdata"
)

var (
	RuleContentDirectoryOnly1Rule = types.RuleContentDirectory{
		Config: types.GlobalRuleConfig{
			Impact: testdata.ImpactStrToInt,
		},
		Rules: map[string]types.RuleContent{
			"rc1": testdata.RuleContent1,
		},
	}

	RuleContentDirectoryOnlyDisabledRule = types.RuleContentDirectory{
		Config: types.GlobalRuleConfig{
			Impact: testdata.ImpactStrToInt,
		},
		Rules: map[string]types.RuleContent{
			"rc5": testdata.RuleContent5,
		},
	}
)

// TODO: test more cases for report endpoint
func TestHTTPServer_ReportEndpoint(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report3RulesExpectedResponse,
		})

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyReportResponse3Rules),
		})
	}, testTimeout)
}

func TestHTTPServer_ReportEndpoint_UnavailableContentService(t *testing.T) {
	var emptyResponse *types.RuleContentDirectory
	err := loadMockRuleContentDir(emptyResponse)
	assert.NotNil(t, err)

	expectedBody := `
		{
		   "status" : "Content directory cache has been empty for too long time; timeout triggered"
		}
	`
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report3RulesExpectedResponse,
		})

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusServiceUnavailable,
			Body:       expectedBody,
		})
	}, testTimeout)
}

// Reproducer for Bug 1977858
func TestHTTPServer_ReportEndpointNoContent(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report1RuleExpectedResponse,
		})

		// previously was InternalServerError, but it was changed as an edge-case which will appear as "No issues found"
		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyReportResponse1RuleNoContent),
		})
	}, testTimeout)
}

// Reproducer for Bug 1977858
func TestHTTPServer_ReportEndpointNoContentFor2Rules(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(&RuleContentDirectoryOnly1Rule)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report3RulesExpectedResponse,
		})

		// 1 rule returned, but count = 3
		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyReportResponse3Rules2NoContent),
		})
	}, testTimeout)
}

func TestHTTPServer_ReportEndpoint_WithOnlyOSDEndpoint(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report3RulesExpectedResponse,
		})

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint + "?" + server.OSDEligibleParam + "=true",
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyReportResponse3RulesWithOnlyOSD),
		})
	}, testTimeout)
}

func TestHTTPServer_ReportEndpoint_WithDisabledRules(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory5Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report3Rules1DisabledExpectedResponse,
		})

		// Same as previous one for the second endpoint request
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report3Rules1DisabledExpectedResponse,
		})

		// Same as previous one for the third endpoint request
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report3Rules1DisabledExpectedResponse,
		})

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint + "?" + server.GetDisabledParam + "=false",
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyReportResponse3RulesOnlyEnabled),
		})

		// Not using the parameter gets the same result as using with =false
		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyReportResponse3RulesOnlyEnabled),
		})

		// Enabling the parameter
		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint + "?" + server.GetDisabledParam + "=true",
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyReportResponse3RulesAll),
		})
	}, testTimeout)
}

func TestHTTPServer_ReportEndpoint_WithDisabledRulesAndMissingContent(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(&RuleContentDirectoryOnlyDisabledRule)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report3Rules1DisabledExpectedResponse,
		})

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyEmptyResponse),
		})
	}, testTimeout)
}

// TODO: test more cases for rule endpoint
func TestHTTPServer_RuleEndpoint(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: ira_server.RuleEndpoint,
			EndpointArgs: []interface{}{
				testdata.OrgID,
				testdata.ClusterName,
				testdata.UserID,
				fmt.Sprintf("%v|%v", testdata.RuleErrorKey1.RuleModule, testdata.RuleErrorKey1.ErrorKey),
			},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report3SingleRuleExpectedResponse,
		})

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.SingleRuleEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName, fmt.Sprintf("%v|%v", testdata.RuleErrorKey1.RuleModule, testdata.RuleErrorKey1.ErrorKey)},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyReportResponse3SingleRule),
		})
	}, testTimeout)
}

func TestHTTPServer_RuleEndpoint_UnavailableContentService(t *testing.T) {
	var emptyResponse *types.RuleContentDirectory
	err := loadMockRuleContentDir(emptyResponse)
	assert.NotNil(t, err)

	expectedBody := `
		{
		   "status" : "Content directory cache has been empty for too long time; timeout triggered"
		}
	`

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: ira_server.RuleEndpoint,
			EndpointArgs: []interface{}{
				testdata.OrgID,
				testdata.ClusterName,
				testdata.UserID,
				fmt.Sprintf("%v|%v", testdata.RuleErrorKey1.RuleModule, testdata.RuleErrorKey1.ErrorKey),
			},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report3SingleRuleExpectedResponse,
		})

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.SingleRuleEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName, fmt.Sprintf("%v|%v", testdata.RuleErrorKey1.RuleModule, testdata.RuleErrorKey1.ErrorKey)},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusServiceUnavailable,
			Body:       expectedBody,
		})
	}, testTimeout)
}

func TestHTTPServer_RuleEndpoint_WithOSD(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: ira_server.RuleEndpoint,
			EndpointArgs: []interface{}{
				testdata.OrgID,
				testdata.ClusterName,
				testdata.UserID,
				fmt.Sprintf("%v|%v", testdata.RuleErrorKey1.RuleModule, testdata.RuleErrorKey1.ErrorKey),
			},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report3SingleRuleExpectedResponse,
		})

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.SingleRuleEndpoint + "?" + server.OSDEligibleParam + "=true",
			EndpointArgs: []interface{}{testdata.ClusterName, fmt.Sprintf("%v|%v", testdata.RuleErrorKey1.RuleModule, testdata.RuleErrorKey1.ErrorKey)},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyReportResponse3SingleRule),
		})
	}, testTimeout)
}

func TestHTTPServer_RuleEndpoint_WithNotOSDRule(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: ira_server.RuleEndpoint,
			EndpointArgs: []interface{}{
				testdata.OrgID,
				testdata.ClusterName,
				testdata.UserID,
				fmt.Sprintf("%v|%v", testdata.RuleErrorKey2.RuleModule, testdata.RuleErrorKey2.ErrorKey),
			},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report3SingleRule2ExpectedResponse,
		})

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.SingleRuleEndpoint + "?" + server.OSDEligibleParam + "=true",
			EndpointArgs: []interface{}{testdata.ClusterName, fmt.Sprintf("%v|%v", testdata.RuleErrorKey2.RuleModule, testdata.RuleErrorKey2.ErrorKey)},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusNotFound,
			Body:       helpers.ToJSONString(SmartProxyReportResponse3NoRuleFound),
		})
	}, testTimeout)
}

// TestHTTPServer_GetContent
func TestHTTPServer_GetContent(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: server.Content,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(GetContentResponse3Rules),
			BodyChecker: ruleInContentChecker,
		})

	}, testTimeout)
}

// TestHTTPServer_OverviewEndpoint
func TestHTTPServer_OverviewEndpoint(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		// prepare list of organizations response
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ClustersForOrganizationEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(responses.BuildOkResponseWithData("clusters", []string{string(testdata.ClusterName)})),
		})

		// prepare report for cluster
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report3RulesExpectedResponse,
		})

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: server.OverviewEndpoint,
			OrgID:    testdata.OrgID,
			UserID:   testdata.UserID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(OverviewResponse),
		})
	}, testTimeout)
}

func TestHTTPServer_OverviewEndpoint_UnavailableContentService(t *testing.T) {
	var emptyResponse *types.RuleContentDirectory
	err := loadMockRuleContentDir(emptyResponse)
	assert.NotNil(t, err)

	expectedBody := `
		{
		   "status" : "Content directory cache has been empty for too long time; timeout triggered"
		}
	`

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		// prepare list of organizations response
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ClustersForOrganizationEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(responses.BuildOkResponseWithData("clusters", []string{string(testdata.ClusterName)})),
		})

		// prepare report for cluster
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report3RulesExpectedResponse,
		})

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: server.OverviewEndpoint,
			OrgID:    testdata.OrgID,
			UserID:   testdata.UserID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusServiceUnavailable,
			Body:       expectedBody,
		})
	}, testTimeout)
}

func TestInternalOrganizations(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(
		createRuleContentDirectoryFromRuleContent(
			[]types.RuleContent{RuleContentInternal1},
		),
	)
	assert.Nil(t, err)

	for _, testCase := range []struct {
		TestName           string
		ServerConfig       *server.Configuration
		ExpectedStatusCode int
		MockAuthToken      string
	}{
		{
			"Internal organizations enabled, Request denied",
			&serverConfigInternalOrganizations1,
			http.StatusForbidden,
			badJWTAuthBearer,
		},
		{
			"Internal organizations enabled, Request allowed",
			&serverConfigInternalOrganizations1,
			http.StatusOK,
			goodJWTAuthBearer,
		},
		{
			"Internal organizations disabled, Request allowed",
			&serverConfigJWT,
			http.StatusOK,
			badJWTAuthBearer,
		},
	} {
		t.Run(testCase.TestName, func(t *testing.T) {
			helpers.RunTestWithTimeout(t, func(t testing.TB) {
				helpers.AssertAPIRequest(t, testCase.ServerConfig, nil, nil, nil, nil, &helpers.APIRequest{
					Method:             http.MethodGet,
					Endpoint:           server.RuleContent,
					EndpointArgs:       []interface{}{internalTestRuleModule},
					AuthorizationToken: testCase.MockAuthToken,
				}, &helpers.APIResponse{
					StatusCode: testCase.ExpectedStatusCode,
				})
			}, testTimeout)
		})
	}
}

// TestRuleNames checks the REST API server behaviour for rules endpoint
func TestRuleNames(t *testing.T) {
	for _, testCase := range []struct {
		TestName           string
		ServerConfig       *server.Configuration
		ExpectedStatusCode int
		MockAuthToken      string
	}{
		{
			"Internal orgs enabled, no authentication",
			&serverConfigInternalOrganizations1,
			http.StatusForbidden,
			"",
		},
		{
			"Internal orgs enabled, authentication provided",
			&serverConfigInternalOrganizations1,
			http.StatusOK,
			goodJWTAuthBearer,
		},
	} {
		t.Run(testCase.TestName, func(t *testing.T) {
			helpers.RunTestWithTimeout(t, func(t testing.TB) {
				helpers.AssertAPIRequest(t, testCase.ServerConfig, nil, nil, nil, nil, &helpers.APIRequest{
					Method:             http.MethodGet,
					Endpoint:           server.RuleIDs,
					AuthorizationToken: testCase.MockAuthToken,
				}, &helpers.APIResponse{
					StatusCode: testCase.ExpectedStatusCode,
				})
			}, testTimeout)
		})
	}
}

// TestRuleNamesResponse checks the REST API status and response
func TestRuleNamesResponse(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(
		createRuleContentDirectoryFromRuleContent(
			[]types.RuleContent{RuleContentInternal1, testdata.RuleContent1},
		),
	)
	assert.Nil(t, err)

	expectedBody := `
		{
			"rules": ["ccx_rules_ocp.external.rules.node_installer_degraded", "foo.rules.internal.bar"],
			"status": "ok"
		}
	`
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		helpers.AssertAPIRequest(t, &serverConfigInternalOrganizations1, nil, nil, nil, nil, &helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.RuleIDs,
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        expectedBody,
			BodyChecker: ruleIDsChecker,
		})
	}, testTimeout)

	expectedBody = `
		{
			"rules": ["ccx_rules_ocp.external.rules.node_installer_degraded"],
			"status": "ok"
		}`
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		helpers.AssertAPIRequest(t, &serverConfigInternalOrganizations2, nil, nil, nil, nil, &helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.RuleIDs,
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        expectedBody,
			BodyChecker: ruleIDsChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_OverviewWithClusterIDsEndpoint
func TestHTTPServer_OverviewWithClusterIDsEndpoint(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		// prepare reports reponse
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ReportForListOfClustersPayloadEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(data.AggregatorReportForClusterList),
			},
		)

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:   http.MethodPost,
			Endpoint: server.OverviewEndpoint,
			OrgID:    testdata.OrgID,
			Body:     helpers.ToJSONString(data.ClusterIDListInReq),
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(OverviewResponsePostEndpoint),
		})
	}, testTimeout)
}

// TestHTTPServer_OverviewWithClusterIDsEndpoint_UnavailableContentService
func TestHTTPServer_OverviewWithClusterIDsEndpoint_UnavailableContentService(t *testing.T) {
	var emptyResponse *types.RuleContentDirectory
	err := loadMockRuleContentDir(emptyResponse)
	assert.NotNil(t, err)

	expectedBody := `
		{
		   "status" : "Content directory cache has been empty for too long time; timeout triggered"
		}`

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		// prepare reports reponse
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ReportForListOfClustersPayloadEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(data.AggregatorReportForClusterList),
			},
		)

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:   http.MethodPost,
			Endpoint: server.OverviewEndpoint,
			OrgID:    testdata.OrgID,
			Body:     helpers.ToJSONString(data.ClusterIDListInReq),
		}, &helpers.APIResponse{
			StatusCode: http.StatusServiceUnavailable,
			Body:       expectedBody,
		})
	}, testTimeout)
}

// TestHTTPServer_RecommendationsListEndpoint2Rules_ImpactingMissing
func TestHTTPServer_RecommendationsListEndpoint2Rules_ImpactingMissing(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(
		createRuleContentDirectoryFromRuleContent(
			[]types.RuleContent{testdata.RuleContent1, testdata.RuleContent2},
		),
	)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterList := make([]types.ClusterName, 2)
		for i := range clusterList {
			clusterList[i] = testdata.GetRandomClusterID()
		}

		reqBody, _ := json.Marshal(clusterList)

		respBody := `{"recommendations":{"%v":%v,"%v":%v},"status":"ok"}`
		respBody = fmt.Sprintf(respBody,
			testdata.Rule1CompositeID, 2,
			testdata.Rule2CompositeID, 1,
		)

		// prepare response from aggregator for list of clusters
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ClustersForOrganizationEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(responses.BuildOkResponseWithData("clusters", clusterList)),
		})

		// prepare response from aggregator for recommendations
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.RecommendationsListEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		ruleAcksBody := `{"disabledRules":[],"status":"ok"}`

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
				EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ruleAcksBody,
			},
		)

		helpers.AssertAPIv2Request(t, &serverConfigJWT, nil, nil, nil, nil, &helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.RecommendationsListEndpoint,
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(GetRecommendationsResponse2Rules2Clusters),
			BodyChecker: recommendationInResponseChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_RecommendationsListEndpoint2Rules_ImpactingMissing1RuleDisabled
func TestHTTPServer_RecommendationsListEndpoint2Rules_ImpactingMissing1RuleDisabled(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(
		createRuleContentDirectoryFromRuleContent(
			[]types.RuleContent{testdata.RuleContent1, testdata.RuleContent2},
		),
	)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterList := make([]types.ClusterName, 2)
		for i := range clusterList {
			clusterList[i] = testdata.GetRandomClusterID()
		}

		reqBody, _ := json.Marshal(clusterList)

		respBody := `{"recommendations":{"%v":%v,"%v":%v},"status":"ok"}`
		respBody = fmt.Sprintf(respBody,
			testdata.Rule1CompositeID, 0,
			testdata.Rule2CompositeID, 0,
		)

		// prepare response from aggregator for list of clusters
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ClustersForOrganizationEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(responses.BuildOkResponseWithData("clusters", clusterList)),
		})

		// prepare response from aggregator for recommendations
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.RecommendationsListEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		/*
			ruleAcksBody := `{
				"disabledRules":[
					{
						"org_id": 1,
						"user_id": "1",
						"rule_id": "%v",
						"error_key": "%v",
						"justification": "justification",
						"created_at": %v,
						"updated_at": %v
					}
				],
				"status":"ok"
			}`
		*/
		ruleAcksBody := `{
			"disabledRules":[
				{
					"rule_id": "%v",
					"error_key": "%v"
				}
			],
			"status":"ok"
		}`
		ruleAcksBody = fmt.Sprintf(ruleAcksBody, testdata.Rule1ID, testdata.ErrorKey1)

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
				EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ruleAcksBody,
			},
		)

		helpers.AssertAPIv2Request(t, &serverConfigJWT, nil, nil, nil, nil, &helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.RecommendationsListEndpoint,
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(GetRecommendationsResponse2Rules1Disabled0Clusters),
			BodyChecker: recommendationInResponseChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_RecommendationsListEndpoint2Rules
func TestHTTPServer_RecommendationsListEndpoint2Rules1MissingContent(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(
		createRuleContentDirectoryFromRuleContent(
			[]types.RuleContent{testdata.RuleContent1},
		),
	)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterList := make([]types.ClusterName, 2)
		for i := range clusterList {
			clusterList[i] = testdata.GetRandomClusterID()
		}

		reqBody, _ := json.Marshal(clusterList)

		respBody := `{"recommendations":{"%v":%v,"%v":%v},"status":"ok"}`
		respBody = fmt.Sprintf(respBody,
			testdata.Rule1CompositeID, 2,
			testdata.Rule2CompositeID, 1,
		)

		// prepare response from aggregator for list of clusters
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ClustersForOrganizationEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(responses.BuildOkResponseWithData("clusters", clusterList)),
		})

		// prepare response from aggregator for recommendations
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.RecommendationsListEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		ruleAcksBody := `{"disabledRules":[],"status":"ok"}`

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
				EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ruleAcksBody,
			},
		)

		helpers.AssertAPIv2Request(t, &serverConfigJWT, nil, nil, nil, nil, &helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.RecommendationsListEndpoint,
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(GetRecommendationsResponse1Rule2Cluster),
		})
	}, testTimeout)
}

// TestHTTPServer_RecommendationsListEndpoint_NoRuleContent
func TestHTTPServer_RecommendationsListEndpoint_NoRuleContent(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterList := make([]types.ClusterName, 2)
		for i := range clusterList {
			clusterList[i] = testdata.GetRandomClusterID()
		}

		reqBody, _ := json.Marshal(clusterList)

		respBody := `{"recommendations":{"%v":%v,"%v":%v,"%v":%v},"status":"ok"}`
		respBody = fmt.Sprintf(respBody,
			testdata.Rule1CompositeID, 2,
			testdata.Rule2CompositeID, 2,
			testdata.Rule3CompositeID, 1,
		)

		// prepare response from aggregator for list of clusters
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ClustersForOrganizationEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(responses.BuildOkResponseWithData("clusters", clusterList)),
		})

		// prepare response from aggregator for recommendations
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.RecommendationsListEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		ruleAcksBody := `{"disabledRules":[],"status":"ok"}`

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
				EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ruleAcksBody,
			},
		)

		helpers.AssertAPIv2Request(t, &serverConfigJWT, nil, nil, nil, nil, &helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.RecommendationsListEndpoint,
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(GetRecommendationsResponse0Rules),
		})
	}, testTimeout)
}

// TestHTTPServer_RecommendationsListEndpoint3Rules1Internal0Clusters_ImpactingTrue
func TestHTTPServer_RecommendationsListEndpoint3Rules1Internal0Clusters_ImpactingTrue(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(
		createRuleContentDirectoryFromRuleContent(
			[]types.RuleContent{testdata.RuleContent1, testdata.RuleContent2, RuleContentInternal1},
		),
	)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterList := make([]types.ClusterName, 2)
		for i := range clusterList {
			clusterList[i] = testdata.GetRandomClusterID()
		}

		reqBody, _ := json.Marshal(clusterList)

		respBody := `{"recommendations":{},"status":"ok"}`

		// prepare response from aggregator for list of clusters
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ClustersForOrganizationEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(responses.BuildOkResponseWithData("clusters", clusterList)),
		})

		// prepare response from aggregator for recommendations
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.RecommendationsListEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		ruleAcksBody := `{"disabledRules":[],"status":"ok"}`

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
				EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ruleAcksBody,
			},
		)

		helpers.AssertAPIv2Request(t, &serverConfigJWT, nil, nil, nil, nil, &helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.RecommendationsListEndpoint + "?" + server.ImpactingParam + "=true",
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(GetRecommendationsResponse0Rules),
		})
	}, testTimeout)
}

// TestHTTPServer_RecommendationsListEndpoint3Rules1Internal0Clusters_ImpactingFalse
func TestHTTPServer_RecommendationsListEndpoint3Rules1Internal0Clusters_ImpactingFalse(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(
		createRuleContentDirectoryFromRuleContent(
			[]types.RuleContent{testdata.RuleContent1, testdata.RuleContent2, RuleContentInternal1},
		),
	)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterList := make([]types.ClusterName, 2)
		for i := range clusterList {
			clusterList[i] = testdata.GetRandomClusterID()
		}

		reqBody, _ := json.Marshal(clusterList)

		respBody := `{"recommendations":{},"status":"ok"}`

		// prepare response from aggregator for list of clusters
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ClustersForOrganizationEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(responses.BuildOkResponseWithData("clusters", clusterList)),
		})

		// prepare response from aggregator for recommendations
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.RecommendationsListEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		ruleAcksBody := `{"disabledRules":[],"status":"ok"}`

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
				EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ruleAcksBody,
			},
		)

		helpers.AssertAPIv2Request(t, &serverConfigJWT, nil, nil, nil, nil, &helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.RecommendationsListEndpoint + "?" + server.ImpactingParam + "=false",
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(GetRecommendationsResponse2Rules0Clusters),
			BodyChecker: recommendationInResponseChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_RecommendationsListEndpoint2Rules1Internal2Clusters_ImpactingMissing
func TestHTTPServer_RecommendationsListEndpoint2Rules1Internal2Clusters_ImpactingMissing(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(
		createRuleContentDirectoryFromRuleContent(
			[]types.RuleContent{testdata.RuleContent1, RuleContentInternal1},
		),
	)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterList := make([]types.ClusterName, 2)
		for i := range clusterList {
			clusterList[i] = testdata.GetRandomClusterID()
		}

		reqBody, _ := json.Marshal(clusterList)

		respBody := `{"recommendations":{"%v":%v},"status":"ok"}`
		respBody = fmt.Sprintf(respBody,
			testdata.Rule1CompositeID, 2,
		)

		// prepare response from aggregator for list of clusters
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ClustersForOrganizationEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(responses.BuildOkResponseWithData("clusters", clusterList)),
		})

		// prepare response from aggregator for recommendations
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.RecommendationsListEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		ruleAcksBody := `{"disabledRules":[],"status":"ok"}`

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
				EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ruleAcksBody,
			},
		)

		helpers.AssertAPIv2Request(t, &serverConfigJWT, nil, nil, nil, nil, &helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.RecommendationsListEndpoint,
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(GetRecommendationsResponse1Rule2Cluster),
			BodyChecker: recommendationInResponseChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_RecommendationsListEndpoint4Rules1Internal2Clusters_ImpactingMissing
func TestHTTPServer_RecommendationsListEndpoint4Rules1Internal2Clusters_ImpactingMissing(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(
		createRuleContentDirectoryFromRuleContent(
			[]types.RuleContent{
				testdata.RuleContent1,
				testdata.RuleContent2,
				testdata.RuleContent3,
				RuleContentInternal1,
			},
		),
	)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterList := make([]types.ClusterName, 2)
		for i := range clusterList {
			clusterList[i] = testdata.GetRandomClusterID()
		}

		reqBody, _ := json.Marshal(clusterList)

		respBody := `{"recommendations":{"%v":%v},"status":"ok"}`
		respBody = fmt.Sprintf(respBody,
			testdata.Rule1CompositeID, 1,
		)

		// prepare response from aggregator for list of clusters
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ClustersForOrganizationEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(responses.BuildOkResponseWithData("clusters", clusterList)),
		})

		// prepare response from aggregator for recommendations
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.RecommendationsListEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		ruleAcksBody := `{"disabledRules":[],"status":"ok"}`

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
				EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ruleAcksBody,
			},
		)

		helpers.AssertAPIv2Request(t, &serverConfigJWT, nil, nil, nil, nil, &helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.RecommendationsListEndpoint,
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(GetRecommendationsResponse3Rules1Cluster),
			BodyChecker: recommendationInResponseChecker,
		})
	}, testTimeout)
}

// previously returned duplicate response, making the response JSON invalid
func TestHTTPServer_RecommendationsListEndpoint_BadToken(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		helpers.AssertAPIv2Request(t, &serverConfigJWT, nil, nil, nil, nil, &helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.RecommendationsListEndpoint,
			AuthorizationToken: badJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode: http.StatusInternalServerError,
		})
	}, testTimeout)
}

// previously returned the error from strconv.Bool == 500
func TestHTTPServer_RecommendationsListEndpoint_BadImpactingParam(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		helpers.AssertAPIv2Request(t, &serverConfigJWT, nil, nil, nil, nil, &helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.RecommendationsListEndpoint + "?" + server.ImpactingParam + "=badbool",
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode: http.StatusBadRequest,
		})
	}, testTimeout)
}

// TestHTTPServer_GetRecommendationContent
func TestHTTPServer_GetRecommendationContent(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(
		createRuleContentDirectoryFromRuleContent(
			[]types.RuleContent{testdata.RuleContent1, RuleContentInternal1},
		),
	)
	assert.Nil(t, err)

	for _, testCase := range []struct {
		TestName           string
		ServerConfig       *server.Configuration
		RuleID             types.RuleID
		ExpectedStatusCode int
		ExpectedResponse   interface{}
	}{
		{
			"ok",
			&serverConfigJWT,
			testdata.Rule1CompositeID,
			http.StatusOK,
			GetRuleContentRecommendationContent1,
		},
		{
			"internal OK",
			&serverConfigInternalOrganizations1,
			internalRuleID,
			http.StatusOK,
			nil,
		},
		{
			"internal forbidden",
			&serverConfigInternalOrganizations2,
			internalRuleID,
			http.StatusForbidden,
			nil,
		},
		{
			"not found",
			&serverConfigJWT,
			testdata.Rule5CompositeID,
			http.StatusNotFound,
			nil,
		},
		{
			"invalid rule ID",
			&serverConfigJWT,
			"invalid rule id",
			http.StatusBadRequest,
			nil,
		},
	} {
		t.Run(testCase.TestName, func(t *testing.T) {
			helpers.RunTestWithTimeout(t, func(t testing.TB) {
				var response helpers.APIResponse
				if testCase.ExpectedResponse == nil {
					response = helpers.APIResponse{
						StatusCode: testCase.ExpectedStatusCode,
					}
				} else {
					response = helpers.APIResponse{
						StatusCode: testCase.ExpectedStatusCode,
						Body:       helpers.ToJSONString(testCase.ExpectedResponse),
					}
				}

				helpers.AssertAPIv2Request(t, testCase.ServerConfig, nil, nil, nil, nil, &helpers.APIRequest{
					Method:             http.MethodGet,
					Endpoint:           server.RuleContentV2,
					EndpointArgs:       []interface{}{testCase.RuleID},
					AuthorizationToken: goodJWTAuthBearer,
				}, &response)
			}, testTimeout)
		})
	}
}

// TestHTTPServer_GetRecommendationContentWithUserData
func TestHTTPServer_GetRecommendationContentWithUserData(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(
		createRuleContentDirectoryFromRuleContent(
			[]types.RuleContent{testdata.RuleContent1, RuleContentInternal1},
		),
	)
	assert.Nil(t, err)

	for _, testCase := range []struct {
		TestName           string
		ServerConfig       *server.Configuration
		UserVote           types.UserVote
		RuleID             types.RuleID
		ExpectedStatusCode int
		ExpectedResponse   interface{}
	}{
		{
			"no vote",
			&serverConfigJWT,
			types.UserVoteNone,
			testdata.Rule1CompositeID,
			http.StatusOK,
			GetRuleContentRecommendationContentWithUserData1,
		},
		{
			"with rule like",
			&serverConfigJWT,
			types.UserVoteLike,
			testdata.Rule1CompositeID,
			http.StatusOK,
			GetRuleContentRecommendationContentWithUserData2RatingLike,
		},
		{
			"with rule dislike",
			&serverConfigJWT,
			types.UserVoteDislike,
			testdata.Rule1CompositeID,
			http.StatusOK,
			GetRuleContentRecommendationContentWithUserData3RatingDislike,
		},
		{
			"internal OK",
			&serverConfigInternalOrganizations1,
			types.UserVoteDislike,
			internalRuleID,
			http.StatusOK,
			nil,
		},
		{
			"internal forbidden",
			&serverConfigInternalOrganizations2,
			types.UserVoteDislike,
			internalRuleID,
			http.StatusForbidden,
			nil,
		},
		{
			"not found",
			&serverConfigJWT,
			types.UserVoteDislike,
			testdata.Rule5CompositeID,
			http.StatusNotFound,
			nil,
		},
		{
			"invalid rule ID",
			&serverConfigJWT,
			types.UserVoteDislike,
			"invalid rule id",
			http.StatusBadRequest,
			nil,
		},
	} {
		t.Run(testCase.TestName, func(t *testing.T) {
			helpers.RunTestWithTimeout(t, func(t testing.TB) {
				defer helpers.CleanAfterGock(t)
				rating := fmt.Sprintf(`{"rule":"%v","rating":%v}`, testCase.RuleID, testCase.UserVote)
				aggregatorResponse := fmt.Sprintf(`{"rating":%s}`, rating)

				// prepare response from aggregator for recommendations
				helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
					&helpers.APIRequest{
						Method:       http.MethodGet,
						Endpoint:     ira_server.GetRating,
						EndpointArgs: []interface{}{testCase.RuleID, testdata.OrgID, userIDOnGoodJWTAuthBearer},
					},
					&helpers.APIResponse{
						StatusCode: http.StatusOK,
						Body:       aggregatorResponse,
					},
				)

				var response helpers.APIResponse
				if testCase.ExpectedResponse == nil {
					response = helpers.APIResponse{
						StatusCode: testCase.ExpectedStatusCode,
					}
				} else {
					response = helpers.APIResponse{
						StatusCode: testCase.ExpectedStatusCode,
						Body:       helpers.ToJSONString(testCase.ExpectedResponse),
					}
				}

				helpers.AssertAPIv2Request(t, testCase.ServerConfig, nil, nil, nil, nil, &helpers.APIRequest{
					Method:             http.MethodGet,
					Endpoint:           server.RuleContentWithUserData,
					EndpointArgs:       []interface{}{testCase.RuleID},
					AuthorizationToken: goodJWTAuthBearer,
				}, &response)
			}, testTimeout)
		})
	}
}

func TestHTTPServer_GroupsEndpoint(t *testing.T) {
	groupsChannel := make(chan []groups.Group)
	errorFoundChannel := make(chan bool)
	errorChannel := make(chan error)

	records := make([]groups.Group, 1)
	go func() { groupsChannel <- records }()
	go func() { errorFoundChannel <- false }()

	expectedBody := `
		{
			"groups": [
				{
					"description": "",
					"tags": null,
					"title":""
				}
			],
			"status": "ok"
		}`
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		helpers.AssertAPIRequest(t, nil, nil, groupsChannel, errorFoundChannel, errorChannel, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: server.RuleGroupsEndpoint,
			OrgID:    testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       expectedBody,
		})
	}, testTimeout)
}

func TestHTTPServer_GroupsEndpoint_UnavailableContentService(t *testing.T) {
	groupsChannel := make(chan []groups.Group)
	errorFoundChannel := make(chan bool)
	errorChannel := make(chan error)

	go func() { errorFoundChannel <- true }()
	go func() { errorChannel <- &content.RuleContentDirectoryTimeoutError{} }()

	expectedBody := `
		{
			"status" : "Content directory cache has been empty for too long time; timeout triggered"
		}`

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		helpers.AssertAPIRequest(t, nil, nil, groupsChannel, errorFoundChannel, errorChannel, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: server.RuleGroupsEndpoint,
			OrgID:    testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusServiceUnavailable,
			Body:       expectedBody,
		})
	}, testTimeout)
}

// TestServeInfoMap checks the REST API server behaviour for info endpoint
func TestServeInfoMap(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:   http.MethodGet,
		Endpoint: "info",
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
	})
}
