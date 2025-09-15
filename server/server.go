/*
Copyright © 2020, 2021, 2022, 2023  Red Hat, Inc.

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
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	// we just have to import this package in order to expose pprof
	// interface in debug mode
	// disable "G108 (CWE-): Profiling endpoint is automatically exposed on /debug/pprof"
	_ "net/http/pprof" // #nosec G108

	"github.com/RedHatInsights/insights-content-service/groups"
	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-operator-utils/responses"
	utypes "github.com/RedHatInsights/insights-operator-utils/types"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"
	ctypes "github.com/RedHatInsights/insights-results-types"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-smart-proxy/amsclient"
	"github.com/RedHatInsights/insights-results-smart-proxy/auth"
	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"

	"github.com/RedHatInsights/insights-results-smart-proxy/types"
)

const (
	// contentTypeHeader represents Content-Type header name
	contentTypeHeader = "Content-Type"

	// userAgentHeader is used to retrieve the User Agent set in the request for special cases
	userAgentHeader = "User-Agent"

	// insightsOperatorUserAgent is a product name set in the requests made by the Insights Operator
	// to be shown in the OCP Web console
	insightsOperatorUserAgent = "insights-operator"

	// acmUserAgent is a product name set in the requests to be shown in the Advanced Cluster Manager
	acmUserAgent = "acm-operator"

	// browserUserAgent is the standard product name set by web browsers (requests made via OCM, OCP Advisor, ..)
	browserUserAgent = "Mozilla"

	// openAPIGeneratorUserAgent is the product name set by OpenAPI-generated  in iqe tests clients
	openAPIGeneratorUserAgent = "OpenAPI-Generator"

	// pythonRequestsUserAgent is the product name set by Python requests library in iqe tests
	pythonRequestsUserAgent = "python-requests"

	// nonRelevantUserAgent is a test user agent used in iqe tests to verify unknown user agent handling
	nonRelevantUserAgent = "non-relevant-user-agent"

	// JSONContentType represents the application/json content type
	JSONContentType = "application/json; charset=utf-8"

	// orgIDTag represent the tags for printing orgID in the logs
	orgIDTag = "orgID"

	// userIDTag represent the tags for printing user ID (account number) in the logs
	userIDTag = "userID"

	// clusterIDTag is used for printing cluster IDs
	clusterIDTag = "clusterID"

	// dotReport ".report" string present in the ruleID in most tables
	dotReport = ".report"

	ackedRulesError      = "Unable to retrieve list of acked rules"
	compositeRuleIDError = "Error generating composite rule ID"
	clusterListError     = "problem reading cluster list for org"
	ruleContentError     = "unable to get content for rule"
)

// HTTPServer is an implementation of Server interface
type HTTPServer struct {
	Config            Configuration
	InfoParams        map[string]string
	ServicesConfig    services.Configuration
	amsClient         amsclient.AMSClient
	GroupsChannel     chan []groups.Group
	ErrorFoundChannel chan bool
	ErrorChannel      chan error
	Serv              *http.Server
	redis             services.RedisInterface
	rbacClient        auth.RBACClient
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
func New(config Configuration,
	servicesConfig services.Configuration,
	amsClient amsclient.AMSClient,
	redis services.RedisInterface,
	groupsChannel chan []groups.Group,
	errorFoundChannel chan bool,
	errorChannel chan error,
	rbacClient auth.RBACClient,
) *HTTPServer {
	return &HTTPServer{
		Config:            config,
		InfoParams:        make(map[string]string),
		ServicesConfig:    servicesConfig,
		amsClient:         amsClient,
		redis:             redis,
		GroupsChannel:     groupsChannel,
		ErrorFoundChannel: errorFoundChannel,
		ErrorChannel:      errorChannel,
		rbacClient:        rbacClient,
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

	// Add custom metrics middleware to capture user-agent information
	router.Use(MetricsMiddleware)

	// Set up authentication and authorization middleware
	server.setupAuthMiddleware(router)

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

func (server *HTTPServer) addEndpointsToRouter(router *mux.Router) {
	// It is possible to use special REST API endpoints in debug mode
	if server.Config.Debug {
		server.adddbgEndpointsToRouter(router)
	}
	server.addV1EndpointsToRouter(router)
	server.addV2EndpointsToRouter(router)
}

// Start method starts HTTP or HTTPS server.
func (server *HTTPServer) Start() error {
	address := server.Config.Address
	log.Info().Msgf("Starting HTTP server at '%s'", address)
	router := server.Initialize()
	server.Serv = &http.Server{
		Addr:              address,
		Handler:           router,
		ReadTimeout:       1 * time.Minute,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
	}
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
func (server *HTTPServer) proxyTo(baseURL string, options *ProxyOptions) func(http.ResponseWriter, *http.Request) {
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
			log.Error().Err(err).Str(urlStr, request.RequestURI).Msg("Error during endpoint %s URL parsing")
			handleServerError(writer, err)
			return
		}

		client := http.Client{}
		req, err := http.NewRequest(request.Method, endpointURL.String(), request.Body)
		if err != nil {
			panic(err)
		}

		copyHeader(request.Header, req.Header)

		response, body, err := sendRequest(client, req, options)
		if err != nil {
			server.evaluateProxyError(writer, err, baseURL)
			return
		}

		// Maybe this code should be on responses.SendRaw or something like that
		err = responses.Send(response.StatusCode, writer, body)
		if err != nil {
			log.Error().Err(err).Msg("Error writing the response")
			handleServerError(writer, err)
			return
		}
	}
}

// evaluateProxyError handles detected error in proxyTo
// according to its type and the requested baseURL
func (server *HTTPServer) evaluateProxyError(writer http.ResponseWriter, err error, baseURL string) {
	if _, ok := err.(*url.Error); ok {
		switch baseURL {
		case server.ServicesConfig.AggregatorBaseEndpoint:
			handleServerError(writer, &AggregatorServiceUnavailableError{})
		case server.ServicesConfig.ContentBaseEndpoint:
			handleServerError(writer, &ContentServiceUnavailableError{})
		default:
			handleServerError(writer, err)
		}
	} else {
		handleServerError(writer, err)
	}
}

func sendRequest(
	client http.Client, req *http.Request, options *ProxyOptions,
) (*http.Response, []byte, error) {
	log.Debug().Msgf("Connecting to %s", req.URL.RequestURI())
	response, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Str(urlStr, req.URL.RequestURI()).Msg("Error during retrieve from URL")
		return nil, nil, err
	}

	if options != nil {
		var err error
		response, err = modifyResponse(options.ResponseModifiers, response)
		if err != nil {
			return nil, nil, err
		}
	}

	defer services.CloseResponseBody(response)

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error().Err(err).Str(urlStr, req.RequestURI).Msg("Error while retrieving content from request")
		return nil, nil, err
	}

	return response, body, nil
}

func (server *HTTPServer) composeEndpoint(baseEndpoint, currentEndpoint string) (*url.URL, error) {
	endpoint := strings.TrimPrefix(currentEndpoint, server.Config.APIv1Prefix)
	endpoint = strings.TrimPrefix(endpoint, server.Config.APIv2Prefix)
	endpoint = strings.TrimPrefix(endpoint, server.Config.APIdbgPrefix)

	joinedURL, err := url.JoinPath(baseEndpoint, endpoint)
	if err != nil {
		log.Error().Err(err).Str("api", baseEndpoint).Str("endpoint", currentEndpoint).Msg("Error while joining endpoint to given API URL")
		return nil, err
	}
	return url.Parse(joinedURL)
}

func copyHeader(srcHeaders, dstHeaders http.Header) {
	for headerKey, headerValues := range srcHeaders {
		for _, value := range headerValues {
			dstHeaders.Add(headerKey, value)
		}
	}
}

func (server HTTPServer) getClusterInfoFromAMS(orgID ctypes.OrgID) (
	clusterInfoList []types.ClusterInfo,
	err error,
) {
	// providing nil filters will mean default filters will be applied
	clusterInfoList, err = server.amsClient.GetClustersForOrganization(orgID, nil, nil)
	if err != nil {
		log.Error().Err(err).Int(orgIDTag, int(orgID)).Msg("Error retrieving clusters from AMS API")
		return
	}
	log.Info().Int(orgIDTag, int(orgID)).Msgf("Number of clusters retrieved from the AMS API: %v", len(clusterInfoList))
	return
}

// readClusterInfoForOrgID returns a list of cluster info types and a map of cluster display names
func (server HTTPServer) readClusterInfoForOrgID(orgID ctypes.OrgID) (
	[]types.ClusterInfo,
	error,
) {
	if server.amsClient != nil {
		clusterInfoList, err := server.getClusterInfoFromAMS(orgID)
		if err != nil {
			log.Error().Err(err).Int(orgIDTag, int(orgID)).Msg("Error retrieving cluster info from AMS API")
			return clusterInfoList, err
		}

		return clusterInfoList, nil
	}

	if !server.Config.UseOrgClustersFallback {
		err := fmt.Errorf("amsclient not initialized")
		log.Error().Err(err).Send()
		return nil, err
	}

	log.Info().Msg("amsclient not initialized. Using fallback mechanism")
	clusterIDs, err := server.getClusterDetailsFromAggregator(orgID)
	if err != nil {
		log.Error().Err(err).Msg("error retrieving clusters from aggregator")
		return nil, err
	}

	// fill in empty display names
	clusterInfo := make([]types.ClusterInfo, 0)

	for _, clusterID := range clusterIDs {
		clusterInfo = append(clusterInfo, types.ClusterInfo{
			ID:          clusterID,
			DisplayName: string(clusterID),
		})
	}
	return clusterInfo, nil
}

// getClusterDetailsFromAggregator reads the list of clusters for a given organization from aggregator
func (server HTTPServer) getClusterDetailsFromAggregator(orgID ctypes.OrgID) ([]ctypes.ClusterName, error) {
	log.Debug().Msg("retrieving cluster IDs from aggregator")

	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.ClustersForOrganizationEndpoint,
		orgID,
	)

	// #nosec G107
	response, err := http.Get(aggregatorURL)
	if err != nil {
		log.Error().Err(err).Msg("problem getting cluster list from aggregator")
		if _, ok := err.(*url.Error); ok {
			return nil, &AggregatorServiceUnavailableError{}
		}
		return nil, err
	}

	var recvMsg struct {
		Status   string               `json:"status"`
		Clusters []ctypes.ClusterName `json:"clusters"`
	}

	defer services.CloseResponseBody(response)

	err = json.NewDecoder(response.Body).Decode(&recvMsg)
	return recvMsg.Clusters, err
}

// readAggregatorReportForClusterID reads report from aggregator,
// handles errors by sending corresponding message to the user.
// Returns report and bool value set to true if there was no errors
func (server HTTPServer) readAggregatorReportForClusterID(
	orgID ctypes.OrgID, clusterID ctypes.ClusterName, userID ctypes.UserID, writer http.ResponseWriter,
) (*ctypes.ReportResponse, bool) {
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
		if _, ok := err.(*url.Error); ok {
			handleServerError(writer, &AggregatorServiceUnavailableError{})
		} else {
			log.Error().Str(clusterIDTag, string(clusterID)).Err(err).Msg("readAggregatorReportForClusterID unexpected error for cluster")
			handleServerError(writer, err)
		}
		return nil, false
	}

	var aggregatorResponse struct {
		Report *ctypes.ReportResponse `json:"report"`
		Status string                 `json:"status"`
	}

	defer services.CloseResponseBody(aggregatorResp)

	responseBytes, err := io.ReadAll(aggregatorResp.Body)
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
		log.Error().Str(clusterIDTag, string(clusterID)).Err(err).Msg("readAggregatorReportForClusterID error unmarshaling response for cluster")
		handleServerError(writer, err)
		return nil, false
	}
	logClusterInfos(orgID, clusterID, aggregatorResponse.Report.Report)

	return aggregatorResponse.Report, true
}

// readAggregatorReportMetainfoForClusterID reads report metainfo from Aggregator,
// handles errors by sending corresponding message to the user.
// Returns report and bool value set to true if there was no errors
func (server HTTPServer) readAggregatorReportMetainfoForClusterID(
	orgID ctypes.OrgID, clusterID ctypes.ClusterName, userID ctypes.UserID, writer http.ResponseWriter,
) (*ctypes.ReportResponseMetainfo, bool) {
	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.ReportMetainfoEndpoint,
		orgID,
		clusterID,
		userID,
	)

	// #nosec G107
	aggregatorResp, err := http.Get(aggregatorURL)
	if err != nil {
		if _, ok := err.(*url.Error); ok {
			handleServerError(writer, &AggregatorServiceUnavailableError{})
		} else {
			handleServerError(writer, err)
		}
		return nil, false
	}

	var aggregatorResponse struct {
		Metainfo *ctypes.ReportResponseMetainfo `json:"metainfo"`
		Status   string                         `json:"status"`
	}

	defer services.CloseResponseBody(aggregatorResp)

	responseBytes, err := io.ReadAll(aggregatorResp.Body)
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

	return aggregatorResponse.Metainfo, true
}

func (server HTTPServer) readAggregatorReportForClusterList(
	orgID ctypes.OrgID, clusterList []string, writer http.ResponseWriter,
) (*ctypes.ClusterReports, bool) {
	clist := strings.Join(clusterList, ",")
	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.ReportForListOfClustersEndpoint,
		orgID,
		clist)

	// #nosec G107
	aggregatorResp, err := http.Get(aggregatorURL)
	if err != nil {
		if _, ok := err.(*url.Error); ok {
			handleServerError(writer, &AggregatorServiceUnavailableError{})
		} else {
			handleServerError(writer, err)
		}
		return nil, false
	}

	var aggregatorResponse ctypes.ClusterReports

	defer services.CloseResponseBody(aggregatorResp)

	responseBytes, err := io.ReadAll(aggregatorResp.Body)
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
	logClustersReport(orgID, aggregatorResponse.Reports)

	return &aggregatorResponse, true
}

func (server HTTPServer) readAggregatorReportForClusterListFromBody(
	orgID ctypes.OrgID, request *http.Request, writer http.ResponseWriter,
) (*ctypes.ClusterReports, bool) {
	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.ReportForListOfClustersPayloadEndpoint,
		orgID,
	)

	body, err := io.ReadAll(request.Body)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}
	// #nosec G107
	aggregatorResp, err := http.Post(aggregatorURL, JSONContentType, bytes.NewBuffer(body))
	if err != nil {
		if _, ok := err.(*url.Error); ok {
			handleServerError(writer, &AggregatorServiceUnavailableError{})
		} else {
			handleServerError(writer, err)
		}
		return nil, false
	}

	if reportResponse, ok := handleReportsResponse(aggregatorResp, writer); ok {
		logClustersReport(orgID, reportResponse.Reports)
		return reportResponse, true
	}
	return nil, false
}

// handleReportsResponse analyses the aggregator's response and
// writes an appropriate response to the client, handling any
// possible error in the meantime
func handleReportsResponse(response *http.Response, writer http.ResponseWriter) (*ctypes.ClusterReports, bool) {
	var aggregatorResponse ctypes.ClusterReports

	defer services.CloseResponseBody(response)

	responseBytes, err := io.ReadAll(response.Body)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

	if response.StatusCode != http.StatusOK {
		err := responses.Send(response.StatusCode, writer, responseBytes)
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
	orgID ctypes.OrgID, clusterID ctypes.ClusterName, userID ctypes.UserID, ruleID ctypes.RuleID, errorKey ctypes.ErrorKey, writer http.ResponseWriter,
) (*ctypes.RuleOnReport, bool) {
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
		if _, ok := err.(*url.Error); ok {
			handleServerError(writer, &AggregatorServiceUnavailableError{})
		} else {
			handleServerError(writer, err)
		}
		return nil, false
	}

	var aggregatorResponse struct {
		Report *ctypes.RuleOnReport `json:"report"`
		Status string               `json:"status"`
	}

	defer services.CloseResponseBody(aggregatorResp)

	responseBytes, err := io.ReadAll(aggregatorResp.Body)
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
	logClusterInfo(orgID, clusterID, aggregatorResponse.Report)

	return aggregatorResponse.Report, true
}

func (server HTTPServer) fetchAggregatorReport(
	writer http.ResponseWriter, request *http.Request,
) (aggregatorResponse *ctypes.ReportResponse, successful bool, clusterID ctypes.ClusterName) {
	clusterID, successful = httputils.ReadClusterName(writer, request)
	// Error message handled by function
	if !successful {
		log.Info().Msg("fetchAggregatorReport unable to read clusterID")
		return
	}

	orgID, userID, err := server.GetCurrentOrgIDUserIDFromToken(request)
	if err != nil {
		log.Error().Err(err).Interface("clusterID", clusterID).Msg("fetchAggregatorReport unable to get orgID or userID for cluster")
		handleServerError(writer, err)
		return
	}
	log.Info().Msgf("fetchAggregatorReport orgID %v userID %v for cluster %v", orgID, userID, clusterID)

	aggregatorResponse, successful = server.readAggregatorReportForClusterID(orgID, clusterID, userID, writer)
	if !successful {
		log.Error().Msg("fetchAggregatorReport unable to get response from aggregator")
		return
	}
	return
}

// fetchAggregatorReportMetainfo method tries to fetch metainformation about
// report for selected cluster.
func (server HTTPServer) fetchAggregatorReportMetainfo(
	writer http.ResponseWriter, request *http.Request,
) (aggregatorResponse *ctypes.ReportResponseMetainfo, successful bool, clusterID ctypes.ClusterName) {
	clusterID, successful = httputils.ReadClusterName(writer, request)
	// Error message handled by function
	if !successful {
		return
	}

	orgID, userID, err := server.GetCurrentOrgIDUserIDFromToken(request)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	aggregatorResponse, successful = server.readAggregatorReportMetainfoForClusterID(orgID, clusterID, userID, writer)
	if !successful {
		return
	}
	return
}

// fetchAggregatorReports method access the Insights Results Aggregator to read
// reports for given list of clusters. Then the response structure is
// constructed from data returned by Aggregator.
func (server HTTPServer) fetchAggregatorReports(
	writer http.ResponseWriter, request *http.Request,
) (*ctypes.ClusterReports, bool) {
	// cluster list is specified in path (part of URL)
	clusterList, successful := httputils.ReadClusterListFromPath(writer, request)
	// Error message handled by function
	if !successful {
		return nil, false
	}

	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

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
) (*ctypes.ClusterReports, bool) {
	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

	aggregatorResponse, successful := server.readAggregatorReportForClusterListFromBody(orgID, request, writer)
	if !successful {
		return nil, false
	}
	return aggregatorResponse, true
}

// SetAMSInfoInReport tries to retrieve the display name and managed status of the cluster using
// the configured AMS client. If no info is retrieved, it sets the cluster's external
// ID as display name.
func (server HTTPServer) SetAMSInfoInReport(clusterID types.ClusterName, report *types.SmartProxyReportV2) {
	if server.amsClient != nil {
		clusterInfo := server.amsClient.GetClusterDetailsFromExternalClusterID(clusterID)
		report.Meta.Managed = clusterInfo.Managed
		if clusterInfo.DisplayName != "" {
			report.Meta.DisplayName = clusterInfo.DisplayName
			return
		}
	}
	report.Meta.DisplayName = string(clusterID)
}

func (server HTTPServer) buildReportEndpointResponse(
	writer http.ResponseWriter, request *http.Request,
	aggregatorResponse *ctypes.ReportResponse,
	clusterID types.ClusterName,
	osdFlag bool,
) (visibleRules []types.RuleWithContentResponse, rulesCount int, err error) {
	includeDisabled, err := readGetDisabledParam(request)
	if err != nil {
		handleServerError(writer, err)
		return
	}
	log.Debug().Msgf("Cluster ID: %v; %s flag = %t", clusterID, GetDisabledParam, includeDisabled)

	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		log.Error().Msg(authTokenFormatError)
		handleServerError(writer, err)
		return
	}

	acks, err := server.readListOfAckedRules(orgID)
	if err != nil {
		log.Error().Err(err).Int(orgIDTag, int(orgID)).Msg("Unable to retrieve list of acked rules for given organization")
		// server error has been handled already
		return
	}

	systemWideRuleDisables := generateRuleAckMap(acks)

	visibleRules, noContentRulesCnt, disabledRulesCnt, err := filterRulesInResponse(
		aggregatorResponse.Report, osdFlag, includeDisabled, systemWideRuleDisables,
	)
	log.Debug().Msgf("Cluster ID: %v; visible rules %d, no content rules %d, disabled rules %d",
		clusterID, len(visibleRules), noContentRulesCnt, disabledRulesCnt,
	)

	if _, ok := err.(*content.RuleContentDirectoryTimeoutError); ok {
		handleServerError(writer, err)
		return nil, 0, err
	}

	rulesCount = server.getRuleCount(visibleRules, noContentRulesCnt, disabledRulesCnt, clusterID)
	return
}

func sendReportReponse(writer http.ResponseWriter, report interface{}) {
	err := responses.SendOK(writer, responses.BuildOkResponseWithData(reportStr, report))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// reportEndpointV1 serves /report endpoint without cluster_name field in the metadata.
// For requests made by the Insights Operator, we need to take managed clusters into account and
// communicate with the AMS API to retrieve the information about the cluster. Other consumers of this
// endpoints MUSTN'T be affected. From OCP v4.11, the IO uses the API v2 equivalent, which already works
// as expected, but we must fix this behaviour for request made by earlier versions. For more info, see
// https://issues.redhat.com/browse/CCXDEV-9393 and the linked issue.
func (server HTTPServer) reportEndpointV1(writer http.ResponseWriter, request *http.Request) {
	var managedCluster bool

	orgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		log.Error().Msg(authTokenFormatError)
		handleServerError(writer, err)
		return
	}

	aggregatorResponse, successful, clusterID := server.fetchAggregatorReport(writer, request)
	if !successful {
		return
	}

	// Uses SmartProxyReportV1 type for backward compatibility
	report := types.SmartProxyReportV1{
		Meta: types.ReportResponseMetaV1{
			LastCheckedAt: aggregatorResponse.Meta.LastCheckedAt,
		},
	}

	// we need to differentiate between requests made by Insights Operator and all other requests
	// see function comments for more info
	userAgentProduct := server.getKnownUserAgentProduct(request)

	if userAgentProduct == insightsOperatorUserAgent {
		// request made by insights-operator, we need to retrieve the managed status of a cluster from AMS
		if server.amsClient != nil {
			clusterInfo, err := server.amsClient.GetSingleClusterInfoForOrganization(orgID, clusterID)
			if err != nil {
				log.Warn().Err(err).Msg("unable to retrieve info from AMS API")
				handleServerError(writer, err)
				return
			}
			managedCluster = clusterInfo.Managed
		}
	} else {
		// request NOT made by Insights Operator, we're expecting the managed status in the URL param
		managedCluster, err = readOSDEligible(request)
		if err != nil {
			log.Err(err).Msgf("Cluster ID: %v; Got error while parsing `%s` value", clusterID, OSDEligibleParam)
		}
		log.Debug().Msgf("Cluster ID: %v; %s flag = %t", clusterID, OSDEligibleParam, managedCluster)
	}

	if report.Data, report.Meta.Count, err = server.buildReportEndpointResponse(
		writer, request, aggregatorResponse, clusterID, managedCluster); err == nil {
		sendReportReponse(writer, report)
	}
}

// reportEndpointV2 serves /report endpoint with cluster_name field in the metadata
func (server HTTPServer) reportEndpointV2(writer http.ResponseWriter, request *http.Request) {
	aggregatorResponse, successful, clusterID := server.fetchAggregatorReport(writer, request)
	if !successful {
		return
	}

	report := types.SmartProxyReportV2{}

	server.SetAMSInfoInReport(clusterID, &report)

	var err error

	if report.Data, report.Meta.Count, err = server.buildReportEndpointResponse(
		writer, request, aggregatorResponse, clusterID, report.Meta.Managed); err == nil {
		// fill in timestamps
		report.Meta.LastCheckedAt = aggregatorResponse.Meta.LastCheckedAt
		report.Meta.GatheredAt = aggregatorResponse.Meta.GatheredAt

		fillImpacted(report.Data, aggregatorResponse.Report)
		sendReportReponse(writer, report)
	}
}

func fillImpacted(
	rulesWithContent []types.RuleWithContentResponse,
	aggregatorReports []ctypes.RuleOnReport) {
	idReport := make(map[string]ctypes.RuleOnReport, len(aggregatorReports))

	for _, v := range aggregatorReports {
		id := string(v.ErrorKey) + string(v.Module)
		idReport[id] = v
	}

	for i, ruleWithContent := range rulesWithContent {
		id := string(ruleWithContent.ErrorKey) + string(ruleWithContent.RuleID)
		report, ok := idReport[id]
		CreatedAtTime, err := time.Parse(time.RFC3339, string(report.CreatedAt))
		if err != nil {
			log.Warn().Err(err).Msgf("fillImpacted: invalid time format %v", report.CreatedAt)
			continue
		}
		if ok && !CreatedAtTime.IsZero() {
			ruleWithContent.Impacted = report.CreatedAt
			rulesWithContent[i] = ruleWithContent
		}
	}
}

func (server HTTPServer) getKnownUserAgentProduct(request *http.Request) (userAgentProduct string) {
	userAgentProduct = readUserAgentHeaderProduct(request)

	switch userAgentProduct {
	case insightsOperatorUserAgent:
		log.Debug().Msg("request made by Insights Operator to be shown in the OCP Web console")
	case acmUserAgent:
		log.Debug().Msg("request made by ACM Operator to be shown in the the Advanced Cluster Management")
	case browserUserAgent:
		log.Debug().Msg("request made by a regular web browser")
	case openAPIGeneratorUserAgent:
		log.Debug().Msg("request made by OpenAPI-generated test client from iqe tests")
	case pythonRequestsUserAgent:
		log.Debug().Msg("request made by Python requests library probably from iqe tests")
	case nonRelevantUserAgent:
		log.Debug().Msg("request made by non-relevant-user-agent test case from iqe tests")
	default:
		log.Error().Str(userAgentHeader, request.Header.Get(userAgentHeader)).
			Str("userAgentProduct", userAgentProduct).
			Msg("improper or unknown user agent product")
	}

	return
}

// readMetainfo method retrieves metainformations for report stored in
// Aggregator's database and return the retrieved info to requester via
// response payload. The payload has type types.ReportResponseMetainfo
func (server HTTPServer) reportMetainfoEndpoint(writer http.ResponseWriter, request *http.Request) {
	aggregatorResponse, successful, clusterID := server.fetchAggregatorReportMetainfo(writer, request)
	if !successful {
		return
	}

	log.Debug().Msgf("Metainfo returned by aggregator for cluster %s: %v", clusterID, aggregatorResponse)

	err := responses.SendOK(writer, responses.BuildOkResponseWithData("metainfo", aggregatorResponse))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// getRuleCount returns the number of visible rules without those that do not have content
func (server HTTPServer) getRuleCount(visibleRules []types.RuleWithContentResponse,
	noContentRulesCnt int,
	disabledRulesCnt int,
	clusterID ctypes.ClusterName,
) int {
	totalRuleCnt := len(visibleRules) + noContentRulesCnt

	// Edge case where rules are hitting, but we don't have content for any of them.
	// This case should appear as "No issues found" in customer-facing applications, because the only
	// thing we could show is rule module + error key, which have no informational value to customers.
	if len(visibleRules) == 0 && noContentRulesCnt > 0 && disabledRulesCnt == 0 {
		log.Error().Interface("clusterID", clusterID).
			Msg("Rules are hitting, but we don't have content for any of them.")
		totalRuleCnt = 0
	}
	return totalRuleCnt
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
// clients can use as many cluster ID as they want without any (real) limits.
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
) (*ctypes.RuleOnReport, bool) {
	clusterID, successful := httputils.ReadClusterName(writer, request)
	// Error message handled by function
	if !successful {
		return nil, false
	}

	ruleID, errorKey, err := readRuleIDWithErrorKey(writer, request)
	if err != nil {
		return nil, false
	}

	orgID, userID, err := server.GetCurrentOrgIDUserIDFromToken(request)
	if err != nil {
		handleServerError(writer, err)
		return nil, false
	}

	aggregatorResponse, successful := server.readAggregatorRuleForClusterID(orgID, clusterID, userID, ruleID, errorKey, writer)
	if !successful {
		return nil, false
	}
	return aggregatorResponse, true
}

func (server HTTPServer) singleRuleEndpoint(writer http.ResponseWriter, request *http.Request) {
	var rule *types.RuleWithContentResponse
	var filtered bool
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
	rule, filtered, err = content.FetchRuleContent(aggregatorResponse, osdFlag)

	if err != nil || filtered {
		handleFetchRuleContentError(writer, err)
		return
	}

	if rule.Internal {
		err = server.checkInternalRulePermissions(request)
		if err != nil {
			log.Error().Err(err).Send()
			handleServerError(writer, err)
			return
		}
	}

	err = responses.SendOK(writer, responses.BuildOkResponseWithData(reportStr, *rule))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

func handleFetchRuleContentError(writer http.ResponseWriter, err error) {
	if _, ok := err.(*content.RuleContentDirectoryTimeoutError); ok {
		log.Error().Err(err).Send()
		handleServerError(writer, err)
		return
	}
	err = responses.SendNotFound(writer, "Rule was not found")
	if err != nil {
		handleServerError(writer, err)
		return
	}
}

// checkInternalRulePermissions method checks if organizations for internal
// rules are enabled if so, retrieves the org_id from request/token and returns
// whether that ID is on the list of allowed organizations to access internal
// rules
func (server *HTTPServer) checkInternalRulePermissions(request *http.Request) error {
	if !server.Config.EnableInternalRulesOrganizations || !server.Config.Auth {
		return nil
	}

	requestOrgID, err := server.GetCurrentOrgID(request)
	if err != nil {
		log.Error().Err(err).Msg("error retrieving org_id from token")
		return err
	}

	log.Debug().Msgf("Checking internal rule permissions for Organization ID: %v", requestOrgID)
	for _, allowedID := range server.Config.InternalRulesOrganizations {
		if requestOrgID == allowedID {
			log.Info().Msgf("Organization %v is allowed access to internal rules", requestOrgID)
			return nil
		}
	}

	// If the loop ends without returning nil, then an authentication error should be raised
	const message = "This organization is not allowed to access this recommendation"
	return &auth.AuthenticationError{ErrString: message}
}

// getGroupsConfig retrieves the groups configuration from a channel to get the
// latest valid one
func (server HTTPServer) getGroupsConfig() (
	ruleGroups []groups.Group,
	err error,
) {
	var errorFound bool
	ruleGroups = []groups.Group{}

	select {
	case val, ok := <-server.ErrorFoundChannel:
		if !ok {
			log.Error().Msg("errorFound channel is closed")
			return
		}
		errorFound = val
	default:
		fmt.Println("errorFound channel is empty")
		return
	}

	if errorFound {
		err = <-server.ErrorChannel
		if _, ok := err.(*content.RuleContentDirectoryTimeoutError); ok {
			log.Error().Err(err).Send()
		}
		log.Error().Err(err).Msg("Error occurred during groups retrieval from content service")
		return nil, err
	}

	groupsConfig := <-server.GroupsChannel
	if groupsConfig == nil {
		err := errors.New("no groups retrieved")
		log.Error().Err(err).Msg("groups cannot be retrieved from content service. Check logs")
		return nil, err
	}

	return groupsConfig, nil
}

func isDisabledForOrgRule(aggregatorRule ctypes.RuleOnReport, systemWideDisabledRules map[types.RuleID]bool) bool {
	if len(systemWideDisabledRules) > 0 {
		selector := types.RuleID(
			fmt.Sprintf("%v|%v",
				strings.TrimSuffix(string(aggregatorRule.Module), dotReport),
				aggregatorRule.ErrorKey,
			),
		)
		log.Debug().Msgf("org-wide disabled rule ID %v|%v", aggregatorRule.Module, aggregatorRule.ErrorKey)
		return systemWideDisabledRules[selector]
	}
	return false
}

func isDisabledRule(aggregatorRule ctypes.RuleOnReport, systemWideDisabledRules map[types.RuleID]bool) bool {
	if aggregatorRule.Disabled {
		log.Debug().Msgf("on report disabled rule ID %v|%v", aggregatorRule.Module, aggregatorRule.ErrorKey)
		return true
	}
	return isDisabledForOrgRule(aggregatorRule, systemWideDisabledRules)
}

// filterRulesInResponse returns an array of RuleWithContentResponse with only the rules that matches 3 criteria:
// - The rule has content from the content-service
// - The disabled filter is not match
// - The OSD elegible filter is not match
func filterRulesInResponse(aggregatorReport []ctypes.RuleOnReport, filterOSD, getDisabled bool,
	systemWideDisabledRules map[types.RuleID]bool) (
	okRules []types.RuleWithContentResponse,
	noContentRulesCnt int,
	disabledRulesCnt int,
	contentError error,
) {
	log.Debug().Bool(GetDisabledParam, getDisabled).Bool(OSDEligibleParam, filterOSD).Msg("Filtering rules in report")
	okRules = []types.RuleWithContentResponse{}
	disabledRulesCnt, noContentRulesCnt = 0, 0

	for i := range aggregatorReport {
		aggregatorRule := aggregatorReport[i]
		if !getDisabled && isDisabledRule(aggregatorRule, systemWideDisabledRules) {
			log.Debug().Msgf("disabled rule ID %v|%v", aggregatorRule.Module, aggregatorRule.ErrorKey)
			disabledRulesCnt++
			continue
		}

		rule, filtered, err := content.FetchRuleContent(&aggregatorRule, filterOSD)
		if err != nil {
			if !filtered {
				// rule has not been filtered by OSDEligible field
				log.Debug().Msgf("no content rule ID %v|%v", aggregatorRule.Module, aggregatorRule.ErrorKey)
				noContentRulesCnt++
			}
			if _, ok := err.(*content.RuleContentDirectoryTimeoutError); ok {
				// error occured during communication with Content Service
				log.Error().Err(err).Send()
				contentError = err
				return
			}
			continue
		}

		if filtered {
			// rule has been filtered by OSDEligible field
			log.Debug().Msgf("osd filtered rule ID %v|%v", aggregatorRule.Module, aggregatorRule.ErrorKey)
			continue
		}

		okRules = append(okRules, *rule)
	}

	return
}

// Method readListOfClusterDisabledRules returns rules with a list of clusters for which the user had
// disabled the rule (if any)
func (server *HTTPServer) readListOfClusterDisabledRules(orgID types.OrgID) ([]ctypes.DisabledRule, error) {
	// wont be used anywhere else
	var response struct {
		Status        string                `json:"status"`
		DisabledRules []ctypes.DisabledRule `json:"rules"`
	}

	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.ListOfDisabledRules,
		orgID,
	)

	// #nosec G107
	resp, err := http.Get(aggregatorURL)
	if err != nil {
		return nil, err
	}

	// check the aggregator response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		err := fmt.Errorf("error reading disabled rules from aggregator: %v", resp.StatusCode)
		return nil, err
	}

	defer services.CloseResponseBody(resp)

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.DisabledRules, nil
}

// Method readListOfClusterDisabledRules returns user disabled rules for given cluster list
func (server *HTTPServer) readListOfDisabledRulesForClusters(
	writer http.ResponseWriter,
	orgID ctypes.OrgID,
	clusterList []ctypes.ClusterName,
) ([]ctypes.DisabledRule, error) {
	// wont be used anywhere else
	var response struct {
		Status        string                `json:"status"`
		DisabledRules []ctypes.DisabledRule `json:"rules"`
	}

	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.ListOfDisabledRulesForClusters,
		orgID,
	)

	jsonMarshalled, err := json.Marshal(clusterList)
	if err != nil {
		log.Error().Err(err).Msg("readListOfDisabledRulesForClusters problem unmarshalling cluster list")
		handleServerError(writer, err)
		return nil, err
	}

	// #nosec G107
	resp, err := http.Post(aggregatorURL, JSONContentType, bytes.NewBuffer(jsonMarshalled))
	if err != nil {
		log.Error().Err(err).Msg("readListOfDisabledRulesForClusters problem getting response from aggregator")
		if _, ok := err.(*url.Error); ok {
			handleServerError(writer, &AggregatorServiceUnavailableError{})
		} else {
			handleServerError(writer, err)
		}
		return nil, err
	}

	if resp.StatusCode == http.StatusInternalServerError {
		log.Error().Msg("failed to get response from aggregator")
		handleServerError(writer, err)
		return nil, err
	}

	// check the aggregator response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		err := fmt.Errorf("error reading disabled rules from aggregator: %v", resp.StatusCode)
		return nil, err
	}

	defer services.CloseResponseBody(resp)

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.DisabledRules, nil
}

// getClusterListAndUserData returns a list of clusters, rule hits for these clusters from
// aggregator, as well as rule acknowledgements and user disabled rules
func (server *HTTPServer) getClusterListAndUserData(
	writer http.ResponseWriter,
	orgID types.OrgID,
	userID types.UserID,
) (
	clusterInfoList []types.ClusterInfo,
	clusterRecommendationMap ctypes.ClusterRecommendationMap,
	ackedRulesMap map[ctypes.RuleID]bool,
	disabledRulesPerCluster map[ctypes.ClusterName][]ctypes.RuleID,
) {
	tStart := time.Now()
	// get list of clusters from AMS API or aggregator
	clusterInfoList, err := server.readClusterInfoForOrgID(orgID)
	if err != nil {
		log.Error().Err(err).Int(orgIDTag, int(orgID)).Msg(clusterListError)
		handleServerError(writer, err)
		return
	}
	log.Info().Uint32(orgIDTag, uint32(orgID)).Msgf("time spent in AMS API %s", time.Since(tStart))

	clusterRecommendationMap, err = server.getClustersAndRecommendations(writer, orgID, userID, types.GetClusterNames(clusterInfoList))
	if err != nil {
		log.Error().
			Err(err).
			Int(orgIDTag, int(orgID)).
			Str(userIDTag, string(userID)).
			Msgf("problem getting clusters and impacting recommendations from aggregator for cluster list (# of clusters %v)", len(clusterInfoList))

		return
	}

	// get a map of acknowledged rules
	ackedRulesMap, err = server.getRuleAcksMap(orgID)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// retrieve list of cluster IDs and single disabled rules for each cluster
	disabledRulesPerCluster = server.getUserDisabledRulesPerCluster(orgID)
	log.Debug().Uint32(orgIDTag, uint32(orgID)).Msgf("time since getClusterListAndUserData start %s", time.Since(tStart))

	return
}

// getWorkloadsForCluster returns []types.WorkloadsForCluster{} when no workloads were found for given cluster/namespace.
// returns nil upon receiving an unexpected error from aggregator.
func (server *HTTPServer) getWorkloadsForCluster(
	orgID types.OrgID,
	clusterID types.ClusterName,
	namespace types.Namespace,
) (
	workloads types.WorkloadsForCluster, err error,
) {
	var response struct {
		Status    string                    `json:"status"`
		Workloads types.WorkloadsForCluster `json:"workloads"`
	}

	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.DVOWorkloadRecommendationsSingleNamespace,
		orgID, namespace.UUID, clusterID,
	)

	// #nosec G107
	resp, err := http.Get(aggregatorURL)
	if err != nil {
		return
	}

	if resp.StatusCode == http.StatusNotFound {
		return workloads, &utypes.ItemNotFoundError{
			ItemID: fmt.Sprintf("cluster=%s;namespace=%s", clusterID, namespace.UUID)}
	}

	// check the aggregator response
	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("error reading workloads from aggregator: %v", resp.StatusCode)
		return
	}

	defer services.CloseResponseBody(resp)

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return
	}

	return response.Workloads, nil
}

// getWorkloadsForOrganization returns a list of workloads for given organization ID.
// Empty slice is returned when aggregator responds with 404 Not Found.
// Nil is returned when any other unexpected error occurs.
func (server *HTTPServer) getWorkloadsForOrganization(
	orgID types.OrgID, writer http.ResponseWriter, clusterInfo []types.ClusterInfo,
) ([]types.WorkloadsForNamespace, error) {
	// wont be used anywhere else
	var response struct {
		Status    string                        `json:"status"`
		Workloads []types.WorkloadsForNamespace `json:"workloads"`
	}

	aggregatorURL := httputils.MakeURLToEndpoint(
		server.ServicesConfig.AggregatorBaseEndpoint,
		ira_server.DVOWorkloadRecommendations,
		orgID,
	)

	// marshalling a list to JSON is much faster than marshaling a map
	clusterPayload := make([]types.ClusterName, len(clusterInfo))
	for i, clusterInfo := range clusterInfo {
		clusterPayload[i] = clusterInfo.ID
	}

	body, err := json.Marshal(clusterPayload)
	if err != nil {
		log.Error().Err(err).Msg("unable to marshal cluster list body")
		return nil, err
	}

	// #nosec G107
	resp, err := http.Post(aggregatorURL, JSONContentType, bytes.NewBuffer(body))
	if err != nil {
		if _, ok := err.(*url.Error); ok {
			handleServerError(writer, &AggregatorServiceUnavailableError{})
		} else {
			handleServerError(writer, err)
		}
		return nil, nil
	}

	if resp.StatusCode == http.StatusNotFound {
		return []types.WorkloadsForNamespace{}, nil
	}

	// check the aggregator response
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("error reading workloads from aggregator: %v", resp.StatusCode)
		return nil, err
	}

	defer services.CloseResponseBody(resp)

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return response.Workloads, nil
}
