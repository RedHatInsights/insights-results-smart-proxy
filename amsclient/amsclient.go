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
	"time"

	"github.com/google/uuid"
	sdk "github.com/openshift-online/ocm-sdk-go"
	accMgmt "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/rs/zerolog/log"

	utypes "github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/RedHatInsights/insights-results-smart-proxy/types"
)

const (
	// defaultPageSize is the page size used when it is not defined in the configuration
	defaultPageSize = 500

	// strings for logging and errors
	orgNoInternalID              = "Organization doesn't have proper internal ID"
	orgMoreInternalOrgs          = "More than one internal organization for the given orgID"
	orgIDRequestFailure          = "Request to get the organization info failed"
	subscriptionListRequestError = "problem executing subscription list request"
	orgIDTag                     = "OrgID"
	clusterIDTag                 = "ClusterID"
	numberOfClusterTag           = "Number of clusters"

	// StatusDeprovisioned indicates the corresponding cluster subscription status
	StatusDeprovisioned = "Deprovisioned"
	// StatusArchived indicates the corresponding cluster subscription status
	StatusArchived = "Archived"
	// StatusReserved means the cluster has reserved resources, but isn't initialized yet.
	StatusReserved = "Reserved"
)

var (
	// DefaultStatusNegativeFilters are filters that are applied to the AMS API subscriptions query when the filters are empty
	// We are either not interested in clusters in these states (Archived, Deprovisioned) or the cluster's
	// initialization hasn't finished yet (Reserved), meaning the cluster is not ready to start sending Insights archives,
	// as it might not even have a Cluster UUID assigned yet. When the initialization succeeds or fails, the cluster's
	// state becomes either Active or Deprovisioned.
	DefaultStatusNegativeFilters = []string{StatusArchived, StatusDeprovisioned, StatusReserved}
)

// AMSClient allow us to interact the AMS API
type AMSClient interface {
	GetClustersForOrganization(types.OrgID, []string, []string) (
		clusterInfoList []types.ClusterInfo,
		err error,
	)
	GetClusterDetailsFromExternalClusterID(types.ClusterName) (
		clusterInfo types.ClusterInfo,
	)
	GetSingleClusterInfoForOrganization(types.OrgID, types.ClusterName) (
		types.ClusterInfo, error,
	)
	GetMultiClusterInfoForOrganization(orgID types.OrgID, clusterIDs, statusFilter, statusNegativeFilter []string) (
		clusterInfoList []types.ClusterInfo, err error,
	)
}

// amsClientImpl is an implementation of the AMSClient interface
type amsClientImpl struct {
	connection *sdk.Connection
	pageSize   int
}

// NewAMSClient create an AMSClient from the configuration
func NewAMSClient(conf Configuration) (AMSClient, error) {
	return NewAMSClientWithTransport(conf, nil)
}

// NewAMSClientWithTransport creates an AMSClient from the configuration, enabling to use a transport wrapper
func NewAMSClientWithTransport(conf Configuration, transport http.RoundTripper) (AMSClient, error) {
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

	return &amsClientImpl{
		connection: conn,
		pageSize:   conf.PageSize,
	}, nil
}

// GetClustersForOrganization retrieves the clusters for a given organization using the default client
// it allows to filter the clusters by their status (statusNegativeFilter will exclude the clusters with status in that list)
// If nil is passed for filters, default filters will be applied. To select empty filters, pass an empty slice.
func (c *amsClientImpl) GetClustersForOrganization(orgID types.OrgID, statusFilter, statusNegativeFilter []string) (
	clusterInfoList []types.ClusterInfo,
	err error,
) {
	log.Debug().Uint32(orgIDTag, uint32(orgID)).Msg("Looking up active clusters for the organization")
	log.Debug().Uint32(orgIDTag, uint32(orgID)).Msgf("GetClustersForOrganization start. AMS client page size %v", c.pageSize)

	tStart := time.Now()

	internalOrgID, err := c.GetInternalOrgIDFromExternal(orgID)
	if err != nil {
		return
	}

	if statusNegativeFilter == nil {
		statusNegativeFilter = DefaultStatusNegativeFilters
	}

	searchQuery := generateSearchParameter(internalOrgID, statusFilter, statusNegativeFilter)
	subscriptionListRequest := c.connection.AccountsMgmt().V1().Subscriptions().List()

	clusterInfoList, err = c.executeSubscriptionListRequest(subscriptionListRequest, searchQuery)
	if err != nil {
		log.Error().Err(err).Uint32(orgIDTag, uint32(orgID)).Msg(subscriptionListRequestError)
		return
	}

	log.Info().Uint32(orgIDTag, uint32(orgID)).Msgf("GetClustersForOrganization from AMS API took %s", time.Since(tStart))
	return
}

// GetClusterDetailsFromExternalClusterID retrieves the cluster_id and display_name
// associated to a cluster using the default AMS client
func (c *amsClientImpl) GetClusterDetailsFromExternalClusterID(externalID types.ClusterName) (
	clusterInfo types.ClusterInfo,
) {
	log.Debug().Str(clusterIDTag, string(externalID)).Msg("Looking up details for the cluster")
	tStart := time.Now()

	searchQuery := fmt.Sprintf("external_cluster_id = '%s'", externalID)
	subscriptionListRequest := c.connection.AccountsMgmt().V1().Subscriptions().List()

	clusterInfoList, err := c.executeSubscriptionListRequest(subscriptionListRequest, searchQuery)
	if err != nil {
		log.Error().Err(err).Str(clusterIDTag, string(externalID)).Msg(subscriptionListRequestError)
		return
	}
	if clusterInfoList == nil {
		return
	}

	clusterInfo = clusterInfoList[0]
	log.Debug().Str(clusterIDTag, string(externalID)).Msgf("GetClusterDetailsFromExternalClusterID from AMS API took %s", time.Since(tStart))
	return
}

// GetSingleClusterInfoForOrganization retrieves the cluster_id and display_name
// associated to a cluster of the given organization using the default AMS client
func (c *amsClientImpl) GetSingleClusterInfoForOrganization(orgID types.OrgID, clusterID types.ClusterName) (
	clusterInfo types.ClusterInfo, err error,
) {
	tStart := time.Now()

	internalOrgID, err := c.GetInternalOrgIDFromExternal(orgID)
	if err != nil {
		return
	}

	searchQuery := fmt.Sprintf("organization_id = '%s' and external_cluster_id = '%s'", internalOrgID, clusterID)

	subscriptionListRequest := c.connection.AccountsMgmt().V1().Subscriptions().List()
	clusterInfoList, err := c.executeSubscriptionListRequest(subscriptionListRequest, searchQuery)
	if err != nil {
		log.Error().Err(err).Str(clusterIDTag, string(clusterID)).Msg(subscriptionListRequestError)
		return
	}
	if clusterInfoList == nil {
		return clusterInfo, &utypes.ItemNotFoundError{ItemID: clusterID}
	}

	log.Info().Str(clusterIDTag, string(clusterID)).Msgf(
		"GetSingleClusterInfoForOrganization from AMS API took %s", time.Since(tStart),
	)
	return clusterInfoList[0], nil
}

// GetMultiClusterInfoForOrganization retrieves the cluster_id and display_name
// associated to a given list of clusters of a given organization using the default AMS client
func (c *amsClientImpl) GetMultiClusterInfoForOrganization(orgID types.OrgID, clusterIDs, statusFilter, statusNegativeFilter []string) (
	clusterInfoList []types.ClusterInfo, err error,
) {
	tStart := time.Now()

	internalOrgID, err := c.GetInternalOrgIDFromExternal(orgID)
	if err != nil {
		return
	}

	if statusNegativeFilter == nil {
		statusNegativeFilter = DefaultStatusNegativeFilters
	}

	searchQuery := generateMulticlusterSearchQuery(internalOrgID, clusterIDs, statusFilter, statusNegativeFilter)

	subscriptionListRequest := c.connection.AccountsMgmt().V1().Subscriptions().List()
	clusterInfoList, err = c.executeSubscriptionListRequest(subscriptionListRequest, searchQuery)
	if err != nil {
		log.Error().Err(err).Uint32(orgIDTag, uint32(orgID)).Msg(subscriptionListRequestError)
		return
	}

	log.Info().Uint32(orgIDTag, uint32(orgID)).Int(numberOfClusterTag, len(clusterIDs)).Msgf(
		"GetMultiClusterInfoForOrganization from AMS API took %s", time.Since(tStart),
	)
	return clusterInfoList, nil
}

// GetInternalOrgIDFromExternal will retrieve the internal organization ID from an external one using AMS API
func (c *amsClientImpl) GetInternalOrgIDFromExternal(orgID types.OrgID) (string, error) {
	log.Debug().Uint32(orgIDTag, uint32(orgID)).Msg(
		"Looking for the internal organization ID for an external one",
	)
	orgsListRequest := c.connection.AccountsMgmt().V1().Organizations().List()
	response, err := orgsListRequest.
		Search(fmt.Sprintf("external_id = %d", orgID)).
		Fields("id,external_id").
		Send()

	if err != nil {
		log.Error().Err(err).Msg(orgIDRequestFailure)
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

func (c *amsClientImpl) executeSubscriptionListRequest(
	subscriptionListRequest *accMgmt.SubscriptionsListRequest,
	searchQuery string,
) (
	clusterInfoList []types.ClusterInfo,
	err error,
) {
	uniqueClusterMap := make(map[string]struct{})

	for pageNum := 1; ; pageNum++ {
		var err error
		subscriptionListRequest = subscriptionListRequest.
			Size(c.pageSize).
			Page(pageNum).
			Fields("external_cluster_id,display_name,cluster_id,managed,status").
			Search(searchQuery)

		response, err := subscriptionListRequest.Send()

		if err != nil {
			return clusterInfoList, err
		}

		// When an empty page is returned, then exit the loop
		if response.Size() == 0 {
			break
		}

		for _, item := range response.Items().Slice() {
			clusterIDstr, ok := item.GetExternalClusterID()
			// we could exclude empty external_cluster_id in the query, but we want to log these special clusters
			if !ok || clusterIDstr == "" {
				if id, ok := item.GetID(); ok {
					log.Warn().Str("InternalClusterID", id).Msg("cluster has no external ID")
				} else {
					log.Error().Msgf("No external or internal cluster ID. Cluster [%v]", item)
				}

				continue
			}

			if _, err := uuid.Parse(clusterIDstr); err != nil {
				log.Error().Str(clusterIDTag, clusterIDstr).Msg("Invalid cluster UUID")
				continue
			}

			// check for duplicates; add to unique struct
			if _, exists := uniqueClusterMap[clusterIDstr]; exists {
				continue
			}
			uniqueClusterMap[clusterIDstr] = struct{}{}

			displayName, ok := item.GetDisplayName()
			if !ok {
				displayName = clusterIDstr
			}

			managed, ok := item.GetManaged()
			if !ok {
				log.Warn().Str(clusterIDTag, clusterIDstr).Msg("cluster has no managed attribute")
			}

			status, ok := item.GetStatus()
			if !ok {
				log.Warn().Str(clusterIDTag, clusterIDstr).Msg("cannot retrieve status of cluster")
			}

			clusterID := types.ClusterName(clusterIDstr)
			clusterInfoList = append(clusterInfoList, types.ClusterInfo{
				ID:          clusterID,
				DisplayName: displayName,
				Managed:     managed,
				Status:      status,
			})
		}
	}

	return
}
