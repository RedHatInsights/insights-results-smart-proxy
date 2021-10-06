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

	ruleContent, err := content.GetRuleContent(ruleID)
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
	var impactingOnly = true

	authToken, err := server.GetAuthToken(request)
	if err != nil {
		handleServerError(writer, err)
		return
	}
	userID := authToken.AccountNumber

	orgID, successful := httputils.ReadOrganizationID(writer, request, server.Config.Auth)
	if !successful {
		// already handled in readOrganizationID ?
		return
	}

	// get the list of active clusters if AMS API is available
	clusterList, err := server.readClusterIDsForOrgID(orgID)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	impactingParam, err := readImpactingParam(request)
	if err != nil {
		log.Err(err).Msgf("Error parsing `%s` URL parameter. Defaulting to true.", ImpactingParam)
	} else {
		impactingOnly = impactingParam
	}

	impactingRecommendations, err := server.getImpactingRecommendations(writer, orgID, userID, clusterList)

	recommendationList = make([]stypes.RecommendationListView, 0)

	if impactingOnly {
		// retrieve content only for impacting recommendations
		for ruleID, impactingClustersCnt := range impactingRecommendations {
			ruleContent, err := content.GetRecommendationContent(ruleID)
			if err != nil {
				log.Error().Err(err).Msgf("unable to get content for rule with id %v", ruleID)
				continue
			}

			recommendationList = append(recommendationList, stypes.RecommendationListView{
				RuleID:              ruleID,
				Description:         ruleContent.Description,
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
	} else {
		// retrieve content for all external rules and fill in impacted clusters
		externalRuleIDs := content.GetExternalRuleIDs()

		for _, ruleID := range externalRuleIDs {
			ruleContent, err := content.GetRecommendationContent(ruleID)
			if err != nil {
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
		"recommendations/organizations/{org_id}/users/{user_id}/list", // FIXME
		orgID,
		userID,
	)

	jsonMarshalled, err := json.Marshal(clusterList)
	if err != nil {
		handleServerError(writer, err)
		return nil, nil
	}

	// #nosec G107
	aggregatorResp, err := http.Post(aggregatorURL, "application/json", bytes.NewBuffer(jsonMarshalled))
	if err != nil {
		handleServerError(writer, err)
		return nil, nil
	}

	responseBytes, err := ioutil.ReadAll(aggregatorResp.Body)
	if err != nil {
		handleServerError(writer, err)
		return nil, nil
	}

	if aggregatorResp.StatusCode != http.StatusOK {
		err := responses.Send(aggregatorResp.StatusCode, writer, responseBytes)
		if err != nil {
			handleServerError(writer, err)
		}
		return nil, nil
	}

	err = json.Unmarshal(responseBytes, &aggregatorResponse)
	if err != nil {
		handleServerError(writer, err)
		return nil, nil
	}

	return aggregatorResponse.Recommendations, nil
}

// getContent retrieves all the static content tied with groups info
func (server HTTPServer) getContentWithGroups(writer http.ResponseWriter, request *http.Request) {
	// Generate an array of RuleContent
	allRules := content.GetAllContent()
	var rules []types.RuleContent

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
