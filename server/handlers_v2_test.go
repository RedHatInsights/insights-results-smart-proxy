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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	iou_helpers "github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"
	ctypes "github.com/RedHatInsights/insights-results-types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
	data "github.com/RedHatInsights/insights-results-smart-proxy/tests/testdata"
	"github.com/RedHatInsights/insights-results-smart-proxy/types"
)

const (
	dotReportRuleModuleSuffix = ".report"
)

var (
	receivedTimestampTest  = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	processedTimestampTest = time.Now().UTC().Format(time.RFC3339)
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
			EndpointArgs: []interface{}{testdata.OrgID},
			Body:         rating,
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       aggregatorResponse,
		},
	)

	helpers.AssertAPIv2Request(
		t,
		&helpers.DefaultServerConfig,
		nil,
		nil,
		nil,
		nil,
		&helpers.APIRequest{
			Method:      http.MethodPost,
			Endpoint:    server.Rating,
			Body:        rating,
			XRHIdentity: goodXRHAuthToken,
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
	disabledAt := time.Now().UTC().Format(time.RFC3339)
	justificationNote := "justification test"
	clusters := []types.ClusterName{data.ClusterInfoResult2Clusters[0].ID, data.ClusterInfoResult2Clusters[1].ID}

	defer helpers.CleanAfterGock(t)

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	impactedClustersResponse := `
	{
		"clusters":[
			{
				"cluster":"%v",
				"cluster_name": "%v",
				"impacted": "",
				"last_checked_at":"",
				"meta": {
					"cluster_version": "%v"
				}
			}
		],
		"status":"ok"
	}
	`
	impactedClustersResponse = fmt.Sprintf(
		impactedClustersResponse, clusters[0], "", testdata.ClusterVersion,
	)
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.RuleClusterDetailEndpoint,
			EndpointArgs: []interface{}{testdata.Rule1CompositeID, testdata.OrgID, userIDInGoodAuthToken},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       impactedClustersResponse,
		},
	)

	disabledClustersResponse := `
	{
		"clusters":[
			{
				"cluster_id": "%v",
				"cluster_name": "%v",
				"disabled_at": "%v",
				"justification": "%v"
			}
		],
		"status":"ok"
	}
	`
	disabledClustersResponse = fmt.Sprintf(disabledClustersResponse, clusters[1], "", disabledAt, justificationNote)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledClusters,
			EndpointArgs: []interface{}{testdata.Rule1ID + dotReportRuleModuleSuffix, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       disabledClustersResponse,
		},
	)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{},
		"status":"ok"
	}
	`

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusNotFound,
			Body:       ackedRuleAggregatorResponse,
		},
	)

	// prepare list of organizations response
	amsClientMock := helpers.AMSClientWithOrgResults(
		testdata.OrgID,
		data.ClusterInfoResult2Clusters,
	)

	expectedResponse := `
	{
		"data": {
			"enabled": [
				{
					"cluster": "%v",
					"cluster_name": "%v",
					"impacted": "",
					"last_checked_at": "",
					"meta": {
						"cluster_version": "%v"
					}
				}
			],
			"disabled": [
				{
					"cluster_id": "%v",
					"cluster_name": "%v",
					"disabled_at": "%v",
					"justification": "%v"
				}
			]
		},
		"status":"ok"
	}
	`

	expectedResponse = fmt.Sprintf(
		expectedResponse, clusters[0], data.ClusterDisplayName1, testdata.ClusterVersion,
		clusters[1], data.ClusterDisplayName2, disabledAt, justificationNote,
	)

	testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

	iou_helpers.AssertAPIRequest(
		t,
		testServer,
		serverConfigXRH.APIv2Prefix,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ClustersDetail,
			EndpointArgs: []interface{}{testdata.Rule1CompositeID},
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       expectedResponse,
		},
	)
}

func TestHTTPServer_ClustersDetailEndpointAggregatorResponseOk_ImpactedClusterDisabled(t *testing.T) {
	disabledAt := time.Now().UTC().Format(time.RFC3339)
	justificationNote := "justification test"
	clusters := []types.ClusterName{data.ClusterInfoResult2Clusters[0].ID, data.ClusterInfoResult2Clusters[1].ID}

	defer helpers.CleanAfterGock(t)

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	impactedClustersResponse := `
	{
		"clusters":[
			{
				"cluster":"%v",
				"cluster_name": "%v"
			}
		],
		"status":"ok"
	}
	`
	impactedClustersResponse = fmt.Sprintf(impactedClustersResponse, clusters[0], "")

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.RuleClusterDetailEndpoint,
			EndpointArgs: []interface{}{testdata.Rule1CompositeID, testdata.OrgID, userIDInGoodAuthToken},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       impactedClustersResponse,
		},
	)

	disabledClustersResponse := `
	{
		"clusters":[
			{
				"cluster_id": "%v",
				"cluster_name": "%v",
				"disabled_at": "%v",
				"justification": "%v"
			}
		],
		"status":"ok"
	}
	`
	// same cluster disabled
	disabledClustersResponse = fmt.Sprintf(disabledClustersResponse, clusters[0], "", disabledAt, justificationNote)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledClusters,
			EndpointArgs: []interface{}{testdata.Rule1ID + dotReportRuleModuleSuffix, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       disabledClustersResponse,
		},
	)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{},
		"status":"ok"
	}
	`

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusNotFound,
			Body:       ackedRuleAggregatorResponse,
		},
	)

	// prepare list of organizations response
	amsClientMock := helpers.AMSClientWithOrgResults(
		testdata.OrgID,
		data.ClusterInfoResult2Clusters,
	)

	expectedResponse := `
	{
		"data": {
			"enabled": [],
			"disabled": [
				{
					"cluster_id": "%v",
					"cluster_name": "%v",
					"disabled_at": "%v",
					"justification": "%v"
				}
			]
		},
		"status":"ok"
	}
	`

	expectedResponse = fmt.Sprintf(expectedResponse, clusters[0], data.ClusterDisplayName1, disabledAt, justificationNote)

	testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

	iou_helpers.AssertAPIRequest(
		t,
		testServer,
		serverConfigXRH.APIv2Prefix,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ClustersDetail,
			EndpointArgs: []interface{}{testdata.Rule1CompositeID},
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       expectedResponse,
		},
	)
}

func TestHTTPServer_ClustersDetailEndpointAggregatorResponseOk_DisabledClusterNotActive(t *testing.T) {
	disabledAt := time.Now().UTC().Format(time.RFC3339)
	justificationNote := "justification test"
	// first cluster isn't in the AMS API response
	clusters := []types.ClusterName{testdata.ClusterName, data.ClusterInfoResult2Clusters[1].ID}

	defer helpers.CleanAfterGock(t)

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	impactedClustersResponse := `
	{
		"clusters":[
			{
				"cluster":"%v",
				"cluster_name": "%v"
			}
		],
		"status":"ok"
	}
	`
	impactedClustersResponse = fmt.Sprintf(impactedClustersResponse, clusters[1], "")
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.RuleClusterDetailEndpoint,
			EndpointArgs: []interface{}{testdata.Rule1CompositeID, testdata.OrgID, userIDInGoodAuthToken},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       impactedClustersResponse,
		},
	)

	disabledClustersResponse := `
	{
		"clusters":[
			{
				"cluster_id": "%v",
				"cluster_name": "%v",
				"disabled_at": "%v",
				"justification": "%v"
			}
		],
		"status":"ok"
	}
	`
	// cluster that isn't returned from AMS API is disabled, we must omit this cluster from the response
	disabledClustersResponse = fmt.Sprintf(disabledClustersResponse, testdata.ClusterName, "", disabledAt, justificationNote)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledClusters,
			EndpointArgs: []interface{}{testdata.Rule1ID + dotReportRuleModuleSuffix, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       disabledClustersResponse,
		},
	)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{},
		"status":"ok"
	}
	`

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusNotFound,
			Body:       ackedRuleAggregatorResponse,
		},
	)

	// prepare list of organizations response
	amsClientMock := helpers.AMSClientWithOrgResults(
		testdata.OrgID,
		data.ClusterInfoResult2Clusters,
	)

	expectedResponse := `
	{
		"data": {
			"enabled": [
				{
					"cluster": "%v",
					"cluster_name": "%v",
					"impacted":"",
					"last_checked_at":"",
					"meta": {
						"cluster_version": ""
					}
				}
			],
			"disabled": []
		},
		"status":"ok"
	}
	`

	// 2nd cluster is there
	expectedResponse = fmt.Sprintf(expectedResponse, clusters[1], data.ClusterDisplayName2)

	testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

	iou_helpers.AssertAPIRequest(
		t,
		testServer,
		serverConfigXRH.APIv2Prefix,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ClustersDetail,
			EndpointArgs: []interface{}{testdata.Rule1CompositeID},
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       expectedResponse,
		},
	)
}

func TestHTTPServer_ClustersDetailEndpointAggregatorResponseOk_AckedRule_CCXDEV_7099_Reproducer(t *testing.T) {
	disabledAt := time.Now().UTC().Format(time.RFC3339)
	justificationNote := "justification test"
	clusters := []types.ClusterName{data.ClusterInfoResult2Clusters[0].ID, data.ClusterInfoResult2Clusters[1].ID}

	defer helpers.CleanAfterGock(t)

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	impactedClustersResponse := `
	{
		"clusters":[
			{
				"cluster":"%v",
				"cluster_name": "%v"
			},
			{
				"cluster":"%v",
				"cluster_name": "%v"
			}
		],
		"status":"ok"
	}
	`
	impactedClustersResponse = fmt.Sprintf(impactedClustersResponse,
		clusters[0], "",
		clusters[1], "",
	)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.RuleClusterDetailEndpoint,
			EndpointArgs: []interface{}{testdata.Rule1CompositeID, testdata.OrgID, userIDInGoodAuthToken},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       impactedClustersResponse,
		},
	)

	// no single cluster disabled
	disabledClustersResponse := `{
		"clusters":[],
		"status":"ok"
	}`
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledClusters,
			EndpointArgs: []interface{}{testdata.Rule1ID + dotReportRuleModuleSuffix, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       disabledClustersResponse,
		},
	)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{
			"rule_id": "%v",
			"error_key": "%v",
			"justification": "%v",
			"created_by": "",
			"created_at": {
				"Time": "%v",
				"Valid": true
			},
			"updated_at": {
				"Time": "%v",
				"Valid": true
			}
		},
		"status":"ok"
	}
	`

	ackedRuleAggregatorResponse = fmt.Sprintf(ackedRuleAggregatorResponse,
		testdata.Rule1ID, testdata.ErrorKey1, justificationNote, disabledAt, disabledAt,
	)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ackedRuleAggregatorResponse,
		},
	)
	// prepare list of organizations response
	amsClientMock := helpers.AMSClientWithOrgResults(
		testdata.OrgID,
		data.ClusterInfoResult2Clusters,
	)

	expectedResponse := `
	{
		"data": {
			"enabled": [],
			"disabled": [
				{
					"cluster_id": "%v",
					"cluster_name": "%v",
					"disabled_at": "%v",
					"justification": "%v"
				},
				{
					"cluster_id": "%v",
					"cluster_name": "%v",
					"disabled_at": "%v",
					"justification": "%v"
				}
			]
		},
		"status":"ok"
	}
	`

	expectedResponse = fmt.Sprintf(expectedResponse,
		clusters[0], data.ClusterDisplayName1, disabledAt, justificationNote,
		clusters[1], data.ClusterDisplayName2, disabledAt, justificationNote,
	)

	testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

	// rule acked == all rules must be marked as disabled
	iou_helpers.AssertAPIRequest(
		t,
		testServer,
		serverConfigXRH.APIv2Prefix,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ClustersDetail,
			EndpointArgs: []interface{}{testdata.Rule1CompositeID},
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       expectedResponse,
		},
	)
}

func TestHTTPServer_ClustersDetailEndpointAggregatorResponseOk_AckedRule_DisabledCluster_CCXDEV_7099_Reproducer(t *testing.T) {
	disabledAt := time.Now().UTC().Format(time.RFC3339)
	justificationNote := "justification test"
	singleClusterDisableJustification := "single cluster justification"
	clusters := []types.ClusterName{data.ClusterInfoResult2Clusters[0].ID, data.ClusterInfoResult2Clusters[1].ID}

	defer helpers.CleanAfterGock(t)

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	impactedClustersResponse := `
	{
		"clusters":[
			{
				"cluster":"%v",
				"cluster_name": "%v"
			},
			{
				"cluster":"%v",
				"cluster_name": "%v"
			}
		],
		"status":"ok"
	}
	`
	impactedClustersResponse = fmt.Sprintf(impactedClustersResponse,
		clusters[0], "",
		clusters[1], "",
	)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.RuleClusterDetailEndpoint,
			EndpointArgs: []interface{}{testdata.Rule1CompositeID, testdata.OrgID, userIDInGoodAuthToken},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       impactedClustersResponse,
		},
	)

	// one cluster disabled
	disabledClustersResponse := `
	{
		"clusters":[
			{
				"cluster_id": "%v",
				"cluster_name": "%v",
				"disabled_at": "%v",
				"justification": "%v"
			}
		],
		"status":"ok"
	}
	`
	disabledClustersResponse = fmt.Sprintf(disabledClustersResponse, clusters[0], "", disabledAt, singleClusterDisableJustification)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledClusters,
			EndpointArgs: []interface{}{testdata.Rule1ID + dotReportRuleModuleSuffix, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       disabledClustersResponse,
		},
	)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{
			"rule_id": "%v",
			"error_key": "%v",
			"justification": "%v",
			"created_by": "",
			"created_at": {
				"Time": "%v",
				"Valid": true
			},
			"updated_at": {
				"Time": "%v",
				"Valid": true
			}
		},
		"status":"ok"
	}
	`

	ackedRuleAggregatorResponse = fmt.Sprintf(ackedRuleAggregatorResponse,
		testdata.Rule1ID, testdata.ErrorKey1, justificationNote, disabledAt, disabledAt,
	)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ackedRuleAggregatorResponse,
		},
	)
	// prepare list of organizations response
	amsClientMock := helpers.AMSClientWithOrgResults(
		testdata.OrgID,
		data.ClusterInfoResult2Clusters,
	)

	expectedResponse := `
	{
		"data": {
			"enabled": [],
			"disabled": [
				{
					"cluster_id": "%v",
					"cluster_name": "%v",
					"disabled_at": "%v",
					"justification": "%v"
				},
				{
					"cluster_id": "%v",
					"cluster_name": "%v",
					"disabled_at": "%v",
					"justification": "%v"
				}
			]
		},
		"status":"ok"
	}
	`

	// single cluster disable justification has priority over acks
	expectedResponse = fmt.Sprintf(expectedResponse,
		clusters[0], data.ClusterDisplayName1, disabledAt, singleClusterDisableJustification,
		clusters[1], data.ClusterDisplayName2, disabledAt, justificationNote,
	)

	testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

	// rule acked == all rules must be marked as disabled
	iou_helpers.AssertAPIRequest(
		t,
		testServer,
		serverConfigXRH.APIv2Prefix,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ClustersDetail,
			EndpointArgs: []interface{}{testdata.Rule1CompositeID},
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       expectedResponse,
		},
	)
}

func TestHTTPServer_ClustersDetailEndpointAMSManagedClusters(t *testing.T) {
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

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		disabledClustersResponse := `{
			"clusters":[],
			"status":"ok"
		}`

		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ListOfDisabledClusters,
				EndpointArgs: []interface{}{testdata.Rule2ID + dotReportRuleModuleSuffix, testdata.ErrorKey2, testdata.OrgID},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       disabledClustersResponse,
			},
		)

		ackedRuleAggregatorResponse := `
		{
			"disabledRule":{},
			"status":"ok"
		}
		`

		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.ReadRuleSystemWide,
				EndpointArgs: []interface{}{testdata.Rule2ID, testdata.ErrorKey2, testdata.OrgID},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusNotFound,
				Body:       ackedRuleAggregatorResponse,
			},
		)

		expectedResponse := `
			{
				"data": {
					"enabled": [],
					"disabled": []
				},
				"status":"ok"
			}
			`
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		// cluster is managed, but rule is not == must not show as hitting
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.ClustersDetail,
				EndpointArgs: []interface{}{testdata.Rule2CompositeID},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedResponse,
			},
		)
	}, testTimeout)
}

// TestHTTPServer_ClustersDetailEndpointAggregatorResponse400 verifies that
// the 400 Bad Request and the response body from aggregator are correctly
// forwarded to the client
func TestHTTPServer_ClustersDetailEndpointAggregatorResponse400(t *testing.T) {
	defer helpers.CleanAfterGock(t)

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	response := `{"status": "Error during parsing param 'rule_selector' with value 'X'. Error: 'Param rule_selector is not a valid rule selector (plugin_name|error_key)'"}`

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.RuleClusterDetailEndpoint,
			EndpointArgs: []interface{}{testdata.Rule1CompositeID, testdata.OrgID, userIDInGoodAuthToken},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusBadRequest,
			Body:       response,
		},
	)

	disabledClustersResponse := `
	{
		"clusters":[],
		"status":"ok"
	}
	`
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledClusters,
			EndpointArgs: []interface{}{testdata.Rule1ID + dotReportRuleModuleSuffix, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       disabledClustersResponse,
		},
	)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{},
		"status":"ok"
	}
	`

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusNotFound,
			Body:       ackedRuleAggregatorResponse,
		},
	)

	helpers.AssertAPIv2Request(
		t,
		&helpers.DefaultServerConfig,
		nil,
		nil,
		nil,
		nil,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ClustersDetail,
			EndpointArgs: []interface{}{"X"},
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusBadRequest,
			Body:       response,
		},
	)
}

// TestHTTPServer_ClustersDetailEndpointAggregatorResponse404 verifies that
// the 404 Not Found and the response body from aggregator are correctly
// forwarded to the client
func TestHTTPServer_ClustersDetailEndpointAggregatorResponse404(t *testing.T) {
	defer helpers.CleanAfterGock(t)

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	aggregatorResponse := `{"status":"Item with ID plugin.1|EK_1 was not found in the storage"}`

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.RuleClusterDetailEndpoint,
			EndpointArgs: []interface{}{testdata.Rule1CompositeID, testdata.OrgID, userIDInGoodAuthToken},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusNotFound,
			Body:       aggregatorResponse,
		},
	)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledClusters,
			EndpointArgs: []interface{}{testdata.Rule1ID + dotReportRuleModuleSuffix, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusNotFound,
			Body:       aggregatorResponse,
		},
	)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{},
		"status":"ok"
	}
	`

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusNotFound,
			Body:       ackedRuleAggregatorResponse,
		},
	)

	response := `
	{
		"data": {
			"enabled": [],
			"disabled": []
		},
		"status":"ok"
	}
	`

	helpers.AssertAPIv2Request(
		t,
		&helpers.DefaultServerConfig,
		nil,
		nil,
		nil,
		nil,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ClustersDetail,
			EndpointArgs: []interface{}{testdata.Rule1CompositeID},
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       response,
		},
	)
}

// TestHTTPServer_ClustersDetailEndpointAggregatorResponse500 verifies that
// the 500 Internal Error and the response body from aggregator are correctly
// forwarded to the client
func TestHTTPServer_ClustersDetailEndpointAggregatorResponse500(t *testing.T) {
	defer helpers.CleanAfterGock(t)

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	aggregatorResponse := `{"status": "Internal Server Error"}`
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.RuleClusterDetailEndpoint,
			EndpointArgs: []interface{}{testdata.Rule1CompositeID, testdata.OrgID, userIDInGoodAuthToken},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       aggregatorResponse,
		},
	)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledClusters,
			EndpointArgs: []interface{}{testdata.Rule1ID + dotReportRuleModuleSuffix, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       aggregatorResponse,
		},
	)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       aggregatorResponse,
		},
	)

	helpers.AssertAPIv2Request(
		t,
		&helpers.DefaultServerConfig,
		nil,
		nil,
		nil,
		nil,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ClustersDetail,
			EndpointArgs: []interface{}{testdata.Rule1CompositeID},
			XRHIdentity:  goodXRHAuthToken,
		}, &helpers.APIResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       aggregatorResponse,
		},
	)
}

func TestHTTPServer_GetSingleClusterInfo(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := data.GetRandomClusterInfoList(3)

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		expectedResponse := `
		{
			"cluster": {
				"cluster_id": "%s",
				"display_name": "%s",
				"managed": %t,
				"status": "%s"
			},
			"status":"ok"
		}
		`
		expectedResponse = fmt.Sprintf(expectedResponse, clusterInfoList[0].ID, clusterInfoList[0].DisplayName,
			clusterInfoList[0].Managed, clusterInfoList[0].Status,
		)
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.ClusterInfoEndpoint,
				EndpointArgs: []interface{}{clusterInfoList[0].ID},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedResponse,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetSingleClusterInfoClusterNotFound(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			[]types.ClusterInfo{},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.ClusterInfoEndpoint,
				EndpointArgs: []interface{}{testdata.ClusterName},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusNotFound,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetRequestStatusForCluster_RedisNil(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.StatusOfRequestID,
				EndpointArgs: []interface{}{testdata.ClusterName, "requestID1"},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusInternalServerError,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetRequestStatusForCluster_RedisError500(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey, services.ScanBatchCount).SetErr(errors.New("Redis server failure"))

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.StatusOfRequestID,
				EndpointArgs: []interface{}{testdata.ClusterName, "requestID1"},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusInternalServerError,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetRequestStatusForCluster_NoRequestsForCluster(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey, services.ScanBatchCount).SetVal([]string{}, 0)

		// no request IDs found
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.StatusOfRequestID,
				EndpointArgs: []interface{}{testdata.ClusterName, "requestID1"},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusNotFound,
				Body:       fmt.Sprintf(`{"status":"%v"}`, server.RequestsForClusterNotFound),
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetRequestStatusForCluster_RequestNotFound(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey, services.ScanBatchCount).SetVal([]string{"requestIDNotTheOne", "requestIDAlsoNotTheOne"}, 0)

		// request IDs found but don't match the requested one
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.StatusOfRequestID,
				EndpointArgs: []interface{}{testdata.ClusterName, "requestID1"},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusNotFound,
				Body:       fmt.Sprintf(`{"status":"%v"}`, server.RequestIDNotFound),
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetRequestStatusForCluster_BadRequestClusterID(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		// mock server not needed because the request will not get to part requiring Redis
		redisClient, _ := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		// invalid clusterID
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.StatusOfRequestID,
				EndpointArgs: []interface{}{testdata.BadClusterName, "requestID1"}, // bad cluster name
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusBadRequest,
				Body:       `{"status":"Error during parsing param 'cluster' with value 'aaaa'. Error: 'invalid UUID length: 4'"}`,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetRequestStatusForCluster_BadRequestID(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		// mock server not needed because the request will not get to part requiring Redis
		redisClient, _ := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		// invalid requestID in endpoint arg
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.StatusOfRequestID,
				EndpointArgs: []interface{}{testdata.ClusterName, "_"}, // invalid requestID
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusBadRequest,
				Body:       `{"status":"Error during parsing param 'request_id' with value '_'. Error: 'invalid request ID: '_''"}`,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetRequestStatusForCluster_BadAuthToken(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		// mock server not needed because the request will not get to part requiring Redis
		redisClient, _ := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		// bad token
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.StatusOfRequestID,
				EndpointArgs: []interface{}{testdata.ClusterName, "requestID1"},
				XRHIdentity:  invalidXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusForbidden,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetRequestStatusForCluster_SingleRequestID(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey, services.ScanBatchCount).SetVal([]string{"requestID1"}, 0)

		expectedResponse := fmt.Sprintf(`{"cluster":"%v","requestID":"%v","status":"processed"}`, testdata.ClusterName, "requestID1")

		// given request ID found in the list
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.StatusOfRequestID,
				EndpointArgs: []interface{}{testdata.ClusterName, "requestID1"},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedResponse,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetRequestStatusForCluster_RequestIDOnSecondPage(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey, services.ScanBatchCount).SetVal([]string{"requestID1"}, 42)
		// requested request ID is found on the 2nd page returned from Redis (more Redis scenarios covered in services package)
		redisServer.ExpectScan(42, expectedKey, services.ScanBatchCount).SetVal([]string{"requestID123"}, 0)

		expectedResponse := fmt.Sprintf(`{"cluster":"%v","requestID":"%v","status":"processed"}`, testdata.ClusterName, "requestID123")

		// given request ID found in the list
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.StatusOfRequestID,
				EndpointArgs: []interface{}{testdata.ClusterName, "requestID123"},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedResponse,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetRequestsForCluster_OK1Request(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		expectedKey1stCommand := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey1stCommand, services.ScanBatchCount).SetVal([]string{"requestID1"}, 0)

		expectedKey2ndCommand := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName, "requestID1")
		redisServer.ExpectHMGet(
			expectedKey2ndCommand, services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
		).SetVal([]interface{}{"requestID1", receivedTimestampTest, processedTimestampTest})

		expectedResponse := fmt.Sprintf(`{
			"cluster":"%v",
			"status":"ok",
			"requests":[
				{"processed":"%v", "received":"%v", "requestID":"requestID1", "valid":true}
			]
		}`, testdata.ClusterName, processedTimestampTest, receivedTimestampTest)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.ListAllRequestIDs,
				EndpointArgs: []interface{}{testdata.ClusterName},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedResponse,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetRequestsForCluster_OK3Requests(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		requestIDs := make([]string, 3)
		for i := range requestIDs {
			requestIDs[i] = fmt.Sprintf("requestID%d", i)
		}

		expectedKey1stCommand := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey1stCommand, services.ScanBatchCount).SetVal([]string{requestIDs[0], requestIDs[1], requestIDs[2]}, 0)

		for i := range requestIDs {
			expectedKey2ndCommand := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName, requestIDs[i])

			redisServer.ExpectHMGet(
				expectedKey2ndCommand, services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
			).SetVal([]interface{}{requestIDs[i], receivedTimestampTest, processedTimestampTest})
		}

		expectedResponse := fmt.Sprintf(`{
			"cluster":"%v",
			"status":"ok",
			"requests":[
				{"processed":"%v", "received":"%v", "requestID":"%v", "valid":true},
				{"processed":"%v", "received":"%v", "requestID":"%v", "valid":true},
				{"processed":"%v", "received":"%v", "requestID":"%v", "valid":true}
			]
		}`, testdata.ClusterName,
			processedTimestampTest, receivedTimestampTest, requestIDs[0],
			processedTimestampTest, receivedTimestampTest, requestIDs[1],
			processedTimestampTest, receivedTimestampTest, requestIDs[2],
		)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.ListAllRequestIDs,
				EndpointArgs: []interface{}{testdata.ClusterName},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedResponse,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetRequestsForCluster_RequestsNotFound(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey, services.ScanBatchCount).SetVal([]string{}, 0)

		// 2nd Redis call is not expected

		// no request IDs found for given cluster
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.ListAllRequestIDs,
				EndpointArgs: []interface{}{testdata.ClusterName},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusNotFound,
				Body:       fmt.Sprintf(`{"status":"%v"}`, server.RequestsForClusterNotFound),
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetRequestsForCluster_BadRequestClusterID(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		// mock server not needed because the request will not get to part requiring Redis
		redisClient, _ := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		// invalid clusterID
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.ListAllRequestIDs,
				EndpointArgs: []interface{}{testdata.BadClusterName}, // bad cluster name
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusBadRequest,
				Body:       `{"status":"Error during parsing param 'cluster' with value 'aaaa'. Error: 'invalid UUID length: 4'"}`,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetRequestsForCluster_BadAuthToken(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		// mock server not needed because the request will not get to part requiring Redis
		redisClient, _ := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		// invalid clusterID
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.ListAllRequestIDs,
				EndpointArgs: []interface{}{testdata.ClusterName},
				XRHIdentity:  invalidXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusForbidden,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetRequestsForCluster_NoRedis(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.ListAllRequestIDs,
				EndpointArgs: []interface{}{testdata.ClusterName},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusInternalServerError,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetRequestsForCluster_RedisError500(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		expectedKey1stCommand := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey1stCommand, services.ScanBatchCount).SetErr(errors.New("Redis server failure"))

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.ListAllRequestIDs,
				EndpointArgs: []interface{}{testdata.ClusterName},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusInternalServerError,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetRequestsForCluster_RedisError500_2ndCmd(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		expectedKey1stCommand := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey1stCommand, services.ScanBatchCount).SetVal([]string{"requestID1"}, 0)

		expectedKey2ndCommand := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName, "requestID1")
		redisServer.ExpectHMGet(
			expectedKey2ndCommand, services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
		).SetErr(errors.New("redis server failure"))

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.ListAllRequestIDs,
				EndpointArgs: []interface{}{testdata.ClusterName},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusInternalServerError,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetReportForRequest_OK1Request(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName, "requestID1")
		redisServer.ExpectHMGet(
			expectedKey, services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
		).SetVal([]interface{}{"requestID1", receivedTimestampTest, processedTimestampTest})

		expectedResponse := fmt.Sprintf(`{
			"cluster":"%v",
			"status":"ok",
			"requests":[
				{"processed":"%v", "received":"%v", "requestID":"requestID1", "valid":true}
			]
		}`, testdata.ClusterName, processedTimestampTest, receivedTimestampTest)

		requestIDList := []types.RequestID{"requestID1"}
		reqBody, _ := json.Marshal(requestIDList)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     server.ListAllRequestIDs,
				EndpointArgs: []interface{}{testdata.ClusterName},
				XRHIdentity:  goodXRHAuthToken,
				Body:         reqBody,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedResponse,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetRequestsForClusterPostVariant_OK3Request1Found(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		requestIDs := make([]string, 3)
		for i := range requestIDs {
			requestIDs[i] = fmt.Sprintf("requestID%d", i)

			expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName, requestIDs[i])

			// only third request is found, 1st and 2nd are not
			if i == 2 {
				redisServer.ExpectHMGet(
					expectedKey, services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
				).SetVal([]interface{}{requestIDs[2], receivedTimestampTest, processedTimestampTest})
			} else {
				redisServer.ExpectHMGet(
					expectedKey, services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
				).SetVal([]interface{}{nil, nil, nil})
			}
		}

		expectedResponse := fmt.Sprintf(`{
			"cluster":"%v",
			"status":"ok",
			"requests":[
				{"processed":"", "received":"", "requestID":"requestID0", "valid":false},
				{"processed":"", "received":"", "requestID":"requestID1", "valid":false},
				{"processed":"%v", "received":"%v", "requestID":"requestID2", "valid":true}
			]
		}`, testdata.ClusterName, processedTimestampTest, receivedTimestampTest)

		requestIDList := []types.RequestID{types.RequestID(requestIDs[0]), types.RequestID(requestIDs[1]), types.RequestID(requestIDs[2])}
		reqBody, _ := json.Marshal(requestIDList)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     server.ListAllRequestIDs,
				EndpointArgs: []interface{}{testdata.ClusterName},
				XRHIdentity:  goodXRHAuthToken,
				Body:         reqBody,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedResponse,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetRequestsForClusterPostVariant_OK1RequestNotFound(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName, "requestID1")
		redisServer.ExpectHMGet(
			expectedKey, services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
		).SetVal([]interface{}{nil, nil, nil})

		expectedResponse := fmt.Sprintf(`{
			"cluster":"%v",
			"status":"ok",
			"requests":[
				{"processed":"", "received":"", "requestID":"requestID1", "valid":false}
			]
		}`, testdata.ClusterName)

		requestIDList := []types.RequestID{"requestID1"}
		reqBody, _ := json.Marshal(requestIDList)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     server.ListAllRequestIDs,
				EndpointArgs: []interface{}{testdata.ClusterName},
				XRHIdentity:  goodXRHAuthToken,
				Body:         reqBody,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedResponse,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetRequestsForClusterPostVariant_NoRedis(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, nil, nil, nil, nil, nil)

		requestIDList := []types.RequestID{"requestID1"}
		reqBody, _ := json.Marshal(requestIDList)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     server.ListAllRequestIDs,
				EndpointArgs: []interface{}{testdata.ClusterName},
				XRHIdentity:  goodXRHAuthToken,
				Body:         reqBody,
			}, &helpers.APIResponse{
				StatusCode: http.StatusInternalServerError,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetRequestsForClusterPostVariant_RedisError500(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName, "requestID1")
		redisServer.ExpectHMGet(
			expectedKey, services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
		).SetErr(errors.New("Redis server failure"))

		requestIDList := []types.RequestID{"requestID1"}
		reqBody, _ := json.Marshal(requestIDList)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     server.ListAllRequestIDs,
				EndpointArgs: []interface{}{testdata.ClusterName},
				XRHIdentity:  goodXRHAuthToken,
				Body:         reqBody,
			}, &helpers.APIResponse{
				StatusCode: http.StatusInternalServerError,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetRequestsForClusterPostVariant_BadRequestClusterID(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		// mock server not needed because the request will not get to part requiring Redis
		redisClient, _ := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		requestIDList := []types.RequestID{"requestID1"}
		reqBody, _ := json.Marshal(requestIDList)

		// invalid clusterID
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     server.ListAllRequestIDs,
				EndpointArgs: []interface{}{testdata.BadClusterName}, // bad cluster name
				XRHIdentity:  goodXRHAuthToken,
				Body:         reqBody,
			}, &helpers.APIResponse{
				StatusCode: http.StatusBadRequest,
				Body:       `{"status":"Error during parsing param 'cluster' with value 'aaaa'. Error: 'invalid UUID length: 4'"}`,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetRequestsForClusterPostVariant_BadAuthToken(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		// mock server not needed because the request will not get to part requiring Redis
		redisClient, _ := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		requestIDList := []types.RequestID{"requestID1"}
		reqBody, _ := json.Marshal(requestIDList)

		// invalid clusterID
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     server.ListAllRequestIDs,
				EndpointArgs: []interface{}{testdata.ClusterName},
				XRHIdentity:  invalidXRHAuthToken,
				Body:         reqBody,
			}, &helpers.APIResponse{
				StatusCode: http.StatusForbidden,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetRequestsForClusterPostVariant_NoBody(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		// mock server not needed because the request will not get to part requiring Redis
		redisClient, _ := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		// invalid clusterID
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     server.ListAllRequestIDs,
				EndpointArgs: []interface{}{testdata.ClusterName},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusBadRequest,
				Body:       `{"status":"client didn't provide request body"}`,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetRequestsForClusterPostVariant_BadBodyContent(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		// mock server not needed because the request will not get to part requiring Redis
		redisClient, _ := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		// invalid clusterID
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     server.ListAllRequestIDs,
				EndpointArgs: []interface{}{testdata.ClusterName},
				XRHIdentity:  goodXRHAuthToken,
				Body:         "body is not JSON",
			}, &helpers.APIResponse{
				StatusCode: http.StatusBadRequest,
				Body:       `{"status":"client didn't provide a valid request body"}`,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetRequestsForClusterPostVariant_BadRequestID(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		// mock server not needed because the request will not get to part requiring Redis
		redisClient, _ := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		requestIDList := []types.RequestID{"_"}
		reqBody, _ := json.Marshal(requestIDList)

		// invalid requestID in body
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     server.ListAllRequestIDs,
				EndpointArgs: []interface{}{testdata.ClusterName},
				XRHIdentity:  goodXRHAuthToken,
				Body:         reqBody,
			}, &helpers.APIResponse{
				StatusCode: http.StatusBadRequest,
				Body:       `{"status":"Error during parsing param 'request_id' with value '_'. Error: 'invalid request ID: '_''"}`,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetReportForRequest_OK_RequestNotFound(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		// redis expects
		expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName, "requestID1")
		redisServer.ExpectHMGet(
			expectedKey, services.RequestIDFieldName, services.RuleHitsFieldName,
		).SetVal([]interface{}{nil, nil})

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.RuleHitsForRequestID,
				EndpointArgs: []interface{}{testdata.ClusterName, "requestID1"},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusNotFound,
				Body:       `{"status":"Item with ID requestID1 was not found in the storage"}`,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetReportForRequest_OK_NoRuleHits(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		// redis expects
		expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName, "requestID1")
		redisServer.ExpectHMGet(
			expectedKey, services.RequestIDFieldName, services.RuleHitsFieldName,
		).SetVal([]interface{}{"requestID1", ""})

		// gock expects
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ResponseNoRulesDisabledSystemWide,
		})

		cluster := []types.ClusterName{testdata.ClusterName}
		reqBody, _ := json.Marshal(cluster)
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

		expectedResponse := fmt.Sprintf(`{
			"cluster":"%v",
			"status":"processed",
			"requestID":"requestID1",
			"report":[]
		}`, testdata.ClusterName)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.RuleHitsForRequestID,
				EndpointArgs: []interface{}{testdata.ClusterName, "requestID1"},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedResponse,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetReportForRequest_OK_1RuleHit(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		// redis expects
		expectedRuleHits := fmt.Sprintf("%v|%v", testdata.Rule1ID, testdata.ErrorKey1)
		expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName, "requestID1")
		redisServer.ExpectHMGet(
			expectedKey, services.RequestIDFieldName, services.RuleHitsFieldName,
		).SetVal([]interface{}{"requestID1", expectedRuleHits})

		// gock expects
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ResponseNoRulesDisabledSystemWide,
		})

		cluster := []types.ClusterName{testdata.ClusterName}
		reqBody, _ := json.Marshal(cluster)
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

		expectedResponse := fmt.Sprintf(`{
			"cluster":"%v",
			"status":"processed",
			"requestID":"requestID1",
			"report":[
				{"description":"%v", "error_key":"%v", "rule_fqdn":"%v", "total_risk":%v}
			]
		}`, testdata.ClusterName, testdata.RuleWithContent1.Generic, testdata.ErrorKey1, testdata.Rule1ID, testdata.RuleWithContent1.TotalRisk)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.RuleHitsForRequestID,
				EndpointArgs: []interface{}{testdata.ClusterName, "requestID1"},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedResponse,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetReportForRequest_BadAuthToken(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		// mock server not needed because the request will not get to part requiring Redis
		redisClient, _ := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		// invalid clusterID
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.RuleHitsForRequestID,
				EndpointArgs: []interface{}{testdata.ClusterName, "requestID1"},
				XRHIdentity:  invalidXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusForbidden,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetReportForRequest_BadRequestClusterID(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		// mock server not needed because the request will not get to part requiring Redis
		redisClient, _ := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		// invalid clusterID
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.RuleHitsForRequestID,
				EndpointArgs: []interface{}{testdata.BadClusterName, "requestID1"}, // invalid clusterID
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusBadRequest,
				Body:       `{"status":"Error during parsing param 'cluster' with value 'aaaa'. Error: 'invalid UUID length: 4'"}`,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetReportForRequest_BadRequestID(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		// mock server not needed because the request will not get to part requiring Redis
		redisClient, _ := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		// invalid requestID in endpoint arg
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.RuleHitsForRequestID,
				EndpointArgs: []interface{}{testdata.ClusterName, "_"}, // invalid request ID
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusBadRequest,
				Body:       `{"status":"Error during parsing param 'request_id' with value '_'. Error: 'invalid request ID: '_''"}`,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetReportForRequest_NoRuleContent(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(
			createRuleContentDirectoryFromRuleContent(
				[]ctypes.RuleContent{},
			),
		)
		assert.Nil(t, err)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		// redis expects
		expectedRuleHits := fmt.Sprintf("%v|%v", testdata.Rule1ID, testdata.ErrorKey1)
		expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName, "requestID1")
		redisServer.ExpectHMGet(
			expectedKey, services.RequestIDFieldName, services.RuleHitsFieldName,
		).SetVal([]interface{}{"requestID1", expectedRuleHits})

		// gock expects
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ResponseNoRulesDisabledSystemWide,
		})

		cluster := []types.ClusterName{testdata.ClusterName}
		reqBody, _ := json.Marshal(cluster)
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

		expectedResponse := fmt.Sprintf(`{
			"cluster":"%v",
			"status":"processed",
			"requestID":"requestID1",
			"report":[]
		}`, testdata.ClusterName)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.RuleHitsForRequestID,
				EndpointArgs: []interface{}{testdata.ClusterName, "requestID1"},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedResponse,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetReportForRequest_OK_2RuleHits1Acked(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		// redis expects
		expectedRuleHits := fmt.Sprintf("%v|%v,%v|%v", testdata.Rule1ID, testdata.ErrorKey1, testdata.Rule2ID, testdata.ErrorKey2)
		expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName, "requestID1")
		redisServer.ExpectHMGet(
			expectedKey, services.RequestIDFieldName, services.RuleHitsFieldName,
		).SetVal([]interface{}{"requestID1", expectedRuleHits})

		// gock expects

		// rule 1 disabled system wide (acked)
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(ResponseRule1DisabledSystemWide),
		})

		cluster := []types.ClusterName{testdata.ClusterName}
		reqBody, _ := json.Marshal(cluster)
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

		expectedResponse := fmt.Sprintf(`{
			"cluster":"%v",
			"status":"processed",
			"requestID":"requestID1",
			"report":[
				{"description":"%v", "error_key":"%v", "rule_fqdn":"%v", "total_risk":%v}
			]
		}`, testdata.ClusterName, testdata.RuleWithContent2.Generic, testdata.ErrorKey2, testdata.Rule2ID, testdata.RuleWithContent2.TotalRisk)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.RuleHitsForRequestID,
				EndpointArgs: []interface{}{testdata.ClusterName, "requestID1"},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedResponse,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetReportForRequest_OK_3RuleHits1Acked1Disabled(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		// redis expects

		// 3 rule hits
		expectedRuleHits := fmt.Sprintf(
			"%v|%v,%v|%v,%v|%v", testdata.Rule1ID, testdata.ErrorKey1, testdata.Rule2ID, testdata.ErrorKey2,
			testdata.Rule3ID, testdata.ErrorKey3,
		)
		expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName, "requestID1")
		redisServer.ExpectHMGet(
			expectedKey, services.RequestIDFieldName, services.RuleHitsFieldName,
		).SetVal([]interface{}{"requestID1", expectedRuleHits})

		// gock expects

		// rule 1 disabled system wide (acked)
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(ResponseRule1DisabledSystemWide),
		})

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
		ruleDisablesBody = fmt.Sprintf(ruleDisablesBody, testdata.ClusterName, testdata.Rule2ID, testdata.ErrorKey2)

		cluster := []types.ClusterName{testdata.ClusterName}
		reqBody, _ := json.Marshal(cluster)
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

		// rule1 acked, rule2 disabled
		expectedResponse := fmt.Sprintf(`{
			"cluster":"%v",
			"status":"processed",
			"requestID":"requestID1",
			"report":[
				{"description":"%v", "error_key":"%v", "rule_fqdn":"%v", "total_risk":%v}
			]
		}`, testdata.ClusterName, testdata.RuleWithContent3.Generic, testdata.ErrorKey3, testdata.Rule3ID, testdata.RuleWithContent3.TotalRisk)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.RuleHitsForRequestID,
				EndpointArgs: []interface{}{testdata.ClusterName, "requestID1"},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedResponse,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetReportForRequest_AggregatorError(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		// redis expects
		expectedRuleHits := fmt.Sprintf("%v|%v", testdata.Rule1ID, testdata.ErrorKey1)
		expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName, "requestID1")
		redisServer.ExpectHMGet(
			expectedKey, services.RequestIDFieldName, services.RuleHitsFieldName,
		).SetVal([]interface{}{"requestID1", expectedRuleHits})

		// gock expects
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusInternalServerError,
		})

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.RuleHitsForRequestID,
				EndpointArgs: []interface{}{testdata.ClusterName, "requestID1"},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusInternalServerError,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetReportForRequest_AggregatorError_2ndCall(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, &redisClient, nil, nil, nil, nil)

		// redis expects
		expectedRuleHits := fmt.Sprintf("%v|%v", testdata.Rule1ID, testdata.ErrorKey1)
		expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName, "requestID1")
		redisServer.ExpectHMGet(
			expectedKey, services.RequestIDFieldName, services.RuleHitsFieldName,
		).SetVal([]interface{}{"requestID1", expectedRuleHits})

		// gock expects
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
			EndpointArgs: []interface{}{testdata.OrgID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ResponseNoRulesDisabledSystemWide,
		})

		// 2nd call fails
		cluster := []types.ClusterName{testdata.ClusterName}
		reqBody, _ := json.Marshal(cluster)
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.ListOfDisabledRulesForClusters,
				EndpointArgs: []interface{}{testdata.OrgID},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusInternalServerError,
			},
		)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.RuleHitsForRequestID,
				EndpointArgs: []interface{}{testdata.ClusterName, "requestID1"},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusInternalServerError,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetReportForRequest_NoRedis(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, nil, nil, nil, nil, nil)

		requestIDList := []types.RequestID{"requestID1"}
		reqBody, _ := json.Marshal(requestIDList)

		expectedResponse := `{"status": "Internal Server Error"}`

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.RuleHitsForRequestID,
				EndpointArgs: []interface{}{testdata.ClusterName, "requestID1"},
				XRHIdentity:  goodXRHAuthToken,
				Body:         reqBody,
			}, &helpers.APIResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       expectedResponse,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_DVONamespaceListEndpoint_NoWorkloads(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		// prepare list of organizations response
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			data.ClusterInfoResult,
		)

		reqBody, _ := json.Marshal(data.ClusterList1Cluster)
		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.DVOWorkloadRecommendations,
				EndpointArgs: []interface{}{testdata.OrgID},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusNotFound,
			},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		expectedResponse := `{"status": "ok", "workloads": []}`

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:      http.MethodGet,
				Endpoint:    server.DVONamespaceListEndpoint,
				XRHIdentity: goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedResponse,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_DVONamespaceListEndpoint_OK(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		// prepare list of organizations response
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			data.ClusterInfoResult2Clusters,
		)

		now := time.Now().UTC().Format(time.RFC3339)
		aggrResponse := struct {
			Status    string                        `json:"status"`
			Workloads []types.WorkloadsForNamespace `json:"workloads"`
		}{
			Status: "ok",
			Workloads: []types.WorkloadsForNamespace{
				{
					Cluster: types.Cluster{
						UUID: string(data.ClusterInfoResult2Clusters[0].ID),
					},
					Namespace: types.Namespace{
						UUID:     fmt.Sprint(uuid.New()),
						FullName: "namespace1",
					},
					Metadata: types.Metadata{
						Recommendations: 2,
						Objects:         2,
						ReportedAt:      now,
						LastCheckedAt:   now,
					},
					RecommendationsHitCount: map[string]int{
						string(testdata.Rule1CompositeID): 1,
						string(testdata.Rule2CompositeID): 1,
					},
				},
			},
		}

		reqBody, _ := json.Marshal(data.ClusterList2Clusters)
		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.DVOWorkloadRecommendations,
				EndpointArgs: []interface{}{testdata.OrgID},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(aggrResponse),
			},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		expectedResponse := types.DVONamespaceListResponse{
			Status: "ok",
			Workloads: []types.Workload{
				{
					Cluster:   aggrResponse.Workloads[0].Cluster,
					Namespace: aggrResponse.Workloads[0].Namespace,
					Metadata:  aggrResponse.Workloads[0].Metadata,
				},
			},
		}
		// filled in by smart-proxy
		expectedResponse.Workloads[0].Cluster.DisplayName = data.ClusterDisplayName1
		expectedResponse.Workloads[0].Metadata.HighestSeverity = 2
		hitsBySeverity := map[int]int{
			1: 1,
			2: 1,
		}
		expectedResponse.Workloads[0].Metadata.HitsBySeverity = hitsBySeverity

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:      http.MethodGet,
				Endpoint:    server.DVONamespaceListEndpoint,
				XRHIdentity: goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(expectedResponse),
			},
		)
	}, testTimeout)
}

func TestHTTPServer_DVONamespaceListEndpoint_NoAuthToken(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		// prepare list of organizations response
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			data.ClusterInfoResult,
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:   http.MethodGet,
				Endpoint: server.DVONamespaceListEndpoint,
			}, &helpers.APIResponse{
				StatusCode: http.StatusForbidden,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_DVONamespaceListEndpoint_NoAMS(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:      http.MethodGet,
				Endpoint:    server.DVONamespaceListEndpoint,
				XRHIdentity: goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusInternalServerError,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_DVONamespaceListEndpoint_AggregatorError(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		// prepare list of organizations response
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			data.ClusterInfoResult,
		)

		reqBody, _ := json.Marshal(data.ClusterList1Cluster)
		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.DVOWorkloadRecommendations,
				EndpointArgs: []interface{}{testdata.OrgID},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusInternalServerError,
			},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:      http.MethodGet,
				Endpoint:    server.DVONamespaceListEndpoint,
				XRHIdentity: goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusInternalServerError,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_DVONamespaceListEndpoint_RecommendationDoesNotExist(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		// prepare list of organizations response
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			data.ClusterInfoResult2Clusters,
		)

		now := time.Now().UTC().Format(time.RFC3339)
		aggrResponse := struct {
			Status    string                        `json:"status"`
			Workloads []types.WorkloadsForNamespace `json:"workloads"`
		}{
			Status: "ok",
			Workloads: []types.WorkloadsForNamespace{
				{
					Cluster: types.Cluster{
						UUID: string(data.ClusterInfoResult2Clusters[0].ID),
					},
					Namespace: types.Namespace{
						UUID:     fmt.Sprint(uuid.New()),
						FullName: "namespace1",
					},
					Metadata: types.Metadata{
						Recommendations: 2,
						Objects:         2,
						ReportedAt:      now,
						LastCheckedAt:   now,
					},
					RecommendationsHitCount: map[string]int{
						string("non-existent rule ID"):    1,
						string(testdata.Rule2CompositeID): 1,
					},
				},
			},
		}

		reqBody, _ := json.Marshal(data.ClusterList2Clusters)
		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.DVOWorkloadRecommendations,
				EndpointArgs: []interface{}{testdata.OrgID},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(aggrResponse),
			},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		expectedResponse := types.DVONamespaceListResponse{
			Status: "ok",
			Workloads: []types.Workload{
				{
					Cluster:   aggrResponse.Workloads[0].Cluster,
					Namespace: aggrResponse.Workloads[0].Namespace,
					Metadata:  aggrResponse.Workloads[0].Metadata,
				},
			},
		}
		// filled in by smart-proxy, wrong rule ID is simply ommitted
		expectedResponse.Workloads[0].Cluster.DisplayName = data.ClusterDisplayName1
		expectedResponse.Workloads[0].Metadata.HighestSeverity = 2
		hitsBySeverity := map[int]int{
			1: 0, // <-- wrong rule ID not counted
			2: 1,
		}
		expectedResponse.Workloads[0].Metadata.HitsBySeverity = hitsBySeverity

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:      http.MethodGet,
				Endpoint:    server.DVONamespaceListEndpoint,
				XRHIdentity: goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(expectedResponse),
			},
		)
	}, testTimeout)
}

func TestHTTPServer_DVONamespaceListEndpoint_FilterOutInactiveClusters(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		// prepare list of organizations response
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			data.ClusterInfoResult2Clusters,
		)

		now := time.Now().UTC().Format(time.RFC3339)
		aggrResponse := struct {
			Status    string                        `json:"status"`
			Workloads []types.WorkloadsForNamespace `json:"workloads"`
		}{
			Status: "ok",
			Workloads: []types.WorkloadsForNamespace{
				{
					Cluster: types.Cluster{
						UUID: string(testdata.ClusterName), // <-- cluster is not in the list of active clusters from AMS API
					},
					Namespace: types.Namespace{
						UUID:     fmt.Sprint(uuid.New()),
						FullName: "namespace2",
					},
					Metadata: types.Metadata{
						Recommendations: 1,
						Objects:         1,
						ReportedAt:      now,
						LastCheckedAt:   now,
					},
					RecommendationsHitCount: map[string]int{
						string(testdata.Rule1CompositeID): 1,
					},
				},
				{
					Cluster: types.Cluster{
						UUID: string(data.ClusterInfoResult2Clusters[0].ID),
					},
					Namespace: types.Namespace{
						UUID:     fmt.Sprint(uuid.New()),
						FullName: "namespace1",
					},
					Metadata: types.Metadata{
						Recommendations: 2,
						Objects:         2,
						ReportedAt:      now,
						LastCheckedAt:   now,
					},
					RecommendationsHitCount: map[string]int{
						string(testdata.Rule1CompositeID): 1,
						string(testdata.Rule2CompositeID): 1,
					},
				},
			},
		}

		reqBody, _ := json.Marshal(data.ClusterList2Clusters)
		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodPost,
				Endpoint:     ira_server.DVOWorkloadRecommendations,
				EndpointArgs: []interface{}{testdata.OrgID},
				Body:         reqBody,
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(aggrResponse),
			},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		expectedResponse := types.DVONamespaceListResponse{
			Status: "ok",
			Workloads: []types.Workload{
				{
					Cluster:   aggrResponse.Workloads[1].Cluster,
					Namespace: aggrResponse.Workloads[1].Namespace,
					Metadata:  aggrResponse.Workloads[1].Metadata,
				},
			},
		}
		// filled in by smart-proxy
		expectedResponse.Workloads[0].Cluster.DisplayName = data.ClusterDisplayName1
		expectedResponse.Workloads[0].Metadata.HighestSeverity = 2
		hitsBySeverity := map[int]int{
			1: 1,
			2: 1,
		}
		expectedResponse.Workloads[0].Metadata.HitsBySeverity = hitsBySeverity

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:      http.MethodGet,
				Endpoint:    server.DVONamespaceListEndpoint,
				XRHIdentity: goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(expectedResponse),
			},
		)
	}, testTimeout)
}

func TestHTTPServer_DVONamespaceForCluster1_ClusterNotFound(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		// prepare list of organizations response
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			data.ClusterInfoResult,
		)

		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.DVOWorkloadRecommendationsSingleNamespace,
				EndpointArgs: []interface{}{testdata.OrgID, data.NamespaceUUID1, testdata.ClusterName},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusNotFound,
			},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.DVONamespaceForClusterEndpoint,
				XRHIdentity:  goodXRHAuthToken,
				EndpointArgs: []interface{}{data.NamespaceUUID1, testdata.ClusterName},
			}, &helpers.APIResponse{
				StatusCode: http.StatusNotFound,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_DVONamespaceForCluster1_ClusterFoundNoWorkloads(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		// prepare list of organizations response
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			data.ClusterInfoResult,
		)

		now := time.Now().UTC().Format(time.RFC3339)
		aggrResp := struct {
			Status    string                    `json:"status"`
			Workloads types.WorkloadsForCluster `json:"workloads"`
		}{
			Status: "ok",
			Workloads: types.WorkloadsForCluster{
				Cluster: types.Cluster{
					UUID: string(testdata.ClusterName),
				},
				Namespace: types.Namespace{
					UUID: data.NamespaceUUID1,
				},
				Metadata: types.Metadata{
					Recommendations: 0,
					Objects:         0,
					ReportedAt:      now,
					LastCheckedAt:   now,
				},
				Recommendations: []types.DVORecommendation{},
			},
		}

		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.DVOWorkloadRecommendationsSingleNamespace,
				EndpointArgs: []interface{}{testdata.OrgID, data.NamespaceUUID1, testdata.ClusterName},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(aggrResp),
			},
		)

		expectedResponse := types.WorkloadsForCluster{
			Status:          "ok",
			Cluster:         aggrResp.Workloads.Cluster,
			Namespace:       aggrResp.Workloads.Namespace,
			Metadata:        aggrResp.Workloads.Metadata,
			Recommendations: aggrResp.Workloads.Recommendations,
		}
		expectedResponse.Cluster.DisplayName = data.ClusterDisplayName1
		expectedResponse.Metadata.HitsBySeverity = map[int]int{
			1: 0,
			2: 0,
		}

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.DVONamespaceForClusterEndpoint,
				XRHIdentity:  goodXRHAuthToken,
				EndpointArgs: []interface{}{data.NamespaceUUID1, testdata.ClusterName},
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(expectedResponse),
			},
		)
	}, testTimeout)
}

func TestHTTPServer_DVONamespaceForCluster1_ClusterFoundWithWorkloads(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		// prepare list of organizations response
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			data.ClusterInfoResult,
		)

		now := time.Now().UTC().Format(time.RFC3339)
		aggrResp := struct {
			Status    string                    `json:"status"`
			Workloads types.WorkloadsForCluster `json:"workloads"`
		}{
			Status: "ok",
			Workloads: types.WorkloadsForCluster{
				Cluster: types.Cluster{
					UUID: string(testdata.ClusterName),
				},
				Namespace: types.Namespace{
					UUID: data.NamespaceUUID1,
				},
				Metadata: types.Metadata{
					Recommendations: 2,
					Objects:         2,
					ReportedAt:      now,
					LastCheckedAt:   now,
				},
				Recommendations: []types.DVORecommendation{
					{
						Check: string(testdata.Rule1CompositeID),
						TemplateData: map[string]interface{}{
							"samples": []map[string]interface{}{
								{"name": "displayname"},
							},
						},
						Objects: []types.DVOObject{
							{
								Kind: "pod",
								UID:  uuid.NewString(),
								Name: "test_display_name",
							},
						},
					},
					{
						Check: string(testdata.Rule2CompositeID),
						TemplateData: map[string]interface{}{
							"samples": []map[string]interface{}{
								{
									"name": "displayname",
								},
							},
						},
						Objects: []types.DVOObject{
							{
								Kind: "pod",
								UID:  uuid.NewString(),
								Name: "test_display_name1",
							},
							{
								Kind: "pod",
								UID:  uuid.NewString(),
								Name: "test_display_name2",
							},
						},
					},
				},
			},
		}

		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.DVOWorkloadRecommendationsSingleNamespace,
				EndpointArgs: []interface{}{testdata.OrgID, data.NamespaceUUID1, testdata.ClusterName},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(aggrResp),
			},
		)

		expectedResponse := types.WorkloadsForCluster{
			Status:          "ok",
			Cluster:         aggrResp.Workloads.Cluster,
			Namespace:       aggrResp.Workloads.Namespace,
			Metadata:        aggrResp.Workloads.Metadata,
			Recommendations: aggrResp.Workloads.Recommendations,
		}
		expectedResponse.Cluster.DisplayName = data.ClusterDisplayName1
		expectedResponse.Metadata.HighestSeverity = 2
		expectedResponse.Metadata.HitsBySeverity = map[int]int{
			1: 1,
			2: 2,
		}
		expectedResponse.Recommendations[0].Details = testdata.RuleErrorKey1.Description
		expectedResponse.Recommendations[0].Resolution = testdata.RuleErrorKey1.Resolution
		expectedResponse.Recommendations[0].MoreInfo = testdata.RuleErrorKey1.MoreInfo
		expectedResponse.Recommendations[0].Reason = testdata.RuleErrorKey1.Reason
		expectedResponse.Recommendations[0].TotalRisk = testdata.RuleWithContent1.TotalRisk
		expectedResponse.Recommendations[0].Modified = testdata.RuleErrorKey1.PublishDate.UTC().Format(time.RFC3339)
		expectedResponse.Recommendations[0].TemplateData = aggrResp.Workloads.Recommendations[0].TemplateData
		expectedResponse.Recommendations[0].Objects = aggrResp.Workloads.Recommendations[0].Objects

		expectedResponse.Recommendations[1].Details = testdata.RuleErrorKey2.Description
		expectedResponse.Recommendations[1].Resolution = testdata.RuleErrorKey2.Resolution
		expectedResponse.Recommendations[1].MoreInfo = testdata.RuleErrorKey2.MoreInfo
		expectedResponse.Recommendations[1].Reason = testdata.RuleErrorKey2.Reason
		expectedResponse.Recommendations[1].TotalRisk = testdata.RuleWithContent2.TotalRisk
		expectedResponse.Recommendations[1].Modified = testdata.RuleErrorKey2.PublishDate.UTC().Format(time.RFC3339)
		expectedResponse.Recommendations[1].TemplateData = aggrResp.Workloads.Recommendations[1].TemplateData
		expectedResponse.Recommendations[1].Objects = aggrResp.Workloads.Recommendations[1].Objects

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.DVONamespaceForClusterEndpoint,
				XRHIdentity:  goodXRHAuthToken,
				EndpointArgs: []interface{}{data.NamespaceUUID1, testdata.ClusterName},
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(expectedResponse),
			},
		)
	}, testTimeout)
}

func TestHTTPServer_DVONamespaceForCluster1_BadAuthToken(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		// prepare list of organizations response
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			data.ClusterInfoResult,
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.DVONamespaceForClusterEndpoint,
				EndpointArgs: []interface{}{data.NamespaceUUID1, testdata.ClusterName},
			}, &helpers.APIResponse{
				StatusCode: http.StatusForbidden,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_DVONamespaceForCluster1_BadClusterID(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		// prepare list of organizations response
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			data.ClusterInfoResult,
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.DVONamespaceForClusterEndpoint,
				EndpointArgs: []interface{}{data.NamespaceUUID1, "bad cluster ID"},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusBadRequest,
			},
		)
	}, testTimeout)
}

// TestHTTPServer_DVONamespaceForCluster1_NonUUIDNamespaceID non-UUID namespace_ids are produced by test rules/molodec,
// we need to allow anything
func TestHTTPServer_DVONamespaceForCluster1_NonUUIDNamespaceID(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		// prepare list of organizations response
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			data.ClusterInfoResult,
		)

		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.DVOWorkloadRecommendationsSingleNamespace,
				EndpointArgs: []interface{}{testdata.OrgID, "namespace ID", testdata.ClusterName},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusNotFound,
			},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.DVONamespaceForClusterEndpoint,
				EndpointArgs: []interface{}{"namespace ID", testdata.ClusterName},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusNotFound,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_DVONamespaceForCluster1_BadNamespaceID(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		// prepare list of organizations response
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			data.ClusterInfoResult,
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		var longID string
		for i := 0; i < 8; i++ {
			longID += uuid.NewString()
		}

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.DVONamespaceForClusterEndpoint,
				EndpointArgs: []interface{}{longID, testdata.ClusterName}, // very long string as namespace_id (only validation we're doing)
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusBadRequest,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_DVONamespaceForCluster1_NoAMS(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.DVONamespaceForClusterEndpoint,
				XRHIdentity:  goodXRHAuthToken,
				EndpointArgs: []interface{}{data.NamespaceUUID1, testdata.ClusterName},
			}, &helpers.APIResponse{
				StatusCode: http.StatusServiceUnavailable,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_DVONamespaceForCluster1_ClusterFoundWithWorkloads_RuleContentError(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&ctypes.RuleContentDirectory{})
		assert.Nil(t, err)

		// prepare list of organizations response
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			data.ClusterInfoResult,
		)

		now := time.Now().UTC().Format(time.RFC3339)
		aggrResp := struct {
			Status    string                    `json:"status"`
			Workloads types.WorkloadsForCluster `json:"workloads"`
		}{
			Status: "ok",
			Workloads: types.WorkloadsForCluster{
				Cluster: types.Cluster{
					UUID: string(testdata.ClusterName),
				},
				Namespace: types.Namespace{
					UUID: data.NamespaceUUID1,
				},
				Metadata: types.Metadata{
					Recommendations: 2,
					Objects:         2,
					ReportedAt:      now,
					LastCheckedAt:   now,
				},
				Recommendations: []types.DVORecommendation{
					{
						Check: string(testdata.Rule1CompositeID),
					},
					{
						Check: string(testdata.Rule2CompositeID),
					},
				},
			},
		}

		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.DVOWorkloadRecommendationsSingleNamespace,
				EndpointArgs: []interface{}{testdata.OrgID, data.NamespaceUUID1, testdata.ClusterName},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(aggrResp),
			},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.DVONamespaceForClusterEndpoint,
				XRHIdentity:  goodXRHAuthToken,
				EndpointArgs: []interface{}{data.NamespaceUUID1, testdata.ClusterName},
			}, &helpers.APIResponse{
				StatusCode: http.StatusInternalServerError,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_DVONamespaceForCluster1_ClusterFoundWithWorkloads_NotFoundInAMS(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		// error from AMS API
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			nil,
		)

		now := time.Now().UTC().Format(time.RFC3339)
		aggrResp := struct {
			Status    string                    `json:"status"`
			Workloads types.WorkloadsForCluster `json:"workloads"`
		}{
			Status: "ok",
			Workloads: types.WorkloadsForCluster{
				Cluster: types.Cluster{
					UUID: string(testdata.ClusterName),
				},
				Namespace: types.Namespace{
					UUID: data.NamespaceUUID1,
				},
				Metadata: types.Metadata{
					Recommendations: 2,
					Objects:         2,
					ReportedAt:      now,
					LastCheckedAt:   now,
				},
				Recommendations: []types.DVORecommendation{
					{
						Check: string(testdata.Rule1CompositeID),
					},
					{
						Check: string(testdata.Rule2CompositeID),
					},
				},
			},
		}

		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.DVOWorkloadRecommendationsSingleNamespace,
				EndpointArgs: []interface{}{testdata.OrgID, data.NamespaceUUID1, testdata.ClusterName},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.ToJSONString(aggrResp),
			},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.DVONamespaceForClusterEndpoint,
				XRHIdentity:  goodXRHAuthToken,
				EndpointArgs: []interface{}{data.NamespaceUUID1, testdata.ClusterName},
			}, &helpers.APIResponse{
				StatusCode: http.StatusNotFound,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_DVONamespaceForCluster1_AggregatorError(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		// error from AMS API
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			data.ClusterInfoResult,
		)

		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.DVOWorkloadRecommendationsSingleNamespace,
				EndpointArgs: []interface{}{testdata.OrgID, data.NamespaceUUID1, testdata.ClusterName},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusInternalServerError,
			},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.DVONamespaceForClusterEndpoint,
				XRHIdentity:  goodXRHAuthToken,
				EndpointArgs: []interface{}{data.NamespaceUUID1, testdata.ClusterName},
			}, &helpers.APIResponse{
				StatusCode: http.StatusInternalServerError,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_DVONamespaceForCluster1_AggregatorBadResponse(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		// error from AMS API
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			data.ClusterInfoResult,
		)

		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     ira_server.DVOWorkloadRecommendationsSingleNamespace,
				EndpointArgs: []interface{}{testdata.OrgID, data.NamespaceUUID1, testdata.ClusterName},
			},
			&helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       "bad response format",
			},
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.DVONamespaceForClusterEndpoint,
				XRHIdentity:  goodXRHAuthToken,
				EndpointArgs: []interface{}{data.NamespaceUUID1, testdata.ClusterName},
			}, &helpers.APIResponse{
				StatusCode: http.StatusInternalServerError,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_DVONamespaceForCluster1_AggregatorUnavailable(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
		assert.Nil(t, err)

		// error from AMS API
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			data.ClusterInfoResult,
		)

		// gock expect missing == aggregator request will timeout

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.DVONamespaceForClusterEndpoint,
				XRHIdentity:  goodXRHAuthToken,
				EndpointArgs: []interface{}{data.NamespaceUUID1, testdata.ClusterName},
			}, &helpers.APIResponse{
				StatusCode: http.StatusServiceUnavailable,
			},
		)
	}, testTimeout)
}
