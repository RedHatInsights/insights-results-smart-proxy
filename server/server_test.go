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

package server_test

import (
	"bytes"
	"encoding/gob"
	"github.com/RedHatInsights/insights-results-smart-proxy/content"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	ics_server "github.com/RedHatInsights/insights-content-service/server"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ira_server "github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	gock "gopkg.in/h2non/gock.v1"

	"github.com/RedHatInsights/insights-results-smart-proxy/server"
	"github.com/RedHatInsights/insights-results-smart-proxy/services"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
)

const (
	testTimeout = 10 * time.Second
)

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

func TestServerStartError(t *testing.T) {
	testServer := server.New(server.Configuration{
		Address:   "localhost:99999",
		APIPrefix: "",
	}, services.Configuration{
		AggregatorBaseEndpoint: "http://localhost:8081/api/v1/",
		ContentBaseEndpoint:    "http://localhost:8082/api/v1/",
	},
		nil,
	)

	err := testServer.Start()
	assert.EqualError(t, err, "listen tcp: address 99999: invalid port")
}

func MustGobSerialize(t testing.TB, obj interface{}) []byte {
	buf := new(bytes.Buffer)

	err := gob.NewEncoder(buf).Encode(obj)
	helpers.FailOnError(t, err)

	res, err := ioutil.ReadAll(buf)
	helpers.FailOnError(t, err)

	return res
}

func TestHTTPServer_ReportEndpoint(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		defer gock.Off()

		gock.New(helpers.DefaultServicesConfig.AggregatorBaseEndpoint).
			Get("/").
			AddMatcher(helpers.NewGockAPIEndpointMatcher(ira_server.ReportEndpoint)).
			Reply(200).
			JSON(testdata.Report3RulesExpectedResponse)

		gock.New(helpers.DefaultServicesConfig.ContentBaseEndpoint).
			Get("/").
			AddMatcher(helpers.NewGockAPIEndpointMatcher(ics_server.AllContentEndpoint)).
			Reply(200).
			Body(bytes.NewBuffer(MustGobSerialize(t, &testdata.RuleContentDirectory3Rules)))

		go content.RunUpdateContentLoop(helpers.DefaultServicesConfig)

		helpers.AssertAPIRequest(t, nil, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.ClusterName},
			UserID:       testdata.UserID,
			OrgID:        testdata.OrgID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusOK,
			Body:       helpers.ToJSONString(testdata.SmartProxyReportResponse3Rules),
		})
	}, testTimeout)
}

// TODO: test more cases for report endpoint
