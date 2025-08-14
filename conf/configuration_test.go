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
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	clowder "github.com/redhatinsights/app-common-go/pkg/api/v1"

	"github.com/RedHatInsights/insights-operator-utils/logger"
	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	types "github.com/RedHatInsights/insights-results-types"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-smart-proxy/conf"
	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"
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
func TestLoadConfiguration(_ *testing.T) {
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
	assert.Equal(t, "/api/dbg/", serverCfg.APIdbgPrefix)
	assert.Equal(t, "/api/v1/", serverCfg.APIv1Prefix)
	assert.Equal(t, "/api/v2/", serverCfg.APIv2Prefix)
}

func TestLoadConfigurationFromFile(t *testing.T) {
	config := `[server]
		address = ":8080"
		api_dbg_prefix = "/api/dbg/"
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
		APIdbgPrefix:                     "/api/dbg/",
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
	tmpFile, err := os.CreateTemp("/tmp", "tmp_config_*.toml")
	if err != nil {
		return "", err
	}

	if _, err := tmpFile.WriteString(configData); err != nil {
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
		APIdbgPrefix:                     "/api/dbg/",
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
	// explicit Clowder config
	clowder.LoadedConfig = &clowder.AppConfig{}

	os.Clearenv()

	mustSetEnv(t, "CCX_NOTIFICATION_SERVICE_CONFIG_FILE", "tests/config1")
	mustSetEnv(t, "ACG_CONFIG", "tests/clowder_config.json")
	mustLoadConfiguration("CCX_NOTIFICATION_SERVICE_CONFIG_FILE")
}

// TestLoadConfigurationFromEnvVariableClowderEnabledNotSupported tests loading.
// the config file for testing from an environment variable. Clowder config is
// available and clowder is supported in this environment.
func TestLoadConfigurationFromEnvVariableClowderEnabledAndSupported(t *testing.T) {
	// explicit Clowder config
	clowder.LoadedConfig = &clowder.AppConfig{}

	os.Clearenv()

	mustSetEnv(t, "CCX_NOTIFICATION_SERVICE_CONFIG_FILE", "tests/config1")
	mustSetEnv(t, "ACG_CONFIG", "tests/clowder_config.json")
	mustSetEnv(t, "CLOWDER_ENABLED", "true")
	mustLoadConfiguration("CCX_NOTIFICATION_SERVICE_CONFIG_FILE")
}

func setEnvVariables(t *testing.T) {
	os.Clearenv()

	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY__SERVER__ADDRESS", ":8080")
	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY__SERVER__API_DBG_PREFIX", "/api/dbg/")
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

// TestGetAMCClientConfiguration tests loading the AMS configuration sub-tree
func TestGetAMCClientConfiguration(t *testing.T) {
	/* Load following configuration:

	   [amsclient]
	   url = "https://api.openshift.com"
	   client_id = "-client-id-"
	   client_secret = "-top-secret-"
	   page_size = 6000
	   cluster_list_caching = false
	*/

	TestLoadConfiguration(t)
	helpers.FailOnError(t, os.Chdir(".."))

	// call the tested function
	amsConfiguration := conf.GetAMSClientConfiguration()

	// check returned structure
	assert.Equal(t, "", amsConfiguration.Token)
	assert.Equal(t, "-client-id-", amsConfiguration.ClientID)
	assert.Equal(t, "-top-secret-", amsConfiguration.ClientSecret)
	assert.Equal(t, "https://api.openshift.com", amsConfiguration.URL)
	assert.Equal(t, 6000, amsConfiguration.PageSize)
	assert.Equal(t, false, amsConfiguration.ClusterListCaching)
}

// TestGetSetupConfiguration tests loading the Setup configuration sub-tree
func TestGetSetupConfiguration(t *testing.T) {
	/* Load following configuration:

	[setup]
	internal_rules_organizations_csv_file = "tests/internal_organizations_test.csv"

	*/

	TestLoadConfiguration(t)
	helpers.FailOnError(t, os.Chdir(".."))

	// call the tested function
	setupConfiguration := conf.GetSetupConfiguration()

	// check returned structure
	assert.Equal(t, "tests/internal_organizations_test.csv", setupConfiguration.InternalRulesOrganizationsCSVFile)
}

// TestGetRedisConfiguration tests loading the Redis configuration sub-tree
func TestGetRedisConfiguration(t *testing.T) {
	/* Load following configuration:

	[redis]
	database = 42
	endpoint = "localhost:6379"
	password = "-redis-password-"
	timeout_seconds = 30

	*/

	TestLoadConfiguration(t)
	helpers.FailOnError(t, os.Chdir(".."))

	// call the tested function
	redisConfiguration := conf.GetRedisConfiguration()

	// check returned structure
	assert.Equal(t, 42, redisConfiguration.RedisDatabase)
	assert.Equal(t, "localhost:6379", redisConfiguration.RedisEndpoint)
	assert.Equal(t, "-redis-password-", redisConfiguration.RedisPassword)
	assert.Equal(t, 30, redisConfiguration.RedisTimeoutSeconds)
}

// TestGetMetricsConfiguration tests loading the metrics configuration sub-tree
func TestGetMetricsConfiguration(t *testing.T) {
	/* Load following configuration:

	[metrics]
	namespace = "smart_proxy"

	*/

	TestLoadConfiguration(t)
	helpers.FailOnError(t, os.Chdir(".."))

	// call the tested function
	metricsConfiguration := conf.GetMetricsConfiguration()

	// check returned structure
	assert.Equal(t, "smart_proxy", metricsConfiguration.Namespace)
}

// TestGetSentryLoggingConfiguration tests loading the sentry logging configuration sub-tree
func TestGetSentryLoggingConfiguration(t *testing.T) {
	/* Load following configuration:

	[sentry]
	dsn = "test_dsn"
	environment = "test_env"

	*/

	TestLoadConfiguration(t)
	helpers.FailOnError(t, os.Chdir(".."))

	// call the tested function
	sentryConfiguration := conf.GetSentryLoggingConfiguration()

	// check returned structure
	assert.Equal(t, "test_dsn", sentryConfiguration.SentryDSN)
	assert.Equal(t, "test_env", sentryConfiguration.SentryEnvironment)
}

// TestGetCloudWatchConfiguration tests loading the cloud watch configuration sub-tree
func TestGetCloudWatchConfiguration(t *testing.T) {
	TestLoadConfiguration(t)
	helpers.FailOnError(t, os.Chdir(".."))

	// call the tested function
	cloudWatchConfiguration := conf.GetCloudWatchConfiguration()

	// check returned structure
	assert.Equal(t, logger.CloudWatchConfiguration{
		AWSAccessID:     "",
		AWSSecretKey:    "",
		AWSSessionToken: "",
		AWSRegion:       "",
		LogGroup:        "",
		StreamName:      "",
		Debug:           false,
	}, cloudWatchConfiguration)
}

// TestGetLoggingConfiguration tests loading the logging configuration sub-tree
func TestGetLoggingConfiguration(t *testing.T) {
	/* Load following configuration:

	[logging]
	debug = true

	*/

	TestLoadConfiguration(t)
	helpers.FailOnError(t, os.Chdir(".."))

	// call the tested function
	loggingConfiguration := conf.GetLoggingConfiguration()

	// check returned structure
	assert.Equal(t, logger.LoggingConfiguration{
		Debug:                      true,
		LogLevel:                   "",
		LoggingToCloudWatchEnabled: false,
	}, loggingConfiguration)
}

// TestLoadRBACConfiguration tests loading the RBAC configuration sub-tree
func TestLoadRBACConfiguration(t *testing.T) {
	TestLoadConfiguration(t)
	helpers.FailOnError(t, os.Chdir(".."))

	cfg := conf.GetRBACConfiguration()

	assert.Equal(t, "https://api.openshift.com", cfg.URL)
	assert.Equal(t, false, cfg.EnforceAuth)
}

// TestClowderConfigForRedis tests loading the config file for testing from an
// environment variable. Clowder config is enabled in this case, checking the Redis
// configuration.
func TestClowderConfigForRedis(t *testing.T) {
	os.Clearenv()

	var hostname = "redis"
	var port = 6379
	var username = "user"
	var password = "password"

	// explicit Redis config
	clowder.LoadedConfig = &clowder.AppConfig{
		InMemoryDb: &clowder.InMemoryDBConfig{
			Hostname: hostname,
			Port:     port,
			Username: &username,
			Password: &password,
		},
	}

	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY_CONFIG_FILE", "../tests/config1")
	mustSetEnv(t, "ACG_CONFIG", "tests/clowder_config.json")

	err := conf.LoadConfiguration("config")
	assert.NoError(t, err, "Failed loading configuration file")

	redisCfg := conf.GetRedisConfiguration()
	assert.Equal(t, fmt.Sprintf("%s:%d", hostname, port), redisCfg.RedisEndpoint)
	assert.Equal(t, username, redisCfg.RedisUsername)
	assert.Equal(t, password, redisCfg.RedisPassword)
}
