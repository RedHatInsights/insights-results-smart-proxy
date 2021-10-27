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

import (
	"net/http"
	"path/filepath"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// ReportEndpointV2 https://issues.redhat.com/browse/CCXDEV-5097
	ReportEndpointV2 = "cluster/{cluster}/reports"

	// ClustersDetail https://issues.redhat.com/browse/CCXDEV-5088
	ClustersDetail = "rule/{rule_selector}/clusters_detail/"

	// RecommendationsListEndpoint lists all recommendations with a number of impacted clusters.
	RecommendationsListEndpoint = "rule/"

	// RuleContentV2 https://issues.redhat.com/browse/CCXDEV-5094
	// additionally group info is added too
	// https://github.com/RedHatInsights/insights-results-smart-proxy/pull/604
	RuleContentV2 = "rule/{rule_id}/content"

	// RuleContentWithUserData returns same as RuleContentV2, but includes user-specific data
	RuleContentWithUserData = "rule/{rule_id}"

	// ContentV2 returns all the static content avaiable for the user
	ContentV2 = "content"

	// Endpoints to acknowledge rule and to manipulate with
	// acknowledgements.

	// AckListEndpoint list acks from this account where the rule is
	// active. Will return an empty list if this account has no acks.
	AckListEndpoint = "ack"

	// AckGetEndpoint read the acknowledgement info about disabled rule.
	// Acks are created, deleted, and queried by Insights rule ID, not
	// by their own ack ID.
	AckGetEndpoint = "ack/{rule_id}"

	// AckAcknowledgePostEndpoint acknowledges (and therefore hides) a rule
	// from view in an account. If there's already an acknowledgement of
	// this rule by this account, then return that. Otherwise, a new ack is
	// created.
	AckAcknowledgePostEndpoint = "ack"

	// AckUpdateEndpoint updates an acknowledgement for a rule, by rule ID.
	// A new justification can be supplied. The username is taken from the
	// authenticated request. The updated ack is returned.
	AckUpdateEndpoint = "ack/{rule_id}"

	// AckDeleteEndpoint deletes an acknowledgement for a rule, by its rule
	// ID. If the ack existed, it is deleted and a 204 is returned.
	// Otherwise, a 404 is returned.
	AckDeleteEndpoint = "ack/{rule_id}"
	// Rating endpoint will get/modify the vote for a rule id by the user
	Rating = "rating"
)

// addV2EndpointsToRouter adds API V2 specific endpoints to the router
func (server *HTTPServer) addV2EndpointsToRouter(router *mux.Router) {
	apiV2Prefix := server.Config.APIv2Prefix
	openAPIv2URL := apiV2Prefix + filepath.Base(server.Config.APIv2SpecFile)
	aggregatorBaseEndpoint := server.ServicesConfig.AggregatorBaseEndpoint

	// Common REST API endpoints
	router.HandleFunc(apiV2Prefix+MainEndpoint, server.mainEndpoint).Methods(http.MethodGet)

	// Reports endpoints
	server.addV2ReportsEndpointsToRouter(router, apiV2Prefix, aggregatorBaseEndpoint)

	// Content related endpoints
	server.addV2ContentEndpointsToRouter(router, apiV2Prefix)

	// Rules related endpoints
	server.addV2RuleEndpointsToRouter(router, apiV2Prefix, aggregatorBaseEndpoint)

	// Prometheus metrics
	router.Handle(apiV2Prefix+MetricsEndpoint, promhttp.Handler()).Methods(http.MethodGet)

	// OpenAPI specs
	router.HandleFunc(
		openAPIv2URL,
		httputils.CreateOpenAPIHandler(server.Config.APIv2SpecFile, server.Config.Debug, true),
	).Methods(http.MethodGet)
}

// addV2ReportsEndpointsToRouter method registers handlers for endpoints that
// return cluster report or reports to client
func (server *HTTPServer) addV2ReportsEndpointsToRouter(router *mux.Router, apiPrefix, aggregatorBaseURL string) {
	router.HandleFunc(apiPrefix+ReportEndpointV2, server.reportEndpoint).Methods(http.MethodGet, http.MethodOptions)

	router.HandleFunc(apiPrefix+RecommendationsListEndpoint, server.getRecommendations).Methods(http.MethodGet)
}

// addV2RuleEndpointsToRouter method registers handlers for endpoints that handle
// rule-related operations (voting etc.)
func (server *HTTPServer) addV2RuleEndpointsToRouter(router *mux.Router, apiPrefix, aggregatorBaseEndpoint string) {
	// Acknowledgement-related endpoints. Please look into acks_handlers.go
	// and acks_utils.go for more information about these endpoints
	// prepared to be compatible with RHEL Insights Advisor.
	router.HandleFunc(apiPrefix+AckListEndpoint, server.readAckList).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+AckGetEndpoint, server.getAcknowledge).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+AckAcknowledgePostEndpoint, server.acknowledgePost).Methods(http.MethodPost)
	router.HandleFunc(apiPrefix+AckUpdateEndpoint, server.updateAcknowledge).Methods(http.MethodPut)
	router.HandleFunc(apiPrefix+AckDeleteEndpoint, server.deleteAcknowledge).Methods(http.MethodDelete)
	router.HandleFunc(apiPrefix+Rating, server.postRating).Methods(http.MethodPost)
	// Clusters for given recommendation endpoint
	router.HandleFunc(apiPrefix+ClustersDetail, server.getClustersDetailForRule).Methods(http.MethodGet)
}

// addV2ContentEndpointsToRouter method registers handlers for endpoints that
// returns content to clients
func (server HTTPServer) addV2ContentEndpointsToRouter(router *mux.Router, apiPrefix string) {
	router.HandleFunc(apiPrefix+RuleContentV2, server.getRecommendationContent).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+RuleContentWithUserData, server.getRecommendationContentWithUserData).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+ContentV2, server.getContentWithGroups).Methods(http.MethodGet)
}
