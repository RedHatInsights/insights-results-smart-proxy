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

// Package server contains implementation of REST API server (HTTPServer) for the
// Insights results smart proxy service. In current version, the following
//
// Please note that API_PREFIX is part of server configuration (see Configuration). Also please note that
// JSON format is used to transfer data between server and clients.
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
	"time"
	// we just have to import this package in order to expose pprof interface in debug mode
	// disable "G108 (CWE-): Profiling endpoint is automatically exposed on /debug/pprof"
	// #nosec G108
	_ "net/http/pprof"
	"path/filepath"

	"github.com/RedHatInsights/insights-content-service/groups"
	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-operator-utils/types"
	mapset "github.com/deckarep/golang-set"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"

	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"

	proxy_types "github.com/RedHatInsights/insights-results-smart-proxy/types"
)

// HTTPServer in an implementation of Server interface
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

// New constructs new implementation of Server interface
func New(config Configuration, servicesConfig services.Configuration, groupsChannel chan []groups.Group) *HTTPServer {
	return &HTTPServer{
		Config:         config,
		ServicesConfig: servicesConfig,
		GroupsChannel:  groupsChannel,
	}
}

func (server *HTTPServer) mainEndpoint(writer http.ResponseWriter, _ *http.Request) {
	err := responses.SendOK(writer, responses.BuildOkResponse())
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// readUserID tries to retrieve user ID from request. If any error occurs, error response is send back to client.
func (server *HTTPServer) readUserID(request *http.Request, writer http.ResponseWriter) (types.UserID, error) {
	userID, err := server.GetCurrentUserID(request)
	if err != nil {
		const message = "Unable to get user id"
		log.Error().Err(err).Msg(message)
		handleServerError(writer, err)
		return "", err
	}

	return userID, nil
}

// serveAPISpecFile serves an OpenAPI specifications file specified in config file
func (server HTTPServer) serveAPISpecFile(writer http.ResponseWriter, request *http.Request) {
	absPath, err := filepath.Abs(server.Config.APISpecFile)
	if err != nil {
		const message = "Error creating absolute path of OpenAPI spec file"
		log.Error().Err(err).Msg(message)
		handleServerError(writer, err)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	http.ServeFile(writer, request, absPath)
}

// Initialize perform the server initialization
func (server *HTTPServer) Initialize() http.Handler {
	log.Info().Msgf("Initializing HTTP server at '%s'", server.Config.Address)

	router := mux.NewRouter().StrictSlash(true)
	router.Use(httputils.LogRequest)

	apiPrefix := server.Config.APIPrefix

	metricsURL := apiPrefix + MetricsEndpoint
	openAPIURL := apiPrefix + filepath.Base(server.Config.APISpecFile)

	// enable authentication, but only if it is setup in configuration
	if server.Config.Auth {
		// we have to enable authentication for all endpoints, including endpoints
		// for Prometheus metrics and OpenAPI specification, because there is not
		// single prefix of other REST API calls. The special endpoints needs to
		// be handled in middleware which is not optimal
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

// Start starts server
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

// Stop stops server's execution
func (server *HTTPServer) Stop(ctx context.Context) error {
	return server.Serv.Shutdown(ctx)
}

// redirectTo
func (server HTTPServer) redirectTo(baseURL string) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		endpointURL, err := server.composeEndpoint(baseURL, request.RequestURI)

		if err != nil {
			log.Error().Err(err).Msg("Error during endpoint URL parsing")
			handleServerError(writer, err)
		}

		// test service available
		_, err = http.Get(endpointURL.String())
		if err != nil {
			log.Error().Err(err).Msg("Aggregator service unavailable")

			if _, ok := err.(*url.Error); ok {
				err = &AggregatorServiceUnavailableError{}
			}

			handleServerError(writer, err)
		}

		log.Info().Msgf("Redirecting to %s", endpointURL.String())
		http.Redirect(writer, request, endpointURL.String(), 302)
	}
}

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

func (server HTTPServer) reportEndpoint(writer http.ResponseWriter, request *http.Request) {
	aggregatorResponse, successful := server.fetchAggregatorReport(writer, request)
	if !successful {
		return
	}

	var rules []proxy_types.RuleWithContentResponse

	for _, aggregatorRule := range aggregatorResponse.Report {
		ruleID := aggregatorRule.Module
		errorKey := aggregatorRule.ErrorKey

		ruleWithContent, err := content.GetRuleWithErrorKeyContent(ruleID, errorKey)
		if err != nil {
			handleServerError(writer, err)
			return
		}

		rule := proxy_types.RuleWithContentResponse{
			CreatedAt:    ruleWithContent.PublishDate.UTC().Format(time.RFC3339),
			Description:  ruleWithContent.Description,
			ErrorKey:     errorKey,
			Generic:      ruleWithContent.Generic,
			Reason:       ruleWithContent.Reason,
			Resolution:   ruleWithContent.Resolution,
			TotalRisk:    ruleWithContent.TotalRisk,
			RiskOfChange: ruleWithContent.RiskOfChange,
			RuleID:       ruleID,
			TemplateData: aggregatorRule.TemplateData,
			Tags:         ruleWithContent.Tags,
			UserVote:     aggregatorRule.UserVote,
			Disabled:     aggregatorRule.Disabled,
		}

		rules = append(rules, rule)
	}

	report := proxy_types.SmartProxyReport{
		Meta: aggregatorResponse.Meta,
		Data: rules,
	}

	err := responses.SendOK(writer, responses.BuildOkResponseWithData("report", report))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

func (server HTTPServer) findRule(
	writer http.ResponseWriter, report []types.RuleOnReport, requestRuleID types.RuleID,
) (proxy_types.RuleWithContentResponse, bool) {
	var rule proxy_types.RuleWithContentResponse
	found := false

	for _, aggregatorRule := range report {
		ruleID := aggregatorRule.Module
		if ruleID == requestRuleID {
			errorKey := aggregatorRule.ErrorKey

			ruleWithContent, err := content.GetRuleWithErrorKeyContent(ruleID, errorKey)
			if err != nil {
				handleServerError(writer, err)
				return rule, false
			}

			rule = proxy_types.RuleWithContentResponse{
				CreatedAt:    ruleWithContent.PublishDate.UTC().Format(time.RFC3339),
				Description:  ruleWithContent.Description,
				ErrorKey:     errorKey,
				Generic:      ruleWithContent.Generic,
				Reason:       ruleWithContent.Reason,
				Resolution:   ruleWithContent.Resolution,
				TotalRisk:    ruleWithContent.TotalRisk,
				RiskOfChange: ruleWithContent.RiskOfChange,
				RuleID:       ruleID,
				TemplateData: aggregatorRule.TemplateData,
				Tags:         ruleWithContent.Tags,
				UserVote:     aggregatorRule.UserVote,
				Disabled:     aggregatorRule.Disabled,
				Internal:     ruleWithContent.Internal,
			}
			found = true
			break
		}
	}

	if !found {
		handleServerError(writer, &types.ItemNotFoundError{
			ItemID: fmt.Sprintf("%v", requestRuleID),
		})
		return rule, false
	}

	return rule, true
}

func (server HTTPServer) singleRuleEndpoint(writer http.ResponseWriter, request *http.Request) {
	ruleID, err := readRuleID(writer, request)
	if err != nil {
		return
	}

	aggregatorResponse, successful := server.fetchAggregatorReport(writer, request)
	// Error message handled by function
	if !successful {
		return
	}

	rule, successful := server.findRule(writer, aggregatorResponse.Report, ruleID)
	// Error message handled by function
	if !successful {
		return
	}

	if rule.Internal {
		err := server.checkInternalRulePermissions(request)
		if err != nil {
			handleServerError(writer, err)
			return
		}
	}

	err = responses.SendOK(writer, responses.BuildOkResponseWithData("report", rule))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// checkInternalRulePermissions checks if organizations for internal rules are enabled
// if so, retrieves the org_id from request/token and returns whether that ID is on the list
// of allowed organizations to access internal rules
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
		return nil, nil
	}

	totalRisks := mapset.NewSet()
	tags := mapset.NewSet()

	for _, rule := range aggregatorResponse.Report {
		ruleID := rule.Module
		errorKey := rule.ErrorKey
		ruleWithContent, err := content.GetRuleWithErrorKeyContent(ruleID, errorKey)
		if err != nil {
			return nil, err
		}
		totalRisks.Add(ruleWithContent.TotalRisk)

		for _, tag := range ruleWithContent.Tags {
			tags.Add(tag)
		}
	}

	return &proxy_types.ClusterOverview{
		TotalRisksHit: totalRisks,
		TagsHit:       tags,
	}, nil
}
