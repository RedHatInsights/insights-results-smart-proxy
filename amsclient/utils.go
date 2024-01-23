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

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const UUIDv4_LENGTH = 32

// generateSearchParameter generates a search string for given org_id and desired statuses
func generateSearchParameter(orgID string, allowedStatuses, disallowedStatuses []string) string {
	searchQuery := fmt.Sprintf("organization_id is '%s' and cluster_id != ''", orgID)

	if len(allowedStatuses) > 0 {
		clusterIDQuery := " and status in ('" + strings.Join(allowedStatuses, "','") + "')"
		searchQuery += clusterIDQuery
	}

	if len(disallowedStatuses) > 0 {
		clusterIDQuery := " and status not in ('" + strings.Join(disallowedStatuses, "','") + "')"
		searchQuery += clusterIDQuery
	}

	return searchQuery
}

// joinClusterNames is a helper function to avoid looping over a []types.ClusterName
// and converting all its elements to string just so we can use strings.Join
func joinClusterNames(clusterIDs []types.ClusterName, separator string) string {
	var builder strings.Builder

	// Preallocate the string using the length of UUIDv4 * (number of items + separators)
	totalLength := len(clusterIDs) * (UUIDv4_LENGTH + len(separator)) // UUIDv4 has 32 characters

	// Preallocate the string with calculated length
	builder.Grow(totalLength)

	// Append the cluster names
	for i, id := range clusterIDs {
		builder.WriteString(string(id))
		if i < len(clusterIDs)-1 {
			builder.WriteString(separator)
		}
	}

	return builder.String()
}

// generateMulticlusterSearchQuery generates a search string for given org_id, list of clusters and desired statuses
func generateMulticlusterSearchQuery(orgID string, clusterIDs []types.ClusterName, allowedStatuses, disallowedStatuses []string) string {
	searchQuery := fmt.Sprintf("organization_id is '%s'", orgID)

	if len(clusterIDs) > 0 {
		clusterIDQuery := " and cluster_id in ('" + joinClusterNames(clusterIDs, ",") + "')"
		searchQuery += clusterIDQuery
	}

	if len(allowedStatuses) > 0 {
		clusterIDQuery := " and status in ('" + strings.Join(allowedStatuses, "','") + "')"
		searchQuery += clusterIDQuery
	}

	if len(disallowedStatuses) > 0 {
		clusterIDQuery := " and status not in ('" + strings.Join(disallowedStatuses, "','") + "')"
		searchQuery += clusterIDQuery
	}

	return searchQuery
}
