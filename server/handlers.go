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

	"github.com/RedHatInsights/insights-results-smart-proxy/content"
)

// getGroups retrieves the groups configuration from a channel to get the latest valid one
// and sends the response back to the client
func (server *HTTPServer) getGroups(writer http.ResponseWriter, _ *http.Request) {
	groupsConfig := <-server.GroupsChannel
	if groupsConfig == nil {
		err := errors.New("no groups retrieved")
		log.Error().Err(err).Msg("groups cannot be retrieved from content service. Check logs")
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
	ruleID, err := readRuleID(writer, request)
	if err != nil {
		// already handled in readRuleID
		return
	}

	ruleContent, err := content.GetRuleContent(ruleID)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// check for internal rule permissions
	if internal := content.IsRuleInternal(ruleID); internal == true {
		ok := server.checkInternalRulePermissions(writer, request)
		if ok != true {
			// handled in function
			return
		}
	}

	err = responses.SendOK(writer, responses.BuildOkResponseWithData("content", ruleContent))
	if err != nil {
		handleServerError(writer, err)
		return
	}
}
