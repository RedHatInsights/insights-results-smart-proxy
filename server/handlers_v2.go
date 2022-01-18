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
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-content-service/groups"
	"github.com/RedHatInsights/insights-operator-utils/generators"
	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-operator-utils/responses"
	utypes "github.com/RedHatInsights/insights-operator-utils/types"
	ctypes "github.com/RedHatInsights/insights-results-types"

	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"

	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	"github.com/RedHatInsights/insights-results-smart-proxy/types"
)

const (
	// OnlyImpacting flag to only return impacting recommendations on GET /rule/
	OnlyImpacting = iota
	// IncludingImpacting flag to return all recommendations including impacting ones on GET /rule/
	IncludingImpacting
	// ExcludingImpacting flag to return all recommendations excluding impacting ones on GET /rule/
	ExcludingImpacting
	// OkMsg is in status field with HTTP 200 response
	OkMsg       = "ok"
	selectorStr = "selector"
)

// getContentCheckInternal retrieves static content for the given ruleID and if the rule is internal,
// checks if user has permissions to access it.
func (server HTTPServer) getContentCheckInternal(ruleID ctypes.RuleID, request *http.Request) (
	ruleContent *types.RuleWithContent,
	err error,
) {
	ruleContent, err = content.GetContentForRecommendation(ruleID)
	if err != nil {
		return
	}

	// check for internal rule permissions
	if internal := content.IsRuleInternal(ruleID); internal == true {
		err = server.checkInternalRulePermissions(request)
		if err != nil {
			return
		}
	}

	return
}

// getRuleWithGroups retrieves static content for the given ruleID along with rule groups
func (server HTTPServer) getRuleWithGroups(
	writer http.ResponseWriter,
	request *http.Request,
	ruleID ctypes.RuleID,
) (
	ruleContent *types.RuleWithContent,
	ruleGroups []groups.Group,
	err error,
) {
	ruleContent, err = server.getContentCheckInternal(ruleID, request)
	if err != nil {
		log.Error().Msgf("error retrieving rule content for rule ID %v", ruleID)
		return
	}

	// retrieve the latest groups configuration
	ruleGroups, err = server.getGroupsConfig()
	if err != nil {
		log.Error().Msgf("error retrieving rule groups")
		return
	}

	return
}

// getRecommendationContent retrieves the static content for the given ruleID tied
// with groups info. rule ID is expected to be the composite rule ID (rule.module|ERROR_KEY)
func (server HTTPServer) getRecommendationContent(writer http.ResponseWriter, request *http.Request) {
	ruleID, err := readCompositeRuleID(writer, request)
	if err != nil {
		log.Error().Err(err).Msgf("error retrieving rule ID from request")
		handleServerError(writer, err)
		return
	}

	ruleContent, ruleGroups, err := server.getRuleWithGroups(writer, request, ruleID)
	if err != nil {
		log.Error().Err(err).Msgf("error retrieving rule content and groups for rule ID %v", ruleID)
		handleServerError(writer, err)
		return
	}

	contentResponse := types.RecommendationContent{
		// RuleID in rule.module|ERROR_KEY format
		RuleSelector: ctypes.RuleSelector(ruleID),
		Description:  ruleContent.Description,
		Generic:      ruleContent.Generic,
		Reason:       ruleContent.Reason,
		Resolution:   ruleContent.Resolution,
		MoreInfo:     ruleContent.MoreInfo,
		TotalRisk:    uint8(ruleContent.TotalRisk),
		RiskOfChange: uint8(ruleContent.RiskOfChange),
		Impact:       uint8(ruleContent.Impact),
		Likelihood:   uint8(ruleContent.Likelihood),
		PublishDate:  ruleContent.PublishDate,
		Tags:         ruleContent.Tags,
	}

	// prepare data structure for building response
	responseContent := make(map[string]interface{})
	responseContent["status"] = OkMsg
	responseContent["groups"] = ruleGroups
	responseContent["content"] = contentResponse

	// send response to client
	err = responses.SendOK(writer, responseContent)
	if err != nil {
		handleServerError(writer, err)
		return
	}
}

// getRecommendationContent retrieves the static content for the given ruleID tied
// with groups info. rule ID is expected to be the composite rule ID (rule.module|ERROR_KEY)
func (server HTTPServer) getRecommendationContentWithUserData(writer http.ResponseWriter, request *http.Request) {
	orgID, userID, err := server.readOrgIDAndUserIDFromToken(writer, request)
	if err != nil {
		log.Err(err).Msg(orgIDTokenError)
		return
	}

	ruleID, err := readCompositeRuleID(writer, request)
	if err != nil {
		log.Error().Err(err).Msgf("error retrieving rule ID from request")
		handleServerError(writer, err)
		return
	}

	ruleContent, ruleGroups, err := server.getRuleWithGroups(writer, request, ruleID)
	if err != nil {
		log.Error().Err(err).Msgf("error retrieving rule content and groups for rule ID %v", ruleID)
		handleServerError(writer, err)
		return
	}

	rating, err := server.getRatingForRecommendation(writer, orgID, userID, ruleID)
	if err != nil {
		switch err.(type) {
		case *utypes.ItemNotFoundError:
			break
		case *url.Error:
			log.Error().Err(err).Msgf("aggregator is not responding")
			handleServerError(writer, &AggregatorServiceUnavailableError{})
			return
		default:
			handleServerError(writer, err)
			return
		}
	}

	ruleModule, errorKey, err := types.RuleIDWithErrorKeyFromCompositeRuleID(ruleID)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// ignoring the response and the possible error.
	// We are just interested on know if the rule is system disabled or not
	_, ackFound, _ := server.readRuleDisableStatus(
		ctypes.Component(ruleModule),
		errorKey,
		orgID,
		userID,
	)

	// fill in user rating and other DB stuff from aggregator
	contentResponse := types.RecommendationContentUserData{
		// RuleID in rule.module|ERROR_KEY format
		RuleSelector: ctypes.RuleSelector(ruleID),
		Description:  ruleContent.Description,
		Generic:      ruleContent.Generic,
		Reason:       ruleContent.Reason,
		Resolution:   ruleContent.Resolution,
		MoreInfo:     ruleContent.MoreInfo,
		TotalRisk:    uint8(ruleContent.TotalRisk),
		RiskOfChange: uint8(ruleContent.RiskOfChange),
		Impact:       uint8(ruleContent.Impact),
		Likelihood:   uint8(ruleContent.Likelihood),
		PublishDate:  ruleContent.PublishDate,
		Rating:       rating.Rating,
		AckedCount:   0,
		Tags:         ruleContent.Tags,
		Disabled:     ackFound,
	}

	// prepare data structure for building response
	responseContent := make(map[string]interface{})
	responseContent["status"] = OkMsg
	responseContent["groups"] = ruleGroups
	responseContent["content"] = contentResponse

	// send response to client
	err = responses.SendOK(writer, responseContent)
	if err != nil {
		log.Error().Err(err).Msgf(problemSendingResponseError)
		handleServerError(writer, err)
		return
	}
}

// getRecommendations retrieves all recommendations with a count of impacted clusters
// By default returns only those recommendations that currently hit atleast one cluster, but it's
// possible to show all recommendations by passing a URL parameter `impacting`
func (server HTTPServer) getRecommendations(writer http.ResponseWriter, request *http.Request) {
	var recommendationList []types.RecommendationListView
	tStart := time.Now()

	userID, orgID, impactingFlag, err := server.readParamsGetRecommendations(writer, request)
	if err != nil {
		// everything handled
		log.Error().Err(err).Msgf("problem reading necessary params from request")
		return
	}
	log.Info().Int(orgIDTag, int(orgID)).Str(userIDTag, string(userID)).Msg("getRecommendations start")

	// get the list of active clusters if AMS API is available, otherwise from our DB
	clusterList, err := server.readClusterIDsForOrgID(orgID)
	if err != nil {
		log.Error().Err(err).Int(orgIDTag, int(orgID)).Msgf("problem reading cluster list for org")
		handleServerError(writer, err)
		return
	}

	tStartImpacting := time.Now()
	impactingRecommendations, err := server.getImpactingRecommendations(writer, orgID, userID, clusterList)
	if err != nil {
		log.Error().
			Err(err).
			Int(orgIDTag, int(orgID)).
			Str(userIDTag, string(userID)).
			Msgf("problem getting impacting recommendations from aggregator for cluster list: %v", clusterList)

		return
	}
	log.Info().Uint32(orgIDTag, uint32(orgID)).Msgf(
		"getRecommendations get impacting recommendations from aggregator took %s", time.Since(tStartImpacting),
	)

	// retrieve rule acknowledgements (disable/enable)
	acks, err := server.readListOfAckedRules(orgID, userID)
	if err != nil {
		log.Error().Err(err).Msg(ackedRulesError)
		// server error has been handled already
		return
	}

	recommendationList, err = getRecommendationsFillUserData(impactingRecommendations, impactingFlag, acks)
	if err != nil {
		log.Error().Err(err).Msg("problem getting recommendation content")
		handleServerError(writer, err)
		return
	}
	log.Info().
		Int(orgIDTag, int(orgID)).
		Str(userIDTag, string(userID)).
		Msgf("number of final recommendations: %d", len(recommendationList))

	resp := make(map[string]interface{})
	resp["status"] = OkMsg
	resp["recommendations"] = recommendationList

	log.Info().Uint32(orgIDTag, uint32(orgID)).Msgf(
		"getRecommendations took %s", time.Since(tStart),
	)
	err = responses.SendOK(writer, resp)
	if err != nil {
		log.Error().Err(err).Msgf(problemSendingResponseError)
		handleServerError(writer, err)
		return
	}
}

func (server HTTPServer) getRuleAcksAndDisables(orgID types.OrgID, userID types.UserID) (
	ackedRulesMap map[ctypes.RuleID]bool,
	disabledRulesPerCluster map[ctypes.ClusterName][]ctypes.RuleID,
) {
	ackedRulesMap = make(map[ctypes.RuleID]bool)
	disabledRulesPerCluster = make(map[ctypes.ClusterName][]ctypes.RuleID)

	// retrieve rule acknowledgements (disable/enable for all clusters)
	ackedRules, err := server.readListOfAckedRules(orgID, userID)
	if err != nil {
		log.Error().Err(err).Msg(ackedRulesError)
		// server error has been handled already
		return
	}
	// put rule acks in a map so we only iterate over them once
	ackedRulesMap = generateRuleAckMap(ackedRules)

	// retrieve list of cluster IDs and single disabled rules for each cluster
	disabledRulesPerCluster = server.getUserDisabledRulesPerCluster(userID)

	return
}

// getClustersView retrieves all clusters for given organization, retrieves the impacting rules for each cluster
// from aggregator and returns a list of clusters, total number of hitting rules and a count of impacting rules
// by severity = total risk = critical, high, moderate, low
func (server HTTPServer) getClustersView(writer http.ResponseWriter, request *http.Request) {
	tStart := time.Now()

	orgID, userID, err := server.readOrgIDAndUserIDFromToken(writer, request)
	if err != nil {
		log.Err(err).Msg(orgIDTokenError)
		return
	}
	log.Info().Int(orgIDTag, int(orgID)).Str(userIDTag, string(userID)).Msg("getClustersView start")

	// get a list of clusters from AMS API
	clusterInfoList, err := server.readClusterInfoForOrgID(orgID)
	if err != nil {
		log.Error().Err(err).Int(orgIDTag, int(orgID)).Msg("problem reading cluster list for org")
		handleServerError(writer, err)
		return
	}
	log.Info().Uint32(orgIDTag, uint32(orgID)).Msgf(
		"getClustersView list of clusters %s", clusterInfoList)

	tStartImpacting := time.Now()
	clusterRecommendationMap, err := server.getClustersAndRecommendations(writer, orgID, userID, types.GetClusterNames(clusterInfoList))
	if err != nil {
		log.Error().
			Err(err).
			Int(orgIDTag, int(orgID)).
			Str(userIDTag, string(userID)).
			Msgf("problem getting clusters and impacting recommendations from aggregator for cluster list: %v", clusterInfoList)

		return
	}
	log.Info().Uint32(orgIDTag, uint32(orgID)).Msgf(
		"getClustersView getting clusters and impacting recommendations from aggregator took %s", time.Since(tStartImpacting),
	)

	ackedRulesMap, disabledRulesPerCluster := server.getRuleAcksAndDisables(orgID, userID)

	clusterViewResponse, err := matchClusterInfoAndUserData(
		clusterInfoList, clusterRecommendationMap, ackedRulesMap, disabledRulesPerCluster,
	)
	if err != nil {
		log.Error().Uint32(orgIDTag, uint32(orgID)).Err(err).Msg("error matching cluster list and rule severities")
		handleServerError(writer, err)
	}
	log.Info().Uint32(orgIDTag, uint32(orgID)).Msgf("getClustersView final number %v", len(clusterViewResponse))

	resp := make(map[string]interface{})
	metaCount := map[string]int{
		"count": len(clusterViewResponse),
	}
	resp["status"] = OkMsg
	resp["meta"] = metaCount
	resp["data"] = clusterViewResponse

	log.Info().Uint32(orgIDTag, uint32(orgID)).Msgf("getClustersView took %s", time.Since(tStart))

	err = responses.SendOK(writer, resp)
	if err != nil {
		log.Error().Err(err).Msgf(problemSendingResponseError)
		handleServerError(writer, err)
		return
	}
}

// matchClusterInfoAndUserData matches data from AMS API, rule hits from aggregator + user data from aggregator
// regarding disabled rules and calculates the numbers of hitting rules based on their severity (total risk)
func matchClusterInfoAndUserData(
	clusterInfoList []types.ClusterInfo,
	clusterRecommendationsMap ctypes.ClusterRecommendationMap,
	systemWideDisabledRules map[ctypes.RuleID]bool,
	disabledRulesPerCluster map[ctypes.ClusterName][]ctypes.RuleID,
) (
	[]types.ClusterListView, error,
) {
	clusterListView := make([]types.ClusterListView, 0)

	recommendationSeverities, uniqueSeverities, err := content.GetExternalRuleSeverities()
	if err != nil {
		return clusterListView, err
	}

	// iterates over clusters and their hitting recommendations, accesses map to the get rule severity
	for i := range clusterInfoList {
		clusterViewItem := types.ClusterListView{
			ClusterID:       clusterInfoList[i].ID,
			ClusterName:     clusterInfoList[i].DisplayName,
			HitsByTotalRisk: make(map[int]int),
		}

		// zero in unique severities to have constitent response
		for _, severity := range uniqueSeverities {
			clusterViewItem.HitsByTotalRisk[severity] = 0
		}

		// check if there are any hitting recommendations
		if hittingRecommendations, any := clusterRecommendationsMap[clusterViewItem.ClusterID]; any {
			clusterViewItem.LastCheckedAt = hittingRecommendations.CreatedAt.UTC().Format(time.RFC3339)

			// filter out acked and disabled rules
			enabledOnlyRecommendations := filterOutDisabledRules(
				hittingRecommendations.Recommendations, clusterViewItem.ClusterID,
				systemWideDisabledRules, disabledRulesPerCluster,
			)

			for _, ruleID := range enabledOnlyRecommendations {
				if ruleSeverity, found := recommendationSeverities[ruleID]; found {
					clusterViewItem.HitsByTotalRisk[ruleSeverity]++
					clusterViewItem.TotalHitCount++
				} else {
					// rule content is missing for this rule; mimicking behaviour of other apps such as OCM = skip rule
					log.Error().Msgf("rule content was not found for following rule ID. Skipping rule %v.", ruleID)
				}
			}
		}

		clusterListView = append(clusterListView, clusterViewItem)
	}

	return clusterListView, nil
}

// filterOutDisabledRules filters out system-wide disabled rules (rule acknowledgement) and rules which had been
// disabled on a single cluster basis.
func filterOutDisabledRules(
	hittingRecommendations []ctypes.RuleID,
	clusterID ctypes.ClusterName,
	systemWideDisabledRules map[ctypes.RuleID]bool,
	disabledRulesPerCluster map[ctypes.ClusterName][]ctypes.RuleID,
) (enabledOnlyRecommendations []ctypes.RuleID) {

	for _, hittingRuleID := range hittingRecommendations {
		// no need to continue, rule has been acked
		if systemWideDisabledRules[hittingRuleID] {
			continue
		}

		// try to find rule ID in list of disabled rules, if any
		ruleDisabled := false
		if disabledRulesList, any := disabledRulesPerCluster[clusterID]; any {
			for _, disabledRuleID := range disabledRulesList {
				if disabledRuleID == hittingRuleID {
					ruleDisabled = true
				}
			}
		}

		if !ruleDisabled {
			enabledOnlyRecommendations = append(enabledOnlyRecommendations, hittingRuleID)
		}
	}

	return
}

// Method getUserDisabledRulesPerCluster returns a map of cluster IDs with a list of disabled rules for each cluster
func (server *HTTPServer) getUserDisabledRulesPerCluster(userID types.UserID) (
	disabledRulesPerCluster map[ctypes.ClusterName][]ctypes.RuleID,
) {
	listOfDisabledRules, err := server.readListOfClusterDisabledRules(userID)
	if err != nil {
		log.Error().Err(err).Msg("error retrieving list of disabled rules")
		return
	}

	disabledRulesPerCluster = make(map[ctypes.ClusterName][]ctypes.RuleID)
	for i := range listOfDisabledRules {
		disabledRule := &listOfDisabledRules[i]

		compositeRuleID, err := generators.GenerateCompositeRuleID(ctypes.RuleFQDN(disabledRule.RuleID), disabledRule.ErrorKey)
		if err != nil {
			log.Error().Err(err).Msgf(compositeRuleIDError, disabledRule.RuleID, disabledRule.ErrorKey)
			continue
		}

		if ruleList, found := disabledRulesPerCluster[disabledRule.ClusterID]; found {
			disabledRulesPerCluster[disabledRule.ClusterID] = append(ruleList, compositeRuleID)
		} else {
			disabledRulesPerCluster[disabledRule.ClusterID] = []ctypes.RuleID{compositeRuleID}
		}
	}
	return
}

func generateRuleAckMap(acks []ctypes.SystemWideRuleDisable) (ruleAcksMap map[ctypes.RuleID]bool) {
	ruleAcksMap = make(map[ctypes.RuleID]bool)
	for i := range acks {
		ack := &acks[i]
		compositeRuleID, err := generators.GenerateCompositeRuleID(ctypes.RuleFQDN(ack.RuleID), ack.ErrorKey)
		if err == nil {
			ruleAcksMap[compositeRuleID] = true
		} else {
			log.Error().Err(err).Msgf(compositeRuleIDError, ack.RuleID, ack.ErrorKey)
		}
	}
	return
}

func generateImpactingRuleIDList(impactingRecommendations ctypes.RecommendationImpactedClusters) (ruleIDList []ctypes.RuleID) {
	ruleIDList = make([]ctypes.RuleID, len(impactingRecommendations))
	i := 0
	for ruleID := range impactingRecommendations {
		ruleIDList[i] = ruleID
		i++
	}
	return
}

func getRecommendationsFillUserData(
	impactingRecommendations ctypes.RecommendationImpactedClusters,
	impactingFlag types.ImpactingFlag,
	acks []ctypes.SystemWideRuleDisable,
) (
	recommendationList []types.RecommendationListView,
	err error,
) {
	var ruleIDList []ctypes.RuleID
	// put rule acks in a map so we only iterate over them once
	ruleAcksMap := generateRuleAckMap(acks)

	if impactingFlag == OnlyImpacting {
		// retrieve content only for impacting rules
		ruleIDList = generateImpactingRuleIDList(impactingRecommendations)
	} else {
		// retrieve content for all external rules and decide whether exclude impacting in loop
		ruleIDList, err = content.GetExternalRuleIDs()
		if err != nil {
			log.Error().Err(err).Msg("unable to retrieve external rule ids from content directory")
			return
		}
	}

	recommendationList = make([]types.RecommendationListView, 0)

	for _, ruleID := range ruleIDList {
		// rule is disabled if found in the ack map
		_, ruleDisabled := ruleAcksMap[ruleID]

		impactingClustersCnt, found := impactingRecommendations[ruleID]
		if found && impactingFlag == ExcludingImpacting {
			// rule is impacting, but requester doesn't want them
			continue
		}

		ruleContent, err := content.GetContentForRecommendation(ruleID)
		if err != nil {
			if err, ok := err.(*content.RuleContentDirectoryTimeoutError); ok {
				return recommendationList, err
			}
			// simply omit the rule as we can't display anything
			log.Error().Err(err).Msgf("unable to get content for rule with id %v", ruleID)
			continue
		}

		recommendationList = append(recommendationList, types.RecommendationListView{
			RuleID:              ruleID,
			Description:         ruleContent.Description,
			Generic:             ruleContent.Generic,
			PublishDate:         ruleContent.PublishDate,
			TotalRisk:           uint8(ruleContent.TotalRisk),
			Impact:              uint8(ruleContent.Impact),
			Likelihood:          uint8(ruleContent.Likelihood),
			Tags:                ruleContent.Tags,
			Disabled:            ruleDisabled,
			RiskOfChange:        uint8(ruleContent.RiskOfChange),
			ImpactedClustersCnt: impactingClustersCnt,
		})
	}

	return
}

// getImpactingRecommendations retrieves a list of recommendations from aggregator based on the list of clusters
func (server HTTPServer) getImpactingRecommendations(
	writer http.ResponseWriter,
	orgID ctypes.OrgID,
	userID ctypes.UserID,
	clusterList []ctypes.ClusterName,
) (ctypes.RecommendationImpactedClusters, error) {

	var aggregatorResponse struct {
		Recommendations ctypes.RecommendationImpactedClusters `json:"recommendations"`
		Status          string                                `json:"status"`
	}

	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.RecommendationsListEndpoint,
		orgID,
		userID,
	)

	jsonMarshalled, err := json.Marshal(clusterList)
	if err != nil {
		log.Error().Err(err).Msgf("problem unmarshalling cluster list")
		handleServerError(writer, err)
		return nil, err
	}

	// #nosec G107
	aggregatorResp, err := http.Post(aggregatorURL, JSONContentType, bytes.NewBuffer(jsonMarshalled))
	if err != nil {
		log.Error().Err(err).Msgf("problem getting response from aggregator")
		handleServerError(writer, err)
		return nil, err
	}

	responseBytes, err := ioutil.ReadAll(aggregatorResp.Body)
	if err != nil {
		log.Error().Err(err).Msgf("problem reading response body")
		handleServerError(writer, err)
		return nil, err
	}

	if aggregatorResp.StatusCode != http.StatusOK {
		err := responses.Send(aggregatorResp.StatusCode, writer, responseBytes)
		if err != nil {
			log.Error().Err(err).Msgf(problemSendingResponseError)
			handleServerError(writer, err)
		}
		return nil, err
	}

	err = json.Unmarshal(responseBytes, &aggregatorResponse)
	if err != nil {
		log.Error().Err(err).Msgf("problem unmarshalling JSON response")
		handleServerError(writer, err)
		return nil, err
	}

	return aggregatorResponse.Recommendations, nil
}

// getClustersAndRecommendations retrieves a list of recommendations from aggregator based on the list of clusters
func (server HTTPServer) getClustersAndRecommendations(
	writer http.ResponseWriter,
	orgID ctypes.OrgID,
	userID ctypes.UserID,
	clusterList []ctypes.ClusterName,
) (ctypes.ClusterRecommendationMap, error) {

	var aggregatorResponse struct {
		Clusters ctypes.ClusterRecommendationMap `json:"clusters"`
		Status   string                          `json:"status"`
	}

	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.ClustersRecommendationsListEndpoint,
		orgID,
		userID,
	)

	jsonMarshalled, err := json.Marshal(clusterList)
	if err != nil {
		log.Error().Err(err).Msg("problem unmarshalling cluster list")
		handleServerError(writer, err)
		return nil, err
	}

	// #nosec G107
	aggregatorResp, err := http.Post(aggregatorURL, JSONContentType, bytes.NewBuffer(jsonMarshalled))
	if err != nil {
		log.Error().Err(err).Msgf("problem getting response from aggregator")
		if _, ok := err.(*url.Error); ok {
			handleServerError(writer, &AggregatorServiceUnavailableError{})
		} else {
			handleServerError(writer, err)
		}
		return nil, err
	}

	responseBytes, err := ioutil.ReadAll(aggregatorResp.Body)
	if err != nil {
		log.Error().Err(err).Msgf("problem reading response body")
		handleServerError(writer, err)
		return nil, err
	}

	if aggregatorResp.StatusCode != http.StatusOK {
		err := responses.Send(aggregatorResp.StatusCode, writer, responseBytes)
		if err != nil {
			log.Error().Err(err).Msgf(problemSendingResponseError)
			handleServerError(writer, err)
		}
		return nil, err
	}

	err = json.Unmarshal(responseBytes, &aggregatorResponse)
	if err != nil {
		log.Error().Err(err).Msgf("problem unmarshalling JSON response")
		handleServerError(writer, err)
		return nil, err
	}

	return aggregatorResponse.Clusters, nil
}

// getContent retrieves all the static content tied with groups info
func (server HTTPServer) getContentWithGroups(writer http.ResponseWriter, request *http.Request) {
	// Generate an array of RuleContent
	allRules, err := content.GetAllContentV2()
	if err != nil {
		handleServerError(writer, err)
		return
	}

	var rules []types.RuleContentV2

	if err := server.checkInternalRulePermissions(request); err != nil {
		for _, rule := range allRules {
			if !content.IsRuleInternal(ctypes.RuleID(rule.Plugin.PythonModule)) {
				rules = append(rules, rule)
			}
		}
	} else {
		rules = allRules
	}

	// retrieve the latest groups configuration
	ruleGroups, err := server.getGroupsConfig()
	if err != nil {
		handleServerError(writer, err)
		return
	}
	// prepare data structure for building response
	responseContent := make(map[string]interface{})
	responseContent["status"] = OkMsg
	responseContent["groups"] = ruleGroups
	responseContent["content"] = rules

	// send response to client
	err = responses.SendOK(writer, responseContent)
	if err != nil {
		handleServerError(writer, err)
		return
	}
}

// getImpactedClustersFromAggregator sends GET to aggregator with or without content
// depending on the list of active clusters provided by the AMS client.
func getImpactedClustersFromAggregator(
	url string,
	activeClusters []ctypes.ClusterName,
) (resp *http.Response, err error) {
	if len(activeClusters) < 1 {
		// #nosec G107
		resp, err = http.Get(url)
		return
	}

	// generate JSON payload of the format "clusters": []clusters
	var jsonBody []byte
	jsonBody, err = json.Marshal(
		map[string][]ctypes.ClusterName{"clusters": activeClusters})
	if err != nil {
		log.Err(err).Msg("Couldn't encode list of active clusters to valid JSON, aborting")
		return
	}

	// GET method with list of active clusters in payload to avoid possible URL length problems
	var req *http.Request
	req, err = http.NewRequest(http.MethodGet, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return
	}

	req.Header.Set(contentTypeHeader, JSONContentType)
	client := &http.Client{}
	resp, err = client.Do(req)
	return
}

// getImpactedClusters retrieves a list of clusters affected by the given recommendation from aggregator
func (server HTTPServer) getImpactedClusters(
	writer http.ResponseWriter,
	orgID ctypes.OrgID,
	userID ctypes.UserID,
	selector ctypes.RuleSelector,
	activeClustersInfo []types.ClusterInfo,
) (
	[]ctypes.HittingClustersData,
	error,
) {
	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.RuleClusterDetailEndpoint,
		selector,
		orgID,
		userID,
	)

	activeClusters := types.GetClusterNames(activeClustersInfo)
	aggregatorResp, err := getImpactedClustersFromAggregator(aggregatorURL, activeClusters)
	// if http.Get fails for whatever reason
	if err != nil {
		handleServerError(writer, err)
		return []ctypes.HittingClustersData{}, err
	}

	if aggregatorResp.StatusCode == http.StatusOK {
		var response struct {
			Clusters []ctypes.HittingClustersData `json:"clusters"`
			Status   string                       `json:"status"`
		}

		err := json.NewDecoder(aggregatorResp.Body).Decode(&response)
		if err != nil {
			return []ctypes.HittingClustersData{}, err
		}

		return response.Clusters, nil
	}

	return []ctypes.HittingClustersData{}, nil
}

// getClustersDetailForRule retrieves all the clusters affected by the recommendation
// By default returns only those recommendations that currently hit at least one cluster, but it's
// possible to show all recommendations by passing a URL parameter `impacting`
func (server HTTPServer) getClustersDetailForRule(writer http.ResponseWriter, request *http.Request) {
	selector, successful := httputils.ReadRuleSelector(writer, request)
	if !successful {
		return
	}
	orgID, userID, err := server.readOrgIDAndUserIDFromToken(writer, request)
	if err != nil {
		log.Err(err).Msg(orgIDTokenError)
		return
	}

	if _, err = content.GetContentForRecommendation(ctypes.RuleID(selector)); err != nil {
		// The given rule selector does not exit
		handleServerError(writer, err)
		return
	}

	// Get list of clusters for given organization
	activeClustersInfo, err := server.readClusterInfoForOrgID(orgID)
	if err != nil {
		log.Error().Err(err).Int(orgIDTag, int(orgID)).Msg("Error retrieving cluster IDs from AMS API. Using empty list.")
	}

	// get the list of clusters affected by given rule from aggregator and
	impactedClusters, err := server.getImpactedClusters(writer, orgID, userID, selector, activeClustersInfo)
	if err != nil {
		log.Error().Err(err).Int(orgIDTag, int(orgID)).Str(userIDTag, string(userID)).Str(selectorStr, string(selector)).
			Msg("Couldn't get impacted clusters for given rule selector")
		handleServerError(writer, err)
		return
	}

	disabledClusters, err := server.getListOfDisabledClusters(orgID, userID, selector)
	if err != nil {
		log.Error().Err(err).Int(orgIDTag, int(orgID)).Str(userIDTag, string(userID)).Str(selectorStr, string(selector)).
			Msg("Couldn't retrieve disabled clusters for given rule selector")
		handleServerError(writer, err)
		return
	}

	err = server.processClustersDetailResponse(impactedClusters, disabledClusters, activeClustersInfo, writer)
	if err != nil {
		log.Error().Err(err).Int(orgIDTag, int(orgID)).Str(userIDTag, string(userID)).Str(selectorStr, string(selector)).
			Msg("Couldn't process response for clusters detail")
		handleServerError(writer, err)
		return
	}
}

// getListOfDisabledClusters reads list of disabled clusters from aggregator
func (server *HTTPServer) getListOfDisabledClusters(
	orgID types.OrgID, userID types.UserID, ruleSelector ctypes.RuleSelector,
) ([]ctypes.DisabledClusterInfo, error) {
	var response struct {
		Status           string                       `json:"status"`
		DisabledClusters []ctypes.DisabledClusterInfo `json:"clusters"`
	}

	// rule selector has already been validated
	splitRuleID := strings.Split(string(ruleSelector), "|")

	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.ListOfDisabledClusters,
		splitRuleID[0],
		splitRuleID[1],
		userID,
	)

	// #nosec G107
	resp, err := http.Get(aggregatorURL)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		err := fmt.Errorf("error reading disabled clusters from aggregator: %v", resp.StatusCode)
		return nil, err
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.DisabledClusters, nil
}

// processClustersDetailResponse processes responses from aggregator and AMS API and sends a response
func (server *HTTPServer) processClustersDetailResponse(
	impactedClusters []ctypes.HittingClustersData,
	disabledClusters []ctypes.DisabledClusterInfo,
	clusterInfo []types.ClusterInfo,
	writer http.ResponseWriter,
) error {

	data := types.ClustersDetailData{
		EnabledClusters:  make([]ctypes.HittingClustersData, 0),
		DisabledClusters: make([]ctypes.DisabledClusterInfo, 0),
	}
	// disabledMap is used to filter out the impacted clusters
	disabledMap := make(map[types.ClusterName]ctypes.DisabledClusterInfo)
	clusterNamesMap := types.ClusterInfoArrayToMap(clusterInfo)

	// filter out inactive clusters from disabled; fill in display names
	for _, disabledC := range disabledClusters {
		// omit clusters that weren't retrieved from AMS API
		if displayName, found := clusterNamesMap[disabledC.ClusterID]; found {
			disabledC.ClusterName = displayName
			disabledMap[disabledC.ClusterID] = disabledC
			data.DisabledClusters = append(data.DisabledClusters, disabledC)
		}
	}

	for _, impactedC := range impactedClusters {
		// omit disabled clusters
		if _, disabled := disabledMap[impactedC.Cluster]; disabled {
			continue
		}
		impactedC.Name = clusterNamesMap[impactedC.Cluster]
		data.EnabledClusters = append(data.EnabledClusters, impactedC)
	}

	response := types.ClustersDetailResponse{
		Status: OkMsg,
		Data:   data,
	}

	return responses.Send(http.StatusOK, writer, response)
}
