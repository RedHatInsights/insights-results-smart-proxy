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
			EndpointArgs: []interface{}{testdata.Rule1CompositeID, testdata.OrgID, userIDOnGoodJWTAuthBearer},
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

	testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil, nil)

	iou_helpers.AssertAPIRequest(
		t,
		testServer,
		serverConfigJWT.APIv2Prefix,
		&helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.ClustersDetail,
			EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
			AuthorizationToken: goodJWTAuthBearer,
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
			EndpointArgs: []interface{}{testdata.Rule1CompositeID, testdata.OrgID, userIDOnGoodJWTAuthBearer},
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

	testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil, nil)

	iou_helpers.AssertAPIRequest(
		t,
		testServer,
		serverConfigJWT.APIv2Prefix,
		&helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.ClustersDetail,
			EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
			AuthorizationToken: goodJWTAuthBearer,
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
			EndpointArgs: []interface{}{testdata.Rule1CompositeID, testdata.OrgID, userIDOnGoodJWTAuthBearer},
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

	testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil, nil)

	iou_helpers.AssertAPIRequest(
		t,
		testServer,
		serverConfigJWT.APIv2Prefix,
		&helpers.APIRequest{
			Method:             http.MethodGet,
			Endpoint:           server.ClustersDetail,
			EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
			AuthorizationToken: goodJWTAuthBearer,
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

		expectedResponse := `
			{
				"data": {
					"enabled": [],
					"disabled": []
				},
				"status":"ok"
			}
			`
		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil, nil)

		// cluster is managed, but rule is not == must not show as hitting
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.ClustersDetail,
				EndpointArgs:       []interface{}{testdata.Rule2CompositeID},
				AuthorizationToken: goodJWTAuthBearer,
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
			EndpointArgs: []interface{}{testdata.Rule1CompositeID, testdata.OrgID, userIDOnGoodJWTAuthBearer},
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
			EndpointArgs:       []interface{}{"X"},
			AuthorizationToken: goodJWTAuthBearer,
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
			EndpointArgs: []interface{}{testdata.Rule1CompositeID, testdata.OrgID, userIDOnGoodJWTAuthBearer},
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
			EndpointArgs: []interface{}{testdata.Rule1CompositeID, testdata.OrgID, userIDOnGoodJWTAuthBearer},
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
		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.ClusterInfoEndpoint,
				EndpointArgs:       []interface{}{clusterInfoList[0].ID},
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, amsClientMock, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.ClusterInfoEndpoint,
				EndpointArgs:       []interface{}{testdata.ClusterName},
				AuthorizationToken: goodJWTAuthBearer,
			}, &helpers.APIResponse{
				StatusCode: http.StatusNotFound,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetRequestStatusForCluster_RedisError500(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey, 0).SetErr(errors.New("Redis server failure"))

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.StatusOfRequestID,
				EndpointArgs:       []interface{}{testdata.ClusterName, "requestID1"},
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey, 0).SetVal([]string{}, 0)

		// no request IDs found
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.StatusOfRequestID,
				EndpointArgs:       []interface{}{testdata.ClusterName, "requestID1"},
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey, 0).SetVal([]string{"requestIDNotTheOne", "requestIDAlsoNotTheOne"}, 0)

		// request IDs found but don't match the requested one
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.StatusOfRequestID,
				EndpointArgs:       []interface{}{testdata.ClusterName, "requestID1"},
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		// invalid clusterID
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.StatusOfRequestID,
				EndpointArgs:       []interface{}{testdata.BadClusterName, "requestID1"}, // bad cluster name
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		// invalid requestID in endpoint arg
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.StatusOfRequestID,
				EndpointArgs:       []interface{}{testdata.ClusterName, "_"}, // invalid requestID
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		// bad token
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.StatusOfRequestID,
				EndpointArgs:       []interface{}{testdata.ClusterName, "requestID1"},
				AuthorizationToken: unparsableJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey, 0).SetVal([]string{"requestID1"}, 0)

		expectedResponse := fmt.Sprintf(`{"cluster":"%v","requestID":"%v","status":"processed"}`, testdata.ClusterName, "requestID1")

		// given request ID found in the list
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.StatusOfRequestID,
				EndpointArgs:       []interface{}{testdata.ClusterName, "requestID1"},
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey, 0).SetVal([]string{"requestID1"}, 42)
		// requested request ID is found on the 2nd page returned from Redis (more Redis scenarios covered in services package)
		redisServer.ExpectScan(42, expectedKey, 0).SetVal([]string{"requestID123"}, 0)

		expectedResponse := fmt.Sprintf(`{"cluster":"%v","requestID":"%v","status":"processed"}`, testdata.ClusterName, "requestID123")

		// given request ID found in the list
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.StatusOfRequestID,
				EndpointArgs:       []interface{}{testdata.ClusterName, "requestID123"},
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		expectedKey1stCommand := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey1stCommand, 0).SetVal([]string{"requestID1"}, 0)

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
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.ListAllRequestIDs,
				EndpointArgs:       []interface{}{testdata.ClusterName},
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		requestIDs := make([]string, 3)
		for i := range requestIDs {
			requestIDs[i] = fmt.Sprintf("requestID%d", i)
		}

		expectedKey1stCommand := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey1stCommand, 0).SetVal([]string{requestIDs[0], requestIDs[1], requestIDs[2]}, 0)

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
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.ListAllRequestIDs,
				EndpointArgs:       []interface{}{testdata.ClusterName},
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey, 0).SetVal([]string{}, 0)

		// 2nd Redis call is not expected

		// no request IDs found for given cluster
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.ListAllRequestIDs,
				EndpointArgs:       []interface{}{testdata.ClusterName},
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		// invalid clusterID
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.ListAllRequestIDs,
				EndpointArgs:       []interface{}{testdata.BadClusterName}, // bad cluster name
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		// invalid clusterID
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.ListAllRequestIDs,
				EndpointArgs:       []interface{}{testdata.ClusterName},
				AuthorizationToken: unparsableJWTAuthBearer,
			}, &helpers.APIResponse{
				StatusCode: http.StatusForbidden,
			},
		)

	}, testTimeout)
}

func TestHTTPServer_GetRequestsForCluster_RedisError500(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		expectedKey1stCommand := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey1stCommand, 0).SetErr(errors.New("Redis server failure"))

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.ListAllRequestIDs,
				EndpointArgs:       []interface{}{testdata.ClusterName},
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		expectedKey1stCommand := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName)
		redisServer.ExpectScan(0, expectedKey1stCommand, 0).SetVal([]string{"requestID1"}, 0)

		expectedKey2ndCommand := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName, "requestID1")
		redisServer.ExpectHMGet(
			expectedKey2ndCommand, services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
		).SetErr(errors.New("redis server failure"))

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.ListAllRequestIDs,
				EndpointArgs:       []interface{}{testdata.ClusterName},
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

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
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodPost,
				Endpoint:           server.ListAllRequestIDs,
				EndpointArgs:       []interface{}{testdata.ClusterName},
				AuthorizationToken: goodJWTAuthBearer,
				Body:               reqBody,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

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
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodPost,
				Endpoint:           server.ListAllRequestIDs,
				EndpointArgs:       []interface{}{testdata.ClusterName},
				AuthorizationToken: goodJWTAuthBearer,
				Body:               reqBody,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

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
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodPost,
				Endpoint:           server.ListAllRequestIDs,
				EndpointArgs:       []interface{}{testdata.ClusterName},
				AuthorizationToken: goodJWTAuthBearer,
				Body:               reqBody,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       expectedResponse,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}

func TestHTTPServer_GetRequestsForClusterPostVariant_RedisError500(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(tt testing.TB) {
		defer helpers.CleanAfterGock(t)

		redisClient, redisServer := helpers.GetMockRedis()

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName, "requestID1")
		redisServer.ExpectHMGet(
			expectedKey, services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
		).SetErr(errors.New("Redis server failure"))

		requestIDList := []types.RequestID{"requestID1"}
		reqBody, _ := json.Marshal(requestIDList)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodPost,
				Endpoint:           server.ListAllRequestIDs,
				EndpointArgs:       []interface{}{testdata.ClusterName},
				AuthorizationToken: goodJWTAuthBearer,
				Body:               reqBody,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		requestIDList := []types.RequestID{"requestID1"}
		reqBody, _ := json.Marshal(requestIDList)

		// invalid clusterID
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodPost,
				Endpoint:           server.ListAllRequestIDs,
				EndpointArgs:       []interface{}{testdata.BadClusterName}, // bad cluster name
				AuthorizationToken: goodJWTAuthBearer,
				Body:               reqBody,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		requestIDList := []types.RequestID{"requestID1"}
		reqBody, _ := json.Marshal(requestIDList)

		// invalid clusterID
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodPost,
				Endpoint:           server.ListAllRequestIDs,
				EndpointArgs:       []interface{}{testdata.ClusterName},
				AuthorizationToken: unparsableJWTAuthBearer,
				Body:               reqBody,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		// invalid clusterID
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodPost,
				Endpoint:           server.ListAllRequestIDs,
				EndpointArgs:       []interface{}{testdata.ClusterName},
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		// invalid clusterID
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodPost,
				Endpoint:           server.ListAllRequestIDs,
				EndpointArgs:       []interface{}{testdata.ClusterName},
				AuthorizationToken: goodJWTAuthBearer,
				Body:               "body is not JSON",
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		requestIDList := []types.RequestID{"_"}
		reqBody, _ := json.Marshal(requestIDList)

		// invalid requestID in body
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodPost,
				Endpoint:           server.ListAllRequestIDs,
				EndpointArgs:       []interface{}{testdata.ClusterName},
				AuthorizationToken: goodJWTAuthBearer,
				Body:               reqBody,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		// redis expects
		expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName, "requestID1")
		redisServer.ExpectHMGet(
			expectedKey, services.RequestIDFieldName, services.RuleHitsFieldName,
		).SetVal([]interface{}{nil, nil})

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.RuleHitsForRequestID,
				EndpointArgs:       []interface{}{testdata.ClusterName, "requestID1"},
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

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
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.RuleHitsForRequestID,
				EndpointArgs:       []interface{}{testdata.ClusterName, "requestID1"},
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

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
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.RuleHitsForRequestID,
				EndpointArgs:       []interface{}{testdata.ClusterName, "requestID1"},
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		// invalid clusterID
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.RuleHitsForRequestID,
				EndpointArgs:       []interface{}{testdata.ClusterName, "requestID1"},
				AuthorizationToken: unparsableJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		// invalid clusterID
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.RuleHitsForRequestID,
				EndpointArgs:       []interface{}{testdata.BadClusterName, "requestID1"}, // invalid clusterID
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

		// invalid requestID in endpoint arg
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.RuleHitsForRequestID,
				EndpointArgs:       []interface{}{testdata.ClusterName, "_"}, // invalid request ID
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

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
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.RuleHitsForRequestID,
				EndpointArgs:       []interface{}{testdata.ClusterName, "requestID1"},
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

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
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.RuleHitsForRequestID,
				EndpointArgs:       []interface{}{testdata.ClusterName, "requestID1"},
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

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
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.RuleHitsForRequestID,
				EndpointArgs:       []interface{}{testdata.ClusterName, "requestID1"},
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

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
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.RuleHitsForRequestID,
				EndpointArgs:       []interface{}{testdata.ClusterName, "requestID1"},
				AuthorizationToken: goodJWTAuthBearer,
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

		testServer := helpers.CreateHTTPServer(&serverConfigJWT, nil, nil, &redisClient, nil, nil, nil)

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
			serverConfigJWT.APIv2Prefix,
			&helpers.APIRequest{
				Method:             http.MethodGet,
				Endpoint:           server.RuleHitsForRequestID,
				EndpointArgs:       []interface{}{testdata.ClusterName, "requestID1"},
				AuthorizationToken: goodJWTAuthBearer,
			}, &helpers.APIResponse{
				StatusCode: http.StatusInternalServerError,
			},
		)

		helpers.RedisExpectationsMet(t, redisServer)
	}, testTimeout)
}
