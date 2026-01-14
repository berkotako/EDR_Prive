// DLP Policy Models

package models

import "time"

// DLPPolicy represents a data loss prevention policy
type DLPPolicy struct {
	ID               string                 `json:"id"`
	TenantID         string                 `json:"tenant_id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Severity         string                 `json:"severity"` // low, medium, high, critical
	Enabled          bool                   `json:"enabled"`
	RuleType         string                 `json:"rule_type"` // fingerprint, regex, ml
	Config           map[string]interface{} `json:"config,omitempty"`
	FingerprintCount int                    `json:"fingerprint_count"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// CreateDLPPolicyRequest is the request body for creating a policy
type CreateDLPPolicyRequest struct {
	TenantID    string                 `json:"tenant_id" binding:"required"`
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity" binding:"required"`
	Enabled     bool                   `json:"enabled"`
	RuleType    string                 `json:"rule_type" binding:"required"`
	Config      map[string]interface{} `json:"config"`
}

// UpdateDLPPolicyRequest is the request body for updating a policy
type UpdateDLPPolicyRequest struct {
	Name        *string                 `json:"name"`
	Description *string                 `json:"description"`
	Severity    *string                 `json:"severity"`
	Enabled     *bool                   `json:"enabled"`
	Config      *map[string]interface{} `json:"config"`
}

// AddFingerprintsRequest adds fingerprints to a policy
type AddFingerprintsRequest struct {
	Fingerprints []string `json:"fingerprints" binding:"required"`
	Source       string   `json:"source"` // file, text, database
}

// TestDLPPolicyRequest tests a policy against sample data
type TestDLPPolicyRequest struct {
	PolicyID string `json:"policy_id" binding:"required"`
	TestData string `json:"test_data" binding:"required"`
}

// TestDLPPolicyResponse returns test results
type TestDLPPolicyResponse struct {
	Matches        []DLPMatch `json:"matches"`
	ScanDurationMs int64      `json:"scan_duration_ms"`
	DataSizeBytes  int        `json:"data_size_bytes"`
}

// DLPMatch represents a detected sensitive data pattern
type DLPMatch struct {
	PolicyID   string  `json:"policy_id"`
	PolicyName string  `json:"policy_name"`
	Offset     int     `json:"offset"`
	Length     int     `json:"length"`
	Confidence float64 `json:"confidence"`
	MatchType  string  `json:"match_type"` // exact, partial, fuzzy
}
