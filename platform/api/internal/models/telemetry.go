// Telemetry and Event Query Models

package models

import "time"

// TelemetryEvent represents a security event from the ClickHouse database
type TelemetryEvent struct {
	EventID          string                 `json:"event_id"`
	AgentID          string                 `json:"agent_id"`
	TenantID         string                 `json:"tenant_id"`
	Timestamp        time.Time              `json:"timestamp"`
	ServerTimestamp  time.Time              `json:"server_timestamp"`
	EventType        string                 `json:"event_type"`
	MitreTactic      string                 `json:"mitre_tactic,omitempty"`
	MitreTechnique   string                 `json:"mitre_technique,omitempty"`
	Severity         uint8                  `json:"severity"`
	Hostname         string                 `json:"hostname"`
	OSType           string                 `json:"os_type,omitempty"`
	Payload          map[string]interface{} `json:"payload,omitempty"`
	ProcessName      string                 `json:"process_name,omitempty"`
	FilePath         string                 `json:"file_path,omitempty"`
	DstIP            string                 `json:"dst_ip,omitempty"`
	DstPort          uint16                 `json:"dst_port,omitempty"`
	Username         string                 `json:"username,omitempty"`
	IngestionDate    time.Time              `json:"ingestion_date"`
}

// QueryEventsRequest defines the request parameters for querying events
type QueryEventsRequest struct {
	TenantID         string   `json:"tenant_id" binding:"required"`
	StartTime        string   `json:"start_time" binding:"required"` // ISO 8601 format
	EndTime          string   `json:"end_time" binding:"required"`
	EventTypes       []string `json:"event_types,omitempty"`
	AgentIDs         []string `json:"agent_ids,omitempty"`
	Hostnames        []string `json:"hostnames,omitempty"`
	MinSeverity      *uint8   `json:"min_severity,omitempty"`
	MitreTactics     []string `json:"mitre_tactics,omitempty"`
	MitreTechniques  []string `json:"mitre_techniques,omitempty"`
	ProcessNames     []string `json:"process_names,omitempty"`
	FilePaths        []string `json:"file_paths,omitempty"`
	DstIPs           []string `json:"dst_ips,omitempty"`
	SearchText       string   `json:"search_text,omitempty"` // Full-text search in payload
	Limit            int      `json:"limit,omitempty"`
	Offset           int      `json:"offset,omitempty"`
	OrderBy          string   `json:"order_by,omitempty"` // timestamp, severity, hostname
	OrderDirection   string   `json:"order_direction,omitempty"` // asc, desc
}

// QueryEventsResponse wraps the query results with metadata
type QueryEventsResponse struct {
	Events      []TelemetryEvent `json:"events"`
	Total       int64            `json:"total"`
	Limit       int              `json:"limit"`
	Offset      int              `json:"offset"`
	QueryTimeMs int64            `json:"query_time_ms"`
}

// StatisticsRequest defines parameters for statistics queries
type StatisticsRequest struct {
	TenantID  string `json:"tenant_id" binding:"required"`
	StartTime string `json:"start_time" binding:"required"`
	EndTime   string `json:"end_time" binding:"required"`
}

// Statistics represents aggregate statistics for events
type Statistics struct {
	TotalEvents       int64                  `json:"total_events"`
	EventsByType      map[string]int64       `json:"events_by_type"`
	EventsBySeverity  map[uint8]int64        `json:"events_by_severity"`
	EventsByHost      map[string]int64       `json:"events_by_host"`
	TopMitreTactics   []MitreStat            `json:"top_mitre_tactics"`
	TopMitreTechniques []MitreStat           `json:"top_mitre_techniques"`
	UniqueAgents      int64                  `json:"unique_agents"`
	UniqueHosts       int64                  `json:"unique_hosts"`
	TimeRange         TimeRange              `json:"time_range"`
}

// MitreStat represents statistics for MITRE tactics/techniques
type MitreStat struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	EventCount  int64  `json:"event_count"`
	Percentage  float64 `json:"percentage"`
}

// TimeRange represents a time period
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// MITRETactic represents a MITRE ATT&CK tactic
type MITRETactic struct {
	TacticID    string `json:"tactic_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}

// MITRETechnique represents a MITRE ATT&CK technique
type MITRETechnique struct {
	TechniqueID string   `json:"technique_id"`
	TacticID    string   `json:"tactic_id,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Platforms   []string `json:"platforms,omitempty"`
	DataSources []string `json:"data_sources,omitempty"`
	URL         string   `json:"url,omitempty"`
}

// MITRECoverage represents detection coverage for MITRE framework
type MITRECoverage struct {
	TenantID         string                        `json:"tenant_id"`
	TotalTechniques  int                           `json:"total_techniques"`
	DetectedCount    int                           `json:"detected_count"`
	CoveragePercent  float64                       `json:"coverage_percent"`
	CoverageByTactic map[string]TacticCoverage     `json:"coverage_by_tactic"`
	DetectedTechniques []DetectedTechnique         `json:"detected_techniques"`
}

// TacticCoverage represents coverage for a specific tactic
type TacticCoverage struct {
	TacticID        string  `json:"tactic_id"`
	TacticName      string  `json:"tactic_name"`
	TotalTechniques int     `json:"total_techniques"`
	DetectedCount   int     `json:"detected_count"`
	CoveragePercent float64 `json:"coverage_percent"`
}

// DetectedTechnique represents a detected technique with event count
type DetectedTechnique struct {
	TechniqueID string `json:"technique_id"`
	TechniqueName string `json:"technique_name,omitempty"`
	EventCount  int64  `json:"event_count"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
}

// AlertRule represents an alerting rule
type AlertRule struct {
	ID          string                 `json:"id"`
	LicenseID   string                 `json:"license_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Severity    string                 `json:"severity"`
	Enabled     bool                   `json:"enabled"`
	Condition   map[string]interface{} `json:"condition"`
	Actions     []map[string]interface{} `json:"actions,omitempty"`
	CreatedBy   string                 `json:"created_by,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// CreateAlertRuleRequest is the request body for creating an alert rule
type CreateAlertRuleRequest struct {
	LicenseID   string                   `json:"license_id" binding:"required"`
	Name        string                   `json:"name" binding:"required"`
	Description string                   `json:"description"`
	Severity    string                   `json:"severity" binding:"required"`
	Enabled     bool                     `json:"enabled"`
	Condition   map[string]interface{}   `json:"condition" binding:"required"`
	Actions     []map[string]interface{} `json:"actions"`
	CreatedBy   string                   `json:"created_by"`
}

// UpdateAlertRuleRequest is the request body for updating an alert rule
type UpdateAlertRuleRequest struct {
	Name        *string                   `json:"name"`
	Description *string                   `json:"description"`
	Severity    *string                   `json:"severity"`
	Enabled     *bool                     `json:"enabled"`
	Condition   *map[string]interface{}   `json:"condition"`
	Actions     *[]map[string]interface{} `json:"actions"`
}
