// Copyright 2023 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	iou_helpers "github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/testdata"
	ctypes "github.com/RedHatInsights/insights-results-types"
	"github.com/stretchr/testify/assert"
)

const upgradeRecommended = `
{
	"upgrade_recommendation": {
		"upgrade_recommended": true,
		"upgrade_risks_predictors": {
			"alerts": null,
			"operator_conditions": null
		}
	},
	"meta": {
		"last_checked_at": "0001-01-01T00:00:00Z"
	},
	"status":"ok"
}
`
const upgradeNotRecommended = `
{
	"upgrade_recommendation": {
		"upgrade_recommended": false,
		"upgrade_risks_predictors": {
			"alerts": [
				{
					"name": "alert1",
					"namespace": "namespace1",
					"severity": "info",
					"url": "https://my-cluster.com/monitoring/alerts?orderBy=asc&sortBy=Severity&alert-name=alert1"
				}
			],
			"operator_conditions": [
				{
					"name": "foc1",
					"condition": "ExampleCondition",
					"reason": "Example reason",
					"url": "https://my-cluster.com/k8s/cluster/config.openshift.io~v1~ClusterOperator/foc1"
				}
			]
		}
	},
	"meta": {
		"last_checked_at": "0001-01-01T00:00:00Z"
	},
	"status":"ok"
}
`

const upgradeRecommendedMultiClusterOk = `
{
	"status":"ok",
	"predictions": [
		{
			"cluster_id": "%s",
			"prediction_status": "ok",
			"upgrade_recommended": true,
			"upgrade_risks_predictors": {
				"alerts": [],
				"operator_conditions": []
			},
			"last_checked_at": "0001-01-01T00:00:00Z"
		},
		{
			"cluster_id": "%s",
			"prediction_status": "ok",
			"upgrade_recommended": true,
			"upgrade_risks_predictors": {
				"alerts": [],
				"operator_conditions": []
			},
			"last_checked_at": "0001-01-01T00:00:00Z"
		}
	]
}
`

const upgradeRecommendedMultiClusterTwoOkOneNoData = `
{
	"status":"ok",
	"predictions": [
		{
			"cluster_id": "34c3ecc5-624a-49a5-bab8-4fdc5e51a266",
			"prediction_status": "ok",
			"upgrade_recommended": true,
			"upgrade_risks_predictors": {
				"alerts": [],
				"operator_conditions": []
			},
			"last_checked_at": "0001-01-01T00:00:00Z"
		},
		{
			"cluster_id": "2b9195d4-85d4-428f-944b-4b46f08911f8",
			"prediction_status": "ok",
			"upgrade_recommended": true,
			"upgrade_risks_predictors": {
				"alerts": [],
				"operator_conditions": []
			},
			"last_checked_at": "0001-01-01T00:00:00Z"
		},
		{
			"cluster_id": "aae0ff10-9892-4572-b77f-73eb3e39825f",
			"prediction_status": "No data for the cluster",
			"upgrade_recommended": null,
			"upgrade_risks_predictors": null,
			"last_checked_at": null
		}
	]
}
`

const upgradeMultiClusterTwoClustersNoData = `
{
	"status":"ok",
	"predictions": [
		{
			"cluster_id": "34c3ecc5-624a-49a5-bab8-4fdc5e51a266",
			"prediction_status": "No data for the cluster",
			"upgrade_recommended": null,
			"upgrade_risks_predictors": null,
			"last_checked_at": null
		},
		{
			"cluster_id": "aae0ff10-9892-4572-b77f-73eb3e39825f",
			"prediction_status": "No data for the cluster",
			"upgrade_recommended": null,
			"upgrade_risks_predictors": null,
			"last_checked_at": null
		}
	]
}
`

const multiClusterURPRequestBody = `{"clusters": ["34c3ecc5-624a-49a5-bab8-4fdc5e51a288"]}`

func checkBodyAsMap(t testing.TB, expected, got []byte) {
	var expectedObj, gotObj map[string]interface{}

	err := json.Unmarshal(expected, &expectedObj)
	if err != nil {
		err = fmt.Errorf(`expected is not JSON. value = "%v", err = "%v"`, string(expected), err)
	}
	assert.NoError(t, err)

	err = json.Unmarshal(got, &gotObj)
	if err != nil {
		err = fmt.Errorf(`got is not JSON. value = "%v", err = "%v"`, string(got), err)
	}
	assert.NoError(t, err)

	assert.Equal(
		t,
		expectedObj,
		gotObj,
		fmt.Sprintf(`%v
and
%v
should represent the same json`, string(expected), string(got)),
	)
}

func checkBodyRaw(t testing.TB, expected, got []byte) {
	assert.Equal(t, expected, got)
}

func generateUUIDs(count int) []string {
	uuids := make([]string, count)
	for i := 0; i < count; i++ {
		uuids[i] = uuid.New().String()
	}
	return uuids
}

func TestHTTPServer_GetUpgradeRisksPrediction(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := testdata.GetRandomClusterInfoListAllUnManaged(3)
		cluster := clusterInfoList[0].ID

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		expectedResponse := upgradeRecommended
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.UpgradeRisksPredictionEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     "cluster/{clusterId}/upgrade-risks-prediction",
				EndpointArgs: []interface{}{cluster},
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       testdata.UpgradeRecommended,
			},
		)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.UpgradeRisksPredictionEndpoint,
				EndpointArgs: []interface{}{cluster},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode:  http.StatusOK,
				Body:        expectedResponse,
				BodyChecker: checkBodyAsMap,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetUpgradeRisksPredictionNotRecommended(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := testdata.GetRandomClusterInfoListAllUnManaged(3)
		cluster := clusterInfoList[0].ID

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		expectedResponse := upgradeNotRecommended
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.UpgradeRisksPredictionEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     "cluster/{clusterId}/upgrade-risks-prediction",
				EndpointArgs: []interface{}{cluster},
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       testdata.UpgradeNotRecommended,
			},
		)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.UpgradeRisksPredictionEndpoint,
				EndpointArgs: []interface{}{cluster},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode:  http.StatusOK,
				Body:        expectedResponse,
				BodyChecker: checkBodyAsMap,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetUpgradeRisksPredictionOfflineAMS(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		cluster := testdata.GetRandomClusterInfoListAllUnManaged(1)[0].ID
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.UpgradeRisksPredictionEndpoint,
				EndpointArgs: []interface{}{cluster},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusServiceUnavailable,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetUpgradeRisksPredictionClusterNotBelonging(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := testdata.GetRandomClusterInfoListAllUnManaged(3)
		cluster := testdata.GetRandomClusterInfoListAllUnManaged(1)[0].ID

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.UpgradeRisksPredictionEndpoint,
				EndpointArgs: []interface{}{cluster},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusNotFound,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetUpgradeRisksPredictionNotFound(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := testdata.GetRandomClusterInfoListAllUnManaged(3)
		cluster := clusterInfoList[0].ID

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.UpgradeRisksPredictionEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     "cluster/{clusterId}/upgrade-risks-prediction",
				EndpointArgs: []interface{}{cluster},
			}, &helpers.APIResponse{
				StatusCode: http.StatusNotFound,
			},
		)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.UpgradeRisksPredictionEndpoint,
				EndpointArgs: []interface{}{cluster},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusNotFound,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetUpgradeRisksPredictionInvalidResponse(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := testdata.GetRandomClusterInfoListAllUnManaged(3)
		cluster := clusterInfoList[0].ID

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.UpgradeRisksPredictionEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     "cluster/{clusterId}/upgrade-risks-prediction",
				EndpointArgs: []interface{}{cluster},
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       `this is not a valid response`,
			},
		)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.UpgradeRisksPredictionEndpoint,
				EndpointArgs: []interface{}{cluster},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusBadRequest,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetUpgradeRisksPredictionClusterHasNoData(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := testdata.GetRandomClusterInfoListAllUnManaged(3)
		cluster := clusterInfoList[0].ID

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		helpers.GockExpectAPIRequest(
			t,
			helpers.DefaultServicesConfig.UpgradeRisksPredictionEndpoint,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     "cluster/{clusterId}/upgrade-risks-prediction",
				EndpointArgs: []interface{}{cluster},
			}, &helpers.APIResponse{
				StatusCode: http.StatusNotFound,
				Body:       "No data for the cluster",
			},
		)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.UpgradeRisksPredictionEndpoint,
				EndpointArgs: []interface{}{cluster},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode:  http.StatusNotFound,
				Body:        `No data for the cluster`,
				BodyChecker: checkBodyRaw,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetUpgradeRisksPredictionUnavailableDataEngineering(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		clusterInfoList := testdata.GetRandomClusterInfoListAllUnManaged(3)
		cluster := clusterInfoList[0].ID

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.UpgradeRisksPredictionEndpoint,
				EndpointArgs: []interface{}{cluster},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusServiceUnavailable,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetUpgradeRisksPredictionManagedCluster(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := testdata.GetRandomClusterInfoListAllManaged(1)
		cluster := clusterInfoList[0].ID

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, amsClientMock, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.UpgradeRisksPredictionEndpoint,
				EndpointArgs: []interface{}{cluster},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusNoContent,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetUpgradeRisksPrediction__timesout(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		clusterInfoList := testdata.GetRandomClusterInfoListAllUnManaged(3)
		cluster := clusterInfoList[0].ID

		// prepare response from amsclient for list of clusters
		amsClientMock := helpers.AMSClientWithOrgResults(
			testdata.OrgID,
			clusterInfoList,
		)

		dataEngServer := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(6 * time.Second)
				_, err := fmt.Fprint(w, upgradeRecommended)
				assert.NoError(t, err)
			}))
		defer dataEngServer.Close()

		servicesConfig := helpers.DefaultServicesConfig
		servicesConfig.UpgradeRisksPredictionEndpoint = dataEngServer.URL
		testServer := helpers.CreateHTTPServer(
			&helpers.DefaultServerConfig, &servicesConfig, amsClientMock,
			nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.UpgradeRisksPredictionEndpoint,
				EndpointArgs: []interface{}{cluster},
				XRHIdentity:  goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode: http.StatusServiceUnavailable,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetMulticlusterURPNoBody(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, nil, nil, nil, nil, nil)
		expectedResponse := `{"status":"client didn't provide request body"}`
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:      http.MethodPost,
				Endpoint:    server.UpgradeRisksPredictionMultiClusterEndpoint,
				XRHIdentity: goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode:  http.StatusBadRequest,
				Body:        expectedResponse,
				BodyChecker: checkBodyAsMap,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetMulticlusterUpgradeRisksServiceUnvailable(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, nil, nil, nil, nil, nil)
		expectedResponse := `{"status":"Upgrade Failure Prediction service is unreachable"}`
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:      http.MethodPost,
				Endpoint:    server.UpgradeRisksPredictionMultiClusterEndpoint,
				XRHIdentity: goodXRHAuthToken,
				Body:        helpers.ToJSONString(testdata.ClusterIDListInReq),
			}, &helpers.APIResponse{
				StatusCode:  http.StatusServiceUnavailable,
				Body:        expectedResponse,
				BodyChecker: checkBodyAsMap,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetMulticlusterUpgradeRisksPredictionTwoClusters(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		dataEngServer := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var clusters ctypes.ClusterListInRequest
				err := json.NewDecoder(r.Body).Decode(&clusters)
				assert.NoError(t, err)
				val := fmt.Sprintf(upgradeRecommendedMultiClusterOk, clusters.Clusters[0], clusters.Clusters[1])
				_, err = fmt.Fprint(w, val)
				assert.NoError(t, err)
			}))
		defer dataEngServer.Close()

		servicesConfig := helpers.DefaultServicesConfig
		servicesConfig.UpgradeRisksPredictionEndpoint = dataEngServer.URL
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, &servicesConfig, nil, nil, nil, nil, nil, nil)

		cluster1 := "34c3ecc5-624a-49a5-bab8-4fdc5e51a266"
		cluster2 := "34c3ecc5-624a-49a5-bab8-4fdc5e51a288"
		reqBody := fmt.Sprintf(`{"clusters": ["%s", "%s"]}`, cluster1, cluster2)
		expectedResponse := fmt.Sprintf(upgradeRecommendedMultiClusterOk, cluster1, cluster2)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:      http.MethodPost,
				Endpoint:    server.UpgradeRisksPredictionMultiClusterEndpoint,
				Body:        reqBody,
				XRHIdentity: goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode:  http.StatusOK,
				Body:        expectedResponse,
				BodyChecker: checkBodyAsMap,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetMulticlusterUpgradeRisksPredictionNoData(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		dataEngServer := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var clusters ctypes.ClusterListInRequest
				err := json.NewDecoder(r.Body).Decode(&clusters)
				assert.NoError(t, err)
				_, err = fmt.Fprint(w, upgradeMultiClusterTwoClustersNoData)
				assert.NoError(t, err)
			}))
		defer dataEngServer.Close()

		servicesConfig := helpers.DefaultServicesConfig
		servicesConfig.UpgradeRisksPredictionEndpoint = dataEngServer.URL
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, &servicesConfig, nil, nil, nil, nil, nil, nil)

		// Same as response from data-eng, but omiting empty values
		expectedResponse := `
			{
				"predictions": [
					{
						"cluster_id": "34c3ecc5-624a-49a5-bab8-4fdc5e51a266",
						"prediction_status":"No data for the cluster"
					},
					{
						"cluster_id": "aae0ff10-9892-4572-b77f-73eb3e39825f",
						"prediction_status":"No data for the cluster"
					}
				],
				"status": "ok"
			}
		`
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:      http.MethodPost,
				Endpoint:    server.UpgradeRisksPredictionMultiClusterEndpoint,
				Body:        multiClusterURPRequestBody,
				XRHIdentity: goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode:  http.StatusOK,
				Body:        expectedResponse,
				BodyChecker: checkBodyAsMap,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetMulticlusterUpgradeRisksPredictionOneNoData(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		dataEngServer := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var clusters ctypes.ClusterListInRequest
				err := json.NewDecoder(r.Body).Decode(&clusters)
				assert.NoError(t, err)
				_, err = fmt.Fprint(w, upgradeRecommendedMultiClusterTwoOkOneNoData)
				assert.NoError(t, err)
			}))
		defer dataEngServer.Close()

		servicesConfig := helpers.DefaultServicesConfig
		servicesConfig.UpgradeRisksPredictionEndpoint = dataEngServer.URL

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, &servicesConfig, nil, nil, nil, nil, nil, nil)

		// Same as response from data-eng, but omiting empty values
		expectedResponse := `
			{
				"status":"ok",
				"predictions": [
					{
						"cluster_id": "34c3ecc5-624a-49a5-bab8-4fdc5e51a266",
						"prediction_status": "ok",
						"upgrade_recommended": true,
						"upgrade_risks_predictors": {
							"alerts": [],
							"operator_conditions": []
						},
						"last_checked_at": "0001-01-01T00:00:00Z"
					},
					{
						"cluster_id": "2b9195d4-85d4-428f-944b-4b46f08911f8",
						"prediction_status": "ok",
						"upgrade_recommended": true,
						"upgrade_risks_predictors": {
							"alerts": [],
							"operator_conditions": []
						},
						"last_checked_at": "0001-01-01T00:00:00Z"
					},
					{
						"cluster_id": "aae0ff10-9892-4572-b77f-73eb3e39825f",
						"prediction_status": "No data for the cluster"
					}
				]
			}
		`
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:      http.MethodPost,
				Endpoint:    server.UpgradeRisksPredictionMultiClusterEndpoint,
				Body:        multiClusterURPRequestBody,
				XRHIdentity: goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode:  http.StatusOK,
				Body:        expectedResponse,
				BodyChecker: checkBodyAsMap,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetMulticlusterUpgradeRisksPredictionMaxAllowedClusters(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		dataEngServer := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var clusters ctypes.ClusterListInRequest
				err := json.NewDecoder(r.Body).Decode(&clusters)
				assert.NoError(t, err)
				val := fmt.Sprintf(upgradeRecommendedMultiClusterOk, clusters.Clusters[0], clusters.Clusters[1])
				_, err = fmt.Fprint(w, val)
				assert.NoError(t, err)
			}))
		defer dataEngServer.Close()
		servicesConfig := helpers.DefaultServicesConfig
		servicesConfig.UpgradeRisksPredictionEndpoint = dataEngServer.URL
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, &servicesConfig, nil, nil, nil, nil, nil, nil)

		clusters := generateUUIDs(server.MaxAllowedClusters)
		reqBody := fmt.Sprintf(`{"clusters": ["%s"]}`, strings.Join(clusters, `","`))
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:      http.MethodPost,
				Endpoint:    server.UpgradeRisksPredictionMultiClusterEndpoint,
				Body:        reqBody,
				XRHIdentity: goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode:  http.StatusOK,
				Body:        fmt.Sprintf(upgradeRecommendedMultiClusterOk, clusters[0], clusters[1]),
				BodyChecker: checkBodyAsMap,
			},
		)
	}, testTimeout)
}

func TestHTTPServer_GetMulticlusterURPOverMaxAllowed(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, nil, nil, nil, nil, nil)
		clusters := generateUUIDs(server.MaxAllowedClusters + 1)
		reqBody := fmt.Sprintf(`{"clusters": ["%s"]}`, strings.Join(clusters, `","`))

		expectedResponse := `{"status":"the maximum amount of clusters allowed are 100"}`
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigXRH.APIv2Prefix,
			&helpers.APIRequest{
				Method:      http.MethodPost,
				Endpoint:    server.UpgradeRisksPredictionMultiClusterEndpoint,
				Body:        reqBody,
				XRHIdentity: goodXRHAuthToken,
			}, &helpers.APIResponse{
				StatusCode:  http.StatusBadRequest,
				Body:        expectedResponse,
				BodyChecker: checkBodyAsMap,
			},
		)
	}, testTimeout)
}
