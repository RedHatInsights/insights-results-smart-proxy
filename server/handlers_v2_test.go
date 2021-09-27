package server_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
)

func TestHTTPServer_SetRating(t *testing.T) {
	defer helpers.CleanAfterGock(t)

	rating := `{"rule": "rule_module|error_key","rating":-1}`
	aggregatorResponse := fmt.Sprintf(`{"status":"ok", "ratings":%s}`, rating)

	// prepare content
	helpers.GockExpectAPIRequest(
		t,
		helpers.DefaultServicesConfig.AggregatorBaseEndpoint,
		&helpers.APIRequest{
			Method:       http.MethodPost,
			Endpoint:     ira_server.Rating,
			EndpointArgs: []interface{}{testdata.UserID, testdata.OrgID},
			Body:         rating,
		},
		&helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       aggregatorResponse,
		},
	)

	helpers.AssertAPIRequest(
		t,
		nil,
		nil,
		nil,
		&helpers.APIRequest{
			Method:   http.MethodPost,
			Endpoint: server.Rating,
			Body:     rating,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       rating,
		},
	)
}
