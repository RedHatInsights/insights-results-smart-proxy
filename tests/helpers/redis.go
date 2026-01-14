// Copyright 2021, 2022, 2023 Red Hat, Inc
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

package helpers

import (
	"testing"

	redisutils "github.com/RedHatInsights/insights-operator-utils/redis"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"

	"github.com/go-redis/redismock/v9"
)

var (
	// DefaultRedisConf is the default Redis configuration used in tests
	DefaultRedisConf services.RedisConfiguration
)

// set default configuration
func init() {
	DefaultRedisConf = services.RedisConfiguration{
		RedisEndpoint:       "localhost:6379",
		RedisDatabase:       0,
		RedisPassword:       "psw",
		RedisTimeoutSeconds: 30,
	}
}

// GetMockRedis is used to get a mocked Redis client to expect and respond to queries
func GetMockRedis() (
	mockClient services.RedisClient, mockServer redismock.ClientMock,
) {
	client, mockServer := redismock.NewClientMock()
	mockClient = services.RedisClient{
		Client: redisutils.Client{Connection: client},
	}
	return
}

// RedisExpectationsMet helper function used to ensure mock expectations were met
func RedisExpectationsMet(t *testing.T, mock redismock.ClientMock) {
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
