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

package server

import (
	types "github.com/RedHatInsights/insights-results-types"
)

// Configuration represents configuration of REST API HTTP server
type Configuration struct {
	Address                          string        `mapstructure:"address" toml:"address"`
	APIdbgPrefix                     string        `mapstructure:"api_dbg_prefix" toml:"api_dbg_prefix"`
	APIv1Prefix                      string        `mapstructure:"api_v1_prefix" toml:"api_v1_prefix"`
	APIv2Prefix                      string        `mapstructure:"api_v2_prefix" toml:"api_v2_prefix"`
	APIv1SpecFile                    string        `mapstructure:"api_v1_spec_file" toml:"api_v1_spec_file"`
	APIv2SpecFile                    string        `mapstructure:"api_v2_spec_file" toml:"api_v2_spec_file"`
	Debug                            bool          `mapstructure:"debug" toml:"debug"`
	Auth                             bool          `mapstructure:"auth" toml:"auth"`
	AuthType                         string        `mapstructure:"auth_type" toml:"auth_type"`
	UseHTTPS                         bool          `mapstructure:"use_https" toml:"use_https"`
	EnableCORS                       bool          `mapstructure:"enable_cors" toml:"enable_cors"`
	EnableInternalRulesOrganizations bool          `mapstructure:"enable_internal_rules_organizations" toml:"enable_internal_rules_organizations"`
	InternalRulesOrganizations       []types.OrgID `mapstructure:"internal_rules_organizations" toml:"internal_rules_organizations"`
	LogAuthToken                     bool          `mapstructure:"log_auth_token" toml:"log_auth_token"`
	UseOrgClustersFallback           bool          `mapstructure:"org_clusters_fallback" toml:"org_clusters_fallback"`
	UseRBAC                          bool          `mapstructure:"use_rbac" toml:"use_rbac"`
}
