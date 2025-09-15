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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-operator-utils/responses"
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
	// RequestIDParam parameter name in the URL for request IDs
	RequestIDParam = "request_id"
	// NamespaceIDParam parameter name in the URL for namespace UUIDs
	NamespaceIDParam = "namespace"
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
		// Sending response without logging error in handleServerError as the function already sends warning logs
		respErr := responses.SendBadRequest(writer, fmt.Sprintf(
			"Error during parsing param '%s' with value '%s'. Error: '%s'",
			RuleIDParamName, ruleIDWithErrorKey, err.Error(),
		))
		if respErr != nil {
			log.Error().Err(respErr).Msg("Error sending bad request response")
		}
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
		log.Warn().Err(err).Msg(message)
		return
	}

	compositeRuleIDValidator := regexp.MustCompile(`^([a-zA-Z_0-9.]+)[|]([a-zA-Z_0-9.]+)$`)
	isCompositeRuleIDValid := compositeRuleIDValidator.MatchString(ruleIDParam)

	if !isCompositeRuleIDValid {
		msg := fmt.Errorf("invalid composite rule ID. Must be in the format 'rule.plugin.module|ERROR_KEY'")
		err = &RouterParsingError{
			ParamName:  RuleIDParamName,
			ParamValue: ruleIDParam,
			ErrString:  msg.Error(),
		}
		log.Warn().Err(err).Msg("invalid composite rule ID")
		return
	}

	ruleID = ctypes.RuleID(ruleIDParam)
	return
}

func (server *HTTPServer) readParamsGetRecommendations(writer http.ResponseWriter, request *http.Request) (
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
		log.Warn().Err(err).Msgf("Error parsing `%s` URL parameter.", ImpactingParam)
		handleServerError(writer, &RouterParsingError{
			ParamName: ImpactingParam,
			ErrString: "Unparsable boolean value",
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

// readUserAgentHeaderProduct returns the produt part of the standard User Agent syntax
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/User-Agent#syntax
func readUserAgentHeaderProduct(request *http.Request) (userAgentProduct string) {
	userAgent := request.Header.Get(userAgentHeader)
	if userAgent == "" {
		return
	}

	userAgentSplit := strings.Split(userAgent, "/")

	// we're only interested in the product name
	userAgentProduct = userAgentSplit[0]
	return
}

// ValidateRequestID checks that the request ID has proper format.
// Converted request ID is returned if everything is okay, otherwise an error is returned.
func ValidateRequestID(requestID string) (types.RequestID, error) {
	IDValidator := regexp.MustCompile(`^[a-zA-Z0-9]+$`)

	if !IDValidator.MatchString(requestID) {
		message := fmt.Sprintf("invalid request ID: '%s'", requestID)
		err := errors.New(message)
		log.Error().Err(err).Msg(message)
		return "", err
	}

	return types.RequestID(requestID), nil
}

// readRequestID retrieves request ID from request
// if it's not possible, it writes http error to the writer and returns error
func readRequestID(writer http.ResponseWriter, request *http.Request) (types.RequestID, error) {
	requestID, err := httputils.GetRouterParam(request, RequestIDParam)
	if err != nil {
		handleServerError(writer, err)
		return "", err
	}

	validatedRequestID, err := ValidateRequestID(requestID)
	if err != nil {
		err := &RouterParsingError{
			ParamName:  RequestIDParam,
			ParamValue: requestID,
			ErrString:  err.Error(),
		}
		handleServerError(writer, err)
		return "", err
	}

	return validatedRequestID, nil
}

func readRequestIDList(writer http.ResponseWriter, request *http.Request) (
	[]types.RequestID, error,
) {
	var requestList []string
	// check if there's any body provided in the request sent by client
	if request.ContentLength <= 0 {
		err := &NoBodyError{}
		handleServerError(writer, err)
		return nil, err
	}

	err := json.NewDecoder(request.Body).Decode(&requestList)
	if err != nil {
		log.Error().Err(err).Msg("unable to retrieve request ID list from request body")
		err := &BadBodyContent{}
		handleServerError(writer, err)
		return nil, err
	}

	validatedRequestList := make([]types.RequestID, len(requestList))
	for i, requestID := range requestList {
		validatedRequestID, err := ValidateRequestID(requestID)
		if err != nil {
			err := &RouterParsingError{
				ParamName:  RequestIDParam,
				ParamValue: requestID,
				ErrString:  err.Error(),
			}
			handleServerError(writer, err)
			return nil, err
		}

		validatedRequestList[i] = validatedRequestID
	}

	return validatedRequestList, nil
}

// readNamespace retrieves namespace UUID from request
// if it's not possible, it writes http error to the writer and returns error
func readNamespace(writer http.ResponseWriter, request *http.Request) (
	namespace types.Namespace, err error,
) {
	namespaceID, err := httputils.GetRouterParam(request, NamespaceIDParam)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	validatedNamespaceID, err := validateNamespaceID(namespaceID)
	if err != nil {
		err = &RouterParsingError{
			ParamName:  NamespaceIDParam,
			ParamValue: namespaceID,
			ErrString:  err.Error(),
		}
		handleServerError(writer, err)
		return
	}

	namespace.UUID = validatedNamespaceID

	return
}

// rule tests used by molodec use non-UUID namespace IDs, we must allow any garbage
// until that's resolved
func validateNamespaceID(namespace string) (string, error) {
	IDValidator := regexp.MustCompile(`^.{1,256}$`)

	if !IDValidator.MatchString(namespace) {
		message := fmt.Sprintf("invalid namespace ID: '%s'", namespace)
		err := errors.New(message)
		log.Error().Err(err).Msg(message)
		return "", err
	}

	return namespace, nil
}
