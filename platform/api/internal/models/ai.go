// AI-Powered Threat Analysis Models

package models

import "time"

// AIProvider represents the LLM provider
type AIProvider string

const (
	ProviderOpenAI    AIProvider = "openai"
	ProviderAnthropic AIProvider = "anthropic"
	ProviderLocal     AIProvider = "local"
)

// AnalysisType represents the type of AI analysis
type AnalysisType string

const (
	AnalysisIncidentSummary   AnalysisType = "incident_summary"
	AnalysisAttackChain       AnalysisType = "attack_chain"
	AnalysisThreatReport      AnalysisType = "threat_report"
	AnalysisRemediationPlan   AnalysisType = "remediation_plan"
	AnalysisRootCause         AnalysisType = "root_cause"
	AnalysisRiskAssessment    AnalysisType = "risk_assessment"
	AnalysisTrendAnalysis     AnalysisType = "trend_analysis"
)

// GenerateSummaryRequest requests AI analysis of security events
type GenerateSummaryRequest struct {
	TenantID      string                 `json:"tenant_id" binding:"required"`
	EventIDs      []string               `json:"event_ids,omitempty"`
	AlertRuleID   string                 `json:"alert_rule_id,omitempty"`
	TimeRange     *TimeRange             `json:"time_range,omitempty"`
	AnalysisType  AnalysisType           `json:"analysis_type" binding:"required"`
	Provider      AIProvider             `json:"provider,omitempty"`
	IncludeIOCs   bool                   `json:"include_iocs"`
	IncludeMITRE  bool                   `json:"include_mitre"`
	CustomPrompt  string                 `json:"custom_prompt,omitempty"`
	Context       map[string]interface{} `json:"context,omitempty"`
}

// ThreatSummary represents the AI-generated analysis
type ThreatSummary struct {
	ID               string                 `json:"id"`
	TenantID         string                 `json:"tenant_id"`
	AnalysisType     AnalysisType           `json:"analysis_type"`
	Provider         AIProvider             `json:"provider"`
	Summary          string                 `json:"summary"`
	KeyFindings      []string               `json:"key_findings"`
	AttackChain      *AttackChain           `json:"attack_chain,omitempty"`
	IOCs             *IOCExtraction         `json:"iocs,omitempty"`
	MITREMapping     []string               `json:"mitre_mapping,omitempty"`
	RemediationSteps []RemediationStep      `json:"remediation_steps,omitempty"`
	RiskScore        *RiskScore             `json:"risk_score,omitempty"`
	Recommendations  []string               `json:"recommendations"`
	EventCount       int                    `json:"event_count"`
	TimeRange        TimeRange              `json:"time_range"`
	GeneratedAt      time.Time              `json:"generated_at"`
	TokensUsed       int                    `json:"tokens_used,omitempty"`
	ProcessingTimeMs int64                  `json:"processing_time_ms"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// AttackChain represents the reconstructed attack sequence
type AttackChain struct {
	InitialAccess    *ChainStep   `json:"initial_access,omitempty"`
	Execution        []ChainStep  `json:"execution,omitempty"`
	Persistence      []ChainStep  `json:"persistence,omitempty"`
	PrivilegeEsc     []ChainStep  `json:"privilege_escalation,omitempty"`
	DefenseEvasion   []ChainStep  `json:"defense_evasion,omitempty"`
	CredentialAccess []ChainStep  `json:"credential_access,omitempty"`
	Discovery        []ChainStep  `json:"discovery,omitempty"`
	LateralMovement  []ChainStep  `json:"lateral_movement,omitempty"`
	Collection       []ChainStep  `json:"collection,omitempty"`
	Exfiltration     []ChainStep  `json:"exfiltration,omitempty"`
	Impact           []ChainStep  `json:"impact,omitempty"`
	Timeline         []ChainStep  `json:"timeline"`
	Narrative        string       `json:"narrative"`
}

// ChainStep represents a step in the attack chain
type ChainStep struct {
	Timestamp       time.Time `json:"timestamp"`
	EventID         string    `json:"event_id,omitempty"`
	EventType       string    `json:"event_type"`
	Hostname        string    `json:"hostname"`
	Description     string    `json:"description"`
	MITRETechnique  string    `json:"mitre_technique,omitempty"`
	Severity        uint8     `json:"severity"`
	Indicators      []string  `json:"indicators,omitempty"`
}

// IOCExtraction represents extracted indicators of compromise
type IOCExtraction struct {
	IPAddresses      []IOC `json:"ip_addresses,omitempty"`
	Domains          []IOC `json:"domains,omitempty"`
	FileHashes       []IOC `json:"file_hashes,omitempty"`
	FilePaths        []IOC `json:"file_paths,omitempty"`
	RegistryKeys     []IOC `json:"registry_keys,omitempty"`
	ProcessNames     []IOC `json:"process_names,omitempty"`
	CommandLines     []IOC `json:"command_lines,omitempty"`
	URLs             []IOC `json:"urls,omitempty"`
	EmailAddresses   []IOC `json:"email_addresses,omitempty"`
	Usernames        []IOC `json:"usernames,omitempty"`
}

// IOC represents a single indicator of compromise
type IOC struct {
	Value       string   `json:"value"`
	Type        string   `json:"type"`
	Confidence  float64  `json:"confidence"` // 0.0 to 1.0
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	EventCount  int      `json:"event_count"`
	Context     string   `json:"context,omitempty"`
	ThreatIntel *ThreatIntelMatch `json:"threat_intel,omitempty"`
}

// ThreatIntelMatch represents a match with threat intelligence
type ThreatIntelMatch struct {
	Source      string   `json:"source"`
	ThreatActor string   `json:"threat_actor,omitempty"`
	Campaign    string   `json:"campaign,omitempty"`
	Malware     string   `json:"malware,omitempty"`
	Confidence  float64  `json:"confidence"`
	LastUpdated time.Time `json:"last_updated"`
}

// RemediationStep represents a recommended remediation action
type RemediationStep struct {
	Priority    string   `json:"priority"` // critical, high, medium, low
	Action      string   `json:"action"`
	Description string   `json:"description"`
	Commands    []string `json:"commands,omitempty"`
	Automated   bool     `json:"automated"`
	EstimatedTime string `json:"estimated_time,omitempty"`
}

// RiskScore represents the calculated risk assessment
type RiskScore struct {
	Overall        float64            `json:"overall"` // 0.0 to 10.0
	Likelihood     float64            `json:"likelihood"`
	Impact         float64            `json:"impact"`
	Urgency        string             `json:"urgency"` // immediate, high, medium, low
	Factors        []RiskFactor       `json:"factors"`
	Justification  string             `json:"justification"`
}

// RiskFactor represents a factor contributing to risk
type RiskFactor struct {
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Weight      float64 `json:"weight"`
	Score       float64 `json:"score"`
}

// AIAnalysisHistory represents stored AI analysis
type AIAnalysisHistory struct {
	ID              string       `json:"id"`
	TenantID        string       `json:"tenant_id"`
	AnalysisType    AnalysisType `json:"analysis_type"`
	Provider        AIProvider   `json:"provider"`
	Summary         string       `json:"summary"`
	EventCount      int          `json:"event_count"`
	TokensUsed      int          `json:"tokens_used"`
	CreatedAt       time.Time    `json:"created_at"`
	CreatedBy       string       `json:"created_by,omitempty"`
}

// AIConfig represents AI service configuration
type AIConfig struct {
	Provider        AIProvider `json:"provider"`
	OpenAIKey       string     `json:"openai_key,omitempty"`
	OpenAIModel     string     `json:"openai_model,omitempty"`
	AnthropicKey    string     `json:"anthropic_key,omitempty"`
	AnthropicModel  string     `json:"anthropic_model,omitempty"`
	MaxTokens       int        `json:"max_tokens"`
	Temperature     float64    `json:"temperature"`
	Enabled         bool       `json:"enabled"`
}

// GetAIConfigRequest retrieves AI configuration
type GetAIConfigRequest struct {
	LicenseID string `json:"license_id" binding:"required"`
}

// UpdateAIConfigRequest updates AI configuration
type UpdateAIConfigRequest struct {
	LicenseID      string      `json:"license_id" binding:"required"`
	Provider       *AIProvider `json:"provider"`
	OpenAIKey      *string     `json:"openai_key"`
	OpenAIModel    *string     `json:"openai_model"`
	AnthropicKey   *string     `json:"anthropic_key"`
	AnthropicModel *string     `json:"anthropic_model"`
	MaxTokens      *int        `json:"max_tokens"`
	Temperature    *float64    `json:"temperature"`
	Enabled        *bool       `json:"enabled"`
}

// ListAnalysisHistoryRequest lists past AI analyses
type ListAnalysisHistoryRequest struct {
	TenantID     string       `json:"tenant_id" binding:"required"`
	AnalysisType AnalysisType `json:"analysis_type,omitempty"`
	Limit        int          `json:"limit,omitempty"`
	Offset       int          `json:"offset,omitempty"`
}

// RegenerateAnalysisRequest regenerates a previous analysis with new context
type RegenerateAnalysisRequest struct {
	AnalysisID   string                 `json:"analysis_id" binding:"required"`
	CustomPrompt string                 `json:"custom_prompt,omitempty"`
	Context      map[string]interface{} `json:"context,omitempty"`
}
