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

package services

import (
	"time"
)

// RedisConfiguration represents configuration of Redis client
type RedisConfiguration struct {
	RedisEndpoint       string `mapstructure:"endpoint" toml:"endpoint"`
	RedisDatabase       int    `mapstructure:"database" toml:"database"`
	RedisTimeoutSeconds int    `mapstructure:"timeout_seconds" toml:"timeout_seconds"`
	RedisPassword       string `mapstructure:"password" toml:"password"`
	RedisUsername       string `mapstructure:"username" toml:"username"`
}

// Configuration represents configuration of services on which smart-proxy depends.
type Configuration struct {
	AggregatorBaseEndpoint         string        `mapstructure:"aggregator" toml:"aggregator"`
	ContentBaseEndpoint            string        `mapstructure:"content" toml:"content"`
	UpgradeRisksPredictionEndpoint string        `mapstructure:"upgrade_risks_prediction" toml:"upgrade_risks_prediction"`
	GroupsPollingTime              time.Duration `mapstructure:"groups_poll_time" toml:"groups_poll_time"`
	ContentDirectoryTimeout        time.Duration `mapstructure:"content_directory_timeout" toml:"content_directory_timeout"`
}
