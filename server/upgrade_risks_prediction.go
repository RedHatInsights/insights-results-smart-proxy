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
	"encoding/json"
	"io"
	"net/http"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-operator-utils/responses"
	utypes "github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/RedHatInsights/insights-results-smart-proxy/types"

	"github.com/rs/zerolog/log"
)

// UpgradeRisksPredictionServiceEndpoint endport for the upgrade prediction service
const UpgradeRisksPredictionServiceEndpoint = "upgrade-risks-prediction/cluster/{cluster}"

// method upgradeRisksPrediction return a recommendation to upgrade or not a cluster
// and a list of the alerts/operator conditions that were taken into account if the
// upgrade is not recommended.
//
// Response format should look like:
//
//	{
//		"upgrade_recommended": false,
//		"upgrade_risks_predictors": {
//			"alerts": ["alert1", "alert2"],
//			"operator_conditions": ["foc1", "foc2"]
//		}
//	}
func (server *HTTPServer) upgradeRisksPrediction(writer http.ResponseWriter, request *http.Request) {
	if server.amsClient == nil {
		log.Error().Msgf("AMS API connection is not initialized")
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

	isOwner := server.amsClient.IsClusterInOrganization(orgID, clusterID)

	if !isOwner {
		handleServerError(writer, &utypes.ItemNotFoundError{ItemID: clusterID})
		return
	}

	// Request to Data Engineering Service to retrieve the result
	predictionResponse, successful := server.fetchPrediction(clusterID, writer)
	if !successful || predictionResponse == nil {
		// Error already handled or not OK status, already returned
		return
	}

	err = responses.SendOK(
		writer,
		responses.BuildOkResponseWithData(
			"upgrade_recommendation", predictionResponse,
		),
	)
	if err != nil {
		if err != nil {
			log.Error().Err(err).Msg(responseDataError)
		}
	}
}

func (server *HTTPServer) fetchPrediction(
	cluster types.ClusterName,
	writer http.ResponseWriter,
) (*types.UpgradeRecommendation, bool) {
	dataEngURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.UpgradeRisksPredictionEndpoint,
		UpgradeRisksPredictionServiceEndpoint,
		cluster,
	)

	response, err := http.Get(dataEngURL)
	if err != nil {
		log.Error().Str(clusterIDTag, string(cluster)).Err(err).Msg("fetchPrediction unexpected error for cluster")
		handleServerError(writer, err)
		return nil, false
	}

	responseBytes, err := io.ReadAll(response.Body)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

	if response.StatusCode != http.StatusOK {
		err := responses.Send(response.StatusCode, writer, responseBytes)
		if err != nil {
			log.Error().Err(err).Msg(responseDataError)
		}
		return nil, false
	}
	responseData := &types.UpgradeRecommendation{}
	err = json.Unmarshal(responseBytes, &responseData)
	if err != nil {
		log.Error().Str(clusterIDTag, string(cluster)).Err(err).Msg("error unmarshalling data-engineering response")
		handleServerError(writer, err)
		return nil, false
	}

	return responseData, true
}
