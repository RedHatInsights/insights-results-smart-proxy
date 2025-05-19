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

	data "github.com/RedHatInsights/insights-results-smart-proxy/tests/testdata"

	"github.com/RedHatInsights/insights-content-service/groups"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ctypes "github.com/RedHatInsights/insights-results-types"
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
	internalTestRuleModule = "ccx_ocp_rules.internal.bar"
	internalRuleID         = internalTestRuleModule + "|" + testdata.ErrorKey1
)

// TODO: consider moving to data repo
var (
	userIDInGoodAuthToken = "1"
	// goodXRHAuthToken is in following structure https://docs.google.com/document/d/1PAzJqcUXlxg7t5cX1lsPsQtBPT_95bZbyB9Iiv_ekGM/
	// with expected values (org_id == 1, account_number == 1)
	goodXRHAuthToken = `eyJpZGVudGl0eSI6eyJhY2NvdW50X251bWJlciI6IjEiLCJvcmdfaWQiOiIxIiwidHlwZSI6IlVzZXIiLCJ1c2VyIjp7InVzZXJuYW1lIjoiamRvZSIsInVzZXJfaWQiOiIxIiwiZW1haWwiOiJqZG9lQGFjbWUuY29tIiwiZmlyc3RfbmFtZSI6IkpvaG4iLCJsYXN0X25hbWUiOiJEb2UiLCJpc19hY3RpdmUiOnRydWUsImlzX29yZ19hZG1pbiI6ZmFsc2UsImlzX2ludGVybmFsIjpmYWxzZSwibG9jYWxlIjoiZW5fVVMifSwiaW50ZXJuYWwiOnsib3JnX2lkIjoiMSIsImF1dGhfdHlwZSI6ImJhc2ljLWF1dGgiLCJhdXRoX3RpbWUiOjYzMDB9fX0K`
	// badXRHAuthToken has correct structure, but unexpected values (org_id == 1234, account_number == 1234)
	badXRHAuthToken = `eyJpZGVudGl0eSI6eyJhY2NvdW50X251bWJlciI6IjEyMzQiLCJvcmdfaWQiOiIxMjM0IiwidHlwZSI6IlVzZXIiLCJ1c2VyIjp7InVzZXJuYW1lIjoiamRvZSIsInVzZXJfaWQiOiIxMjM0IiwiZW1haWwiOiJqZG9lQGFjbWUuY29tIiwiZmlyc3RfbmFtZSI6IkpvaG4iLCJsYXN0X25hbWUiOiJEb2UiLCJpc19hY3RpdmUiOnRydWUsImlzX29yZ19hZG1pbiI6ZmFsc2UsImlzX2ludGVybmFsIjpmYWxzZSwibG9jYWxlIjoiZW5fVVMifSwiaW50ZXJuYWwiOnsib3JnX2lkIjoiMTIzNCIsImF1dGhfdHlwZSI6ImJhc2ljLWF1dGgiLCJhdXRoX3RpbWUiOjYzMDB9fX0K`
	// anemicXRHAuthToken has correct structure, but is missing account_number
	anemicXRHAuthToken  = `eyJpZGVudGl0eSI6eyJvcmdfaWQiOiIxIiwidHlwZSI6IlVzZXIiLCJ1c2VyIjp7InVzZXJuYW1lIjoiamRvZSIsInVzZXJfaWQiOiIxIiwiZW1haWwiOiJqZG9lQGFjbWUuY29tIiwiZmlyc3RfbmFtZSI6IkpvaG4iLCJsYXN0X25hbWUiOiJEb2UiLCJpc19hY3RpdmUiOnRydWUsImlzX29yZ19hZG1pbiI6ZmFsc2UsImlzX2ludGVybmFsIjpmYWxzZSwibG9jYWxlIjoiZW5fVVMifSwiaW50ZXJuYWwiOnsib3JnX2lkIjoiMSIsImF1dGhfdHlwZSI6ImJhc2ljLWF1dGgiLCJhdXRoX3RpbWUiOjYzMDB9fX0K`
	invalidXRHAuthToken = `invalid token`
	testTimeStr         = "2021-01-02T15:04:05Z"
	testTimestamp       = types.Timestamp(testTimeStr)

	serverConfigXRH = server.Configuration{
		Address:                          ":8081",
		APIv1Prefix:                      "/api/v1/",
		APIv2Prefix:                      "/api/v2/",
		APIv1SpecFile:                    "server/api/v1/openapi.json",
		APIv2SpecFile:                    "server/api/v2/openapi.json",
		Debug:                            true,
		Auth:                             true,
		AuthType:                         "xrh",
		UseHTTPS:                         false,
		EnableCORS:                       false,
		EnableInternalRulesOrganizations: false,
		InternalRulesOrganizations:       []ctypes.OrgID{1},
		UseRBAC:                          false,
	}

	serverConfigInternalOrganizations1 = server.Configuration{
		Address:                          ":8081",
		APIv1Prefix:                      "/api/v1/",
		APIv2Prefix:                      "/api/v2/",
		APIv1SpecFile:                    "server/api/v1/openapi.json",
		APIv2SpecFile:                    "server/api/v2/openapi.json",
		Debug:                            true,
		Auth:                             true,
		AuthType:                         "xrh",
		UseHTTPS:                         false,
		EnableCORS:                       false,
		EnableInternalRulesOrganizations: true,
		InternalRulesOrganizations:       []ctypes.OrgID{1},
	}

	// Same as previous one, but different InternalRulesOrganizations
	// This one won't match with the authentication token used
	serverConfigInternalOrganizations2 = server.Configuration{
		Address:                          ":8081",
		APIv1Prefix:                      "/api/v1/",
		APIv2Prefix:                      "/api/v2/",
		APIv1SpecFile:                    "server/api/v1/openapi.json",
		APIv2SpecFile:                    "server/api/v2/openapi.json",
		Debug:                            true,
		Auth:                             true,
		AuthType:                         "xrh",
		UseHTTPS:                         false,
		EnableCORS:                       false,
		EnableInternalRulesOrganizations: true,
		InternalRulesOrganizations:       []ctypes.OrgID{2},
	}

	SmartProxyReportResponse1RuleNoContent = struct {
		Status string                  `json:"status"`
		Report *types.SmartProxyReport `json:"report"`
	}{
		Status: "ok",
		Report: &SmartProxyReport1RuleNoContentNoAMSClient,
	}

	SmartProxyReportResponse3Rules2NoContent = struct {
		Status string                  `json:"status"`
		Report *types.SmartProxyReport `json:"report"`
	}{
		Status: "ok",
		Report: &SmartProxyReport3Rules2NoContentNoAMSClient,
	}

	SmartProxyReportResponse3Rules = struct {
		Status string                  `json:"status"`
		Report *types.SmartProxyReport `json:"report"`
	}{
		Status: "ok",
		Report: &SmartProxyReport3RulesNoAMSClient,
	}

	SmartProxyReport1RuleNoContentNoAMSClient = types.SmartProxyReport{
		Meta: types.ReportResponseMeta{
			DisplayName:   string(testdata.ClusterName),
			Count:         0,
			LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		},
		Data: []types.RuleWithContentResponse{},
	}

	Report3RulesData = []types.RuleWithContentResponse{
		{
			RuleID:       testdata.Rule1.Module,
			ErrorKey:     testdata.RuleErrorKey1.ErrorKey,
			CreatedAt:    testdata.RuleErrorKey1.PublishDate.UTC().Format(time.RFC3339),
			Description:  testdata.RuleErrorKey1.Description,
			Generic:      testdata.RuleErrorKey1.Generic,
			Reason:       testdata.RuleErrorKey1.Reason,
			Resolution:   testdata.RuleErrorKey1.Resolution,
			MoreInfo:     testdata.RuleErrorKey1.MoreInfo,
			TotalRisk:    calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood),
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
			Reason:       testdata.RuleErrorKey2.Reason,
			Resolution:   testdata.RuleErrorKey2.Resolution,
			MoreInfo:     testdata.RuleErrorKey2.MoreInfo,
			TotalRisk:    calculateTotalRisk(testdata.RuleErrorKey2.Impact, testdata.RuleErrorKey2.Likelihood),
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
			Reason:       testdata.RuleErrorKey3.Reason,
			Resolution:   testdata.RuleErrorKey3.Resolution,
			MoreInfo:     testdata.RuleErrorKey3.MoreInfo,
			TotalRisk:    calculateTotalRisk(testdata.RuleErrorKey3.Impact, testdata.RuleErrorKey3.Likelihood),
			Disabled:     testdata.Rule3Disabled,
			UserVote:     types.UserVoteNone,
			TemplateData: testdata.Rule3ExtraData,
			Tags:         testdata.RuleErrorKey3.Tags,
		},
	}

	SmartProxyReport3RulesNoAMSClient = types.SmartProxyReport{
		Meta: types.ReportResponseMeta{
			DisplayName:   string(testdata.ClusterName),
			Count:         3,
			LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		},
		Data: Report3RulesData,
	}

	Report3Rules2NoContentData = []types.RuleWithContentResponse{
		{
			RuleID:       testdata.Rule1.Module,
			ErrorKey:     testdata.RuleErrorKey1.ErrorKey,
			CreatedAt:    testdata.RuleErrorKey1.PublishDate.UTC().Format(time.RFC3339),
			Description:  testdata.RuleErrorKey1.Description,
			Generic:      testdata.RuleErrorKey1.Generic,
			Reason:       testdata.RuleErrorKey1.Reason,
			Resolution:   testdata.RuleErrorKey1.Resolution,
			MoreInfo:     testdata.RuleErrorKey1.MoreInfo,
			TotalRisk:    calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood),
			Disabled:     testdata.Rule1Disabled,
			UserVote:     types.UserVoteNone,
			TemplateData: testdata.Rule1ExtraData,
			Tags:         testdata.RuleErrorKey1.Tags,
		},
	}

	SmartProxyReport3Rules2NoContentNoAMSClient = types.SmartProxyReport{
		Meta: types.ReportResponseMeta{
			DisplayName:   string(testdata.ClusterName),
			Count:         3,
			LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		},
		Data: Report3Rules2NoContentData,
	}

	Report3RulesWithOnlyOSDData = []types.RuleWithContentResponse{
		{
			RuleID:       testdata.Rule1.Module,
			ErrorKey:     testdata.RuleErrorKey1.ErrorKey,
			CreatedAt:    testdata.RuleErrorKey1.PublishDate.UTC().Format(time.RFC3339),
			Description:  testdata.RuleErrorKey1.Description,
			Generic:      testdata.RuleErrorKey1.Generic,
			Reason:       testdata.RuleErrorKey1.Reason,
			Resolution:   testdata.RuleErrorKey1.Resolution,
			MoreInfo:     testdata.RuleErrorKey1.MoreInfo,
			TotalRisk:    calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood),
			Disabled:     testdata.Rule1Disabled,
			UserVote:     types.UserVoteNone,
			TemplateData: testdata.Rule1ExtraData,
			Tags:         testdata.RuleErrorKey1.Tags,
		},
	}

	Report3RulesOnlyEnabledData = []types.RuleWithContentResponse{
		{
			RuleID:       testdata.Rule1.Module,
			ErrorKey:     testdata.RuleErrorKey1.ErrorKey,
			CreatedAt:    testdata.RuleErrorKey1.PublishDate.UTC().Format(time.RFC3339),
			Description:  testdata.RuleErrorKey1.Description,
			Generic:      testdata.RuleErrorKey1.Generic,
			Reason:       testdata.RuleErrorKey1.Reason,
			Resolution:   testdata.RuleErrorKey1.Resolution,
			MoreInfo:     testdata.RuleErrorKey1.MoreInfo,
			TotalRisk:    calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood),
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
			Reason:       testdata.RuleErrorKey2.Reason,
			Resolution:   testdata.RuleErrorKey2.Resolution,
			MoreInfo:     testdata.RuleErrorKey2.MoreInfo,
			TotalRisk:    calculateTotalRisk(testdata.RuleErrorKey2.Impact, testdata.RuleErrorKey2.Likelihood),
			Disabled:     testdata.Rule2Disabled,
			UserVote:     types.UserVoteNone,
			TemplateData: testdata.Rule2ExtraData,
			Tags:         testdata.RuleErrorKey2.Tags,
		},
	}

	Report3RulesWithDisabledData = []types.RuleWithContentResponse{
		{
			RuleID:       testdata.Rule1.Module,
			ErrorKey:     testdata.RuleErrorKey1.ErrorKey,
			CreatedAt:    testdata.RuleErrorKey1.PublishDate.UTC().Format(time.RFC3339),
			Description:  testdata.RuleErrorKey1.Description,
			Generic:      testdata.RuleErrorKey1.Generic,
			Reason:       testdata.RuleErrorKey1.Reason,
			Resolution:   testdata.RuleErrorKey1.Resolution,
			MoreInfo:     testdata.RuleErrorKey1.MoreInfo,
			TotalRisk:    calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood),
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
			Reason:       testdata.RuleErrorKey2.Reason,
			Resolution:   testdata.RuleErrorKey2.Resolution,
			MoreInfo:     testdata.RuleErrorKey2.MoreInfo,
			TotalRisk:    calculateTotalRisk(testdata.RuleErrorKey2.Impact, testdata.RuleErrorKey2.Likelihood),
			Disabled:     testdata.Rule2Disabled,
			UserVote:     types.UserVoteNone,
			TemplateData: testdata.Rule2ExtraData,
			Tags:         testdata.RuleErrorKey2.Tags,
		},
		{
			RuleID:       testdata.Rule5.Module,
			ErrorKey:     testdata.RuleErrorKey5.ErrorKey,
			CreatedAt:    testdata.RuleErrorKey5.PublishDate.UTC().Format(time.RFC3339),
			Description:  testdata.RuleErrorKey5.Description,
			Generic:      testdata.RuleErrorKey5.Generic,
			Reason:       testdata.RuleErrorKey5.Reason,
			Resolution:   testdata.RuleErrorKey5.Resolution,
			MoreInfo:     testdata.RuleErrorKey5.MoreInfo,
			TotalRisk:    calculateTotalRisk(testdata.RuleErrorKey5.Impact, testdata.RuleErrorKey5.Likelihood),
			Disabled:     testdata.Rule5Disabled,
			UserVote:     types.UserVoteNone,
			TemplateData: testdata.Rule5ExtraData,
			Tags:         testdata.RuleErrorKey5.Tags,
		},
	}

	SmartProxyReportResponse3RulesWithOnlyOSD = struct {
		Status string                  `json:"status"`
		Report *types.SmartProxyReport `json:"report"`
	}{
		Status: "ok",
		Report: &SmartProxyReport3RulesWithOnlyOSDNoAMSClient,
	}

	SmartProxyReportResponse3RulesOnlyEnabled = struct {
		Status string                  `json:"status"`
		Report *types.SmartProxyReport `json:"report"`
	}{
		Status: "ok",
		Report: &SmartProxyReport3RulesOnlyEnabledNoAMSClient,
	}

	SmartProxyEmptyResponse = struct {
		Status string                  `json:"status"`
		Report *types.SmartProxyReport `json:"report"`
	}{
		Status: "ok",
		Report: &SmartProxyReportEmptyCount2NoAMSClient,
	}

	SmartProxyReportResponse3RulesAll = struct {
		Status string                  `json:"status"`
		Report *types.SmartProxyReport `json:"report"`
	}{
		Status: "ok",
		Report: &SmartProxyReport3RulesWithDisabledNoAMSClient,
	}

	SmartProxyReport3RulesWithOnlyOSDNoAMSClient = types.SmartProxyReport{
		Meta: types.ReportResponseMeta{
			DisplayName:   string(testdata.ClusterName),
			Count:         1,
			LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		},
		Data: Report3RulesWithOnlyOSDData,
	}

	SmartProxyReport3RulesOnlyEnabledNoAMSClient = types.SmartProxyReport{
		Meta: types.ReportResponseMeta{
			DisplayName:   string(testdata.ClusterName),
			Count:         2,
			LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		},
		Data: Report3RulesOnlyEnabledData,
	}

	SmartProxyReportEmptyCount2NoAMSClient = types.SmartProxyReport{
		Meta: types.ReportResponseMeta{
			DisplayName:   string(testdata.ClusterName),
			Count:         2,
			LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		},
		Data: []types.RuleWithContentResponse{},
	}

	SmartProxyReport3RulesWithDisabledNoAMSClient = types.SmartProxyReport{
		Meta: types.ReportResponseMeta{
			DisplayName:   string(testdata.ClusterName),
			Count:         3,
			LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		},
		Data: Report3RulesWithDisabledData,
	}

	GetContentResponse3Rules = struct {
		Status string                 `json:"status"`
		Rules  []ctypes.RuleContentV1 `json:"content"`
	}{
		Status: "ok",
		Rules: []ctypes.RuleContentV1{
			content.RuleContentToV1(&testdata.RuleContent1),
			content.RuleContentToV1(&testdata.RuleContent2),
			content.RuleContentToV1(&testdata.RuleContent3),
		},
	}

	RuleContentInternal1 = ctypes.RuleContent{
		Summary:    testdata.Rule1.Summary,
		Generic:    testdata.Rule1.Generic,
		Reason:     testdata.Rule1.Reason,
		Resolution: testdata.Rule1.Resolution,
		MoreInfo:   testdata.Rule1.MoreInfo,
		Plugin: ctypes.RulePluginInfo{
			Name:         testdata.Rule1.Name,
			NodeID:       "",
			ProductCode:  "",
			PythonModule: internalTestRuleModule,
		},
		ErrorKeys: map[string]ctypes.RuleErrorKeyContent{
			"ek1": {
				Summary:    testdata.RuleErrorKey1.Summary,
				Generic:    testdata.RuleErrorKey1.Generic,
				Reason:     testdata.RuleErrorKey1.Reason,
				Resolution: testdata.RuleErrorKey1.Resolution,
				MoreInfo:   testdata.RuleErrorKey1.MoreInfo,
				Metadata: ctypes.ErrorKeyMetadata{
					Description: testdata.RuleErrorKey1.Description,
					Impact: ctypes.Impact{
						Name:   "test_impact",
						Impact: testdata.RuleErrorKey1.Impact,
					},
					Likelihood:  testdata.RuleErrorKey1.Likelihood,
					PublishDate: testdata.RuleErrorKey1.PublishDate.UTC().Format(time.RFC3339),
					Tags:        testdata.RuleErrorKey1.Tags,
					Status:      "active",
				},
			},
		},
	}

	OverviewResponseRules123Enabled = struct {
		Status   string                 `json:"status"`
		Overview map[string]interface{} `json:"overview"`
	}{
		Status: "ok",
		Overview: map[string]interface{}{
			"clusters_hit": 1,
			"hit_by_risk": map[string]int{
				"1": 1,
				"2": 2,
			},
			"hit_by_tag": map[string]int{
				"openshift":            1,
				"osd_customer":         1,
				"service_availability": 1,
			},
		},
	}

	OverviewResponseManagedRules = struct {
		Status   string                 `json:"status"`
		Overview map[string]interface{} `json:"overview"`
	}{
		Status: "ok",
		Overview: map[string]interface{}{
			"clusters_hit": 1,
			"hit_by_risk": map[string]int{
				"1": 1,
			},
			"hit_by_tag": map[string]int{
				"openshift":            1,
				"osd_customer":         1,
				"service_availability": 1,
			},
		},
	}

	OverviewResponseRule1DisabledRule2Enabled = struct {
		Status   string                 `json:"status"`
		Overview map[string]interface{} `json:"overview"`
	}{
		Status: "ok",
		Overview: map[string]interface{}{
			"clusters_hit": 1,
			"hit_by_risk": map[string]int{
				"2": 2,
			},
			"hit_by_tag": map[string]int{},
		},
	}

	OverviewResponseRule1EnabledRule2Disabled = struct {
		Status   string                 `json:"status"`
		Overview map[string]interface{} `json:"overview"`
	}{
		Status: "ok",
		Overview: map[string]interface{}{
			"clusters_hit": 1,
			"hit_by_risk": map[string]int{
				"1": 1,
				"2": 1,
			},
			"hit_by_tag": map[string]int{
				"openshift":            1,
				"osd_customer":         1,
				"service_availability": 1,
			},
		},
	}

	OverviewResponseRule5DisabledRules1And2Enabled = struct {
		Status   string                 `json:"status"`
		Overview map[string]interface{} `json:"overview"`
	}{
		Status: "ok",
		Overview: map[string]interface{}{
			"clusters_hit": 1,
			"hit_by_risk": map[string]int{
				"1": 1,
				"2": 1,
			},
			"hit_by_tag": map[string]int{
				"openshift":            1,
				"osd_customer":         1,
				"service_availability": 1,
			},
		},
	}

	OverviewResponsePostEndpoint = struct {
		Status   string                 `json:"status"`
		Overview map[string]interface{} `json:"overview"`
	}{
		Status: "ok",
		Overview: map[string]interface{}{
			"clusters_hit": 2,
			"hit_by_risk": map[string]int{
				"1": 1,
				"2": 2,
			},
			"hit_by_tag": map[string]int{
				"openshift":            1,
				"osd_customer":         1,
				"service_availability": 1,
			},
		},
	}

	OverviewResponsePostEndpointRule1Disabled = struct {
		Status   string                 `json:"status"`
		Overview map[string]interface{} `json:"overview"`
	}{
		Status: "ok",
		Overview: map[string]interface{}{
			"clusters_hit": 2,
			"hit_by_risk": map[string]int{
				"2": 2,
			},
			"hit_by_tag": map[string]int{},
		},
	}

	OverviewResponsePostEndpointRule2Disabled = struct {
		Status   string                 `json:"status"`
		Overview map[string]interface{} `json:"overview"`
	}{
		Status: "ok",
		Overview: map[string]interface{}{
			"clusters_hit": 2,
			"hit_by_risk": map[string]int{
				"1": 1,
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
			Reason:       testdata.RuleErrorKey1.Reason,
			Resolution:   testdata.RuleErrorKey1.Resolution,
			MoreInfo:     testdata.RuleErrorKey1.MoreInfo,
			TotalRisk:    calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood),
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

	GetRecommendationsResponse1Rule2Cluster = struct {
		Status          string                         `json:"status"`
		Recommendations []types.RecommendationListView `json:"recommendations"`
	}{
		Status: "ok",
		Recommendations: []types.RecommendationListView{
			{
				RuleID:              testdata.Rule1CompositeID,
				Description:         testdata.RuleErrorKey1.Description,
				Generic:             testdata.RuleErrorKey1.Generic,
				PublishDate:         testdata.RuleErrorKey1.PublishDate,
				TotalRisk:           uint8(calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood)),
				ResolutionRisk:      uint8(testdata.RuleErrorKey1.ResolutionRisk),
				Impact:              uint8(testdata.RuleErrorKey1.Impact),
				Likelihood:          uint8(testdata.RuleErrorKey1.Likelihood),
				Tags:                testdata.RuleErrorKey1.Tags,
				Disabled:            false,
				ImpactedClustersCnt: 2,
			},
		},
	}

	GetRecommendationsResponse2Rules0Clusters = struct {
		Status          string                         `json:"status"`
		Recommendations []types.RecommendationListView `json:"recommendations"`
	}{
		Status: "ok",
		Recommendations: []types.RecommendationListView{
			{
				RuleID:              testdata.Rule1CompositeID,
				Description:         testdata.RuleErrorKey1.Description,
				Generic:             testdata.RuleErrorKey1.Generic,
				PublishDate:         testdata.RuleErrorKey1.PublishDate,
				TotalRisk:           uint8(calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood)),
				ResolutionRisk:      uint8(testdata.RuleErrorKey1.ResolutionRisk),
				Impact:              uint8(testdata.RuleErrorKey1.Impact),
				Likelihood:          uint8(testdata.RuleErrorKey1.Likelihood),
				Tags:                testdata.RuleErrorKey1.Tags,
				Disabled:            false,
				ImpactedClustersCnt: 0,
			},
			{
				RuleID:              testdata.Rule2CompositeID,
				Description:         testdata.RuleErrorKey2.Description,
				Generic:             testdata.RuleErrorKey2.Generic,
				PublishDate:         testdata.RuleErrorKey2.PublishDate,
				TotalRisk:           uint8(calculateTotalRisk(testdata.RuleErrorKey2.Impact, testdata.RuleErrorKey2.Likelihood)),
				ResolutionRisk:      uint8(testdata.RuleErrorKey2.ResolutionRisk),
				Impact:              uint8(testdata.RuleErrorKey2.Impact),
				Likelihood:          uint8(testdata.RuleErrorKey2.Likelihood),
				Tags:                testdata.RuleErrorKey2.Tags,
				Disabled:            false,
				ImpactedClustersCnt: 0,
			},
		},
	}

	GetRecommendationsResponse2Rules1Disabled1Acked = struct {
		Status          string                         `json:"status"`
		Recommendations []types.RecommendationListView `json:"recommendations"`
	}{
		Status: "ok",
		Recommendations: []types.RecommendationListView{
			{
				RuleID:              testdata.Rule1CompositeID,
				Description:         testdata.RuleErrorKey1.Description,
				Generic:             testdata.RuleErrorKey1.Generic,
				PublishDate:         testdata.RuleErrorKey1.PublishDate,
				TotalRisk:           uint8(calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood)),
				ResolutionRisk:      uint8(testdata.RuleErrorKey1.ResolutionRisk),
				Impact:              uint8(testdata.RuleErrorKey1.Impact),
				Likelihood:          uint8(testdata.RuleErrorKey1.Likelihood),
				Tags:                testdata.RuleErrorKey1.Tags,
				Disabled:            true, // acked flag
				ImpactedClustersCnt: 2,
			},
			{
				RuleID:              testdata.Rule2CompositeID,
				Description:         testdata.RuleErrorKey2.Description,
				Generic:             testdata.RuleErrorKey2.Generic,
				PublishDate:         testdata.RuleErrorKey2.PublishDate,
				TotalRisk:           uint8(calculateTotalRisk(testdata.RuleErrorKey2.Impact, testdata.RuleErrorKey2.Likelihood)),
				ResolutionRisk:      uint8(testdata.RuleErrorKey2.ResolutionRisk),
				Impact:              uint8(testdata.RuleErrorKey2.Impact),
				Likelihood:          uint8(testdata.RuleErrorKey2.Likelihood),
				Tags:                testdata.RuleErrorKey2.Tags,
				Disabled:            false,
				ImpactedClustersCnt: 1, // one cluster is disabled
			},
		},
	}

	GetRecommendationsResponse2Rules2Clusters = struct {
		Status          string                         `json:"status"`
		Recommendations []types.RecommendationListView `json:"recommendations"`
	}{
		Status: "ok",
		Recommendations: []types.RecommendationListView{
			{
				RuleID:              testdata.Rule1CompositeID,
				Description:         testdata.RuleErrorKey1.Description,
				Generic:             testdata.RuleErrorKey1.Generic,
				PublishDate:         testdata.RuleErrorKey1.PublishDate,
				TotalRisk:           uint8(calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood)),
				ResolutionRisk:      uint8(testdata.RuleErrorKey1.ResolutionRisk),
				Impact:              uint8(testdata.RuleErrorKey1.Impact),
				Likelihood:          uint8(testdata.RuleErrorKey1.Likelihood),
				Tags:                testdata.RuleErrorKey1.Tags,
				Disabled:            false,
				ImpactedClustersCnt: 2,
			},
			{
				RuleID:              testdata.Rule2CompositeID,
				Description:         testdata.RuleErrorKey2.Description,
				Generic:             testdata.RuleErrorKey2.Generic,
				PublishDate:         testdata.RuleErrorKey2.PublishDate,
				TotalRisk:           uint8(calculateTotalRisk(testdata.RuleErrorKey2.Impact, testdata.RuleErrorKey2.Likelihood)),
				ResolutionRisk:      uint8(testdata.RuleErrorKey2.ResolutionRisk),
				Impact:              uint8(testdata.RuleErrorKey2.Impact),
				Likelihood:          uint8(testdata.RuleErrorKey2.Likelihood),
				Tags:                testdata.RuleErrorKey2.Tags,
				Disabled:            false,
				ImpactedClustersCnt: 1,
			},
		},
	}

	GetRecommendationsResponse2Rules2Clusters1Managed = struct {
		Status          string                         `json:"status"`
		Recommendations []types.RecommendationListView `json:"recommendations"`
	}{
		Status: "ok",
		Recommendations: []types.RecommendationListView{
			{
				RuleID:              testdata.Rule1CompositeID,
				Description:         testdata.RuleErrorKey1.Description,
				Generic:             testdata.RuleErrorKey1.Generic,
				PublishDate:         testdata.RuleErrorKey1.PublishDate,
				TotalRisk:           uint8(calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood)),
				ResolutionRisk:      uint8(testdata.RuleErrorKey1.ResolutionRisk),
				Impact:              uint8(testdata.RuleErrorKey1.Impact),
				Likelihood:          uint8(testdata.RuleErrorKey1.Likelihood),
				Tags:                testdata.RuleErrorKey1.Tags,
				Disabled:            false,
				ImpactedClustersCnt: 2,
			},
			{
				RuleID:              testdata.Rule2CompositeID,
				Description:         testdata.RuleErrorKey2.Description,
				Generic:             testdata.RuleErrorKey2.Generic,
				PublishDate:         testdata.RuleErrorKey2.PublishDate,
				TotalRisk:           uint8(calculateTotalRisk(testdata.RuleErrorKey2.Impact, testdata.RuleErrorKey2.Likelihood)),
				ResolutionRisk:      uint8(testdata.RuleErrorKey2.ResolutionRisk),
				Impact:              uint8(testdata.RuleErrorKey2.Impact),
				Likelihood:          uint8(testdata.RuleErrorKey2.Likelihood),
				Tags:                testdata.RuleErrorKey2.Tags,
				Disabled:            false,
				ImpactedClustersCnt: 0,
			},
		},
	}

	GetRecommendationsResponse3Rules1Cluster = struct {
		Status          string                         `json:"status"`
		Recommendations []types.RecommendationListView `json:"recommendations"`
	}{
		Status: "ok",
		Recommendations: []types.RecommendationListView{
			{
				RuleID:              testdata.Rule1CompositeID,
				Description:         testdata.RuleErrorKey1.Description,
				Generic:             testdata.RuleErrorKey1.Generic,
				PublishDate:         testdata.RuleErrorKey1.PublishDate,
				TotalRisk:           uint8(calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood)),
				ResolutionRisk:      uint8(testdata.RuleErrorKey1.ResolutionRisk),
				Impact:              uint8(testdata.RuleErrorKey1.Impact),
				Likelihood:          uint8(testdata.RuleErrorKey1.Likelihood),
				Tags:                testdata.RuleErrorKey1.Tags,
				Disabled:            false,
				ImpactedClustersCnt: 1,
			},
			{
				RuleID:              testdata.Rule2CompositeID,
				Description:         testdata.RuleErrorKey2.Description,
				Generic:             testdata.RuleErrorKey2.Generic,
				PublishDate:         testdata.RuleErrorKey2.PublishDate,
				TotalRisk:           uint8(calculateTotalRisk(testdata.RuleErrorKey2.Impact, testdata.RuleErrorKey2.Likelihood)),
				ResolutionRisk:      uint8(testdata.RuleErrorKey2.ResolutionRisk),
				Impact:              uint8(testdata.RuleErrorKey2.Impact),
				Likelihood:          uint8(testdata.RuleErrorKey2.Likelihood),
				Tags:                testdata.RuleErrorKey2.Tags,
				Disabled:            false,
				ImpactedClustersCnt: 0,
			},
			{
				RuleID:              testdata.Rule3CompositeID,
				Description:         testdata.RuleErrorKey3.Description,
				Generic:             testdata.RuleErrorKey3.Generic,
				PublishDate:         testdata.RuleErrorKey3.PublishDate,
				TotalRisk:           uint8(calculateTotalRisk(testdata.RuleErrorKey3.Impact, testdata.RuleErrorKey3.Likelihood)),
				ResolutionRisk:      uint8(testdata.RuleErrorKey3.ResolutionRisk),
				Impact:              uint8(testdata.RuleErrorKey3.Impact),
				Likelihood:          uint8(testdata.RuleErrorKey3.Likelihood),
				Tags:                testdata.RuleErrorKey3.Tags,
				Disabled:            false,
				ImpactedClustersCnt: 0,
			},
		},
	}

	GetRecommendationsResponse0Rules = struct {
		Status          string                         `json:"status"`
		Recommendations []types.RecommendationListView `json:"recommendations"`
	}{
		Status:          "ok",
		Recommendations: []types.RecommendationListView{},
	}

	GetRuleContentRecommendationContent1 = struct {
		Content types.RecommendationContent `json:"content"`
		Groups  []groups.Group              `json:"groups"`
		Status  string                      `json:"status"`
	}{
		Content: types.RecommendationContent{
			RuleSelector: ctypes.RuleSelector(testdata.Rule1CompositeID),
			Description:  testdata.RuleErrorKey1.Description,
			Generic:      testdata.RuleErrorKey1.Generic,
			Reason:       testdata.RuleErrorKey1.Reason,
			Resolution:   testdata.RuleErrorKey1.Resolution,
			MoreInfo:     testdata.RuleErrorKey1.MoreInfo,
			TotalRisk:    uint8(calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood)),
			Impact:       uint8(testdata.RuleErrorKey1.Impact),
			Likelihood:   uint8(testdata.RuleErrorKey1.Likelihood),
			PublishDate:  testdata.RuleErrorKey1.PublishDate,
			Tags:         testdata.RuleErrorKey1.Tags,
		},
		Groups: []groups.Group{},
		Status: "ok",
	}

	GetRuleContentRecommendationContentWithUserData1 = struct {
		Content types.RecommendationContentUserData `json:"content"`
		Groups  []groups.Group                      `json:"groups"`
		Status  string                              `json:"status"`
	}{
		Content: types.RecommendationContentUserData{
			RuleSelector:   ctypes.RuleSelector(testdata.Rule1CompositeID),
			Description:    testdata.RuleErrorKey1.Description,
			Generic:        testdata.RuleErrorKey1.Generic,
			Reason:         testdata.RuleErrorKey1.Reason,
			Resolution:     testdata.RuleErrorKey1.Resolution,
			MoreInfo:       testdata.RuleErrorKey1.MoreInfo,
			TotalRisk:      uint8(calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood)),
			ResolutionRisk: uint8(testdata.RuleErrorKey1.ResolutionRisk),
			Impact:         uint8(testdata.RuleErrorKey1.Impact),
			Likelihood:     uint8(testdata.RuleErrorKey1.Likelihood),
			PublishDate:    testdata.RuleErrorKey1.PublishDate,
			Rating:         types.UserVoteNone,
			AckedCount:     0,
			Tags:           testdata.RuleErrorKey1.Tags,
			Disabled:       false,
		},
		Groups: []groups.Group{},
		Status: "ok",
	}

	GetRuleContentRecommendationContentWithUserData2RatingLike = struct {
		Content types.RecommendationContentUserData `json:"content"`
		Groups  []groups.Group                      `json:"groups"`
		Status  string                              `json:"status"`
	}{
		Content: types.RecommendationContentUserData{
			RuleSelector:   ctypes.RuleSelector(testdata.Rule1CompositeID),
			Description:    testdata.RuleErrorKey1.Description,
			Generic:        testdata.RuleErrorKey1.Generic,
			Reason:         testdata.RuleErrorKey1.Reason,
			Resolution:     testdata.RuleErrorKey1.Resolution,
			MoreInfo:       testdata.RuleErrorKey1.MoreInfo,
			TotalRisk:      uint8(calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood)),
			ResolutionRisk: uint8(testdata.RuleErrorKey1.ResolutionRisk),
			Impact:         uint8(testdata.RuleErrorKey1.Impact),
			Likelihood:     uint8(testdata.RuleErrorKey1.Likelihood),
			PublishDate:    testdata.RuleErrorKey1.PublishDate,
			Rating:         types.UserVoteLike,
			AckedCount:     0,
			Tags:           testdata.RuleErrorKey1.Tags,
			Disabled:       false,
		},
		Groups: []groups.Group{},
		Status: "ok",
	}

	GetRuleContentRecommendationContentWithUserData3RatingDislike = struct {
		Content types.RecommendationContentUserData `json:"content"`
		Groups  []groups.Group                      `json:"groups"`
		Status  string                              `json:"status"`
	}{
		Content: types.RecommendationContentUserData{
			RuleSelector:   ctypes.RuleSelector(testdata.Rule1CompositeID),
			Description:    testdata.RuleErrorKey1.Description,
			Generic:        testdata.RuleErrorKey1.Generic,
			Reason:         testdata.RuleErrorKey1.Reason,
			Resolution:     testdata.RuleErrorKey1.Resolution,
			MoreInfo:       testdata.RuleErrorKey1.MoreInfo,
			TotalRisk:      uint8(calculateTotalRisk(testdata.RuleErrorKey1.Impact, testdata.RuleErrorKey1.Likelihood)),
			ResolutionRisk: uint8(testdata.RuleErrorKey1.ResolutionRisk),
			Impact:         uint8(testdata.RuleErrorKey1.Impact),
			Likelihood:     uint8(testdata.RuleErrorKey1.Likelihood),
			PublishDate:    testdata.RuleErrorKey1.PublishDate,
			Rating:         types.UserVoteDislike,
			AckedCount:     0,
			Tags:           testdata.RuleErrorKey1.Tags,
			Disabled:       false,
		},
		Groups: []groups.Group{},
		Status: "ok",
	}

	GetClustersResponse0Clusters = struct {
		Meta     map[string]interface{}  `json:"meta"`
		Status   string                  `json:"status"`
		Clusters []types.ClusterListView `json:"data"`
	}{
		Meta: map[string]interface{}{
			"count": 0,
		},
		Status:   "ok",
		Clusters: []types.ClusterListView{},
	}

	// cluster data filled in in test cases
	GetClustersResponse2ClusterNoHits = struct {
		Meta     map[string]interface{}  `json:"meta"`
		Status   string                  `json:"status"`
		Clusters []types.ClusterListView `json:"data"`
	}{
		Meta: map[string]interface{}{
			"count": 2,
		},
		Status: "ok",
		Clusters: []types.ClusterListView{
			{
				ClusterID:       "",
				ClusterName:     "",
				LastCheckedAt:   testTimestamp,
				TotalHitCount:   0,
				HitsByTotalRisk: map[int]int{},
			},
			{
				ClusterID:       "",
				ClusterName:     "",
				LastCheckedAt:   testTimestamp,
				TotalHitCount:   0,
				HitsByTotalRisk: map[int]int{},
			},
		},
	}

	// cluster data filled in in test cases, last_checked_at is empty and thus ommitted
	GetClustersResponse2ClusterNoArchiveInDB = struct {
		Meta     map[string]interface{}  `json:"meta"`
		Status   string                  `json:"status"`
		Clusters []types.ClusterListView `json:"data"`
	}{
		Meta: map[string]interface{}{
			"count": 2,
		},
		Status: "ok",
		Clusters: []types.ClusterListView{
			{
				ClusterID:       "",
				ClusterName:     "",
				TotalHitCount:   0,
				HitsByTotalRisk: map[int]int{},
			},
			{
				ClusterID:       "",
				ClusterName:     "",
				TotalHitCount:   0,
				HitsByTotalRisk: map[int]int{},
			},
		},
	}

	// cluster data filled in in test cases
	GetClustersResponse2ClusterWithHits = struct {
		Meta     map[string]interface{}  `json:"meta"`
		Status   string                  `json:"status"`
		Clusters []types.ClusterListView `json:"data"`
	}{
		Meta: map[string]interface{}{
			"count": 2,
		},
		Status: "ok",
		Clusters: []types.ClusterListView{
			{
				ClusterID:     "",
				ClusterName:   "",
				LastCheckedAt: testTimestamp,
				TotalHitCount: 1,
				// HitsByTotalRisk always has all unique total risks to have consistent response
				HitsByTotalRisk: map[int]int{
					1: 1,
					2: 0,
				},
			},
			{
				ClusterID:     "",
				ClusterName:   "",
				LastCheckedAt: testTimestamp,
				TotalHitCount: 2,
				HitsByTotalRisk: map[int]int{
					1: 0,
					2: 2,
				},
			},
		},
	}

	// cluster data filled in in test cases
	GetClustersResponse2ClusterWithHitsCluster1Managed = struct {
		Meta     map[string]interface{}  `json:"meta"`
		Status   string                  `json:"status"`
		Clusters []types.ClusterListView `json:"data"`
	}{
		Meta: map[string]interface{}{
			"count": 2,
		},
		Status: "ok",
		Clusters: []types.ClusterListView{
			{
				ClusterID:     "",
				ClusterName:   "",
				LastCheckedAt: testTimestamp,
				TotalHitCount: 1,
				// HitsByTotalRisk always has all unique total risks to have consistent response
				HitsByTotalRisk: map[int]int{
					1: 1,
					2: 0,
				},
			},
			{
				ClusterID:     "",
				ClusterName:   "",
				LastCheckedAt: testTimestamp,
				TotalHitCount: 2,
				HitsByTotalRisk: map[int]int{
					1: 1,
					2: 1,
				},
			},
		},
	}

	GetClustersResponse2ClusterWithHitsCluster1WithVersion = struct {
		Meta     map[string]interface{}  `json:"meta"`
		Status   string                  `json:"status"`
		Clusters []types.ClusterListView `json:"data"`
	}{
		Meta: map[string]interface{}{
			"count": 2,
		},
		Status: "ok",
		Clusters: []types.ClusterListView{
			{
				ClusterID:     "",
				ClusterName:   "",
				LastCheckedAt: testTimestamp,
				TotalHitCount: 1,
				// HitsByTotalRisk always has all unique total risks to have consistent response
				HitsByTotalRisk: map[int]int{
					1: 1,
					2: 0,
				},
				Version: testdata.ClusterVersion,
			},
			{
				ClusterID:     "",
				ClusterName:   "",
				LastCheckedAt: testTimestamp,
				TotalHitCount: 2,
				HitsByTotalRisk: map[int]int{
					1: 1,
					2: 1,
				},
			},
		},
	}

	// cluster data filled in in test cases
	GetClustersResponse2ClusterWithHits1Rule = struct {
		Meta     map[string]interface{}  `json:"meta"`
		Status   string                  `json:"status"`
		Clusters []types.ClusterListView `json:"data"`
	}{
		Meta: map[string]interface{}{
			"count": 2,
		},
		Status: "ok",
		Clusters: []types.ClusterListView{
			{
				ClusterID:     "",
				ClusterName:   "",
				LastCheckedAt: testTimestamp,
				TotalHitCount: 0,
				// HitsByTotalRisk always has all unique total risks to have consistent response
				HitsByTotalRisk: map[int]int{
					1: 0,
					2: 0,
				},
			},
			{
				ClusterID:     "",
				ClusterName:   "",
				LastCheckedAt: testTimestamp,
				TotalHitCount: 1,
				HitsByTotalRisk: map[int]int{
					1: 0,
					2: 1,
				},
			},
		},
	}

	// cluster data filled in in test cases
	GetClustersResponse2ClusterWithHits1RuleDisabled = struct {
		Meta     map[string]interface{}  `json:"meta"`
		Status   string                  `json:"status"`
		Clusters []types.ClusterListView `json:"data"`
	}{
		Meta: map[string]interface{}{
			"count": 2,
		},
		Status: "ok",
		Clusters: []types.ClusterListView{
			{
				ClusterID:     "",
				ClusterName:   "",
				LastCheckedAt: testTimestamp,
				TotalHitCount: 1,
				// HitsByTotalRisk always has all unique total risks to have consistent response
				HitsByTotalRisk: map[int]int{
					1: 1,
					2: 0,
				},
			},
			{
				ClusterID:     "",
				ClusterName:   "",
				LastCheckedAt: testTimestamp,
				TotalHitCount: 1,
				HitsByTotalRisk: map[int]int{
					1: 0,
					2: 1,
				},
			},
		},
	}

	ReportResponseMetainfoNoReports = ctypes.ReportResponseMetainfo{
		Count:         -1,
		LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		StoredAt:      types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
	}

	ReportMetainfoAPIResponseNoReports = struct {
		Status   string                         `json:"status"`
		Metainfo *ctypes.ReportResponseMetainfo `json:"metainfo"`
	}{
		Status:   "ok",
		Metainfo: &ReportResponseMetainfoNoReports,
	}

	ReportResponseMetainfoTwoReports = ctypes.ReportResponseMetainfo{
		Count:         2,
		LastCheckedAt: types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
		StoredAt:      types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)),
	}

	ReportMetainfoAPIResponseTwoReports = struct {
		Status   string                         `json:"status"`
		Metainfo *ctypes.ReportResponseMetainfo `json:"metainfo"`
	}{
		Status:   "ok",
		Metainfo: &ReportResponseMetainfoTwoReports,
	}

	ReportMetainfoAPIResponseInvalidJSON = struct {
		Status string `json:"status"`
	}{
		Status: "invalid character 'T' looking for beginning of value",
	}

	ReportMetainfoAPIResponseInvalidClusterName = struct {
		Status string `json:"status"`
	}{
		Status: "Error during parsing param 'cluster' with value 'not-proper-cluster-name'. Error: 'invalid UUID length: 23'",
	}
)

// TODO: move to utils
func calculateTotalRisk(impact, likelihood int) int {
	return (impact + likelihood) / 2
}

func createRuleContentDirectoryFromRuleContent(rulesContent []ctypes.RuleContent) *ctypes.RuleContentDirectory {
	rules := make(map[string]ctypes.RuleContent)

	for index, rule := range rulesContent {
		key := fmt.Sprintf("rc%d", index)
		rules[key] = rule
	}
	ruleContentDirectory := ctypes.RuleContentDirectory{
		Config: ctypes.GlobalRuleConfig{
			Impact: testdata.ImpactStrToInt,
		},
		Rules: rules,
	}
	return &ruleContentDirectory
}

func loadMockRuleContentDir(ruleContentDir *ctypes.RuleContentDirectory) error {
	content.SetRuleContentDirectory(ruleContentDir)
	err := content.WaitForContentDirectoryToBeReady()
	if err != nil {
		return err
	}
	content.LoadRuleContent(ruleContentDir)
	return nil
}

func init() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func TestServerStartError(t *testing.T) {
	testServer := server.New(server.Configuration{
		Address:     "localhost:99999",
		APIv1Prefix: "",
		APIv2Prefix: "",
	}, services.Configuration{
		AggregatorBaseEndpoint: "http://localhost:8081/api/v1/",
		ContentBaseEndpoint:    "http://localhost:8082/api/v1/",
		// GroupsPollingTime:      2 * time.Minute,
	},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	err := testServer.Start()
	assert.EqualError(t, err, "listen tcp: address 99999: invalid port")
}

func TestAddCORSHeaders(t *testing.T) {
	helpers.AssertAPIRequest(t, &helpers.DefaultServerConfigCORS, &helpers.DefaultServicesConfig, nil, nil, nil, &helpers.APIRequest{
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

func TestHTTPServer_SetAMSInfoInReportNoAMSClient(t *testing.T) {
	report := types.SmartProxyReportV2{}
	config := helpers.DefaultServerConfig
	testServer := helpers.CreateHTTPServer(&config, nil, nil, nil, nil, nil, nil, nil)
	testServer.SetAMSInfoInReport(testdata.ClusterName, &report)
	assert.Equal(t, string(testdata.ClusterName), report.Meta.DisplayName)
}

func TestHTTPServer_SetAMSInfoInReportAMSClientClusterIDFound(t *testing.T) {
	report := types.SmartProxyReportV2{}
	config := helpers.DefaultServerConfig
	// prepare list of organizations response
	amsClientMock := helpers.AMSClientWithOrgResults(
		testdata.OrgID,
		data.ClusterInfoResult,
	)
	testServer := helpers.CreateHTTPServer(&config, nil, amsClientMock, nil, nil, nil, nil, nil)
	testServer.SetAMSInfoInReport(testdata.ClusterName, &report)
	assert.Equal(t, data.ClusterDisplayName1, report.Meta.DisplayName)
}

// TestInfoEndpointNoAuthToken checks that the info endpoint can be accessed without authenticating
func TestInfoEndpointNoAuthToken(t *testing.T) {
	t.Run("test the info endpoint v1", func(t *testing.T) {
		helpers.AssertAPIRequest(t, &helpers.DefaultServerConfig, &helpers.DefaultServicesConfig, nil, nil, nil, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: server.InfoEndpoint,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
		})
	})
	t.Run("test the info endpoint v2", func(t *testing.T) {
		helpers.AssertAPIv2Request(t, &helpers.DefaultServerConfig, &helpers.DefaultServicesConfig, nil, nil, nil, &helpers.APIRequest{
			Method:   http.MethodGet,
			Endpoint: server.InfoEndpoint,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
		})
	})
}

// TestComposeEndpoint checks that composeEndpoint method correctly remove the API prefix
func TestComposeEndpoint(t *testing.T) {
	type testCase struct {
		name     string
		endpoint string
	}
	testCases := []testCase{
		{
			"test removal of v1 API prefix",
			"/api/v1/endpoint",
		},
		{
			"test removal of v2 API prefix",
			"/api/dbg/endpoint",
		},
		{
			"test removal of dbg API prefix",
			"/api/dbg/endpoint",
		},
		{
			"test that nothing is removed if prefix is missing",
			"/endpoint",
		},
	}

	testServer := helpers.CreateHTTPServer(&helpers.DefaultServerConfig, nil, nil, nil, nil, nil, nil, nil)
	baseEndpoint := helpers.DefaultServicesConfig.AggregatorBaseEndpoint
	expectedResponse := "http://localhost:8080/endpoint"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			endpoint, err := server.ComposeEndpoint(testServer, baseEndpoint, tc.endpoint)
			assert.NoError(t, err)
			assert.Equal(t, expectedResponse, endpoint.String())
		})
	}
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
		Content []types.RuleContentV1
	}

	var expectedResp Response
	var gotResp Response

	if err := json.Unmarshal(expected, &expectedResp); err != nil {
		err = fmt.Errorf(`"expected" is not JSON. value = "%v", err = "%v"`, expected, err)
		helpers.FailOnError(t, err)
	}

	if err := json.Unmarshal(got, &gotResp); err != nil {
		err = fmt.Errorf(`"got" is not JSON. value = "%v", err = "%v"`, string(got), err)
		helpers.FailOnError(t, err)
	}

	assert.ElementsMatch(t, expectedResp.Content, gotResp.Content)
}

func recommendationInResponseChecker(t testing.TB, expected, got []byte) {
	type Response struct {
		Status          string                         `json:"status"`
		Recommendations []types.RecommendationListView `json:"recommendations"`
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

	assert.ElementsMatch(t, expectedResp.Recommendations, gotResp.Recommendations)
}

func clusterInResponseChecker(t testing.TB, expected, got []byte) {
	type Response struct {
		Meta     map[string]interface{}  `json:"meta"`
		Status   string                  `json:"status"`
		Clusters []types.ClusterListView `json:"data"`
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

	assert.ElementsMatch(t, expectedResp.Clusters, gotResp.Clusters)
}

func ackInResponseChecker(t testing.TB, expected, got []byte) {
	var expectedResp, gotResp ctypes.AcknowledgementsResponse

	if err := json.Unmarshal(expected, &expectedResp); err != nil {
		err = fmt.Errorf(`"expected" is not JSON. value = "%v", err = "%v"`, expected, err)
		helpers.FailOnError(t, err)
	}

	if err := json.Unmarshal(got, &gotResp); err != nil {
		err = fmt.Errorf(`"got" is not JSON. value = "%v", err = "%v"`, got, err)
		helpers.FailOnError(t, err)
	}

	assert.ElementsMatch(t, expectedResp.Data, gotResp.Data)
	assert.Equal(t, expectedResp.Metadata.Count, gotResp.Metadata.Count)
}

func TestFillImpacted(t *testing.T) {
	var response []types.RuleWithContentResponse
	var aggregatorReport []ctypes.RuleOnReport

	resp0 := types.RuleWithContentResponse{
		RuleID:   "rid0",
		ErrorKey: "ek0",
	}
	resp1 := types.RuleWithContentResponse{
		RuleID:   "rid1",
		ErrorKey: "ek1",
	}
	resp2 := types.RuleWithContentResponse{
		RuleID:   "rid2",
		ErrorKey: "ek2",
	}
	respNa := types.RuleWithContentResponse{
		RuleID:   "111",
		ErrorKey: "111",
	}
	report0 := ctypes.RuleOnReport{
		Module:    "rid0",
		ErrorKey:  "ek0",
		CreatedAt: types.Timestamp(time.Now().UTC().Format(time.RFC3339)),
	}
	report1 := ctypes.RuleOnReport{
		Module:    "rid1",
		ErrorKey:  "ek1",
		CreatedAt: types.Timestamp(time.Time{}.UTC().Format(time.RFC3339)),
	}
	report2 := ctypes.RuleOnReport{
		Module:    "rid2",
		ErrorKey:  "ek2",
		CreatedAt: "wrong time format",
	}
	reportNa := ctypes.RuleOnReport{
		Module:   "000",
		ErrorKey: "000",
	}

	response = append(response, resp0, resp1, resp2, respNa)
	aggregatorReport = append(aggregatorReport, report0, report1, report2, reportNa)

	server.FillImpacted(response, aggregatorReport)
	assert.Equal(t, response[0].Impacted, report0.CreatedAt)
	assert.True(t, len(response[1].Impacted) == 0)
	assert.True(t, len(response[2].Impacted) == 0)

	jsonResp, err := json.Marshal(response)
	assert.NoError(t, err)
	assert.NotContains(t, string(jsonResp), "0001-01-01T00:00:00Z")
}
