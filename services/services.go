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

// Package services contains interface implementations to other REST API
// services that are called from Smart Proxy. Currently the interface is
// implemented for Content Service.
package services

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/RedHatInsights/insights-content-service/groups"
	types "github.com/RedHatInsights/insights-results-types"
	"github.com/rs/zerolog/log"
)

const (
	// ContentEndpoint is the content-service endpoint for getting the static content for all rules
	ContentEndpoint = "content"
	// GroupsEndpoint is the content-service endpoint for getting the list of groups
	GroupsEndpoint = "groups"

	urlStr = "url"
)

func getFromURL(endpoint string) (*http.Response, error) {
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		log.Error().Err(err).Str(urlStr, endpoint).Msg("Error during endpoint URL parsing")
		return nil, err
	}

	log.Debug().Msgf("Connecting to %s", parsedURL.String())

	resp, err := http.Get(parsedURL.String())
	if err != nil {
		log.Error().Err(err).Str(urlStr, parsedURL.String()).Msg("Error during retrieve of URL")
		return nil, err
	}

	return resp, nil
}

// GetGroups get the list of groups from content-service
func GetGroups(conf Configuration) ([]groups.Group, error) {
	type groupsResponse struct {
		Status string         `json:"status"`
		Groups []groups.Group `json:"groups"`
	}
	var receivedMsg groupsResponse

	log.Debug().Msg("Updating groups information")

	resp, err := getFromURL(conf.ContentBaseEndpoint + GroupsEndpoint) //nolint:bodyclose // TODO: remove once the bodyclose library fixes this bug

	if err != nil {
		// Log already shown
		return nil, err
	}

	defer CloseResponseBody(resp)

	err = json.NewDecoder(resp.Body).Decode(&receivedMsg)

	if err != nil {
		log.Error().Err(err).Msg("Error while decoding groups answer from content-service")
		return nil, err
	}

	log.Debug().Msgf("Received %d groups", len(receivedMsg.Groups))
	return receivedMsg.Groups, nil
}

// GetContent get the static rule content from content-service
func GetContent(conf Configuration) (*types.RuleContentDirectory, error) {
	log.Debug().Msg("getting rules static content")
	resp, err := getFromURL(conf.ContentBaseEndpoint + ContentEndpoint) //nolint:bodyclose // TODO: remove once the bodyclose library fixes this bug

	if err != nil {
		return nil, err
	}

	defer CloseResponseBody(resp)

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var receivedContent types.RuleContentDirectory
	err = gob.NewDecoder(bytes.NewReader(respBytes)).Decode(&receivedContent)
	if err != nil {
		log.Error().Err(err).Msg("error trying to decode rules content from received answer")
		return nil, err
	}

	log.Debug().Msgf("Got %d rules from content-service", len(receivedContent.Rules))

	return &receivedContent, nil
}

// CloseResponseBody is used to close the response body so that there are no
// memory leaks in the TCP socket: CCXDEV-10514
func CloseResponseBody(response *http.Response) {
	if response == nil {
		return
	}
	err := response.Body.Close()
	if err != nil {
		log.Error().Err(err).Msg("error closing acking response")
	}
}
