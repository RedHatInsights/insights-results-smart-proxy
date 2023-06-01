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
	"context"
	"errors"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-results-smart-proxy/services"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
)

var (
	ctx              context.Context
	defaultRedisConf services.RedisConfiguration
)

// set default configuration
func init() {
	ctx = context.Background()

	defaultRedisConf = services.RedisConfiguration{
		RedisEndpoint:       "localhost:6379",
		RedisDatabase:       0,
		RedisPassword:       "psw",
		RedisTimeoutSeconds: 30,
	}
}

func getMockRedis() (
	mockClient services.RedisClient, mockServer redismock.ClientMock,
) {
	client, mockServer := redismock.NewClientMock()
	mockClient = services.RedisClient{
		Client: client,
	}
	return
}

func redisExpectationsMet(t *testing.T, mock redismock.ClientMock) {
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestNewRedisClient(t *testing.T) {
	client, err := services.NewRedisClient(defaultRedisConf)
	assert.NotNil(t, client)
	assert.NoError(t, err)
}

func TestNewRedisClientFail(t *testing.T) {
	conf := defaultRedisConf
	conf.RedisDatabase = -1

	client, err := services.NewRedisClient(conf)
	assert.Nil(t, client)
	assert.Error(t, err)
}

func TestCreateRedisClientOK(t *testing.T) {
	client, err := services.CreateRedisClient(defaultRedisConf)
	assert.NoError(t, err)

	options := client.Options()
	assert.NoError(t, err)
	assert.Equal(t, options.Addr, defaultRedisConf.RedisEndpoint)
	assert.Equal(t, options.DB, defaultRedisConf.RedisDatabase)
	assert.Equal(t, options.Password, defaultRedisConf.RedisPassword)
	assert.Equal(t, options.ReadTimeout, time.Duration(defaultRedisConf.RedisTimeoutSeconds)*time.Second)
}

func TestCreateRedisClientBadAddress(t *testing.T) {
	conf := defaultRedisConf
	conf.RedisEndpoint = ""
	client, err := services.CreateRedisClient(conf)
	assert.Nil(t, client)
	assert.Error(t, err)
}

func TestCreateRedisClientDBIndexOutOfRange(t *testing.T) {
	conf := defaultRedisConf
	// Redis supports "only" 16 different databases with indices 0-15
	conf.RedisDatabase = 16
	client, err := services.CreateRedisClient(conf)
	assert.Nil(t, client)
	assert.Error(t, err)
}

func TestRedisHealthCheckOK(t *testing.T) {
	client, server := getMockRedis()

	server.ExpectPing().SetVal("PONG")

	err := client.HealthCheck()
	assert.NoError(t, err)

	redisExpectationsMet(t, server)
}

func TestRedisHealthCheckError(t *testing.T) {
	client, server := getMockRedis()

	server.ExpectPing().SetErr(errors.New("mock error"))

	err := client.HealthCheck()
	assert.Error(t, err)

	redisExpectationsMet(t, server)
}

func TestRedisHealthCheckBadResponse(t *testing.T) {
	client, server := getMockRedis()

	// cover 2nd condition
	server.ExpectPing().SetVal("ka-boom")

	err := client.HealthCheck()
	assert.Error(t, err)

	redisExpectationsMet(t, server)
}
