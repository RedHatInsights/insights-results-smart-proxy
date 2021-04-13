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
	"fmt"
	"net/http"
	"testing"
	"time"

	ics_server "github.com/RedHatInsights/insights-content-service/server"
	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
)

// TODO: test more cases for report endpoint
func TestHTTPServer_ReportEndpoint(t *testing.T) {
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

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.ContentBaseEndpoint, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: ics_server.AllContentEndpoint,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.MustGobSerialize(t, testdata.RuleContentDirectory3Rules),
		})

		go content.RunUpdateContentLoop(helpers.DefaultServicesConfig)
		defer content.StopUpdateContentLoop()

		helpers.AssertAPIRequest(t, nil, nil, nil, &helpers.APIRequest{
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

func TestHTTPServer_ReportEndpointNoContent(t *testing.T) {
	time.Sleep(1 * time.Second)
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

		// content-service responses with 3 rules
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.ContentBaseEndpoint, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: ics_server.AllContentEndpoint,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.MustGobSerialize(t, testdata.RuleContentDirectory3Rules),
		})

		go content.RunUpdateContentLoop(helpers.DefaultServicesConfig)
		defer content.StopUpdateContentLoop()

		helpers.AssertAPIRequest(t, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       helpers.ToJSONString(SmartProxyReportResponse1RuleNoContent),
		})
	}, testTimeout)
}

func TestHTTPServer_ReportEndpoint_WithOnlyOSDEndpoint(t *testing.T) {
	time.Sleep(1 * time.Second)
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

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.ContentBaseEndpoint, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: ics_server.AllContentEndpoint,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.MustGobSerialize(t, testdata.RuleContentDirectory3Rules),
		})

		go content.RunUpdateContentLoop(helpers.DefaultServicesConfig)
		defer content.StopUpdateContentLoop()

		helpers.AssertAPIRequest(t, nil, nil, nil, &helpers.APIRequest{
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
	time.Sleep(1 * time.Second)
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

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.ContentBaseEndpoint, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: ics_server.AllContentEndpoint,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.MustGobSerialize(t, testdata.RuleContentDirectory5Rules),
		})

		go content.RunUpdateContentLoop(helpers.DefaultServicesConfig)
		defer content.StopUpdateContentLoop()

		helpers.AssertAPIRequest(t, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint + "?" + server.GetDisabledParam + "=false",
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyReportResponse3RulesOnlyEnabled),
		})
	}, testTimeout)
}

// TODO: test more cases for rule endpoint
func TestHTTPServer_RuleEndpoint(t *testing.T) {
	time.Sleep(1 * time.Second)
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

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.ContentBaseEndpoint, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: ics_server.AllContentEndpoint,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.MustGobSerialize(t, testdata.RuleContentDirectory3Rules),
		})

		go content.RunUpdateContentLoop(helpers.DefaultServicesConfig)
		defer content.StopUpdateContentLoop()

		helpers.AssertAPIRequest(t, nil, nil, nil, &helpers.APIRequest{
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

func TestHTTPServer_RuleEndpoint_WithOSD(t *testing.T) {
	time.Sleep(1 * time.Second)
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

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.ContentBaseEndpoint, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: ics_server.AllContentEndpoint,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.MustGobSerialize(t, testdata.RuleContentDirectory3Rules),
		})

		go content.RunUpdateContentLoop(helpers.DefaultServicesConfig)
		defer content.StopUpdateContentLoop()

		helpers.AssertAPIRequest(t, nil, nil, nil, &helpers.APIRequest{
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
	time.Sleep(1 * time.Second)
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

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.ContentBaseEndpoint, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: ics_server.AllContentEndpoint,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.MustGobSerialize(t, testdata.RuleContentDirectory3Rules),
		})

		go content.RunUpdateContentLoop(helpers.DefaultServicesConfig)
		defer content.StopUpdateContentLoop()

		helpers.AssertAPIRequest(t, nil, nil, nil, &helpers.APIRequest{
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
	content.ResetContent()
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)
		// Setup Content
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.ContentBaseEndpoint, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: ics_server.AllContentEndpoint,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.MustGobSerialize(t, testdata.RuleContentDirectory3Rules),
		})

		go content.RunUpdateContentLoop(helpers.DefaultServicesConfig)
		defer content.StopUpdateContentLoop()

		helpers.AssertAPIRequest(t, nil, nil, nil, &helpers.APIRequest{
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
	time.Sleep(1 * time.Second)
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		// prepare content
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.ContentBaseEndpoint, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: ics_server.AllContentEndpoint,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.MustGobSerialize(t, testdata.RuleContentDirectory3Rules),
		})

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

		go content.RunUpdateContentLoop(helpers.DefaultServicesConfig)
		defer content.StopUpdateContentLoop()

		helpers.AssertAPIRequest(t, nil, nil, nil, &helpers.APIRequest{
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

func TestInternalOrganizations(t *testing.T) {
	loadMockRuleContentDir([]types.RuleContent{RuleContentInternal1})

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
				helpers.AssertAPIRequest(t, testCase.ServerConfig, nil, nil, &helpers.APIRequest{
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
				helpers.AssertAPIRequest(t, testCase.ServerConfig, nil, nil, &helpers.APIRequest{
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
	content.ResetContent()
	loadMockRuleContentDir([]types.RuleContent{RuleContentInternal1, testdata.RuleContent1})

	expectedBody := `
		{
			"rules": ["ccx_rules_ocp.external.rules.node_installer_degraded", "foo.rules.internal.bar"],
			"status": "ok"
		}
	`
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		helpers.AssertAPIRequest(t, &serverConfigInternalOrganizations1, nil, nil, &helpers.APIRequest{
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
		helpers.AssertAPIRequest(t, &serverConfigInternalOrganizations2, nil, nil, &helpers.APIRequest{
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
