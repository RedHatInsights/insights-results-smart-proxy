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
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	ctypes "github.com/RedHatInsights/insights-results-types"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-smart-proxy/types"
)

const (
	// OSDEligibleParam parameter
	OSDEligibleParam = "osd_eligible"
	// GetDisabledParam parameter
	GetDisabledParam = "get_disabled"
	// ImpactingParam parameter used to show/hide recommendations not hitting any clusters
	ImpactingParam = "impacting"
	// RuleIDParamName parameter name in the URL
	RuleIDParamName = "rule_id"
)

func readRuleIDWithErrorKey(writer http.ResponseWriter, request *http.Request) (ctypes.RuleID, ctypes.ErrorKey, error) {
	ruleIDWithErrorKey, err := httputils.GetRouterParam(request, RuleIDParamName)
	if err != nil {
		const message = "unable to get rule id"
		log.Error().Err(err).Msg(message)
		handleServerError(writer, err)
		return ctypes.RuleID(""), ctypes.ErrorKey(""), err
	}

	ruleID, errorKey, err := types.RuleIDWithErrorKeyFromCompositeRuleID(ctypes.RuleID(ruleIDWithErrorKey))
	if err != nil {
		handleServerError(writer, &RouterParsingError{
			paramName:  RuleIDParamName,
			paramValue: ruleIDWithErrorKey,
			errString:  err.Error(),
		})
		return ctypes.RuleID(""), ctypes.ErrorKey(""), err
	}

	return ruleID, errorKey, nil
}

func readCompositeRuleID(request *http.Request) (
	ruleID ctypes.RuleID,
	err error,
) {
	ruleIDParam, err := httputils.GetRouterParam(request, RuleIDParamName)
	if err != nil {
		const message = "unable to get rule id"
		log.Error().Err(err).Msg(message)
		return
	}

	compositeRuleIDValidator := regexp.MustCompile(`^([a-zA-Z_0-9.]+)[|]([a-zA-Z_0-9.]+)$`)
	isCompositeRuleIDValid := compositeRuleIDValidator.MatchString(ruleIDParam)

	if !isCompositeRuleIDValid {
		msg := fmt.Errorf("invalid composite rule ID. Must be in the format 'rule.plugin.module|ERROR_KEY'")
		err = &RouterParsingError{
			paramName:  RuleIDParamName,
			paramValue: ruleIDParam,
			errString:  msg.Error(),
		}
		log.Error().Err(err)
		return
	}

	ruleID = ctypes.RuleID(ruleIDParam)
	return
}

func (server HTTPServer) readParamsGetRecommendations(writer http.ResponseWriter, request *http.Request) (
	userID ctypes.UserID,
	orgID ctypes.OrgID,
	impactingFlag types.ImpactingFlag,
	err error,
) {

	orgID, userID, err = server.GetCurrentOrgIDUserIDFromToken(request)
	if err != nil {
		log.Err(err).Msg(orgIDTokenError)
		handleServerError(writer, err)
		return
	}

	impactingParam := request.URL.Query().Get(ImpactingParam)
	if impactingParam == "" {
		// impacting control flag is missing, display all recommendations
		impactingFlag = IncludingImpacting
		return
	}

	impactingParamBool, err := readImpactingParam(request)
	if err != nil {
		log.Err(err).Msgf("Error parsing `%s` URL parameter.", ImpactingParam)
		handleServerError(writer, &RouterParsingError{
			paramName: ImpactingParam,
			errString: "Unparsable boolean value",
		})
		return
	}

	if impactingParamBool {
		// param impacting=true means to only include impacting recommendations
		impactingFlag = OnlyImpacting
	} else {
		// param impacting=false means to return all rules that aren't impacting any clusters
		impactingFlag = ExcludingImpacting
	}

	return
}

// readQueryParam return the value of the parameter in the query. If not found, defaults to false
func readQueryBoolParam(name string, defaultValue bool, request *http.Request) (bool, error) {
	value := request.URL.Query().Get(name)
	if value == "" {
		return defaultValue, nil
	}
	return strconv.ParseBool(value)
}

// readGetDisabledParam returns the value of the "get_disabled" parameter in query
// if available
func readGetDisabledParam(request *http.Request) (bool, error) {
	return readQueryBoolParam(GetDisabledParam, false, request)
}

// readOSDEligibleParam returns the value of the "osd_eligible" parameter in query
// if available
func readOSDEligible(request *http.Request) (bool, error) {
	return readQueryBoolParam(OSDEligibleParam, false, request)
}

// readImpactingParam returns the value of the "impacting" parameter in query if available
func readImpactingParam(request *http.Request) (bool, error) {
	return readQueryBoolParam(ImpactingParam, true, request)
}
