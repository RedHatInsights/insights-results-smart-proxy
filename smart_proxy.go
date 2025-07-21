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

// Entry point to the insights results smart proxy REST API service.
// This file contains functions needed to start the service from command line.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/RedHatInsights/insights-content-service/groups"
	"github.com/RedHatInsights/insights-operator-utils/logger"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-smart-proxy/amsclient"
	"github.com/RedHatInsights/insights-results-smart-proxy/auth"
	"github.com/RedHatInsights/insights-results-smart-proxy/conf"
	"github.com/RedHatInsights/insights-results-smart-proxy/metrics"
	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"

	proxy_content "github.com/RedHatInsights/insights-results-smart-proxy/content"
)

// ExitCode represents numeric value returned to parent process when the
// current process finishes
type ExitCode int

const (
	// ExitStatusOK means that the service have finished with success
	ExitStatusOK = iota
	// ExitStatusServerError means that the HTTP server cannot be initialized
	ExitStatusServerError
	defaultConfigFileName = "config"
	//Constants for commands
	commandStartService = "start-service"
	commandPrintVersion = "print-version"
	commandPrintHelp    = "print-help"
	commandPrintConfig  = "print-config"
	commandPrintEnv     = "print-env"
)

const helpMessageTemplate = `
Smart Proxy service for insights results

Usage:

    %+v [command]

The commands are:

    <EMPTY>             starts smart-proxy
    start-service       starts smart-proxy
    help                prints help
    print-help          prints help
    print-config        prints current configuration set by files & env variables
    print-env           prints env variables
    print-version-info  prints version info

`

// serverInstance represents instance of REST API server
var serverInstance *server.HTTPServer

// printHelp function displays help on the standard output.
func printHelp() ExitCode {
	fmt.Printf(helpMessageTemplate, os.Args[0])
	return ExitStatusOK
}

// printConfig function displays loaded configuration on the standard output.
func printConfig() ExitCode {
	configBytes, err := json.MarshalIndent(conf.Config, "", "    ")

	if err != nil {
		log.Error().Err(err).Send()
		return 1
	}

	// convert configuration to string and displays it to standard output
	fmt.Println(string(configBytes))

	return ExitStatusOK
}

// printEnv function prints all environment variables to standard output.
func printEnv() ExitCode {
	for _, keyVal := range os.Environ() {
		fmt.Println(keyVal)
	}

	return ExitStatusOK
}

// startService function starts service and returns error code.
func startServer() ExitCode {
	_ = conf.GetSetupConfiguration()
	serverCfg := conf.GetServerConfiguration()
	metricsCfg := conf.GetMetricsConfiguration()
	servicesCfg := conf.GetServicesConfiguration()
	amsConfig := conf.GetAMSClientConfiguration()
	redisConf := conf.GetRedisConfiguration()
	rbacCfg := conf.GetRBACConfiguration()
	groupsChannel := make(chan []groups.Group)
	errorFoundChannel := make(chan bool)
	errorChannel := make(chan error)

	if metricsCfg.Namespace != "" {
		metrics.AddAPIMetricsWithNamespace(metricsCfg.Namespace)
	}

	amsClient, err := amsclient.NewAMSClient(amsConfig)
	if err != nil {
		log.Error().Err(err).Msg("Cannot init the AMSClient, using old approach")
		amsClient = nil
	} else {
		log.Info().Msg("AMSClient successfully created")
	}

	redisClient, err := services.NewRedisClient(redisConf)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize Redis server")
		return ExitStatusServerError
	}
	if err = redisClient.HealthCheck(); err != nil {
		log.Error().Err(err).Msg("failed to ping Redis server")
		return ExitStatusServerError
	}
	log.Info().Msg("Redis client created, Redis server is responding")

	rbac, err := auth.NewRBACClient(&rbacCfg)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize RBAC client")
		return ExitStatusServerError
	}
	serverInstance = server.New(serverCfg, servicesCfg, amsClient, redisClient, groupsChannel, errorFoundChannel, errorChannel, rbac)

	// fill-in additional info used by /info endpoint handler
	fillInInfoParams(serverInstance.InfoParams)

	proxy_content.SetContentDirectoryTimeout(servicesCfg.ContentDirectoryTimeout)
	go updateGroupInfo(servicesCfg, groupsChannel, errorFoundChannel, errorChannel)
	go proxy_content.RunUpdateContentLoop(servicesCfg)

	err = serverInstance.Start()
	if err != nil {
		log.Error().Err(err).Msg("HTTP(s) start error")
		return ExitStatusServerError
	}

	return ExitStatusOK
}

// fillInInfoParams function fills-in additional info used by /info endpoint
// handler
func fillInInfoParams(params map[string]string) {
	params["BuildVersion"] = BuildVersion
	params["BuildTime"] = BuildTime
	params["BuildBranch"] = BuildBranch
	params["BuildCommit"] = BuildCommit
	params["UtilsVersion"] = UtilsVersion
}

// updateGroupInfo function is run in a goroutine. It runs forever, waiting for 1 of 2 events: a Ticker or a channel
// * If ticker comes first, the groups configuration is updated, doing a request to the content-service
// * If the channel comes first, the latest valid groups configuration is send through the channel
func updateGroupInfo(servicesConf services.Configuration,
	groupsChannel chan []groups.Group,
	errorFoundChannel chan bool,
	errorChannel chan error) {
	var currentGroups []groups.Group
	var currentError error
	var currentErrorFound bool
	retrievedGroups, err := services.GetGroups(servicesConf)

	if err != nil {
		log.Error().Err(err).Msg("Error retrieving groups")
		currentErrorFound = true
		var e *url.Error
		if errors.As(err, &e) {
			currentError = &server.ContentServiceUnavailableError{}
		} else {
			currentError = err
		}
	} else {
		currentErrorFound = false
		currentGroups = retrievedGroups
	}

	uptimeTicker := time.NewTicker(servicesConf.GroupsPollingTime)
	log.Info().Msgf("Updating groups configuration each %f seconds", servicesConf.GroupsPollingTime.Seconds())

	for {
		select {
		case <-uptimeTicker.C:
			retrievedGroups, err = services.GetGroups(servicesConf)
			currentGroups, currentErrorFound, currentError = handleGroupError(err, currentGroups, retrievedGroups)
		case errorFoundChannel <- currentErrorFound:
		case errorChannel <- currentError:
		case groupsChannel <- currentGroups:
		}
	}
}

// handleGroupError handles error after retrieving groups info in updateGroupInfo
func handleGroupError(err error,
	currentGroups []groups.Group,
	newGroups []groups.Group) ([]groups.Group, bool, error) {
	if err != nil {
		log.Error().Err(err).Msg("Error retrieving groups")
		var e *url.Error
		if errors.As(err, &e) {
			return currentGroups, true, &server.ContentServiceUnavailableError{}
		}
		return currentGroups, true, err
	}
	return newGroups, false, nil
}

// handleCommand select the function to be called depending on command argument
func handleCommand(command string) ExitCode {
	switch command {
	case commandStartService:
		return startServer()

	case commandPrintVersion:
		printVersionInfo()
		return ExitStatusOK

	case commandPrintHelp:
		printHelp()
		return ExitStatusOK

	case commandPrintConfig:
		printConfig()
		return ExitStatusOK

	case commandPrintEnv:
		printEnv()
		return ExitStatusOK
	}

	return ExitStatusOK
}

// main represents entry point to CLI client.
func main() {
	err := conf.LoadConfiguration(defaultConfigFileName)

	if err != nil {
		panic(err)
	}

	err = logger.InitZerolog(
		conf.GetLoggingConfiguration(),
		conf.GetCloudWatchConfiguration(),
		conf.GetSentryLoggingConfiguration(),
	)
	if err != nil {
		panic(err)
	}

	var (
		showHelp    bool
		showVersion bool
	)
	flag.BoolVar(&showHelp, "help", false, "Show the help")
	flag.BoolVar(&showVersion, "version", false, "Show the version and exit")
	flag.Parse()

	if showHelp {
		os.Exit(int(printHelp()))
	}

	if showVersion {
		printVersionInfo()
		os.Exit(ExitStatusOK)
	}

	args := flag.Args()

	command := commandStartService
	if len(args) >= 1 {
		command = strings.ToLower(strings.TrimSpace(args[0]))
	}

	os.Exit(int(handleCommand(command)))
}
