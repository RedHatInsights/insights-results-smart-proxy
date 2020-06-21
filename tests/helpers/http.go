package helpers

import (
	"testing"
	"time"

	"github.com/RedHatInsights/insights-content-service/groups"
	"github.com/RedHatInsights/insights-results-aggregator-utils/tests/helpers"

	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"
)

type APIRequest = helpers.APIRequest
type APIResponse = helpers.APIResponse

var (
	ExecuteRequest             = helpers.ExecuteRequest
	CheckResponseBodyJSON      = helpers.CheckResponseBodyJSON
	AssertReportResponsesEqual = helpers.AssertReportResponsesEqual
	NewGockAPIEndpointMatcher = helpers.NewGockAPIEndpointMatcher
)

var (
	DefaultServerConfig = server.Configuration{
		Address:     ":8081",
		APIPrefix:   "/api/v1/",
		APISpecFile: "openapi.json",
		Debug:       true,
		Auth:        false,
		AuthType:    "",
		UseHTTPS:    false,
		EnableCORS:  false,
	}

	DefaultServicesConfig = services.Configuration{
		AggregatorBaseEndpoint: "http://localhost:8080/",
		ContentBaseEndpoint:    "http://localhost:8082/",
		GroupsPollingTime:      1 * time.Minute,
	}
)

// AssertAPIRequest creates new server with provided
// serverConfig, servicesConfig (you can leave them nil to use the default ones),
// groupsChannel and contentChannel(can be nil)
// sends api request and checks api response (see docs for APIRequest and APIResponse)
func AssertAPIRequest(
	t testing.TB,
	serverConfig *server.Configuration,
	servicesConfig *services.Configuration,
	groupsChannel chan []groups.Group,
	request *helpers.APIRequest,
	expectedResponse *helpers.APIResponse,
) {
	if serverConfig == nil {
		serverConfig = &DefaultServerConfig
	}
	if servicesConfig == nil {
		servicesConfig = &DefaultServicesConfig
	}

	testServer := server.New(
		*serverConfig,
		*servicesConfig,
		groupsChannel,
	)

	helpers.AssertAPIRequest(t, testServer, serverConfig.APIPrefix, request, expectedResponse)
}

