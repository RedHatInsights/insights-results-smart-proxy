// Copyright 2020, 2021, 2022 Red Hat, Inc
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
	"errors"
	"fmt"
	"io"
	"net/http"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-operator-utils/responses"
	types "github.com/RedHatInsights/insights-results-types"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"
	sptypes "github.com/RedHatInsights/insights-results-smart-proxy/types"
)

const filledIn = "ok"
const infoEndpoint = "info"

// infoEndpointStruct represent response for /info endpoint from Insights
// Results Aggregator or from Content Service
type infoEndpointStruct struct {
	Status string            `json:"status"`
	Info   map[string]string `json:"info"`
}

// getGroups sends the latest valid groups configuration to the client in
// standard HTTP response
func (server *HTTPServer) getGroups(writer http.ResponseWriter, _ *http.Request) {
	// retrieve the latest groups configuration
	groupsConfig, err := server.getGroupsConfig()
	if err != nil {
		handleServerError(writer, err)
		return
	}

	responseContent := make(map[string]interface{})
	responseContent["status"] = "ok"
	responseContent["groups"] = groupsConfig
	err = responses.SendOK(writer, responseContent)
	if err != nil {
		log.Error().Err(err).Msg("Cannot send response")
		handleServerError(writer, err)
	}
}

// getContentForRule retrieves the static content for the given ruleID
func (server HTTPServer) getContentForRuleV1(writer http.ResponseWriter, request *http.Request) {
	ruleID, successful := httputils.ReadRuleID(writer, request)
	if !successful {
		// already handled in readRuleID
		return
	}

	ruleContent, err := content.GetRuleContentV1(ruleID)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// check for internal rule permissions
	if internal := content.IsRuleInternal(ruleID); internal {
		err := server.checkInternalRulePermissions(request)
		if err != nil {
			handleServerError(writer, err)
			return
		}
	}

	err = responses.SendOK(writer, responses.BuildOkResponseWithData("content", ruleContent))
	if err != nil {
		handleServerError(writer, err)
		return
	}
}

// getContent retrieves all the static content
func (server HTTPServer) getContentV1(writer http.ResponseWriter, request *http.Request) {
	// Generate an array of RuleContent
	allRules, err := content.GetAllContentV1()

	if err != nil {
		log.Error().Err(err)
		handleServerError(writer, err)
		return
	}

	var rules []sptypes.RuleContentV1

	if err := server.checkInternalRulePermissions(request); err != nil {
		for _, rule := range allRules {
			if !content.IsRuleInternal(types.RuleID(rule.Plugin.PythonModule)) {
				rules = append(rules, rule)
			}
		}
	} else {
		rules = allRules
	}

	err = responses.SendOK(writer, responses.BuildOkResponseWithData("content", rules))
	if err != nil {
		handleServerError(writer, err)
		return
	}
}

// getClustersForOrg retrieves the list of clusters belonging to this organization
func (server HTTPServer) getClustersForOrg(writer http.ResponseWriter, request *http.Request) {
	// readOrganizationID is done only for checking the authentication
	_, successful := httputils.ReadOrganizationID(writer, request, server.Config.Auth)
	if !successful {
		// already handled in readOrganizationID ?
		return
	}

	server.proxyTo(server.ServicesConfig.AggregatorBaseEndpoint, nil)(writer, request)
}

// getRuleIDs returns a list of the names of the rules
func (server HTTPServer) getRuleIDs(writer http.ResponseWriter, request *http.Request) {
	allRuleIDs, err := content.GetRuleIDs()

	if err != nil {
		log.Error().Err(err)
		handleServerError(writer, err)
		return
	}

	var ruleIDs []string

	if err := server.checkInternalRulePermissions(request); err != nil {
		for _, rule := range allRuleIDs {
			if !content.IsRuleInternal(types.RuleID(rule)) {
				ruleIDs = append(ruleIDs, rule)
			}
		}
	} else {
		ruleIDs = allRuleIDs
	}

	if err := responses.SendOK(writer, responses.BuildOkResponseWithData("rules", ruleIDs)); err != nil {
		log.Error().Err(err)
		handleServerError(writer, err)
		return
	}
}

func (server HTTPServer) getOrganizationOverview(
	clusterInfoList []sptypes.ClusterInfo,
	clusterRecommendationsMap types.ClusterRecommendationMap,
	systemWideDisabledRules map[types.RuleID]bool,
	disabledRulesPerCluster map[types.ClusterName][]types.RuleID,
) (
	sptypes.OrgOverviewResponse, error,
) {
	overview := sptypes.OrgOverviewResponse{
		ClustersHitByTotalRisk: make(map[int]int),
		ClustersHitByTag:       make(map[string]int),
	}

	// iterates over clusters and their hitting recommendations, accesses map to the get rule severity
	for i := range clusterInfoList {
		clusterInfo := &clusterInfoList[i]

		// check if there are any hitting recommendations
		hittingRecommendations, any := clusterRecommendationsMap[clusterInfo.ID]
		if !any {
			continue
		}

		// filter out acked and disabled rules
		enabledOnlyRecommendations := filterOutDisabledRules(
			hittingRecommendations.Recommendations, clusterInfo.ID,
			systemWideDisabledRules, disabledRulesPerCluster,
		)

		var filteredRecommendations int
		for _, ruleID := range enabledOnlyRecommendations {
			ruleContent, err := content.GetContentForRecommendation(ruleID)
			if err != nil {
				if err, ok := err.(*content.RuleContentDirectoryTimeoutError); ok {
					return overview, err
				}
				// missing rule content, simply omit the rule as we can't display anything
				log.Error().Err(err).Msgf("unable to get content for rule with id %v", ruleID)
				filteredRecommendations++
				continue
			}

			if clusterInfo.Managed && !ruleContent.OSDCustomer {
				// cluster is managed, therefore must count only managed rules
				filteredRecommendations++
				continue
			}

			overview.ClustersHitByTotalRisk[ruleContent.TotalRisk]++

			for _, tag := range ruleContent.Tags {
				overview.ClustersHitByTag[tag]++
			}
		}

		// to avoid edge case where all rules are filtered
		if len(enabledOnlyRecommendations)-filteredRecommendations > 0 {
			overview.ClustersHit++
		}
	}

	return overview, nil
}

// overviewEndpoint returns a map with an overview of number of clusters hit by rules
func (server HTTPServer) overviewEndpoint(writer http.ResponseWriter, request *http.Request) {
	orgID, userID, err := server.GetCurrentOrgIDUserIDFromToken(request)
	if err != nil {
		log.Err(err).Msg(orgIDTokenError)
		handleServerError(writer, err)
		return
	}
	log.Info().Int(orgIDTag, int(orgID)).Str(userIDTag, string(userID)).Msg("getClustersView start")

	clusterList, clusterRuleHits, ackedRulesMap, disabledRules := server.getClusterListAndUserData(
		writer,
		orgID,
		userID,
	)

	overview, err := server.getOrganizationOverview(clusterList, clusterRuleHits, ackedRulesMap, disabledRules)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	if err = responses.SendOK(writer, responses.BuildOkResponseWithData("overview", overview)); err != nil {
		handleServerError(writer, err)
		return
	}
}

// overviewEndpointWithClusterIDs returns a map with an overview of number of clusters hit by rules
func (server HTTPServer) overviewEndpointWithClusterIDs(writer http.ResponseWriter, request *http.Request) {
	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// get reports for the cluster list in body
	log.Info().Msg("Retrieving reports for clusters to generate org_overview")
	aggregatorResponse, ok := server.fetchAggregatorReportsUsingRequestBodyClusterList(writer, request)
	if !ok {
		// errors already handled
		return
	}

	// retrieve rule acknowledgements (disable/enable)
	acks, err := server.readListOfAckedRules(orgID)
	if err != nil {
		log.Error().Err(err).Msg(ackedRulesError)
		// server error has been handled already
		return
	}
	orgWideDisabledRules := generateRuleAckMap(acks)

	r, err := generateOrgOverview(aggregatorResponse, orgWideDisabledRules)

	if err != nil {
		handleServerError(writer, err)
		return
	}

	if err = responses.SendOK(writer, responses.BuildOkResponseWithData("overview", r)); err != nil {
		handleServerError(writer, err)
		return
	}
}

// generateOrgOverview generates an OrgOverviewResponse from the aggregator's response
func generateOrgOverview(
	aggregatorReport *types.ClusterReports,
	orgWideDisabledRules map[types.RuleID]bool,
) (sptypes.OrgOverviewResponse, error) {
	clustersHits := 0
	hitsByTotalRisk := make(map[int]int)
	hitsByTags := make(map[string]int)

	for _, singleReport := range aggregatorReport.Reports {
		var clusterReport types.ReportRules

		if err := json.Unmarshal(singleReport, &clusterReport); err != nil {
			log.Error().Err(err).Msgf("The report %v is not ok", singleReport)
			continue
		}

		if len(clusterReport.HitRules) == 0 {
			continue
		}

		//TO-DO: If we have a cluster where all the rules are disabled, it will still count. is that ok?
		clustersHits++

		for _, rule := range clusterReport.HitRules {
			if isDisabledRule(rule, orgWideDisabledRules) {
				continue
			}

			ruleID := rule.Module
			errorKey := rule.ErrorKey
			ruleWithContent, err := content.GetRuleWithErrorKeyContent(ruleID, errorKey)
			if err != nil {
				if _, ok := err.(*content.RuleContentDirectoryTimeoutError); ok {
					return sptypes.OrgOverviewResponse{}, err
				}
				log.Error().Err(err).Msgf("Unable to retrieve content for rule %s", ruleID)
				continue
			}

			hitsByTotalRisk[ruleWithContent.TotalRisk]++

			for _, tag := range ruleWithContent.Tags {
				hitsByTags[tag]++
			}
		}

	}

	return sptypes.OrgOverviewResponse{
		ClustersHit:            clustersHits,
		ClustersHitByTotalRisk: hitsByTotalRisk,
		ClustersHitByTag:       hitsByTags,
	}, nil
}

// infoMap returns map of additional information about this service, Insights
// Results Aggregator, and Smart Proxy
func (server *HTTPServer) infoMap(writer http.ResponseWriter, request *http.Request) {
	// prepare response data structure
	response := sptypes.InfoResponse{
		SmartProxy:     server.fillInSmartProxyInfoParams(),
		ContentService: server.fillInContentServiceInfoParams(),
		Aggregator:     server.fillInAggregatorInfoParams(),
	}

	// try to send the response to client
	err := responses.SendOK(writer, responses.BuildOkResponseWithData("info", response))
	if err != nil {
		log.Error().Err(err)
		handleServerError(writer, err)
		return
	}
}

// fillInSmartProxyInfoParams method fills-in info parameters needed for /info
// REST API endpoint for the Smart Proxy itself
func (server *HTTPServer) fillInSmartProxyInfoParams() map[string]string {
	// fill-in info params for Smart Proxy
	if server.InfoParams == nil {
		const msg = "InfoParams is empty"
		err := errors.New(msg)
		log.Error().Err(err)

		// don't fail, just fill in the field
		m := make(map[string]string)
		m["status"] = msg
		return m
	}

	// info params for Smart Proxy is filled-in properly
	m := server.InfoParams
	m["status"] = filledIn
	return m
}

// fillInContentServiceInfoParams method fills-in info parameters needed for
// /info REST API endpoint for the Content Service
func (server *HTTPServer) fillInContentServiceInfoParams() map[string]string {
	// try to access Content Service
	url := httputils.MakeURLToEndpoint(
		server.ServicesConfig.ContentBaseEndpoint,
		infoEndpoint)
	return infoFromService(url)
}

// fillInAggregatorInfoParams method fills-in info parameters needed for /info
// REST API endpoint for the Insights Results Aggregator
func (server *HTTPServer) fillInAggregatorInfoParams() map[string]string {
	// try to access Insights Results Aggregator
	url := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		infoEndpoint)
	return infoFromService(url)
}

// infoFromService retrieves info parameters through /info endpoint and make a
// map from it
func infoFromService(url string) map[string]string {
	log.Info().Str("URL to service endpoint", url).Msg("Getting info from service")
	m, err := readInfoAPIEndpoint(url)

	// service access was not ok
	if err != nil {
		log.Error().Err(err).Msg("Error retrieving info from service")
		m := make(map[string]string)
		m["status"] = err.Error()
		return m
	}

	// service access was ok, so let's just add a status field into the map
	m["status"] = filledIn
	return m
}

// readInfoAPIEndpoint function performs REST API request and parse the
// returned response
func readInfoAPIEndpoint(url string) (map[string]string, error) {
	// perform GET request to given service
	// nolint:bodyclose
	response, err := http.Get(url) // #nosec G107

	// error happening during GET request
	if err != nil {
		return nil, err
	}

	defer services.CloseResponseBody(response)

	// check the status code
	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("Improper status code %d", response.StatusCode)
		return nil, err
	}

	// try to read response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		err = errors.New("Problem reading response from /info endpoint")
		return nil, err
	}

	// try to unmarshal response body
	var decoded infoEndpointStruct

	err = json.Unmarshal(body, &decoded)
	if err != nil {
		err = errors.New("Problem unmarshalling JSON response from /info endpoint")
		return nil, err
	}

	// unmarshalling was ok, return the Info part (whatever it contains)
	return decoded.Info, nil
}
