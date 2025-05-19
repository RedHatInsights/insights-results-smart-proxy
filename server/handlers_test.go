// Copyright 2020, 2021, 2022, 2023 Red Hat, Inc
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

	// "github.com/RedHatInsights/insights-content-service/groups"
	"github.com/RedHatInsights/insights-content-service/groups"
	"github.com/RedHatInsights/insights-operator-utils/responses"
	iou_helpers "github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"

	// "github.com/RedHatInsights/insights-results-smart-proxy/content"
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

	ResponseNoRulesDisabledPerCluster = `{"rules":[],"status":"ok"}`

	ResponseRule1DisabledSystemWide = struct {
		Status      string                         `json:"status"`
		RuleDisable []ctypes.SystemWideRuleDisable `json:"disabledRules"`
	}{
		Status: "ok",
		RuleDisable: []ctypes.SystemWideRuleDisable{
			{
				OrgID:         testdata.OrgID,
				UserID:        testdata.UserID,
				RuleID:        testdata.Rule1ID,
				ErrorKey:      testdata.ErrorKey1,
				Justification: "Rule 1 disabled for testing purposes",
			},
		},
	}

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

	v2ReportNoContent = types.SmartProxyReportV2{
		Meta: types.ReportResponseMetaV2{
			DisplayName:   string(testdata.ClusterName),
			Count:         0,
			Managed:       false,
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

	SmartProxyV2ReportResponse1RuleNoContent = struct {
		Status string                    `json:"status"`
		Report *types.SmartProxyReportV2 `json:"report"`
	}{
		Status: "ok",
		Report: &v2ReportNoContent,
	}

	SmartProxyV2ReportResponse1RuleOnlyOSD = struct {
		Status string                    `json:"status"`
		Report *types.SmartProxyReportV2 `json:"report"`
	}{
		Status: "ok",
		Report: &v2Report3RulesWithOnlyOSD,
	}

	v2Report3RulesWithOnlyOSD = types.SmartProxyReportV2{
		Meta: types.ReportResponseMetaV2{
			DisplayName:   string(testdata.ClusterName),
			Count:         1,
			Managed:       true,
			LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
			GatheredAt:    types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		},
		Data: Report3RulesWithOnlyOSDData,
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

func expectNoRulesDisabledSystemWide(t *testing.TB, orgID types.OrgID) {
	helpers.GockExpectAPIRequest(*t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
		EndpointArgs: []interface{}{orgID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       ResponseNoRulesDisabledSystemWide,
	})
}

func expectNoRulesDisabledPerCluster(t *testing.TB, orgID types.OrgID) {
	helpers.GockExpectAPIRequest(*t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledRules,
			EndpointArgs: []interface{}{orgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ResponseNoRulesDisabledPerCluster,
		},
	)
}

// TODO: test more cases for report endpoint
func TestHTTPServer_ReportEndpoint(t *testing.T) {
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

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
			XRHIdentity:  goodXRHAuthToken,
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

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusServiceUnavailable,
			Body:       expectedBody,
		})
	}, testTimeout)
}

// Reproducer for Bug 1977858
func TestHTTPServer_ReportEndpointNoContent(t *testing.T) {
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

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		// previously was InternalServerError, but it was changed as an edge-case which will appear as "No issues found"
		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyV1ReportResponse1RuleNoContent),
		})
	}, testTimeout)
}

func TestHTTPServer_ReportEndpointV2NoContent(t *testing.T) {
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := make([]types.ClusterInfo, 0)

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, userIDInGoodAuthToken},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report1RuleExpectedResponse,
		})

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		expectedJSONBody := helpers.ToJSONString(SmartProxyV2ReportResponse1RuleNoContent)
		// previously was InternalServerError, but it was changed as an edge-case which will appear as "No issues found"
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpointV2,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       types.UserID(userIDInGoodAuthToken),
			OrgID:        testdata.OrgID,
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       expectedJSONBody,
		})
	}, testTimeout)
}

// TestHTTPServer_ReportEndpointV2TestAMSData tests that data from AMS API (mocked) is passed correctly to the response
func TestHTTPServer_ReportEndpointV2TestAMSData(t *testing.T) {
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := data.GetRandomClusterInfoList(3)

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, clusterInfoList[0].ID, userIDInGoodAuthToken},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report1RuleExpectedResponse,
		})

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		resp := SmartProxyV2ReportResponse1RuleNoContent
		resp.Report.Meta.DisplayName = clusterInfoList[0].DisplayName
		resp.Report.Meta.Managed = clusterInfoList[0].Managed

		expectedJSONBody := helpers.ToJSONString(resp)

		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpointV2,
			EndpointArgs: []interface{}{clusterInfoList[0].ID},
			UserID:       types.UserID(userIDInGoodAuthToken),
			OrgID:        testdata.OrgID,
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       expectedJSONBody,
		})
	}, testTimeout)
}

func TestHTTPServer_ReportEndpointV2TestManagedClustersRules(t *testing.T) {
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := data.GetRandomClusterInfoList(3)

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		// 3 rules, only 1 of which is managed
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, clusterInfoList[0].ID, userIDInGoodAuthToken},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report3RulesExpectedResponse,
		})

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		resp := SmartProxyV2ReportResponse1RuleOnlyOSD
		resp.Report.Meta.DisplayName = clusterInfoList[0].DisplayName
		resp.Report.Meta.Managed = clusterInfoList[0].Managed

		expectedJSONBody := helpers.ToJSONString(resp)

		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpointV2,
			EndpointArgs: []interface{}{clusterInfoList[0].ID},
			UserID:       types.UserID(userIDInGoodAuthToken),
			OrgID:        testdata.OrgID,
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       expectedJSONBody,
		})
	}, testTimeout)
}

// Reproducer for Bug 1977858
func TestHTTPServer_ReportEndpointNoContentFor2Rules(t *testing.T) {
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

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		// 1 rule returned, but count = 3
		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyV1ReportResponse3Rules2NoContent),
		})
	}, testTimeout)
}

func TestHTTPServer_ReportEndpoint_WithOnlyOSDEndpoint(t *testing.T) {
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

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint + "?" + server.OSDEligibleParam + "=true",
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyV1ReportResponse3RulesWithOnlyOSD),
		})
	}, testTimeout)
}

func TestHTTPServer_ReportEndpoint_InsightsOperatorUserAgentManagedCluster(t *testing.T) {
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := make([]types.ClusterInfo, 2)
		for i := range clusterInfoList {
			clusterInfoList[i] = data.GetRandomClusterInfo()
			// clusters are managed
			clusterInfoList[i].Managed = true
		}

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, clusterInfoList[0].ID, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report3RulesExpectedResponse,
		})

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		ioUserAgent := fmt.Sprintf("insights-operator/one10time200gather184a34f6a168926d93c330 cluster/_%v_", clusterInfoList[0].ID)
		extraHeaders := make(http.Header, 1)
		extraHeaders["User-Agent"] = []string{ioUserAgent}

		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv1Prefix, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{clusterInfoList[0].ID},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
			XRHIdentity:  goodXRHAuthToken,
			ExtraHeaders: extraHeaders,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			// expecting the same response as if providing the osd_eligible flag
			Body: helpers.ToJSONString(SmartProxyV1ReportResponse3RulesWithOnlyOSD),
		})
	}, testTimeout)
}

func TestHTTPServer_ReportEndpoint_InsightsOperatorUserAgentNonManagedCluster(t *testing.T) {
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := make([]types.ClusterInfo, 2)
		for i := range clusterInfoList {
			clusterInfoList[i] = data.GetRandomClusterInfo()
			// clusters arent managed
			clusterInfoList[i].Managed = false
		}

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, clusterInfoList[0].ID, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       testdata.Report3RulesExpectedResponse,
		})

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		// any other User Agent will get the same old v1 response
		ioUserAgent := fmt.Sprintf("acm-operator/v2.3.0 cluster/%v", clusterInfoList[0].ID)
		extraHeaders := make(http.Header, 1)
		extraHeaders["User-Agent"] = []string{ioUserAgent}

		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv1Prefix, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{clusterInfoList[0].ID},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
			XRHIdentity:  goodXRHAuthToken,
			ExtraHeaders: extraHeaders,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyV1ReportResponse3Rules),
		})
	}, testTimeout)
}

func TestHTTPServer_ReportEndpoint_WithDisabledRulesForCluster(t *testing.T) {
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

			expectNoRulesDisabledSystemWide(&t, testdata.OrgID)
		}

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint + "?" + server.GetDisabledParam + "=false",
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
			XRHIdentity:  goodXRHAuthToken,
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
			XRHIdentity:  goodXRHAuthToken,
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
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyV1ReportResponse3RulesAll),
		})
	}, testTimeout)
}

func TestHTTPServer_ReportEndpoint_WithDisabledRulesForClusterAndMissingContent(t *testing.T) {
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

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyV1EmptyResponseDisabledRulesMissingContent),
		})
	}, testTimeout)
}

func TestHTTPServer_ReportEndpoint_WithClusterAndSystemWideDisabledRules(t *testing.T) {
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
				EndpointArgs: []interface{}{testdata.OrgID},
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
			XRHIdentity:  goodXRHAuthToken,
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
			XRHIdentity:  goodXRHAuthToken,
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
			XRHIdentity:  goodXRHAuthToken,
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
			XRHIdentity:  goodXRHAuthToken,
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
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(ReportMetainfoAPIResponseTwoReports),
		})
	}, testTimeout)
}

// TestHTTPServer_ReportMetainfoEndpointForbidden checks how HTTP codes are
// handled in report/info endpoint handler.
func TestHTTPServer_ReportMetainfoEndpointForbidden(t *testing.T) {
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
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusForbidden,
		})
	}, testTimeout)
}

// TestHTTPServer_ReportMetainfoEndpointImproperJSON check the /report/info
// endpoint when improper response is returned from Aggregator REST API.
func TestHTTPServer_ReportMetainfoEndpointImproperJSON(t *testing.T) {
	const metainfoResponse = "THIS_IS_NOT_JSON"

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
			XRHIdentity:  goodXRHAuthToken,
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
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusBadRequest,
			Body:       helpers.ToJSONString(ReportMetainfoAPIResponseInvalidClusterName),
		})
	}, testTimeout)
}

// TODO: test more cases for rule endpoint
func TestHTTPServer_RuleEndpoint(t *testing.T) {
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
			Method:   http.MethodGet,
			Endpoint: server.SingleRuleEndpoint,
			EndpointArgs: []interface{}{
				testdata.ClusterName, fmt.Sprintf("%v|%v", testdata.RuleErrorKey1.RuleModule, testdata.RuleErrorKey1.ErrorKey),
			},
			UserID:      testdata.UserID,
			OrgID:       testdata.OrgID,
			XRHIdentity: goodXRHAuthToken,
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
			Method:   http.MethodGet,
			Endpoint: server.SingleRuleEndpoint,
			EndpointArgs: []interface{}{
				testdata.ClusterName, fmt.Sprintf("%v|%v", testdata.RuleErrorKey1.RuleModule, testdata.RuleErrorKey1.ErrorKey),
			},
			UserID:      testdata.UserID,
			OrgID:       testdata.OrgID,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusServiceUnavailable,
			Body:       expectedBody,
		})
	}, testTimeout)
}

func TestHTTPServer_RuleEndpoint_WithOSD(t *testing.T) {
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
			Method:   http.MethodGet,
			Endpoint: server.SingleRuleEndpoint + "?" + server.OSDEligibleParam + "=true",
			EndpointArgs: []interface{}{
				testdata.ClusterName, fmt.Sprintf("%v|%v", testdata.RuleErrorKey1.RuleModule, testdata.RuleErrorKey1.ErrorKey),
			},
			UserID:      testdata.UserID,
			OrgID:       testdata.OrgID,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyReportResponse3SingleRule),
		})
	}, testTimeout)
}

func TestHTTPServer_RuleEndpoint_WithNotOSDRule(t *testing.T) {
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
			Method:   http.MethodGet,
			Endpoint: server.SingleRuleEndpoint + "?" + server.OSDEligibleParam + "=true",
			EndpointArgs: []interface{}{
				testdata.ClusterName, fmt.Sprintf("%v|%v", testdata.RuleErrorKey2.RuleModule, testdata.RuleErrorKey2.ErrorKey),
			},
			UserID:      testdata.UserID,
			OrgID:       testdata.OrgID,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusNotFound,
			Body:       helpers.ToJSONString(SmartProxyReportResponse3NoRuleFound),
		})
	}, testTimeout)
}

// TestHTTPServer_GetContent
func TestHTTPServer_GetContent(t *testing.T) {
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.Content,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(GetContentResponse3Rules),
			BodyChecker: ruleInContentChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_OverviewEndpoint
func TestHTTPServer_OverviewEndpoint(t *testing.T) {
	err := loadMockRuleContentDir(
		createRuleContentDirectoryFromRuleContent(
			[]ctypes.RuleContent{
				testdata.RuleContent1,
				testdata.RuleContent2,
				testdata.RuleContent3,
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
					"recommendations": ["%v","%v","%v"]
				}
			}
		}`
		respBody = fmt.Sprintf(respBody,
			clusterInfoList[0].ID, testTimeStr, testdata.Rule1CompositeID,
			testdata.Rule2CompositeID, testdata.Rule3CompositeID,
		)

		// prepare list of organizations response
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		// prepare response from aggregator
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ClustersRecommendationsListEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		expectNoRulesDisabledPerCluster(&t, testdata.OrgID)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			helpers.DefaultServerConfig.APIv1Prefix,
			&helpers.APIRequest{
				Method:      http.MethodGet,
				Endpoint:    server.OverviewEndpoint,
				XRHIdentity: goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(OverviewResponseRules123Enabled),
			},
		)
	}, testTimeout)
}

// TestHTTPServer_OverviewEndpointManagedClustersRules tests behaviour when a managed cluster is hitting non-managed rules
// Scenario without managed clusters is tested in other test cases
func TestHTTPServer_OverviewEndpointManagedClustersRules(t *testing.T) {
	err := loadMockRuleContentDir(
		createRuleContentDirectoryFromRuleContent(
			[]ctypes.RuleContent{
				testdata.RuleContent1, // rule 1 is managed (has osd_customer tag)
				testdata.RuleContent2,
				testdata.RuleContent3,
			},
		),
	)
	assert.Nil(t, err)

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := make([]types.ClusterInfo, 2)
		for i := range clusterInfoList {
			clusterInfoList[i] = data.GetRandomClusterInfo()
			clusterInfoList[i].Managed = true // make all clusters managed
		}
		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		// managed cluster; 1 managed rule, 2 non-managed rules
		respBody := `{
			"clusters":{
				"%v": {
					"created_at": "%v",
					"recommendations": ["%v","%v","%v"]
				}
			}
		}`
		respBody = fmt.Sprintf(respBody,
			clusterInfoList[0].ID, testTimeStr, testdata.Rule1CompositeID,
			testdata.Rule2CompositeID, testdata.Rule3CompositeID,
		)

		// prepare list of organizations response
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		// prepare response from aggregator
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ClustersRecommendationsListEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		expectNoRulesDisabledPerCluster(&t, testdata.OrgID)

		// managed cluster; 1 managed rule, 2 non-managed rules == only 1 rule must count
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			helpers.DefaultServerConfig.APIv1Prefix,
			&helpers.APIRequest{
				Method:      http.MethodGet,
				Endpoint:    server.OverviewEndpoint,
				XRHIdentity: goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(OverviewResponseManagedRules),
			},
		)
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
					"recommendations": ["%v","%v","%v"]
				}
			}
		}`
		respBody = fmt.Sprintf(respBody,
			clusterInfoList[0].ID, testTimeStr, testdata.Rule1CompositeID,
			testdata.Rule2CompositeID, testdata.Rule3CompositeID,
		)

		// prepare list of organizations response
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			// data.ClusterInfoResult,
			clusterInfoList,
		)

		// prepare response from aggregator
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ClustersRecommendationsListEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		expectNoRulesDisabledPerCluster(&t, testdata.OrgID)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, helpers.DefaultServerConfig.APIv1Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.OverviewEndpoint,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusServiceUnavailable,
			Body:       expectedBody,
		})
	}, testTimeout)
}

func TestHTTPServer_OverviewGetEndpointDisabledRule(t *testing.T) {
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory5Rules)
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
					"recommendations": ["%v","%v","%v"]
				}
			}
		}`
		respBody = fmt.Sprintf(respBody,
			clusterInfoList[0].ID, testTimeStr, testdata.Rule1CompositeID,
			testdata.Rule2CompositeID, testdata.Rule3CompositeID,
		)

		// prepare list of organizations response
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		// prepare response from aggregator
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ClustersRecommendationsListEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		// Rule 2 is disabled org-wide
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(ResponseRule2DisabledSystemWide),
		})

		expectNoRulesDisabledPerCluster(&t, testdata.OrgID)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			helpers.DefaultServerConfig.APIv1Prefix,
			&helpers.APIRequest{
				Method:      http.MethodGet,
				Endpoint:    server.OverviewEndpoint,
				XRHIdentity: goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(OverviewResponseRule1EnabledRule2Disabled),
			},
		)

		// prepare response from aggregator
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ClustersRecommendationsListEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		// Now rule 1 is disabled org-wide
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(ResponseRule1DisabledSystemWide),
		})

		expectNoRulesDisabledPerCluster(&t, testdata.OrgID)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			helpers.DefaultServerConfig.APIv1Prefix,
			&helpers.APIRequest{
				Method:      http.MethodGet,
				Endpoint:    server.OverviewEndpoint,
				XRHIdentity: goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(OverviewResponseRule1DisabledRule2Enabled),
			},
		)
	}, testTimeout)
}

// TestHTTPServer_OverviewEndpointWithFallback
func TestHTTPServer_OverviewEndpointWithFallback(t *testing.T) {
	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
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
					"recommendations": ["%v","%v","%v"]
				}
			}
		}`
		respBody = fmt.Sprintf(respBody,
			clusterInfoList[0].ID, testTimeStr, testdata.Rule1CompositeID,
			testdata.Rule2CompositeID, testdata.Rule3CompositeID,
		)

		// prepare list of organizations response
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ClustersForOrganizationEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(responses.BuildOkResponseWithData("clusters", clusterList)),
		})

		// prepare response from aggregator
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ClustersRecommendationsListEndpoint,
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		expectNoRulesDisabledPerCluster(&t, testdata.OrgID)

		config := helpers.DefaultServerConfig
		config.UseOrgClustersFallback = true
		testServer := helpers.CreateHTTPServer(&config, nil, nil, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			helpers.DefaultServerConfig.APIv1Prefix,
			&helpers.APIRequest{
				Method:      http.MethodGet,
				Endpoint:    server.OverviewEndpoint,
				OrgID:       testdata.OrgID,
				UserID:      ctypes.UserID(userIDInGoodAuthToken),
				XRHIdentity: goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(OverviewResponseRules123Enabled),
			})
	}, testTimeout)
}

func TestInternalOrganizations(t *testing.T) {
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
			badXRHAuthToken,
		},
		{
			"Internal organizations enabled, Request denied due to unparsable token",
			&serverConfigInternalOrganizations1,
			http.StatusForbidden,
			invalidXRHAuthToken,
		},
		{
			// testing anemic tenant (user without account_number). must pass after EBS/org_id migration
			"Internal organizations enabled, Request allowed for anemic tenant",
			&serverConfigInternalOrganizations1,
			http.StatusOK,
			anemicXRHAuthToken,
		},
		{
			"Internal organizations enabled, Request denied due to invalid type in token",
			&serverConfigInternalOrganizations1,
			http.StatusForbidden,
			invalidXRHAuthToken,
		},
		{
			"Internal organizations enabled, Request allowed",
			&serverConfigInternalOrganizations1,
			http.StatusOK,
			goodXRHAuthToken,
		},
		{
			"Internal organizations disabled, Request allowed",
			&serverConfigXRH,
			http.StatusOK,
			goodXRHAuthToken,
		},
	} {
		t.Run(testCase.TestName, func(t *testing.T) {
			helpers.RunTestWithTimeout(t, func(t testing.TB) {
				helpers.AssertAPIRequest(t, testCase.ServerConfig, nil, nil, nil, nil, &helpers.APIRequest{
					Method:       http.MethodGet,
					Endpoint:     server.RuleContent,
					EndpointArgs: []interface{}{internalTestRuleModule},
					XRHIdentity:  testCase.MockAuthToken,
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
			goodXRHAuthToken,
		},
	} {
		t.Run(testCase.TestName, func(t *testing.T) {
			helpers.RunTestWithTimeout(t, func(t testing.TB) {
				helpers.AssertAPIRequest(t, testCase.ServerConfig, nil, nil, nil, nil, &helpers.APIRequest{
					Method:      http.MethodGet,
					Endpoint:    server.RuleIDs,
					XRHIdentity: testCase.MockAuthToken,
				}, &helpers.APIResponse{
					StatusCode: testCase.ExpectedStatusCode,
				})
			}, testTimeout)
		})
	}
}

// TestRuleNamesResponse checks the REST API status and response
func TestRuleNamesResponse(t *testing.T) {
	err := loadMockRuleContentDir(
		createRuleContentDirectoryFromRuleContent(
			[]ctypes.RuleContent{RuleContentInternal1, testdata.RuleContent1},
		),
	)
	assert.Nil(t, err)

	expectedBody := `
		{
			"rules": ["ccx_rules_ocp.external.rules.node_installer_degraded", "ccx_ocp_rules.internal.bar"],
			"status": "ok"
		}
	`
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		helpers.AssertAPIRequest(t, &serverConfigInternalOrganizations1, nil, nil, nil, nil, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.RuleIDs,
			XRHIdentity: goodXRHAuthToken,
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
			Method:      http.MethodGet,
			Endpoint:    server.RuleIDs,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        expectedBody,
			BodyChecker: ruleIDsChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_OverviewWithClusterIDsEndpoint
func TestHTTPServer_OverviewWithClusterIDsEndpoint(t *testing.T) {
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

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:      http.MethodPost,
			Endpoint:    server.OverviewEndpoint,
			OrgID:       testdata.OrgID,
			UserID:      testdata.UserID,
			Body:        helpers.ToJSONString(data.ClusterIDListInReq),
			XRHIdentity: goodXRHAuthToken,
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

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:      http.MethodPost,
			Endpoint:    server.OverviewEndpoint,
			OrgID:       testdata.OrgID,
			UserID:      testdata.UserID,
			Body:        helpers.ToJSONString(data.ClusterIDListInReq),
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusServiceUnavailable,
			Body:       expectedBody,
		})
	}, testTimeout)
}

// TestHTTPServer_OverviewWithClusterIDsEndpoint
func TestHTTPServer_OverviewWithClusterIDsEndpointDisabledRules(t *testing.T) {
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

		// Rule 1 is disabled org-wide
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(ResponseRule1DisabledSystemWide),
		})

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:      http.MethodPost,
			Endpoint:    server.OverviewEndpoint,
			OrgID:       testdata.OrgID,
			UserID:      testdata.UserID,
			Body:        helpers.ToJSONString(data.ClusterIDListInReq),
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(OverviewResponsePostEndpointRule1Disabled),
		})

		// Now rule2 is disabled org-wide
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

		// Rule 1 is disabled org-wide
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(ResponseRule2DisabledSystemWide),
		})

		helpers.AssertAPIRequest(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
			Method:      http.MethodPost,
			Endpoint:    server.OverviewEndpoint,
			OrgID:       testdata.OrgID,
			UserID:      testdata.UserID,
			Body:        helpers.ToJSONString(data.ClusterIDListInReq),
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(OverviewResponsePostEndpointRule2Disabled),
		})
	}, testTimeout)
}

// TestHTTPServer_RecommendationsListEndpoint2Rules_ImpactingMissing
func TestHTTPServer_RecommendationsListEndpoint2Rules_ImpactingMissing(t *testing.T) {
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

		respBody := `{"recommendations":{"%v":["%v","%v"],"%v":["%v"]},"status":"ok"}`
		respBody = fmt.Sprintf(respBody,
			testdata.Rule1CompositeID, clusterList[0], clusterList[1],
			testdata.Rule2CompositeID, clusterList[0],
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
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		ruleDisablesBody := `{"rules":[],"status":"ok"}`
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ListOfDisabledRulesForClusters,
				EndpointArgs: []interface{}{testdata.OrgID},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ruleDisablesBody,
			},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.RecommendationsListEndpoint,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(GetRecommendationsResponse2Rules2Clusters),
			BodyChecker: recommendationInResponseChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_RecommendationsListEndpoint2Rules_ImpactingMissing1RuleDisabled1Acked
func TestHTTPServer_RecommendationsListEndpoint2Rules_ImpactingMissing1RuleDisabled1Acked(t *testing.T) {
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

		respBody := `{"recommendations":{"%v":["%v","%v"],"%v":["%v","%v"]},"status":"ok"}`
		respBody = fmt.Sprintf(respBody,
			testdata.Rule1CompositeID, clusterList[0], clusterList[1],
			testdata.Rule2CompositeID, clusterList[0], clusterList[1],
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
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
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
				EndpointArgs: []interface{}{testdata.OrgID},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ruleAcksBody,
			},
		)

		// rule 2 is disabled for one cluster
		ruleDisablesBody := `{
			"rules":[
				{
					"ClusterID": "%v",
					"RuleID": "%v.report",
					"ErrorKey": "%v"
				}
			],
			"status":"ok"
		}`
		ruleDisablesBody = fmt.Sprintf(ruleDisablesBody, clusterInfoList[0].ID, testdata.Rule2ID, testdata.ErrorKey2)

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ListOfDisabledRulesForClusters,
				EndpointArgs: []interface{}{testdata.OrgID},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ruleDisablesBody,
			},
		)

		// one rule acked; one rule user disabled (not counted as impacting)
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.RecommendationsListEndpoint,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(GetRecommendationsResponse2Rules1Disabled1Acked),
			BodyChecker: recommendationInResponseChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_RecommendationsListEndpoint2Rules1MissingContent
func TestHTTPServer_RecommendationsListEndpoint2Rules1MissingContent(t *testing.T) {
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

		respBody := `{"recommendations":{"%v":["%v","%v"],"%v":["%v","%v"]},"status":"ok"}`
		respBody = fmt.Sprintf(respBody,
			testdata.Rule1CompositeID, clusterList[0], clusterList[1],
			testdata.Rule2CompositeID, clusterList[0], clusterList[1],
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
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		ruleDisablesBody := `{"rules":[],"status":"ok"}`
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ListOfDisabledRulesForClusters,
				EndpointArgs: []interface{}{testdata.OrgID},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ruleDisablesBody,
			},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.RecommendationsListEndpoint,
			XRHIdentity: goodXRHAuthToken,
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

		err := loadMockRuleContentDir(&ctypes.RuleContentDirectory{})
		assert.Nil(t, err)
		clusterInfoList := make([]types.ClusterInfo, 2)
		for i := range clusterInfoList {
			clusterInfoList[i] = data.GetRandomClusterInfo()
		}

		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		respBody := `{"recommendations":{"%v":["%v","%v"],"%v":["%v","%v"]},"status":"ok"}`
		respBody = fmt.Sprintf(respBody,
			testdata.Rule1CompositeID, clusterList[0], clusterList[1],
			testdata.Rule2CompositeID, clusterList[0], clusterList[1],
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
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		ruleDisablesBody := `{"rules":[],"status":"ok"}`
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ListOfDisabledRulesForClusters,
				EndpointArgs: []interface{}{testdata.OrgID},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ruleDisablesBody,
			},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.RecommendationsListEndpoint,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(GetRecommendationsResponse0Rules),
		})
	}, testTimeout)
}

// TestHTTPServer_RecommendationsListEndpoint3Rules1Internal0Clusters_ImpactingTrue
func TestHTTPServer_RecommendationsListEndpoint3Rules1Internal0Clusters_ImpactingTrue(t *testing.T) {
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
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		ruleDisablesBody := `{"rules":[],"status":"ok"}`
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ListOfDisabledRulesForClusters,
				EndpointArgs: []interface{}{testdata.OrgID},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ruleDisablesBody,
			},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.RecommendationsListEndpoint + "?" + server.ImpactingParam + "=true",
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(GetRecommendationsResponse0Rules),
		})
	}, testTimeout)
}

// TestHTTPServer_RecommendationsListEndpoint3Rules1Internal0Clusters_ImpactingFalse
func TestHTTPServer_RecommendationsListEndpoint3Rules1Internal0Clusters_ImpactingFalse(t *testing.T) {
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
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		ruleDisablesBody := `{"rules":[],"status":"ok"}`
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ListOfDisabledRulesForClusters,
				EndpointArgs: []interface{}{testdata.OrgID},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ruleDisablesBody,
			},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.RecommendationsListEndpoint + "?" + server.ImpactingParam + "=false",
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(GetRecommendationsResponse2Rules0Clusters),
			BodyChecker: recommendationInResponseChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_RecommendationsListEndpoint2Rules1Internal2Clusters_ImpactingMissing
func TestHTTPServer_RecommendationsListEndpoint2Rules1Internal2Clusters_ImpactingMissing(t *testing.T) {
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

		respBody := `{"recommendations":{"%v":["%v","%v"]},"status":"ok"}`
		respBody = fmt.Sprintf(respBody,
			testdata.Rule1CompositeID, clusterList[0], clusterList[1],
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
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		ruleDisablesBody := `{"rules":[],"status":"ok"}`
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ListOfDisabledRulesForClusters,
				EndpointArgs: []interface{}{testdata.OrgID},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ruleDisablesBody,
			},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.RecommendationsListEndpoint,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(GetRecommendationsResponse1Rule2Cluster),
			BodyChecker: recommendationInResponseChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_RecommendationsListEndpoint4Rules1Internal2Clusters_ImpactingMissing
func TestHTTPServer_RecommendationsListEndpoint4Rules1Internal2Clusters_ImpactingMissing(t *testing.T) {
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

		respBody := `{"recommendations":{"%v":["%v"]},"status":"ok"}`
		respBody = fmt.Sprintf(respBody,
			testdata.Rule1CompositeID, clusterList[0],
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
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		ruleDisablesBody := `{"rules":[],"status":"ok"}`
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ListOfDisabledRulesForClusters,
				EndpointArgs: []interface{}{testdata.OrgID},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ruleDisablesBody,
			},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.RecommendationsListEndpoint,
			XRHIdentity: goodXRHAuthToken,
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

		helpers.AssertAPIv2Request(t, &helpers.DefaultServerConfig, nil, nil, nil, nil, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.RecommendationsListEndpoint,
			XRHIdentity: invalidXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusForbidden,
		})
	}, testTimeout)
}

// previously returned the error from strconv.Bool == 500
func TestHTTPServer_RecommendationsListEndpoint_BadImpactingParam(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		helpers.AssertAPIv2Request(t, &helpers.DefaultServerConfig, nil, nil, nil, nil, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.RecommendationsListEndpoint + "?" + server.ImpactingParam + "=badbool",
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusBadRequest,
		})
	}, testTimeout)
}

func TestHTTPServer_RecommendationsListEndpointAMSManagedClusters(t *testing.T) {
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
			clusterInfoList[i].Managed = true
		}

		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		respBody := `{"recommendations":{"%v":["%v","%v"],"%v":["%v","%v"]},"status":"ok"}`
		respBody = fmt.Sprintf(respBody,
			testdata.Rule1CompositeID, clusterList[0], clusterList[1],
			testdata.Rule2CompositeID, clusterList[0], clusterList[1],
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
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		ruleDisablesBody := `{"rules":[],"status":"ok"}`
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ListOfDisabledRulesForClusters,
				EndpointArgs: []interface{}{testdata.OrgID},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ruleDisablesBody,
			},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.RecommendationsListEndpoint,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(GetRecommendationsResponse2Rules2Clusters1Managed),
			BodyChecker: recommendationInResponseChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_GetRecommendationContent
func TestHTTPServer_GetRecommendationContent(t *testing.T) {
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
			&helpers.DefaultServerConfig,
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
			&helpers.DefaultServerConfig,
			testdata.Rule5CompositeID,
			http.StatusNotFound,
			nil,
		},
		{
			"invalid rule ID",
			&helpers.DefaultServerConfig,
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
					Method:       http.MethodGet,
					Endpoint:     server.RuleContentV2,
					EndpointArgs: []interface{}{testCase.RuleID},
					XRHIdentity:  goodXRHAuthToken,
				}, &response)
			}, testTimeout)
		})
	}
}

// TestHTTPServer_GetRecommendationContentWithUserData
func TestHTTPServer_GetRecommendationContentWithUserData(t *testing.T) {
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
			&helpers.DefaultServerConfig,
			ctypes.UserVoteNone,
			testdata.Rule1CompositeID,
			http.StatusOK,
			GetRuleContentRecommendationContentWithUserData1,
		},
		{
			"with rule like",
			&helpers.DefaultServerConfig,
			ctypes.UserVoteLike,
			testdata.Rule1CompositeID,
			http.StatusOK,
			GetRuleContentRecommendationContentWithUserData2RatingLike,
		},
		{
			"with rule dislike",
			&helpers.DefaultServerConfig,
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
			&helpers.DefaultServerConfig,
			ctypes.UserVoteDislike,
			testdata.Rule5CompositeID,
			http.StatusNotFound,
			nil,
		},
		{
			"invalid rule ID",
			&helpers.DefaultServerConfig,
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
						EndpointArgs: []interface{}{testCase.RuleID, testdata.OrgID},
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
						EndpointArgs: []interface{}{ruleID, errorKey, testdata.OrgID},
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
					Method:       http.MethodGet,
					Endpoint:     server.RuleContentWithUserData,
					EndpointArgs: []interface{}{testCase.RuleID},
					XRHIdentity:  goodXRHAuthToken,
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
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		expectNoRulesDisabledPerCluster(&t, testdata.OrgID)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.ClustersRecommendationsEndpoint,
			XRHIdentity: goodXRHAuthToken,
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
		err := loadMockRuleContentDir(&ctypes.RuleContentDirectory{})
		assert.Nil(t, err)
		clusterInfoList := data.GetRandomClusterInfoList(2)

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
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		expectNoRulesDisabledPerCluster(&t, testdata.OrgID)

		resp := GetClustersResponse2ClusterNoHits
		for i := range clusterInfoList {
			resp.Clusters[i].ClusterID = clusterInfoList[i].ID
			resp.Clusters[i].ClusterName = clusterInfoList[i].DisplayName
			resp.Clusters[i].Managed = clusterInfoList[i].Managed
			resp.Clusters[i].LastCheckedAt = "" // will be empty because we don't have the cluster in our DB
		}

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.ClustersRecommendationsEndpoint,
			XRHIdentity: goodXRHAuthToken,
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
		err := loadMockRuleContentDir(&ctypes.RuleContentDirectory{})
		assert.Nil(t, err)
		clusterInfoList := data.GetRandomClusterInfoList(2)

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
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		expectNoRulesDisabledPerCluster(&t, testdata.OrgID)

		resp := GetClustersResponse2ClusterNoHits
		for i := range clusterInfoList {
			resp.Clusters[i].ClusterID = clusterInfoList[i].ID
			resp.Clusters[i].ClusterName = clusterInfoList[i].DisplayName
			resp.Clusters[i].Managed = clusterInfoList[i].Managed
			resp.Clusters[i].LastCheckedAt = testTimestamp
		}

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.ClustersRecommendationsEndpoint,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(resp),
			BodyChecker: clusterInResponseChecker,
		})
	}, testTimeout)
}

func TestHTTPServer_ClustersRecommendationsEndpoint_NoReportInDB(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)
		err := loadMockRuleContentDir(&ctypes.RuleContentDirectory{})
		assert.Nil(t, err)
		clusterInfoList := data.GetRandomClusterInfoList(2)

		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		respBody := `{
			"clusters":{
			}
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
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		expectNoRulesDisabledPerCluster(&t, testdata.OrgID)

		resp := GetClustersResponse2ClusterNoArchiveInDB
		for i := range clusterInfoList {
			// cluster display name is filled, but last_checked_at is ommitted
			resp.Clusters[i].ClusterID = clusterInfoList[i].ID
			resp.Clusters[i].ClusterName = clusterInfoList[i].DisplayName
			resp.Clusters[i].Managed = clusterInfoList[i].Managed
		}

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.ClustersRecommendationsEndpoint,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(resp),
			BodyChecker: clusterInResponseChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_ClustersRecommendationsEndpoint_2ClustersFilled tests clusters received from AMS API with rule hits
func TestHTTPServer_ClustersRecommendationsEndpoint_2ClustersFilled(t *testing.T) {
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

		clusterInfoList := data.GetRandomClusterInfoList(2)

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
			clusterInfoList[0].ID, testTimeStr, testdata.Rule1CompositeID, // total_risk == 1
			clusterInfoList[1].ID, testTimeStr, testdata.Rule2CompositeID, testdata.Rule3CompositeID, // total_risk == 2, 2
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
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		expectNoRulesDisabledPerCluster(&t, testdata.OrgID)

		resp := GetClustersResponse2ClusterWithHits
		for i := range clusterInfoList {
			resp.Clusters[i].ClusterID = clusterInfoList[i].ID
			resp.Clusters[i].ClusterName = clusterInfoList[i].DisplayName
			resp.Clusters[i].Managed = clusterInfoList[i].Managed
		}

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.ClustersRecommendationsEndpoint,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(resp),
			BodyChecker: clusterInResponseChecker,
		})
	}, testTimeout)
}

func TestHTTPServer_ClustersRecommendationsEndpoint_2Clusters1Managed(t *testing.T) {
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

		clusterInfoList := data.GetRandomClusterInfoList(2)

		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		// cluster 1 is managed, so must only show managed rule 1
		respBody := `{
			"clusters":{
				"%v": {
					"created_at": "%v",
					"recommendations": ["%v","%v"]
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
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		expectNoRulesDisabledPerCluster(&t, testdata.OrgID)

		resp := GetClustersResponse2ClusterWithHitsCluster1Managed
		for i := range clusterInfoList {
			resp.Clusters[i].ClusterID = clusterInfoList[i].ID
			resp.Clusters[i].ClusterName = clusterInfoList[i].DisplayName
			resp.Clusters[i].Managed = clusterInfoList[i].Managed
		}

		// cluster 1 is managed, so must only show 1 rule. cluster 2 will show both rules.
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.ClustersRecommendationsEndpoint,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(resp),
			BodyChecker: clusterInResponseChecker,
		})
	}, testTimeout)
}

func TestHTTPServer_ClustersRecommendationsEndpoint_2Clusters1WithVersion(t *testing.T) {
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

		clusterInfoList := data.GetRandomClusterInfoList(2)

		clusterList := types.GetClusterNames(clusterInfoList)
		reqBody, _ := json.Marshal(clusterList)

		// cluster 1 is managed, so must only show managed rule 1
		respBody := `{
			"clusters":{
				"%v": {
					"created_at": "%v",
					"recommendations": ["%v","%v"],
					"meta": {
						"cluster_version": "%v"
					}
				},
				"%v": {
					"created_at": "%v",
					"recommendations": ["%v","%v"]
				}
			}
		}`
		respBody = fmt.Sprintf(respBody,
			clusterInfoList[0].ID, testTimeStr, testdata.Rule1CompositeID, testdata.Rule2CompositeID, testdata.ClusterVersion,
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
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		expectNoRulesDisabledPerCluster(&t, testdata.OrgID)

		resp := GetClustersResponse2ClusterWithHitsCluster1WithVersion
		for i := range clusterInfoList {
			resp.Clusters[i].ClusterID = clusterInfoList[i].ID
			resp.Clusters[i].ClusterName = clusterInfoList[i].DisplayName
			resp.Clusters[i].Managed = clusterInfoList[i].Managed
		}

		// cluster 1 is managed, so must only show 1 rule. cluster 2 will show both rules.
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.ClustersRecommendationsEndpoint,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(resp),
			BodyChecker: clusterInResponseChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_ClustersRecommendationsEndpoint_AckedRule tests clusters with an acked rule hitting both
func TestHTTPServer_ClustersRecommendationsEndpoint_AckedRule(t *testing.T) {
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

		clusterInfoList := data.GetRandomClusterInfoList(2)

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
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
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
				EndpointArgs: []interface{}{testdata.OrgID},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       ruleAcksBody,
			},
		)

		expectNoRulesDisabledPerCluster(&t, testdata.OrgID)

		resp := GetClustersResponse2ClusterWithHits1Rule
		for i := range clusterInfoList {
			resp.Clusters[i].ClusterID = clusterInfoList[i].ID
			resp.Clusters[i].ClusterName = clusterInfoList[i].DisplayName
			resp.Clusters[i].Managed = clusterInfoList[i].Managed
		}

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.ClustersRecommendationsEndpoint,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(resp),
			BodyChecker: clusterInResponseChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_ClustersRecommendationsEndpoint_DisabledRuleSingleCluster tests clusters with a disabled rule on one of them
func TestHTTPServer_ClustersRecommendationsEndpoint_DisabledRuleSingleCluster(t *testing.T) {
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

		clusterInfoList := data.GetRandomClusterInfoList(2)

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
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       respBody,
			},
		)

		// acks empty
		expectNoRulesDisabledSystemWide(&t, testdata.OrgID)

		// rule 1 disabled for only one cluster
		disabledRulesBody := `{
			"rules":[
				{
					"ClusterID": "%v",
					"RuleID": "%v.report",
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
				EndpointArgs: []interface{}{testdata.OrgID},
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
			resp.Clusters[i].Managed = clusterInfoList[i].Managed
		}

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.ClustersRecommendationsEndpoint,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(resp),
			BodyChecker: clusterInResponseChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_ClustersRecommendationsEndpoint_DisabledAndAcked tests clusters with a disabled rule on one of them and another acked rule
func TestHTTPServer_ClustersRecommendationsEndpoint_DisabledAndAcked(t *testing.T) {
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

		clusterInfoList := data.GetRandomClusterInfoList(2)

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
				EndpointArgs: []interface{}{testdata.OrgID, userIDInGoodAuthToken},
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
				EndpointArgs: []interface{}{testdata.OrgID},
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
					"RuleID": "%v.report",
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
				EndpointArgs: []interface{}{testdata.OrgID},
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
			resp.Clusters[i].Managed = clusterInfoList[i].Managed
		}

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(t, testServer, serverConfigXRH.APIv2Prefix, &helpers.APIRequest{
			Method:      http.MethodGet,
			Endpoint:    server.ClustersRecommendationsEndpoint,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode:  http.StatusOK,
			Body:        helpers.ToJSONString(resp),
			BodyChecker: clusterInResponseChecker,
		})
	}, testTimeout)
}

// TestHTTPServer_GroupsEndpoint tests the groups endpoint for both API versions
// TODO: fix race condition/deadlock, then this test can be enabled again
// If this test fails again, please refer to CCXDEV-11314 and attach the CI run
func TestHTTPServer_GroupsEndpoint(t *testing.T) {
	for _, prefix := range []string{serverConfigXRH.APIv1Prefix, serverConfigXRH.APIv2Prefix} {
		t.Run(prefix, func(t *testing.T) {
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

			testServer := helpers.CreateHTTPServer(
				&helpers.DefaultServerConfig,
				nil, nil, nil,
				groupsChannel,
				errorFoundChannel,
				errorChannel,
				nil,
			)

			req := &helpers.APIRequest{
				Method:      http.MethodGet,
				Endpoint:    server.RuleGroupsEndpoint,
				OrgID:       testdata.OrgID,
				XRHIdentity: goodXRHAuthToken,
			}
			expectedResponse := &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedBody,
			}

			helpers.RunTestWithTimeout(t, func(t testing.TB) {
				iou_helpers.AssertAPIRequest(t, testServer, prefix, req, expectedResponse)
			}, 30*time.Second)
		})
	}
}

// TODO: fix race condition/deadlock, then this test can be enabled again
// If this test fails again, please refer to CCXDEV-11314 and attach the CI run
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
			Method:      http.MethodGet,
			Endpoint:    server.RuleGroupsEndpoint,
			OrgID:       testdata.OrgID,
			XRHIdentity: goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusServiceUnavailable,
			Body:       expectedBody,
		})
	}, 30*time.Second)
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

func TestHTTPServer_GetClustersForOrganizationOk(t *testing.T) {
	testCases := []struct {
		name               string
		amsMockClusterList []types.ClusterInfo
		token              string
		orgID              string
		expectedResponse   string
		expectedStatusCode int
	}{
		{
			name:               "ok 2 clusters",
			amsMockClusterList: data.ClusterInfoResult2Clusters,
			orgID:              fmt.Sprint(testdata.OrgID),
			token:              goodXRHAuthToken,
			expectedResponse: fmt.Sprintf(
				`{"clusters":["%v","%v"],"status":"ok"}`,
				data.ClusterInfoResult2Clusters[0].ID,
				data.ClusterInfoResult2Clusters[1].ID,
			),
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "ok 0 clusters",
			amsMockClusterList: []types.ClusterInfo{},
			orgID:              fmt.Sprint(testdata.OrgID),
			token:              goodXRHAuthToken,
			expectedResponse:   `{"clusters":[],"status":"ok"}`,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "no permission for this org; bad org ID in URL",
			amsMockClusterList: data.ClusterInfoResult2Clusters,
			orgID:              fmt.Sprint(testdata.Org2ID),
			// wrong org ID in URL
			token:              goodXRHAuthToken,
			expectedResponse:   `{"status":"you have no permissions to get or change info about the organization with ID 2; you can access info about organization with ID 1"}`,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "no permission for this org; bad org ID in token",
			amsMockClusterList: []types.ClusterInfo{},
			orgID:              fmt.Sprint(testdata.OrgID),
			// wrong org ID in token
			token:              badXRHAuthToken,
			expectedResponse:   `{"status":"you have no permissions to get or change info about the organization with ID 1; you can access info about organization with ID 1234"}`,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "invalid token",
			amsMockClusterList: []types.ClusterInfo{},
			orgID:              fmt.Sprint(testdata.OrgID),
			token:              invalidXRHAuthToken,
			expectedResponse:   `{"status":"Malformed authentication token"}`,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			name:               "bad org ID in URL",
			amsMockClusterList: []types.ClusterInfo{},
			orgID:              "orgID not numeric",
			token:              goodXRHAuthToken,
			expectedResponse:   `{"status":"Error during parsing param 'organization' with value 'orgID not numeric'. Error: 'unsigned integer expected'"}`,
			expectedStatusCode: http.StatusBadRequest,
		},
	}
	for _, test := range testCases {
		helpers.RunTestWithTimeout(t, func(t testing.TB) {
			// prepare list of organizations response
			amsClientMock := helpers.AMSClientWithOrgResults(
				testdata.OrgID,
				test.amsMockClusterList,
			)

			testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

			iou_helpers.AssertAPIRequest(
				t,
				testServer,
				serverConfigXRH.APIv1Prefix,
				&helpers.APIRequest{
					Method:       http.MethodGet,
					Endpoint:     server.ClustersForOrganizationEndpoint,
					EndpointArgs: []interface{}{test.orgID},
					XRHIdentity:  test.token,
				}, &helpers.APIResponse{
					StatusCode: test.expectedStatusCode,
					Body:       test.expectedResponse,
				},
			)
		}, testTimeout)
	}
}

func TestHTTPServer_GetClustersForOrganizationAggregatorFallback(t *testing.T) {
	clusterInfoList := make([]types.ClusterInfo, 2)
	for i := range clusterInfoList {
		clusterInfoList[i] = data.GetRandomClusterInfo()
	}
	clusterList := types.GetClusterNames(clusterInfoList)

	// prepare response from aggregator
	helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     ira_server.ClustersForOrganizationEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       helpers.ToJSONString(responses.BuildOkResponseWithData("clusters", clusterList)),
	})

	config := helpers.DefaultServerConfig
	config.UseOrgClustersFallback = true
	// no AMS client
	testServer := helpers.CreateHTTPServer(&config, nil, nil, nil, nil, nil, nil, nil)

	iou_helpers.AssertAPIRequest(
		t,
		testServer,
		serverConfigXRH.APIv1Prefix,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ClustersForOrganizationEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID},
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(responses.BuildOkResponseWithData("clusters", clusterList)),
		},
	)
}
