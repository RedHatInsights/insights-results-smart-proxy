// Copyright 2021 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package amsclient_test

import (
	"crypto/rsa"
	"net/http"
	"testing"
	"time"

	"github.com/bmizerany/assert"
	"github.com/dgrijalva/jwt-go"
	"github.com/rs/zerolog"
	"gopkg.in/h2non/gock.v1"

	"github.com/RedHatInsights/insights-results-smart-proxy/amsclient"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/testdata"
)

const (
	organizationsSearchEndpoint = "api/accounts_mgmt/v1/organizations?fields=id%%2Cexternal_id&search=external_id+%%3D+{orgID}"
	subscriptionsSearchEndpoint = "api/accounts_mgmt/v1/subscriptions?fields=external_cluster_id&page={pageNum}&search=organization_id+is+%%27{orgID}%%27&size={pageSize}"

	subscriptionsSearchEndpointWithFilter = ("api/accounts_mgmt/v1/subscriptions?fields=external_cluster_id&page={pageNum}&" +
		"search=organization_id+is+%%27{orgID}%%27+and+status+in+%%28%%27{status1}%%27%%2C%%27{status2}%%27%%29&size={pageSize}")
)

var (
	// Public and private key that will be used to sign and verify tokens in the tests:
	jwtPublicKey  *rsa.PublicKey
	jwtPrivateKey *rsa.PrivateKey

	defaultConfig amsclient.Configuration
)

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
	var err error

	jwtPublicKey, err = jwt.ParseRSAPublicKeyFromPEM([]byte(jwtPublicKeyPEM))
	if err != nil {
		panic(err)
	}

	jwtPrivateKey, err = jwt.ParseRSAPrivateKeyFromPEM([]byte(jwtPrivateKeyPEM))
	if err != nil {
		panic(err)
	}

	defaultConfig = amsclient.Configuration{
		Token:    MakeTokenString("Bearer", 5*time.Minute),
		URL:      "https://localhost:8080",
		PageSize: 100,
	}
}

// TestClientCreation tests if creating a client works as expected
func TestClientCreationError(t *testing.T) {
	// define a configuration based on default, but without token
	config := defaultConfig

	// Expicitely set all authentication configuration to ""
	config.Token = ""
	config.ClientID = ""
	config.ClientSecret = ""

	_, err := amsclient.NewAMSClient(config)
	assert.NotEqual(t, nil, err)
}

func TestGetOrganization(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	c, err := amsclient.NewAMSClientWithTransport(defaultConfig, gock.DefaultTransport)
	helpers.FailOnError(t, err)

	// prepare organizations response
	helpers.GockExpectAPIRequest(t, defaultConfig.URL, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     organizationsSearchEndpoint,
		EndpointArgs: []interface{}{testdata.ExternalOrgID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: helpers.ToJSONString(testdata.OrganizationResponse),
	})

	orgID, err := c.GetInternalOrgIDFromExternal(testdata.ExternalOrgID)
	helpers.FailOnError(t, err)
	assert.Equal(t, testdata.InternalOrgID, orgID)
}

func TestOrganizationBadResponses(t *testing.T) {
	c, err := amsclient.NewAMSClientWithTransport(defaultConfig, gock.DefaultTransport)
	helpers.FailOnError(t, err)
	defer helpers.CleanAfterGock(t)

	// prepare 3 responses that will cause different errors
	// response OK, but unexpected data
	helpers.GockExpectAPIRequest(t, defaultConfig.URL, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     organizationsSearchEndpoint,
		EndpointArgs: []interface{}{testdata.ExternalOrgID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: helpers.ToJSONString(testdata.OrganizationResponse2IDs),
	})

	// response Error
	helpers.GockExpectAPIRequest(t, defaultConfig.URL, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     organizationsSearchEndpoint,
		EndpointArgs: []interface{}{testdata.ExternalOrgID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusNotFound,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	})

	_, err = c.GetInternalOrgIDFromExternal(testdata.ExternalOrgID)
	assert.NotEqual(t, nil, err)

	_, err = c.GetInternalOrgIDFromExternal(testdata.ExternalOrgID)
	assert.NotEqual(t, nil, err)
}

func TestClusterForOrganization(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	c, err := amsclient.NewAMSClientWithTransport(defaultConfig, gock.DefaultTransport)
	helpers.FailOnError(t, err)

	// prepare organizations response
	helpers.GockExpectAPIRequest(t, defaultConfig.URL, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     organizationsSearchEndpoint,
		EndpointArgs: []interface{}{testdata.ExternalOrgID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: helpers.ToJSONString(testdata.OrganizationResponse),
	})

	// prepare cluster list requests
	helpers.GockExpectAPIRequest(t, defaultConfig.URL, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     subscriptionsSearchEndpoint,
		EndpointArgs: []interface{}{1, testdata.InternalOrgID, defaultConfig.PageSize},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: helpers.ToJSONString(testdata.SubscriptionsResponse),
	})
	// second and more requests will be done until the last one returns an empty response (0 sized)
	helpers.GockExpectAPIRequest(t, defaultConfig.URL, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     subscriptionsSearchEndpoint,
		EndpointArgs: []interface{}{2, testdata.InternalOrgID, defaultConfig.PageSize},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: helpers.ToJSONString(testdata.SubscriptionEmptyResponse),
	})

	clusterList := c.GetClustersForOrganization(testdata.ExternalOrgID, nil, nil)
	assert.Equal(t, 2, len(clusterList))
}

func TestClusterForOrganizationWithFiltering(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	c, err := amsclient.NewAMSClientWithTransport(defaultConfig, gock.DefaultTransport)
	helpers.FailOnError(t, err)

	// prepare organizations response
	helpers.GockExpectAPIRequest(t, defaultConfig.URL, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     organizationsSearchEndpoint,
		EndpointArgs: []interface{}{testdata.ExternalOrgID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: helpers.ToJSONString(testdata.OrganizationResponse),
	})

	// prepare cluster list requests
	helpers.GockExpectAPIRequest(t, defaultConfig.URL, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     subscriptionsSearchEndpointWithFilter,
		EndpointArgs: []interface{}{1, testdata.InternalOrgID, amsclient.StatusArchived, amsclient.StatusDeprovisioned, defaultConfig.PageSize},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: helpers.ToJSONString(testdata.SubscriptionsResponse),
	})
	// second and more requests will be done until the last one returns an empty response (0 sized)
	helpers.GockExpectAPIRequest(t, defaultConfig.URL, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     subscriptionsSearchEndpointWithFilter,
		EndpointArgs: []interface{}{2, testdata.InternalOrgID, amsclient.StatusArchived, amsclient.StatusDeprovisioned, defaultConfig.PageSize},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: helpers.ToJSONString(testdata.SubscriptionEmptyResponse),
	})

	clusterList := c.GetClustersForOrganization(
		testdata.ExternalOrgID,
		[]string{amsclient.StatusArchived, amsclient.StatusDeprovisioned},
		nil,
	)
	assert.Equal(t, 2, len(clusterList))
}
