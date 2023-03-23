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
	"fmt"
	"net/http"

	"github.com/RedHatInsights/insights-results-smart-proxy/content"

	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/rs/zerolog/log"
)

const (
	// responseDataError is used as the error message when the responses functions return an error
	responseDataError = "Unexpected error during response data encoding"

	orgIDTokenError             = "error retrieving orgID and userID from auth token"
	problemSendingResponseError = "problem sending response"
)

// RouterMissingParamError missing parameter in request
type RouterMissingParamError struct {
	paramName string
}

func (e *RouterMissingParamError) Error() string {
	return fmt.Sprintf("Missing required param from request: %v", e.paramName)
}

// RouterParsingError parsing error, for example string when we expected integer
type RouterParsingError struct {
	paramName  string
	paramValue interface{}
	errString  string
}

func (e *RouterParsingError) Error() string {
	return fmt.Sprintf(
		"Error during parsing param '%v' with value '%v'. Error: '%v'",
		e.paramName, e.paramValue, e.errString,
	)
}

// AuthenticationError happens during auth problems, for example malformed token
type AuthenticationError struct {
	errString string
}

func (e *AuthenticationError) Error() string {
	return e.errString
}

// NoBodyError error meaning that client didn't provide body when it's required
type NoBodyError struct{}

func (*NoBodyError) Error() string {
	return "client didn't provide request body"
}

// BadBodyContent error meaning that client didn't provide a meaningful body content when it's required
type BadBodyContent struct{}

func (*BadBodyContent) Error() string {
	return "client didn't provide a valid request body"
}

// ContentServiceUnavailableError error is used when the content service cannot be reached
type ContentServiceUnavailableError struct{}

func (*ContentServiceUnavailableError) Error() string {
	return "Content service is unreachable"
}

// AggregatorServiceUnavailableError error is used when the aggregator service cannot be reached
type AggregatorServiceUnavailableError struct{}

func (*AggregatorServiceUnavailableError) Error() string {
	return "Aggregator service is unreachable"
}

// UpgradesDataEngServiceUnavailableError error is used when the ccx-upgrades-data-eng service cannot be reached
type UpgradesDataEngServiceUnavailableError struct{}

func (*UpgradesDataEngServiceUnavailableError) Error() string {
	return "Upgrade Failure Prediction service is unreachable"
}

// AMSAPIUnavailableError error is used when AMS API is not available and is the only source of data
type AMSAPIUnavailableError struct{}

func (*AMSAPIUnavailableError) Error() string {
	return "AMS API is unreachable"
}

// ParamsParsingError error meaning that the cluster name cannot be handled
type ParamsParsingError struct{}

func (*ParamsParsingError) Error() string {
	return "the parameters contains invalid characters and cannot be used"
}

// handleServerError handles separate server errors and sends appropriate responses
func handleServerError(writer http.ResponseWriter, err error) {
	log.Error().Err(err).Msg("handleServerError()")

	var respErr error

	switch err := err.(type) {
	case *RouterMissingParamError, *RouterParsingError, *json.SyntaxError, *NoBodyError, *ParamsParsingError, *BadBodyContent:
		respErr = responses.SendBadRequest(writer, err.Error())
	case *json.UnmarshalTypeError:
		respErr = responses.SendBadRequest(writer, "bad type in json data")
	case *types.ItemNotFoundError:
		respErr = responses.SendNotFound(writer, err.Error())
	case *types.NoContentError:
		respErr = responses.SendNoContent(writer)
	case *AuthenticationError:
		respErr = responses.SendForbidden(writer, err.Error())
	case *ContentServiceUnavailableError, *AggregatorServiceUnavailableError,
		*AMSAPIUnavailableError, *content.RuleContentDirectoryTimeoutError,
		*UpgradesDataEngServiceUnavailableError:
		respErr = responses.SendServiceUnavailable(writer, err.Error())
	default:
		respErr = responses.SendInternalServerError(writer, "Internal Server Error")
	}

	if respErr != nil {
		log.Error().Err(respErr).Msg(responseDataError)
	}
}
