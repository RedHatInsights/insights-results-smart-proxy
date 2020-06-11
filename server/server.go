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
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	// we just have to import this package in order to expose pprof interface in debug mode
	// disable "G108 (CWE-): Profiling endpoint is automatically exposed on /debug/pprof"
	// #nosec G108
	_ "net/http/pprof"
	"path/filepath"

	"github.com/RedHatInsights/insights-content-service/content"
	"github.com/RedHatInsights/insights-content-service/groups"
	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-smart-proxy/services"
)

// HTTPServer in an implementation of Server interface
type HTTPServer struct {
	Config         Configuration
	ServicesConfig services.Configuration
	GroupsChannel  chan []groups.Group
	ContentChannel chan content.RuleContentDirectory
	Serv           *http.Server
}

// New constructs new implementation of Server interface
func New(config Configuration, servicesConfig services.Configuration, groupsChannel chan []groups.Group, contentChannel chan content.RuleContentDirectory) *HTTPServer {
	return &HTTPServer{
		Config:         config,
		ServicesConfig: servicesConfig,
		GroupsChannel:  groupsChannel,
		ContentChannel: contentChannel,
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

// addCORSHeaders - middleware for adding headers that should be in any response
func (server *HTTPServer) addCORSHeaders(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			nextHandler.ServeHTTP(w, r)
		})
}

// handleOptionsMethod - middleware for handling OPTIONS method
func (server *HTTPServer) handleOptionsMethod(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
			} else {
				nextHandler.ServeHTTP(w, r)
			}
		})
}

// Initialize perform the server initialization
func (server *HTTPServer) Initialize() http.Handler {
	log.Info().Msgf("Initializing HTTP server at '%s'", server.Config.Address)

	router := mux.NewRouter().StrictSlash(true)
	router.Use(server.LogRequest)

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
		router.Use(server.addCORSHeaders)
		router.Use(server.handleOptionsMethod)
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

func (server HTTPServer) proxyTo(baseURL string) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		log.Info().Msg("Handling response as a proxy")
		endpointURL, err := server.composeEndpoint(baseURL, request.RequestURI)

		if err != nil {
			log.Error().Err(err).Msgf("Error during endpoint %s URL parsing", request.RequestURI)
			handleServerError(writer, err)
		}

		client := http.Client{}
		req, _ := http.NewRequest(request.Method, endpointURL.String(), request.Body)
		copyHeader(request.Header, req.Header)

		log.Debug().Msgf("Connecting to %s", endpointURL.String())
		response, err := client.Do(req)
		if err != nil {
			log.Error().Err(err).Msgf("Error during retrieve of %s", endpointURL.String())
			handleServerError(writer, err)
		}

		content, err := ioutil.ReadAll(response.Body)

		if err != nil {
			log.Error().Err(err).Msgf("Error while retrieving content from request to %s", endpointURL.String())
			handleServerError(writer, err)
		}
		// Maybe this code should be on responses.SendRaw or something like that
		err = responses.Send(response.StatusCode, writer, content)
		if err != nil {
			log.Error().Err(err).Msgf("Error writing the response")
			handleServerError(writer, err)
		}
	}
}

// getGroups retrives the groups configuration from a channel to get the latest valid one and send the response back to the client
func (server *HTTPServer) getGroups(writer http.ResponseWriter, request *http.Request) {
	groupsConfig := <-server.GroupsChannel
	if groupsConfig == nil {
		err := errors.New("No groups retrieved")
		log.Error().Err(err).Msg("Groups cannot be retrieved from content service. Check logs")
		handleServerError(writer, err)
		return
	}

	responseContent := make(map[string]interface{})
	responseContent["status"] = "ok"
	responseContent["groups"] = groupsConfig
	err := responses.SendOK(writer, responseContent)
	if err != nil {
		log.Error().Err(err).Msg("Cannot send response")
		handleServerError(writer, err)
	}
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
