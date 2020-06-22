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

package content

import (
	"strings"
	"time"

	"github.com/RedHatInsights/insights-content-service/content"
	"github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/rs/zerolog/log"
)

// TODO: consider moving parsing to content service

// loadRuleContent loads the parsed rule content into the storage
func loadRuleContent(contentDir *content.RuleContentDirectory) {
	for _, rule := range contentDir.Rules {
		ruleID := types.RuleID(rule.Plugin.PythonModule)

		rulesWithContentStorage.SetRule(ruleID, rule)

		for errorKey, errorProperties := range rule.ErrorKeys {
			impact, found := contentDir.Config.Impact[errorProperties.Metadata.Impact]
			if !found {
				log.Error().Msgf(`impact "%v" doesn't have integer representation'`, impact)
				continue
			}
			var isActive bool
			switch strings.ToLower(strings.TrimSpace(errorProperties.Metadata.Status)) {
			case "active":
				isActive = true
			case "inactive":
				isActive = false
			default:
				log.Error().Msgf("invalid rule error key status: '%s'", errorProperties.Metadata.Status)
				return
			}

			publishDate, err := time.Parse(time.RFC3339, errorProperties.Metadata.PublishDate)
			if err != nil {
				log.Error().Msgf(
					`invalid to parse time "%v" using layout "%v"`,
					errorProperties.Metadata.PublishDate,
					time.RFC3339,
				)
				return
			}

			rulesWithContentStorage.SetRuleWithContent(ruleID, types.ErrorKey(errorKey), &types.RuleWithContent{
				Module:       ruleID,
				Name:         rule.Plugin.Name,
				Summary:      rule.Summary,
				Reason:       rule.Reason,
				Resolution:   rule.Resolution,
				MoreInfo:     rule.MoreInfo,
				ErrorKey:     types.ErrorKey(errorKey),
				Condition:    errorProperties.Metadata.Condition,
				Description:  errorProperties.Metadata.Description,
				TotalRisk:    calculateTotalRisk(impact, errorProperties.Metadata.Likelihood),
				RiskOfChange: calculateRiskOfChange(impact, errorProperties.Metadata.Likelihood),
				PublishDate:  publishDate,
				Active:       isActive,
				Generic:      errorProperties.Generic,
				Tags:         errorProperties.Metadata.Tags,
			})
		}
	}
}

// TODO: move to utils
func calculateTotalRisk(impact, likelihood int) int {
	return (impact + likelihood) / 2
}

// TODO: move to utils
func calculateRiskOfChange(impact, likelihood int) int {
	// TODO: actually calculate
	return 0
}

// TODO: move to utils
func commaSeparatedStrToTags(str string) []string {
	str = strings.TrimSpace(str)
	if len(str) == 0 {
		return []string{}
	}

	return strings.Split(str, ",")
}
