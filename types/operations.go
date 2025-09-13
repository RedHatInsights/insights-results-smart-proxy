// Copyright 2021 Red Hat, Inc
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

package types

import (
	"fmt"
	"regexp"
	"strings"

	ctypes "github.com/RedHatInsights/insights-results-types"
	"github.com/rs/zerolog/log"
)

// GetClusterNames extract the ClusterName from an array of ClusterInfo
func GetClusterNames(clustersInfo []ClusterInfo) []ClusterName {
	retval := make([]ClusterName, len(clustersInfo))

	for i, info := range clustersInfo {
		retval[i] = info.ID
	}

	return retval
}

// ClusterInfoArrayToMap convert an array of ClusterInfo elements into a map using
// ClusterName as key
func ClusterInfoArrayToMap(clustersInfo []ClusterInfo) (retval map[ctypes.ClusterName]ClusterInfo) {
	retval = make(map[ctypes.ClusterName]ClusterInfo)

	for _, clusterInfo := range clustersInfo {
		retval[clusterInfo.ID] = clusterInfo
	}

	return
}

// RuleIDWithErrorKeyFromCompositeRuleID get a pair RuleID + ErrorKey from a composite rule identifier
func RuleIDWithErrorKeyFromCompositeRuleID(compositeRuleID ctypes.RuleID) (ctypes.RuleID, ctypes.ErrorKey, error) {
	splitedRuleID := strings.Split(string(compositeRuleID), "|")

	if len(splitedRuleID) != 2 {
		err := fmt.Errorf("invalid rule ID, it must contain only rule ID and error key separated by |")
		log.Warn().Msgf("Error during parsing param 'rule_id' with value '%s'. Error: '%s'", string(compositeRuleID), err.Error())
		return ctypes.RuleID(""), ctypes.ErrorKey(""), err
	}

	IDValidator := regexp.MustCompile(`^[a-zA-Z_0-9.]+$`)

	isRuleIDValid := IDValidator.MatchString(splitedRuleID[0])
	isErrorKeyValid := IDValidator.MatchString(splitedRuleID[1])

	if !isRuleIDValid || !isErrorKeyValid {
		err := fmt.Errorf("invalid rule ID, each part of ID must contain only latin characters, number, underscores or dots")
		log.Warn().Msgf("Error during parsing param 'rule_id' with value '%s'. Error: '%s'", string(compositeRuleID), err.Error())
		return ctypes.RuleID(""), ctypes.ErrorKey(""), err
	}

	return ctypes.RuleID(splitedRuleID[0]), ctypes.ErrorKey(splitedRuleID[1]), nil
}
