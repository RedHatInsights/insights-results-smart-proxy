// Copyright 2021, 2022, 2023 Red Hat, Inc
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
	"strings"

	"github.com/RedHatInsights/insights-operator-utils/generators"
	types "github.com/RedHatInsights/insights-results-types"
	"github.com/rs/zerolog/log"
)

const (
	ruleIDStr   = "ruleID"
	errorKeyStr = "errorKey"
	urlStr      = "url"
	reportStr   = "report"
)

func logClusterInfos(orgID types.OrgID, clusterID types.ClusterName, ruleHits []types.RuleOnReport) {
	logMessage := fmt.Sprintf("rule hits for %d.%s:", orgID, clusterID)
	for _, ruleHit := range ruleHits {
		logMessage += fmt.Sprintf("\n\trule: %s; error key: %s", ruleHit.Module, ruleHit.ErrorKey)
	}
	log.Debug().Msg(logMessage)
}

func logClusterInfo(orgID types.OrgID, clusterID types.ClusterName, response *types.RuleOnReport) {
	logClusterInfos(orgID, clusterID, []types.RuleOnReport{*response})
}

func logClustersReport(orgID types.OrgID, reports map[types.ClusterName]json.RawMessage) {
	var report types.ReportRules
	for clusterName, jsonReport := range reports {
		err := json.Unmarshal(jsonReport, &report)
		if err != nil {
			log.Error().
				Err(err).
				Int(orgIDTag, int(orgID)).
				Str(clusterIDTag, string(clusterName)).
				Msgf("can't log report for cluster. raw json: [%v]", string(jsonReport))
			continue
		}
		logClusterInfos(orgID, clusterName, report.HitRules)
	}
}

// generateCompositeRuleIDFromDisabled trims ".report" from given disabled rule module and generates composite rule ID
func generateCompositeRuleIDFromDisabled(disabledRule types.DisabledRule) (
	compositeRuleID types.RuleID, err error,
) {
	// the records in v1-related enable/disable DB tables contain ".report" suffix which needs to be
	// (hopefully) temporarily trimmed for v2-related functionality
	trimmedRuleModule := strings.TrimSuffix(string(disabledRule.RuleID), dotReport)

	compositeRuleID, err = generators.GenerateCompositeRuleID(
		types.RuleFQDN(trimmedRuleModule),
		disabledRule.ErrorKey,
	)
	return
}
