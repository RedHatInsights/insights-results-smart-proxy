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

package amsclient

import (
	"fmt"
	"net/http"

	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	// defaultPageSize is the page size used when it is not defined in the configuration
	defaultPageSize = 100

	// strings for logging and errors
	orgNoInternalID     = "Organization doesn't have proper internal ID"
	orgMoreInternalOrgs = "More than one internal organization for the given orgID"
	orgIDTag            = "OrgID"

	// StatusDeprovisioned indicates the corresponding cluster subscription status
	StatusDeprovisioned = "Deprovisioned"
	// StatusArchived indicates the corresponding cluster subscription status
	StatusArchived = "Archived"
)

// AMSClient allow us to interact the AMS API
type AMSClient struct {
	connection *sdk.Connection
	pageSize   int
}

// NewAMSClient create an AMSClient from the configuration
func NewAMSClient(conf Configuration) (*AMSClient, error) {
	return NewAMSClientWithTransport(conf, nil)
}

// NewAMSClientWithTransport creates an AMSClient from the configuration, enabling to use a transport wrapper
func NewAMSClientWithTransport(conf Configuration, transport http.RoundTripper) (*AMSClient, error) {
	log.Info().Msg("Creating amsclient...")
	builder := sdk.NewConnectionBuilder().URL(conf.URL)

	if transport != nil {
		builder.TransportWrapper(func(http.RoundTripper) http.RoundTripper { return transport })
	}

	if conf.ClientID != "" && conf.ClientSecret != "" {
		builder = builder.Client(conf.ClientID, conf.ClientSecret)
	} else if conf.Token != "" {
		builder = builder.Tokens(conf.Token)
	} else {
		err := fmt.Errorf("No credentials provided. Cannot create the API client")
		log.Error().Err(err).Msg("Cannot create the connection builder")
		return nil, err
	}

	conn, err := builder.Build()

	if err != nil {
		log.Error().Err(err).Msg("Unable to build the connection to AMS API")
		return nil, err
	}

	if conf.PageSize <= 0 {
		conf.PageSize = defaultPageSize
	}

	return &AMSClient{
		connection: conn,
		pageSize:   conf.PageSize,
	}, nil
}

// GetClustersForOrganization retrieves the clusters for a given organization using the default client
// it allows to filter the clusters by their status (statusNegativeFilter will exclude the clusters with status in that list)
func (c *AMSClient) GetClustersForOrganization(orgID types.OrgID, statusFilter, statusNegativeFilter []string) (
	[]types.ClusterName,
	error,
) {
	log.Debug().Uint32(orgIDTag, uint32(orgID)).Msg("Looking cluster for the organization")
	var retval []types.ClusterName = []types.ClusterName{}

	internalOrgID, err := c.GetInternalOrgIDFromExternal(orgID)
	if err != nil {
		return retval, err
	}

	searchQuery := generateSearchParameter(internalOrgID, statusFilter, statusNegativeFilter)
	subscriptionListRequest := c.connection.AccountsMgmt().V1().Subscriptions().List()

	for pageNum := 1; ; pageNum++ {
		subscriptionListRequest = subscriptionListRequest.
			Size(c.pageSize).
			Page(pageNum).
			Fields("external_cluster_id").
			Search(searchQuery)

		log.Debug().Uint32(orgIDTag, uint32(orgID)).Msgf("Sending following request to AMS API: %v", subscriptionListRequest)
		response, err := subscriptionListRequest.Send()

		if err != nil {
			return retval, err
		}

		// When an empty page is returned, then exit the loop
		if response.Size() == 0 {
			break
		}

		for _, item := range response.Items().Slice() {
			clusterID, ok := item.GetExternalClusterID()
			if !ok {
				if id, ok := item.GetID(); ok {
					log.Info().Str("IntClusterID", id).Msg("Not external cluster ID")
				} else {
					log.Info().Msg("Not external cluster ID")
				}

				continue
			}
			retval = append(retval, types.ClusterName(clusterID))
		}
	}

	return retval, nil
}

// GetInternalOrgIDFromExternal will retrieve the internal organization ID from an external one using AMS API
func (c *AMSClient) GetInternalOrgIDFromExternal(orgID types.OrgID) (string, error) {
	log.Debug().Uint32(orgIDTag, uint32(orgID)).Msg(
		"Looking for the internal organization ID for an external one",
	)
	orgsListRequest := c.connection.AccountsMgmt().V1().Organizations().List()
	response, err := orgsListRequest.
		Search(fmt.Sprintf("external_id = %d", orgID)).
		Fields("id,external_id").
		Send()

	if err != nil {
		log.Error().Err(err).Msg("")
		return "", err
	}

	if response.Items().Len() != 1 {
		log.Error().Uint32(orgIDTag, uint32(orgID)).Msg(orgMoreInternalOrgs)
		return "", fmt.Errorf(orgMoreInternalOrgs)
	}

	internalID, ok := response.Items().Get(0).GetID()
	if !ok {
		log.Error().Uint32(orgIDTag, uint32(orgID)).Msg(orgNoInternalID)
		return "", fmt.Errorf(orgNoInternalID)
	}

	return internalID, nil
}
