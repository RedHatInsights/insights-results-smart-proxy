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

// Package conf contains definition of data type named Config that represents
// configuration of Smart Proxy service. This package also contains function
// named LoadConfiguration that can be used to load configuration from provided
// configuration file and/or from environment variables. Additionally several
// specific functions named GetServerConfiguration, GetServicesConfiguration,
// GetSetupConfiguration, GetMetricsConfiguration, GetLoggingConfiguration and
// GetCloudWatchConfiguration are to be used to return specific configuration
// options.
//
// Generated documentation is available at:
// https://godoc.org/github.com/RedHatInsights/insights-results-smart-proxy/conf
//
// Documentation in literate-programming-style is available at:
// https://redhatinsights.github.io/insights-results-smart-proxy/packages/conf/configuration.html
package conf

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/RedHatInsights/insights-operator-utils/logger"
	"github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	// configFileEnvVariableName is name of environment variable that
	// contains name of configuration file
	configFileEnvVariableName = "INSIGHTS_RESULTS_SMART_PROXY_CONFIG_FILE"

	// envPrefix is prefix for all environment variables that contains
	// various configuration options
	envPrefix = "INSIGHTS_RESULTS_SMART_PROXY_"
)

// SetupConfiguration should only be used at startup
type SetupConfiguration struct {
	InternalRulesOrganizationsCSVFile string `mapstructure:"internal_rules_organizations_csv_file" toml:"internal_rules_organizations_csv_file"`
}

// MetricsConfiguration defines configuration for metrics
type MetricsConfiguration struct {
	Namespace string `mapstructure:"namespace" toml:"namespace"`
}

// Config has exactly the same structure as *.toml file
var Config struct {
	ServerConf     server.Configuration           `mapstructure:"server" toml:"server"`
	ServicesConf   services.Configuration         `mapstructure:"services" toml:"services"`
	SetupConf      SetupConfiguration             `mapstructure:"setup" toml:"setup"`
	MetricsConf    MetricsConfiguration           `mapstructure:"metrics" toml:"metrics"`
	LoggingConf    logger.LoggingConfiguration    `mapstructure:"logging" toml:"logging"`
	CloudWatchConf logger.CloudWatchConfiguration `mapstructure:"cloudwatch" toml:"cloudwatch"`
}

// LoadConfiguration loads configuration from defaultConfigFile, file set in
// configFileEnvVariableName or from env
func LoadConfiguration(defaultConfigFile string) error {
	configFile, specified := os.LookupEnv(configFileEnvVariableName)
	if specified {
		// we need to separate the directory name and filename without
		// extension
		directory, basename := filepath.Split(configFile)
		file := strings.TrimSuffix(basename, filepath.Ext(basename))
		// parse the configuration
		viper.SetConfigName(file)
		viper.AddConfigPath(directory)
	} else {
		// parse the configuration
		viper.SetConfigName(defaultConfigFile)
		viper.AddConfigPath(".")
	}

	err := viper.ReadInConfig()
	if _, isNotFoundError := err.(viper.ConfigFileNotFoundError); !specified && isNotFoundError {
		// viper is not smart enough to understand the structure of
		// config by itself
		fakeTomlConfigWriter := new(bytes.Buffer)

		err := toml.NewEncoder(fakeTomlConfigWriter).Encode(Config)
		if err != nil {
			return err
		}

		fakeTomlConfig := fakeTomlConfigWriter.String()

		viper.SetConfigType("toml")

		err = viper.ReadConfig(strings.NewReader(fakeTomlConfig))
		if err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("fatal error config file: %s", err)
	}

	// override config from env if there's variable in env
	viper.AutomaticEnv()
	viper.SetEnvPrefix(envPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "__"))

	return viper.Unmarshal(&Config)
}

// GetServerConfiguration returns server configuration
func GetServerConfiguration() server.Configuration {
	err := checkIfFileExists(Config.ServerConf.APISpecFile)
	if err != nil {
		log.Fatal().Err(err).Msg("All customer facing APIs MUST serve the current OpenAPI specification")
	}

	Config.ServerConf.InternalRulesOrganizations = getInternalRulesOrganizations()

	return Config.ServerConf
}

// GetServicesConfiguration returns the services endpoints configuration
func GetServicesConfiguration() services.Configuration {
	return Config.ServicesConf
}

// GetSetupConfiguration returns the setup configuration only to be used at
// startup
func GetSetupConfiguration() SetupConfiguration {
	return Config.SetupConf
}

// GetMetricsConfiguration returns the metrics configuration
func GetMetricsConfiguration() MetricsConfiguration {
	return Config.MetricsConf
}

// GetLoggingConfiguration returns logging configuration
func GetLoggingConfiguration() logger.LoggingConfiguration {
	return Config.LoggingConf
}

// GetCloudWatchConfiguration returns cloudwatch configuration
func GetCloudWatchConfiguration() logger.CloudWatchConfiguration {
	return Config.CloudWatchConf
}

// checkIfFileExists returns nil if path doesn't exist or isn't a file,
// otherwise it returns corresponding error
func checkIfFileExists(path string) error {
	if len(path) == 0 {
		return fmt.Errorf("Empty path provided")
	}
	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("OpenAPI spec file path does not exist. Path: '%v'", path)
	} else if err != nil {
		return err
	}

	if fileMode := fileInfo.Mode(); !fileMode.IsRegular() {
		return fmt.Errorf("OpenAPI spec file path is not a file. Path: '%v'", path)
	}

	return nil
}

func getInternalRulesOrganizations() []types.OrgID {
	if !Config.ServerConf.EnableInternalRulesOrganizations {
		log.Info().Msg("Internal rules request filtering disabled")
		return nil
	}

	if len(Config.SetupConf.InternalRulesOrganizationsCSVFile) == 0 {
		log.Fatal().Msgf("Internal organizations enabled, but none supplied")
	}

	internalRulesCSVData, err := ioutil.ReadFile(Config.SetupConf.InternalRulesOrganizationsCSVFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Internal organizations file could not be opened")
	}

	internalOrganizations, err := loadOrgIDsFromCSV(bytes.NewBuffer(internalRulesCSVData))
	if err != nil {
		log.Fatal().Err(err).Msg("Internal organizations CSV could not be processed")
	}

	log.Info().Msgf("Internal rules request filtering enabled. Organizations allowed: %v", internalOrganizations)
	return internalOrganizations
}

// loadOrgIDsFromCSV creates a new CSV reader and returns a list of
// organization IDs
func loadOrgIDsFromCSV(r io.Reader) ([]types.OrgID, error) {
	orgIDs := make([]types.OrgID, 0)

	reader := csv.NewReader(r)

	lines, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV file: %v", err)
	}

	for index, line := range lines {
		if index == 0 {
			continue // skip header
		}

		orgID, err := strconv.ParseUint(line[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf(
				"organization ID on line %v in CSV is not numerical. Found value: %v",
				index+1, line[0],
			)
		}

		orgIDs = append(orgIDs, types.OrgID(orgID))
	}

	return orgIDs, nil
}
