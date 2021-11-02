/*
Copyright © 2020, 2021 Red Hat, Inc.

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

// Package main_test contains declaration of unit tests for the main package of
// Insights Results Smart Proxy service
package main_test

import (
	"os"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/bmizerany/assert"
	testify "github.com/stretchr/testify/assert"

	main "github.com/RedHatInsights/insights-results-smart-proxy"
	"github.com/RedHatInsights/insights-results-smart-proxy/conf"
)

const (
	testsTimeout = 60 * time.Second
)

func mustSetEnv(t *testing.T, key, val string) {
	err := os.Setenv(key, val)
	helpers.FailOnError(t, err)
}

func mustLoadConfiguration(path string) {
	err := conf.LoadConfiguration(path)
	if err != nil {
		panic(err)
	}
}

func setEnvSettings(t *testing.T, settings map[string]string) {
	os.Clearenv()

	for key, val := range settings {
		mustSetEnv(t, key, val)
	}

	mustLoadConfiguration("/non_existing_path")
}

func TestStartServer_BadServerAddress(t *testing.T) {
	setEnvSettings(t, map[string]string{
		"INSIGHTS_RESULTS_SMART_PROXY__SERVER__ADDRESS":            "non-existing-host:1",
		"INSIGHTS_RESULTS_SMART_PROXY__SERVER__API_V1_SPEC_FILE":   "server/api/v1/openapi.json",
		"INSIGHTS_RESULTS_SMART_PROXY__SERVER__API_V2_SPEC_FILE":   "server/api/v2/openapi.json",
		"INSIGHTS_RESULTS_SMART_PROXY__SERVICES__GROUPS_POLL_TIME": "60s",
	})

	_ = main.StartServer()
	// assert.Equal(t, main.ExitStatusServerError, errCode)
}

// TestPrintVersionInfo is dummy ATM - we'll check versions etc. in integration tests.
// TODO: add check for actual messages that are printed to standard output
func TestPrintVersionInfo(t *testing.T) {
	main.PrintVersionInfo()
}

// TestPrintHelp checks that printing help returns OK exit code.
// TODO: add check for actual messages that are printed to standard output
func TestPrintHelp(t *testing.T) {
	assert.Equal(t, main.ExitStatusOK, int(main.PrintHelp()))
}

// TestPrintConfig checks that printing configuration info returns OK exit code.
// TODO: add check for actual messages that are printed to standard output
func TestPrintConfig(t *testing.T) {
	assert.Equal(t, main.ExitStatusOK, int(main.PrintConfig()))
}

// TestPrintEnv checks that printing environment variables returns OK exit code.
// TODO: add check for actual messages that are printed to standard output
func TestPrintEnv(t *testing.T) {
	assert.Equal(t, main.ExitStatusOK, int(main.PrintEnv()))
}

// TestFillInInfoParams test the behaviour of function fillInInfoParams
func TestFillInInfoParams(t *testing.T) {
	// map to be used by this unit test
	m := make(map[string]string)

	// preliminary test if Go Universe is still ok
	testify.Empty(t, m, "Map should be empty at the beginning")

	// try to fill-in all info params
	main.FillInInfoParams(m)

	// preliminary test if Go Universe is still ok
	testify.Len(t, m, 5, "Map should contains exactly five items")

	// does the map contain all expected keys?
	testify.Contains(t, m, "BuildVersion")
	testify.Contains(t, m, "BuildTime")
	testify.Contains(t, m, "BuildBranch")
	testify.Contains(t, m, "BuildCommit")
	testify.Contains(t, m, "UtilsVersion")
}
