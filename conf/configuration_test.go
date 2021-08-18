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

package conf_test

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/RedHatInsights/insights-results-smart-proxy/conf"
	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

func mustLoadConfiguration(path string) {
	err := conf.LoadConfiguration(path)
	if err != nil {
		panic(err)
	}
}

func removeFile(t *testing.T, filename string) {
	err := os.Remove(filename)
	helpers.FailOnError(t, err)
}

// TestLoadConfiguration loads a configuration file for testing
func TestLoadConfiguration(t *testing.T) {
	os.Clearenv()
	mustLoadConfiguration("tests/config1")
}

// TestLoadConfigurationEnvVariable tests loading the config. file for testing from an environment variable
func TestLoadConfigurationEnvVariable(t *testing.T) {
	os.Clearenv()

	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY_CONFIG_FILE", "../tests/config1")

	mustLoadConfiguration("foobar")
}

// TestLoadingConfigurationFailure tests loading a non-existent configuration file
func TestLoadingConfigurationFailure(t *testing.T) {
	os.Clearenv()

	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY_CONFIG_FILE", "non existing file")

	err := conf.LoadConfiguration("")
	assert.Contains(t, err.Error(), `fatal error config file: Config File "non existing file" Not Found in`)
}

// TestLoadServerConfiguration tests loading the server configuration sub-tree
func TestLoadServerConfiguration(t *testing.T) {
	TestLoadConfiguration(t)
	helpers.FailOnError(t, os.Chdir(".."))

	serverCfg := conf.GetServerConfiguration()

	assert.Equal(t, ":8080", serverCfg.Address)
	assert.Equal(t, "/api/v1/", serverCfg.APIv1Prefix)
	assert.Equal(t, "/api/v2/", serverCfg.APIv2Prefix)
}

func TestLoadConfigurationFromFile(t *testing.T) {
	config := `[server]
		address = ":8080"
		api_v1_prefix = "/api/v1/"
		api_v2_prefix = "/api/v2/"
		api_v1_spec_file = "server/api/v1/openapi.json"
		api_v2_spec_file = "server/api/v2/openapi.json"
		debug = true
		use_https = false
		enable_cors = true
		enable_internal_rules_organizations = false
		internal_rules_organizations = []
	`

	tmpFilename, err := GetTmpConfigFile(config)
	helpers.FailOnError(t, err)

	defer removeFile(t, tmpFilename)

	os.Clearenv()
	mustSetEnv(t, conf.ConfigFileEnvVariableName, tmpFilename)
	mustLoadConfiguration("../tests/config1")

	assert.Equal(t, server.Configuration{
		Address:                          ":8080",
		APIv1Prefix:                      "/api/v1/",
		APIv1SpecFile:                    "server/api/v1/openapi.json",
		APIv2Prefix:                      "/api/v2/",
		APIv2SpecFile:                    "server/api/v2/openapi.json",
		AuthType:                         "xrh",
		Debug:                            true,
		UseHTTPS:                         false,
		EnableCORS:                       true,
		EnableInternalRulesOrganizations: false,
		InternalRulesOrganizations:       []types.OrgID(nil),
	}, conf.GetServerConfiguration())
}

// TestGetInternalRulesOrganizations tests if the internal organizations CSV file gets loaded properly
func TestGetInternalRulesOrganizations(t *testing.T) {
	os.Clearenv()
	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY__SERVER__ENABLE_INTERNAL_RULES_ORGANIZATIONS", "true")

	mustLoadConfiguration("tests/config1")

	assert.Equal(t, []types.OrgID{
		types.OrgID(1),
		types.OrgID(2),
		types.OrgID(3),
	}, conf.GetInternalRulesOrganizations())
}

// TestLoadOrgIDsFromCSVExtraParam tests incorrect CSV format
func TestLoadOrgIDsFromCSVExtraParam(t *testing.T) {
	extraParamCSV := `OrgID
1,2
3
`
	r := strings.NewReader(extraParamCSV)
	_, err := conf.LoadOrgIDsFromCSV(r)
	assert.EqualError(t, err, "error reading CSV file: record on line 2: wrong number of fields")
}

// TestLoadOrgIDsFromCSVNonInt tests non-integer ID in CSV
func TestLoadOrgIDsFromCSVNonInt(t *testing.T) {
	nonIntIDCSV := `OrgID
str
3
`
	r := strings.NewReader(nonIntIDCSV)
	_, err := conf.LoadOrgIDsFromCSV(r)
	assert.EqualError(t, err, "organization ID on line 2 in CSV is not numerical. Found value: str")
}

func GetTmpConfigFile(configData string) (string, error) {
	tmpFile, err := ioutil.TempFile("/tmp", "tmp_config_*.toml")
	if err != nil {
		return "", err
	}

	if _, err := tmpFile.Write([]byte(configData)); err != nil {
		return "", err
	}

	if err := tmpFile.Close(); err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

func mustSetEnv(t *testing.T, key, val string) {
	err := os.Setenv(key, val)
	helpers.FailOnError(t, err)
}

func TestLoadConfigurationFromEnv(t *testing.T) {
	setEnvVariables(t)

	mustLoadConfiguration("/non_existing_path")

	assert.Equal(t, server.Configuration{
		Address:                          ":8080",
		APIv1Prefix:                      "/api/v1/",
		APIv1SpecFile:                    "server/api/v1/openapi.json",
		APIv2Prefix:                      "/api/v2/",
		APIv2SpecFile:                    "server/api/v2/openapi.json",
		AuthType:                         "xrh",
		Debug:                            true,
		UseHTTPS:                         false,
		EnableCORS:                       true,
		EnableInternalRulesOrganizations: false,
		InternalRulesOrganizations:       []types.OrgID(nil),
	}, conf.GetServerConfiguration())

	expectedGroupsPollTime, _ := time.ParseDuration("60s")
	assert.Equal(t, services.Configuration{
		AggregatorBaseEndpoint: "http://localhost:8080/api/v1",
		ContentBaseEndpoint:    "http://localhost:8081/api/v1",
		GroupsPollingTime:      expectedGroupsPollTime,
	}, conf.GetServicesConfiguration())
}

// TestLoadConfigurationFromEnvVariableClowderEnabledNotSupported tests loading.
// the config file for testing from an environment variable. Clowder config is
// available but clowder is not supported in this environment.
func TestLoadConfigurationFromEnvVariableClowderEnabledNotSupported(t *testing.T) {
	os.Clearenv()

	mustSetEnv(t, "CCX_NOTIFICATION_SERVICE_CONFIG_FILE", "tests/config1")
	mustSetEnv(t, "ACG_CONFIG", "tests/clowder_config.json")
	mustLoadConfiguration("CCX_NOTIFICATION_SERVICE_CONFIG_FILE")
}

// TestLoadConfigurationFromEnvVariableClowderEnabledNotSupported tests loading.
// the config file for testing from an environment variable. Clowder config is
// available and clowder is supported in this environment.
func TestLoadConfigurationFromEnvVariableClowderEnabledAndSupported(t *testing.T) {
	os.Clearenv()
	mustSetEnv(t, "CCX_NOTIFICATION_SERVICE_CONFIG_FILE", "tests/config1")
	mustSetEnv(t, "ACG_CONFIG", "tests/clowder_config.json")
	mustSetEnv(t, "CLOWDER_ENABLED", "true")
	mustLoadConfiguration("CCX_NOTIFICATION_SERVICE_CONFIG_FILE")
}

func setEnvVariables(t *testing.T) {
	os.Clearenv()

	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY__SERVER__ADDRESS", ":8080")
	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY__SERVER__API_V1_PREFIX", "/api/v1/")
	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY__SERVER__API_V1_SPEC_FILE", "server/api/v1/openapi.json")
	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY__SERVER__API_V2_PREFIX", "/api/v2/")
	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY__SERVER__API_V2_SPEC_FILE", "server/api/v2/openapi.json")
	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY__SERVER__DEBUG", "true")
	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY__SERVER__ENABLE_INTERNAL_RULES_ORGANIZATIONS", "false")

	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY__SERVICES__AGGREGATOR", "http://localhost:8080/api/v1")
	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY__SERVICES__CONTENT", "http://localhost:8081/api/v1")

	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY__SERVICES__GROUPS_POLL_TIME", "60s")
}
