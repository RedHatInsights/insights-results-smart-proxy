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

	"github.com/RedHatInsights/insights-operator-utils/types"
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

	// OrganizationResponse2IDs contains a correct response, but with 2 orgs, which should not happen
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
			},
			{
				"display_name":        ClusterDisplayName2,
				"external_cluster_id": ClusterName2,
				"id":                  "1YfQLCOCZZOEXgOp8uIbqe5i5z2",
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
)
