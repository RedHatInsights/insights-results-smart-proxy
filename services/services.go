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
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/RedHatInsights/insights-content-service/groups"
	"github.com/rs/zerolog/log"
)

const (
	// GroupsEndpoint is the content-service endpoint for getting the list of groups
	GroupsEndpoint = "groups"
)

// GetGroups get the list of groups from content-service
func GetGroups(conf Configuration) ([]groups.Group, error) {
	type groupsResponse struct {
		Status string         `json:"status"`
		Groups []groups.Group `json:"groups"`
	}
	var receivedMsg groupsResponse

	log.Info().Msg("Updating groups information")

	groupsURL, err := url.Parse(conf.ContentBaseEndpoint + GroupsEndpoint)

	if err != nil {
		log.Error().Err(err).Msgf("Error during endpoint %s URL parsing", groupsURL.String())
		return nil, err
	}

	log.Debug().Msgf("Connecting to %s", groupsURL.String())
	resp, err := http.Get(groupsURL.String())

	if err != nil {
		log.Error().Err(err).Msgf("Error during retrieve of %s", groupsURL.String())
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
