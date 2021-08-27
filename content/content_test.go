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

	local_types "github.com/RedHatInsights/insights-results-smart-proxy/types"

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

func TestResetContentWhenUpdating(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		defer helpers.CleanAfterGock(t)
		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.ContentBaseEndpoint, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: ics_server.AllContentEndpoint,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.MustGobSerialize(t, testdata.RuleContentDirectory5Rules),
		})

		content.UpdateContent(helpers.DefaultServicesConfig)

		helpers.GockExpectAPIRequest(t, helpers.DefaultServicesConfig.ContentBaseEndpoint, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: ics_server.AllContentEndpoint,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.MustGobSerialize(t, testdata.RuleContentDirectory3Rules),
		})

		content.UpdateContent(helpers.DefaultServicesConfig)

		ruleIDs := content.GetRuleIDs()
		assert.Equal(t, len(testdata.RuleContentDirectory3Rules.Rules), len(ruleIDs))
	}, testTimeout)
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

func TestFetchRuleContent_OSDEligibleNotRequiredAdmin(t *testing.T) {
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

		rule := testdata.RuleOnReport1
		ruleContent, success, osdFiltered := content.FetchRuleContent(rule, true)
		assert.True(t, success)
		assert.False(t, osdFiltered)
		assert.NotNil(t, ruleContent)

		ruleID := testdata.RuleOnReport1.Module
		errorKey := testdata.RuleOnReport1.ErrorKey
		ruleWithContent, _ := content.GetRuleWithErrorKeyContent(ruleID, errorKey)
		ruleWithContentResponse := &local_types.RuleWithContentResponse{
			CreatedAt:       ruleWithContent.PublishDate.UTC().Format(time.RFC3339),
			Description:     ruleWithContent.Description,
			ErrorKey:        errorKey,
			Generic:         ruleWithContent.Generic,
			Reason:          ruleWithContent.Reason,
			Resolution:      ruleWithContent.Resolution,
			MoreInfo:        ruleWithContent.MoreInfo,
			TotalRisk:       ruleWithContent.TotalRisk,
			RiskOfChange:    ruleWithContent.RiskOfChange,
			RuleID:          ruleID,
			TemplateData:    rule.TemplateData,
			Tags:            ruleWithContent.Tags,
			UserVote:        rule.UserVote,
			Disabled:        rule.Disabled,
			DisableFeedback: rule.DisableFeedback,
			DisabledAt:      rule.DisabledAt,
			Internal:        ruleWithContent.Internal,
		}

		assert.Equal(t, ruleWithContentResponse, ruleContent)

	}, testTimeout)
}

func TestFetchRuleContent_NotOSDEligible(t *testing.T) {
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

		rule := testdata.RuleOnReport1
		ruleContent, success, osdFiltered := content.FetchRuleContent(rule, false)
		assert.True(t, success)
		assert.False(t, osdFiltered)
		assert.NotNil(t, ruleContent)

		ruleID := testdata.RuleOnReport1.Module
		errorKey := testdata.RuleOnReport1.ErrorKey
		ruleWithContent, _ := content.GetRuleWithErrorKeyContent(ruleID, errorKey)
		ruleWithContentResponse := &local_types.RuleWithContentResponse{
			CreatedAt:       ruleWithContent.PublishDate.UTC().Format(time.RFC3339),
			Description:     ruleWithContent.Description,
			ErrorKey:        errorKey,
			Generic:         ruleWithContent.Generic,
			Reason:          ruleWithContent.Reason,
			Resolution:      ruleWithContent.Resolution,
			MoreInfo:        ruleWithContent.MoreInfo,
			TotalRisk:       ruleWithContent.TotalRisk,
			RiskOfChange:    ruleWithContent.RiskOfChange,
			RuleID:          ruleID,
			TemplateData:    rule.TemplateData,
			Tags:            ruleWithContent.Tags,
			UserVote:        rule.UserVote,
			Disabled:        rule.Disabled,
			DisableFeedback: rule.DisableFeedback,
			DisabledAt:      rule.DisabledAt,
			Internal:        ruleWithContent.Internal,
		}

		assert.Equal(t, ruleWithContentResponse, ruleContent)

	}, testTimeout)
}

func TestFetchRuleContent_DisabledRuleExist(t *testing.T) {
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

		var rule = types.RuleOnReport{
			Module:          testdata.Rule1.Module,
			ErrorKey:        testdata.RuleErrorKey1.ErrorKey,
			UserVote:        types.UserVoteNone,
			Disabled:        true,
			DisableFeedback: "",
			DisabledAt:      "",
			TemplateData:    testdata.Rule1ExtraData,
		}

		ruleContent, success, osdFiltered := content.FetchRuleContent(rule, false)
		assert.True(t, success)
		assert.False(t, osdFiltered)
		assert.NotNil(t, ruleContent)

	}, testTimeout)
}

func TestFetchRuleContent_RuleDoesNotExist(t *testing.T) {
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

		var rule = types.RuleOnReport{
			Module:          types.RuleID("ccx_rules_ocp.deprecated_a_long_time_ago_should_not_exist"),
			ErrorKey:        testdata.RuleErrorKey1.ErrorKey,
			UserVote:        types.UserVoteNone,
			Disabled:        false,
			DisableFeedback: "",
			DisabledAt:      "",
			TemplateData:    nil,
		}

		ruleContent, success, _ := content.FetchRuleContent(rule, false)
		assert.False(t, success)
		assert.Nil(t, ruleContent)

	}, testTimeout)
}

func TestUpdateContentInvalidStatus(t *testing.T) {
	defer content.ResetContent()

	ruleContent := testdata.RuleContent4
	ek := ruleContent.ErrorKeys[testdata.ErrorKey4]
	ek.Metadata.Status = "foo"
	ruleContent.ErrorKeys[testdata.ErrorKey4] = ek

	ruleContentDirectory := types.RuleContentDirectory{
		Config: types.GlobalRuleConfig{
			Impact: testdata.ImpactStrToInt,
		},
		Rules: map[string]types.RuleContent{
			"rc4": ruleContent,
		},
	}

	content.LoadRuleContent(&ruleContentDirectory)

	_, err := content.GetRuleWithErrorKeyContent(testdata.Rule4ID, testdata.ErrorKey4)
	assert.NotNil(t, err)
}

func TestUpdateContentMissingStatus(t *testing.T) {
	defer content.ResetContent()

	ruleContent := testdata.RuleContent4
	ek := ruleContent.ErrorKeys[testdata.ErrorKey4]
	ek.Metadata.Status = ""
	ruleContent.ErrorKeys[testdata.ErrorKey4] = ek

	ruleContentDirectory := types.RuleContentDirectory{
		Config: types.GlobalRuleConfig{
			Impact: testdata.ImpactStrToInt,
		},
		Rules: map[string]types.RuleContent{
			"rc4": ruleContent,
		},
	}

	content.LoadRuleContent(&ruleContentDirectory)

	_, err := content.GetRuleWithErrorKeyContent(testdata.Rule4ID, testdata.ErrorKey4)
	helpers.FailOnError(t, err)
}

func TestUpdateContentInvalidPublishDate(t *testing.T) {
	defer content.ResetContent()

	ruleContent := testdata.RuleContent4
	ek := ruleContent.ErrorKeys[testdata.ErrorKey4]
	ek.Metadata.PublishDate = "invalid date"
	ruleContent.ErrorKeys[testdata.ErrorKey4] = ek

	ruleContentDirectory := types.RuleContentDirectory{
		Config: types.GlobalRuleConfig{
			Impact: testdata.ImpactStrToInt,
		},
		Rules: map[string]types.RuleContent{
			"rc4": ruleContent,
		},
	}

	content.LoadRuleContent(&ruleContentDirectory)

	_, err := content.GetRuleWithErrorKeyContent(testdata.Rule4ID, testdata.ErrorKey4)
	assert.NotNil(t, err)
}

func TestUpdateContentMissingPublishDate(t *testing.T) {
	defer content.ResetContent()

	ruleContent := testdata.RuleContent4
	ek := ruleContent.ErrorKeys[testdata.ErrorKey4]
	ek.Metadata.PublishDate = ""
	ruleContent.ErrorKeys[testdata.ErrorKey4] = ek

	ruleContentDirectory := types.RuleContentDirectory{
		Config: types.GlobalRuleConfig{
			Impact: testdata.ImpactStrToInt,
		},
		Rules: map[string]types.RuleContent{
			"rc4": ruleContent,
		},
	}

	content.LoadRuleContent(&ruleContentDirectory)

	_, err := content.GetRuleWithErrorKeyContent(testdata.Rule4ID, testdata.ErrorKey4)
	helpers.FailOnError(t, err)
}
