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
	"bytes"
	"net/http"
	"testing"
	"time"

	ics_server "github.com/RedHatInsights/insights-content-service/server"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	gock "gopkg.in/h2non/gock.v1"

	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
	"github.com/RedHatInsights/insights-results-smart-proxy/types"
)

const (
	testTimeout = 10 * time.Second
)

// TODO: consider moving to data repo
var (
	SmartProxyReportResponse3Rules = struct {
		Status string                  `json:"status"`
		Report *types.SmartProxyReport `json:"report"`
	}{
		Status: "ok",
		Report: &SmartProxyReport3Rules,
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
				TemplateData: testdata.Rule1.MoreInfo,
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
				TemplateData: testdata.Rule2.MoreInfo,
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
				TemplateData: testdata.Rule3.MoreInfo,
				Tags:         testdata.RuleErrorKey3.Tags,
			},
		},
	}
)

// TODO: move to utils
func calculateTotalRisk(impact, likelihood int) int {
	return (impact + likelihood) / 2
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

func TestHTTPServer_ReportEndpoint(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		defer gock.Off()

		gock.New(helpers.DefaultServicesConfig.AggregatorBaseEndpoint).
			Get("/").
			AddMatcher(helpers.NewGockAPIEndpointMatcher(ira_server.ReportEndpoint)).
			Reply(200).
			JSON(testdata.Report3RulesExpectedResponse)

		gock.New(helpers.DefaultServicesConfig.ContentBaseEndpoint).
			Get("/").
			AddMatcher(helpers.NewGockAPIEndpointMatcher(ics_server.AllContentEndpoint)).
			Reply(200).
			Body(bytes.NewBuffer(helpers.MustGobSerialize(t, testdata.RuleContentDirectory3Rules)))

		go content.RunUpdateContentLoop(helpers.DefaultServicesConfig)

		helpers.AssertAPIRequest(t, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(SmartProxyReportResponse3Rules),
		})
	}, testTimeout)
}

// TODO: test more cases for report endpoint
