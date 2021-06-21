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

	"github.com/RedHatInsights/insights-operator-utils/types"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
)

const (
	ClusterName1 = "00000000-bbbb-cccc-dddd-eeeeeeeeeeee"
	ClusterName2 = "11111111-bbbb-cccc-dddd-eeeeeeeeeeee"
	ClusterName3 = "22222222-bbbb-cccc-dddd-eeeeeeeeeeee"
	GeneratedAt  = "2020-03-06T12:00:00Z"
)

var (
	// ClusterIDListInReq represent the unmarshalled body for a request
	// with a cluster list
	ClusterIDListInReq = types.ClusterListInRequest{
		Clusters: []string{
			ClusterName1,
			ClusterName2,
			ClusterName3,
		},
	}

	// ClusterIDInURL is the comma separated version of the ClusterIDListInReq
	ClusterIDInURL = strings.Join(ClusterIDListInReq.Clusters, ",")

	ReportCluster1 = types.ReportRules{
		HitRules: []types.RuleOnReport{
			testdata.RuleOnReport1,
			testdata.RuleOnReport2,
		},
	}

	// AggregatorReportForClusterList
	AggregatorReportForClusterList = types.ClusterReports{
		ClusterList: []types.ClusterName{
			types.ClusterName(ClusterName1),
			types.ClusterName(ClusterName2),
			types.ClusterName(ClusterName3),
		},
		Errors: []types.ClusterName{},
		Reports: map[types.ClusterName]json.RawMessage{
			ClusterName1: whateverToJSONRawMessage(ReportCluster1),
			ClusterName2: json.RawMessage([]byte("{}")),
			ClusterName3: json.RawMessage([]byte("{}")),
		},
		GeneratedAt: GeneratedAt,
		Status:      "ok",
	}
)

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
