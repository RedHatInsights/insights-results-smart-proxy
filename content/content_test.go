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
package content_test

import (
	"testing"

	cs_content "github.com/RedHatInsights/insights-content-service/content"
	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"

	"github.com/RedHatInsights/insights-results-smart-proxy/content"
)

func TestUpdateContentBadTime(t *testing.T) {
	// using testdata.RuleContent4 because contains datetime in a different format
	ruleContentDirectory := cs_content.RuleContentDirectory{
		Config: cs_content.GlobalRuleConfig{
			Impact: testdata.ImpactStrToInt,
		},
		Rules: map[string]cs_content.RuleContent{
			"rc4": testdata.RuleContent4,
		},
	}

	content.LoadRuleContent(&ruleContentDirectory)
	close(content.RuleContentDirectoryReady)
	_, err := content.GetRuleWithErrorKeyContent(
		testdata.Rule4ID, testdata.ErrorKey4)
	helpers.FailOnError(t, err)
}
