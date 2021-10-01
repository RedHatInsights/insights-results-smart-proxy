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

package server

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-operator-utils/types"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/rs/zerolog/log"
)

func (server *HTTPServer) postRating(writer http.ResponseWriter, request *http.Request) {
	log.Info().Msg("postRating")

	orgID, userID, err := server.readOrgIDAndUserIDFromToken(writer, request)
	if err != nil {
		return
	}

	log.Info().Int32("org_id", int32(orgID)).Str("user_id", string(userID)).Msg("Extraced user and org")

	rating, succesful := server.postRatingToAggregator(orgID, userID, request, writer)
	if !succesful {
		log.Error().Msg("Unable to get response from aggregator")
		// All errors already handled
	}

	bodyContent, err := json.Marshal(rating)
	if err != nil {
		log.Error().Err(err).Msg("Unable to unmarshall the response from aggregator")
		handleServerError(writer, err)
	}

	err = responses.Send(http.StatusOK, writer, bodyContent)
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

func (server HTTPServer) postRatingToAggregator(
	orgID types.OrgID, userID types.UserID, request *http.Request, writer http.ResponseWriter,
) (*types.RuleRating, bool) {
	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.Rating,
		orgID,
		userID,
	)

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}
	// #nosec G107
	aggregatorResp, err := http.Post(aggregatorURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

	var aggregatorResponse struct {
		Rating types.RuleRating `json:"ratings"`
		Status string           `json:"status"`
	}

	err = json.NewDecoder(aggregatorResp.Body).Decode(&aggregatorResponse)
	if err != nil {
		log.Error().Err(err).Msg("Unable to understand aggregator's reponse")
		handleServerError(writer, err)
	}

	return &aggregatorResponse.Rating, true
}
