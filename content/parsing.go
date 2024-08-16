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
	"errors"
	"strings"
	"time"

	"github.com/RedHatInsights/insights-operator-utils/collections"
	ctypes "github.com/RedHatInsights/insights-results-types"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-smart-proxy/types"
)

const (
	internalRuleStr = "internal"
	ocsRuleStr      = "ocs"
	ruleIDStr       = "ruleID"
	errorKeyStr     = "errorKey"
)

var (
	timeParseFormats = []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
	}
)

// TODO: consider moving parsing to content service

// LoadRuleContent loads the parsed rule content into the storage
func LoadRuleContent(contentDir *ctypes.RuleContentDirectory) {
	s := getEmptyRulesWithContentMap()
	for i, rule := range contentDir.Rules {
		ruleID := ctypes.RuleID(rule.Plugin.PythonModule)

		for errorKey, errorProperties := range rule.ErrorKeys {
			impact := errorProperties.Metadata.Impact

			// we allow empty/missing, but not incorrect
			active, success, missing := getActiveStatus(errorProperties.Metadata.Status)
			if !success {
				log.Error().Interface(ruleIDStr, ruleID).Str(errorKeyStr, errorKey).Msg(`invalid status attribute`)
				continue
			} else if missing {
				log.Debug().Interface(ruleIDStr, ruleID).Str(errorKeyStr, errorKey).Msg(`missing status attribute`)
			}

			// we allow empty/missing, but not incorrect format
			publishDate, missing, err := timeParse(errorProperties.Metadata.PublishDate)
			if err != nil {
				log.Error().Interface(ruleIDStr, ruleID).Str(errorKeyStr, errorKey).Err(err).Msg(`improper publish_date attribute`)
				continue
			} else if missing {
				log.Debug().Interface(ruleIDStr, ruleID).Str(errorKeyStr, errorKey).Msg(`missing publish_date attribute`)
			}

			totalRisk := calculateTotalRisk(impact.Impact, errorProperties.Metadata.Likelihood)

			ruleTmp := contentDir.Rules[i]
			if ruleTmpErrorKey, ok := ruleTmp.ErrorKeys[errorKey]; ok {
				ruleTmpErrorKey.TotalRisk = totalRisk
				ruleTmp.ErrorKeys[errorKey] = ruleTmpErrorKey
			}
			// sets "plugin" level, containing usual fields + list of error keys
			s.SetRule(ruleID, &ruleTmp)

			s.SetRuleWithContent(ruleID, ctypes.ErrorKey(errorKey), &types.RuleWithContent{
				Module:         ruleID,
				Name:           rule.Plugin.Name,
				Generic:        errorProperties.Generic,
				Summary:        errorProperties.Summary,
				Reason:         errorProperties.Reason,
				Resolution:     errorProperties.Resolution,
				MoreInfo:       errorProperties.MoreInfo,
				ErrorKey:       ctypes.ErrorKey(errorKey),
				Description:    errorProperties.Metadata.Description,
				TotalRisk:      totalRisk,
				ResolutionRisk: errorProperties.Metadata.ResolutionRisk,
				Impact:         impact.Impact,
				Likelihood:     errorProperties.Metadata.Likelihood,
				PublishDate:    publishDate,
				Active:         active,
				Internal:       IsRuleInternal(ruleID),
				Tags:           errorProperties.Metadata.Tags,
				OSDCustomer:    collections.StringInSlice("osd_customer", errorProperties.Metadata.Tags),
			})
		}
	}
	rulesWithContentStorage = s
}

// According to rule content specification, it's explicitly defined as floor((impact + likelihood) / 2), which
// is the default behaviour in Go
func calculateTotalRisk(impact, likelihood int) int {
	return (impact + likelihood) / 2
}

// TODO: move to utils
func commaSeparatedStrToTags(str string) []string {
	str = strings.TrimSpace(str)
	if str == "" {
		return []string{}
	}

	return strings.Split(str, ",")
}

func timeParse(value string) (publishDate time.Time, missing bool, err error) {
	missing = false
	publishDate = time.Time{}

	if value == "" {
		missing = true
		return
	}

	for _, datetimeLayout := range timeParseFormats {
		publishDate, err = time.Parse(datetimeLayout, value)

		if err != nil {
			log.Info().Msgf(
				`unable to parse time "%v" using layout "%v"`,
				value, datetimeLayout,
			)
			continue
		}
		break
	}

	if err != nil {
		log.Error().Err(err).Msg("problem parsing publish_date")
		err = errors.New("invalid format of publish_date")
	}

	return
}

// Reads Status string, first returned bool is active status, second bool is a success check
func getActiveStatus(status string) (active, success, missing bool) {
	active, success, missing = false, false, false

	status = strings.ToLower(strings.TrimSpace(status))

	switch status {
	case "active":
		active = true
		success = true
	case "inactive":
		success = true
	case "":
		success = true
		missing = true
	default:
		log.Error().Str("status", status).Msg("invalid rule error key status")
	}

	return
}

// IsRuleInternal tries to look for the word "internal" in the ruleID / rule module,
// because it's currently not specified anywhere on it's own
func IsRuleInternal(ruleID ctypes.RuleID) bool {
	splitRuleID := strings.Split(string(ruleID), ".")
	return len(splitRuleID) > 1 &&
		(splitRuleID[1] == internalRuleStr ||
			splitRuleID[1] == ocsRuleStr)
}
