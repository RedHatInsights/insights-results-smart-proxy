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

	"github.com/RedHatInsights/insights-operator-utils/collections"
	"github.com/RedHatInsights/insights-operator-utils/types"
	local_types "github.com/RedHatInsights/insights-results-smart-proxy/types"
	"github.com/rs/zerolog/log"
)

const internalRuleStr = "internal"

var (
	timeParseFormats = []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
	}
)

// TODO: consider moving parsing to content service

// LoadRuleContent loads the parsed rule content into the storage
func LoadRuleContent(contentDir *types.RuleContentDirectory) {
	for i, rule := range contentDir.Rules {
		ruleID := types.RuleID(rule.Plugin.PythonModule)

		for errorKey, errorProperties := range rule.ErrorKeys {
			impact, found := contentDir.Config.Impact[errorProperties.Metadata.Impact]
			if !found {
				log.Error().Msgf(`impact "%v" doesn't have integer representation' (skipping)`, impact)
				continue
			}

			isActive, success := getActiveStatus(errorProperties.Metadata.Status)
			if success != true {
				log.Error().Msgf(`fatal: rule ID %v with key %v has invalid status`, ruleID, errorKey)
				return
			}

			publishDate, err := timeParse(errorProperties.Metadata.PublishDate)
			if err != nil {
				log.Error().Msgf(`fatal: rule ID %v with key %v has improper datetime attribute`, ruleID, errorKey)
				return
			}

			totalRisk := calculateTotalRisk(impact, errorProperties.Metadata.Likelihood)

			ruleTmp := contentDir.Rules[i]
			if ruleTmpErrorKey, ok := ruleTmp.ErrorKeys[errorKey]; ok {
				ruleTmpErrorKey.TotalRisk = totalRisk
				ruleTmp.ErrorKeys[errorKey] = ruleTmpErrorKey
			}
			rulesWithContentStorage.SetRule(ruleID, ruleTmp)

			rulesWithContentStorage.SetRuleWithContent(ruleID, types.ErrorKey(errorKey), &local_types.RuleWithContent{
				Module:          ruleID,
				Name:            rule.Plugin.Name,
				Summary:         rule.Summary,
				Reason:          rule.Reason,
				Resolution:      rule.Resolution,
				MoreInfo:        rule.MoreInfo,
				ErrorKey:        types.ErrorKey(errorKey),
				Description:     errorProperties.Metadata.Description,
				TotalRisk:       totalRisk,
				RiskOfChange:    calculateRiskOfChange(impact, errorProperties.Metadata.Likelihood),
				PublishDate:     publishDate,
				Active:          isActive,
				Internal:        IsRuleInternal(ruleID),
				Generic:         errorProperties.Generic,
				Tags:            errorProperties.Metadata.Tags,
				NotRequireAdmin: collections.StringInSlice("osd_customer", errorProperties.Metadata.Tags),
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

func timeParse(value string) (time.Time, error) {
	var err error
	for _, datetimeLayout := range timeParseFormats {
		parsedDate, err := time.Parse(datetimeLayout, value)

		if err == nil {
			return parsedDate, nil
		}

		log.Info().Msgf(
			`unable to parse time "%v" using layout "%v"`,
			value, datetimeLayout,
		)
	}

	log.Error().Err(err)

	return time.Time{}, err
}

// Reads Status string, first returned bool is active status, second bool is a success check
func getActiveStatus(status string) (bool, bool) {
	var isActive, success bool

	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active":
		isActive = true
		success = true
	case "inactive":
		isActive = false
		success = true
	default:
		log.Error().Msgf("invalid rule error key status: '%s'", status)
		success = false
	}

	return isActive, success
}

// IsRuleInternal tries to look for the word "internal" in the ruleID / rule module,
// because it's currently not specified anywhere on it's own
// TODO: add field indicating restricted/internal status to one of Rule structs in content-service
func IsRuleInternal(ruleID types.RuleID) bool {
	splitRuleID := strings.Split(string(ruleID), ".")
	for _, ruleIDPart := range splitRuleID {
		if ruleIDPart == internalRuleStr {
			return true
		}
	}
	return false
}
