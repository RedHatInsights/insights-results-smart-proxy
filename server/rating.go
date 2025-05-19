// Copyright 2021, 2022 Red Hat, Inc
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
	"fmt"
	"io"
	"net/http"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-operator-utils/responses"
	utypes "github.com/RedHatInsights/insights-operator-utils/types"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"
	ctypes "github.com/RedHatInsights/insights-results-types"
	"github.com/rs/zerolog/log"
)

// postRating handles the POST method for Rating endpoint
func (server *HTTPServer) postRating(writer http.ResponseWriter, request *http.Request) {
	log.Debug().Msg("postRating")

	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	log.Debug().Uint32("org_id", uint32(orgID)).Msg("Extracted user and org")

	rating, successful := server.postRatingToAggregator(orgID, request, writer)
	if !successful {
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

// postRatingToAggregator asks aggregator for update the rating for a given rule by the current user/org
func (server HTTPServer) postRatingToAggregator(
	orgID ctypes.OrgID, request *http.Request, writer http.ResponseWriter,
) (*ctypes.RuleRating, bool) {
	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.Rating,
		orgID,
	)

	body, err := io.ReadAll(request.Body)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}
	// #nosec G107
	// nolint:bodyclose // TODO: remove once the bodyclose library fixes this bug
	aggregatorResp, err := http.Post(aggregatorURL, JSONContentType, bytes.NewBuffer(body))
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

	defer services.CloseResponseBody(aggregatorResp)

	var aggregatorResponse struct {
		Rating ctypes.RuleRating `json:"ratings"`
		Status string            `json:"status"`
	}

	err = json.NewDecoder(aggregatorResp.Body).Decode(&aggregatorResponse)
	if err != nil {
		log.Error().Err(err).Msg("Unable to understand aggregator's response")
		handleServerError(writer, err)
	}

	return &aggregatorResponse.Rating, true
}

// getRatingForRecommendation retrieves user rating for recommendation from aggregator
func (server HTTPServer) getRatingForRecommendation(
	orgID ctypes.OrgID,
	ruleID ctypes.RuleID,
) (
	ruleRating ctypes.RuleRating,
	err error,
) {
	ruleRating.Rule = string(ruleID)
	ruleRating.Rating = 0

	var aggregatorResponse struct {
		Rating ctypes.RuleRating `json:"rating"`
		Status string            `json:"status"`
	}

	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.GetRating,
		ruleID,
		orgID,
	)

	// #nosec G107
	// nolint:bodyclose // TODO: remove once the bodyclose library fixes this bug
	aggregatorResp, err := http.Get(aggregatorURL)
	if err != nil {
		log.Error().Err(err).Str(urlStr, aggregatorURL).Msg("problem getting URL from aggregator")
		return
	}

	defer services.CloseResponseBody(aggregatorResp)

	responseBytes, err := io.ReadAll(aggregatorResp.Body)
	if err != nil {
		log.Error().Err(err).Str(urlStr, aggregatorURL).Msg("problem reading response from URL from aggregator")
		return
	}

	if aggregatorResp.StatusCode == http.StatusNotFound {
		log.Debug().Msgf("rule rating for rule %v not found", ruleID)
		return ruleRating, &utypes.ItemNotFoundError{}
	}

	if aggregatorResp.StatusCode != http.StatusOK {
		err = fmt.Errorf(
			"problem retrieving rating from aggregator for rule %v. Status code: %v",
			ruleID,
			aggregatorResp.StatusCode,
		)
		log.Error().Err(err).Send()
		return
	}

	err = json.Unmarshal(responseBytes, &aggregatorResponse)
	if err != nil {
		log.Error().Err(err).Msg("problem unmarshalling aggregator response")
		return
	}

	return aggregatorResponse.Rating, nil
}
