/*
Copyright © 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package server contains implementation of REST API server (HTTPServer) for
// the Insights results smart proxy service. In current version, the following
//
// Please note that API_PREFIX is part of server configuration (see
// Configuration). Also please note that JSON format is used to transfer data
// between server and clients.
//
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	// we just have to import this package in order to expose pprof
	// interface in debug mode
	// disable "G108 (CWE-): Profiling endpoint is automatically exposed on /debug/pprof"
	// #nosec G108
	_ "net/http/pprof"
	"path/filepath"

	"github.com/RedHatInsights/insights-content-service/groups"
	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"

	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"

	proxy_types "github.com/RedHatInsights/insights-results-smart-proxy/types"
)

// HTTPServer is an implementation of Server interface
type HTTPServer struct {
	Config         Configuration
	ServicesConfig services.Configuration
	GroupsChannel  chan []groups.Group
	Serv           *http.Server
}

// RequestModifier is a type of function which modifies request when proxying
type RequestModifier func(request *http.Request) (*http.Request, error)

// ResponseModifier is a type of function which modifies response when proxying
type ResponseModifier func(response *http.Response) (*http.Response, error)

// ProxyOptions alters behaviour of proxy server for each endpoint.
// For example, you can set custom request and response modifiers
type ProxyOptions struct {
	RequestModifiers  []RequestModifier
	ResponseModifiers []ResponseModifier
}

// New function constructs new implementation of Server interface.
func New(config Configuration, servicesConfig services.Configuration, groupsChannel chan []groups.Group) *HTTPServer {
	return &HTTPServer{
		Config:         config,
		ServicesConfig: servicesConfig,
		GroupsChannel:  groupsChannel,
	}
}

// mainEndpoint method handles requests to the main endpoint.
func (server *HTTPServer) mainEndpoint(writer http.ResponseWriter, _ *http.Request) {
	err := responses.SendOK(writer, responses.BuildOkResponse())
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// Initialize method performs the server initialization, including
// registration of all handlers.
func (server *HTTPServer) Initialize() http.Handler {
	log.Info().Msgf("Initializing HTTP server at '%s'", server.Config.Address)

	router := mux.NewRouter().StrictSlash(true)
	router.Use(httputils.LogRequest)

	apiPrefix := server.Config.APIPrefix

	metricsURL := apiPrefix + MetricsEndpoint
	openAPIURL := apiPrefix + filepath.Base(server.Config.APISpecFile)

	// enable authentication, but only if it is setup in configuration
	if server.Config.Auth {
		// we have to enable authentication for all endpoints,
		// including endpoints for Prometheus metrics and OpenAPI
		// specification, because there is not single prefix of other
		// REST API calls. The special endpoints needs to be handled in
		// middleware which is not optimal
		noAuthURLs := []string{
			metricsURL,
			openAPIURL,
			metricsURL + "?", // to be able to test using Frisby
			openAPIURL + "?", // to be able to test using Frisby
		}
		router.Use(func(next http.Handler) http.Handler { return server.Authentication(next, noAuthURLs) })
	}

	if server.Config.EnableCORS {
		headersOK := handlers.AllowedHeaders([]string{
			"Content-Type",
			"Content-Length",
			"Accept-Encoding",
			"X-CSRF-Token",
			"Authorization",
		})
		originsOK := handlers.AllowedOrigins([]string{"*"})
		methodsOK := handlers.AllowedMethods([]string{
			http.MethodPost,
			http.MethodGet,
			http.MethodOptions,
			http.MethodPut,
			http.MethodDelete,
		})
		credsOK := handlers.AllowCredentials()
		corsMiddleware := handlers.CORS(originsOK, headersOK, methodsOK, credsOK)
		router.Use(corsMiddleware)
	}

	server.addEndpointsToRouter(router)

	return router
}

// Start method starts HTTP or HTTPS server.
func (server *HTTPServer) Start() error {
	address := server.Config.Address
	log.Info().Msgf("Starting HTTP server at '%s'", address)
	router := server.Initialize()
	server.Serv = &http.Server{Addr: address, Handler: router}
	var err error

	if server.Config.UseHTTPS {
		err = server.Serv.ListenAndServeTLS("server.crt", "server.key")
	} else {
		err = server.Serv.ListenAndServe()
	}
	if err != nil && err != http.ErrServerClosed {
		log.Error().Err(err).Msg("Unable to start HTTP/S server")
		return err
	}

	return nil
}

// Stop method stops server's execution.
func (server *HTTPServer) Stop(ctx context.Context) error {
	return server.Serv.Shutdown(ctx)
}

// modifyRequest function modifies HTTP request during proxying it to another
// service.
// TODO: move to utils?
func modifyRequest(requestModifiers []RequestModifier, request *http.Request) (*http.Request, error) {
	for _, modifier := range requestModifiers {
		var err error
		request, err = modifier(request)
		if err != nil {
			return nil, err
		}
	}

	return request, nil
}

// modifyResponse function modifies HTTP response returned by another service
// during proxying.
// TODO: move to utils?
func modifyResponse(responseModifiers []ResponseModifier, response *http.Response) (*http.Response, error) {
	for _, modifier := range responseModifiers {
		var err error
		response, err = modifier(response)
		if err != nil {
			return nil, err
		}
	}

	return response, nil
}

// proxyTo method constructs proxy function to proxy request to another
// service.
func (server HTTPServer) proxyTo(baseURL string, options *ProxyOptions) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		if options != nil {
			var err error
			request, err = modifyRequest(options.RequestModifiers, request)
			if err != nil {
				handleServerError(writer, err)
				return
			}
		}

		log.Info().Msg("Handling response as a proxy")

		endpointURL, err := server.composeEndpoint(baseURL, request.RequestURI)
		if err != nil {
			log.Error().Err(err).Msgf("Error during endpoint %s URL parsing", request.RequestURI)
			handleServerError(writer, err)
			return
		}

		client := http.Client{}
		req, err := http.NewRequest(request.Method, endpointURL.String(), request.Body)
		if err != nil {
			panic(err)
		}

		copyHeader(request.Header, req.Header)

		response, body, successful := server.sendRequest(client, req, options, writer)
		if !successful {
			return
		}

		// Maybe this code should be on responses.SendRaw or something like that
		err = responses.Send(response.StatusCode, writer, body)
		if err != nil {
			log.Error().Err(err).Msgf("Error writing the response")
			handleServerError(writer, err)
			return
		}
	}
}

func (server HTTPServer) sendRequest(
	client http.Client, req *http.Request, options *ProxyOptions, writer http.ResponseWriter,
) (*http.Response, []byte, bool) {
	log.Debug().Msgf("Connecting to %s", req.RequestURI)
	response, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Msgf("Error during retrieve of %s", req.RequestURI)
		handleServerError(writer, err)
		return nil, nil, false
	}

	if options != nil {
		var err error
		response, err = modifyResponse(options.ResponseModifiers, response)
		if err != nil {
			handleServerError(writer, err)
			return nil, nil, false
		}
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error().Err(err).Msgf("Error while retrieving content from request to %s", req.RequestURI)
		handleServerError(writer, err)
		return nil, nil, false
	}

	return response, body, true
}

func (server HTTPServer) composeEndpoint(baseEndpoint string, currentEndpoint string) (*url.URL, error) {
	endpoint := strings.TrimPrefix(currentEndpoint, server.Config.APIPrefix)
	return url.Parse(baseEndpoint + endpoint)
}

func copyHeader(srcHeaders http.Header, dstHeaders http.Header) {
	for headerKey, headerValues := range srcHeaders {
		for _, value := range headerValues {
			dstHeaders.Add(headerKey, value)
		}
	}
}

// readClusterIDsForOrgID reads the list of clusters for a given
// organization from aggregator
func (server HTTPServer) readClusterIDsForOrgID(orgID types.OrgID) ([]types.ClusterName, error) {
	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.ClustersForOrganizationEndpoint,
		orgID,
	)

	// #nosec G107
	response, err := http.Get(aggregatorURL)
	if err != nil {
		return nil, err
	}

	var recvMsg struct {
		Status   string              `json:"status"`
		Clusters []types.ClusterName `json:"clusters"`
	}

	err = json.NewDecoder(response.Body).Decode(&recvMsg)
	return recvMsg.Clusters, err
}

// readAggregatorReportForClusterID reads report from aggregator,
// handles errors by sending corresponding message to the user.
// Returns report and bool value set to true if there was no errors
func (server HTTPServer) readAggregatorReportForClusterID(
	orgID types.OrgID, clusterID types.ClusterName, userID types.UserID, writer http.ResponseWriter,
) (*types.ReportResponse, bool) {
	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.ReportEndpoint,
		orgID,
		clusterID,
		userID,
	)

	// #nosec G107
	aggregatorResp, err := http.Get(aggregatorURL)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

	var aggregatorResponse struct {
		Report *types.ReportResponse `json:"report"`
		Status string                `json:"status"`
	}

	responseBytes, err := ioutil.ReadAll(aggregatorResp.Body)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

	if aggregatorResp.StatusCode != http.StatusOK {
		err := responses.Send(aggregatorResp.StatusCode, writer, responseBytes)
		if err != nil {
			log.Error().Err(err).Msg(responseDataError)
		}
		return nil, false
	}

	err = json.Unmarshal(responseBytes, &aggregatorResponse)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

	return aggregatorResponse.Report, true
}

func (server HTTPServer) readAggregatorReportForClusterList(
	orgID types.OrgID, clusterList []string, writer http.ResponseWriter,
) (*types.ClusterReports, bool) {
	clist := strings.Join(clusterList, ",")
	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.ReportForListOfClustersEndpoint,
		orgID,
		clist)

	// #nosec G107
	aggregatorResp, err := http.Get(aggregatorURL)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

	var aggregatorResponse types.ClusterReports

	responseBytes, err := ioutil.ReadAll(aggregatorResp.Body)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

	if aggregatorResp.StatusCode != http.StatusOK {
		err := responses.Send(aggregatorResp.StatusCode, writer, responseBytes)
		if err != nil {
			log.Error().Err(err).Msg(responseDataError)
		}
		return nil, false
	}

	err = json.Unmarshal(responseBytes, &aggregatorResponse)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

	return &aggregatorResponse, true
}

// readAggregatorRuleForClusterID reads report from aggregator,
// handles errors by sending corresponding message to the user.
// Returns report and bool value set to true if there was no errors
func (server HTTPServer) readAggregatorRuleForClusterID(
	orgID types.OrgID, clusterID types.ClusterName, userID types.UserID, ruleID types.RuleID, errorKey types.ErrorKey, writer http.ResponseWriter,
) (*types.RuleOnReport, bool) {
	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.RuleEndpoint,
		orgID,
		clusterID,
		userID,
		fmt.Sprintf("%v|%v", ruleID, errorKey),
	)

	// #nosec G107
	aggregatorResp, err := http.Get(aggregatorURL)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

	var aggregatorResponse struct {
		Report *types.RuleOnReport `json:"report"`
		Status string              `json:"status"`
	}

	responseBytes, err := ioutil.ReadAll(aggregatorResp.Body)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

	if aggregatorResp.StatusCode != http.StatusOK {
		err := responses.Send(aggregatorResp.StatusCode, writer, responseBytes)
		if err != nil {
			log.Error().Err(err).Msg(responseDataError)
		}
		return nil, false
	}

	err = json.Unmarshal(responseBytes, &aggregatorResponse)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

	return aggregatorResponse.Report, true
}

func (server HTTPServer) fetchAggregatorReport(
	writer http.ResponseWriter, request *http.Request,
) (*types.ReportResponse, bool) {
	clusterID, successful := httputils.ReadClusterName(writer, request)
	// Error message handled by function
	if !successful {
		return nil, false
	}

	authToken, err := server.GetAuthToken(request)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

	userID := authToken.AccountNumber
	orgID := authToken.Internal.OrgID

	aggregatorResponse, successful := server.readAggregatorReportForClusterID(orgID, clusterID, userID, writer)
	if !successful {
		return nil, false
	}
	return aggregatorResponse, true
}

// fetchAggregatorReports method access the Insights Results Aggregator to read
// reports for given list of clusters. Then the response structure is
// constructed from data returned by Aggregator.
func (server HTTPServer) fetchAggregatorReports(
	writer http.ResponseWriter, request *http.Request,
) (*types.ClusterReports, bool) {
	// cluster list is specified in path (part of URL)
	clusterList, successful := httputils.ReadClusterListFromPath(writer, request)
	// Error message handled by function
	if !successful {
		return nil, false
	}

	authToken, err := server.GetAuthToken(request)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

	orgID := authToken.Internal.OrgID

	aggregatorResponse, successful := server.readAggregatorReportForClusterList(orgID, clusterList, writer)
	if !successful {
		return nil, false
	}
	return aggregatorResponse, true
}

// fetchAggregatorReportsUsingRequestBodyClusterList method access the Insights
// Results Aggregator to read reports for given list of clusters. Then the
// response structure is constructed from data returned by Aggregator.
func (server HTTPServer) fetchAggregatorReportsUsingRequestBodyClusterList(
	writer http.ResponseWriter, request *http.Request,
) (*types.ClusterReports, bool) {
	// cluster list is specified in request body
	clusterList, successful := httputils.ReadClusterListFromBody(writer, request)
	// Error message handled by function
	if !successful {
		return nil, false
	}

	authToken, err := server.GetAuthToken(request)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

	orgID := authToken.Internal.OrgID

	aggregatorResponse, successful := server.readAggregatorReportForClusterList(orgID, clusterList, writer)
	if !successful {
		return nil, false
	}
	return aggregatorResponse, true
}

func (server HTTPServer) reportEndpoint(writer http.ResponseWriter, request *http.Request) {
	aggregatorResponse, successful := server.fetchAggregatorReport(writer, request)
	if !successful {
		return
	}

	includeDisabled, err := readGetDisabledParam(request)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	osdFlag, err := readOSDEligible(request)
	if err != nil {
		log.Err(err).Msgf("Got error while parsing `%s` value", OSDEligibleParam)
	}

	log.Info().Msgf("%s flag = %t", GetDisabledParam, includeDisabled)
	log.Info().Msgf("%s flag = %t", OSDEligibleParam, osdFlag)

	rules, disabledRules, rulesWithoutContent := filterRulesResponse(aggregatorResponse.Report, osdFlag, includeDisabled)

	report := proxy_types.SmartProxyReport{
		Meta: types.ReportResponseMeta{
			LastCheckedAt: aggregatorResponse.Meta.LastCheckedAt,
			Count:         len(rules) + rulesWithoutContent,
		},
		Data: rules,
	}

	status := http.StatusOK

	// This condition checks that the only rules for the cluster have missing content
	if rulesWithoutContent > 0 && len(rules) == 0 && disabledRules == 0 {
		status = http.StatusInternalServerError
	}

	err = responses.Send(status, writer, responses.BuildOkResponseWithData("report", report))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// reportForListOfClustersEndpoint is a handler that returns reports for
// several clusters that all need to belong to one organization specified in
// request path. List of clusters is specified in request path as well which
// means that clients needs to deal with URL limit (around 2000 characters).
func (server HTTPServer) reportForListOfClustersEndpoint(writer http.ResponseWriter, request *http.Request) {
	// try to read results from Insights Results Aggregator service
	aggregatorResponse, successful := server.fetchAggregatorReports(writer, request)
	if !successful {
		return
	}

	// send the response back to client
	err := responses.Send(http.StatusOK, writer, aggregatorResponse)
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// reportForListOfClustersPayloadEndpoint is a handler that returns reports for
// several clusters that all need to belong to one organization specified in
// request path. List of clusters is specified in request body which means that
// clients can use as many cluster ID as the wont without any (real) limits.
func (server HTTPServer) reportForListOfClustersPayloadEndpoint(writer http.ResponseWriter, request *http.Request) {
	// try to read results from Insights Results Aggregator service
	aggregatorResponse, successful := server.fetchAggregatorReportsUsingRequestBodyClusterList(writer, request)
	if !successful {
		return
	}

	// send the response back to client
	err := responses.Send(http.StatusOK, writer, aggregatorResponse)
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

func (server HTTPServer) fetchAggregatorReportRule(
	writer http.ResponseWriter, request *http.Request,
) (*types.RuleOnReport, bool) {
	clusterID, successful := httputils.ReadClusterName(writer, request)
	// Error message handled by function
	if !successful {
		return nil, false
	}

	ruleID, errorKey, err := readRuleIDWithErrorKey(writer, request)
	if err != nil {
		return nil, false
	}

	authToken, err := server.GetAuthToken(request)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

	userID := authToken.AccountNumber
	orgID := authToken.Internal.OrgID

	aggregatorResponse, successful := server.readAggregatorRuleForClusterID(orgID, clusterID, userID, ruleID, errorKey, writer)
	if !successful {
		return nil, false
	}
	return aggregatorResponse, true
}

func (server HTTPServer) singleRuleEndpoint(writer http.ResponseWriter, request *http.Request) {
	var rule *proxy_types.RuleWithContentResponse
	var err error

	aggregatorResponse, successful := server.fetchAggregatorReportRule(writer, request)
	// Error message handled by function
	if !successful {
		return
	}

	osdFlag, err := readOSDEligible(request)
	if err != nil {
		log.Err(err).Msgf("Got error while parsing `%s` value", OSDEligibleParam)
	}
	rule, successful, _ = content.FetchRuleContent(*aggregatorResponse, osdFlag)

	if !successful {
		err := responses.SendNotFound(writer, "Rule was not found")
		if err != nil {
			handleServerError(writer, err)
			return
		}
		return
	}

	if rule.Internal {
		err = server.checkInternalRulePermissions(request)
		if err != nil {
			handleServerError(writer, err)
			return
		}
	}

	err = responses.SendOK(writer, responses.BuildOkResponseWithData("report", *rule))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// checkInternalRulePermissions method checks if organizations for internal
// rules are enabled if so, retrieves the org_id from request/token and returns
// whether that ID is on the list of allowed organizations to access internal
// rules
func (server HTTPServer) checkInternalRulePermissions(request *http.Request) error {
	if !server.Config.EnableInternalRulesOrganizations || !server.Config.Auth {
		return nil
	}

	authToken, err := server.GetAuthToken(request)
	if err != nil {
		return err
	}

	requestOrgID := types.OrgID(authToken.Internal.OrgID)

	log.Info().Msgf("Checking internal rule permissions for Organization ID: %v", requestOrgID)
	for _, allowedID := range server.Config.InternalRulesOrganizations {
		if requestOrgID == allowedID {
			log.Info().Msgf("Organization %v is allowed access to internal rules", requestOrgID)
			return nil
		}
	}

	// If the loop ends without returning nil, then an authentication error should be raised
	const message = "This organization is not allowed to access this recommendation"
	log.Error().Msg(message)
	return &AuthenticationError{errString: message}
}

func (server HTTPServer) newExtractUserIDFromTokenToURLRequestModifier(newEndpoint string) RequestModifier {
	return func(request *http.Request) (*http.Request, error) {
		identity, err := server.GetAuthToken(request)
		if err != nil {
			return nil, err
		}

		vars := mux.Vars(request)
		vars["user_id"] = string(identity.AccountNumber)

		newURL := httputils.MakeURLToEndpointMapString(server.Config.APIPrefix, newEndpoint, vars)
		request.URL, err = url.Parse(newURL)
		if err != nil {
			return nil, &ParamsParsingError{}
		}

		request.RequestURI = request.URL.RequestURI()

		return request, nil
	}
}

func (server HTTPServer) getOverviewPerCluster(
	clusterName types.ClusterName,
	authToken *types.Identity,
	writer http.ResponseWriter) (*proxy_types.ClusterOverview, error) {

	userID := authToken.AccountNumber
	orgID := authToken.Internal.OrgID
	aggregatorResponse, successful := server.readAggregatorReportForClusterID(orgID, clusterName, userID, writer)
	if !successful {
		log.Info().Msgf("Aggregator doesn't have reports for cluster ID %s", clusterName)
		return nil, nil
	}

	if aggregatorResponse.Meta.Count == 0 {
		log.Info().Msgf("Cluster report doesn't have any hits. Skipping from overview.")
		return nil, nil
	}

	totalRisks := make([]int, 0)
	tags := make([]string, 0)

	for _, rule := range aggregatorResponse.Report {
		ruleID := rule.Module
		errorKey := rule.ErrorKey
		ruleWithContent, err := content.GetRuleWithErrorKeyContent(ruleID, errorKey)
		if err != nil {
			log.Error().Err(err).Msgf("Unable to retrieve content for rule %v|%v", ruleID, errorKey)
			// this rule is not visible in OCM UI either, so we can continue calculating to be consistent
			continue
		}

		totalRisks = append(totalRisks, ruleWithContent.TotalRisk)

		for _, tag := range ruleWithContent.Tags {
			tags = append(tags, tag)
		}
	}

	return &proxy_types.ClusterOverview{
		TotalRisksHit: totalRisks,
		TagsHit:       tags,
	}, nil
}

// filterRulesResponse returns an array of RuleWithContentResponse with only the rules that matches 3 criteria:
// - The rule has content from the content-service
// - The disabled filter is not match
// - The OSD elegible filter is not match
func filterRulesResponse(aggregatorReport []types.RuleOnReport, filterOSD, getDisabled bool) (
	filteredRules []proxy_types.RuleWithContentResponse,
	disabledRules int,
	noContentRules int,
) {
	log.Debug().Bool(GetDisabledParam, getDisabled).Bool(OSDEligibleParam, filterOSD).Msg("Filtering rules in report")
	filteredRules = []proxy_types.RuleWithContentResponse{}
	disabledRules = 0
	noContentRules = 0

	for _, aggregatorRule := range aggregatorReport {
		if aggregatorRule.Disabled && !getDisabled {
			disabledRules++
			continue
		}

		rule, successful, filtered := content.FetchRuleContent(aggregatorRule, filterOSD)
		if !successful {
			if !filtered {
				noContentRules++
			}
			continue
		}

		filteredRules = append(filteredRules, *rule)
	}

	return
}
