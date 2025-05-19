// Copyright 2021, 2022 Red Hat, Inc
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
	"errors"
	"net/http"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"

	utypes "github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/RedHatInsights/insights-results-smart-proxy/amsclient"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/helpers"
	"github.com/RedHatInsights/insights-results-smart-proxy/tests/testdata"
	"github.com/RedHatInsights/insights-results-smart-proxy/types"
)

// OCM SDK encodes URLs with escaped hex characters
// %%2C means ,
// %%3D means =
// %%21 means !
// %28%27 %27%29 means (' ')
const (
	organizationsSearchEndpoint = "api/accounts_mgmt/v1/organizations?fields=id%%2Cexternal_id&search=external_id+%%3D+{orgID}"

	subscriptionsSearchEndpoint = ("api/accounts_mgmt/v1/subscriptions?fields=external_cluster_id%%2Cdisplay_name%%2Ccluster_id%%2Cmanaged%%2Cstatus&page={pageNum}&" +
		"search=organization_id+in+%%28%%27{orgID}%%27%%29+and+cluster_id+%%21%%3D+%%27%%27&size={pageSize}")
	subscriptionsSearchEndpointMultipleOrgs = ("api/accounts_mgmt/v1/subscriptions?fields=external_cluster_id%%2Cdisplay_name%%2Ccluster_id%%2Cmanaged%%2Cstatus&page={pageNum}&" +
		"search=organization_id+in+%%28%%27{orgID1}%%27%%2C%%27{orgID2}%%27%%29+and+cluster_id+%%21%%3D+%%27%%27&size={pageSize}")
	subscriptionsSearchEndpointWithFilter = ("api/accounts_mgmt/v1/subscriptions?fields=external_cluster_id%%2Cdisplay_name%%2Ccluster_id%%2Cmanaged%%2Cstatus&page={pageNum}&" +
		"search=organization_id+in+%%28%%27{orgID}%%27%%29+and+cluster_id+%%21%%3D+%%27%%27+and+status+in+%%28%%27{status1}%%27%%2C%%27{status2}%%27%%29&size={pageSize}")
	subscriptionsSearchEndpointWithDefaultFilter = ("api/accounts_mgmt/v1/subscriptions?fields=external_cluster_id%%2Cdisplay_name%%2Ccluster_id%%2Cmanaged%%2Cstatus&page={pageNum}&" +
		"search=organization_id+in+%%28%%27{orgID}%%27%%29+and+cluster_id+%%21%%3D+%%27%%27+and+status+not+in+%%28%%27{status1}%%27%%2C%%27{status2}%%27%%2C%%27{status3}%%27%%29&size={pageSize}")
	clusterDetailsSearchEndpoint = ("api/accounts_mgmt/v1/subscriptions?fields=external_cluster_id%%2Cdisplay_name%%2Ccluster_id%%2Cmanaged%%2Cstatus&page={pageNum}&" +
		"search=external_cluster_id+%%3D+%%27{clusterID}%%27&size={pageSize}")
	singleClusterInfoEndpoint = ("api/accounts_mgmt/v1/subscriptions?fields=external_cluster_id%%2Cdisplay_name%%2Ccluster_id%%2Cmanaged%%2Cstatus&page={pageNum}&" +
		"search=organization_id+in+%%28%%27{orgID}%%27%%29+and+external_cluster_id+%%3D+%%27{clusterID}%%27&size={pageSize}")
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

	// prepare cluster list request response
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
	assert.ElementsMatch(t, testdata.OKClustersForOrganization, clusterList)
}

func TestClusterForOrganizationNoInternalOrgID(t *testing.T) {
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
		Body: helpers.ToJSONString(testdata.OrganizationResponseNoID),
	})

	_, err = c.GetClustersForOrganization(testdata.ExternalOrgID, nil, []string{})
	assert.Error(t, err)

	var notFoundError *utypes.ItemNotFoundError
	ok := errors.As(err, &notFoundError)

	assert.True(t, ok)
}

func TestClusterForOrganization2InternalOrgIDs(t *testing.T) {
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
		Body: helpers.ToJSONString(testdata.OrganizationResponse2IDs),
	})

	// prepare cluster list request response
	helpers.GockExpectAPIRequest(t, defaultConfig.URL, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     subscriptionsSearchEndpointMultipleOrgs,
		EndpointArgs: []interface{}{1, testdata.InternalOrgID, testdata.InternalOrgID2, defaultConfig.PageSize},
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
		Endpoint:     subscriptionsSearchEndpointMultipleOrgs,
		EndpointArgs: []interface{}{2, testdata.InternalOrgID, testdata.InternalOrgID2, defaultConfig.PageSize},
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
	assert.ElementsMatch(t, testdata.OKClustersForOrganization, clusterList)
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

	// prepare cluster list request response
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
	assert.ElementsMatch(t, testdata.OKClustersForOrganization, clusterList)
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

	// prepare cluster list request response
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
	assert.ElementsMatch(t, testdata.OKClustersForOrganization, clusterList)
}

func TestClusterForOrganizationWithEmptyClusterIDs(t *testing.T) {
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

	// prepare cluster list request response. Response has 2 invalid clusters and 1 valid one
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
		Body: helpers.ToJSONString(testdata.SubscriptionsResponseEmptyClusterIDs),
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
	// we expect 2 out of 3 clusters to be excluded because they don't have external cluster ID
	assert.Equal(t, 1, len(clusterList))
	// cluster 1 is managed (has `managed` attribute set in AMS response)
	assert.Equal(t, true, clusterList[0].Managed)
}

// TestClusterForOrganizationCCXDEV_8829_Reproducer reproducer for https://issues.redhat.com/browse/CCXDEV-8829
func TestClusterForOrganizationCCXDEV_8829_Reproducer(t *testing.T) {
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

	// prepare cluster list request response. Response has 2 invalid clusters and 1 valid one
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
		Body: helpers.ToJSONString(testdata.SubscriptionsResponseInvalidUUID),
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
	// we expect 1 cluster to be excluded because it has invalid UUID
	assert.Equal(t, 1, len(clusterList))
	// remaining cluster has the correct UUID
	assert.Equal(t, types.ClusterName(testdata.ClusterName1), clusterList[0].ID)
}

// TestClusterForOrganizationCCXDEV_11659_Reproducer reproducer for https://issues.redhat.com/browse/CCXDEV-11659
// AMS API can sometimes return duplicated clusters, we must filter the duplicates out.
func TestClusterForOrganizationCCXDEV_11659_Reproducer(t *testing.T) {
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

	// prepare cluster list response. Response has 4 valid clusters, but 2 of them are duplicate.
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
		Body: helpers.ToJSONString(testdata.SubscriptionsResponseDuplicateRecords),
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
	// we expect 2 cluster to be excluded because they are duplicate
	assert.Equal(t, 2, len(clusterList))
	// remaining clusters have the correct UUIDs
	assert.ElementsMatch(t,
		[]types.ClusterName{clusterList[0].ID, clusterList[1].ID},
		[]types.ClusterName{testdata.ClusterName1, testdata.ClusterName2},
	)
	assert.Equal(t, types.ClusterName(testdata.ClusterName1), clusterList[0].ID)
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

	clusterInfo := c.GetClusterDetailsFromExternalClusterID(
		testdata.ClusterName1,
	)

	// cluster 1 is managed (has `managed` attribute set in AMS response)
	assert.Equal(t, clusterInfo, types.ClusterInfo{
		ID:          testdata.ClusterName1,
		DisplayName: testdata.ClusterDisplayName1,
		Managed:     true,
		Status:      testdata.ActiveStatus,
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

func TestGetSingleClusterInfoForOrganizationNotFound(t *testing.T) {
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

	// prepare empty response
	helpers.GockExpectAPIRequest(t, defaultConfig.URL, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     singleClusterInfoEndpoint,
		EndpointArgs: []interface{}{1, testdata.InternalOrgID, testdata.ClusterName1, defaultConfig.PageSize},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: helpers.ToJSONString(testdata.SubscriptionEmptyResponse),
	})

	clusterInfo, err := c.GetSingleClusterInfoForOrganization(
		testdata.ExternalOrgID,
		testdata.ClusterName1,
	)

	assert.Equal(t, clusterInfo, types.ClusterInfo{})
	assert.IsType(t, err, &utypes.ItemNotFoundError{})
}

func TestGetSingleClusterInfoForOrganization(t *testing.T) {
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

	helpers.GockExpectAPIRequest(t, defaultConfig.URL, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     singleClusterInfoEndpoint,
		EndpointArgs: []interface{}{1, testdata.InternalOrgID, testdata.ClusterName1, defaultConfig.PageSize},
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
		Endpoint:     singleClusterInfoEndpoint,
		EndpointArgs: []interface{}{2, testdata.InternalOrgID, testdata.ClusterName1, defaultConfig.PageSize},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: helpers.ToJSONString(testdata.SubscriptionEmptyResponse),
	})

	clusterInfo, err := c.GetSingleClusterInfoForOrganization(
		testdata.ExternalOrgID,
		testdata.ClusterName1,
	)
	helpers.FailOnError(t, err)

	assert.Equal(t, clusterInfo.DisplayName, testdata.ClusterDisplayName1)
	assert.Equal(t, clusterInfo.Managed, true)
	assert.Equal(t, clusterInfo.Status, testdata.ActiveStatus)
}
