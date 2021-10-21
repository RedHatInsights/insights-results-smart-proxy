// Copyright 2020 Red Hat, Inc
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

// Package types contains all user-defined data types used in Smart Proxy REST
// API service. Some types are the same as in other services (especially in
// Insights Results Aggregator, Content Service) and thus they are imported
// (based on) the common package insights-operator-utils/types.
package types

import (
	"time"

	"github.com/RedHatInsights/insights-operator-utils/types"
)

// UserID represents type for user id
type UserID = types.UserID

// ReportResponseMeta contains metadata about the report
type ReportResponseMeta = types.ReportResponseMeta

// Timestamp represents any timestamp in a form gathered from database
type Timestamp = types.Timestamp

// RuleContent is a rename for types.RuleContent
type RuleContent = types.RuleContent

// RuleID is a rename for types.RuleID
type RuleID = types.RuleID

// ImpactingFlag controls the behaviour of 'impacting' param on GET /rule/
type ImpactingFlag int

// RuleWithContentResponse represents a single rule in the response of /report endpoint
type RuleWithContentResponse struct {
	RuleID          types.RuleID    `json:"rule_id"`
	ErrorKey        types.ErrorKey  `json:"-"`
	CreatedAt       string          `json:"created_at"`
	Description     string          `json:"description"`
	Generic         string          `json:"details"`
	Reason          string          `json:"reason"`
	Resolution      string          `json:"resolution"`
	MoreInfo        string          `json:"more_info"`
	TotalRisk       int             `json:"total_risk"`
	RiskOfChange    int             `json:"risk_of_change"`
	Disabled        bool            `json:"disabled"`
	DisableFeedback string          `json:"disable_feedback"`
	DisabledAt      types.Timestamp `json:"disabled_at"`
	Internal        bool            `json:"internal"`
	UserVote        types.UserVote  `json:"user_vote"`
	TemplateData    interface{}     `json:"extra_data"`
	Tags            []string        `json:"tags"`
}

// RecommendationWithContent is a rule content struct used for Insights Advisor
// (because of other Advisor-related functionality, such as rule acknowledging, rating, etc)
type RecommendationWithContent struct {
	// RuleID in rule.module|ERROR_KEY format
	RuleID       types.RuleID   `json:"rule_id"`
	Description  string         `json:"description"`
	Generic      string         `json:"generic"`
	Reason       string         `json:"reason"`
	Resolution   string         `json:"resolution"`
	MoreInfo     string         `json:"more_info"`
	TotalRisk    uint8          `json:"total_risk"`
	RiskOfChange uint8          `json:"risk_of_change"`
	Impact       uint8          `json:"impact"`
	Likelihood   uint8          `json:"likelihood"`
	PublishDate  time.Time      `json:"publish_date"`
	RuleStatus   string         `json:"rule_status"`
	Rating       types.UserVote `json:"rating"`
	AckedCount   uint32         `json:"hosts_acked_count"`
	Tags         []string       `json:"tags"`
}

// SmartProxyReport represents the response of /report endpoint for smart proxy
type SmartProxyReport struct {
	Meta types.ReportResponseMeta  `json:"meta"`
	Data []RuleWithContentResponse `json:"data"`
}

// UserVote is a type for user's vote
type UserVote = types.UserVote

// ClusterOverview type for handling the overview result for each cluster
type ClusterOverview struct {
	TotalRisksHit []int
	TagsHit       []string
}

// OrgOverviewResponse serves as a the API response for /org_overview endpoint
type OrgOverviewResponse struct {
	ClustersHit            int            `json:"clusters_hit"`
	ClustersHitByTotalRisk map[int]int    `json:"hit_by_risk"`
	ClustersHitByTag       map[string]int `json:"hit_by_tag"`
}

const (
	// UserVoteDislike shows user's dislike
	UserVoteDislike = types.UserVoteDislike

	// UserVoteNone shows no vote from user
	UserVoteNone = types.UserVoteNone

	// UserVoteLike shows user's like
	UserVoteLike = types.UserVoteLike
)

// RuleWithContent structure with rule and rule content
type RuleWithContent struct {
	Module          types.RuleID   `json:"module"`
	Name            string         `json:"name"`
	Summary         string         `json:"summary"`
	Reason          string         `json:"reason"`
	Resolution      string         `json:"resolution"`
	MoreInfo        string         `json:"more_info"`
	ErrorKey        types.ErrorKey `json:"error_key"`
	Description     string         `json:"description"`
	TotalRisk       int            `json:"total_risk"`
	RiskOfChange    int            `json:"risk_of_change"`
	Impact          int            `json:"impact"`
	Likelihood      int            `json:"likelihood"`
	PublishDate     time.Time      `json:"publish_date"`
	Active          bool           `json:"active"`
	Internal        bool           `json:"internal"`
	Generic         string         `json:"generic"`
	Tags            []string       `json:"tags"`
	NotRequireAdmin bool
}

// RecommendationListView represents the API response for Advisor /rule/ related endpoints
// RuleStatus is based on acknowledgment table (enabled/disabled)
// RiskOfChange == resolution risk, currently missing from rule content
type RecommendationListView struct {
	// RuleID is in "|" format
	RuleID              types.RuleID              `json:"rule_id"`
	Description         string                    `json:"description"`
	Generic             string                    `json:"generic"`
	PublishDate         time.Time                 `json:"publish_date"`
	TotalRisk           uint8                     `json:"total_risk"`
	Impact              uint8                     `json:"impact"`
	Likelihood          uint8                     `json:"likelihood"`
	Tags                []string                  `json:"tags"`
	RuleStatus          string                    `json:"rule_status"`
	RiskOfChange        uint8                     `json:"risk_of_change"`
	ImpactedClustersCnt types.ImpactedClustersCnt `json:"impacted_clusters_count"`
}

// RuleRating structure with the rule identifier and the rating
type RuleRating = types.RuleRating

// RuleContentV1 version 1 of RuleConted provided by smart proxy
type RuleContentV1 struct {
	Plugin     types.RulePluginInfo             `json:"plugin"`
	ErrorKeys  map[string]RuleErrorKeyContentV1 `json:"error_keys"`
	Generic    string                           `json:"generic"`
	Summary    string                           `json:"summary"`
	Resolution string                           `json:"resolution"`
	MoreInfo   string                           `json:"more_info"`
	Reason     string                           `json:"reason"`
	HasReason  bool
}

// RuleErrorKeyContentV1 is in RuleContentV1
type RuleErrorKeyContentV1 struct {
	Metadata   ErrorKeyMetadataV1 `json:"metadata"`
	TotalRisk  int                `json:"total_risk"`
	Generic    string             `json:"generic"`
	Summary    string             `json:"summary"`
	Resolution string             `json:"resolution"`
	MoreInfo   string             `json:"more_info"`
	Reason     string             `json:"reason"`
	// DONTFIX has_reason until CCXDEV-5021
	HasReason bool
}

// ErrorKeyMetadataV1 is in RuleErrorKeyContentV1
type ErrorKeyMetadataV1 struct {
	Description string   `yaml:"description" json:"description"`
	Impact      string   `yaml:"impact" json:"impact"`
	Likelihood  int      `yaml:"likelihood" json:"likelihood"`
	PublishDate string   `yaml:"publish_date" json:"publish_date"`
	Status      string   `yaml:"status" json:"status"`
	Tags        []string `yaml:"tags" json:"tags"`
}

// RuleContentV2 version 2 of RuleContent provided by smart proxy
type RuleContentV2 struct {
	Plugin     types.RulePluginInfo             `json:"plugin"`
	ErrorKeys  map[string]RuleErrorKeyContentV2 `json:"error_keys"`
	Generic    string                           `json:"generic"`
	Summary    string                           `json:"summary"`
	Resolution string                           `json:"resolution"`
	MoreInfo   string                           `json:"more_info"`
	Reason     string                           `json:"reason"`
	HasReason  bool
}

// RuleErrorKeyContentV2 is in RuleContentV2
type RuleErrorKeyContentV2 struct {
	Metadata   ErrorKeyMetadataV2 `json:"metadata"`
	TotalRisk  int                `json:"total_risk"`
	Generic    string             `json:"generic"`
	Summary    string             `json:"summary"`
	Resolution string             `json:"resolution"`
	MoreInfo   string             `json:"more_info"`
	Reason     string             `json:"reason"`
	// DONTFIX has_reason until CCXDEV-5021
	HasReason bool
}

// ErrorKeyMetadataV2 is in RuleErrorKeyContentV2
type ErrorKeyMetadataV2 struct {
	Description string   `yaml:"description" json:"description"`
	Impact      int      `yaml:"impact" json:"impact"`
	Likelihood  int      `yaml:"likelihood" json:"likelihood"`
	PublishDate string   `yaml:"publish_date" json:"publish_date"`
	Status      string   `yaml:"status" json:"status"`
	Tags        []string `yaml:"tags" json:"tags"`
}
