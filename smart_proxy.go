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

// Entry point to the insights results smart proxy
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/RedHatInsights/insights-content-service/groups"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-smart-proxy/conf"
	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"
)

const (
	// ExitStatusOK means that the tool finished with success
	ExitStatusOK = iota
	// ExitStatusServerError means that the HTTP server cannot be initialized
	ExitStatusServerError
	defaultConfigFileName = "config"
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

var serverInstance *server.HTTPServer

func printHelp() int {
	fmt.Printf(helpMessageTemplate, os.Args[0])
	return ExitStatusOK
}

func printConfig() int {
	configBytes, err := json.MarshalIndent(conf.Config, "", "    ")

	if err != nil {
		log.Error().Err(err)
		return 1
	}

	fmt.Println(string(configBytes))

	return ExitStatusOK
}

func printEnv() int {
	for _, keyVal := range os.Environ() {
		fmt.Println(keyVal)
	}

	return ExitStatusOK
}

// startService starts service and returns error code
func startServer() int {
	serverCfg := conf.GetServerConfiguration()
	servicesCfg := conf.GetServicesConfiguration()
	groupsChannel := make(chan []groups.Group)
	serverInstance = server.New(serverCfg, servicesCfg, groupsChannel)

	go updateGroupInfo(servicesCfg, groupsChannel)

	err := serverInstance.Start()
	if err != nil {
		log.Error().Err(err).Msg("HTTP(s) start error")
		return ExitStatusServerError
	}

	return ExitStatusOK
}

// updateGroupInfo function is run in a goroutine. It runs forever, waiting for 1 of 2 events: a Ticker or a channel
// * If ticker comes first, the groups configuration is updated, doing a request to the content-service
// * If the channel comes first, the latest valid groups configuration is send through the channel
func updateGroupInfo(servicesConf services.Configuration, groupsChannel chan []groups.Group) {
	var currentGroups []groups.Group
	groups, err := services.GetGroups(servicesConf)

	if err != nil {
		log.Error().Err(err).Msg("Error retrieving groups")
	} else {
		currentGroups = groups
	}

	uptimeTicker := time.NewTicker(servicesConf.GroupsPollingTime)
	log.Info().Msgf("Updating groups configuration each %f seconds", servicesConf.GroupsPollingTime.Seconds())

	for {
		select {
		case <-uptimeTicker.C:
			groups, err = services.GetGroups(servicesConf)

			if err != nil {
				log.Error().Err(err).Msg("Error retrieving groups")
			} else {
				currentGroups = groups
			}

		case groupsChannel <- currentGroups:
		}
	}
}

// handleCommand select the function to be called depending on command argument
func handleCommand(command string) int {
	switch command {
	case "start-service":
		return startServer()

	case "print-version":
		printVersionInfo()
		return ExitStatusOK

	case "print-help":
		printHelp()
		return ExitStatusOK

	case "print-config":
		printConfig()
		return ExitStatusOK

	case "print-env":
		printEnv()
		return ExitStatusOK
	}

	return ExitStatusOK
}

func main() {
	err := conf.LoadConfiguration(defaultConfigFileName)

	if err != nil {
		panic(err)
	}

	var (
		showHelp    bool
		showVersion bool
	)
	flag.BoolVar(&showHelp, "help", false, "Show the help")
	flag.BoolVar(&showVersion, "version", false, "Show the version an exit")
	flag.Parse()

	if showHelp {
		printHelp()
		os.Exit(ExitStatusOK)
	}

	if showVersion {
		printVersionInfo()
		os.Exit(ExitStatusOK)
	}

	var args []string
	args = flag.Args()

	command := "start-service"
	if len(args) >= 1 {
		command = strings.ToLower(strings.TrimSpace(args[0]))
	}

	os.Exit(handleCommand(command))
}
