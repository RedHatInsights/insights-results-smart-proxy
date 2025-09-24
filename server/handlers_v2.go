// Copyright 2020, 2021, 2022, 2024 Red Hat, Inc
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
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-content-service/groups"
	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-operator-utils/responses"
	utypes "github.com/RedHatInsights/insights-operator-utils/types"
	ctypes "github.com/RedHatInsights/insights-results-types"

	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"

	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"
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
	// StatusProcessed is a message returned for already processed reports stored in Redis
	StatusProcessed = "processed"
	// RequestsForClusterNotFound is a message returned when no request IDs were found for a given clusterID
	RequestsForClusterNotFound = "Requests for cluster not found"
	// RequestIDNotFound is returned when the requested request ID was not found in the list of request IDs
	// for given cluster
	RequestIDNotFound = "Request ID not found for given org_id and cluster_id"
	// RedisNotInitializedErrorMessage is an error message written into log when Redis client is not initialized properly
	RedisNotInitializedErrorMessage = "redis is not initialized, request can not be finished correctly"
	//AMSApiNotInitializedErrorMessage is an error message written into log when AMS API client is not initialized properly
	AMSApiNotInitializedErrorMessage = "AMS API connection is not initialized"
)

func safeUint8(value int) (uint8, error) {
	if value < 0 {
		return 0, fmt.Errorf("cannot convert negative number to uint8: %d", value)
	}
	if value > math.MaxUint8 {
		return 0, fmt.Errorf("value %d is greater than the maximum uint8 value", value)
	}
	return uint8(value), nil
}

func safeUint32(value int) (uint32, error) {
	if value < 0 {
		return 0, fmt.Errorf("cannot convert negative number to uint32: %d", value)
	}
	if value > math.MaxUint32 {
		return 0, fmt.Errorf("value %d is greater than the maximum uint32 value", value)
	}
	return uint32(value), nil
}

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
	if internal := content.IsRuleInternal(ruleID); internal {
		err = server.checkInternalRulePermissions(request)
		if err != nil {
			log.Error().Err(err).Send()
			return
		}
	}

	return
}

// getRuleWithGroups retrieves static content for the given ruleID along with rule groups
func (server HTTPServer) getRuleWithGroups(
	request *http.Request,
	ruleID ctypes.RuleID,
) (
	ruleContent *types.RuleWithContent,
	ruleGroups []groups.Group,
	err error,
) {
	ruleContent, err = server.getContentCheckInternal(ruleID, request)
	if err != nil {
		log.Error().Interface(ruleIDStr, ruleID).Msg("error retrieving rule content for rule")
		return
	}

	// retrieve the latest groups configuration
	ruleGroups, err = server.getGroupsConfig()
	if err != nil {
		log.Error().Msg("error retrieving rule groups")
		return
	}

	return
}

// getRecommendationContent retrieves the static content for the given ruleID tied
// with groups info. rule ID is expected to be the composite rule ID (rule.module|ERROR_KEY)
func (server HTTPServer) getRecommendationContent(writer http.ResponseWriter, request *http.Request) {
	ruleID, err := readCompositeRuleID(request)
	if err != nil {
		log.Warn().Err(err).Msgf("error retrieving rule ID from request")
		handleServerError(writer, err)
		return
	}

	ruleContent, ruleGroups, err := server.getRuleWithGroups(request, ruleID)
	if err != nil {
		log.Warn().Err(err).Msgf("error retrieving rule content and groups for rule ID %v", ruleID)
		handleServerError(writer, err)
		return
	}

	totalRisk, err := safeUint8(ruleContent.TotalRisk)
	if err != nil {
		handleServerError(writer, err)
		return
	}
	impact, err := safeUint8(ruleContent.Impact)
	if err != nil {
		handleServerError(writer, err)
		return
	}
	likelihood, err := safeUint8(ruleContent.Likelihood)
	if err != nil {
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
		TotalRisk:    totalRisk,
		Impact:       impact,
		Likelihood:   likelihood,
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
	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		log.Err(err).Msg(orgIDTokenError)
		handleServerError(writer, err)
		return
	}

	ruleID, err := readCompositeRuleID(request)
	if err != nil {
		log.Warn().Err(err).Msg("error retrieving rule ID from request")
		handleServerError(writer, err)
		return
	}

	ruleContent, ruleGroups, err := server.getRuleWithGroups(request, ruleID)
	if err != nil {
		log.Warn().Err(err).Interface(ruleIDStr, ruleID).Msg("error retrieving rule content and groups for rule")
		handleServerError(writer, err)
		return
	}

	rating, err := server.getRatingForRecommendation(orgID, ruleID)
	if err != nil {
		switch err.(type) {
		case *utypes.ItemNotFoundError:
			break
		case *url.Error:
			log.Error().Err(err).Msg("aggregator is not responding")
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
	)

	totalRisk, err := safeUint8(ruleContent.TotalRisk)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	resolutionRisk, err := safeUint8(ruleContent.ResolutionRisk)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	impact, err := safeUint8(ruleContent.Impact)
	if err != nil {
		handleServerError(writer, err)
		return
	}
	likelihood, err := safeUint8(ruleContent.Likelihood)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// fill in user rating and other DB stuff from aggregator
	contentResponse := types.RecommendationContentUserData{
		// RuleID in rule.module|ERROR_KEY format
		RuleSelector:   ctypes.RuleSelector(ruleID),
		Description:    ruleContent.Description,
		Generic:        ruleContent.Generic,
		Reason:         ruleContent.Reason,
		Resolution:     ruleContent.Resolution,
		MoreInfo:       ruleContent.MoreInfo,
		TotalRisk:      totalRisk,
		ResolutionRisk: resolutionRisk,
		Impact:         impact,
		Likelihood:     likelihood,
		PublishDate:    ruleContent.PublishDate,
		Rating:         rating.Rating,
		AckedCount:     0,
		Tags:           ruleContent.Tags,
		Disabled:       ackFound,
	}

	// prepare data structure for building response
	responseContent := make(map[string]interface{})
	responseContent["status"] = OkMsg
	responseContent["groups"] = ruleGroups
	responseContent["content"] = contentResponse

	// send response to client
	err = responses.SendOK(writer, responseContent)
	if err != nil {
		log.Error().Err(err).Msg(problemSendingResponseError)
		handleServerError(writer, err)
		return
	}
}

// getRecommendations retrieves all recommendations with a count of impacted clusters
// By default returns only those recommendations that currently hit at least one cluster,
// but it's possible to show all recommendations by passing a URL parameter `impacting`
func (server HTTPServer) getRecommendations(writer http.ResponseWriter, request *http.Request) {
	var recommendationList []types.RecommendationListView
	tStart := time.Now()

	userID, orgID, impactingFlag, err := server.readParamsGetRecommendations(writer, request)
	if err != nil {
		// everything handled
		log.Error().Err(err).Msg("problem reading necessary params from request")
		return
	}

	activeClustersInfo, err := server.readClusterInfoForOrgID(orgID)
	if err != nil {
		log.Warn().Err(err).Int(orgIDTag, int(orgID)).Msg(clusterListError)
		handleServerError(writer, err)
		return
	}
	clusterIDList := types.GetClusterNames(activeClustersInfo)

	tStartImpacting := time.Now()
	impactingRecommendations, err := server.getImpactingRecommendations(
		writer, orgID, userID, clusterIDList,
	)
	if err != nil {
		// log cluster list in case of error even though message might be too large for Kibana/zerolog
		log.Error().
			Err(err).
			Int(orgIDTag, int(orgID)).
			Msgf("problem getting impacting recommendations from aggregator for cluster list (# of clusters: %v)", len(clusterIDList))

		return
	}
	log.Debug().Uint32(orgIDTag, uint32(orgID)).Msgf(
		"getRecommendations get impacting recommendations from aggregator took %s", time.Since(tStartImpacting),
	)

	// get a map of acknowledged rules
	ackedRulesMap, err := server.getRuleAcksMap(orgID)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// retrieve user disabled rules for given list of active clusters
	disabledClustersForRules := server.getRuleDisabledClusters(writer, orgID, clusterIDList)

	recommendationList, err = getFilteredRecommendationsList(
		activeClustersInfo, impactingRecommendations, impactingFlag, ackedRulesMap, disabledClustersForRules,
	)

	if err != nil {
		log.Error().Err(err).Msg("problem getting recommendation content")
		handleServerError(writer, err)
		return
	}
	log.Debug().
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
		log.Error().Err(err).Msg(problemSendingResponseError)
		handleServerError(writer, err)
		return
	}
}

func (server HTTPServer) getRuleAcksMap(orgID types.OrgID) (
	ackedRulesMap map[ctypes.RuleID]bool, err error,
) {
	ackedRulesMap = make(map[ctypes.RuleID]bool)

	// retrieve rule acknowledgements (disable/enable for all clusters)
	ackedRules, err := server.readListOfAckedRules(orgID)
	if err != nil {
		log.Error().Err(err).Msg(ackedRulesError)
		return
	}
	// put rule acks in a map so we only iterate over them once
	ackedRulesMap = generateRuleAckMap(ackedRules)

	return
}

func (server HTTPServer) getRuleDisabledClusters(
	writer http.ResponseWriter,
	orgID types.OrgID,
	clusterList []ctypes.ClusterName,
) (
	ruleDisabledClusters map[types.RuleID][]types.ClusterName,
) {
	ruleDisabledClusters = make(map[types.RuleID][]types.ClusterName)

	listOfDisabledRules, err := server.readListOfDisabledRulesForClusters(writer, orgID, clusterList)
	if err != nil {
		log.Error().Err(err).Msg("error reading disabled rules from aggregator")
		// server error has been handled already
		return
	}

	for _, disabledRule := range listOfDisabledRules {
		compositeRuleID, err := generateCompositeRuleIDFromDisabled(disabledRule)

		if err != nil {
			log.Error().Err(err).Msg("error generating composite rule ID")
			continue
		}

		ruleDisabledClusters[compositeRuleID] = append(ruleDisabledClusters[compositeRuleID], disabledRule.ClusterID)
	}

	return
}

// getClustersView retrieves all clusters for given organization, retrieves the impacting rules for each cluster
// from aggregator and returns a list of clusters, total number of hitting rules and a count of impacting rules
// by severity = total risk = critical, high, moderate, low
func (server HTTPServer) getClustersView(writer http.ResponseWriter, request *http.Request) {
	tStart := time.Now()

	orgID, userID, err := server.GetCurrentOrgIDUserIDFromToken(request)
	if err != nil {
		log.Err(err).Msg(orgIDTokenError)
		handleServerError(writer, err)
		return
	}

	clusterList, clusterRuleHits, ackedRulesMap, disabledRules := server.getClusterListAndUserData(
		writer,
		orgID,
		userID,
	)

	clusterViewResponse, err := matchClusterInfoAndUserData(
		clusterList, clusterRuleHits, ackedRulesMap, disabledRules,
	)
	if err != nil {
		log.Error().Uint32(orgIDTag, uint32(orgID)).Err(err).Msg("getClustersView error generating cluster list response")
		handleServerError(writer, err)
	}
	log.Debug().Uint32(orgIDTag, uint32(orgID)).Msgf("getClustersView final number %v", len(clusterViewResponse))

	resp := make(map[string]interface{})
	metaCount := map[string]int{
		"count": len(clusterViewResponse),
	}
	resp["status"] = OkMsg
	resp["meta"] = metaCount
	resp["data"] = clusterViewResponse

	log.Debug().Uint32(orgIDTag, uint32(orgID)).Msgf("getClustersView took %s", time.Since(tStart))

	err = responses.SendOK(writer, resp)
	if err != nil {
		log.Error().Err(err).Msg(problemSendingResponseError)
		handleServerError(writer, err)
		return
	}
}

// getSingleClusterInfo retrieves information about given cluster from AMS API, such as the user defined display name
func (server HTTPServer) getSingleClusterInfo(writer http.ResponseWriter, request *http.Request) {
	if server.amsClient == nil {
		log.Error().Msg(AMSApiNotInitializedErrorMessage)
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
		log.Warn().Err(err).Msg("problem retrieving cluster info from AMS API")
		handleServerError(writer, err)
		return
	}

	// retrieval failed, but error is nil
	if clusterInfo.ID == "" {
		err := &utypes.ItemNotFoundError{ItemID: clusterID}
		log.Warn().Err(err).Msg("unexpected problem retrieving cluster info from AMS API")
		handleServerError(writer, err)
		return
	}

	if err = responses.SendOK(writer, responses.BuildOkResponseWithData("cluster", clusterInfo)); err != nil {
		log.Error().Err(err).Msg(problemSendingResponseError)
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

	rulesManagedInfo, err := content.GetExternalRulesManagedInfo()
	if err != nil {
		return clusterListView, err
	}

	// iterates over clusters and their hitting recommendations, accesses map to the get rule severity
	for i := range clusterInfoList {
		clusterViewItem := types.ClusterListView{
			ClusterID:       clusterInfoList[i].ID,
			ClusterName:     clusterInfoList[i].DisplayName,
			Managed:         clusterInfoList[i].Managed,
			HitsByTotalRisk: make(map[int]int),
		}

		// zero in unique severities to have constitent response
		for _, severity := range uniqueSeverities {
			clusterViewItem.HitsByTotalRisk[severity] = 0
		}

		// check if there are any hitting recommendations
		if hittingRecommendations, exist := clusterRecommendationsMap[clusterViewItem.ClusterID]; exist {
			clusterViewItem.LastCheckedAt = types.Timestamp(
				hittingRecommendations.CreatedAt.UTC().Format(time.RFC3339),
			)
			clusterViewItem.Version = hittingRecommendations.Meta.Version

			// filter out acked and disabled rules
			enabledOnlyRecommendations := filterOutDisabledRules(
				hittingRecommendations.Recommendations, clusterViewItem.ClusterID,
				systemWideDisabledRules, disabledRulesPerCluster,
			)

			for _, ruleID := range enabledOnlyRecommendations {
				if clusterViewItem.Managed && !rulesManagedInfo[ruleID] {
					// cluster is managed, therefore must show only managed rules
					continue
				}

				if ruleSeverity, found := recommendationSeverities[ruleID]; found {
					clusterViewItem.HitsByTotalRisk[ruleSeverity]++
					clusterViewItem.TotalHitCount++
				} else {
					// rule content is missing for this rule; mimicking behaviour of other apps such as OCM = skip rule
					log.Error().Interface(ruleIDStr, ruleID).Msg("rule content was not found for following rule ID. Skipping it")
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
) (
	enabledOnlyRecommendations []ctypes.RuleID,
) {
	for _, hittingRuleID := range hittingRecommendations {
		// no need to continue, rule has been acked
		if systemWideDisabledRules[hittingRuleID] {
			continue
		}

		// try to find rule ID in list of disabled rules, if any
		ruleDisabled := false
		if disabledRulesList, exists := disabledRulesPerCluster[clusterID]; exists {
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
func (server *HTTPServer) getUserDisabledRulesPerCluster(orgID types.OrgID) (
	disabledRulesPerCluster map[ctypes.ClusterName][]ctypes.RuleID,
) {
	listOfDisabledRules, err := server.readListOfClusterDisabledRules(orgID)
	if err != nil {
		log.Error().Err(err).Msg("error retrieving list of disabled rules")
		return
	}

	disabledRulesPerCluster = make(map[ctypes.ClusterName][]ctypes.RuleID)
	for i := range listOfDisabledRules {
		disabledRule := &listOfDisabledRules[i]

		compositeRuleID, err := generateCompositeRuleIDFromDisabled(*disabledRule)
		if err != nil {
			log.Error().Err(err).Interface(ruleIDStr, disabledRule.RuleID).
				Interface(errorKeyStr, disabledRule.ErrorKey).Msg(compositeRuleIDError)
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

func generateImpactingRuleIDList(impactingRecommendations ctypes.RecommendationImpactedClusters) (ruleIDList []ctypes.RuleID) {
	ruleIDList = make([]ctypes.RuleID, len(impactingRecommendations))
	i := 0
	for ruleID := range impactingRecommendations {
		ruleIDList[i] = ruleID
		i++
	}
	return
}

func excludeDisabledClusters(
	impactingClusters []types.ClusterName,
	disabledClusters []types.ClusterName,
) (filteredClusters []types.ClusterName) {
	for _, impactingID := range impactingClusters {
		disabled := false

		for _, disabledID := range disabledClusters {
			if impactingID == disabledID {
				disabled = true
				break
			}
		}

		if !disabled {
			filteredClusters = append(filteredClusters, impactingID)
		}
	}
	return
}

//nolint:gocyclo
func getFilteredRecommendationsList(
	activeClustersInfo []types.ClusterInfo,
	impactingRecommendations ctypes.RecommendationImpactedClusters,
	impactingFlag types.ImpactingFlag,
	ruleAcksMap map[types.RuleID]bool,
	disabledClustersForRules map[types.RuleID][]types.ClusterName,
) (
	recommendationList []types.RecommendationListView,
	err error,
) {
	clusterInfoMap := types.ClusterInfoArrayToMap(activeClustersInfo)
	recommendationList = make([]types.RecommendationListView, 0)

	var ruleIDList []ctypes.RuleID
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

	// iterate over rules and count impacted clusters, exluding user disabled ones
	for _, ruleID := range ruleIDList {
		var impactedClustersCnt uint32
		var ruleContent *types.RuleWithContent
		// rule has system-wide disabled status if found in the ack map,
		// but the user must be able to see the number of impacted clusters in the UI, so we need to go on
		_, ruleDisabled := ruleAcksMap[ruleID]

		// get list of impacting clusters
		impactingClustersList, found := impactingRecommendations[ruleID]
		if found && impactingFlag == ExcludingImpacting {
			// rule is impacting, but requester doesn't want them
			continue
		}

		// remove any disabled clusters from the total count, if they're impacting
		if disabledClusters, exist := disabledClustersForRules[ruleID]; exist {
			impactingClustersList = excludeDisabledClusters(impactingClustersList, disabledClusters)
		}

		ruleContent, err = content.GetContentForRecommendation(ruleID)
		if err != nil {
			if err, ok := err.(*content.RuleContentDirectoryTimeoutError); ok {
				return recommendationList, err
			}
			// missing rule content, simply omit the rule as we can't display anything
			log.Error().Err(err).Interface(ruleIDStr, ruleID).Msg(ruleContentError)
			continue
		}

		if !ruleContent.OSDCustomer {
			// rule doesn't have osd_customer tag, so it doesn't apply to managed clusters
			for _, clusterID := range impactingClustersList {
				// exclude non-managed clusters from the count
				if !clusterInfoMap[clusterID].Managed {
					impactedClustersCnt++
				}
			}
		} else {
			// rule has osd_customer tag and can be shown for all clusters
			impactedClustersCnt, err = safeUint32(len(impactingClustersList))
			if err != nil {
				return
			}
		}

		recommendationListView, err := parseRecommendationListView(
			ruleID, ruleContent, ruleDisabled, impactedClustersCnt)
		if err != nil {
			return recommendationList, err
		}
		recommendationList = append(recommendationList, recommendationListView)
	}

	return
}

func parseRecommendationListView(
	ruleID types.RuleID, ruleContent *types.RuleWithContent, ruleDisabled bool,
	impactedClustersCnt uint32) (
	types.RecommendationListView, error) {
	recommendationListView := types.RecommendationListView{}
	totalRisk, err := safeUint8(ruleContent.TotalRisk)
	if err != nil {
		return recommendationListView, err
	}

	resolutionRisk, err := safeUint8(ruleContent.ResolutionRisk)
	if err != nil {
		return recommendationListView, err
	}

	impact, err := safeUint8(ruleContent.Impact)
	if err != nil {
		return recommendationListView, err
	}
	likelihood, err := safeUint8(ruleContent.Likelihood)
	if err != nil {
		return recommendationListView, err
	}

	recommendationListView = types.RecommendationListView{
		RuleID:              ruleID,
		Description:         ruleContent.Description,
		Generic:             ruleContent.Generic,
		PublishDate:         ruleContent.PublishDate,
		TotalRisk:           totalRisk,
		ResolutionRisk:      resolutionRisk,
		Impact:              impact,
		Likelihood:          likelihood,
		Tags:                ruleContent.Tags,
		Disabled:            ruleDisabled,
		ImpactedClustersCnt: impactedClustersCnt,
	}
	return recommendationListView, nil
}

// getImpactingRecommendations retrieves a list of recommendations from aggregator based on the list of clusters
func (server HTTPServer) getImpactingRecommendations(
	writer http.ResponseWriter,
	orgID ctypes.OrgID,
	userID ctypes.UserID,
	clusterList []ctypes.ClusterName,
) (
	ctypes.RecommendationImpactedClusters, error,
) {
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
		log.Error().Err(err).Msg("getImpactingRecommendations problem unmarshalling cluster list")
		handleServerError(writer, err)
		return nil, err
	}

	// #nosec G107
	// nolint:bodyclose // TODO: remove once the bodyclose library fixes this bug
	aggregatorResp, err := http.Post(aggregatorURL, JSONContentType, bytes.NewBuffer(jsonMarshalled))
	if err != nil {
		log.Error().Err(err).Msg("getImpactingRecommendations problem getting response from aggregator")
		handleServerError(writer, err)
		return nil, err
	}

	defer services.CloseResponseBody(aggregatorResp)

	responseBytes, err := io.ReadAll(aggregatorResp.Body)
	if err != nil {
		log.Error().Err(err).Msg("getImpactingRecommendations problem reading response body")
		handleServerError(writer, err)
		return nil, err
	}

	if aggregatorResp.StatusCode != http.StatusOK {
		err := responses.Send(aggregatorResp.StatusCode, writer, responseBytes)
		if err != nil {
			log.Error().Err(err).Msg(problemSendingResponseError)
			handleServerError(writer, err)
		}
		return nil, err
	}

	err = json.Unmarshal(responseBytes, &aggregatorResponse)
	if err != nil {
		log.Error().Err(err).Msg("getImpactingRecommendations problem unmarshalling JSON response")
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
		log.Error().Err(err).Msg("getClustersAndRecommendations problem unmarshalling cluster list")
		handleServerError(writer, err)
		return nil, err
	}

	// #nosec G107
	// nolint:bodyclose // TODO: remove once the bodyclose library fixes this bug
	aggregatorResp, err := http.Post(aggregatorURL, JSONContentType, bytes.NewBuffer(jsonMarshalled))
	if err != nil {
		log.Error().Err(err).Msg("getClustersAndRecommendations problem getting response from aggregator")
		if _, ok := err.(*url.Error); ok {
			handleServerError(writer, &AggregatorServiceUnavailableError{})
		} else {
			handleServerError(writer, err)
		}
		return nil, err
	}
	defer services.CloseResponseBody(aggregatorResp)

	responseBytes, err := io.ReadAll(aggregatorResp.Body)
	if err != nil {
		log.Error().Err(err).Msg("getClustersAndRecommendations problem reading response body")
		handleServerError(writer, err)
		return nil, err
	}

	if aggregatorResp.StatusCode != http.StatusOK {
		err := responses.Send(aggregatorResp.StatusCode, writer, responseBytes)
		if err != nil {
			log.Error().Err(err).Msg(problemSendingResponseError)
			handleServerError(writer, err)
		}
		return nil, err
	}

	err = json.Unmarshal(responseBytes, &aggregatorResponse)
	if err != nil {
		log.Error().Err(err).Msg("getClustersAndRecommendations problem unmarshalling JSON response")
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
	useAggregatorFallback bool,
) (
	[]ctypes.HittingClustersData,
	error,
) {
	activeClusters := types.GetClusterNames(activeClustersInfo)
	if len(activeClusters) == 0 && !useAggregatorFallback {
		// empty list from AMS is valid
		return []ctypes.HittingClustersData{}, nil
	}

	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.RuleClusterDetailEndpoint,
		selector,
		orgID,
		userID,
	)

	// nolint:bodyclose // TODO: remove once the bodyclose library fixes this bug
	aggregatorResp, err := getImpactedClustersFromAggregator(aggregatorURL, activeClusters)
	// if http.Get fails for whatever reason
	if err != nil {
		handleServerError(writer, err)
		return []ctypes.HittingClustersData{}, err
	}

	defer services.CloseResponseBody(aggregatorResp)

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
	var useAggregatorFallback bool

	selector, successful := httputils.ReadRuleSelector(writer, request)
	if !successful {
		return
	}
	orgID, userID, err := server.GetCurrentOrgIDUserIDFromToken(request)
	if err != nil {
		log.Err(err).Msg(orgIDTokenError)
		handleServerError(writer, err)
		return
	}

	recommendation, err := content.GetContentForRecommendation(ctypes.RuleID(selector))
	if err != nil {
		// The given rule selector does not exit
		handleServerError(writer, err)
		return
	}

	// Get list of clusters for given organization
	activeClustersInfo, err := server.readClusterInfoForOrgID(orgID)
	if err != nil {
		log.Error().Err(err).Int(orgIDTag, int(orgID)).Msg("Error retrieving cluster IDs from AMS API. Will retrieve cluster list from aggregator.")
		useAggregatorFallback = true
	}

	// if the recommendation is not intended to be used with OpenShift Dedicated ("managed") clusters, we must exclude them
	if !recommendation.OSDCustomer {
		filteredClusters := make([]types.ClusterInfo, 0)

		for _, cluster := range activeClustersInfo {
			// skipping managed clusters, because recommendation isn't managed
			if !cluster.Managed {
				filteredClusters = append(filteredClusters, cluster)
			}
		}

		activeClustersInfo = filteredClusters
	}

	// get the list of clusters affected by given rule from aggregator and
	impactedClusters, err := server.getImpactedClusters(writer, orgID, userID, selector, activeClustersInfo, useAggregatorFallback)
	if err != nil {
		log.Error().Err(err).Int(orgIDTag, int(orgID)).Str(userIDTag, string(userID)).Str(selectorStr, string(selector)).
			Msg("Couldn't get impacted clusters for given rule selector")
		handleServerError(writer, err)
		return
	}

	disabledClusters, acknowledge, ackFound, err := server.getListOfDisabledClustersAndAck(orgID, selector)
	if err != nil {
		log.Error().Err(err).Int(orgIDTag, int(orgID)).Str(userIDTag, string(userID)).Str(selectorStr, string(selector)).
			Msg("Couldn't retrieve disabled clusters or ack for given rule selector")
		handleServerError(writer, err)
		return
	}

	err = server.processClustersDetailResponse(impactedClusters, disabledClusters, activeClustersInfo, acknowledge, ackFound, writer)
	if err != nil {
		log.Error().Err(err).Int(orgIDTag, int(orgID)).Str(userIDTag, string(userID)).Str(selectorStr, string(selector)).
			Msg("Couldn't process response for clusters detail")
		handleServerError(writer, err)
		return
	}
}

// getListOfDisabledClusters reads list of disabled clusters from aggregator
func (server *HTTPServer) getListOfDisabledClusters(
	orgID types.OrgID, ruleSelector ctypes.RuleSelector,
) ([]ctypes.DisabledClusterInfo, error) {
	var response struct {
		Status           string                       `json:"status"`
		DisabledClusters []ctypes.DisabledClusterInfo `json:"clusters"`
	}

	ruleID, errorKey, err := types.RuleIDWithErrorKeyFromCompositeRuleID(ctypes.RuleID(ruleSelector))
	if err != nil {
		return nil, err
	}

	// rules disabled using v1 enable/disable endpoints include '.report' in the module
	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.ListOfDisabledClusters,
		ruleID+dotReport,
		errorKey,
		orgID,
	)

	// #nosec G107
	// nolint:bodyclose // TODO: remove once the bodyclose library fixes this bug
	resp, err := http.Get(aggregatorURL)
	if err != nil {
		return nil, err
	}

	defer services.CloseResponseBody(resp)

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

// getListOfDisabledClustersAndAck reads list of disabled clusters from aggregator and gets
// information about rule ack
func (server *HTTPServer) getListOfDisabledClustersAndAck(
	orgID types.OrgID, ruleSelector ctypes.RuleSelector,
) (
	disabledClusters []ctypes.DisabledClusterInfo,
	acknowledge ctypes.Acknowledgement,
	ackFound bool,
	err error,
) {
	disabledClusters, err = server.getListOfDisabledClusters(orgID, ruleSelector)
	if err != nil {
		log.Error().Err(err).Int(orgIDTag, int(orgID)).Str(selectorStr, string(ruleSelector)).
			Msg("Couldn't retrieve disabled clusters for given rule selector")
		return
	}

	ruleID, errorKey, err := types.RuleIDWithErrorKeyFromCompositeRuleID(ctypes.RuleID(ruleSelector))
	if err != nil {
		return
	}

	acknowledge, ackFound, err = server.readRuleDisableStatus(
		ctypes.Component(ruleID),
		errorKey,
		orgID,
	)
	if err != nil {
		log.Error().Err(err).Int(orgIDTag, int(orgID)).Str(selectorStr, string(ruleSelector)).
			Msg("Couldn't retrieve rule acknowledge status for given rule selector")
		return
	}

	return
}

// processClustersDetailResponse processes responses from aggregator and AMS API and sends a response
func (server *HTTPServer) processClustersDetailResponse(
	impactedClusters []ctypes.HittingClustersData,
	disabledClusters []ctypes.DisabledClusterInfo,
	clusterInfo []types.ClusterInfo,
	acknowledge ctypes.Acknowledgement,
	ruleAcked bool,
	writer http.ResponseWriter,
) error {
	data := types.ClustersDetailData{
		EnabledClusters:  make([]ctypes.HittingClustersData, 0),
		DisabledClusters: make([]ctypes.DisabledClusterInfo, 0),
	}
	// disabledMap is used to filter out the impacted clusters
	disabledMap := make(map[types.ClusterName]ctypes.DisabledClusterInfo)
	clusterInfoMap := types.ClusterInfoArrayToMap(clusterInfo)

	// filter out inactive clusters from disabled; fill in display names
	for _, disabledC := range disabledClusters {
		// omit clusters that weren't retrieved from AMS API
		if cluster, found := clusterInfoMap[disabledC.ClusterID]; found {
			disabledC.ClusterName = cluster.DisplayName
			disabledMap[disabledC.ClusterID] = disabledC
			data.DisabledClusters = append(data.DisabledClusters, disabledC)
		}
	}

	for _, impactedC := range impactedClusters {
		// omit disabled clusters
		if _, disabled := disabledMap[impactedC.Cluster]; disabled {
			continue
		}
		impactedC.Name = clusterInfoMap[impactedC.Cluster].DisplayName

		if ruleAcked {
			disabledAt, err := time.Parse(time.RFC3339, acknowledge.CreatedAt)
			if err != nil {
				log.Error().Err(err).Str("createdAt", acknowledge.CreatedAt).Msg("error parsing time as RFC3339")
				disabledAt = time.Time{}
			}
			disabledCluster := ctypes.DisabledClusterInfo{
				ClusterID:     impactedC.Cluster,
				ClusterName:   impactedC.Name,
				DisabledAt:    disabledAt,
				Justification: acknowledge.Justification,
			}
			data.DisabledClusters = append(data.DisabledClusters, disabledCluster)
		} else {
			data.EnabledClusters = append(data.EnabledClusters, impactedC)
		}
	}

	response := types.ClustersDetailResponse{
		Status: OkMsg,
		Data:   data,
	}

	return responses.Send(http.StatusOK, writer, response)
}

// getRequestStatusForCluster method implements endpoint that should return a status
// for given request ID.
func (server *HTTPServer) getRequestStatusForCluster(writer http.ResponseWriter, request *http.Request) {
	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		log.Error().Msg(authTokenFormatError)
		handleServerError(writer, err)
		return
	}

	clusterID, successful := httputils.ReadClusterName(writer, request)
	if !successful {
		// error handled by function
		return
	}

	requestID, err := readRequestID(writer, request)
	if err != nil {
		// error handled by function
		return
	}

	// make sure we don't access server.redis when it's nil
	if !server.checkRedisClientReadiness(writer) {
		// error has been handled already
		return
	}

	// get request ID list from Redis using SCAN command
	requestIDsForCluster, err := server.redis.GetRequestIDsForClusterID(orgID, clusterID)
	if err != nil {
		handleServerError(writer, err)
		return
	}
	if len(requestIDsForCluster) == 0 {
		err := responses.SendNotFound(writer, RequestsForClusterNotFound)
		if err != nil {
			log.Error().Err(err).Msg(responseDataError)
		}
		return
	}

	// try to find the required request ID in list of requests IDs from Redis
	var found bool
	for _, storedRequestID := range requestIDsForCluster {
		if storedRequestID == requestID {
			found = true
			break
		}
	}

	if !found {
		err := responses.SendNotFound(writer, RequestIDNotFound)
		if err != nil {
			log.Error().Err(err).Msg(responseDataError)
		}
		return
	}

	// prepare data structure
	responseData := map[string]interface{}{}
	responseData["cluster"] = string(clusterID)
	responseData["requestID"] = requestID
	responseData["status"] = StatusProcessed

	// send response to client
	err = responses.SendOK(writer, responseData)
	if err != nil {
		handleServerError(writer, err)
		return
	}
}

// getRequestsForCluster method implements endpoint that should return a list of
// all request IDs and their details for given cluster
func (server *HTTPServer) getRequestsForCluster(writer http.ResponseWriter, request *http.Request) {
	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		log.Error().Msg(authTokenFormatError)
		handleServerError(writer, err)
		return
	}

	clusterID, successful := httputils.ReadClusterName(writer, request)
	if !successful {
		// error handled by function
		return
	}

	// make sure we don't access server.redis when it's nil
	if !server.checkRedisClientReadiness(writer) {
		// error has been handled already
		return
	}

	// get request ID list from Redis using SCAN command
	requestIDsForCluster, err := server.redis.GetRequestIDsForClusterID(orgID, clusterID)
	if err != nil {
		handleServerError(writer, err)
		return
	}
	if len(requestIDsForCluster) == 0 {
		err := responses.SendNotFound(writer, RequestsForClusterNotFound)
		if err != nil {
			log.Error().Err(err).Msg(responseDataError)
		}
		return
	}

	// get data for each request ID. Omit missing keys in case the data expired in the meantime
	requestIDsData, err := server.redis.GetTimestampsForRequestIDs(orgID, clusterID, requestIDsForCluster, true)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// prepare data structure
	responseData := map[string]interface{}{}
	responseData["cluster"] = string(clusterID)
	responseData["requests"] = requestIDsData
	responseData["status"] = OkMsg

	// send response to client
	err = responses.SendOK(writer, responseData)
	if err != nil {
		handleServerError(writer, err)
		return
	}
}

// getRequestsForCluster method implements endpoint that should return a list of
// request IDs and their details for given cluster and given list of request IDs provided in request body
func (server *HTTPServer) getRequestsForClusterPostVariant(writer http.ResponseWriter, request *http.Request) {
	const logMsg = "getRequestsForClusterPostVariant"

	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		log.Error().Msg(authTokenFormatError)
		handleServerError(writer, err)
		return
	}

	log.Debug().Uint32(orgIDTag, uint32(orgID)).Msg(logMsg)

	clusterID, successful := httputils.ReadClusterName(writer, request)
	if !successful {
		// error handled by function
		return
	}

	log.Debug().Str("selected cluster", string(clusterID)).Msg(logMsg)

	// get request ID list from request body
	requestIDsForCluster, err := readRequestIDList(writer, request)
	if err != nil {
		// error handled by function
		return
	}

	log.Debug().
		Uint32(orgIDTag, uint32(orgID)).
		Str("selected cluster", string(clusterID)).
		Int("IDS count", len(requestIDsForCluster)).
		Msg("requestIDs")

	// make sure we don't access server.redis when it's nil
	if !server.checkRedisClientReadiness(writer) {
		// error has been handled already
		return
	}

	// get data for each request ID. Don't omit missing keys, because requester wants to know which are valid
	requestIDsData, err := server.redis.GetTimestampsForRequestIDs(orgID, clusterID, requestIDsForCluster, false)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// prepare data structure
	responseData := map[string]interface{}{}
	responseData["cluster"] = string(clusterID)
	responseData["requests"] = requestIDsData
	responseData["status"] = OkMsg

	// send response to client
	err = responses.SendOK(writer, responseData)
	if err != nil {
		handleServerError(writer, err)
		return
	}
}

// getReportForRequest method implements endpoint that should return
// simplified result for given request ID
func (server *HTTPServer) getReportForRequest(writer http.ResponseWriter, request *http.Request) {
	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		log.Error().Msg(authTokenFormatError)
		handleServerError(writer, err)
		return
	}

	clusterID, successful := httputils.ReadClusterName(writer, request)
	if !successful {
		// error handled by function
		return
	}

	requestID, err := readRequestID(writer, request)
	if err != nil {
		// error handled by function
		return
	}

	// make sure we don't access server.redis when it's nil
	if !server.checkRedisClientReadiness(writer) {
		// error has been handled already
		return
	}

	// get rule hits from Redis
	ruleHits, err := server.redis.GetRuleHitsForRequest(orgID, clusterID, requestID)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// get a map of acknowledged rules
	ackedRulesMap, err := server.getRuleAcksMap(orgID)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// retrieve user disabled rules for given cluster
	disabledRulesForCluster, err := server.getDisabledRulesForClusterMap(writer, orgID, clusterID)
	if err != nil {
		log.Error().Err(err).Msg("problem getting user disabled rules for cluster")
		// server error has been handled already
		return
	}

	filteredRuleHits := filterRulesGetContent(ruleHits, ackedRulesMap, disabledRulesForCluster)

	// prepare response
	responseData := map[string]interface{}{}
	responseData["cluster"] = string(clusterID)
	responseData["requestID"] = requestID
	responseData["status"] = StatusProcessed
	responseData["report"] = filteredRuleHits

	// send response to client
	err = responses.SendOK(writer, responseData)
	if err != nil {
		handleServerError(writer, err)
		return
	}
}

func filterRulesGetContent(
	ruleHits []types.RuleID,
	ackedRules map[ctypes.RuleID]bool,
	disabledRulesForCluster map[ctypes.RuleID]bool,
) []types.SimplifiedRuleHit {
	// initialize the return value so that it's not nil (and in API response null)
	filteredRuleHits := []types.SimplifiedRuleHit{}

	for _, ruleID := range ruleHits {
		// skip acked rule
		if _, found := ackedRules[ruleID]; found {
			continue
		}

		// skip single disabled rules for given cluster
		if _, found := disabledRulesForCluster[ruleID]; found {
			continue
		}

		ruleContent, err := content.GetContentForRecommendation(ruleID)
		if err != nil {
			// rule content not found, log and skip as in other endpoints
			log.Warn().Err(err).Interface(ruleIDStr, ruleID).Msg("error retrieving rule content for rule")
			continue
		}

		ruleID, errorKey, err := types.RuleIDWithErrorKeyFromCompositeRuleID(ruleID)
		if err != nil {
			log.Warn().Msg("error getting rule module and error key from composite rule ID.")
		}
		// fill in data from rule content
		simplifiedRuleHit := types.SimplifiedRuleHit{
			RuleFQDN:    string(ruleID),
			ErrorKey:    string(errorKey),
			Description: ruleContent.Generic,
			TotalRisk:   ruleContent.TotalRisk,
		}

		filteredRuleHits = append(filteredRuleHits, simplifiedRuleHit)
	}

	return filteredRuleHits
}

func (server HTTPServer) getDisabledRulesForClusterMap(
	writer http.ResponseWriter,
	orgID types.OrgID,
	clusterID types.ClusterName,
) (
	disabledRules map[types.RuleID]bool, err error,
) {
	disabledRules = make(map[types.RuleID]bool)

	// use existing endpoint accepting list of clusters
	listOfDisabledRules, err := server.readListOfDisabledRulesForClusters(writer, orgID, []types.ClusterName{clusterID})
	if err != nil {
		log.Error().Err(err).Msg("error reading disabled rules from aggregator")
		handleServerError(writer, err)
		return
	}

	for _, disabledRule := range listOfDisabledRules {
		compositeRuleID, err := generateCompositeRuleIDFromDisabled(disabledRule)

		if err != nil {
			log.Error().Err(err).Msg("error generating composite rule ID")
			continue
		}

		disabledRules[compositeRuleID] = true
	}

	return
}

// checkRedisClientReadiness method checks if Redis client has been initialized
func (server *HTTPServer) checkRedisClientReadiness(writer http.ResponseWriter) bool {
	if server.redis == nil {
		handleServerError(writer, errors.New(RedisNotInitializedErrorMessage))
		return false
	}
	return true
}

// getDVONamespaceList returns a list of all DVO namespaces to which an account has access.
func (server *HTTPServer) getDVONamespaceList(writer http.ResponseWriter, request *http.Request) {
	tStart := time.Now()
	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		log.Error().Msg(authTokenFormatError)
		handleServerError(writer, err)
		return
	}

	// get active clusters info from AMS API
	activeClustersInfo, err := server.readClusterInfoForOrgID(orgID)
	if err != nil {
		log.Error().Err(err).Int(orgIDTag, int(orgID)).Msg(clusterListError)
		handleServerError(writer, err)
		return
	}
	clusterInfoMap := types.ClusterInfoArrayToMap(activeClustersInfo)

	log.Info().Int(orgIDTag, int(orgID)).Msgf("getDVONamespaceList took %v to get %d clusters from AMS API", time.Since(tStart), len(activeClustersInfo))

	// get workloads for clusters
	workloads, err := server.getWorkloadsForOrganization(orgID, writer, activeClustersInfo)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	log.Info().Int(orgIDTag, int(orgID)).Msgf("getDVONamespaceList took %v to get %d workloads from aggregator", time.Since(tStart), len(workloads))

	workloadsProcessed, err := processWorkloadsRecommendations(clusterInfoMap, workloads)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// prepare response
	responseData := map[string]interface{}{}
	responseData["status"] = OkMsg
	responseData["workloads"] = workloadsProcessed

	log.Info().Int(orgIDTag, int(orgID)).Msgf("getDVONamespaceList took %v to process response into %d results", time.Since(tStart), len(workloadsProcessed))

	// send response to client
	err = responses.SendOK(writer, responseData)
	if err != nil {
		handleServerError(writer, err)
		return
	}
}

// processWorkloadsRecommendations filter out inactive clusters; calculate aggregations by severity
func processWorkloadsRecommendations(
	clusterInfoMap map[ctypes.ClusterName]types.ClusterInfo,
	workloadsForCluster []types.WorkloadsForNamespace,
) (
	workloads []types.Workload,
	err error,
) {
	workloads = make([]types.Workload, 0)

	recommendationSeverities, uniqueSeverities, err := content.GetExternalRuleSeverities()
	if err != nil {
		return
	}

	for _, w := range workloadsForCluster {
		// fill in display name
		if clusterInfo, found := clusterInfoMap[ctypes.ClusterName(w.Cluster.UUID)]; found {
			w.Cluster.DisplayName = clusterInfo.DisplayName
		} else {
			// cluster is not active, omitting
			continue
		}

		// fill in all unique severities
		hitsBySeverity := make(map[int]int, 0)
		for _, severity := range uniqueSeverities {
			hitsBySeverity[severity] = 0
		}
		w.Metadata.HitsBySeverity = hitsBySeverity

		// calculate hits by severity and highest severity across all recommendations
		for recommendation, hitCount := range w.RecommendationsHitCount {
			if severity, found := recommendationSeverities[ctypes.RuleID(recommendation)]; found {
				w.Metadata.HitsBySeverity[severity] += hitCount

				if severity > w.Metadata.HighestSeverity {
					w.Metadata.HighestSeverity = severity
				}
			} else {
				log.Info().Msgf("recommendation ID [%v] not found in content. Skipping.", recommendation)
			}
		}

		workloads = append(workloads, types.Workload{
			Cluster:   w.Cluster,
			Namespace: w.Namespace,
			Metadata:  w.Metadata,
		})
	}

	return
}

// getDVONamespacesForCluster returns a DVO workload recommendations for a single namespace within a cluster
func (server *HTTPServer) getDVONamespacesForCluster(writer http.ResponseWriter, request *http.Request) {
	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		log.Error().Msg(authTokenFormatError)
		handleServerError(writer, err)
		return
	}

	clusterID, successful := httputils.ReadClusterName(writer, request)
	// Error message handled by function
	if !successful {
		return
	}

	namespace, err := readNamespace(writer, request)
	if err != nil {
		return
	}

	// get cluster info from AMS API
	if server.amsClient == nil && !server.Config.UseOrgClustersFallback {
		log.Error().Msg("unable to retrieve info about cluster")
		handleServerError(writer, &AMSAPIUnavailableError{})
		return
	}

	clusterInfo, err := server.amsClient.GetSingleClusterInfoForOrganization(orgID, clusterID)
	if err != nil {
		log.Error().Err(err).Int(orgIDTag, int(orgID)).Msg(clusterListError)
		handleServerError(writer, err)
		return
	}

	// get namespace data from aggregator
	workloads, err := server.getWorkloadsForCluster(orgID, clusterID, namespace)
	if err != nil {
		switch err.(type) {
		case *json.SyntaxError:
			msg := "aggregator provided a wrong response"
			log.Error().Err(err).Msg(msg)
			handleServerError(writer, errors.New(msg))
			return
		case *url.Error:
			log.Error().Err(err).Msg("aggregator is not responding")
			handleServerError(writer, &AggregatorServiceUnavailableError{})
			return
		default:
			handleServerError(writer, err)
			return
		}
	}

	workloadsProcessed, err := fillInWorkloadsData(clusterInfo, workloads)
	if err != nil {
		msg := "unable to fill in data from content-service"
		log.Error().Err(err).Msg(msg)
		handleServerError(writer, errors.New(msg))
		return
	}

	// prepare response
	responseData := map[string]interface{}{}
	responseData["status"] = OkMsg
	responseData["cluster"] = workloadsProcessed.Cluster
	responseData["namespace"] = workloadsProcessed.Namespace
	responseData["metadata"] = workloadsProcessed.Metadata
	responseData["recommendations"] = workloadsProcessed.Recommendations

	// send response to client
	err = responses.SendOK(writer, responseData)
	if err != nil {
		handleServerError(writer, err)
		return
	}
}

// fillInWorkloadsData fills in data acquired from content-service
func fillInWorkloadsData(
	clusterInfo types.ClusterInfo,
	workloadsForCluster types.WorkloadsForCluster,
) (
	workloads types.WorkloadsForCluster,
	err error,
) {
	recommendationSeverities, uniqueSeverities, err := content.GetExternalRuleSeverities()
	if err != nil {
		return
	}

	// fill in display name
	workloadsForCluster.Cluster.DisplayName = clusterInfo.DisplayName

	// fill in all unique severities
	hitsBySeverity := make(map[int]int, len(uniqueSeverities))
	for _, severity := range uniqueSeverities {
		hitsBySeverity[severity] = 0
	}
	workloadsForCluster.Metadata.HitsBySeverity = hitsBySeverity

	recommendations := []types.DVORecommendation{}

	// fill in severities and other data from rule content
	for i := range workloadsForCluster.Recommendations {
		recommendation := &workloadsForCluster.Recommendations[i]
		if severity, found := recommendationSeverities[ctypes.RuleID(recommendation.Check)]; found {
			workloadsForCluster.Metadata.HitsBySeverity[severity] += len(recommendation.Objects)

			if severity > workloadsForCluster.Metadata.HighestSeverity {
				workloadsForCluster.Metadata.HighestSeverity = severity
			}
		}

		err = fillDVORecommendationRuleContent(recommendation)
		if err != nil {
			return workloads, err
		}

		recommendations = append(recommendations, *recommendation)
	}

	workloads = types.WorkloadsForCluster{
		Cluster:         workloadsForCluster.Cluster,
		Namespace:       workloadsForCluster.Namespace,
		Metadata:        workloadsForCluster.Metadata,
		Recommendations: recommendations,
	}

	return
}

func fillDVORecommendationRuleContent(recommendation *types.DVORecommendation) error {
	ruleContent, err := content.GetContentForRecommendation(ctypes.RuleID(recommendation.Check))
	if err != nil {
		log.Error().Err(err).Interface("recommendation.Check", recommendation.Check).Msg(ruleContentError)
		return err
	}

	// fill DVORecommendation with data from content service
	recommendation.Details = ruleContent.Description
	recommendation.Resolution = ruleContent.Resolution
	recommendation.MoreInfo = ruleContent.MoreInfo
	recommendation.Reason = ruleContent.Reason
	recommendation.TotalRisk = ruleContent.TotalRisk
	recommendation.Modified = ruleContent.PublishDate.UTC().Format(time.RFC3339)

	return nil
}
