// Copyright 2023 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

// DVONamespaceListResponse is a data structure that represents list of namespaces
// that is returned from REST API endpoint used for Workloads page
type DVONamespaceListResponse struct {
	Status    string     `json:"status"`
	Workloads []Workload `json:"workloads"`
}

// Workload structure represents one workload entry in list of workloads
type Workload struct {
	Cluster   Cluster   `json:"cluster"`
	Namespace Namespace `json:"namespace"`
	Metadata  Metadata  `json:"metadata"`
}

// Cluster structure contains cluster UUID and cluster name
type Cluster struct {
	UUID        string `json:"uuid"`
	DisplayName string `json:"display_name"`
}

// Namespace structure contains basic information about namespace
type Namespace struct {
	UUID     string `json:"uuid"`
	FullName string `json:"name"`
}

// Metadata structure contains basic information about workload metadata
type Metadata struct {
	Recommendations int         `json:"recommendations"`
	Objects         int         `json:"objects"`
	ReportedAt      string      `json:"reported_at"`
	LastCheckedAt   string      `json:"last_checked_at"`
	HighestSeverity int         `json:"highest_severity"`
	HitsBySeverity  map[int]int `json:"hits_by_severity"`
}

// WorkloadsForCluster structure represents workload for one selected cluster
type WorkloadsForCluster struct {
	Status          string              `json:"status"`
	Cluster         Cluster             `json:"cluster"`
	Namespace       Namespace           `json:"namespace"`
	Metadata        Metadata            `json:"metadata"`
	Recommendations []DVORecommendation `json:"recommendations"`
}

// DVORecommendation structure represents one DVO-related recommendation
type DVORecommendation struct {
	Check       string      `json:"check"`
	Description string      `json:"description"`
	Remediation string      `json:"remediation"`
	Objects     []DVOObject `json:"objects"`
}

// DVOObject structure
type DVOObject struct {
	Kind string `json:"kind"`
	UID  string `json:"uid"`
}
