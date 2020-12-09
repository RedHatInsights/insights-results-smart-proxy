/*
Copyright © 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	iou_types "github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
	"github.com/RedHatInsights/insights-results-smart-proxy/types"
)

const (
	testTimeout            = 10 * time.Second
	internalTestRuleModule = "foo.rules.internal.bar"
)

// TODO: consider moving to data repo
var (
	badJWTAuthBearer  = "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2NvdW50X251bWJlciI6IjUyMTM0NzYiLCJvcmdfaWQiOiIxMjM0In0.Y9nNaZXbMEO6nz2EHNaCvHxPM0IaeT7GGR-T8u8h_nr_2b5dYsCQiZGzzkBupRJruHy9K6acgJ08JN2Q28eOAEVk_ZD2EqO43rSOS6oe8uZmVo-nCecdqovHa9PqW8RcZMMxVfGXednw82kKI8j1aT_nbJ1j9JZt3hnHM4wtqydelMij7zKyZLHTWFeZbDDCuEIkeWA6AdIBCMdywdFTSTsccVcxT2rgv4mKpxY1Fn6Vu_Xo27noZW88QhPTHbzM38l9lknGrvJVggrzMTABqWEXNVHbph0lXjPWsP7pe6v5DalYEBN2r3a16A6s3jPfI86cRC6_oeXotlW6je0iKQ"
	goodJWTAuthBearer = "Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhY2NvdW50X251bWJlciI6IjUyMTM0NzYiLCJvcmdfaWQiOiIxIiwianRpIjoiMDU0NDNiOTktZDgyNC00ODBiLWE0YmUtMzc5Nzc0MDVmMDkzIiwiaWF0IjoxNTk0MTI2MzQwLCJleHAiOjE1OTQxNDE4NDd9.pp32mPoypnRjOYE95SrBar0fdLS9t_hndOtP5qUvB-c"

	serverConfigJWT = server.Configuration{
		Address:                          ":8081",
		APIPrefix:                        "/api/v1/",
		APISpecFile:                      "openapi.json",
		Debug:                            true,
		Auth:                             true,
		AuthType:                         "jwt",
		UseHTTPS:                         false,
		EnableCORS:                       false,
		EnableInternalRulesOrganizations: false,
		InternalRulesOrganizations:       []iou_types.OrgID{1},
	}

	serverConfigInternalOrganizations1 = server.Configuration{
		Address:                          ":8081",
		APIPrefix:                        "/api/v1/",
		APISpecFile:                      "openapi.json",
		Debug:                            true,
		Auth:                             true,
		AuthType:                         "jwt",
		UseHTTPS:                         false,
		EnableCORS:                       false,
		EnableInternalRulesOrganizations: true,
		InternalRulesOrganizations:       []iou_types.OrgID{1},
	}

	// Same as previous one, but different InternalRulesOrganizations
	// This one won't match with the authentication token used
	serverConfigInternalOrganizations2 = server.Configuration{
		Address:                          ":8081",
		APIPrefix:                        "/api/v1/",
		APISpecFile:                      "openapi.json",
		Debug:                            true,
		Auth:                             true,
		AuthType:                         "jwt",
		UseHTTPS:                         false,
		EnableCORS:                       false,
		EnableInternalRulesOrganizations: true,
		InternalRulesOrganizations:       []iou_types.OrgID{2},
	}

	SmartProxyReportResponse1RuleNoContent = struct {
		Status string                  `json:"status"`
		Report *types.SmartProxyReport `json:"report"`
	}{
		Status: "ok",
		Report: &SmartProxyReport1RuleNoContent,
	}

	SmartProxyReportResponse3Rules = struct {
		Status string                  `json:"status"`
		Report *types.SmartProxyReport `json:"report"`
	}{
		Status: "ok",
		Report: &SmartProxyReport3Rules,
	}

	SmartProxyReport1RuleNoContent = types.SmartProxyReport{
		Meta: types.ReportResponseMeta{
			Count:         1,
			LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		},
		Data: []types.RuleWithContentResponse{},
	}
	SmartProxyReport3Rules = types.SmartProxyReport{
		Meta: types.ReportResponseMeta{
			Count:         3,
			LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		},
		Data: []types.RuleWithContentResponse{
			{
				RuleID:       testdata.Rule1.Module,
				ErrorKey:     testdata.RuleErrorKey1.ErrorKey,
				CreatedAt:    testdata.RuleErrorKey1.PublishDate.UTC().Format(time.RFC3339),
				Description:  testdata.RuleErrorKey1.Description,
				Generic:      testdata.RuleErrorKey1.Generic,
				Reason:       testdata.Rule1.Reason,
				Resolution:   testdata.Rule1.Resolution,
				TotalRisk:    calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood),
				RiskOfChange: 0,
				Disabled:     testdata.Rule1Disabled,
				UserVote:     types.UserVoteNone,
				TemplateData: testdata.Rule1ExtraData,
				Tags:         testdata.RuleErrorKey1.Tags,
			},
			{
				RuleID:       testdata.Rule2.Module,
				ErrorKey:     testdata.RuleErrorKey2.ErrorKey,
				CreatedAt:    testdata.RuleErrorKey2.PublishDate.UTC().Format(time.RFC3339),
				Description:  testdata.RuleErrorKey2.Description,
				Generic:      testdata.RuleErrorKey2.Generic,
				Reason:       testdata.Rule2.Reason,
				Resolution:   testdata.Rule2.Resolution,
				TotalRisk:    calculateTotalRisk(testdata.RuleErrorKey2.Impact, testdata.RuleErrorKey2.Likelihood),
				RiskOfChange: 0,
				Disabled:     testdata.Rule2Disabled,
				UserVote:     types.UserVoteNone,
				TemplateData: testdata.Rule2ExtraData,
				Tags:         testdata.RuleErrorKey2.Tags,
			},
			{
				RuleID:       testdata.Rule3.Module,
				ErrorKey:     testdata.RuleErrorKey3.ErrorKey,
				CreatedAt:    testdata.RuleErrorKey3.PublishDate.UTC().Format(time.RFC3339),
				Description:  testdata.RuleErrorKey3.Description,
				Generic:      testdata.RuleErrorKey3.Generic,
				Reason:       testdata.Rule3.Reason,
				Resolution:   testdata.Rule3.Resolution,
				TotalRisk:    calculateTotalRisk(testdata.RuleErrorKey3.Impact, testdata.RuleErrorKey3.Likelihood),
				RiskOfChange: 0,
				Disabled:     testdata.Rule3Disabled,
				UserVote:     types.UserVoteNone,
				TemplateData: testdata.Rule3ExtraData,
				Tags:         testdata.RuleErrorKey3.Tags,
			},
		},
	}

	SmartProxyReportResponse3RulesWithOnlyOSD = struct {
		Status string                  `json:"status"`
		Report *types.SmartProxyReport `json:"report"`
	}{
		Status: "ok",
		Report: &SmartProxyReport3RulesWithOnlyOSD,
	}

	SmartProxyReport3RulesWithOnlyOSD = types.SmartProxyReport{
		Meta: types.ReportResponseMeta{
			Count:         1,
			LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		},
		Data: []types.RuleWithContentResponse{
			{
				RuleID:       testdata.Rule1.Module,
				ErrorKey:     testdata.RuleErrorKey1.ErrorKey,
				CreatedAt:    testdata.RuleErrorKey1.PublishDate.UTC().Format(time.RFC3339),
				Description:  testdata.RuleErrorKey1.Description,
				Generic:      testdata.RuleErrorKey1.Generic,
				Reason:       testdata.Rule1.Reason,
				Resolution:   testdata.Rule1.Resolution,
				TotalRisk:    calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood),
				RiskOfChange: 0,
				Disabled:     testdata.Rule1Disabled,
				UserVote:     types.UserVoteNone,
				TemplateData: testdata.Rule1ExtraData,
				Tags:         testdata.RuleErrorKey1.Tags,
			},
		},
	}

	GetContentResponse3Rules = struct {
		Status string                  `json:"status"`
		Rules  []iou_types.RuleContent `json:"content"`
	}{
		Status: "ok",
		Rules: []iou_types.RuleContent{
			testdata.RuleContent1,
			testdata.RuleContent2,
			testdata.RuleContent3,
		},
	}

	RuleContentInternal1 = iou_types.RuleContent{
		Summary:    testdata.Rule1.Summary,
		Reason:     testdata.Rule1.Reason,
		Resolution: testdata.Rule1.Resolution,
		MoreInfo:   testdata.Rule1.MoreInfo,
		Plugin: iou_types.RulePluginInfo{
			Name:         testdata.Rule1.Name,
			NodeID:       "",
			ProductCode:  "",
			PythonModule: internalTestRuleModule,
		},
		ErrorKeys: map[string]iou_types.RuleErrorKeyContent{
			"ek1": {
				Generic: testdata.RuleErrorKey1.Generic,
				Metadata: iou_types.ErrorKeyMetadata{
					Condition:   testdata.RuleErrorKey1.Condition,
					Description: testdata.RuleErrorKey1.Description,
					Impact:      testdata.ImpactIntToStr[testdata.RuleErrorKey1.Impact],
					Likelihood:  testdata.RuleErrorKey1.Likelihood,
					PublishDate: testdata.RuleErrorKey1.PublishDate.UTC().Format(time.RFC3339),
					Tags:        testdata.RuleErrorKey1.Tags,
					Status:      "active",
				},
			},
		},
	}

	OverviewResponse = struct {
		Status   string                 `json:"status"`
		Overview map[string]interface{} `json:"overview"`
	}{
		Status: "ok",
		Overview: map[string]interface{}{
			"clusters_hit": 1,
			"hit_by_risk": map[int]int{
				1: 1,
				2: 1,
			},
			"hit_by_tag": map[string]int{
				"openshift":            1,
				"osd_customer":         1,
				"service_availability": 1,
			},
		},
	}

	SmartProxyReportResponse3SingleRule = struct {
		Status string                        `json:"status"`
		Report types.RuleWithContentResponse `json:"report"`
	}{
		Status: "ok",
		Report: types.RuleWithContentResponse{
			RuleID:       testdata.Rule1.Module,
			ErrorKey:     testdata.RuleErrorKey1.ErrorKey,
			CreatedAt:    testdata.RuleErrorKey1.PublishDate.UTC().Format(time.RFC3339),
			Description:  testdata.RuleErrorKey1.Description,
			Generic:      testdata.RuleErrorKey1.Generic,
			Reason:       testdata.Rule1.Reason,
			Resolution:   testdata.Rule1.Resolution,
			TotalRisk:    calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood),
			RiskOfChange: 0,
			Disabled:     testdata.Rule1Disabled,
			UserVote:     types.UserVoteNone,
			TemplateData: testdata.Rule1ExtraData,
			Tags:         testdata.RuleErrorKey1.Tags,
		},
	}
	SmartProxyReportResponse3NoRuleFound = struct {
		Status string `json:"status"`
	}{
		Status: "Rule was not found",
	}
)

// TODO: move to utils
func calculateTotalRisk(impact, likelihood int) int {
	return (impact + likelihood) / 2
}

func loadMockRuleContentDir(rulesContent []iou_types.RuleContent) {
	rules := make(map[string]iou_types.RuleContent)

	for index, rule := range rulesContent {
		key := fmt.Sprintf("rc%d", index)
		rules[key] = rule
	}
	ruleContentDirectory := iou_types.RuleContentDirectory{
		Config: iou_types.GlobalRuleConfig{
			Impact: testdata.ImpactStrToInt,
		},
		Rules: rules,
	}

	content.LoadRuleContent(&ruleContentDirectory)
	content.WaitForContentDirectoryToBeReady()
}

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

func TestServerStartError(t *testing.T) {
	testServer := server.New(server.Configuration{
		Address:   "localhost:99999",
		APIPrefix: "",
	}, services.Configuration{
		AggregatorBaseEndpoint: "http://localhost:8081/api/v1/",
		ContentBaseEndpoint:    "http://localhost:8082/api/v1/",
	},
		nil,
	)

	err := testServer.Start()
	assert.EqualError(t, err, "listen tcp: address 99999: invalid port")
}

func TestAddCORSHeaders(t *testing.T) {
	helpers.AssertAPIRequest(t, &helpers.DefaultServerConfigCORS, &helpers.DefaultServicesConfig, nil, &helpers.APIRequest{
		Method:   http.MethodOptions,
		Endpoint: server.RuleGroupsEndpoint,
		ExtraHeaders: http.Header{
			"Origin":                         []string{"http://example.com"},
			"Access-Control-Request-Method":  []string{http.MethodOptions},
			"Access-Control-Request-Headers": []string{"X-Csrf-Token,Content-Type,Content-Length"},
		},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Access-Control-Allow-Origin":      "*",
			"Access-Control-Allow-Credentials": "true",
			"Access-Control-Allow-Methods":     http.MethodOptions,
			"Access-Control-Allow-Headers":     "X-Csrf-Token,Content-Type,Content-Length",
		},
	})
}

func ruleIDsChecker(t testing.TB, expected, got []byte) {
	type Response struct {
		Status string   `json:"status"`
		Rules  []string `json:"rules"`
	}

	var expectedResp, gotResp Response

	if err := json.Unmarshal(expected, &expectedResp); err != nil {
		err = fmt.Errorf(`"expected" is not JSON. value = "%v", err = "%v"`, expected, err)
		helpers.FailOnError(t, err)
	}

	if err := json.Unmarshal(got, &gotResp); err != nil {
		err = fmt.Errorf(`"got" is not JSON. value = "%v", err = "%v"`, got, err)
		helpers.FailOnError(t, err)
	}

	assert.ElementsMatch(t, expectedResp.Rules, gotResp.Rules)
}

func ruleInContentChecker(t testing.TB, expected, got []byte) {
	type Response struct {
		Status  string `json:"string"`
		Content []iou_types.RuleContent
	}

	var expectedResp, gotResp Response

	if err := json.Unmarshal(expected, &expectedResp); err != nil {
		err = fmt.Errorf(`"expected" is not JSON. value = "%v", err = "%v"`, expected, err)
		helpers.FailOnError(t, err)
	}

	if err := json.Unmarshal(got, &gotResp); err != nil {
		err = fmt.Errorf(`"got" is not JSON. value = "%v", err = "%v"`, got, err)
		helpers.FailOnError(t, err)
	}

	assert.ElementsMatch(t, expectedResp.Content, gotResp.Content)
}
