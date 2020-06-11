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

package server

import (
	"errors"
	"net/http"

	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/rs/zerolog/log"
)

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

// getContentForRule retrieves the static content for the given ruleID
func (server HTTPServer) getContentForRule(writer http.ResponseWriter, request *http.Request) {
	contentConfig := <-server.ContentChannel

	if contentConfig.Rules == nil {
		err := errors.New("No rules content")
		log.Error().Err(err).Msg("Rules static content cannot be retrieved from content service. Check logs")
		handleServerError(writer, err)
		return
	}

	ruleID, err := readRuleID(writer, request)
	if err != nil {
		// already handled in readRuleID
		return
	}

	stringfiedRuleID := string(ruleID)

	for _, ruleContent := range contentConfig.Rules {
		// Check if the given {rule_id} match with the Python module name, that is used as RuleID
		if stringfiedRuleID == ruleContent.Plugin.PythonModule {
			err = responses.SendOK(writer, responses.BuildOkResponseWithData("content", ruleContent))
			if err != nil {
				log.Error().Err(err)
				handleServerError(writer, err)
			}
			return
		}
	}

	// if the loop ends without finding the ruleID, response with 404 code
	err = responses.SendNotFound(writer, "No content found for the given rule ID")
	if err != nil {
		log.Error().Err(err)
		handleServerError(writer, err)
	}
}
