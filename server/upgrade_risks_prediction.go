// Copyright 2023 Red Hat, Inc
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

package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/RedHatInsights/insights-results-smart-proxy/amsclient"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-operator-utils/responses"
	utypes "github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"
	"github.com/RedHatInsights/insights-results-smart-proxy/types"

	"github.com/rs/zerolog/log"
)

// method upgradeRisksPrediction returns a recommendation to upgrade or not a cluster
// and a list of the alerts/operator conditions that were taken into account if the
// upgrade is not recommended.
//
// Response format should look like:
//
//		{
//			"upgrade_recommended": false,
//			"upgrade_risks_predictors": {
//				"alerts": [
//					{
//						"name": "APIRemovedInNextEUSReleaseInUse",
//						"namespace": "openshift-kube-apiserver",
//						"severity": "info",
//	                 "url": "${CONSOLE_URL}/monitoring/alerts?orderBy=asc&sortBy=Severity&alert-name=${ALERT_NAME}"
//					}
//				],
//				"operator_conditions": [
//					{
//						"name": "authentication",
//						"condition": "Failing",
//						"reason": "AsExpected",
//	                 "url": "${CONSOLE_URL}/k8s/cluster/config.openshift.io~v1~ClusterOperator/${OPERATOR_NAME}"
//					}
//				]
//			}
//		}
func (server *HTTPServer) upgradeRisksPrediction(writer http.ResponseWriter, request *http.Request) {
	if server.amsClient == nil {
		log.Error().Msgf(AMSApiNotInitializedErrorMessage)
		handleServerError(writer, &AMSAPIUnavailableError{})
		return
	}

	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	clusterID, successful := httputils.ReadClusterName(writer, request)
	// error handled by function
	if !successful {
		return
	}

	clusterInfo, err := server.amsClient.GetSingleClusterInfoForOrganization(orgID, clusterID)

	if err != nil {
		log.Error().Err(err).Str(clusterIDTag, string(clusterID)).Msg("failure retrieving the cluster's organization")
		handleServerError(writer, err)
		return
	} else if clusterInfo.ID != clusterID {
		log.Error().Err(err).Str(clusterIDTag, string(clusterID)).Msg("cluster doesn't belong to the expected org")
		handleServerError(writer, &utypes.ItemNotFoundError{ItemID: clusterID})
		return
	}

	if clusterInfo.Managed {
		log.Error().Err(err).Str(clusterIDTag, string(clusterID)).Msg("cluster doesn't belong to the expected org")
		handleServerError(writer, &utypes.NoContentError{
			ErrString: "the upgrade failure prediction service is not available for managed clusters",
		})
		return
	}

	// Request to Data Engineering Service to retrieve the result
	predictionResponse, err := server.fetchUpgradePrediction(clusterID, writer)
	if err != nil || predictionResponse == nil {
		// Error already handled or not OK status, already returned
		return
	}

	response := make(map[string]interface{})
	response["upgrade_recommendation"] = types.UpgradeRecommendation{
		Recommended:     predictionResponse.Recommended,
		RisksPredictors: predictionResponse.RisksPredictors,
	}
	response["status"] = OkMsg

	response["meta"] = types.UpgradeRisksMeta{
		LastCheckedAt: predictionResponse.LastCheckedAt,
	}

	err = responses.SendOK(
		writer,
		response,
	)
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// upgradeRisksPredictionMultiClusters sends a response with a list of predictions for each cluster.
// Each prediction will have a result (true or false) and a list of the alerts and operator conditions
// that were taken into account for non-recommended upgrades.
func (server *HTTPServer) upgradeRisksPredictionMultiCluster(writer http.ResponseWriter, request *http.Request) {
	if server.amsClient == nil {
		log.Error().Msgf(AMSApiNotInitializedErrorMessage)
		handleServerError(writer, &AMSAPIUnavailableError{})
		return
	}

	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// try to read list of cluster IDs
	listOfClusters, successful := httputils.ReadClusterListFromBody(writer, request)
	if !successful {
		// wrong state has been handled already
		return
	}

	numberOfClusers := len(listOfClusters)
	// Limit number of Clusters to 100.
	// If we go over this limit, the way we handle the response from AMS should
	// be modified (default page size is 1 with a size of 100).
	if numberOfClusers > 100 {
		err := responses.SendBadRequest(writer, "Request body should not contain more than 100 cluster IDs")
		if err != nil {
			log.Error().Err(err).Msg(responseDataError)
		}
	}

	clustersInfo, err := server.amsClient.GetMultiClusterInfoForOrganization(orgID, listOfClusters, nil, nil)
	if err != nil {
		log.Error().Err(err).Uint32(orgIDTag, uint32(orgID)).Msg("failure retrieving the clusters information from AMS")
		handleServerError(writer, err)
		return
	}

	managed, unmanaged := amsclient.FilterManagedClusters(clustersInfo)

	log.Info().
		Uint32(orgIDTag, uint32(orgID)).
		Int("IDS count", len(managed)).
		Msg("Number of managed clusters removed from origin request")

	// Request to Data Engineering Service to retrieve the result
	predictionResponse, err := server.fetchMulticlusterUpgradePrediction(unmanaged, writer)
	if err != nil || predictionResponse == nil {
		// Error already handled or not OK status, already returned
		return
	}

	// prepare response
	response := make(map[string]interface{})
	response["status"] = predictionResponse.Status
	predictions := make([]types.UpgradeRisksPrediction, numberOfClusers)

	copy(predictions, predictionResponse.Predictions)

	//  managed clusters to the response with a "OK" status and no prediction nor alerts

	nextPrediction := len(predictionResponse.Predictions)
	for _, cluster := range managed {
		predictions[nextPrediction] = types.UpgradeRisksPrediction{
			PredictionStatus: OkMsg,
			ClusterID:        types.ClusterName(cluster),
		}
		nextPrediction++
	}

	response["predictions"] = predictions

	// send it all
	err = responses.SendOK(
		writer,
		response,
	)
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

func (server *HTTPServer) fetchUpgradePrediction(
	cluster types.ClusterName,
	writer http.ResponseWriter,
) (*types.DataEngResponse, error) {
	dataEngURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.UpgradeRisksPredictionEndpoint,
		UpgradeRisksPredictionEndpoint,
		cluster,
	)

	httpClient := http.Client{
		Timeout: 5 * time.Second,
	}

	// #nosec G107
	// nolint:bodyclose // TODO: remove once the bodyclose library fixes this bug
	response, err := httpClient.Get(dataEngURL)
	if err != nil {
		log.Error().
			Str(clusterIDTag, string(cluster)).
			Err(err).
			Msg("error reaching the data-eng service")
		handleServerError(writer, &UpgradesDataEngServiceUnavailableError{})
		return nil, err
	}

	defer services.CloseResponseBody(response)

	responseBytes, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error().
			Str(clusterIDTag, string(cluster)).
			Err(err).
			Msg("unable to read the body of the response")
		handleServerError(writer, err)
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		err := responses.Send(response.StatusCode, writer, responseBytes)
		if err != nil {
			log.Error().Err(err).Msg(responseDataError)
		}
		return nil, err
	}
	responseData := &types.DataEngResponse{}
	err = json.Unmarshal(responseBytes, &responseData)
	if err != nil {
		log.Error().Str(clusterIDTag, string(cluster)).Err(err).Msg("error unmarshalling data-engineering response")
		handleServerError(writer, err)
		return nil, err
	}

	return responseData, nil
}

func (server *HTTPServer) fetchMulticlusterUpgradePrediction(
	clusters []string,
	writer http.ResponseWriter,
) (*types.UpgradeRisksRecommendations, error) {
	dataEngURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.UpgradeRisksPredictionEndpoint,
		UpgradeRisksPredictionMultiClusterEndpoint,
	)

	clustersMap := make(map[string]interface{})
	clustersMap["clusters"] = clusters

	jsonBody, err := json.Marshal(clustersMap)
	if err != nil {
		log.Error().Err(err).Str("url", dataEngURL).Msg("error marshalling request body")
		handleServerError(writer, err)
		return nil, err
	}

	httpClient := http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, dataEngURL, bytes.NewReader(jsonBody))
	if err != nil {
		panic(err)
	}

	// #nosec G107
	// nolint:bodyclose // TODO: remove once the bodyclose library fixes this bug
	response, err := httpClient.Do(req)
	if err != nil {
		log.Error().
			Err(err).
			Msg("error reaching the data-eng service")
		handleServerError(writer, &UpgradesDataEngServiceUnavailableError{})
		return nil, err
	}

	defer services.CloseResponseBody(response)

	responseBytes, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error().
			Err(err).
			Msg("unable to read the body of the response")
		handleServerError(writer, err)
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		err := responses.Send(response.StatusCode, writer, responseBytes)
		if err != nil {
			log.Error().Err(err).Msg(responseDataError)
		}
		return nil, err
	}
	responseData := &types.UpgradeRisksRecommendations{}
	err = json.Unmarshal(responseBytes, &responseData)
	if err != nil {
		log.Error().Err(err).Str("url", dataEngURL).Msg("error unmarshalling data-engineering response")
		handleServerError(writer, err)
		return nil, err
	}

	return responseData, nil
}
