// Copyright 2022  Red Hat, Inc
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
	"fmt"
	"net/http"
	"net/url"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	utypes "github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	ctypes "github.com/RedHatInsights/insights-results-types"
	"github.com/gorilla/mux"
)

func (server HTTPServer) newExtractUserIDFromTokenToURLRequestModifier(newEndpoint string) RequestModifier {
	return func(request *http.Request) (*http.Request, error) {
		userID, err := server.GetCurrentUserID(request)
		if err != nil {
			return nil, err
		}

		vars := mux.Vars(request)
		vars["user_id"] = fmt.Sprintf("%v", userID)

		newURL := httputils.MakeURLToEndpointMapString(server.Config.APIv1Prefix, newEndpoint, vars)
		request.URL, err = url.Parse(newURL)
		if err != nil {
			return nil, &ParamsParsingError{}
		}

		request.RequestURI = request.URL.RequestURI()

		return request, nil
	}
}

func (server HTTPServer) extractUserIDOrgIDFromTokenToURLRequestModifier(newEndpoint string) RequestModifier {
	return func(request *http.Request) (*http.Request, error) {
		orgID, userID, err := server.GetCurrentOrgIDUserIDFromToken(request)
		if err != nil {
			return nil, &ParamsParsingError{}
		}

		vars := mux.Vars(request)
		vars["user_id"] = string(userID)
		vars["org_id"] = fmt.Sprintf("%v", orgID)

		newURL := httputils.MakeURLToEndpointMapString(server.Config.APIv1Prefix, newEndpoint, vars)
		request.URL, err = url.Parse(newURL)
		if err != nil {
			return nil, &ParamsParsingError{}
		}

		request.RequestURI = request.URL.RequestURI()

		return request, nil
	}
}

// checkRuleIDAndErrorKeyAreValid request modifier that only checks if
// both rule_id and error_key are valid, given the rules and errors defined
// in the content service
func checkRuleIDAndErrorKeyAreValid() RequestModifier {
	return func(request *http.Request) (*http.Request, error) {
		ruleID, err := httputils.GetRouterParam(request, RuleIDParamName)
		if err != nil {
			return nil, err
		}

		errorKey, err := httputils.GetRouterParam(request, "error_key")
		if err != nil {
			return nil, err
		}

		// Check if rule id and error key are valid ones
		_, err = content.GetRuleWithErrorKeyContent(ctypes.RuleID(ruleID), ctypes.ErrorKey(errorKey))

		// if valid, perform request to aggregator and return response as usual
		if err != nil {
			return nil, &utypes.ItemNotFoundError{ItemID: fmt.Sprintf("%s/%s", ruleID, errorKey)}
		}

		return request, nil
	}
}
