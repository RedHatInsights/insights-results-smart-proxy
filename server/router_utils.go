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
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// getRouterParam retrieves parameter from URL like `/organization/{org_id}`
func getRouterParam(request *http.Request, paramName string) (string, error) {
	// try to read value for parameter with given name
	value, found := mux.Vars(request)[paramName]
	// check if parameter with given name has been found
	if !found {
		return "", &RouterMissingParamError{paramName: paramName}
	}

	// everything's ok, we have found the parameter and its value
	return value, nil
}

// getRouterPositiveIntParam retrieves parameter from URL like `/organization/{org_id}`
// and check it for being valid and positive integer, otherwise returns error
func getRouterPositiveIntParam(request *http.Request, paramName string) (uint64, error) {
	value, err := getRouterParam(request, paramName)
	// check if parameter with given name has been found
	if err != nil {
		return 0, err
	}

	// try to parse parameter value as positive integer
	uintValue, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, &RouterParsingError{
			paramName:  paramName,
			paramValue: value,
			errString:  "unsigned integer expected",
		}
	}

	// zero is in range of type uint, but we really need positive/natural
	// numbers
	if uintValue == 0 {
		return 0, &RouterParsingError{
			paramName:  paramName,
			paramValue: value,
			errString:  "positive value expected",
		}
	}

	return uintValue, nil
}

// validateClusterName checks that the cluster name is a valid UUID.
// Converted cluster name is returned if everything is okay, otherwise an error is returned.
func validateClusterName(clusterName string) (types.ClusterName, error) {
	// cluster name is in UUID format
	if _, err := uuid.Parse(clusterName); err != nil {
		message := fmt.Sprintf("invalid cluster name: '%s'. Error: %s", clusterName, err.Error())

		log.Error().Err(err).Msg(message)

		return "", &RouterParsingError{
			paramName: "cluster", paramValue: clusterName, errString: err.Error(),
		}
	}

	return types.ClusterName(clusterName), nil
}

// splitRequestParamArray takes a single HTTP request parameter and splits it
// into a slice of strings. This assumes that the parameter is a comma-separated array.
func splitRequestParamArray(arrayParam string) []string {
	return strings.Split(arrayParam, ",")
}

// handleOrgIDError is handler for situation when organization ID is not
// provided by HTTP request or it has improper format
func handleOrgIDError(writer http.ResponseWriter, err error) {
	log.Error().Err(err).Msg("Error getting organization ID from request")
	handleServerError(writer, err)
}

// handleClusterNameError is handler for situation when cluster name is not
// provided by HTTP request or it has improper format
func handleClusterNameError(writer http.ResponseWriter, err error) {
	log.Error().Msg(err.Error())

	// query parameter 'cluster' can't be found in request, which might be
	// caused by issue in Gorilla mux
	// (not on client side), but let's assume it won't :)
	handleServerError(writer, err)
}

// readClusterName function retrieves cluster name from request
// if it's not possible, it writes http error to the writer and returns error
func readClusterName(writer http.ResponseWriter, request *http.Request) (types.ClusterName, error) {
	clusterName, err := getRouterParam(request, "cluster")
	// check if parameter with cluster name has been found
	// and has correct format
	if err != nil {
		handleClusterNameError(writer, err)
		return "", err
	}

	validatedClusterName, err := validateClusterName(clusterName)
	if err != nil {
		handleClusterNameError(writer, err)
		return "", err
	}
	return validatedClusterName, nil
}

// readOrganizationID retrieves organization id from request
// if it's not possible, it writes http error to the writer and returns error
func readOrganizationID(writer http.ResponseWriter, request *http.Request, auth bool) (types.OrgID, error) {
	organizationID, err := getRouterPositiveIntParam(request, "organization")
	// check if parameter with organization ID has been found
	// and has correct format
	if err != nil {
		handleOrgIDError(writer, err)
		return 0, err
	}
	err = checkPermissions(writer, request, types.OrgID(organizationID), auth)

	return types.OrgID(organizationID), err
}

// checkPermissions function checks if the provided organization ID matches
// with the one of the authenticated user when the authentication is enabled
func checkPermissions(writer http.ResponseWriter, request *http.Request, orgID types.OrgID, auth bool) error {
	identityContext := request.Context().Value(types.ContextKeyUser)
	if identityContext != nil && auth {
		identity := identityContext.(types.Identity)
		if identity.Internal.OrgID != orgID {
			const message = "You have no permissions to get or change info about this organization"
			log.Error().Msg(message)
			handleServerError(writer, &types.ItemNotFoundError{ItemID: orgID})
			return errors.New(message)
		}
	}
	return nil
}

// readClusterNames does the same as `readClusterName`, except for multiple clusters.
func readClusterNames(writer http.ResponseWriter, request *http.Request) ([]types.ClusterName, error) {
	clusterNamesParam, err := getRouterParam(request, "clusters")
	if err != nil {
		message := fmt.Sprintf("Cluster names are not provided %v", err.Error())
		log.Error().Msg(message)

		handleServerError(writer, err)

		return []types.ClusterName{}, err
	}

	clusterNamesConverted := make([]types.ClusterName, 0)
	for _, clusterName := range splitRequestParamArray(clusterNamesParam) {
		convertedName, err := validateClusterName(clusterName)
		if err != nil {
			handleServerError(writer, err)
			return []types.ClusterName{}, err
		}

		clusterNamesConverted = append(clusterNamesConverted, convertedName)
	}

	return clusterNamesConverted, nil
}

// readOrganizationIDs does the same as `readOrganizationID`, except for multiple organizations.
func readOrganizationIDs(writer http.ResponseWriter, request *http.Request) ([]types.OrgID, error) {
	organizationsParam, err := getRouterParam(request, "organizations")
	if err != nil {
		handleOrgIDError(writer, err)
		return []types.OrgID{}, err
	}

	organizationsConverted := make([]types.OrgID, 0)
	for _, orgStr := range splitRequestParamArray(organizationsParam) {
		orgInt, err := strconv.ParseUint(orgStr, 10, 64)
		if err != nil {
			handleServerError(writer, &RouterParsingError{
				paramName:  "organizations",
				paramValue: orgStr,
				errString:  "integer array expected",
			})
			return []types.OrgID{}, err
		}
		organizationsConverted = append(organizationsConverted, types.OrgID(orgInt))
	}

	return organizationsConverted, nil
}

func readRuleIDWithErrorKey(writer http.ResponseWriter, request *http.Request) (types.RuleID, types.ErrorKey, error) {
	ruleIDWithErrorKey, err := getRouterParam(request, "rule_id")
	if err != nil {
		const message = "unable to get rule id"
		log.Error().Err(err).Msg(message)
		handleServerError(writer, err)
		return types.RuleID(0), types.ErrorKey(0), err
	}

	splitedRuleID := strings.Split(string(ruleIDWithErrorKey), "|")

	if len(splitedRuleID) != 2 {
		err = fmt.Errorf("invalid rule ID, it must contain only rule ID and error key separated by |")
		log.Error().Err(err)
		handleServerError(writer, &RouterParsingError{
			paramName:  "rule_id",
			paramValue: ruleIDWithErrorKey,
			errString:  err.Error(),
		})
		return types.RuleID(0), types.ErrorKey(0), err
	}

	IDValidator := regexp.MustCompile(`^[a-zA-Z_0-9.]+$`)

	isRuleIDValid := IDValidator.Match([]byte(splitedRuleID[0]))
	isErrorKeyValid := IDValidator.Match([]byte(splitedRuleID[1]))

	if !isRuleIDValid || !isErrorKeyValid {
		err = fmt.Errorf("invalid rule ID, each part of ID must contain only from latin characters, number, underscores or dots")
		log.Error().Err(err)
		handleServerError(writer, &RouterParsingError{
			paramName:  "rule_id",
			paramValue: ruleIDWithErrorKey,
			errString:  err.Error(),
		})
		return types.RuleID(0), types.ErrorKey(0), err
	}

	return types.RuleID(splitedRuleID[0]), types.ErrorKey(splitedRuleID[1]), nil
}

func readRuleID(writer http.ResponseWriter, request *http.Request) (types.RuleID, error) {
	ruleID, err := getRouterParam(request, "rule_id")
	if err != nil {
		const message = "unable to get rule id"
		log.Error().Err(err).Msg(message)
		handleServerError(writer, err)
		return types.RuleID(0), err
	}

	ruleIDValidator := regexp.MustCompile(`^[a-zA-Z_0-9.]+$`)

	isRuleIDValid := ruleIDValidator.Match([]byte(ruleID))

	if !isRuleIDValid {
		err = fmt.Errorf("invalid rule ID, it must contain only from latin characters, number, underscores or dots")
		log.Error().Err(err)
		handleServerError(writer, &RouterParsingError{
			paramName:  "rule_id",
			paramValue: ruleID,
			errString:  err.Error(),
		})
		return types.RuleID(0), err
	}

	return types.RuleID(ruleID), nil
}

func readErrorKey(writer http.ResponseWriter, request *http.Request) (types.ErrorKey, error) {
	errorKey, err := getRouterParam(request, "error_key")
	if err != nil {
		const message = "unable to get error_key"
		log.Error().Err(err).Msg(message)
		handleServerError(writer, err)
		return types.ErrorKey(0), err
	}

	return types.ErrorKey(errorKey), nil
}
