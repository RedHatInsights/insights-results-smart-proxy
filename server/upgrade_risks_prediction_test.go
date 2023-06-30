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
	"testing"
	"time"

	iou_helpers "github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/testdata"
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
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfigXRH, nil, amsClientMock, nil, nil, nil, nil)

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
			serverConfigJWT.APIv2Prefix,
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
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfigXRH, nil, amsClientMock, nil, nil, nil, nil)

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
			serverConfigJWT.APIv2Prefix,
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
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfigXRH, nil, nil, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
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

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfigXRH, nil, amsClientMock, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
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
		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfigXRH, nil, amsClientMock, nil, nil, nil, nil)

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
			serverConfigJWT.APIv2Prefix,
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

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfigXRH, nil, amsClientMock, nil, nil, nil, nil)
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
			serverConfigJWT.APIv2Prefix,
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

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfigXRH, nil, amsClientMock, nil, nil, nil, nil)
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
			serverConfigJWT.APIv2Prefix,
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

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfigXRH, nil, amsClientMock, nil, nil, nil, nil)
		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
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

		testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfigXRH, nil, amsClientMock, nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
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
				fmt.Fprint(w, upgradeRecommended)
			}))
		defer dataEngServer.Close()

		servicesConfig := helpers.DefaultServicesConfig
		servicesConfig.UpgradeRisksPredictionEndpoint = dataEngServer.URL
		testServer := helpers.CreateHTTPServer(
			&helpers.DefaultServerConfigXRH, &servicesConfig, amsClientMock,
			nil, nil, nil, nil)

		iou_helpers.AssertAPIRequest(
			t,
			testServer,
			serverConfigJWT.APIv2Prefix,
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
