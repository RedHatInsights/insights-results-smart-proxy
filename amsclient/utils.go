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
)

// generateSingleClusterSearch generates a search string for given internal org IDs and desired statuses
func generateSingleClusterSearch(orgIDs []string, allowedStatuses, disallowedStatuses []string) string {
	searchQuery := fmt.Sprintf("organization_id in ('%s') and cluster_id != ''", strings.Join(orgIDs, "','"))

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
