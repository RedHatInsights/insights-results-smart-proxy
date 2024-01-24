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
	"strings"

	"github.com/RedHatInsights/insights-results-smart-proxy/types"
)

const (
	searchQuerySeparator = "','"
)

// searchQueryFor generates a search string that can be appended to an
// existing search query for the given key and values
func searchQueryFor(key string, values []string) string {
	if len(values) > 0 {
		return fmt.Sprintf(" and %s in ('%s')", key, strings.Join(values, searchQuerySeparator))
	}
	return ""
}

func searchQueryForStatuses(allowedStatuses, disallowedStatuses []string) string {
	return searchQueryFor("status", allowedStatuses) + searchQueryFor("status not", disallowedStatuses)
}

// generateSearchParameter generates a search string for given org_id and desired statuses
func generateSearchParameter(orgID string, allowedStatuses, disallowedStatuses []string) string {
	searchQuery := fmt.Sprintf("organization_id is '%s' and cluster_id != ''", orgID)
	searchQuery += searchQueryForStatuses(allowedStatuses, disallowedStatuses)
	return searchQuery
}

// generateMulticlusterSearchQuery generates a search string for given org_id, list of clusters, and desired statuses
func generateMulticlusterSearchQuery(orgID string, clusterIDs, allowedStatuses, disallowedStatuses []string) string {
	searchQuery := fmt.Sprintf("organization_id is '%s'", orgID)
	searchQuery += searchQueryFor("cluster_id", clusterIDs)
	searchQuery += searchQueryForStatuses(allowedStatuses, disallowedStatuses)
	return searchQuery
}

// FilterManagedClusters separates the given slice of ClusterInfo into two slices of
// cluster IDs depending on if they are managed or not.
func FilterManagedClusters(clusters []types.ClusterInfo) (managed []string, unmanaged []string) {
	for _, cluster := range clusters {
		if cluster.Managed {
			managed = append(managed, string(cluster.ID))
		} else {
			unmanaged = append(unmanaged, string(cluster.ID))
		}
	}

	return
}
