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
	URL       string `json:"url"`
}

// OperatorCondition data structure representing a single operator condition
type OperatorCondition struct {
	Name      string `json:"name"`
	Condition string `json:"condition"`
	Reason    string `json:"reason"`
	URL       string `json:"url"`
}

// UpgradeRisksPredictors data structure to store the predictors returned by the data engineering service
type UpgradeRisksPredictors struct {
	Alerts             []Alert             `json:"alerts"`
	OperatorConditions []OperatorCondition `json:"operator_conditions"`
}

// DataEngResponse is the response received from the data-eng service
type DataEngResponse struct {
	Recommended     bool                   `json:"upgrade_recommended"`
	RisksPredictors UpgradeRisksPredictors `json:"upgrade_risks_predictors"`
	LastCheckedAt   Timestamp              `json:"last_checked_at"`
}

// UpgradeRecommendation is the inner data structure for the UpgradeRiskPredictionResponse
type UpgradeRecommendation struct {
	Recommended     bool                   `json:"upgrade_recommended"`
	RisksPredictors UpgradeRisksPredictors `json:"upgrade_risks_predictors"`
}

// UpgradeRisksMeta is a data structure to store metainformation regarding the prediction
type UpgradeRisksMeta struct {
	LastCheckedAt Timestamp `json:"last_checked_at"`
}

// UpgradeRisksPrediction is a data structure to store the prediction status for a cluster, and its
// recommendation and predictors, if any
type UpgradeRisksPrediction struct {
	ClusterID        ClusterName             `json:"cluster_id"`
	PredictionStatus string                  `json:"prediction_status"`
	Recommended      *bool                   `json:"upgrade_recommended,omitempty"`
	RisksPredictors  *UpgradeRisksPredictors `json:"upgrade_risks_predictors,omitempty"`
	LastCheckedAt    *Timestamp              `json:"last_checked_at,omitempty"`
}

// UpgradeRisksRecommendations is the main response structure for the multicluster URP endpoint
type UpgradeRisksRecommendations struct {
	Status      string                   `json:"status"`
	Predictions []UpgradeRisksPrediction `json:"predictions"`
}
