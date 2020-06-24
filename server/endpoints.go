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

	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// MainEndpoint returns status ok
	MainEndpoint = ""
	// ReportEndpoint returns report for provided {cluster}
	ReportEndpoint = "clusters/{cluster}/report"
	// RuleGroupsEndpoint is a simple redirect endpoint to the insights-content-service API specified in configuration
	RuleGroupsEndpoint = "groups"
	// RuleContent returns static content for {rule_id}
	RuleContent = "rules/{rule_id}/content"
	// SingleRuleEndpoint returns single rule with static content for {cluster} and {rule_id}
	SingleRuleEndpoint = "clusters/{cluster}/rules/{rule_id}/report"
	// MetricsEndpoint returns prometheus metrics
	MetricsEndpoint = "metrics"

	// LikeRuleEndpoint likes rule with {rule_id} for {cluster} using current user(from auth header)
	LikeRuleEndpoint = ira_server.LikeRuleEndpoint
	// DislikeRuleEndpoint dislikes rule with {rule_id} for {cluster} using current user(from auth header)
	DislikeRuleEndpoint = ira_server.DislikeRuleEndpoint
	// ResetVoteOnRuleEndpoint resets vote on rule with {rule_id} for {cluster} using current user(from auth header)
	ResetVoteOnRuleEndpoint = ira_server.ResetVoteOnRuleEndpoint
	// ClustersForOrganizationEndpoint returns all clusters for {organization}
	ClustersForOrganizationEndpoint = ira_server.ClustersForOrganizationEndpoint
	// DisableRuleForClusterEndpoint disables a rule for specified cluster
	DisableRuleForClusterEndpoint = ira_server.DisableRuleForClusterEndpoint
	// EnableRuleForClusterEndpoint re-enables a rule for specified cluster
	EnableRuleForClusterEndpoint = ira_server.EnableRuleForClusterEndpoint
	// OrganizationsEndpoint returns all organizations
	OrganizationsEndpoint = ira_server.OrganizationsEndpoint
	// DeleteOrganizationsEndpoint deletes all {organizations}(comma separated array). DEBUG only
	DeleteOrganizationsEndpoint = ira_server.DeleteOrganizationsEndpoint
	// DeleteClustersEndpoint deletes all {clusters}(comma separated array). DEBUG only
	DeleteClustersEndpoint = ira_server.DeleteClustersEndpoint
	// GetVoteOnRuleEndpoint is an endpoint to get vote on rule. DEBUG only
	GetVoteOnRuleEndpoint = ira_server.GetVoteOnRuleEndpoint
)

func (server *HTTPServer) addDebugEndpointsToRouter(router *mux.Router) {
	apiPrefix := server.Config.APIPrefix
	aggregatorEndpoint := server.ServicesConfig.AggregatorBaseEndpoint

	router.HandleFunc(apiPrefix+OrganizationsEndpoint, server.proxyTo(aggregatorEndpoint)).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+DeleteOrganizationsEndpoint, server.proxyTo(aggregatorEndpoint)).Methods(http.MethodDelete)
	router.HandleFunc(apiPrefix+DeleteClustersEndpoint, server.proxyTo(aggregatorEndpoint)).Methods(http.MethodDelete)
	router.HandleFunc(apiPrefix+GetVoteOnRuleEndpoint, server.proxyTo(aggregatorEndpoint)).Methods(http.MethodGet)

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
	router.HandleFunc(apiPrefix+ReportEndpoint, server.reportEndpoint).Methods(http.MethodGet, http.MethodOptions)
	router.HandleFunc(apiPrefix+LikeRuleEndpoint, server.proxyTo(aggregatorEndpoint)).Methods(http.MethodPut, http.MethodOptions)
	router.HandleFunc(apiPrefix+DislikeRuleEndpoint, server.proxyTo(aggregatorEndpoint)).Methods(http.MethodPut, http.MethodOptions)
	router.HandleFunc(apiPrefix+ResetVoteOnRuleEndpoint, server.proxyTo(aggregatorEndpoint)).Methods(http.MethodPut, http.MethodOptions)
	router.HandleFunc(apiPrefix+ClustersForOrganizationEndpoint, server.proxyTo(aggregatorEndpoint)).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+DisableRuleForClusterEndpoint, server.proxyTo(aggregatorEndpoint)).Methods(http.MethodPut, http.MethodOptions)
	router.HandleFunc(apiPrefix+EnableRuleForClusterEndpoint, server.proxyTo(aggregatorEndpoint)).Methods(http.MethodPut, http.MethodOptions)
	router.HandleFunc(apiPrefix+RuleGroupsEndpoint, server.getGroups).Methods(http.MethodGet, http.MethodOptions)
	router.HandleFunc(apiPrefix+RuleContent, server.getContentForRule).Methods(http.MethodGet)

	// Prometheus metrics
	router.Handle(apiPrefix+MetricsEndpoint, promhttp.Handler()).Methods(http.MethodGet)

	// OpenAPI specs
	router.HandleFunc(openAPIURL, server.serveAPISpecFile).Methods(http.MethodGet)
}
