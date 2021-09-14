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
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/gorilla/mux"
	"net/http"
)

const (
	// DbgOrganizationsEndpoint returns all organizations. DEBUG only
	DbgOrganizationsEndpoint = ira_server.OrganizationsEndpoint
	// DbgDeleteOrganizationsEndpoint deletes all {organizations}(comma separated array). DEBUG only
	DbgDeleteOrganizationsEndpoint = ira_server.DeleteOrganizationsEndpoint
	// DbgDeleteClustersEndpoint deletes all {clusters}(comma separated array). DEBUG only
	DbgDeleteClustersEndpoint = ira_server.DeleteClustersEndpoint
	// DbgGetVoteOnRuleEndpoint is an endpoint to get vote on rule. DEBUG only
	DbgGetVoteOnRuleEndpoint = "clusters/{cluster}/rules/{rule_id}/error_key/{error_key}/get_vote"
)

// adddbgEndpointsToRouter adds API dbg specific endpoints to the router
func (server *HTTPServer) adddbgEndpointsToRouter(router *mux.Router) {
	apiPrefix := server.Config.APIdbgPrefix
	aggregatorBaseEndpoint := server.ServicesConfig.AggregatorBaseEndpoint

	router.HandleFunc(apiPrefix+DbgOrganizationsEndpoint, server.proxyTo(aggregatorBaseEndpoint, nil)).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+DbgDeleteOrganizationsEndpoint, server.proxyTo(aggregatorBaseEndpoint, nil)).Methods(http.MethodDelete)
	router.HandleFunc(apiPrefix+DbgDeleteClustersEndpoint, server.proxyTo(aggregatorBaseEndpoint, nil)).Methods(http.MethodDelete)

	router.HandleFunc(apiPrefix+DbgGetVoteOnRuleEndpoint, server.proxyTo(
		aggregatorBaseEndpoint,
		&ProxyOptions{RequestModifiers: []RequestModifier{
			server.newExtractUserIDFromTokenToURLRequestModifier(ira_server.GetVoteOnRuleEndpoint),
		}},
	)).Methods(http.MethodGet)
}
