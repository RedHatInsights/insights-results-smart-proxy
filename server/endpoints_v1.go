// Copyright 2020, 2021  Red Hat, Inc
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
	"net/http"
	"path/filepath"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// MainEndpoint returns status ok
	MainEndpoint = ""
	// OldReportEndpoint is made for backwards compatibility// TODO: remove when UI fixes are merged
	OldReportEndpoint = "report/{org_id}/{cluster}"

	// ReportEndpoint returns report for provided {cluster}
	ReportEndpoint = "clusters/{cluster}/report"

	// ReportMetainfoEndpoint returns (meta)information about report for provided {cluster}
	ReportMetainfoEndpoint = "clusters/{cluster}/report/info"

	// ReportForListOfClustersEndpoint returns rule returns reports for provided list of clusters
	// Reports that are going to be returned are specified by list of cluster IDs that is part of path
	ReportForListOfClustersEndpoint = "clusters/{cluster_list}/reports"
	// ReportForListOfClustersPayloadEndpoint returns the latest reports for the given list of clusters
	// Reports that are going to be returned are specified by list of cluster IDs that is part of request body
	ReportForListOfClustersPayloadEndpoint = "clusters/reports"
	// RuleGroupsEndpoint is a simple redirect endpoint to the insights-content-service API specified in configuration
	RuleGroupsEndpoint = "groups"
	// RuleContent returns static content for {rule_id}
	RuleContent = "rules/{rule_id}/content"
	// RuleIDs returns a list of rule IDs
	RuleIDs = "rule_ids"
	// Content returns all the static content avaiable for the user
	Content = "content"
	// SingleRuleEndpoint returns single rule with static content for {cluster} and {rule_id}
	SingleRuleEndpoint = "clusters/{cluster}/rules/{rule_id}/report"
	// MetricsEndpoint returns prometheus metrics
	MetricsEndpoint = "metrics"
	// LikeRuleEndpoint likes rule with {rule_id} for {cluster} using current user(from auth header)
	LikeRuleEndpoint = "clusters/{cluster}/rules/{rule_id}/error_key/{error_key}/like"
	// DislikeRuleEndpoint dislikes rule with {rule_id} for {cluster} using current user(from auth header)
	DislikeRuleEndpoint = "clusters/{cluster}/rules/{rule_id}/error_key/{error_key}/dislike"
	// ResetVoteOnRuleEndpoint resets vote on rule with {rule_id} for {cluster} using current user(from auth header)
	ResetVoteOnRuleEndpoint = "clusters/{cluster}/rules/{rule_id}/error_key/{error_key}/reset_vote"
	// GetVoteOnRuleEndpoint is an endpoint to get vote on rule. DEBUG only
	GetVoteOnRuleEndpoint = "clusters/{cluster}/rules/{rule_id}/error_key/{error_key}/get_vote"
	// DisableRuleForClusterEndpoint disables a rule for specified cluster
	DisableRuleForClusterEndpoint = "clusters/{cluster}/rules/{rule_id}/error_key/{error_key}/disable"
	// EnableRuleForClusterEndpoint re-enables a rule for specified cluster
	EnableRuleForClusterEndpoint = "clusters/{cluster}/rules/{rule_id}/error_key/{error_key}/enable"
	// DisableRuleFeedbackEndpoint accepts a feedback from user when (s)he disables a rule
	DisableRuleFeedbackEndpoint = "clusters/{cluster}/rules/{rule_id}/error_key/{error_key}/disable_feedback"
	// OverviewEndpoint returns some overview data for the clusters belonging to the org id
	OverviewEndpoint = "org_overview"

	// ClustersForOrganizationEndpoint returns all clusters for {organization}
	ClustersForOrganizationEndpoint = ira_server.ClustersForOrganizationEndpoint
	// OrganizationsEndpoint returns all organizations
	OrganizationsEndpoint = ira_server.OrganizationsEndpoint
	// DeleteOrganizationsEndpoint deletes all {organizations}(comma separated array). DEBUG only
	DeleteOrganizationsEndpoint = ira_server.DeleteOrganizationsEndpoint
	// DeleteClustersEndpoint deletes all {clusters}(comma separated array). DEBUG only
	DeleteClustersEndpoint = ira_server.DeleteClustersEndpoint

	// InfoEndpoint returns basic information about content service
	// version, utils repository version, commit hash etc.
	InfoEndpoint = "info"
)

// addV1EndpointsToRouter adds API V1 specific endpoints to the router
func (server *HTTPServer) addV1EndpointsToRouter(router *mux.Router) {
	apiPrefix := server.Config.APIv1Prefix
	openAPIURL := apiPrefix + filepath.Base(server.Config.APIv1SpecFile)
	aggregatorBaseEndpoint := server.ServicesConfig.AggregatorBaseEndpoint

	// It is possible to use special REST API endpoints in debug mode
	if server.Config.Debug {
		server.addV1DebugEndpointsToRouter(router, apiPrefix, aggregatorBaseEndpoint)
	}

	// Common REST API endpoints
	router.HandleFunc(apiPrefix+MainEndpoint, server.mainEndpoint).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+ClustersForOrganizationEndpoint, server.getClustersForOrg).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+OverviewEndpoint, server.overviewEndpoint).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+OverviewEndpoint, server.overviewEndpointWithClusterIDs).Methods(http.MethodPost)
	router.HandleFunc(apiPrefix+InfoEndpoint, server.infoMap).Methods(http.MethodGet, http.MethodOptions)

	// Reports endpoints
	server.addV1ReportsEndpointsToRouter(router, apiPrefix, aggregatorBaseEndpoint)

	// Content related endpoints
	server.addV1ContentEndpointsToRouter(router)

	// Rules related endpoints
	server.addV1RuleEndpointsToRouter(router, apiPrefix, aggregatorBaseEndpoint)

	// Prometheus metrics
	router.Handle(apiPrefix+MetricsEndpoint, promhttp.Handler()).Methods(http.MethodGet)

	// OpenAPI specs
	router.HandleFunc(
		openAPIURL,
		httputils.CreateOpenAPIHandler(server.Config.APIv1SpecFile, server.Config.Debug, true),
	).Methods(http.MethodGet)
}

// addV1ReportsEndpointsToRouter method registers handlers for endpoints that
// return cluster report or reports to client
func (server *HTTPServer) addV1ReportsEndpointsToRouter(router *mux.Router, apiPrefix, aggregatorBaseURL string) {
	router.HandleFunc(apiPrefix+OldReportEndpoint, server.reportEndpointV1).Methods(http.MethodGet, http.MethodOptions)
	router.HandleFunc(apiPrefix+ReportEndpoint, server.reportEndpointV1).Methods(http.MethodGet, http.MethodOptions)
	router.HandleFunc(apiPrefix+ReportMetainfoEndpoint, server.reportMetainfoEndpoint).Methods(http.MethodGet, http.MethodOptions)
	router.HandleFunc(apiPrefix+ReportForListOfClustersEndpoint, server.reportForListOfClustersEndpoint).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+ReportForListOfClustersPayloadEndpoint, server.reportForListOfClustersPayloadEndpoint).Methods(http.MethodPost)
}

// addV1DebugEndpointsToRouter method registers handlers for all debug endpoints
func (server *HTTPServer) addV1DebugEndpointsToRouter(router *mux.Router, apiPrefix, aggregatorBaseURL string) {
	router.HandleFunc(apiPrefix+OrganizationsEndpoint, server.proxyTo(aggregatorBaseURL, nil)).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+DeleteOrganizationsEndpoint, server.proxyTo(aggregatorBaseURL, nil)).Methods(http.MethodDelete)
	router.HandleFunc(apiPrefix+DeleteClustersEndpoint, server.proxyTo(aggregatorBaseURL, nil)).Methods(http.MethodDelete)

	router.HandleFunc(apiPrefix+GetVoteOnRuleEndpoint, server.proxyTo(
		aggregatorBaseURL,
		&ProxyOptions{RequestModifiers: []RequestModifier{
			server.newExtractUserIDFromTokenToURLRequestModifier(ira_server.GetVoteOnRuleEndpoint),
		}},
	)).Methods(http.MethodGet)
}

// addV1RuleEndpointsToRouter method registers handlers for endpoints that handle
// rule-related operations (voting etc.)
func (server *HTTPServer) addV1RuleEndpointsToRouter(router *mux.Router, apiPrefix, aggregatorBaseEndpoint string) {
	router.HandleFunc(apiPrefix+SingleRuleEndpoint, server.singleRuleEndpoint).Methods(http.MethodGet, http.MethodOptions)

	router.HandleFunc(apiPrefix+LikeRuleEndpoint, server.proxyTo(
		aggregatorBaseEndpoint,
		&ProxyOptions{RequestModifiers: []RequestModifier{
			server.newExtractUserIDFromTokenToURLRequestModifier(ira_server.LikeRuleEndpoint),
		}},
	)).Methods(http.MethodPut, http.MethodOptions)

	router.HandleFunc(apiPrefix+DislikeRuleEndpoint, server.proxyTo(
		aggregatorBaseEndpoint,
		&ProxyOptions{RequestModifiers: []RequestModifier{
			server.newExtractUserIDFromTokenToURLRequestModifier(ira_server.DislikeRuleEndpoint),
		}},
	)).Methods(http.MethodPut, http.MethodOptions)

	router.HandleFunc(apiPrefix+ResetVoteOnRuleEndpoint, server.proxyTo(
		aggregatorBaseEndpoint,
		&ProxyOptions{RequestModifiers: []RequestModifier{
			server.newExtractUserIDFromTokenToURLRequestModifier(ira_server.ResetVoteOnRuleEndpoint),
		}},
	)).Methods(http.MethodPut, http.MethodOptions)

	router.HandleFunc(apiPrefix+DisableRuleForClusterEndpoint, server.proxyTo(
		aggregatorBaseEndpoint,
		&ProxyOptions{RequestModifiers: []RequestModifier{
			server.newExtractUserIDFromTokenToURLRequestModifier(ira_server.DisableRuleForClusterEndpoint),
		}},
	)).Methods(http.MethodPut, http.MethodOptions)

	router.HandleFunc(apiPrefix+EnableRuleForClusterEndpoint, server.proxyTo(
		aggregatorBaseEndpoint,
		&ProxyOptions{RequestModifiers: []RequestModifier{
			server.newExtractUserIDFromTokenToURLRequestModifier(ira_server.EnableRuleForClusterEndpoint),
		}},
	)).Methods(http.MethodPut, http.MethodOptions)

	router.HandleFunc(apiPrefix+DisableRuleFeedbackEndpoint, server.proxyTo(
		aggregatorBaseEndpoint,
		&ProxyOptions{RequestModifiers: []RequestModifier{
			server.newExtractUserIDFromTokenToURLRequestModifier(ira_server.DisableRuleFeedbackEndpoint),
		}},
	)).Methods(http.MethodPost, http.MethodOptions)
}

// addV1ContentEndpointsToRouter method registers handlers for endpoints that
// returns content to clients
func (server HTTPServer) addV1ContentEndpointsToRouter(router *mux.Router) {
	apiPrefix := server.Config.APIv1Prefix

	router.HandleFunc(apiPrefix+RuleGroupsEndpoint, server.getGroups).Methods(http.MethodGet, http.MethodOptions)
	router.HandleFunc(apiPrefix+RuleContent, server.getContentForRuleV1).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+RuleIDs, server.getRuleIDs).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+Content, server.getContentV1).Methods(http.MethodGet)
}
