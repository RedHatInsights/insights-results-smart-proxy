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

// Alert data structure representing a single alert
type Alert struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Severity  string `json:"severity"`
}

// OperatorCondition data structure representing a single operator condition
type OperatorCondition struct {
	Name      string `json:"name"`
	Condition string `json:"condition"`
	Reason    string `json:"reason"`
}

// UpgradeRiskPredictors data structure to store the predictors returned by the data engineering service
type UpgradeRiskPredictors struct {
	Alerts             []Alert             `json:"alerts"`
	OperatorConditions []OperatorCondition `json:"operator_conditions"`
}

// UpgradeRecommendation is the inner data structure for the UpgradeRiskPredictionResponse
type UpgradeRecommendation struct {
	Recommended     bool                  `json:"upgrade_recommended"`
	RisksPredictors UpgradeRiskPredictors `json:"upgrade_risks_predictors"`
}

// UpgradeRiskPredictionResponse is a data structure used as the response for UpgradeRisksPrediction endpoint
type UpgradeRiskPredictionResponse struct {
	UpgradeRecommendation UpgradeRecommendation `json:"upgrade_recommendation"`
}
