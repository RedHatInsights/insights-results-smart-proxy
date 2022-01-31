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
	"time"

	"github.com/RedHatInsights/insights-content-service/groups"
	iou_helpers "github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
	data "github.com/RedHatInsights/insights-results-smart-proxy/tests/testdata"
	"github.com/RedHatInsights/insights-results-smart-proxy/types"
	ctypes "github.com/RedHatInsights/insights-results-types"
	"github.com/stretchr/testify/assert"
)

var (
	RuleContentDirectoryOnly1Rule = ctypes.RuleContentDirectory{
		Config: ctypes.GlobalRuleConfig{
			Impact: testdata.ImpactStrToInt,
		},
		Rules: map[string]ctypes.RuleContent{
			"rc1": testdata.RuleContent1,
		},
	}

	RuleContentDirectoryOnlyDisabledRule = ctypes.RuleContentDirectory{
		Config: ctypes.GlobalRuleConfig{
			Impact: testdata.ImpactStrToInt,
		},
		Rules: map[string]ctypes.RuleContent{
			"rc5": testdata.RuleContent5,
		},
	}

	v1Report1RuleData = []types.RuleWithContentResponse{
		{
			RuleID:       testdata.Rule1.Module,
			ErrorKey:     testdata.RuleErrorKey1.ErrorKey,
			CreatedAt:    testdata.RuleErrorKey1.PublishDate.UTC().Format(time.RFC3339),
			Description:  testdata.RuleErrorKey1.Description,
			Generic:      testdata.RuleErrorKey1.Generic,
			Reason:       testdata.RuleErrorKey1.Reason,
			Resolution:   testdata.RuleErrorKey1.Resolution,
			MoreInfo:     testdata.RuleErrorKey1.MoreInfo,
			TotalRisk:    calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood),
			RiskOfChange: 0,
			Disabled:     testdata.Rule1Disabled,
			UserVote:     types.UserVoteNone,
			TemplateData: testdata.Rule1ExtraData,
			Tags:         testdata.RuleErrorKey1.Tags,
		},
	}

	SmartProxyReportRule1 = struct {
		Status string                    `json:"status"`
		Report *types.SmartProxyReportV1 `json:"report"`
	}{
		Status: "ok",
		Report: &types.SmartProxyReportV1{
			Meta: types.ReportResponseMetaV1{
				Count:         1,
				LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
			},
			Data: v1Report1RuleData,
		},
	}

	ResponseNoRulesDisabledSystemWide = `{
		"meta": {
			"count": 0
		},
		"data": []
	}`

	ResponseRule2DisabledSystemWide = struct {
		Status      string                         `json:"status"`
		RuleDisable []ctypes.SystemWideRuleDisable `json:"disabledRules"`
	}{
		Status: "ok",
		RuleDisable: []ctypes.SystemWideRuleDisable{
			{
				OrgID:         testdata.OrgID,
				UserID:        testdata.UserID,
				RuleID:        testdata.Rule2ID,
				ErrorKey:      testdata.ErrorKey2,
				Justification: "Rule 2 disabled for testing purposes",
			},
		},
	}

	v1Report1RuleNoContent = types.SmartProxyReportV1{
		Meta: types.ReportResponseMetaV1{
			Count:         0,
			LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		},
		Data: []types.RuleWithContentResponse{},
	}

	SmartProxyV1ReportResponse1RuleNoContent = struct {
		Status string                    `json:"status"`
		Report *types.SmartProxyReportV1 `json:"report"`
	}{
		Status: "ok",
		Report: &v1Report1RuleNoContent,
	}

	v1ReportEmptyCount2 = types.SmartProxyReportV1{
		Meta: types.ReportResponseMetaV1{
			Count:         2,
			LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		},
		Data: []types.RuleWithContentResponse{},
	}

	SmartProxyV1EmptyResponseDisabledRulesMissingContent = struct {
		Status string                    `json:"status"`
		Report *types.SmartProxyReportV1 `json:"report"`
	}{
		Status: "ok",
		Report: &v1ReportEmptyCount2,
	}

	v1Report3Rules = types.SmartProxyReportV1{
		Meta: types.ReportResponseMetaV1{
			Count:         3,
			LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		},
		Data: Report3RulesData,
	}

	SmartProxyV1ReportResponse3Rules = struct {
		Status string                    `json:"status"`
		Report *types.SmartProxyReportV1 `json:"report"`
	}{
		Status: "ok",
		Report: &v1Report3Rules,
	}

	v1Report3Rules2NoContent = types.SmartProxyReportV1{
		Meta: types.ReportResponseMetaV1{
			Count:         3,
			LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		},
		Data: Report3Rules2NoContentData,
	}
	SmartProxyV1ReportResponse3Rules2NoContent = struct {
		Status string                    `json:"status"`
		Report *types.SmartProxyReportV1 `json:"report"`
	}{
		Status: "ok",
		Report: &v1Report3Rules2NoContent,
	}

	v1Report3RulesWithOnlyOSD = types.SmartProxyReportV1{
		Meta: types.ReportResponseMetaV1{
			Count:         1,
			LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		},
		Data: Report3RulesWithOnlyOSDData,
	}

	SmartProxyV1ReportResponse3RulesWithOnlyOSD = struct {
		Status string                    `json:"status"`
		Report *types.SmartProxyReportV1 `json:"report"`
	}{
		Status: "ok",
		Report: &v1Report3RulesWithOnlyOSD,
	}

	v1ReportResponse3RulesOnlyEnabled = types.SmartProxyReportV1{
		Meta: types.ReportResponseMetaV1{
			Count:         2,
			LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		},
		Data: Report3RulesOnlyEnabledData,
	}

	SmartProxyV1ReportResponse3RulesOnlyEnabled = struct {
		Status string                    `json:"status"`
		Report *types.SmartProxyReportV1 `json:"report"`
	}{
		Status: "ok",
		Report: &v1ReportResponse3RulesOnlyEnabled,
	}

	v1ReportResponse3RulesWithDisabled = types.SmartProxyReportV1{
		Meta: types.ReportResponseMetaV1{
			Count:         3,
			LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		},
		Data: Report3RulesWithDisabledData,
	}

	SmartProxyV1ReportResponse3RulesWithDisabled = struct {
		Status string                    `json:"status"`
		Report *types.SmartProxyReportV1 `json:"report"`
	}{
		Status: "ok",
		Report: &v1ReportResponse3RulesWithDisabled,
	}

	v1Report3RulesWithDisabled = types.SmartProxyReportV1{
		Meta: types.ReportResponseMetaV1{
			Count:         3,
			LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		},
		Data: Report3RulesWithDisabledData,
	}

	SmartProxyV1ReportResponse3RulesAll = struct {
		Status string                    `json:"status"`
		Report *types.SmartProxyReportV1 `json:"report"`
	}{
		Status: "ok",
		Report: &v1Report3RulesWithDisabled,
	}
)

func expectNoRulesDisabledSystemWide(t *testing.TB) {
	helpers.GockExpectAPIRequest(*t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       ResponseNoRulesDisabledSystemWide,
	})
}

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

		expectNoRulesDisabledSystemWide(&t)

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyV1ReportResponse3Rules),
		})
	}, testTimeout)
}

func TestHTTPServer_ReportEndpoint_UnavailableContentService(t *testing.T) {
	var emptyResponse *ctypes.RuleContentDirectory
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

		expectNoRulesDisabledSystemWide(&t)

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

		expectNoRulesDisabledSystemWide(&t)

		// previously was InternalServerError, but it was changed as an edge-case which will appear as "No issues found"
		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyV1ReportResponse1RuleNoContent),
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

		expectNoRulesDisabledSystemWide(&t)

		// 1 rule returned, but count = 3
		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyV1ReportResponse3Rules2NoContent),
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

		expectNoRulesDisabledSystemWide(&t)

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint + "?" + server.OSDEligibleParam + "=true",
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyV1ReportResponse3RulesWithOnlyOSD),
		})
	}, testTimeout)
}

func TestHTTPServer_ReportEndpoint_WithDisabledRulesForCluster(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory5Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		for i := 0; i < 3; i++ {
			helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ReportEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       testdata.Report3Rules1DisabledExpectedResponse,
			})

			expectNoRulesDisabledSystemWide(&t)
		}

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint + "?" + server.GetDisabledParam + "=false",
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyV1ReportResponse3RulesOnlyEnabled),
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
			Body:       helpers.ToJSONString(SmartProxyV1ReportResponse3RulesOnlyEnabled),
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
			Body:       helpers.ToJSONString(SmartProxyV1ReportResponse3RulesAll),
		})
	}, testTimeout)
}

func TestHTTPServer_ReportEndpoint_WithDisabledRulesForClusterAndMissingContent(t *testing.T) {
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

		expectNoRulesDisabledSystemWide(&t)

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyV1EmptyResponseDisabledRulesMissingContent),
		})
	}, testTimeout)
}

func TestHTTPServer_ReportEndpoint_WithClusterAndSystemWideDisabledRules(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory5Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)
		for i := 0; i < 3; i++ {
			helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ReportEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       testdata.Report3Rules1DisabledExpectedResponse,
			})

			helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
				EndpointArgs: []interface{}{testdata.OrgID, testdata.UserID},
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(ResponseRule2DisabledSystemWide),
			})
		}
		// Get report with get_disabled = false
		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint + "?" + server.GetDisabledParam + "=false",
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyReportRule1),
		})

		// Get report without specifying get_disabled => same result as above
		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyReportRule1),
		})

		// Get report with get_disabled = true
		// => Report contains disabled rules for cluster and org-wide disabled rules
		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint + "?" + server.GetDisabledParam + "=true",
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyV1ReportResponse3RulesAll),
		})
	}, testTimeout)
}

// TestHTTPServer_ReportMetainfoEndpointNoReports check the /report/info
// endpoint when no results are found for given cluster.
func TestHTTPServer_ReportMetainfoEndpointNoReports(t *testing.T) {
	const metainfoResponse = `
		{
		  "metainfo": {
		    "count": -1,
		    "last_checked_at": "1970-01-01T00:00:25Z",
		    "stored_at": "1970-01-01T00:00:25Z"
		  },
		  "status": "ok"
		}`

	defer content.ResetContent()
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)
		// prepare mocked REST API response from Aggregator
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportMetainfoEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       metainfoResponse,
		})

		// check the Smart Proxy report/info endpoint
		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportMetainfoEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(ReportMetainfoAPIResponseNoReports),
		})
	}, testTimeout)
}

// TestHTTPServer_ReportMetainfoEndpointTwoReports check the /report/info
// endpoint when two results are found for given cluster.
func TestHTTPServer_ReportMetainfoEndpointTwoReports(t *testing.T) {
	const metainfoResponse = `
		{
		  "metainfo": {
		    "count": 2,
		    "last_checked_at": "1970-01-01T00:00:25Z",
		    "stored_at": "1970-01-01T00:00:25Z"
		  },
		  "status": "ok"
		}`

	defer content.ResetContent()
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)
		// prepare mocked REST API response from Aggregator
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportMetainfoEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       metainfoResponse,
		})

		// check the Smart Proxy report/info endpoint
		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportMetainfoEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(ReportMetainfoAPIResponseTwoReports),
		})
	}, testTimeout)
}

// TestHTTPServer_ReportMetainfoEndpointForbidden checks how HTTP codes are
// handled in report/info endpoint handler.
func TestHTTPServer_ReportMetainfoEndpointForbidden(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportMetainfoEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusForbidden,
			Body:       "",
		})

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportMetainfoEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusForbidden,
		})
	}, testTimeout)
}

// TestHTTPServer_ReportMetainfoEndpointImproperJSON check the /report/info
// endpoint when improper response is returned from Aggregator REST API.
func TestHTTPServer_ReportMetainfoEndpointImproperJSON(t *testing.T) {
	const metainfoResponse = "THIS_IS_NOT_JSON"

	defer content.ResetContent()
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)
		// prepare mocked REST API response from Aggregator
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportMetainfoEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       metainfoResponse,
		})

		// check the Smart Proxy report/info endpoint
		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportMetainfoEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusBadRequest,
			Body:       helpers.ToJSONString(ReportMetainfoAPIResponseInvalidJSON),
		})
	}, testTimeout)
}

// TestHTTPServer_ReportMetainfoEndpointWrongClusterName check the /report/info
// endpoint for incorrect input
func TestHTTPServer_ReportMetainfoEndpointWrongClusterName(t *testing.T) {
	const metainfoResponse = `
		{
		  "metainfo": {
		    "count": 2,
		    "last_checked_at": "1970-01-01T00:00:25Z",
		    "stored_at": "1970-01-01T00:00:25Z"
		  },
		  "status": "ok"
		}`

	const clusterName = "not-proper-cluster-name"

	defer content.ResetContent()
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)
		// prepare mocked REST API response from Aggregator
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportMetainfoEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, clusterName, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       metainfoResponse,
		})

		// check the Smart Proxy report/info endpoint
		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportMetainfoEndpoint,
			EndpointArgs: []interface{}{clusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusBadRequest,
			Body:       helpers.ToJSONString(ReportMetainfoAPIResponseInvalidClusterName),
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
	var emptyResponse *ctypes.RuleContentDirectory
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
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			data.ClusterInfoResult,
		)

		expectNoRulesDisabledSystemWide(&t)

		// prepare report for cluster
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report3RulesExpectedResponse,
		})

		testServer := helpers.CreateHTTPServer(nil, nil, amsClientMock, nil, nil, nil)
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			helpers.DefaultServerConfig.APIv1Prefix,
			&helpers.APIRequest{
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

// TestHTTPServer_OverviewEndpoint_UnavailableContentService
func TestHTTPServer_OverviewEndpoint_UnavailableContentService(t *testing.T) {
	var emptyResponse *ctypes.RuleContentDirectory
	err := loadMockRuleContentDir(emptyResponse)
	assert.NotNil(t, err)

	expectedBody := `
		{
		   "status" : "Content directory cache has been empty for too long time; timeout triggered"
		}
	`

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			data.ClusterInfoResult,
		)

		expectNoRulesDisabledSystemWide(&t)

		// prepare report for cluster
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report3RulesExpectedResponse,
		})

		testServer := helpers.CreateHTTPServer(nil, nil, amsClientMock, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, helpers.DefaultServerConfig.APIv1Prefix, &helpers.APIRequest{
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
			[]ctypes.RuleContent{RuleContentInternal1},
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
			"Internal organizations enabled, Request denied due to wrong OrgID",
			&serverConfigInternalOrganizations1,
			http.StatusForbidden,
			badJWTAuthBearer,
		},
		{
			"Internal organizations enabled, Request denied due to unparsable token",
			&serverConfigInternalOrganizations1,
			http.StatusForbidden,
			unparsableJWTAuthBearer,
		},
		{
			// This scenario is managed by 3scale, we don't need to check if the token is complete
			"Internal organizations enabled, Request allowed even with incomplete token",
			&serverConfigInternalOrganizations1,
			http.StatusOK,
			incompleteJWTAuthBearer,
		},
		{
			"Internal organizations enabled, Request denied due to invalid type in token",
			&serverConfigInternalOrganizations1,
			http.StatusForbidden,
			invalidJWTAuthBearer,
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
			}, testTimeout*100)
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
			[]ctypes.RuleContent{RuleContentInternal1, testdata.RuleContent1},
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

		// prepare reports response
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

		expectNoRulesDisabledSystemWide(&t)

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:   http.MethodPost,
			Endpoint: server.OverviewEndpoint,
			OrgID:    testdata.OrgID,
			UserID:   testdata.UserID,
			Body:     helpers.ToJSONString(data.ClusterIDListInReq),
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(OverviewResponsePostEndpoint),
		})
	}, testTimeout)
}

// TestHTTPServer_OverviewWithClusterIDsEndpoint_UnavailableContentService
func TestHTTPServer_OverviewWithClusterIDsEndpoint_UnavailableContentService(t *testing.T) {
	var emptyResponse *ctypes.RuleContentDirectory
	err := loadMockRuleContentDir(emptyResponse)
	assert.NotNil(t, err)

	expectedBody := `
		{
		   "status" : "Content directory cache has been empty for too long time; timeout triggered"
		}`

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		// prepare reports response
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

		expectNoRulesDisabledSystemWide(&t)

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:   http.MethodPost,
			Endpoint: server.OverviewEndpoint,
			OrgID:    testdata.OrgID,
			UserID:   testdata.UserID,
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
			[]ctypes.RuleContent{testdata.RuleContent1, testdata.RuleContent2},
		),
	)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := make([]types.ClusterInfo, 2)
		for i := range clusterInfoList {
			clusterInfoList[i] = data.GetRandomClusterInfo()
		}

		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		respBody := `{"recommendations":{"%v":%v,"%v":%v},"status":"ok"}`
		respBody = fmt.Sprintf(respBody,
			testdata.Rule1CompositeID, 2,
			testdata.Rule2CompositeID, 1,
		)

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigJWT.APIv2Prefix, &helpers.APIRequest{
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
			[]ctypes.RuleContent{testdata.RuleContent1, testdata.RuleContent2},
		),
	)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := make([]types.ClusterInfo, 2)
		for i := range clusterInfoList {
			clusterInfoList[i] = data.GetRandomClusterInfo()
		}

		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		respBody := `{"recommendations":{"%v":%v,"%v":%v},"status":"ok"}`
		respBody = fmt.Sprintf(respBody,
			testdata.Rule1CompositeID, 0,
			testdata.Rule2CompositeID, 0,
		)

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigJWT.APIv2Prefix, &helpers.APIRequest{
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

// TestHTTPServer_RecommendationsListEndpoint2Rules1MissingContent
func TestHTTPServer_RecommendationsListEndpoint2Rules1MissingContent(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(
		createRuleContentDirectoryFromRuleContent(
			[]ctypes.RuleContent{testdata.RuleContent1},
		),
	)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := make([]types.ClusterInfo, 2)
		for i := range clusterInfoList {
			clusterInfoList[i] = data.GetRandomClusterInfo()
		}

		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		respBody := `{"recommendations":{"%v":%v,"%v":%v},"status":"ok"}`
		respBody = fmt.Sprintf(respBody,
			testdata.Rule1CompositeID, 2,
			testdata.Rule2CompositeID, 1,
		)

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigJWT.APIv2Prefix, &helpers.APIRequest{
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

		clusterInfoList := make([]types.ClusterInfo, 2)
		for i := range clusterInfoList {
			clusterInfoList[i] = data.GetRandomClusterInfo()
		}

		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		respBody := `{"recommendations":{"%v":%v,"%v":%v,"%v":%v},"status":"ok"}`
		respBody = fmt.Sprintf(respBody,
			testdata.Rule1CompositeID, 2,
			testdata.Rule2CompositeID, 2,
			testdata.Rule3CompositeID, 1,
		)

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigJWT.APIv2Prefix, &helpers.APIRequest{
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
			[]ctypes.RuleContent{testdata.RuleContent1, testdata.RuleContent2, RuleContentInternal1},
		),
	)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := make([]types.ClusterInfo, 2)
		for i := range clusterInfoList {
			clusterInfoList[i] = data.GetRandomClusterInfo()
		}

		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		respBody := `{"recommendations":{},"status":"ok"}`

		// prepare response from amsClient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigJWT.APIv2Prefix, &helpers.APIRequest{
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
			[]ctypes.RuleContent{testdata.RuleContent1, testdata.RuleContent2, RuleContentInternal1},
		),
	)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := make([]types.ClusterInfo, 2)
		for i := range clusterInfoList {
			clusterInfoList[i] = data.GetRandomClusterInfo()
		}

		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		respBody := `{"recommendations":{},"status":"ok"}`

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigJWT.APIv2Prefix, &helpers.APIRequest{
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
			[]ctypes.RuleContent{testdata.RuleContent1, RuleContentInternal1},
		),
	)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := make([]types.ClusterInfo, 2)
		for i := range clusterInfoList {
			clusterInfoList[i] = data.GetRandomClusterInfo()
		}

		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		respBody := `{"recommendations":{"%v":%v},"status":"ok"}`
		respBody = fmt.Sprintf(respBody,
			testdata.Rule1CompositeID, 2,
		)

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigJWT.APIv2Prefix, &helpers.APIRequest{
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
			[]ctypes.RuleContent{
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

		clusterInfoList := make([]types.ClusterInfo, 2)
		for i := range clusterInfoList {
			clusterInfoList[i] = data.GetRandomClusterInfo()
		}

		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		respBody := `{"recommendations":{"%v":%v},"status":"ok"}`
		respBody = fmt.Sprintf(respBody,
			testdata.Rule1CompositeID, 1,
		)

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigJWT.APIv2Prefix, &helpers.APIRequest{
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
			[]ctypes.RuleContent{testdata.RuleContent1, RuleContentInternal1},
		),
	)
	assert.Nil(t, err)

	for _, testCase := range []struct {
		TestName           string
		ServerConfig       *server.Configuration
		RuleID             ctypes.RuleID
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
			[]ctypes.RuleContent{testdata.RuleContent1, RuleContentInternal1},
		),
	)
	assert.Nil(t, err)

	for _, testCase := range []struct {
		TestName           string
		ServerConfig       *server.Configuration
		UserVote           ctypes.UserVote
		RuleID             ctypes.RuleID
		ExpectedStatusCode int
		ExpectedResponse   interface{}
	}{
		{
			"no vote",
			&serverConfigJWT,
			ctypes.UserVoteNone,
			testdata.Rule1CompositeID,
			http.StatusOK,
			GetRuleContentRecommendationContentWithUserData1,
		},
		{
			"with rule like",
			&serverConfigJWT,
			ctypes.UserVoteLike,
			testdata.Rule1CompositeID,
			http.StatusOK,
			GetRuleContentRecommendationContentWithUserData2RatingLike,
		},
		{
			"with rule dislike",
			&serverConfigJWT,
			ctypes.UserVoteDislike,
			testdata.Rule1CompositeID,
			http.StatusOK,
			GetRuleContentRecommendationContentWithUserData3RatingDislike,
		},
		{
			"internal OK",
			&serverConfigInternalOrganizations1,
			ctypes.UserVoteDislike,
			internalRuleID,
			http.StatusOK,
			nil,
		},
		{
			"internal forbidden",
			&serverConfigInternalOrganizations2,
			ctypes.UserVoteDislike,
			internalRuleID,
			http.StatusForbidden,
			nil,
		},
		{
			"not found",
			&serverConfigJWT,
			ctypes.UserVoteDislike,
			testdata.Rule5CompositeID,
			http.StatusNotFound,
			nil,
		},
		{
			"invalid rule ID",
			&serverConfigJWT,
			ctypes.UserVoteDislike,
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
				ruleID, errorKey, _ := types.RuleIDWithErrorKeyFromCompositeRuleID(testCase.RuleID)

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

				// prepare response from aggregator for ack status get
				helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
					&helpers.APIRequest{
						Method:       http.MethodGet,
						Endpoint:     ira_server.ReadRuleSystemWide,
						EndpointArgs: []interface{}{ruleID, errorKey, testdata.OrgID, userIDOnGoodJWTAuthBearer},
					},
					&helpers.APIResponse{
						StatusCode: http.StatusNotFound,
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

// TestHTTPServer_ClustersRecommendationsEndpoint_NoClusters tests no clusters received from AMS API
func TestHTTPServer_ClustersRecommendationsEndpoint_NoClusters(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := make([]types.ClusterInfo, 0)
		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		respBody := `{
			"clusters":{}
		}`

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		// prepare response from aggregator
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ClustersRecommendationsListEndpoint,
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

		disabledRulesBody := `{"rules":[],"status":"ok"}`
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ListOfDisabledRules,
				EndpointArgs: []interface{}{userIDOnGoodJWTAuthBearer},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       disabledRulesBody,
			},
		)

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigJWT.APIv2Prefix, &helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.ClustersRecommendationsEndpoint,
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(GetClustersResponse0Clusters),
		})
	}, testTimeout)
}

// TestHTTPServer_ClustersRecommendationsEndpoint_ClustersFoundNoInsights tests clusters received from AMS API, but we dont have them
// == last_checked_at will be empty
func TestHTTPServer_ClustersRecommendationsEndpoint_ClustersFoundNoInsights(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := make([]types.ClusterInfo, 2)
		for i := range clusterInfoList {
			clusterInfoList[i] = data.GetRandomClusterInfo()
		}

		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		respBody := `{
			"clusters":{}
		}`

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		// prepare response from aggregator
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ClustersRecommendationsListEndpoint,
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

		disabledRulesBody := `{"rules":[],"status":"ok"}`
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ListOfDisabledRules,
				EndpointArgs: []interface{}{userIDOnGoodJWTAuthBearer},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       disabledRulesBody,
			},
		)

		resp := GetClustersResponse2ClusterNoHits
		for i := range clusterInfoList {
			resp.Clusters[i].ClusterID = clusterInfoList[i].ID
			resp.Clusters[i].ClusterName = clusterInfoList[i].DisplayName
			resp.Clusters[i].LastCheckedAt = "" // will be empty because we don't have the cluster in our DB
		}

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigJWT.APIv2Prefix, &helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.ClustersRecommendationsEndpoint,
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(resp),
			BodyChecker: clusterInResponseChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_ClustersRecommendationsEndpoint_NoRuleHits tests clusters received from AMS API, but no rule hits
func TestHTTPServer_ClustersRecommendationsEndpoint_NoRuleHits(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := make([]types.ClusterInfo, 2)
		for i := range clusterInfoList {
			clusterInfoList[i] = data.GetRandomClusterInfo()
		}

		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		respBody := `{
			"clusters":{
				"%v": {
					"created_at": "%v",
					"recommendations": []
				},
				"%v": {
					"created_at": "%v",
					"recommendations": []
				}
			}
		}`
		respBody = fmt.Sprintf(respBody,
			clusterInfoList[0].ID, testTimeStr,
			clusterInfoList[1].ID, testTimeStr,
		)

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		// prepare response from aggregator
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ClustersRecommendationsListEndpoint,
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

		disabledRulesBody := `{"rules":[],"status":"ok"}`
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ListOfDisabledRules,
				EndpointArgs: []interface{}{userIDOnGoodJWTAuthBearer},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       disabledRulesBody,
			},
		)

		resp := GetClustersResponse2ClusterNoHits
		for i := range clusterInfoList {
			resp.Clusters[i].ClusterID = clusterInfoList[i].ID
			resp.Clusters[i].ClusterName = clusterInfoList[i].DisplayName
			resp.Clusters[i].LastCheckedAt = testTimeStr
		}

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigJWT.APIv2Prefix, &helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.ClustersRecommendationsEndpoint,
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(resp),
			BodyChecker: clusterInResponseChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_ClustersRecommendationsEndpoint_2ClustersFilled tests clusters received from AMS API with rule hits
func TestHTTPServer_ClustersRecommendationsEndpoint_2ClustersFilled(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(
		createRuleContentDirectoryFromRuleContent(
			[]ctypes.RuleContent{
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

		clusterInfoList := make([]types.ClusterInfo, 2)
		for i := range clusterInfoList {
			clusterInfoList[i] = data.GetRandomClusterInfo()
		}

		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		// 3 total rules, only 2 unique total risks
		respBody := `{
			"clusters":{
				"%v": {
					"created_at": "%v",
					"recommendations": ["%v"]
				},
				"%v": {
					"created_at": "%v",
					"recommendations": ["%v","%v"]
				}
			}
		}`
		respBody = fmt.Sprintf(respBody,
			clusterInfoList[0].ID, testTimeStr, testdata.Rule1CompositeID, // total_risk = 1
			clusterInfoList[1].ID, testTimeStr, testdata.Rule2CompositeID, testdata.Rule3CompositeID, // total_risk = 2, 2
		)

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		// prepare response from aggregator
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ClustersRecommendationsListEndpoint,
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

		disabledRulesBody := `{"rules":[],"status":"ok"}`
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ListOfDisabledRules,
				EndpointArgs: []interface{}{userIDOnGoodJWTAuthBearer},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       disabledRulesBody,
			},
		)

		resp := GetClustersResponse2ClusterWithHits
		for i := range clusterInfoList {
			resp.Clusters[i].ClusterID = clusterInfoList[i].ID
			resp.Clusters[i].ClusterName = clusterInfoList[i].DisplayName
		}

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigJWT.APIv2Prefix, &helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.ClustersRecommendationsEndpoint,
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(resp),
			BodyChecker: clusterInResponseChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_ClustersRecommendationsEndpoint_AckedRule tests clusters with an acked rule hitting both
func TestHTTPServer_ClustersRecommendationsEndpoint_AckedRule(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(
		createRuleContentDirectoryFromRuleContent(
			[]ctypes.RuleContent{
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

		clusterInfoList := make([]types.ClusterInfo, 2)
		for i := range clusterInfoList {
			clusterInfoList[i] = data.GetRandomClusterInfo()
		}

		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		// 3 total rules, rule 1 hitting both clusters
		respBody := `{
			"clusters":{
				"%v": {
					"created_at": "%v",
					"recommendations": ["%v"]
				},
				"%v": {
					"created_at": "%v",
					"recommendations": ["%v","%v"]
				}
			}
		}`
		respBody = fmt.Sprintf(respBody,
			clusterInfoList[0].ID, testTimeStr, testdata.Rule1CompositeID,
			clusterInfoList[1].ID, testTimeStr, testdata.Rule1CompositeID, testdata.Rule2CompositeID,
		)

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		// prepare response from aggregator
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ClustersRecommendationsListEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

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

		disabledRulesBody := `{"rules":[],"status":"ok"}`
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ListOfDisabledRules,
				EndpointArgs: []interface{}{userIDOnGoodJWTAuthBearer},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       disabledRulesBody,
			},
		)

		resp := GetClustersResponse2ClusterWithHits1Rule
		for i := range clusterInfoList {
			resp.Clusters[i].ClusterID = clusterInfoList[i].ID
			resp.Clusters[i].ClusterName = clusterInfoList[i].DisplayName
		}

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigJWT.APIv2Prefix, &helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.ClustersRecommendationsEndpoint,
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(resp),
			BodyChecker: clusterInResponseChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_ClustersRecommendationsEndpoint_DisabledRuleSingleCluster tests clusters with a disabled rule on one of them
func TestHTTPServer_ClustersRecommendationsEndpoint_DisabledRuleSingleCluster(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(
		createRuleContentDirectoryFromRuleContent(
			[]ctypes.RuleContent{
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

		clusterInfoList := make([]types.ClusterInfo, 2)
		for i := range clusterInfoList {
			clusterInfoList[i] = data.GetRandomClusterInfo()
		}

		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		// 3 total rules, rule 1 hitting both clusters
		respBody := `{
			"clusters":{
				"%v": {
					"created_at": "%v",
					"recommendations": ["%v"]
				},
				"%v": {
					"created_at": "%v",
					"recommendations": ["%v","%v"]
				}
			}
		}`
		respBody = fmt.Sprintf(respBody,
			clusterInfoList[0].ID, testTimeStr, testdata.Rule1CompositeID,
			clusterInfoList[1].ID, testTimeStr, testdata.Rule1CompositeID, testdata.Rule2CompositeID,
		)

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		// prepare response from aggregator
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ClustersRecommendationsListEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		// acks empty
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

		// rule 1 disabled for only one cluster
		disabledRulesBody := `{
			"rules":[
				{
					"ClusterID": "%v",
					"RuleID": "%v",
					"ErrorKey": "%v"
				}
			],
			"status":"ok"
		}`
		disabledRulesBody = fmt.Sprintf(disabledRulesBody, clusterInfoList[1].ID, testdata.Rule1ID, testdata.ErrorKey1)

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ListOfDisabledRules,
				EndpointArgs: []interface{}{userIDOnGoodJWTAuthBearer},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       disabledRulesBody,
			},
		)

		resp := GetClustersResponse2ClusterWithHits1RuleDisabled
		for i := range clusterInfoList {
			resp.Clusters[i].ClusterID = clusterInfoList[i].ID
			resp.Clusters[i].ClusterName = clusterInfoList[i].DisplayName
		}

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigJWT.APIv2Prefix, &helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.ClustersRecommendationsEndpoint,
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(resp),
			BodyChecker: clusterInResponseChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_ClustersRecommendationsEndpoint_DisabledAndAcked tests clusters with a disabled rule on one of them and another acked rule
func TestHTTPServer_ClustersRecommendationsEndpoint_DisabledAndAcked(t *testing.T) {
	defer content.ResetContent()
	err := loadMockRuleContentDir(
		createRuleContentDirectoryFromRuleContent(
			[]ctypes.RuleContent{
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

		clusterInfoList := make([]types.ClusterInfo, 2)
		for i := range clusterInfoList {
			clusterInfoList[i] = data.GetRandomClusterInfo()
		}

		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		// 3 total rules, rule 1 hitting both clusters
		respBody := `{
			"clusters":{
				"%v": {
					"created_at": "%v",
					"recommendations": ["%v, %v"]
				},
				"%v": {
					"created_at": "%v",
					"recommendations": ["%v","%v"]
				}
			}
		}`
		respBody = fmt.Sprintf(respBody,
			clusterInfoList[0].ID, testTimeStr, testdata.Rule1CompositeID, testdata.Rule2CompositeID,
			clusterInfoList[1].ID, testTimeStr, testdata.Rule1CompositeID, testdata.Rule2CompositeID,
		)

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		// prepare response from aggregator
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ClustersRecommendationsListEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID, userIDOnGoodJWTAuthBearer},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		// rule 1 acked
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

		// rule 2 disabled for only 1st cluster
		disabledRulesBody := `{
			"rules":[
				{
					"ClusterID": "%v",
					"RuleID": "%v",
					"ErrorKey": "%v"
				}
			],
			"status":"ok"
		}`
		disabledRulesBody = fmt.Sprintf(disabledRulesBody, clusterInfoList[0].ID, testdata.Rule2ID, testdata.ErrorKey2)

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ListOfDisabledRules,
				EndpointArgs: []interface{}{userIDOnGoodJWTAuthBearer},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       disabledRulesBody,
			},
		)

		resp := GetClustersResponse2ClusterWithHits1Rule
		for i := range clusterInfoList {
			resp.Clusters[i].ClusterID = clusterInfoList[i].ID
			resp.Clusters[i].ClusterName = clusterInfoList[i].DisplayName
		}

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigJWT.APIv2Prefix, &helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.ClustersRecommendationsEndpoint,
			AuthorizationToken: goodJWTAuthBearer,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(resp),
			BodyChecker: clusterInResponseChecker,
		})
	}, testTimeout)
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
