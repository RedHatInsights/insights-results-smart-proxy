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
			"upgrade_risks_predictors": null
		}`

	UpgradeNotRecommended = `
		{
			"upgrade_recommended": false,
			"upgrade_risks_predictors": {
				"alerts": ["` + AlertExample1 + `"],
				"operator_conditions": ["` + OperatorConditionExample1 + `"]
			}
		}
	`

	AlertExample1             = "alert1"
	OperatorConditionExample1 = "foc1"
)
