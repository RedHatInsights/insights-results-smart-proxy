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

	types "github.com/RedHatInsights/insights-results-types"
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

// ClusterName is a rename for types.ClusterName
type ClusterName = types.ClusterName

// OrgID is a rename for types.OrgID
type OrgID = types.OrgID

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

// RecommendationContent is a rule content struct used for Insights Advisor,
type RecommendationContent struct {
	// RuleSelector = rule.module|ERROR_KEY format
	RuleSelector types.RuleSelector `json:"rule_id"`
	Description  string             `json:"description"`
	Generic      string             `json:"generic"`
	Reason       string             `json:"reason"`
	Resolution   string             `json:"resolution"`
	MoreInfo     string             `json:"more_info"`
	TotalRisk    uint8              `json:"total_risk"`
	RiskOfChange uint8              `json:"risk_of_change"`
	Impact       uint8              `json:"impact"`
	Likelihood   uint8              `json:"likelihood"`
	PublishDate  time.Time          `json:"publish_date"`
	Tags         []string           `json:"tags"`
}

// RecommendationContentUserData is a rule content struct with additional Insights Advisor
// related user data, such as rule acknowledging or rating, which requires access to DB/aggregator
type RecommendationContentUserData struct {
	// RuleSelector = rule.module|ERROR_KEY format
	RuleSelector types.RuleSelector `json:"rule_id"`
	Description  string             `json:"description"`
	Generic      string             `json:"generic"`
	Reason       string             `json:"reason"`
	Resolution   string             `json:"resolution"`
	MoreInfo     string             `json:"more_info"`
	TotalRisk    uint8              `json:"total_risk"`
	RiskOfChange uint8              `json:"risk_of_change"`
	Impact       uint8              `json:"impact"`
	Likelihood   uint8              `json:"likelihood"`
	PublishDate  time.Time          `json:"publish_date"`
	Tags         []string           `json:"tags"`
	Rating       types.UserVote     `json:"rating"`
	AckedCount   uint32             `json:"hosts_acked_count"`
	Disabled     bool               `json:"disabled"`
}

// ReportResponseMetaV1 contains metadata for /report endpoint in v1
type ReportResponseMetaV1 struct {
	Count         int       `json:"count"`
	LastCheckedAt Timestamp `json:"last_checked_at"`
}

// ReportResponseMetaV2 contains metadata for /report endpoint in v2
type ReportResponseMetaV2 struct {
	DisplayName   string    `json:"cluster_name"`
	Count         int       `json:"count"`
	LastCheckedAt Timestamp `json:"last_checked_at,omitempty"`
	GatheredAt    Timestamp `json:"gathered_at,omitempty"`
}

// SmartProxyReportV1 represents the response of /report (V1) endpoint for smart proxy
// This structure exists to make sure we comply with the previous API used by some clients
type SmartProxyReportV1 struct {
	Meta ReportResponseMetaV1      `json:"meta"`
	Data []RuleWithContentResponse `json:"data"`
}

// SmartProxyReportV2 represents the response of /report (V2) endpoint for smart proxy
// This structure exists to make sure we comply with the previous API used by some clients
type SmartProxyReportV2 struct {
	Meta ReportResponseMetaV2      `json:"meta"`
	Data []RuleWithContentResponse `json:"data"`
}

// SmartProxyReport represents the response of /report (V2) endpoint for smart proxy
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
	RuleID              types.RuleID `json:"rule_id"`
	Description         string       `json:"description"`
	Generic             string       `json:"generic"`
	PublishDate         time.Time    `json:"publish_date"`
	TotalRisk           uint8        `json:"total_risk"`
	Impact              uint8        `json:"impact"`
	Likelihood          uint8        `json:"likelihood"`
	Tags                []string     `json:"tags"`
	Disabled            bool         `json:"disabled"`
	RiskOfChange        uint8        `json:"risk_of_change"`
	ImpactedClustersCnt uint32       `json:"impacted_clusters_count"`
}

// ClusterListView represents a single item in the response for Clusters List view
type ClusterListView struct {
	ClusterID       types.ClusterName `json:"cluster_id"`
	ClusterName     string            `json:"cluster_name"`
	LastCheckedAt   Timestamp         `json:"last_checked_at,omitempty"`
	TotalHitCount   uint32            `json:"total_hit_count"`
	HitsByTotalRisk map[int]int       `json:"hits_by_total_risk"`
}

// RuleRating structure with the rule identifier and the rating
type RuleRating = types.RuleRating

// RuleContentV1 version 1 of RuleConted provided by smart proxy
type RuleContentV1 = types.RuleContentV1

// RuleErrorKeyContentV1 is in RuleContentV1
type RuleErrorKeyContentV1 = types.RuleErrorKeyContentV1

// ErrorKeyMetadataV1 is in RuleErrorKeyContentV1
type ErrorKeyMetadataV1 = types.ErrorKeyMetadataV1

// RuleContentV2 version 2 of RuleContent provided by smart proxy
type RuleContentV2 = types.RuleContentV2

// RuleErrorKeyContentV2 is in RuleContentV2
type RuleErrorKeyContentV2 = types.RuleErrorKeyContentV2

// ErrorKeyMetadataV2 is in RuleErrorKeyContentV2
type ErrorKeyMetadataV2 = types.ErrorKeyMetadataV2

// InfoResponse is a data structure returned by /info REST API endpoint
type InfoResponse struct {
	SmartProxy     map[string]string `json:"SmartProxy"`
	Aggregator     map[string]string `json:"Aggregator"`
	ContentService map[string]string `json:"ContentService"`
}

// ClusterInfo is a data structure containing some relevant cluster information
type ClusterInfo struct {
	ID          ClusterName
	DisplayName string
}

// ClustersDetailData is the inner data structure for /clusters_detail
type ClustersDetailData struct {
	EnabledClusters  []types.HittingClustersData `json:"enabled"`
	DisabledClusters []types.DisabledClusterInfo `json:"disabled"`
}

// ClustersDetailResponse is a data structure used as the response for /clusters_detail
type ClustersDetailResponse struct {
	Data   ClustersDetailData `json:"data"`
	Status string             `json:"status"`
}
