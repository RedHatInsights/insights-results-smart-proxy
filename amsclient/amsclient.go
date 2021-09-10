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

	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// AMSClient allow us to interact the AMS API
type AMSClient struct {
	connection *sdk.Connection
	pageSize   int
}

// NewAMSClient creates an AMSClient from the configuration
func NewAMSClient(conf Configuration) (*AMSClient, error) {
	conn, err := sdk.NewConnectionBuilder().
		URL(conf.URL).
		Tokens(conf.Token).
		Build()

	if err != nil {
		log.Error().Err(err).Msg("Unable to build the connection to AMS API")
		return nil, err
	}

	return &AMSClient{
		connection: conn,
		pageSize:   conf.PageSize,
	}, nil
}

// GetClustersForOrganization retrieves the clusters for a given organization using the default client
// it allows to filter the clusters by their status (statusNegativeFilter will exclude the clusters with status in that list)
func (c *AMSClient) GetClustersForOrganization(orgID types.OrgID, statusFilter, statusNegativeFilter []string) []types.ClusterName {
	var retval []types.ClusterName = []types.ClusterName{}

	internalOrgID, err := c.getInternalOrgIDFromExternal(orgID)
	if err != nil {
		return retval
	}

	searchQuery := generateSearchParameter(internalOrgID, statusFilter, statusNegativeFilter)
	subscriptionClient := c.connection.AccountsMgmt().V1().Subscriptions()

	for pageNum := 1; ; pageNum++ {
		response, err := subscriptionClient.List().Size(c.pageSize).Page(pageNum).
			Search(searchQuery).Fields("external_cluster_id").Send()

		if err != nil {
			return retval
		}

		// When an empty page is returned, then exit the loop
		if response.Size() == 0 {
			break
		}

		for _, item := range response.Items().Slice() {
			clusterID, ok := item.GetExternalClusterID()
			if !ok {
				fmt.Println("Not external cluster ID")
				continue
			}
			retval = append(retval, types.ClusterName(clusterID))
		}
	}

	return retval
}

func (c *AMSClient) getInternalOrgIDFromExternal(orgID types.OrgID) (string, error) {
	orgsClient := c.connection.AccountsMgmt().V1().Organizations()
	response, err := orgsClient.List().Search(fmt.Sprintf("external_id = %d", orgID)).Send()
	if err != nil {
		log.Error().Err(err).Msg("")
		return "", err
	}

	if response.Items().Len() != 1 {
		log.Error().Int("orgIDs length", response.Items().Len()).Msg("More than one organization for the given orgID")
		return "", fmt.Errorf("More than one organization for the given orgID (%d)", orgID)
	}

	internalID, ok := response.Items().Get(0).GetID()
	if !ok {
		log.Error().Msgf("Organization %d doesn't have proper internal ID", orgID)
		return "", fmt.Errorf("Organization %d doesn't have proper internal ID", orgID)
	}

	return internalID, nil
}
