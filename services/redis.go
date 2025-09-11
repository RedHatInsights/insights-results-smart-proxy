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
	"regexp"
	"strings"

	"github.com/RedHatInsights/insights-operator-utils/redis"
	utypes "github.com/RedHatInsights/insights-operator-utils/types"
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
	// RuleHitsFieldName represent the name of hte field in Redis hash containing simplified rule hits
	RuleHitsFieldName = "rule_hits"
	// ScanBatchCount is the number of records to go through in a single SCAN operation
	ScanBatchCount = 100

	redisCmdExecutionFailedMsg = "failed to execute command against Redis server"
)

var (
	// RequestIDsScanPattern is a glob-style pattern to find all matching keys. Uses ?* instead of * to avoid
	// matching "organization:%v:cluster:%v:request:".
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
	GetRuleHitsForRequest(
		types.OrgID,
		types.ClusterName,
		types.RequestID,
	) ([]types.RuleID, error)
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
		conf.RedisUsername,
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
func (redisClient *RedisClient) GetRequestIDsForClusterID(
	orgID types.OrgID,
	clusterID types.ClusterName,
) (requestIDs []types.RequestID, err error) {
	ctx := context.Background()

	scanKey := fmt.Sprintf(RequestIDsScanPattern, orgID, clusterID)
	log.Debug().Str("Scan key", scanKey).Msg("Key to retrieve request IDs from Redis")

	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = redisClient.Client.Connection.Scan(ctx, cursor, scanKey, ScanBatchCount).Result()
		if err != nil {
			log.Error().Err(err).
				Str("scanKey", scanKey).Uint64("cursor", cursor).
				Msg("failed to execute SCAN command for key and cursor")
			return nil, err
		}

		// get last part of key == request_id
		for _, key := range keys {
			// exclude simplified report keys that are ending with ":reports" suffix
			if strings.HasSuffix(key, ":reports") {
				continue
			}
			keySliced := strings.Split(key, ":")
			requestID := keySliced[len(keySliced)-1]
			requestIDs = append(requestIDs, types.RequestID(requestID))
		}

		if cursor == 0 {
			break
		}
	}
	log.Debug().Msgf("retrieved %d request IDs for cluster_id %v: %v", len(requestIDs), clusterID, requestIDs)

	return
}

// GetTimestampsForRequestIDs retrieves the 'received' and 'processed' timestamps of each Request
// for given list of Request IDs. It doesn't retrieve the whole Hash, but only the fields we need.
// It utilizes Redis pipelines in order to avoid multiple client-server round trips.
func (redisClient *RedisClient) GetTimestampsForRequestIDs(
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
	commands, err := redisClient.Connection.Pipelined(ctx, func(pipe redisV9.Pipeliner) error {
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
			}

			// commands in Redis pipeline are guaranteed to be executed in the order they were issued in,
			// therefore we can get the missing request ID from the original slice
			report.RequestID = string(requestIDs[i])
			report.Valid = false
		} else {
			report.Valid = true
		}

		// everything went fine, add to the response
		requestStatuses = append(requestStatuses, report)
	}

	return
}

// GetRuleHitsForRequest is used to get the rule_hits field from Hash type
// stored in Redis.
func (redisClient *RedisClient) GetRuleHitsForRequest(
	orgID types.OrgID,
	clusterID types.ClusterName,
	requestID types.RequestID,
) (ruleHits []types.RuleID, err error) {
	var simplifiedReport types.SimplifiedReport

	ctx := context.Background()
	key := fmt.Sprintf(SimplifiedReportKey, orgID, clusterID, requestID)

	cmd := redisClient.Connection.HMGet(ctx, key, RequestIDFieldName, RuleHitsFieldName)
	if err = cmd.Err(); err != nil {
		log.Error().Err(err).Msg(redisCmdExecutionFailedMsg)
		return
	}

	err = cmd.Scan(&simplifiedReport)
	if err != nil {
		log.Error().Err(err).Msg("failed to scan result map into a struct")
		return
	}

	// report not found in storage
	if simplifiedReport.RequestID == "" {
		err = &utypes.ItemNotFoundError{ItemID: requestID}
		log.Warn().Err(err).Msgf("request data for request_id %v not found in Redis", requestID)
		return
	}

	log.Debug().Msgf("rule hits CSV retrieved from Redis: %v", simplifiedReport.RuleHitsCSV)

	// validate rule IDs coming from Redis
	ruleHitsSplit := strings.Split(simplifiedReport.RuleHitsCSV, ",")
	for _, ruleHit := range ruleHitsSplit {
		ruleIDRegex := regexp.MustCompile(`^([a-zA-Z_0-9.]+)[|]([a-zA-Z_0-9.]+)$`)

		isRuleIDValid := ruleIDRegex.MatchString(ruleHit)
		if ruleHit == "" {
			log.Debug().Str("RequestID", simplifiedReport.RequestID).Msg("There are no rule hits for given request id")
			continue
		}
		if !isRuleIDValid {
			log.Error().Str("rule_id", ruleHit).Msg("rule_id retrieved from Redis is in invalid format")
			continue
		}

		ruleHits = append(ruleHits, types.RuleID(ruleHit))
	}

	return
}
