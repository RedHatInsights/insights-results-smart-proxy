package types

import "github.com/RedHatInsights/insights-results-aggregator-utils/types"

// UserID represents type for user id
type UserID = types.UserID

// ReportResponseMeta contains metadata about the report
type ReportResponseMeta = types.ReportResponseMeta

// Timestamp represents any timestamp in a form gathered from database
type Timestamp = types.Timestamp

// RuleWithContentResponse represents a single rule in the response of /report endpoint
type RuleWithContentResponse struct {
	RuleID       types.RuleID   `json:"rule_id"`
	ErrorKey     types.ErrorKey `json:"-"`
	CreatedAt    string         `json:"created_at"`
	Description  string         `json:"description"`
	Generic      string         `json:"details"`
	Reason       string         `json:"reason"`
	Resolution   string         `json:"resolution"`
	TotalRisk    int            `json:"total_risk"`
	RiskOfChange int            `json:"risk_of_change"`
	Disabled     bool           `json:"disabled"`
	UserVote     types.UserVote `json:"user_vote"`
	TemplateData interface{}    `json:"extra_data"`
	Tags         []string       `json:"tags"`
}

// ReportResponse represents the response of /report endpoint
type SmartProxyReport struct {
	Meta types.ReportResponseMeta  `json:"meta"`
	Data []RuleWithContentResponse `json:"data"`
}
