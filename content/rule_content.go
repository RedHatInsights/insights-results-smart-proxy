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

// Package content provides API to get rule's content by its `rule id` and `error key`.
// It takes all the work of caching rules taken from content service
package content

import (
	"github.com/RedHatInsights/insights-results-smart-proxy/types"
	ctypes "github.com/RedHatInsights/insights-results-types"
)

func (s *RulesWithContentStorage) getRuleContent(ruleID ctypes.RuleID) (*ctypes.RuleContent, bool) {
	res, found := s.rules[ruleID]
	return res, found
}

// RuleContentToV1 parses insights-results-types.RuleContent to RuleContentV1
func RuleContentToV1(res *ctypes.RuleContent) types.RuleContentV1 {
	resV1 := types.RuleContentV1{}
	resV1.Plugin = res.Plugin
	resV1.Generic = res.Generic
	resV1.Summary = res.Summary
	resV1.Resolution = res.Resolution
	resV1.MoreInfo = res.MoreInfo
	resV1.Reason = res.Reason
	resV1.HasReason = res.HasReason
	resV1.ErrorKeys = map[string]types.RuleErrorKeyContentV1{}
	for k, elem := range res.ErrorKeys {
		resV1.ErrorKeys[k] = types.RuleErrorKeyContentV1{
			Metadata: types.ErrorKeyMetadataV1{
				Description: elem.Metadata.Description,
				Impact:      elem.Metadata.Impact.Name,
				Likelihood:  elem.Metadata.Likelihood,
				PublishDate: elem.Metadata.PublishDate,
				Status:      elem.Metadata.Status,
				Tags:        elem.Metadata.Tags,
			},
			TotalRisk:  elem.TotalRisk,
			Generic:    elem.Generic,
			Summary:    elem.Summary,
			Resolution: elem.Resolution,
			MoreInfo:   elem.MoreInfo,
			Reason:     elem.Reason,
			HasReason:  elem.HasReason,
		}
	}
	return resV1
}

// RuleContentToV2 parses insights-results-types.RuleContent to RuleContentV2
func RuleContentToV2(res *ctypes.RuleContent) types.RuleContentV2 {
	resV2 := types.RuleContentV2{}
	resV2.Plugin = res.Plugin
	resV2.Generic = res.Generic
	resV2.Summary = res.Summary
	resV2.Resolution = res.Resolution
	resV2.MoreInfo = res.MoreInfo
	resV2.Reason = res.Reason
	resV2.HasReason = res.HasReason
	resV2.ErrorKeys = map[string]types.RuleErrorKeyContentV2{}
	for k, elem := range res.ErrorKeys {
		resV2.ErrorKeys[k] = types.RuleErrorKeyContentV2{
			Metadata: types.ErrorKeyMetadataV2{
				Description: elem.Metadata.Description,
				Impact:      elem.Metadata.Impact.Impact,
				Likelihood:  elem.Metadata.Likelihood,
				PublishDate: elem.Metadata.PublishDate,
				Status:      elem.Metadata.Status,
				Tags:        elem.Metadata.Tags,
			},
			TotalRisk:  elem.TotalRisk,
			Generic:    elem.Generic,
			Summary:    elem.Summary,
			Resolution: elem.Resolution,
			MoreInfo:   elem.MoreInfo,
			Reason:     elem.Reason,
			HasReason:  elem.HasReason,
		}
	}
	return resV2
}
