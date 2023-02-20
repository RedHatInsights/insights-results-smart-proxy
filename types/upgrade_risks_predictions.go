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

// UpgradeRiskPredictors data structure to store the predictors returned by the data engineering service
type UpgradeRiskPredictors struct {
	Alerts             []string `json:"alerts"`
	OperatorConditions []string `json:"operator_conditions"`
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
