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

func TestHTTPServer_TestGetAcknowledgeNotFound(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{},
		"status":"ok"
	}
	`

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusNotFound,
			Body:       ackedRuleAggregatorResponse,
		},
	)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodGet,
		Endpoint:           server.AckGetEndpoint,
		AuthorizationToken: goodJWTAuthBearer,
		EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusNotFound,
	})
}

func TestHTTPServer_TestGetAcknowledgeAggregatorError(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusInternalServerError,
		},
	)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodGet,
		Endpoint:           server.AckGetEndpoint,
		AuthorizationToken: goodJWTAuthBearer,
		EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
	})
}

func TestHTTPServer_TestGetAcknowledgeUnparsableAggregatorJSON(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       "{invalid json body",
		},
	)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodGet,
		Endpoint:           server.AckGetEndpoint,
		AuthorizationToken: goodJWTAuthBearer,
		EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
	}, &helpers.APIResponse{
		// also 500, but testing different condition
		StatusCode: http.StatusInternalServerError,
	})
}

func TestHTTPServer_TestGetAcknowledgeFound(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	disabledAt := time.Now()
	disabledAtRFC := disabledAt.UTC().Format(time.RFC3339)
	justificationNote := "justification test"

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{
			"rule_id": "%v",
			"error_key": "%v",
			"justification": "%v",
			"created_by": "",
			"created_at": {
				"Time": "%v",
				"Valid": true
			},
			"updated_at": {
				"Time": "%v",
				"Valid": true
			}
		},
		"status":"ok"
	}
	`
	ackedRuleAggregatorResponse = fmt.Sprintf(ackedRuleAggregatorResponse,
		testdata.Rule1ID, testdata.ErrorKey1, justificationNote, disabledAtRFC, disabledAtRFC,
	)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ackedRuleAggregatorResponse,
		},
	)

	expectedResponse := `
	{
		"rule": "%v",
		"justification": "%v",
		"created_by": "",
		"created_at": "%v",
		"updated_at": "%v"
	}
	`
	expectedResponse = fmt.Sprintf(expectedResponse,
		testdata.Rule1CompositeID, justificationNote, disabledAtRFC, disabledAtRFC,
	)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodGet,
		Endpoint:           server.AckGetEndpoint,
		AuthorizationToken: goodJWTAuthBearer,
		EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       expectedResponse,
	})
}

func TestHTTPServer_TestGetAcknowledgeInvalidRuleIDBadRequest(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	disabledAt := time.Now()
	disabledAtRFC := disabledAt.UTC().Format(time.RFC3339)
	justificationNote := "justification test"

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{
			"rule_id": "%v",
			"error_key": "%v",
			"justification": "%v",
			"created_by": "",
			"created_at": {
				"Time": "%v",
				"Valid": true
			},
			"updated_at": {
				"Time": "%v",
				"Valid": true
			}
		},
		"status":"ok"
	}
	`
	ackedRuleAggregatorResponse = fmt.Sprintf(ackedRuleAggregatorResponse,
		testdata.Rule1ID, testdata.ErrorKey1, justificationNote, disabledAtRFC, disabledAtRFC,
	)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ackedRuleAggregatorResponse,
		},
	)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodGet,
		Endpoint:           server.AckGetEndpoint,
		AuthorizationToken: goodJWTAuthBearer,
		// invalid composite rule ID
		EndpointArgs: []interface{}{testdata.Rule1ID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
	})
}

func TestHTTPServer_TestGetAcknowledgeInvalidToken(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusNotFound,
		},
	)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodGet,
		Endpoint:           server.AckGetEndpoint,
		AuthorizationToken: invalidJWTAuthBearer,
		EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusForbidden,
	})
}

func TestHTTPServer_TestAcknowledgePostFound(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	disabledAt := time.Now()
	disabledAtRFC := disabledAt.UTC().Format(time.RFC3339)
	justificationNote := "justification test"

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{
			"rule_id": "%v",
			"error_key": "%v",
			"justification": "%v",
			"created_by": "",
			"created_at": {
				"Time": "%v",
				"Valid": true
			},
			"updated_at": {
				"Time": "%v",
				"Valid": true
			}
		},
		"status":"ok"
	}
	`
	ackedRuleAggregatorResponse = fmt.Sprintf(ackedRuleAggregatorResponse,
		testdata.Rule1ID, testdata.ErrorKey1, justificationNote, disabledAtRFC, disabledAtRFC,
	)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ackedRuleAggregatorResponse,
		},
	)

	// rule has been acked before, 2nd call to aggregator happens in any case
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ackedRuleAggregatorResponse,
		},
	)

	reqBody := `
	{
		"rule_id": "%v",
		"justification": "%v"
	}
	`
	reqBody = fmt.Sprintf(reqBody, testdata.Rule1CompositeID, justificationNote)

	expectedResponse := `
	{
		"rule": "%v",
		"justification": "%v",
		"created_by": "",
		"created_at": "%v",
		"updated_at": "%v"
	}
	`
	expectedResponse = fmt.Sprintf(expectedResponse,
		testdata.Rule1CompositeID, justificationNote, disabledAtRFC, disabledAtRFC,
	)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodPost,
		Endpoint:           server.AckAcknowledgePostEndpoint,
		AuthorizationToken: goodJWTAuthBearer,
		Body:               reqBody,
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       expectedResponse,
	})
}

func TestHTTPServer_TestAcknowledgePostNewAck(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	disabledAt := time.Now()
	disabledAtRFC := disabledAt.UTC().Format(time.RFC3339)
	justificationNote := "justification test"

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	emptyAggregatorResponse := `
	{
		"disabledRule":{},
		"status":"ok"
	}
	`
	// 1st call to aggregator to find out whether rule has been acked already or not
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusNotFound,
			Body:       emptyAggregatorResponse,
		},
	)

	putBody := `{"justification":"%v"}`
	putBody = fmt.Sprintf(putBody, justificationNote)

	// PUT to aggregator
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodPut,
			Endpoint:     ira_server.DisableRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
			Body:         putBody,
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
		},
	)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{
			"rule_id": "%v",
			"error_key": "%v",
			"justification": "%v",
			"created_by": "",
			"created_at": {
				"Time": "%v",
				"Valid": true
			},
			"updated_at": {
				"Time": "%v",
				"Valid": true
			}
		},
		"status":"ok"
	}
	`
	ackedRuleAggregatorResponse = fmt.Sprintf(ackedRuleAggregatorResponse,
		testdata.Rule1ID, testdata.ErrorKey1, justificationNote, disabledAtRFC, disabledAtRFC,
	)

	// 2nd call to aggregator to confirm data entered DB
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ackedRuleAggregatorResponse,
		},
	)

	reqBody := `
	{
		"rule_id": "%v",
		"justification": "%v"
	}
	`
	reqBody = fmt.Sprintf(reqBody, testdata.Rule1CompositeID, justificationNote)

	expectedResponse := `
	{
		"rule": "%v",
		"justification": "%v",
		"created_by": "",
		"created_at": "%v",
		"updated_at": "%v"
	}
	`
	expectedResponse = fmt.Sprintf(expectedResponse,
		testdata.Rule1CompositeID, justificationNote, disabledAtRFC, disabledAtRFC,
	)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodPost,
		Endpoint:           server.AckAcknowledgePostEndpoint,
		AuthorizationToken: goodJWTAuthBearer,
		Body:               reqBody,
	}, &helpers.APIResponse{
		// 201 CREATED when rule wasn't acked before
		StatusCode: http.StatusCreated,
		Body:       expectedResponse,
	})
}

func TestHTTPServer_TestAcknowledgePostMissingParam(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	justificationNote := "justification test"

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	// missing rule_id
	reqBody := `
	{
		"justification": "%v"
	}
	`
	reqBody = fmt.Sprintf(reqBody, justificationNote)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodPost,
		Endpoint:           server.AckAcknowledgePostEndpoint,
		AuthorizationToken: goodJWTAuthBearer,
		Body:               reqBody,
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
	})
}

func TestHTTPServer_TestAcknowledgePostBadCompositeRuleID(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	justificationNote := "justification test"

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	reqBody := `
	{
		"rule_id": "invalid rule id"
		"justification": "%v"
	}
	`
	reqBody = fmt.Sprintf(reqBody, justificationNote)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodPost,
		Endpoint:           server.AckAcknowledgePostEndpoint,
		AuthorizationToken: goodJWTAuthBearer,
		Body:               reqBody,
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
	})
}

func TestHTTPServer_TestAcknowledgePostAggregatorError1stCall(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	justificationNote := "justification test"

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusInternalServerError,
		},
	)

	reqBody := `
	{
		"rule_id": "%v",
		"justification": "%v"
	}
	`
	reqBody = fmt.Sprintf(reqBody, testdata.Rule1CompositeID, justificationNote)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodPost,
		Endpoint:           server.AckAcknowledgePostEndpoint,
		AuthorizationToken: goodJWTAuthBearer,
		Body:               reqBody,
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
	})
}

func TestHTTPServer_TestAcknowledgePostAggregatorError2ndCall(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	disabledAt := time.Now()
	disabledAtRFC := disabledAt.UTC().Format(time.RFC3339)
	justificationNote := "justification test"

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{
			"rule_id": "%v",
			"error_key": "%v",
			"justification": "%v",
			"created_by": "",
			"created_at": {
				"Time": "%v",
				"Valid": true
			},
			"updated_at": {
				"Time": "%v",
				"Valid": true
			}
		},
		"status":"ok"
	}
	`
	ackedRuleAggregatorResponse = fmt.Sprintf(ackedRuleAggregatorResponse,
		testdata.Rule1ID, testdata.ErrorKey1, justificationNote, disabledAtRFC, disabledAtRFC,
	)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ackedRuleAggregatorResponse,
		},
	)

	emptyAggregatorResponse := `
	{
		"disabledRule":{},
		"status":"ok"
	}
	`
	// 2nd call fails
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       emptyAggregatorResponse,
		},
	)

	reqBody := `
	{
		"rule_id": "%v",
		"justification": "%v"
	}
	`
	reqBody = fmt.Sprintf(reqBody, testdata.Rule1CompositeID, justificationNote)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodPost,
		Endpoint:           server.AckAcknowledgePostEndpoint,
		AuthorizationToken: goodJWTAuthBearer,
		Body:               reqBody,
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
	})
}

func TestHTTPServer_TestAcknowledgePostInvalidToken(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	reqBody := `
	{
		"rule_id": "%v",
		"justification": "%v"
	}
	`
	reqBody = fmt.Sprintf(reqBody, testdata.Rule1CompositeID, "justification")

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodPost,
		Endpoint:           server.AckAcknowledgePostEndpoint,
		AuthorizationToken: invalidJWTAuthBearer,
		Body:               reqBody,
	}, &helpers.APIResponse{
		StatusCode: http.StatusForbidden,
	})
}

func TestHTTPServer_TestAcknowledgeUpdateNotFound(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	justificationNote := "justification test"

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	emptyAggregatorResponse := `
	{
		"disabledRule":{},
		"status":"ok"
	}
	`
	// 1st call to aggregator to find out whether rule has been acked already or not
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			// existing ack not found
			StatusCode: http.StatusNotFound,
			Body:       emptyAggregatorResponse,
		},
	)

	reqBody := `
	{
		"justification": "%v"
	}
	`
	reqBody = fmt.Sprintf(reqBody, justificationNote)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodPut,
		Endpoint:           server.AckUpdateEndpoint,
		EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
		AuthorizationToken: goodJWTAuthBearer,
		Body:               reqBody,
	}, &helpers.APIResponse{
		StatusCode: http.StatusNotFound,
	})
}

func TestHTTPServer_TestAcknowledgeUpdateFound(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	disabledAt := time.Now()
	disabledAtRFC := disabledAt.UTC().Format(time.RFC3339)
	justificationNote := "justification test"
	justificationUpdated := "justification updated"

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{
			"rule_id": "%v",
			"error_key": "%v",
			"justification": "%v",
			"created_by": "",
			"created_at": {
				"Time": "%v",
				"Valid": true
			},
			"updated_at": {
				"Time": "%v",
				"Valid": true
			}
		},
		"status":"ok"
	}
	`
	ackedRuleAggregatorResponse = fmt.Sprintf(ackedRuleAggregatorResponse,
		testdata.Rule1ID, testdata.ErrorKey1, justificationNote, disabledAtRFC, disabledAtRFC,
	)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ackedRuleAggregatorResponse,
		},
	)

	postBody := `{"justification":"%v"}`
	postBody = fmt.Sprintf(postBody, justificationUpdated)

	// POST to aggregator
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodPost,
			Endpoint:     ira_server.UpdateRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
			Body:         postBody,
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
		},
	)

	ackedRuleAggregatorResponseUpdated := `
	{
		"disabledRule":{
			"rule_id": "%v",
			"error_key": "%v",
			"justification": "%v",
			"created_by": "",
			"created_at": {
				"Time": "%v",
				"Valid": true
			},
			"updated_at": {
				"Time": "%v",
				"Valid": true
			}
		},
		"status":"ok"
	}
	`
	ackedRuleAggregatorResponseUpdated = fmt.Sprintf(ackedRuleAggregatorResponseUpdated,
		testdata.Rule1ID, testdata.ErrorKey1, justificationUpdated, disabledAtRFC, disabledAtRFC,
	)

	// 2nd call to aggregator to get results
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ackedRuleAggregatorResponseUpdated,
		},
	)

	reqBody := `
	{
		"justification": "%v"
	}
	`
	reqBody = fmt.Sprintf(reqBody, justificationUpdated)

	expectedResponse := `
	{
		"rule": "%v",
		"justification": "%v",
		"created_by": "",
		"created_at": "%v",
		"updated_at": "%v"
	}
	`
	expectedResponse = fmt.Sprintf(expectedResponse,
		testdata.Rule1CompositeID, justificationUpdated, disabledAtRFC, disabledAtRFC,
	)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodPut,
		Endpoint:           server.AckUpdateEndpoint,
		EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
		AuthorizationToken: goodJWTAuthBearer,
		Body:               reqBody,
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       expectedResponse,
	})
}

func TestHTTPServer_TestAcknowledgeUpdateBadCompositeRuleID(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	justificationNote := "justification test"

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	reqBody := `
	{
		"justification": "%v"
	}
	`
	reqBody = fmt.Sprintf(reqBody, justificationNote)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodPut,
		Endpoint:           server.AckUpdateEndpoint,
		EndpointArgs:       []interface{}{"invalid rule id"},
		AuthorizationToken: goodJWTAuthBearer,
		Body:               reqBody,
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
	})
}

func TestHTTPServer_TestAcknowledgeUpdateAggregatorError1st(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	justificationUpdated := "justification updated"

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusInternalServerError,
		},
	)

	reqBody := `
	{
		"justification": "%v"
	}
	`
	reqBody = fmt.Sprintf(reqBody, justificationUpdated)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodPut,
		Endpoint:           server.AckUpdateEndpoint,
		EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
		AuthorizationToken: goodJWTAuthBearer,
		Body:               reqBody,
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
	})
}

func TestHTTPServer_TestAcknowledgeUpdateAggregatorError2nd(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	disabledAt := time.Now()
	disabledAtRFC := disabledAt.UTC().Format(time.RFC3339)
	justificationNote := "justification test"
	justificationUpdated := "justification updated"

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{
			"rule_id": "%v",
			"error_key": "%v",
			"justification": "%v",
			"created_by": "",
			"created_at": {
				"Time": "%v",
				"Valid": true
			},
			"updated_at": {
				"Time": "%v",
				"Valid": true
			}
		},
		"status":"ok"
	}
	`
	ackedRuleAggregatorResponse = fmt.Sprintf(ackedRuleAggregatorResponse,
		testdata.Rule1ID, testdata.ErrorKey1, justificationNote, disabledAtRFC, disabledAtRFC,
	)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ackedRuleAggregatorResponse,
		},
	)

	postBody := `{"justification":"%v"}`
	postBody = fmt.Sprintf(postBody, justificationUpdated)

	// POST to aggregator
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodPost,
			Endpoint:     ira_server.UpdateRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
			Body:         postBody,
		},
		&helpers.APIResponse{
			StatusCode: http.StatusInternalServerError,
		},
	)

	reqBody := `
	{
		"justification": "%v"
	}
	`
	reqBody = fmt.Sprintf(reqBody, justificationUpdated)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodPut,
		Endpoint:           server.AckUpdateEndpoint,
		EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
		AuthorizationToken: goodJWTAuthBearer,
		Body:               reqBody,
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
	})
}

func TestHTTPServer_TestAcknowledgeUpdateAggregatorError3rd(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	disabledAt := time.Now()
	disabledAtRFC := disabledAt.UTC().Format(time.RFC3339)
	justificationNote := "justification test"
	justificationUpdated := "justification updated"

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{
			"rule_id": "%v",
			"error_key": "%v",
			"justification": "%v",
			"created_by": "",
			"created_at": {
				"Time": "%v",
				"Valid": true
			},
			"updated_at": {
				"Time": "%v",
				"Valid": true
			}
		},
		"status":"ok"
	}
	`
	ackedRuleAggregatorResponse = fmt.Sprintf(ackedRuleAggregatorResponse,
		testdata.Rule1ID, testdata.ErrorKey1, justificationNote, disabledAtRFC, disabledAtRFC,
	)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ackedRuleAggregatorResponse,
		},
	)

	postBody := `{"justification":"%v"}`
	postBody = fmt.Sprintf(postBody, justificationUpdated)

	// POST to aggregator
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodPost,
			Endpoint:     ira_server.UpdateRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
			Body:         postBody,
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
		},
	)

	// 2nd call to aggregator to get results fails
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusInternalServerError,
		},
	)

	reqBody := `
	{
		"justification": "%v"
	}
	`
	reqBody = fmt.Sprintf(reqBody, justificationUpdated)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodPut,
		Endpoint:           server.AckUpdateEndpoint,
		EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
		AuthorizationToken: goodJWTAuthBearer,
		Body:               reqBody,
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
	})
}

func TestHTTPServer_TestAcknowledgeUpdateInvalidToken(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	reqBody := `
	{
		"justification": "%v"
	}
	`
	reqBody = fmt.Sprintf(reqBody, "justification")

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodPut,
		Endpoint:           server.AckUpdateEndpoint,
		EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
		AuthorizationToken: invalidJWTAuthBearer,
		Body:               reqBody,
	}, &helpers.APIResponse{
		StatusCode: http.StatusForbidden,
	})
}

func TestHTTPServer_TestAcknowledgeDeleteFound(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	disabledAt := time.Now()
	disabledAtRFC := disabledAt.UTC().Format(time.RFC3339)
	justificationNote := "justification test"

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{
			"rule_id": "%v",
			"error_key": "%v",
			"justification": "%v",
			"created_by": "",
			"created_at": {
				"Time": "%v",
				"Valid": true
			},
			"updated_at": {
				"Time": "%v",
				"Valid": true
			}
		},
		"status":"ok"
	}
	`
	ackedRuleAggregatorResponse = fmt.Sprintf(ackedRuleAggregatorResponse,
		testdata.Rule1ID, testdata.ErrorKey1, justificationNote, disabledAtRFC, disabledAtRFC,
	)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ackedRuleAggregatorResponse,
		},
	)

	// PUT to aggregator
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodPut,
			Endpoint:     ira_server.EnableRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
		},
	)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodDelete,
		Endpoint:           server.AckDeleteEndpoint,
		EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
		AuthorizationToken: goodJWTAuthBearer,
	}, &helpers.APIResponse{
		StatusCode: http.StatusNoContent,
	})
}

func TestHTTPServer_TestAcknowledgeDeleteNotFound(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{},
		"status":"ok"
	}
	`
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusNotFound,
			Body:       ackedRuleAggregatorResponse,
		},
	)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodDelete,
		Endpoint:           server.AckDeleteEndpoint,
		EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
		AuthorizationToken: goodJWTAuthBearer,
	}, &helpers.APIResponse{
		StatusCode: http.StatusNotFound,
	})
}

func TestHTTPServer_TestAcknowledgeDeleteBadRequest(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodDelete,
		Endpoint:           server.AckDeleteEndpoint,
		EndpointArgs:       []interface{}{"invalid rule id"},
		AuthorizationToken: goodJWTAuthBearer,
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
	})
}

func TestHTTPServer_TestAcknowledgeDeleteInvalidToken(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	reqBody := `
	{
		"justification": "%v"
	}
	`
	reqBody = fmt.Sprintf(reqBody, "justification")

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodDelete,
		Endpoint:           server.AckDeleteEndpoint,
		EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
		AuthorizationToken: invalidJWTAuthBearer,
		Body:               reqBody,
	}, &helpers.APIResponse{
		StatusCode: http.StatusForbidden,
	})
}

func TestHTTPServer_TestAcknowledgeDeleteAggregatorError1st(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{},
		"status":"ok"
	}
	`
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       ackedRuleAggregatorResponse,
		},
	)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodDelete,
		Endpoint:           server.AckDeleteEndpoint,
		EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
		AuthorizationToken: goodJWTAuthBearer,
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
	})
}

func TestHTTPServer_TestAcknowledgeDeleteAggregatorError2nd(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	defer content.ResetContent()

	disabledAt := time.Now()
	disabledAtRFC := disabledAt.UTC().Format(time.RFC3339)
	justificationNote := "justification test"

	err := loadMockRuleContentDir(&testdata.RuleContentDirectory3Rules)
	assert.Nil(t, err)

	ackedRuleAggregatorResponse := `
	{
		"disabledRule":{
			"rule_id": "%v",
			"error_key": "%v",
			"justification": "%v",
			"created_by": "",
			"created_at": {
				"Time": "%v",
				"Valid": true
			},
			"updated_at": {
				"Time": "%v",
				"Valid": true
			}
		},
		"status":"ok"
	}
	`
	ackedRuleAggregatorResponse = fmt.Sprintf(ackedRuleAggregatorResponse,
		testdata.Rule1ID, testdata.ErrorKey1, justificationNote, disabledAtRFC, disabledAtRFC,
	)

	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     ira_server.ReadRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       ackedRuleAggregatorResponse,
		},
	)

	// PUT to aggregator
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodPut,
			Endpoint:     ira_server.EnableRuleSystemWide,
			EndpointArgs: []interface{}{testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID},
		},
		&helpers.APIResponse{
			StatusCode: http.StatusInternalServerError,
		},
	)

	helpers.AssertAPIv2Request(t, nil, nil, nil, nil, nil, &helpers.APIRequest{
		Method:             http.MethodDelete,
		Endpoint:           server.AckDeleteEndpoint,
		EndpointArgs:       []interface{}{testdata.Rule1CompositeID},
		AuthorizationToken: goodJWTAuthBearer,
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
	})
}
