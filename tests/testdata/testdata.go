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
	"encoding/json"
	"strings"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ctypes "github.com/RedHatInsights/insights-results-types"

	"github.com/RedHatInsights/insights-results-smart-proxy/types"
)

const (
	ClusterName1 = "00000000-bbbb-cccc-dddd-eeeeeeeeeeee"
	ClusterName2 = "11111111-bbbb-cccc-dddd-eeeeeeeeeeee"
	ClusterName3 = "22222222-bbbb-cccc-dddd-eeeeeeeeeeee"

	ClusterDisplayName1 = "Cluster 1"
	ClusterDisplayName2 = "Cluster 2"
	ClusterDisplayName3 = "Cluster 3"

	OrgID = testdata.OrgID

	GeneratedAt = "2020-03-06T12:00:00Z"

	NamespaceUUID1 = "00000000-aaaa-bbbb-cccc-eeeeeeeeeeee"
)

var (
	// ClusterIDListInReq represent the unmarshalled body for a request
	// with a cluster list
	ClusterIDListInReq = ctypes.ClusterListInRequest{
		Clusters: []string{
			ClusterName1,
			ClusterName2,
			ClusterName3,
		},
	}

	// ClusterIDInURL is the comma separated version of the ClusterIDListInReq
	ClusterIDInURL = strings.Join(ClusterIDListInReq.Clusters, ",")

	ReportCluster1 = ctypes.ReportRules{
		HitRules: []ctypes.RuleOnReport{
			testdata.RuleOnReport1,
			testdata.RuleOnReport2,
		},
	}

	ReportCluster2 = ctypes.ReportRules{
		HitRules: []ctypes.RuleOnReport{
			testdata.RuleOnReport5,
			testdata.RuleOnReport2,
		},
	}

	// AggregatorReportForClusterList
	AggregatorReportForClusterList = ctypes.ClusterReports{
		ClusterList: []ctypes.ClusterName{
			ctypes.ClusterName(ClusterName1),
			ctypes.ClusterName(ClusterName2),
			ctypes.ClusterName(ClusterName3),
		},
		Errors: []ctypes.ClusterName{},
		Reports: map[ctypes.ClusterName]json.RawMessage{
			ClusterName1: whateverToJSONRawMessage(ReportCluster1),
			ClusterName2: whateverToJSONRawMessage(ReportCluster2),
			ClusterName3: json.RawMessage([]byte("{}")),
		},
		GeneratedAt: GeneratedAt,
		Status:      "ok",
	}

	ClusterInfoResult = []types.ClusterInfo{
		{
			ID:          testdata.ClusterName,
			DisplayName: ClusterDisplayName1,
		},
	}

	ClusterList1Cluster = []types.ClusterName{testdata.ClusterName}

	ClusterInfoResult2Clusters = []types.ClusterInfo{
		{
			ID:          testdata.GetRandomClusterID(),
			DisplayName: ClusterDisplayName1,
		},
		{
			ID:          testdata.GetRandomClusterID(),
			DisplayName: ClusterDisplayName2,
		},
	}

	ClusterList2Clusters = []types.ClusterName{ClusterInfoResult2Clusters[0].ID, ClusterInfoResult2Clusters[1].ID}
)

// GetRandomClusterInfo function returns a ClusterInfo with random ID
// and using the same ID as DisplayName
func GetRandomClusterInfo() types.ClusterInfo {
	clusterID := testdata.GetRandomClusterID()
	return types.ClusterInfo{
		ID:          clusterID,
		DisplayName: string(clusterID),
		Status:      ActiveStatus,
	}
}

// GetRandomClusterInfoList generates a slice of given length with random clusterInfo. Every other cluster has managed=true
func GetRandomClusterInfoList(length int) []types.ClusterInfo {
	clusterInfoList := make([]types.ClusterInfo, length)
	for i := range clusterInfoList {
		clusterInfoList[i] = GetRandomClusterInfo()
		clusterInfoList[i].Managed = i%2 == 0
	}
	return clusterInfoList
}

// GetRandomClusterInfoListAllUnManaged generates a slice of given length with random clusterInfo. Every cluster has managed=false
func GetRandomClusterInfoListAllUnManaged(length int) []types.ClusterInfo {
	clusterInfoList := make([]types.ClusterInfo, length)
	for i := range clusterInfoList {
		clusterInfoList[i] = GetRandomClusterInfo()
		clusterInfoList[i].Managed = false
	}
	return clusterInfoList
}

// GetRandomClusterInfoListAllManaged generates a slice of given length with random clusterInfo. Every cluster has managed=true
func GetRandomClusterInfoListAllManaged(length int) []types.ClusterInfo {
	clusterInfoList := make([]types.ClusterInfo, length)
	for i := range clusterInfoList {
		clusterInfoList[i] = GetRandomClusterInfo()
		clusterInfoList[i].Managed = true
	}
	return clusterInfoList
}

func whateverToJSONRawMessage(obj interface{}) json.RawMessage {
	var result json.RawMessage

	byteRep, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	if err = json.Unmarshal(byteRep, &result); err != nil {
		panic(err)
	}

	return result
}
