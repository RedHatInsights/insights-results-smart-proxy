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
	"github.com/rs/zerolog/log"
)

var (
	// RequestIDsScanPattern is a glob-style pattern to find all matching keys. Uses ?* instead of * to avoid
	// matching "organization:%v:cluster:%v:request:"
	RequestIDsScanPattern = "organization:%v:cluster:%v:request:?*"
)

// RedisInterface represents interface for functions executed against a Redis server
type RedisInterface interface {
	// HealthCheck defined in utils
	HealthCheck() error
	GetRequestIDsForClusterID(
		types.OrgID,
		types.ClusterName,
	) ([]types.RequestID, error)
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
