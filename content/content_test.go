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

package content_test

import (
	"net/http"
	"testing"
	"time"

	ics_server "github.com/RedHatInsights/insights-content-service/server"
	"github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
)

const (
	testTimeout = 10 * time.Second
)

func TestGetRuleContent(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.ContentBaseEndpoint, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: ics_server.AllContentEndpoint,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.MustGobSerialize(t, testdata.RuleContentDirectory3Rules),
		})

		content.UpdateContent(helpers.DefaultServicesConfig)

		ruleContent, err := content.GetRuleContent(testdata.Rule1ID)
		helpers.FailOnError(t, err)
		assert.NotNil(t, ruleContent)

		assert.Equal(t, testdata.RuleContent1, *ruleContent)
	}, testTimeout)
}

func TestGetRuleContent_CallMultipleTimes(t *testing.T) {
	const N = 10

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.ContentBaseEndpoint, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: ics_server.AllContentEndpoint,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.MustGobSerialize(t, testdata.RuleContentDirectory3Rules),
		})

		content.UpdateContent(helpers.DefaultServicesConfig)

		for i := 0; i < N; i++ {
			ruleContent, err := content.GetRuleContent(testdata.Rule1ID)
			helpers.FailOnError(t, err)
			assert.NotNil(t, ruleContent)

			assert.Equal(t, testdata.RuleContent1, *ruleContent)
		}
	}, testTimeout)
}

func TestUpdateContent_CallMultipleTimes(t *testing.T) {
	const N = 10

	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)

		for i := 0; i < N; i++ {
			helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.ContentBaseEndpoint, &helpers.APIRequest{
				Method:   http.MethodGet,
				Endpoint: ics_server.AllContentEndpoint,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       helpers.MustGobSerialize(t, testdata.RuleContentDirectory3Rules),
			})
		}

		for i := 0; i < N; i++ {
			content.UpdateContent(helpers.DefaultServicesConfig)
		}

		for i := 0; i < N; i++ {
			ruleContent, err := content.GetRuleContent(testdata.Rule1ID)
			helpers.FailOnError(t, err)
			assert.NotNil(t, ruleContent)

			assert.Equal(t, testdata.RuleContent1, *ruleContent)
		}
	}, testTimeout)
}

func TestUpdateContentBadTime(t *testing.T) {
	// using testdata.RuleContent4 because contains datetime in a different format
	ruleContentDirectory := types.RuleContentDirectory{
		Config: types.GlobalRuleConfig{
			Impact: testdata.ImpactStrToInt,
		},
		Rules: map[string]types.RuleContent{
			"rc4": testdata.RuleContent4,
		},
	}

	content.LoadRuleContent(&ruleContentDirectory)

	_, err := content.GetRuleWithErrorKeyContent(testdata.Rule4ID, testdata.ErrorKey4)
	helpers.FailOnError(t, err)
}

func TestResetContent(t *testing.T) {
	ruleIDs := content.GetRuleIDs()
	assert.NotEqual(t, 0, len(ruleIDs))
	content.ResetContent()

	ruleIDs = content.GetRuleIDs()
	assert.Equal(t, 0, len(ruleIDs))
}

func TestGetAllContent(t *testing.T) {
	defer content.ResetContent()

	content.LoadRuleContent(&testdata.RuleContentDirectory3Rules)
	rules := content.GetAllContent()
	assert.Equal(t, len(testdata.RuleContentDirectory3Rules.Rules), len(rules))
}
