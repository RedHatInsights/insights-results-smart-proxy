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

package testdata

var (
	UpgradeRecommended = `
		{
			"upgrade_recommended": true,
			"upgrade_risks_predictors": null,
			"last_checked_at": "0001-01-01T00:00:00Z"
		}`

	UpgradeNotRecommended = `
		{
			"upgrade_recommended": false,
			"upgrade_risks_predictors": {
				"alerts": [` + AlertExample1 + `],
				"operator_conditions": [` + OperatorConditionExample1 + `]
			},
			"last_checked_at": "0001-01-01T00:00:00Z"
		}
	`

	AlertExample1             = `{"name": "alert1", "namespace": "namespace1", "severity": "info", "url": "https://my-cluster.com/monitoring/alerts?orderBy=asc&sortBy=Severity&alert-name=alert1"}`
	OperatorConditionExample1 = `{"name": "foc1", "condition": "ExampleCondition", "reason": "Example reason", "url": "https://my-cluster.com/k8s/cluster/config.openshift.io~v1~ClusterOperator/foc1"}`

	UpgradeRecommendedTwoClusters = `[
		{
			"cluster_id": "34c3ecc5-624a-49a5-bab8-4fdc5e51a266",
			"prediction_status": "ok",
			"upgrade_recommended": True,
			"upgrade_risks_predictors": {
				"alerts": [],
				"operator_conditions": [],
			},
			"last_checked_at": test_date.isoformat(),
		},
		{
			"cluster_id": "2b9195d4-85d4-428f-944b-4b46f08911f8",
			"prediction_status": "ok",
			"upgrade_recommended": True,
			"upgrade_risks_predictors": {
				"alerts": [],
				"operator_conditions": [],
			},
			"last_checked_at": test_date.isoformat(),
		}
	]`

	UpgradeRecommendedTwoClustersOkOneNoData = `[
		{
			"cluster_id": "34c3ecc5-624a-49a5-bab8-4fdc5e51a266",
			"prediction_status": "ok",
			"upgrade_recommended": True,
			"upgrade_risks_predictors": {
				"alerts": [],
				"operator_conditions": [],
			},
			"last_checked_at": test_date.isoformat(),
		},
		{
			"cluster_id": "2b9195d4-85d4-428f-944b-4b46f08911f8",
			"prediction_status": "ok",
			"upgrade_recommended": True,
			"upgrade_risks_predictors": {
				"alerts": [],
				"operator_conditions": [],
			},
			"last_checked_at": test_date.isoformat(),
		},
		{
			"cluster_id": "aae0ff10-9892-4572-b77f-73eb3e39825f",
			"prediction_status": "No data for the cluster",
			"upgrade_recommended": None,
			"upgrade_risks_predictors": None,
			"last_checked_at": None,
		},
	]`
)
