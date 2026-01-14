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

// Package services contains interface implementations to other
// services that are called from Smart Proxy.
package services_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	utypes "github.com/RedHatInsights/insights-operator-utils/types"
	data "github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/testdata"
	"github.com/RedHatInsights/insights-results-smart-proxy/types"
	"github.com/stretchr/testify/assert"
)

var (
	errTest                = errors.New("ka-boom")
	receivedTimestampTest  = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	processedTimestampTest = time.Now().UTC().Format(time.RFC3339)
	testRuleHits           = fmt.Sprintf("%v,%v", data.Rule1CompositeID, data.Rule2CompositeID)
)

func TestNewRedisClient(t *testing.T) {
	client, err := services.NewRedisClient(helpers.DefaultRedisConf)
	assert.NotNil(t, client)
	assert.NoError(t, err)
}

func TestNewRedisClientBadAddress(t *testing.T) {
	conf := helpers.DefaultRedisConf
	conf.RedisEndpoint = ""
	// db index == -1
	client, err := services.NewRedisClient(conf)
	assert.Nil(t, client)
	assert.Error(t, err)
}

func TestNewRedisClientDBIndexOutOfRange(t *testing.T) {
	conf := helpers.DefaultRedisConf
	conf.RedisDatabase = -1
	// db index == -1
	client, err := services.NewRedisClient(conf)
	assert.Nil(t, client)
	assert.Error(t, err)
}

func TestRedisGetRequestIDsForClusterID_Empty(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName1)
	server.ExpectScan(0, expectedKey, services.ScanBatchCount).SetVal([]string{}, 0)

	requestIDs, err := client.GetRequestIDsForClusterID(testdata.OrgID, testdata.ClusterName1)
	assert.NoError(t, err)
	assert.Len(t, requestIDs, 0)

	helpers.RedisExpectationsMet(t, server)
}

func TestRedisGetRequestIDsForClusterID_ResultsSinglePage(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName1)

	expectedResponseKeys := make([]string, 2)
	for i := range expectedResponseKeys {
		expectedResponseKeys[i] = fmt.Sprintf("organization:%v:cluster:%v:request:requestID%v", testdata.OrgID, testdata.ClusterName1, i)
	}
	// all results are in a single page -- cursor == 0, so no more calls are expected
	server.ExpectScan(0, expectedKey, services.ScanBatchCount).SetVal(expectedResponseKeys, 0)

	requestIDs, err := client.GetRequestIDsForClusterID(testdata.OrgID, testdata.ClusterName1)
	assert.NoError(t, err)
	assert.Len(t, requestIDs, 2)
	assert.ElementsMatch(t, requestIDs, []types.RequestID{"requestID0", "requestID1"})

	helpers.RedisExpectationsMet(t, server)
}

func TestRedisGetRequestIDsForClusterID_FilterKeys(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName1)

	expectedResponseKeys := make([]string, 3)

	expectedResponseKeys[0] = fmt.Sprintf("organization:%v:cluster:%v:request:requestID0", testdata.OrgID, testdata.ClusterName1)
	expectedResponseKeys[1] = fmt.Sprintf("organization:%v:cluster:%v:request:requestIDe", testdata.OrgID, testdata.ClusterName1)
	expectedResponseKeys[2] = fmt.Sprintf("organization:%v:cluster:%v:request:requestID0:reports", testdata.OrgID, testdata.ClusterName1)

	// all results are in a single page -- cursor == 0, so no more calls are expected
	server.ExpectScan(0, expectedKey, services.ScanBatchCount).SetVal(expectedResponseKeys, 0)

	requestIDs, err := client.GetRequestIDsForClusterID(testdata.OrgID, testdata.ClusterName1)
	assert.NoError(t, err)
	assert.Len(t, requestIDs, 2)
	assert.ElementsMatch(t, requestIDs, []types.RequestID{"requestID0", "requestIDe"})

	helpers.RedisExpectationsMet(t, server)
}

func TestRedisGetRequestIDsForClusterID_ResultsMultiplePages(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedResponseKeys := make([]string, 4)
	for i := range expectedResponseKeys {
		expectedResponseKeys[i] = fmt.Sprintf("organization:%v:cluster:%v:request:requestID%v", testdata.OrgID, testdata.ClusterName1, i)
	}

	expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName1)
	server.ExpectScan(0, expectedKey, services.ScanBatchCount).SetVal([]string{expectedResponseKeys[0], expectedResponseKeys[1]}, 42)
	// returned cursor is expected to be used in the next call
	server.ExpectScan(42, expectedKey, services.ScanBatchCount).SetVal([]string{expectedResponseKeys[2]}, 8)
	// returned cursor is expected to be used in the next call
	server.ExpectScan(8, expectedKey, services.ScanBatchCount).SetVal([]string{expectedResponseKeys[3]}, 0)
	// returned cursor == 0, so no more calls are expected

	requestIDs, err := client.GetRequestIDsForClusterID(testdata.OrgID, testdata.ClusterName1)
	assert.NoError(t, err)
	assert.Len(t, requestIDs, len(expectedResponseKeys))
	assert.ElementsMatch(t, requestIDs, []types.RequestID{"requestID0", "requestID1", "requestID2", "requestID3"})

	helpers.RedisExpectationsMet(t, server)
}

func TestRedisGetRequestIDsForClusterID_Error(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName1)
	server.ExpectScan(0, expectedKey, services.ScanBatchCount).SetErr(errTest)

	requestIDs, err := client.GetRequestIDsForClusterID(testdata.OrgID, testdata.ClusterName1)
	assert.Error(t, err)
	assert.Len(t, requestIDs, 0)

	helpers.RedisExpectationsMet(t, server)
}

func TestRedisGetRequestIDsForClusterID_ErrorInFollowingCalls(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName1)

	expectedResponseKeys := make([]string, 2)
	for i := range expectedResponseKeys {
		expectedResponseKeys[i] = fmt.Sprintf("organization:%v:cluster:%v:request:requestID%v", testdata.OrgID, testdata.ClusterName1, i)
	}

	server.ExpectScan(0, expectedKey, services.ScanBatchCount).SetVal([]string{expectedResponseKeys[0], expectedResponseKeys[1]}, 42)
	server.ExpectScan(42, expectedKey, services.ScanBatchCount).SetErr(errTest)

	// function should return empty list + error if we can't retrieve the whole data set
	requestIDs, err := client.GetRequestIDsForClusterID(testdata.OrgID, testdata.ClusterName1)
	assert.Error(t, err)
	assert.Len(t, requestIDs, 0)

	helpers.RedisExpectationsMet(t, server)
}

func TestGetTimestampsForRequestIDs_OKFound(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName1, "requestID123")

	server.ExpectHMGet(
		expectedKey, services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
	).SetVal([]interface{}{"requestID123", receivedTimestampTest, processedTimestampTest})

	requestStatuses, err := client.GetTimestampsForRequestIDs(testdata.OrgID, testdata.ClusterName1, []types.RequestID{"requestID123"}, true)
	assert.NoError(t, err)
	assert.Len(t, requestStatuses, 1)
	assert.Equal(t, requestStatuses[0].RequestID, "requestID123")
	assert.Equal(t, requestStatuses[0].Received, receivedTimestampTest)
	assert.Equal(t, requestStatuses[0].Processed, processedTimestampTest)
	assert.Equal(t, requestStatuses[0].Valid, true)

	helpers.RedisExpectationsMet(t, server)
}

func TestGetTimestampsForRequestIDs_OKNotFoundOmitMissing(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName1, "requestID123")

	server.ExpectHMGet(
		expectedKey, services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
	).SetVal([]interface{}{nil, nil, nil})

	_, err := client.GetTimestampsForRequestIDs(testdata.OrgID, testdata.ClusterName1, []types.RequestID{"requestID123"}, true)
	assert.NoError(t, err)

	helpers.RedisExpectationsMet(t, server)
}

func TestGetTimestampsForRequestIDs_OKNotFoundIncludeMissing(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName1, "requestID123")

	server.ExpectHMGet(
		expectedKey, services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
	).SetVal([]interface{}{nil, nil, nil})

	requestStatuses, err := client.GetTimestampsForRequestIDs(testdata.OrgID, testdata.ClusterName1, []types.RequestID{"requestID123"}, false)
	assert.NoError(t, err)
	assert.Len(t, requestStatuses, 1)
	assert.Equal(t, requestStatuses[0].RequestID, "requestID123")
	assert.Equal(t, requestStatuses[0].Received, "")
	assert.Equal(t, requestStatuses[0].Processed, "")
	assert.Equal(t, requestStatuses[0].Valid, false)

	helpers.RedisExpectationsMet(t, server)
}

func TestGetTimestampsForRequestIDs_OKMultipleOmitMissing(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKeys := make([]string, 3)
	requestIDs := make([]string, 3)
	for i := range expectedKeys {
		requestIDs[i] = fmt.Sprintf("requestID%d", i)
		expectedKeys[i] = fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName1, requestIDs[i])
	}
	// second request_id won't be found
	expectedResponse := []types.RequestStatus{
		{
			RequestID: requestIDs[0],
			Valid:     true,
			Received:  receivedTimestampTest,
			Processed: processedTimestampTest,
		},
		{
			RequestID: requestIDs[2],
			Valid:     true,
			Received:  receivedTimestampTest,
			Processed: processedTimestampTest,
		},
	}

	server.ExpectHMGet(
		expectedKeys[0], services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
	).SetVal([]interface{}{requestIDs[0], receivedTimestampTest, processedTimestampTest})

	// 2nd result was not found
	server.ExpectHMGet(
		expectedKeys[1], services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
	).SetVal([]interface{}{nil, nil, nil})

	server.ExpectHMGet(
		expectedKeys[2], services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
	).SetVal([]interface{}{requestIDs[2], receivedTimestampTest, processedTimestampTest})

	// omitMissing == true
	requestStatuses, err := client.GetTimestampsForRequestIDs(
		testdata.OrgID, testdata.ClusterName1, []types.RequestID{
			types.RequestID(requestIDs[0]), types.RequestID(requestIDs[1]), types.RequestID(requestIDs[2]),
		}, true,
	)
	assert.NoError(t, err)
	// 2nd result was ommitted
	assert.Len(t, requestStatuses, 2)
	assert.ElementsMatch(t, requestStatuses, expectedResponse)

	helpers.RedisExpectationsMet(t, server)
}

func TestGetTimestampsForRequestIDs_OKMultipleIncludeMissing(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedResponse := make([]types.RequestStatus, 3)
	expectedKeys := make([]string, 3)
	requestIDs := make([]string, 3)
	for i := range expectedKeys {
		requestIDs[i] = fmt.Sprintf("requestID%d", i)
		expectedKeys[i] = fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName1, requestIDs[i])
		if i%2 == 0 {
			expectedResponse[i] = types.RequestStatus{
				RequestID: requestIDs[i],
				Valid:     true,
				Received:  receivedTimestampTest,
				Processed: processedTimestampTest,
			}
		} else {
			expectedResponse[i] = types.RequestStatus{
				RequestID: requestIDs[i],
				Valid:     false,
				Received:  "",
				Processed: "",
			}
		}
	}

	server.ExpectHMGet(
		expectedKeys[0], services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
	).SetVal([]interface{}{requestIDs[0], receivedTimestampTest, processedTimestampTest})

	server.ExpectHMGet(
		expectedKeys[1], services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
	).SetVal([]interface{}{nil, nil, nil})

	server.ExpectHMGet(
		expectedKeys[2], services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
	).SetVal([]interface{}{requestIDs[2], receivedTimestampTest, processedTimestampTest})

	// omitMissing == false
	requestStatuses, err := client.GetTimestampsForRequestIDs(
		testdata.OrgID, testdata.ClusterName1, []types.RequestID{
			types.RequestID(requestIDs[0]), types.RequestID(requestIDs[1]), types.RequestID(requestIDs[2]),
		}, false,
	)
	assert.NoError(t, err)
	assert.Len(t, requestStatuses, 3)
	assert.ElementsMatch(t, requestStatuses, expectedResponse)

	helpers.RedisExpectationsMet(t, server)
}

func TestGetTimestampsForRequestIDs_Error(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName1, "requestID123")

	server.ExpectHMGet(
		expectedKey, services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
	).SetErr(errTest)

	requestStatuses, err := client.GetTimestampsForRequestIDs(testdata.OrgID, testdata.ClusterName1, []types.RequestID{"requestID123"}, true)
	assert.Error(t, err)
	assert.Len(t, requestStatuses, 0)

	helpers.RedisExpectationsMet(t, server)
}

func TestGetTimestampsForRequestIDs_ScanError(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName1, "requestID123")

	// mismatched number of keys and values
	server.ExpectHMGet(
		expectedKey, services.RequestIDFieldName, services.ReceivedTimestampFieldName, services.ProcessedTimestampFieldName,
	).SetVal([]interface{}{"requestID123", receivedTimestampTest})

	requestStatuses, err := client.GetTimestampsForRequestIDs(testdata.OrgID, testdata.ClusterName1, []types.RequestID{"requestID123"}, true)
	assert.Error(t, err)
	assert.Len(t, requestStatuses, 0)

	helpers.RedisExpectationsMet(t, server)
}

func TestGetRuleHitsForRequest_OKFound(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName1, "requestID123")

	server.ExpectHMGet(
		expectedKey, services.RequestIDFieldName, services.RuleHitsFieldName,
	).SetVal([]interface{}{"requestID123", testRuleHits})

	ruleHits, err := client.GetRuleHitsForRequest(testdata.OrgID, testdata.ClusterName1, "requestID123")
	assert.NoError(t, err)
	assert.Len(t, ruleHits, 2)

	helpers.RedisExpectationsMet(t, server)
}

func TestGetRuleHitsForRequest_OKNotFound(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName1, "requestID123")

	server.ExpectHMGet(
		expectedKey, services.RequestIDFieldName, services.RuleHitsFieldName,
	).SetVal([]interface{}{nil, nil})

	ruleHits, err := client.GetRuleHitsForRequest(testdata.OrgID, testdata.ClusterName1, "requestID123")
	assert.Error(t, err)
	assert.IsType(t, err, &utypes.ItemNotFoundError{})
	assert.Len(t, ruleHits, 0)

	helpers.RedisExpectationsMet(t, server)
}

func TestGetRuleHitsForRequest_FoundInvalidRuleID(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName1, "requestID123")

	ruleHits1Invalid := fmt.Sprintf("%v,%v", data.Rule1CompositeID, data.Rule1ID)
	server.ExpectHMGet(
		expectedKey, services.RequestIDFieldName, services.RuleHitsFieldName,
	).SetVal([]interface{}{"requestID123", ruleHits1Invalid})

	ruleHits, err := client.GetRuleHitsForRequest(testdata.OrgID, testdata.ClusterName1, "requestID123")
	// no error, but only 1 rule hit
	assert.NoError(t, err)
	assert.Len(t, ruleHits, 1)

	helpers.RedisExpectationsMet(t, server)
}

func TestGetRuleHitsForRequest_Error(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName1, "requestID123")

	// mismatched number of keys and values
	server.ExpectHMGet(
		expectedKey, services.RequestIDFieldName, services.RuleHitsFieldName,
	).SetErr(errTest)

	ruleHits, err := client.GetRuleHitsForRequest(testdata.OrgID, testdata.ClusterName1, "requestID123")
	assert.Error(t, err)
	assert.Len(t, ruleHits, 0)

	helpers.RedisExpectationsMet(t, server)
}

func TestGetRuleHitsForRequest_OKScanError(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKey := fmt.Sprintf(services.SimplifiedReportKey, testdata.OrgID, testdata.ClusterName1, "requestID123")

	// mismatched number of keys and values
	server.ExpectHMGet(
		expectedKey, services.RequestIDFieldName, services.RuleHitsFieldName,
	).SetVal([]interface{}{"requestID123"})

	ruleHits, err := client.GetRuleHitsForRequest(testdata.OrgID, testdata.ClusterName1, "requestID123")
	assert.Error(t, err)
	assert.Len(t, ruleHits, 0)

	helpers.RedisExpectationsMet(t, server)
}
