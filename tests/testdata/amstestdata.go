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

package testdata

import (
	"fmt"

	sptypes "github.com/RedHatInsights/insights-results-smart-proxy/types"
	types "github.com/RedHatInsights/insights-results-types"
)

const (
	// InternalOrgID represents the internal org id
	InternalOrgID string = "1NKVU4otCIulgoMtgtyA6wajxkQ"
	// InternalOrgID2 represents the internal org id
	InternalOrgID2 string = "1MCBA1vtCIulgoMtjtyE1wapzxR"

	// ExternalOrgID represents an external org id
	ExternalOrgID types.OrgID = 1234

	// InternalClusterName1 represents the AMS internal name for ClusterName1
	InternalClusterName1 string = "1ABCD2abEFcdefGhijkH3lmnopI"
	// InternalClusterName2 represents the AMS internal name for ClusterName2
	InternalClusterName2 string = "9ZYXW8zyVUxwvuTtsrqS7ponmlR"

	// ActiveStatus default status for testing AMS clusters
	ActiveStatus = "Active"
)

var (

	// OrganizationResponse contains a valid response from AMS with just 1 organization
	OrganizationResponse map[string]interface{} = map[string]interface{}{
		"kind":  "OrganizationList",
		"page":  1,
		"size":  1,
		"total": 1,
		"items": []map[string]interface{}{
			{
				"external_id": fmt.Sprint(ExternalOrgID),
				"id":          InternalOrgID,
			},
		},
	}

	// OrganizationResponse2IDs contains a correct response, but with 2 orgs, which might happen temporarily during
	// ownership or membership transfer.
	OrganizationResponse2IDs map[string]interface{} = map[string]interface{}{
		"kind":  "OrganizationList",
		"page":  1,
		"size":  2,
		"total": 2,
		"items": []map[string]interface{}{
			{
				"external_id": fmt.Sprint(ExternalOrgID),
				"id":          InternalOrgID,
			},
			{
				"external_id": fmt.Sprint(ExternalOrgID),
				"id":          InternalOrgID2,
			},
		},
	}

	// OrganizationResponseNoID contains a correct response, but no internal org matching the external one.
	// Might happen as a temprary state before OCM gets the information about a newly created account, but our API
	// shouldn't return 5xx as it's ultimately fixable by the user or external systems.
	OrganizationResponseNoID map[string]interface{} = map[string]interface{}{
		"kind":  "OrganizationList",
		"page":  0,
		"size":  0,
		"total": 0,
		"items": []map[string]interface{}{},
	}

	// SubscriptionsResponse contains a valid response for subscription from AMS, 2 clusters
	SubscriptionsResponse map[string]interface{} = map[string]interface{}{
		"kind":  "SubscriptionList",
		"page":  1,
		"size":  2,
		"total": 2,
		"items": []map[string]interface{}{
			{
				"display_name":        ClusterDisplayName1,
				"external_cluster_id": ClusterName1,
				"id":                  "1YfQ9bR7LTDz24YzfFmaCdeB0sS",
				"managed":             true,
				"status":              ActiveStatus,
			},
			{
				"display_name":        ClusterDisplayName2,
				"external_cluster_id": ClusterName2,
				"id":                  "1YfQLCOCZZOEXgOp8uIbqe5i5z2",
				"managed":             false,
				"status":              ActiveStatus,
			},
		},
	}

	// SubscriptionsResponseInvalidUUID contains a valid response for subscription from AMS, 2 clusters, one with invalid UUID
	SubscriptionsResponseInvalidUUID map[string]interface{} = map[string]interface{}{
		"kind":  "SubscriptionList",
		"page":  1,
		"size":  2,
		"total": 2,
		"items": []map[string]interface{}{
			{
				"display_name":        ClusterDisplayName1,
				"external_cluster_id": ClusterName1,
				"id":                  "1YfQ9bR7LTDz24YzfFmaCdeB0sS",
				"managed":             true,
				"status":              ActiveStatus,
			},
			{
				"display_name":        "",
				"external_cluster_id": "not-uuid",
				"id":                  "1YfQLCOCZZOEXgOp8uIbqe5i5z2",
				"managed":             false,
				"status":              ActiveStatus,
			},
		},
	}
	// SubscriptionsResponseEmptyClusterIDs contains a valid response for subscription from AMS, 3 clusters,
	// but 2 of them are expected to be filtered out, even if they have display name and internal ID,
	// as they don't have the external_cluster_id
	SubscriptionsResponseEmptyClusterIDs map[string]interface{} = map[string]interface{}{
		"kind":  "SubscriptionList",
		"page":  1,
		"size":  2,
		"total": 2,
		"items": []map[string]interface{}{
			{
				"display_name":        ClusterDisplayName1,
				"external_cluster_id": ClusterName1,
				"id":                  "1YfQ9bR7LTDz24YzfFmaCdeB0sS",
				"managed":             true,
				"status":              ActiveStatus,
			},
			{
				"display_name":        ClusterDisplayName2,
				"external_cluster_id": "",
				"id":                  "1QfQ9bR7LTDz24YzfFmaCdeBf86",
				"managed":             false,
				"status":              ActiveStatus,
			},
			{
				"display_name":        "",
				"external_cluster_id": "",
				"id":                  "", // cover edge case condition
				"managed":             false,
				"status":              ActiveStatus,
			},
		},
	}

	// SubscriptionsResponseDuplicateRecords contains a valid response for subscription from AMS, as AMS API
	// can sometimes send duplicate records. Cluster UUID (external_cluster_id) is unique for us, so we must
	// exclude those records.
	SubscriptionsResponseDuplicateRecords map[string]interface{} = map[string]interface{}{
		"kind":  "SubscriptionList",
		"page":  1,
		"size":  4,
		"total": 4,
		"items": []map[string]interface{}{
			{
				"display_name":        ClusterDisplayName1,
				"external_cluster_id": ClusterName1,
				"id":                  "1YfQ9bR7LTDz24YzfFmaCdeB0sS",
				"managed":             true,
				"status":              ActiveStatus,
			},
			{
				"display_name":        ClusterDisplayName2,
				"external_cluster_id": ClusterName2,
				"id":                  "1YfQLCOCZZOEXgOp8uIbqe5i5z2",
				"managed":             false,
				"status":              ActiveStatus,
			}, {
				"display_name":        ClusterDisplayName1,
				"external_cluster_id": ClusterName1,
				"id":                  "1YfQ9bR7LTDz24YzfFmaCdeB0sS",
				"managed":             true,
				"status":              ActiveStatus,
			},
			{
				"display_name":        ClusterDisplayName2,
				"external_cluster_id": ClusterName2,
				"id":                  "1YfQLCOCZZOEXgOp8uIbqe5i5z2",
				"managed":             false,
				"status":              ActiveStatus,
			},
		},
	}

	// SubscriptionEmptyResponse contains a valid response for subscription from AMS, 0 clusters
	SubscriptionEmptyResponse map[string]interface{} = map[string]interface{}{
		"kind":  "SubscriptionList",
		"page":  2,
		"size":  0,
		"total": 2,
		"items": []map[string]interface{}{},
	}

	// OKClustersForOrganization is the expected OK result of GetClustersForOrganization
	OKClustersForOrganization []sptypes.ClusterInfo = []sptypes.ClusterInfo{
		{
			ID:          ClusterName1,
			DisplayName: ClusterDisplayName1,
			Managed:     true,
			Status:      ActiveStatus,
		},
		{
			ID:          ClusterName2,
			DisplayName: ClusterDisplayName2,
			Managed:     false,
			Status:      ActiveStatus,
		},
	}
)
