// Copyright 2023 Red Hat, Inc
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

//deleteAcknowledge

//generateRuleAckMap

package server_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
)

func TestHTTPServer_TestReadAckListNoResult(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	ackedRulesAggregatorResponse := `
	{
		"disabledRules":[],
		"status":"ok"
	}
	`

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
			EndpointArgs: []interface{}{testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ackedRulesAggregatorResponse,
		},
	)

	ackListResponse := `
	{
		"data":[],
		"meta": {
			"count": 0
		}
	}
	`
	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodGet,
		Endpoint:           server.AckListEndpoint,
		AuthorizationToken: goodJWTAuthBearer,
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       ackListResponse,
	})
}

func TestHTTPServer_TestReadAckList1Result(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	disabledAt := time.Now()
	disabledAtRFC := disabledAt.UTC().Format(time.RFC3339)
	justificationNote := "justification test"

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	ackedRulesAggregatorResponse := `
	{
		"disabledRules":[
			{
				"rule_id": "%v",
				"error_key": "%v",
				"justification": "%v",
				"created_at": {
					"Time": "%v",
					"Valid": true
				},
				"updated_at": {
					"Time": "%v",
					"Valid": true
				}
			}
		],
		"status":"ok"
	}
	`
	ackedRulesAggregatorResponse = fmt.Sprintf(
		ackedRulesAggregatorResponse, testdata.Rule1ID, testdata.ErrorKey1, justificationNote, disabledAtRFC, disabledAtRFC,
	)
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
			EndpointArgs: []interface{}{testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ackedRulesAggregatorResponse,
		},
	)

	ackListResponse := `
	{
		"data":[
			{
				"rule": "%v",
				"justification": "%v",
				"created_by": "",
				"created_at": "%v",
				"updated_at": "%v"
			}
		],
		"meta": {
			"count": 1
		}
	}
	`
	ackListResponse = fmt.Sprintf(ackListResponse, testdata.Rule1CompositeID, justificationNote, disabledAtRFC, disabledAtRFC)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodGet,
		Endpoint:           server.AckListEndpoint,
		AuthorizationToken: goodJWTAuthBearer,
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       ackListResponse,
	})
}

func TestHTTPServer_TestReadAckList2Results(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	disabledAt := time.Now()
	disabledAtRFC := disabledAt.UTC().Format(time.RFC3339)
	justificationNote := "justification test"
	justificationNote2 := "different justification"

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	ackedRulesAggregatorResponse := `
	{
		"disabledRules":[
			{
				"rule_id": "%v",
				"error_key": "%v",
				"justification": "%v",
				"created_at": {
					"Time": "%v",
					"Valid": true
				},
				"updated_at": {
					"Time": "%v",
					"Valid": true
				}
			},
			{
				"rule_id": "%v",
				"error_key": "%v",
				"justification": "%v",
				"created_at": {
					"Time": "%v",
					"Valid": true
				},
				"updated_at": {
					"Time": "%v",
					"Valid": true
				}
			}
		],
		"status":"ok"
	}
	`
	ackedRulesAggregatorResponse = fmt.Sprintf(ackedRulesAggregatorResponse,
		testdata.Rule1ID, testdata.ErrorKey1, justificationNote, disabledAtRFC, disabledAtRFC,
		// 2nd entry
		testdata.Rule2ID, testdata.ErrorKey2, justificationNote2, disabledAtRFC, disabledAtRFC,
	)
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
			EndpointArgs: []interface{}{testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ackedRulesAggregatorResponse,
		},
	)

	ackListResponse := `
	{
		"data":[
			{
				"rule": "%v",
				"justification": "%v",
				"created_by": "",
				"created_at": "%v",
				"updated_at": "%v"
			},
			{
				"rule": "%v",
				"justification": "%v",
				"created_by": "",
				"created_at": "%v",
				"updated_at": "%v"
			}
		],
		"meta": {
			"count": 2
		}
	}
	`
	ackListResponse = fmt.Sprintf(ackListResponse,
		testdata.Rule1CompositeID, justificationNote, disabledAtRFC, disabledAtRFC,
		testdata.Rule2CompositeID, justificationNote2, disabledAtRFC, disabledAtRFC,
	)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodGet,
		Endpoint:           server.AckListEndpoint,
		AuthorizationToken: goodJWTAuthBearer,
	}, &helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        ackListResponse,
		BodyChecker: ackInResponseChecker,
	})
}

func TestHTTPServer_TestReadAckListInvalidToken(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	ackedRulesAggregatorResponse := `
	{
		"disabledRules":[],
		"status":"ok"
	}
	`

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
			EndpointArgs: []interface{}{testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ackedRulesAggregatorResponse,
		},
	)

	ackListResponse := `
	{
		"status": "Malformed authentication token"
	}
	`
	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodGet,
		Endpoint:           server.AckListEndpoint,
		AuthorizationToken: badJWTAuthBearer,
	}, &helpers.APIResponse{
		StatusCode: http.StatusForbidden,
		Body:       ackListResponse,
	})
}

func TestHTTPServer_TestReadAckListAggregatorError(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
			EndpointArgs: []interface{}{testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusInternalServerError,
		},
	)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodGet,
		Endpoint:           server.AckListEndpoint,
		AuthorizationToken: goodJWTAuthBearer,
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
	})
}

func TestHTTPServer_TestReadAckListUnparsableAggregatorJSON(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ListOfDisabledRulesSystemWide,
			EndpointArgs: []interface{}{testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       "{invalid json body",
		},
	)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodGet,
		Endpoint:           server.AckListEndpoint,
		AuthorizationToken: goodJWTAuthBearer,
	}, &helpers.APIResponse{
		// also 500, but testing different condition
		StatusCode: http.StatusInternalServerError,
	})
}
