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

package services

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/RedHatInsights/insights-content-service/content"
	"github.com/RedHatInsights/insights-content-service/groups"
	"github.com/rs/zerolog/log"
)

const (
	// ContentEndpoint is the content-service endpoint for getting the static content for all rules
	ContentEndpoint = "content"
	// GroupsEndpoint is the content-service endpoint for getting the list of groups
	GroupsEndpoint = "groups"
)

func getFromURL(endpoint string) (*http.Response, error) {
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		log.Error().Err(err).Msgf("Error during endpoint %s URL parsing", endpoint)
		return nil, err
	}

	log.Debug().Msgf("Connecting to %s", parsedURL.String())

	resp, err := http.Get(parsedURL.String())
	if err != nil {
		log.Error().Err(err).Msgf("Error during retrieve of %s", parsedURL.String())
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

	log.Info().Msg("Updating groups information")

	resp, err := getFromURL(conf.ContentBaseEndpoint + GroupsEndpoint)

	if err != nil {
		// Log already shown
		return nil, err
	}

	err = json.NewDecoder(resp.Body).Decode(&receivedMsg)

	if err != nil {
		log.Error().Err(err).Msg("Error while decoding groups answer from content-service")
		return nil, err
	}

	log.Info().Msgf("Received %d groups", len(receivedMsg.Groups))
	return receivedMsg.Groups, nil
}

// GetContent get the static rule content from content-service
func GetContent(conf Configuration) (*content.RuleContentDirectory, error) {
	type contentResponse struct {
		Status         string `json:"status"`
		EncodedContent []byte `json:"rule-content"`
	}
	var receivedMsg contentResponse

	log.Info().Msg("Updating rules static content")
	resp, err := getFromURL(conf.ContentBaseEndpoint + ContentEndpoint)

	if err != nil {
		// Log already shown
		return nil, err
	}

	err = json.NewDecoder(resp.Body).Decode(&receivedMsg)
	if err != nil {
		log.Error().Err(err).Msg("Error while decoding static content answer from content-service")
		return nil, err
	}

	var receivedContent content.RuleContentDirectory
	encodedContent := bytes.NewBuffer(receivedMsg.EncodedContent)
	err = gob.NewDecoder(encodedContent).Decode(&receivedContent)

	if err != nil {
		log.Error().Err(err).Msg("Error trying to decode rules content from received answer")
		return nil, err
	}

	return &receivedContent, nil
}
