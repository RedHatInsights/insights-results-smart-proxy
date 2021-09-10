// Copyright 2020 Red Hat, Inc
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

// TODO cleanup this file to contain only the debug endpoints

package server

import (
	"encoding/json"
	"net/http"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	sptypes "github.com/RedHatInsights/insights-results-smart-proxy/types"
)

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
func (server HTTPServer) getContentForRule(writer http.ResponseWriter, request *http.Request) {
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

	err = responses.SendOK(writer, responses.BuildOkResponseWithData("content", ruleContent))
	if err != nil {
		handleServerError(writer, err)
		return
	}
}

// getContent retrieves all the static content
func (server HTTPServer) getContent(writer http.ResponseWriter, request *http.Request) {
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

	err := responses.SendOK(writer, responses.BuildOkResponseWithData("content", rules))
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
	return
}

// getRuleIDs returns a list of the names of the rules
func (server HTTPServer) getRuleIDs(writer http.ResponseWriter, request *http.Request) {
	allRuleIDs := content.GetRuleIDs()
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

// overviewEndpoint returns a map with an overview of number of clusters hit by rules
func (server HTTPServer) overviewEndpoint(writer http.ResponseWriter, request *http.Request) {
	authToken, err := server.GetAuthToken(request)
	orgID := authToken.Internal.OrgID
	if err != nil {
		handleServerError(writer, err)
		return
	}

	clustersHits := 0
	hitsByTotalRisk := make(map[int]int)
	hitsByTags := make(map[string]int)

	clusters, err := server.readClusterIDsForOrgID(orgID)
	if err != nil {
		handleServerError(writer, err)
		return
	}
	log.Info().Msgf("Retrieving overview for org_id %v and its clusters: %v", orgID, clusters)

	for _, clusterID := range clusters {
		overview, err := server.getOverviewPerCluster(clusterID, authToken, writer)
		if err != nil {
			log.Error().Err(err).Msgf("Problem handling report for cluster %s.", clusterID)
			continue
		}

		if overview == nil {
			log.Error().Msgf("Overview for cluster %v is empty. Skipping.", clusterID)
			continue
		}

		clustersHits++
		for _, totalRisk := range overview.TotalRisksHit {
			hitsByTotalRisk[totalRisk]++
		}

		for _, tag := range overview.TagsHit {
			hitsByTags[tag]++
		}
	}

	r := sptypes.OrgOverviewResponse{
		ClustersHit:            clustersHits,
		ClustersHitByTotalRisk: hitsByTotalRisk,
		ClustersHitByTag:       hitsByTags,
	}

	if err = responses.SendOK(writer, responses.BuildOkResponseWithData("overview", r)); err != nil {
		handleServerError(writer, err)
		return
	}
}

// overviewEndpointWithClusterIDs returns a map with an overview of number of clusters hit by rules
func (server HTTPServer) overviewEndpointWithClusterIDs(writer http.ResponseWriter, request *http.Request) {
	// get reports for the cluster list in body
	log.Info().Msg("Retrieving reports for clusters to generate org_overview")
	aggregatorResponse, ok := server.fetchAggregatorReportsUsingRequestBodyClusterList(writer, request)
	if !ok {
		// errors already handled
		return
	}

	r := generateOrgOverview(aggregatorResponse)

	if err := responses.SendOK(writer, responses.BuildOkResponseWithData("overview", r)); err != nil {
		handleServerError(writer, err)
		return
	}
}

// generateOrgOverview generates an OrgOverviewResponse from the aggregator's response
func generateOrgOverview(aggregatorReport *types.ClusterReports) sptypes.OrgOverviewResponse {
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

		clustersHits++

		for _, rule := range clusterReport.HitRules {
			if rule.Disabled {
				continue
			}

			ruleID := rule.Module
			errorKey := rule.ErrorKey
			ruleWithContent, err := content.GetRuleWithErrorKeyContent(ruleID, errorKey)
			if err != nil {
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
	}
}
