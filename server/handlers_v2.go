// Copyright 2020, 2021 Red Hat, Inc
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

// handlers for API V2 endpoints

import (
	"net/http"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-operator-utils/responses"

	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	"github.com/RedHatInsights/insights-results-smart-proxy/types"
)

// getContentForRule retrieves the static content for the given ruleID tied
// with groups info
func (server HTTPServer) getContentWithGroupsForRule(writer http.ResponseWriter, request *http.Request) {
	ruleID, successful := httputils.ReadRuleID(writer, request)
	if !successful {
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
		err := server.checkInternalRulePermissions(request)
		if err != nil {
			handleServerError(writer, err)
			return
		}
	}

	// retrieve the latest groups configuration
	groupsConfig, err := server.getGroupsConfig()
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// prepare data structure for building response
	responseContent := make(map[string]interface{})
	responseContent["status"] = "ok"
	responseContent["groups"] = groupsConfig
	responseContent["content"] = ruleContent

	// send response to client
	err = responses.SendOK(writer, responseContent)
	if err != nil {
		handleServerError(writer, err)
		return
	}
}

// getContent retrieves all the static content tied with groups info
func (server HTTPServer) getContentWithGroups(writer http.ResponseWriter, request *http.Request) {
	// Generate an array of RuleContent
	allRules := content.GetAllContent()
	var rules []types.RuleContent

	if err := server.checkInternalRulePermissions(request); err != nil {
		for _, rule := range allRules {
			if !content.IsRuleInternal(types.RuleID(rule.Plugin.PythonModule)) {
				rules = append(rules, rule)
			}
		}
	} else {
		rules = allRules
	}

	// retrieve the latest groups configuration
	groupsConfig, err := server.getGroupsConfig()
	if err != nil {
		handleServerError(writer, err)
		return
	}

	// prepare data structure for building response
	responseContent := make(map[string]interface{})
	responseContent["status"] = "ok"
	responseContent["groups"] = groupsConfig
	responseContent["content"] = rules

	// send response to client
	err = responses.SendOK(writer, responseContent)
	if err != nil {
		handleServerError(writer, err)
		return
	}
}
