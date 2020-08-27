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
	LikeRuleEndpoint = "clusters/{cluster}/rules/{rule_id}/like"
	// DislikeRuleEndpoint dislikes rule with {rule_id} for {cluster} using current user(from auth header)
	DislikeRuleEndpoint = "clusters/{cluster}/rules/{rule_id}/dislike"
	// ResetVoteOnRuleEndpoint resets vote on rule with {rule_id} for {cluster} using current user(from auth header)
	ResetVoteOnRuleEndpoint = "clusters/{cluster}/rules/{rule_id}/reset_vote"
	// GetVoteOnRuleEndpoint is an endpoint to get vote on rule. DEBUG only
	GetVoteOnRuleEndpoint = "clusters/{cluster}/rules/{rule_id}/get_vote"
	// DisableRuleForClusterEndpoint disables a rule for specified cluster
	DisableRuleForClusterEndpoint = "clusters/{cluster}/rules/{rule_id}/disable"
	// EnableRuleForClusterEndpoint re-enables a rule for specified cluster
	EnableRuleForClusterEndpoint = "clusters/{cluster}/rules/{rule_id}/enable"
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
)

func (server *HTTPServer) addDebugEndpointsToRouter(router *mux.Router) {
	apiPrefix := server.Config.APIPrefix
	aggregatorBaseURL := server.ServicesConfig.AggregatorBaseEndpoint

	router.HandleFunc(apiPrefix+OrganizationsEndpoint, server.proxyTo(aggregatorBaseURL, nil)).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+DeleteOrganizationsEndpoint, server.proxyTo(aggregatorBaseURL, nil)).Methods(http.MethodDelete)
	router.HandleFunc(apiPrefix+DeleteClustersEndpoint, server.proxyTo(aggregatorBaseURL, nil)).Methods(http.MethodDelete)
	router.HandleFunc(apiPrefix+GetVoteOnRuleEndpoint, server.proxyTo(
		aggregatorBaseURL,
		&ProxyOptions{RequestModifiers: []RequestModifier{
			server.newExtractUserIDFromTokenToURLRequestModifier(ira_server.GetVoteOnRuleEndpoint),
		}},
	),
	).Methods(http.MethodGet)

	// endpoints for pprof - needed for profiling, ie. usually in debug mode
	router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
}

func (server *HTTPServer) addEndpointsToRouter(router *mux.Router) {
	apiPrefix := server.Config.APIPrefix
	openAPIURL := apiPrefix + filepath.Base(server.Config.APISpecFile)
	aggregatorEndpoint := server.ServicesConfig.AggregatorBaseEndpoint

	// it is possible to use special REST API endpoints in debug mode
	if server.Config.Debug {
		server.addDebugEndpointsToRouter(router)
	}

	// common REST API endpoints
	router.HandleFunc(apiPrefix+MainEndpoint, server.mainEndpoint).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+SingleRuleEndpoint, server.singleRuleEndpoint).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+OldReportEndpoint, server.reportEndpoint).Methods(http.MethodGet, http.MethodOptions)
	router.HandleFunc(apiPrefix+ReportEndpoint, server.reportEndpoint).Methods(http.MethodGet, http.MethodOptions)
	router.HandleFunc(apiPrefix+LikeRuleEndpoint, server.proxyTo(
		aggregatorEndpoint,
		&ProxyOptions{RequestModifiers: []RequestModifier{
			server.newExtractUserIDFromTokenToURLRequestModifier(ira_server.LikeRuleEndpoint),
		}},
	)).Methods(http.MethodPut, http.MethodOptions)
	router.HandleFunc(apiPrefix+DislikeRuleEndpoint, server.proxyTo(
		aggregatorEndpoint,
		&ProxyOptions{RequestModifiers: []RequestModifier{
			server.newExtractUserIDFromTokenToURLRequestModifier(ira_server.DislikeRuleEndpoint),
		}},
	)).Methods(http.MethodPut, http.MethodOptions)
	router.HandleFunc(apiPrefix+ResetVoteOnRuleEndpoint, server.proxyTo(
		aggregatorEndpoint,
		&ProxyOptions{RequestModifiers: []RequestModifier{
			server.newExtractUserIDFromTokenToURLRequestModifier(ira_server.ResetVoteOnRuleEndpoint),
		}},
	)).Methods(http.MethodPut, http.MethodOptions)
	router.HandleFunc(apiPrefix+ClustersForOrganizationEndpoint, server.getClustersForOrg).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+DisableRuleForClusterEndpoint, server.proxyTo(
		aggregatorEndpoint,
		&ProxyOptions{RequestModifiers: []RequestModifier{
			server.newExtractUserIDFromTokenToURLRequestModifier(ira_server.DisableRuleForClusterEndpoint),
		}},
	)).Methods(http.MethodPut, http.MethodOptions)
	router.HandleFunc(apiPrefix+EnableRuleForClusterEndpoint, server.proxyTo(
		aggregatorEndpoint,
		&ProxyOptions{RequestModifiers: []RequestModifier{
			server.newExtractUserIDFromTokenToURLRequestModifier(ira_server.EnableRuleForClusterEndpoint),
		}},
	)).Methods(http.MethodPut, http.MethodOptions)
	router.HandleFunc(apiPrefix+OverviewEndpoint, server.overviewEndpoint).Methods(http.MethodGet)

	// Content related endpoints
	server.addContentEndpointsToRouter(router)

	// Prometheus metrics
	router.Handle(apiPrefix+MetricsEndpoint, promhttp.Handler()).Methods(http.MethodGet)

	// OpenAPI specs
	router.HandleFunc(
		openAPIURL,
		httputils.CreateOpenAPIHandler(server.Config.APISpecFile, server.Config.Debug),
	).Methods(http.MethodGet)
}

func (server HTTPServer) addContentEndpointsToRouter(router *mux.Router) {
	apiPrefix := server.Config.APIPrefix

	router.HandleFunc(apiPrefix+RuleGroupsEndpoint, server.getGroups).Methods(http.MethodGet, http.MethodOptions)
	router.HandleFunc(apiPrefix+RuleContent, server.getContentForRule).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+RuleIDs, server.getRuleIDs).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+Content, server.getContent).Methods(http.MethodGet)
}
