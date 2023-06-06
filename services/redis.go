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
package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/RedHatInsights/insights-operator-utils/redis"
	"github.com/RedHatInsights/insights-results-smart-proxy/types"

	redisV9 "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

const (
	// RequestIDFieldName represents the name of the field in Redis hash containing request_id
	RequestIDFieldName = "request_id"
	// ReceivedTimestampFieldName represents the name of the field in Redis hash containing received timestamp
	ReceivedTimestampFieldName = "received_timestamp"
	// ProcessedTimestampFieldName represents the name of the field in Redis hash containing processed timestamp
	ProcessedTimestampFieldName = "processed_timestamp"
	redisCmdExecutionFailedMsg  = "failed to execute command against Redis server"
)

var (
	// RequestIDsScanPattern is a glob-style pattern to find all matching keys. Uses ?* instead of * to avoid
	// matching "organization:%v:cluster:%v:request:"
	RequestIDsScanPattern = "organization:%v:cluster:%v:request:?*"

	// SimplifiedReportKey is a key under which the information about specific requests is stored
	SimplifiedReportKey = "organization:%v:cluster:%v:request:%v:reports"
)

// RedisInterface represents interface for functions executed against a Redis server
type RedisInterface interface {
	// HealthCheck defined in utils
	HealthCheck() error
	GetRequestIDsForClusterID(
		types.OrgID,
		types.ClusterName,
	) ([]types.RequestID, error)
	GetTimestampsForRequestIDs(
		types.OrgID,
		types.ClusterName,
		[]types.RequestID,
		bool,
	) ([]types.RequestStatus, error)
}

// RedisClient is a local type which embeds the imported redis.Client to include its own functionality
type RedisClient struct {
	redis.Client
}

// NewRedisClient creates a new Redis client based on configuration and returns RedisInterface
func NewRedisClient(conf RedisConfiguration) (RedisInterface, error) {
	client, err := redis.CreateRedisClient(
		conf.RedisEndpoint,
		conf.RedisDatabase,
		conf.RedisPassword,
		conf.RedisTimeoutSeconds,
	)
	if err != nil {
		return nil, err
	}

	return &RedisClient{
		redis.Client{Connection: client},
	}, nil
}

// GetRequestIDsForClusterID retrieves a list of request IDs from Redis.
// "List" of request IDs is in the form of keys with empty values in the following structure:
// organization:{org_id}:cluster:{cluster_id}:request:{request_id1}.
func (redis *RedisClient) GetRequestIDsForClusterID(
	orgID types.OrgID,
	clusterID types.ClusterName,
) (requestIDs []types.RequestID, err error) {
	ctx := context.Background()

	scanKey := fmt.Sprintf(RequestIDsScanPattern, orgID, clusterID)

	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = redis.Client.Connection.Scan(ctx, cursor, scanKey, 0).Result()
		if err != nil {
			log.Error().Err(err).Msgf("failed to execute SCAN command for key '%v' and cursor '%d'", scanKey, cursor)
			return nil, err
		}

		// get last part of key == request_id
		for _, key := range keys {
			keySliced := strings.Split(key, ":")
			requestID := keySliced[len(keySliced)-1]
			requestIDs = append(requestIDs, types.RequestID(requestID))
		}

		if cursor == 0 {
			break
		}
	}
	log.Info().Msgf("retrieved %d request IDs for cluster_id %v: %v", len(requestIDs), clusterID, requestIDs)

	return
}

// GetTimestampsForRequestIDs retrieves the 'received' and 'processed' timestamps of each Request
// for given list of Request IDs. It doesn't retrieve the whole Hash, but only the fields we need.
// It utilizes Redis pipelines in order to avoid multiple client-server round trips.
func (redis *RedisClient) GetTimestampsForRequestIDs(
	orgID types.OrgID,
	clusterID types.ClusterName,
	requestIDs []types.RequestID,
	omitMissing bool,
) (requestStatuses []types.RequestStatus, err error) {
	ctx := context.Background()

	// prepare keys to be used in HMGet commands
	keys := make([]string, len(requestIDs))
	for i, requestID := range requestIDs {
		keys[i] = fmt.Sprintf(SimplifiedReportKey, orgID, clusterID, requestID)
	}

	// queue commands in Redis pipeline. EXEC command is issued upon function exit
	commands, err := redis.Client.Connection.Pipelined(ctx, func(pipe redisV9.Pipeliner) error {
		for _, key := range keys {
			pipe.HMGet(ctx, key, RequestIDFieldName, ReceivedTimestampFieldName, ProcessedTimestampFieldName)
		}
		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg(redisCmdExecutionFailedMsg)
		return requestStatuses, err
	}

	// iterate over results issued in pipeline. Even though we know len(commands),
	// some keys might be missing and we might want to omit them, so we can't initialize slice safely
	for i, cmd := range commands {
		var report types.RequestStatus

		err = cmd.(*redisV9.SliceCmd).Scan(&report)
		if err != nil {
			log.Error().Err(err).Msg(redisCmdExecutionFailedMsg)
			return []types.RequestStatus{}, err
		}

		// omit missing data or invalidate the request
		if report.RequestID == "" {
			if omitMissing {
				continue
			} else {
				// commands in Redis pipeline are guaranteed to be executed in the order they were issued in,
				// therefore we can get the missing request ID from the original slice
				report.RequestID = string(requestIDs[i])
				report.Valid = false
			}
		} else {
			report.Valid = true
		}

		// everything went fine, add to the response
		requestStatuses = append(requestStatuses, report)
	}

	return
}
