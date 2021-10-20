// Copyright 2020, 2021 Red Hat, Inc
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

// handlers for API V2 endpoints

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/rs/zerolog/log"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-operator-utils/types"

	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"

	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	stypes "github.com/RedHatInsights/insights-results-smart-proxy/types"
)

// getContentForRule retrieves the static content for the given ruleID tied
// with groups info
func (server HTTPServer) getContentWithGroupsForRule(writer http.ResponseWriter, request *http.Request) {
	ruleID, successful := httputils.ReadRuleID(writer, request)
	if !successful {
		// already handled in readRuleID
		return
	}

	ruleContent, err := content.GetRuleContentV2(ruleID)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// check for internal rule permissions
	if internal := content.IsRuleInternal(ruleID); internal == true {
		err := server.checkInternalRulePermissions(request)
		if err != nil {
			handleServerError(writer, err)
			return
		}
	}

	// retrieve the latest groups configuration
	groupsConfig, err := server.getGroupsConfig()
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// prepare data structure for building response
	responseContent := make(map[string]interface{})
	responseContent["status"] = "ok"
	responseContent["groups"] = groupsConfig
	responseContent["content"] = ruleContent

	// send response to client
	err = responses.SendOK(writer, responseContent)
	if err != nil {
		handleServerError(writer, err)
		return
	}
}

// getRecommendations retrieves all recommendations with a count of impacted clusters
// By default returns only those recommendations that currently hit atleast one cluster, but it's
// possible to show all recommendations by passing a URL parameter `impacting`
func (server HTTPServer) getRecommendations(writer http.ResponseWriter, request *http.Request) {
	var recommendationList []stypes.RecommendationListView

	userID, orgID, impactingOnly, err := server.readParamsGetRecommendations(writer, request)
	if err != nil {
		// everything handled
		log.Error().Err(err).Msgf("problem reading necessary params from request")
		return
	}

	// get the list of active clusters if AMS API is available, otherwise from our DB
	clusterList, err := server.readClusterIDsForOrgID(orgID)
	if err != nil {
		log.Error().Err(err).Int(orgIDTag, int(orgID)).Msgf("problem reading cluster list for org")
		handleServerError(writer, err)
		return
	}

	impactingRecommendations, err := server.getImpactingRecommendations(writer, orgID, userID, clusterList)
	if err != nil {
		log.Error().
			Err(err).
			Int(orgIDTag, int(orgID)).
			Str("userID", string(userID)).
			Msgf("problem getting impacting recommendations from aggregator for cluster list: %v", clusterList)

		return
	}

	recommendationList, err = getRecommendationsFillImpacted(impactingRecommendations, impactingOnly)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// TODO: get all ACKS from aggregator, match recommendations, content and acks into the final sruct

	resp := make(map[string]interface{})
	resp["status"] = "ok"
	resp["recommendations"] = recommendationList

	err = responses.SendOK(writer, resp)
	if err != nil {
		handleServerError(writer, err)
		return
	}
}

func getRecommendationsFillImpacted(
	impactingRecommendations types.RecommendationImpactedClusters,
	impactingOnly bool,
) (
	recommendationList []stypes.RecommendationListView,
	err error,
) {
	var ruleIDList []types.RuleID

	if impactingOnly {
		// retrieve content only for impacting rules
		ruleIDList = make([]types.RuleID, len(impactingRecommendations))
		i := 0
		for ruleID := range impactingRecommendations {
			ruleIDList[i] = ruleID
			i++
		}
	} else {
		// retrieve content for all external rules
		ruleIDList, err = content.GetExternalRuleIDs()
		if err != nil {
			log.Error().Err(err).Msg("unable to retrieve external rule ids from content directory")
			return
		}
	}

	// we cannot make the list for len(ruleIDList) because if we go by impacting rules, we
	// might be missing content for some of them
	recommendationList = make([]stypes.RecommendationListView, 0)

	for _, ruleID := range ruleIDList {
		ruleContent, err := content.GetContentForRecommendation(ruleID)
		if err != nil {
			// simply omit the rule as we can't display anything
			log.Error().Err(err).Msgf("unable to get content for rule with id %v", ruleID)
			continue
		}

		var impactingClustersCnt types.ImpactedClustersCnt = 0

		if val, ok := impactingRecommendations[ruleID]; ok {
			impactingClustersCnt = val
		}

		recommendationList = append(recommendationList, stypes.RecommendationListView{
			RuleID:              ruleID,
			Description:         ruleContent.Description,
			Generic:             ruleContent.Generic,
			PublishDate:         ruleContent.PublishDate,
			TotalRisk:           uint8(ruleContent.TotalRisk),
			Impact:              uint8(ruleContent.Impact),
			Likelihood:          uint8(ruleContent.Likelihood),
			Tags:                ruleContent.Tags,
			RuleStatus:          "",
			RiskOfChange:        uint8(ruleContent.RiskOfChange),
			ImpactedClustersCnt: impactingClustersCnt,
		})
	}

	return
}

// getImpactingRecommendations retrieves a list of recommendations from aggregator based on the list of clusters
func (server HTTPServer) getImpactingRecommendations(
	writer http.ResponseWriter,
	orgID types.OrgID,
	userID types.UserID,
	clusterList []types.ClusterName,
) (types.RecommendationImpactedClusters, error) {

	var aggregatorResponse struct {
		Recommendations types.RecommendationImpactedClusters `json:"recommendations"`
		Status          string                               `json:"status"`
	}

	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.RecommendationsListEndpoint,
		orgID,
		userID,
	)

	jsonMarshalled, err := json.Marshal(clusterList)
	if err != nil {
		handleServerError(writer, err)
		return nil, err
	}

	// #nosec G107
	aggregatorResp, err := http.Post(aggregatorURL, JSONContentType, bytes.NewBuffer(jsonMarshalled))
	if err != nil {
		handleServerError(writer, err)
		return nil, err
	}

	responseBytes, err := ioutil.ReadAll(aggregatorResp.Body)
	if err != nil {
		handleServerError(writer, err)
		return nil, err
	}

	if aggregatorResp.StatusCode != http.StatusOK {
		err := responses.Send(aggregatorResp.StatusCode, writer, responseBytes)
		if err != nil {
			handleServerError(writer, err)
		}
		return nil, err
	}

	err = json.Unmarshal(responseBytes, &aggregatorResponse)
	if err != nil {
		handleServerError(writer, err)
		return nil, err
	}

	return aggregatorResponse.Recommendations, nil
}

// getContent retrieves all the static content tied with groups info
func (server HTTPServer) getContentWithGroups(writer http.ResponseWriter, request *http.Request) {
	// Generate an array of RuleContent
	allRules, err := content.GetAllContentV2()
	if err != nil {
		handleServerError(writer, err)
		return
	}

	var rules []stypes.RuleContentV2

	if err := server.checkInternalRulePermissions(request); err != nil {
		for _, rule := range allRules {
			if !content.IsRuleInternal(types.RuleID(rule.Plugin.PythonModule)) {
				rules = append(rules, rule)
			}
		}
	} else {
		rules = allRules
	}

	// retrieve the latest groups configuration
	groupsConfig, err := server.getGroupsConfig()
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// prepare data structure for building response
	responseContent := make(map[string]interface{})
	responseContent["status"] = "ok"
	responseContent["groups"] = groupsConfig
	responseContent["content"] = rules

	// send response to client
	err = responses.SendOK(writer, responseContent)
	if err != nil {
		handleServerError(writer, err)
		return
	}
}
