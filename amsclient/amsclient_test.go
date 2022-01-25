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
	"github.com/RedHatInsights/insights-results-smart-proxy/types"
)

const (
	organizationsSearchEndpoint = "api/accounts_mgmt/v1/organizations?fields=id%%2Cexternal_id&search=external_id+%%3D+{orgID}"

	subscriptionsSearchEndpoint = ("api/accounts_mgmt/v1/subscriptions?fields=external_cluster_id%%2Cdisplay_name%%2Ccluster_id&page={pageNum}&" +
		"search=organization_id+is+%%27{orgID}%%27+and+cluster_id+%%21%%3D+%%27%%27&size={pageSize}")
	subscriptionsSearchEndpointWithFilter = ("api/accounts_mgmt/v1/subscriptions?fields=external_cluster_id%%2Cdisplay_name%%2Ccluster_id&page={pageNum}&" +
		"search=organization_id+is+%%27{orgID}%%27+and+cluster_id+%%21%%3D+%%27%%27+and+status+in+%%28%%27{status1}%%27%%2C%%27{status2}%%27%%29&size={pageSize}")
	subscriptionsSearchEndpointWithDefaultFilter = ("api/accounts_mgmt/v1/subscriptions?fields=external_cluster_id%%2Cdisplay_name%%2Ccluster_id&page={pageNum}&" +
		"search=organization_id+is+%%27{orgID}%%27+and+cluster_id+%%21%%3D+%%27%%27+and+status+not+in+%%28%%27{status1}%%27%%2C%%27{status2}%%27%%2C%%27{status3}%%27%%29&size={pageSize}")
	clusterDetailsSearchEndpoint = ("api/accounts_mgmt/v1/subscriptions?fields=external_cluster_id%%2Cdisplay_name%%2Ccluster_id&page={pageNum}&" +
		"search=external_cluster_id+%%3D+%%27{clusterID}%%27&size={pageSize}")
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

	clusterList, err := c.GetClustersForOrganization(testdata.ExternalOrgID, nil, []string{})
	helpers.FailOnError(t, err)
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

	clusterList, err := c.GetClustersForOrganization(
		testdata.ExternalOrgID,
		[]string{amsclient.StatusArchived, amsclient.StatusDeprovisioned},
		[]string{},
	)

	helpers.FailOnError(t, err)
	assert.Equal(t, 2, len(clusterList))
}

func TestClusterForOrganizationWithDefaultFiltering(t *testing.T) {
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
		Method:   http.MethodGet,
		Endpoint: subscriptionsSearchEndpointWithDefaultFilter,
		EndpointArgs: []interface{}{
			1, testdata.InternalOrgID,
			amsclient.StatusArchived, amsclient.StatusDeprovisioned, amsclient.StatusReserved,
			defaultConfig.PageSize,
		},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: helpers.ToJSONString(testdata.SubscriptionsResponse),
	})
	// second and more requests will be done until the last one returns an empty response (0 sized)
	helpers.GockExpectAPIRequest(t, defaultConfig.URL, &helpers.APIRequest{
		Method:   http.MethodGet,
		Endpoint: subscriptionsSearchEndpointWithDefaultFilter,
		EndpointArgs: []interface{}{
			2, testdata.InternalOrgID,
			amsclient.StatusArchived, amsclient.StatusDeprovisioned, amsclient.StatusReserved,
			defaultConfig.PageSize,
		},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: helpers.ToJSONString(testdata.SubscriptionEmptyResponse),
	})

	clusterList, err := c.GetClustersForOrganization(
		testdata.ExternalOrgID,
		nil,
		nil,
	)

	helpers.FailOnError(t, err)
	assert.Equal(t, 2, len(clusterList))
}

func TestGetClustersForOrganizationOnError(t *testing.T) {
	client, err := amsclient.NewAMSClient(defaultConfig)
	helpers.FailOnError(t, err) // Doesn't fail because ocm-sdk doesn't perform any checks

	clusters, err := client.GetClustersForOrganization(testdata.ExternalOrgID, nil, nil)
	if err == nil {
		t.Fail()
	}
	assert.Equal(t, 0, len(clusters))
}

func TestGetClusterDetailsFromExternalClusterId(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	c, err := amsclient.NewAMSClientWithTransport(defaultConfig, gock.DefaultTransport)
	helpers.FailOnError(t, err)

	// prepare subscription filtered by external_cluster_id response
	helpers.GockExpectAPIRequest(t, defaultConfig.URL, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     clusterDetailsSearchEndpoint,
		EndpointArgs: []interface{}{1, testdata.ClusterName1, defaultConfig.PageSize},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: helpers.ToJSONString(testdata.SubscriptionsResponse),
	})

	// second request will get an empty response (0 sized), so no more requests will be made
	helpers.GockExpectAPIRequest(t, defaultConfig.URL, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     clusterDetailsSearchEndpoint,
		EndpointArgs: []interface{}{2, testdata.ClusterName1, defaultConfig.PageSize},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: helpers.ToJSONString(testdata.SubscriptionEmptyResponse),
	})

	clusterListInfo := c.GetClusterDetailsFromExternalClusterID(
		testdata.ClusterName1,
	)

	assert.Equal(t, clusterListInfo, types.ClusterInfo{
		ID:          testdata.ClusterName1,
		DisplayName: testdata.ClusterDisplayName1,
	})
}

func TestGetClusterDetailsUnknownExternalClusterId(t *testing.T) {
	defer helpers.CleanAfterGock(t)
	c, err := amsclient.NewAMSClientWithTransport(defaultConfig, gock.DefaultTransport)
	helpers.FailOnError(t, err)

	// prepare subscription filtered by external_cluster_id response
	helpers.GockExpectAPIRequest(t, defaultConfig.URL, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     clusterDetailsSearchEndpoint,
		EndpointArgs: []interface{}{1, testdata.ClusterName1, defaultConfig.PageSize},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: helpers.ToJSONString(testdata.SubscriptionEmptyResponse),
	})

	clusterListInfo := c.GetClusterDetailsFromExternalClusterID(
		testdata.ClusterName1,
	)

	assert.Equal(t, clusterListInfo, types.ClusterInfo{})
}
