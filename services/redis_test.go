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

	"github.com/RedHatInsights/insights-results-smart-proxy/services"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/testdata"
	"github.com/RedHatInsights/insights-results-smart-proxy/types"
	"github.com/stretchr/testify/assert"
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

func TestRedisGetRequestIDsForClusterIDEmpty(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName1)
	server.ExpectScan(0, expectedKey, 0).SetVal([]string{}, 0)

	requestIDs, err := client.GetRequestIDsForClusterID(testdata.OrgID, testdata.ClusterName1)
	assert.NoError(t, err)
	assert.Len(t, requestIDs, 0)

	helpers.RedisExpectationsMet(t, server)
}

func TestRedisGetRequestIDsForClusterIDResultsSinglePage(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName1)

	expectedResponseKeys := make([]string, 2)
	for i := range expectedResponseKeys {
		expectedResponseKeys[i] = fmt.Sprintf("organization:%v:cluster:%v:request:requestID%v", testdata.OrgID, testdata.ClusterName1, i)
	}
	// all results are in a single page -- cursor == 0, so no more calls are expected
	server.ExpectScan(0, expectedKey, 0).SetVal(expectedResponseKeys, 0)

	requestIDs, err := client.GetRequestIDsForClusterID(testdata.OrgID, testdata.ClusterName1)
	assert.NoError(t, err)
	assert.Len(t, requestIDs, 2)
	assert.ElementsMatch(t, requestIDs, []types.RequestID{"requestID0", "requestID1"})

	helpers.RedisExpectationsMet(t, server)
}

func TestRedisGetRequestIDsForClusterIDResultsMultiplePages(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedResponseKeys := make([]string, 4)
	for i := range expectedResponseKeys {
		expectedResponseKeys[i] = fmt.Sprintf("organization:%v:cluster:%v:request:requestID%v", testdata.OrgID, testdata.ClusterName1, i)
	}

	expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName1)
	server.ExpectScan(0, expectedKey, 0).SetVal([]string{expectedResponseKeys[0], expectedResponseKeys[1]}, 42)
	// returned cursor is expected to be used in the next call
	server.ExpectScan(42, expectedKey, 0).SetVal([]string{expectedResponseKeys[2]}, 8)
	// returned cursor is expected to be used in the next call
	server.ExpectScan(8, expectedKey, 0).SetVal([]string{expectedResponseKeys[3]}, 0)
	// returned cursor == 0, so no more calls are expected

	requestIDs, err := client.GetRequestIDsForClusterID(testdata.OrgID, testdata.ClusterName1)
	assert.NoError(t, err)
	assert.Len(t, requestIDs, len(expectedResponseKeys))
	assert.ElementsMatch(t, requestIDs, []types.RequestID{"requestID0", "requestID1", "requestID2", "requestID3"})

	helpers.RedisExpectationsMet(t, server)
}

func TestRedisGetRequestIDsForClusterIDError(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName1)
	server.ExpectScan(0, expectedKey, 0).SetErr(errors.New("ka-boom"))

	requestIDs, err := client.GetRequestIDsForClusterID(testdata.OrgID, testdata.ClusterName1)
	assert.Error(t, err)
	assert.Len(t, requestIDs, 0)

	helpers.RedisExpectationsMet(t, server)
}

func TestRedisGetRequestIDsForClusterIDErrorInFollowingCalls(t *testing.T) {
	client, server := helpers.GetMockRedis()

	expectedKey := fmt.Sprintf(services.RequestIDsScanPattern, testdata.OrgID, testdata.ClusterName1)

	expectedResponseKeys := make([]string, 2)
	for i := range expectedResponseKeys {
		expectedResponseKeys[i] = fmt.Sprintf("organization:%v:cluster:%v:request:requestID%v", testdata.OrgID, testdata.ClusterName1, i)
	}

	server.ExpectScan(0, expectedKey, 0).SetVal([]string{expectedResponseKeys[0], expectedResponseKeys[1]}, 42)
	server.ExpectScan(42, expectedKey, 0).SetErr(errors.New("ka-boom"))

	// function should return empty list + error if we can't retrieve the whole data set
	requestIDs, err := client.GetRequestIDsForClusterID(testdata.OrgID, testdata.ClusterName1)
	assert.Error(t, err)
	assert.Len(t, requestIDs, 0)

	helpers.RedisExpectationsMet(t, server)
}
