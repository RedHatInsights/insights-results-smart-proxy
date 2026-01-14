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

	// ClusterInfoEndpoint provides information about given cluster retrieved from AMS API
	ClusterInfoEndpoint = "cluster/{cluster}/info"

	// UpgradeRisksPredictionEndpoint returns the prediction about upgrading
	// the given cluster.
	UpgradeRisksPredictionEndpoint = "cluster/{cluster}/upgrade-risks-prediction"
	// UpgradeRisksPredictionMultiClusterEndpoint returns the prediction about upgrading
	// a set of clusters
	UpgradeRisksPredictionMultiClusterEndpoint = "upgrade-risks-prediction" // #nosec G101

	// ClustersDetail https://issues.redhat.com/browse/CCXDEV-5088
	ClustersDetail = "rule/{rule_selector}/clusters_detail"

	// RecommendationsListEndpoint lists all recommendations with a number of impacted clusters.
	RecommendationsListEndpoint = "rule"

	// ClustersRecommendationsEndpoint returns a list of all clusters, number of impacting rules and number of rules by total risk
	ClustersRecommendationsEndpoint = "clusters"

	// RuleContentV2 https://issues.redhat.com/browse/CCXDEV-5094
	// additionally group info is added too
	// https://github.com/RedHatInsights/insights-results-smart-proxy/pull/604
	RuleContentV2 = "rule/{rule_id}/content"

	// RuleContentWithUserData returns same as RuleContentV2, but includes user-specific data
	RuleContentWithUserData = "rule/{rule_id}"

	// ContentV2 returns all the static content available for the user
	ContentV2 = "content"

	// Endpoints to manipulate with simplified rule results stored
	// independently under "tracker_id" identifier in Redis

	// ListAllRequestIDs should return list of all request IDs detected for
	// given cluster. In reality the list is refreshing as old request IDs
	// are forgotten after 24 hours
	ListAllRequestIDs = "cluster/{cluster}/requests"

	// StatusOfRequestID should return status of processing one given
	// request ID
	StatusOfRequestID = "cluster/{cluster}/request/{request_id}/status"

	// RuleHitsForRequestID should return simplified results for given
	// cluster and requestID
	RuleHitsForRequestID = "cluster/{cluster}/request/{request_id}/report"

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

	// DVONamespaceForClusterEndpoint Returns the list of all namespaces (i.e. array of objects) to
	// which this particular account has access filtered by {cluster_name}.
	// Each object contains the namespace ID, the namespace display name if
	// available, the cluster ID under which this namespace is created
	// (repeated input), and the number of affecting recommendations for
	// this namespace as well.
	//
	// BDD scenarios for this endpoint:
	// https://github.com/RedHatInsights/insights-behavioral-spec/blob/main/features/DVO_Recommendations/Smart_Proxy_REST_API.feature
	DVONamespaceForClusterEndpoint = "namespaces/dvo/{namespace}/cluster/{cluster}"

	// DVONamespaceListEndpoint returns a list of all DVO namespaces to
	// which an account has access. Each entry contains the
	// namespace ID, the namespace display name (if available), the cluster
	// ID under which this namespace was created, and the number of
	// recommendations affecting this namespace.
	//
	// BDD scenarios for this endpoint:
	// https://github.com/RedHatInsights/insights-behavioral-spec/blob/main/features/DVO_Recommendations/Smart_Proxy_REST_API.feature
	DVONamespaceListEndpoint = "namespaces/dvo"
)

// addV2EndpointsToRouter adds API V2 specific endpoints to the router
func (server *HTTPServer) addV2EndpointsToRouter(router *mux.Router) {
	apiV2Prefix := server.Config.APIv2Prefix
	openAPIv2URL := apiV2Prefix + filepath.Base(server.Config.APIv2SpecFile)

	// Common REST API endpoints
	router.HandleFunc(apiV2Prefix+MainEndpoint, server.mainEndpoint).Methods(http.MethodGet)
	router.HandleFunc(apiV2Prefix+RuleGroupsEndpoint, server.getGroups).Methods(http.MethodGet, http.MethodOptions)

	// Reports endpoints
	server.addV2ReportsEndpointsToRouter(router, apiV2Prefix)

	// Content related endpoints
	server.addV2ContentEndpointsToRouter(router, apiV2Prefix)

	// Rules related endpoints
	server.addV2RuleEndpointsToRouter(router, apiV2Prefix)

	// Endpoints requiring Redis to work
	server.addV2RedisEndpointsToRouter(router, apiV2Prefix)

	// Endpoints related to DVO workload recommendations
	server.addV2DVOEndpointsToRouter(router, apiV2Prefix)

	// Prometheus metrics
	router.Handle(apiV2Prefix+MetricsEndpoint, promhttp.Handler()).Methods(http.MethodGet)

	router.HandleFunc(apiV2Prefix+InfoEndpoint, server.infoMap).Methods(http.MethodGet, http.MethodOptions)
	router.HandleFunc(apiV2Prefix+UpgradeRisksPredictionEndpoint, server.upgradeRisksPrediction).Methods(http.MethodGet)
	router.HandleFunc(apiV2Prefix+UpgradeRisksPredictionMultiClusterEndpoint, server.upgradeRisksPredictionMultiCluster).Methods(http.MethodPost)

	// OpenAPI specs
	router.HandleFunc(
		openAPIv2URL,
		httputils.CreateOpenAPIHandler(server.Config.APIv2SpecFile, server.Config.Debug, true),
	).Methods(http.MethodGet)
}

// addV2RedisEndpointsToRouter method registers handlers for endpoints that depend on our Redis storage
// to provide responses.
func (server *HTTPServer) addV2RedisEndpointsToRouter(router *mux.Router, apiPrefix string) {
	router.HandleFunc(apiPrefix+ListAllRequestIDs, server.getRequestsForCluster).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+ListAllRequestIDs, server.getRequestsForClusterPostVariant).Methods(http.MethodPost)
	router.HandleFunc(apiPrefix+StatusOfRequestID, server.getRequestStatusForCluster).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+RuleHitsForRequestID, server.getReportForRequest).Methods(http.MethodGet)
}

// addV2DVOEndpointsToRouter method registers handlers for endpoints related to DVO workloads
func (server *HTTPServer) addV2DVOEndpointsToRouter(router *mux.Router, apiPrefix string) {
	router.HandleFunc(apiPrefix+DVONamespaceForClusterEndpoint, server.getDVONamespacesForCluster).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+DVONamespaceListEndpoint, server.getDVONamespaceList).Methods(http.MethodGet)
}

// addV2ReportsEndpointsToRouter method registers handlers for endpoints that
// return cluster report or reports to client
func (server *HTTPServer) addV2ReportsEndpointsToRouter(router *mux.Router, apiPrefix string) {
	router.HandleFunc(apiPrefix+ReportEndpointV2, server.reportEndpointV2).Methods(http.MethodGet, http.MethodOptions)
	router.HandleFunc(apiPrefix+ClusterInfoEndpoint, server.getSingleClusterInfo).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+RecommendationsListEndpoint, server.getRecommendations).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+ClustersRecommendationsEndpoint, server.getClustersView).Methods(http.MethodGet)
}

// addV2RuleEndpointsToRouter method registers handlers for endpoints that handle
// rule-related operations (voting etc.)
func (server *HTTPServer) addV2RuleEndpointsToRouter(router *mux.Router, apiPrefix string) {
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
