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
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
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
	assert.Equal(t, "/api/v1/", serverCfg.APIPrefix)
}

func TestLoadConfigurationFromFile(t *testing.T) {
	config := `[server]
		address = ":8080"
		api_prefix = "/api/v1/"
		api_spec_file = "openapi.json"
		debug = true
		use_https = false
		enable_cors = true
	`

	tmpFilename, err := GetTmpConfigFile(config)
	helpers.FailOnError(t, err)

	defer removeFile(t, tmpFilename)

	os.Clearenv()
	mustSetEnv(t, conf.ConfigFileEnvVariableName, tmpFilename)
	mustLoadConfiguration("../tests/config1")

	assert.Equal(t, server.Configuration{
		Address:     ":8080",
		APIPrefix:   "/api/v1/",
		APISpecFile: "openapi.json",
		AuthType:    "xrh",
		Debug:       true,
		UseHTTPS:    false,
		EnableCORS:  true,
	}, conf.GetServerConfiguration())
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
		Address:     ":8080",
		APIPrefix:   "/api/v1/",
		APISpecFile: "openapi.json",
		AuthType:    "xrh",
		Debug:       true,
		UseHTTPS:    false,
		EnableCORS:  true,
	}, conf.GetServerConfiguration())

	assert.Equal(t, services.Configuration{
		AggregatorBaseEndpoint: "http://localhost:8080/api/v1",
		ContentBaseEndpoint:    "http://localhost:8081/api/v1",
	}, conf.GetServicesConfiguration())
}

func setEnvVariables(t *testing.T) {
	os.Clearenv()

	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY__SERVER__ADDRESS", ":8080")
	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY__SERVER__API_PREFIX", "/api/v1/")
	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY__SERVER__API_SPEC_FILE", "openapi.json")
	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY__SERVER__DEBUG", "true")

	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY__SERVICES__AGGREGATOR", "http://localhost:8080/api/v1")
	mustSetEnv(t, "INSIGHTS_RESULTS_SMART_PROXY__SERVICES__CONTENT", "http://localhost:8081/api/v1")
}
