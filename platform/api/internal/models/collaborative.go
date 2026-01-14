// Collaborative Threat Hunting Models

package models

import "time"

// SharedRule represents a community-shared detection rule
type SharedRule struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	RuleType        string                 `json:"rule_type"` // yara, sigma, custom_query, alert_rule
	Content         string                 `json:"content"`
	Metadata        map[string]interface{} `json:"metadata"`
	MITRETactics    []string               `json:"mitre_tactics,omitempty"`
	MITRETechniques []string               `json:"mitre_techniques,omitempty"`
	Tags            []string               `json:"tags,omitempty"`
	Author          string                 `json:"author"` // Anonymized or username
	Organization    string                 `json:"organization,omitempty"` // Optional, anonymized
	SubmittedAt     time.Time              `json:"submitted_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	UpvoteCount     int                    `json:"upvote_count"`
	DownvoteCount   int                    `json:"downvote_count"`
	DownloadCount   int                    `json:"download_count"`
	CommentCount    int                    `json:"comment_count"`
	FalsePositiveRate *float64             `json:"false_positive_rate,omitempty"`
	EffectivenessScore *float64            `json:"effectiveness_score,omitempty"`
	Status          string                 `json:"status"` // pending, approved, rejected
	IsVerified      bool                   `json:"is_verified"` // Verified by community or admins
}

// PublishRuleRequest is the request to publish a rule to the community
type PublishRuleRequest struct {
	Name            string                 `json:"name" binding:"required"`
	Description     string                 `json:"description" binding:"required"`
	RuleType        string                 `json:"rule_type" binding:"required"`
	Content         string                 `json:"content" binding:"required"`
	Metadata        map[string]interface{} `json:"metadata"`
	MITRETactics    []string               `json:"mitre_tactics"`
	MITRETechniques []string               `json:"mitre_techniques"`
	Tags            []string               `json:"tags"`
	Anonymous       bool                   `json:"anonymous"`
	LicenseID       string                 `json:"license_id" binding:"required"`
}

// SearchRulesRequest searches for shared rules
type SearchRulesRequest struct {
	Query           string   `json:"query,omitempty"`
	RuleType        string   `json:"rule_type,omitempty"`
	MITRETactics    []string `json:"mitre_tactics,omitempty"`
	MITRETechniques []string `json:"mitre_techniques,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	MinUpvotes      int      `json:"min_upvotes,omitempty"`
	VerifiedOnly    bool     `json:"verified_only"`
	SortBy          string   `json:"sort_by,omitempty"` // popular, recent, effectiveness
	Limit           int      `json:"limit,omitempty"`
	Offset          int      `json:"offset,omitempty"`
}

// RuleVoteRequest represents a vote on a rule
type RuleVoteRequest struct {
	RuleID    string `json:"rule_id" binding:"required"`
	LicenseID string `json:"license_id" binding:"required"`
	VoteType  string `json:"vote_type" binding:"required"` // upvote, downvote
}

// RuleCommentRequest adds a comment to a rule
type RuleCommentRequest struct {
	RuleID    string `json:"rule_id" binding:"required"`
	LicenseID string `json:"license_id" binding:"required"`
	Comment   string `json:"comment" binding:"required"`
	Anonymous bool   `json:"anonymous"`
}

// RuleComment represents a comment on a shared rule
type RuleComment struct {
	ID          string    `json:"id"`
	RuleID      string    `json:"rule_id"`
	Author      string    `json:"author"`
	Comment     string    `json:"comment"`
	CreatedAt   time.Time `json:"created_at"`
	UpvoteCount int       `json:"upvote_count"`
}

// DownloadRuleRequest downloads a rule for local use
type DownloadRuleRequest struct {
	RuleID    string `json:"rule_id" binding:"required"`
	LicenseID string `json:"license_id" binding:"required"`
}

// ReportRuleRequest reports a rule for review
type ReportRuleRequest struct {
	RuleID    string `json:"rule_id" binding:"required"`
	LicenseID string `json:"license_id" binding:"required"`
	Reason    string `json:"reason" binding:"required"`
	Details   string `json:"details"`
}

// SharedIOC represents a community-shared indicator of compromise
type SharedIOC struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"` // ip, domain, hash, email, url
	Value         string    `json:"value"`
	Description   string    `json:"description"`
	ThreatType    string    `json:"threat_type,omitempty"` // malware, phishing, c2, etc
	Confidence    float64   `json:"confidence"` // 0.0 to 1.0
	Tags          []string  `json:"tags,omitempty"`
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
	SubmittedBy   string    `json:"submitted_by"` // Anonymized
	SubmittedAt   time.Time `json:"submitted_at"`
	ReportCount   int       `json:"report_count"` // Number of orgs reporting this IOC
	IsVerified    bool      `json:"is_verified"`
}

// PublishIOCRequest publishes an IOC to the community
type PublishIOCRequest struct {
	Type        string   `json:"type" binding:"required"`
	Value       string   `json:"value" binding:"required"`
	Description string   `json:"description"`
	ThreatType  string   `json:"threat_type"`
	Confidence  float64  `json:"confidence"`
	Tags        []string `json:"tags"`
	LicenseID   string   `json:"license_id" binding:"required"`
	Anonymous   bool     `json:"anonymous"`
}

// SearchIOCsRequest searches for shared IOCs
type SearchIOCsRequest struct {
	Query       string   `json:"query,omitempty"`
	Type        string   `json:"type,omitempty"`
	ThreatType  string   `json:"threat_type,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	MinConfidence float64 `json:"min_confidence,omitempty"`
	VerifiedOnly bool    `json:"verified_only"`
	Limit        int     `json:"limit,omitempty"`
	Offset       int     `json:"offset,omitempty"`
}

// HuntingQuery represents a saved threat hunting query
type HuntingQuery struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Query           string                 `json:"query"`
	QueryLanguage   string                 `json:"query_language"` // kql, spl, sql, custom
	Category        string                 `json:"category"` // lateral_movement, data_exfil, etc
	MITRETechniques []string               `json:"mitre_techniques,omitempty"`
	Tags            []string               `json:"tags,omitempty"`
	Author          string                 `json:"author"`
	SubmittedAt     time.Time              `json:"submitted_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	UseCount        int                    `json:"use_count"`
	Rating          float64                `json:"rating"`
	RatingCount     int                    `json:"rating_count"`
	IsPublic        bool                   `json:"is_public"`
}

// PublishQueryRequest publishes a hunting query
type PublishQueryRequest struct {
	Name            string   `json:"name" binding:"required"`
	Description     string   `json:"description" binding:"required"`
	Query           string   `json:"query" binding:"required"`
	QueryLanguage   string   `json:"query_language" binding:"required"`
	Category        string   `json:"category"`
	MITRETechniques []string `json:"mitre_techniques"`
	Tags            []string `json:"tags"`
	LicenseID       string   `json:"license_id" binding:"required"`
	Anonymous       bool     `json:"anonymous"`
}

// CommunityStats represents collaborative hunting statistics
type CommunityStats struct {
	TotalRules       int     `json:"total_rules"`
	TotalIOCs        int     `json:"total_iocs"`
	TotalQueries     int     `json:"total_queries"`
	TotalContributors int    `json:"total_contributors"`
	RulesByType      map[string]int `json:"rules_by_type"`
	IOCsByType       map[string]int `json:"iocs_by_type"`
	TopContributors  []ContributorStat `json:"top_contributors"`
	RecentActivity   []ActivityItem    `json:"recent_activity"`
}

// ContributorStat represents contributor statistics
type ContributorStat struct {
	Author          string `json:"author"`
	RuleCount       int    `json:"rule_count"`
	IOCCount        int    `json:"ioc_count"`
	TotalUpvotes    int    `json:"total_upvotes"`
	ReputationScore int    `json:"reputation_score"`
}

// ActivityItem represents recent community activity
type ActivityItem struct {
	Type        string    `json:"type"` // rule_published, ioc_shared, query_shared
	ItemID      string    `json:"item_id"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Timestamp   time.Time `json:"timestamp"`
}
