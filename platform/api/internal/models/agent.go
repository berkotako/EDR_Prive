// Agent Management Models

package models

import "time"

// Agent represents a deployed EDR agent
type Agent struct {
	ID            string                 `json:"id"`
	AgentID       string                 `json:"agent_id"`
	LicenseID     string                 `json:"license_id"`
	Hostname      string                 `json:"hostname"`
	IPAddress     string                 `json:"ip_address,omitempty"`
	OSType        string                 `json:"os_type,omitempty"`
	OSVersion     string                 `json:"os_version,omitempty"`
	AgentVersion  string                 `json:"agent_version,omitempty"`
	Status        string                 `json:"status"` // active, inactive, offline, error
	LastSeen      *time.Time             `json:"last_seen,omitempty"`
	CPUUsage      *float64               `json:"cpu_usage,omitempty"`
	MemoryUsageMB *int                   `json:"memory_usage_mb,omitempty"`
	EventsSent    int64                  `json:"events_sent"`
	Config        map[string]interface{} `json:"config,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// AgentRegistrationRequest is sent when an agent first registers
type AgentRegistrationRequest struct {
	AgentID       string `json:"agent_id" binding:"required"`
	LicenseKey    string `json:"license_key" binding:"required"`
	Hostname      string `json:"hostname" binding:"required"`
	IPAddress     string `json:"ip_address"`
	OSType        string `json:"os_type" binding:"required"`
	OSVersion     string `json:"os_version"`
	AgentVersion  string `json:"agent_version" binding:"required"`
}

// UpdateAgentRequest updates agent metadata
type UpdateAgentRequest struct {
	Hostname      *string  `json:"hostname"`
	IPAddress     *string  `json:"ip_address"`
	OSVersion     *string  `json:"os_version"`
	AgentVersion  *string  `json:"agent_version"`
	Status        *string  `json:"status"`
	CPUUsage      *float64 `json:"cpu_usage"`
	MemoryUsageMB *int     `json:"memory_usage_mb"`
}

// UpdateAgentConfigRequest updates agent configuration
type UpdateAgentConfigRequest struct {
	Config map[string]interface{} `json:"config" binding:"required"`
}

// AgentHeartbeat is sent periodically by agents
type AgentHeartbeat struct {
	AgentID       string   `json:"agent_id" binding:"required"`
	CPUUsage      float64  `json:"cpu_usage"`
	MemoryUsageMB int      `json:"memory_usage_mb"`
	EventsSent    int64    `json:"events_sent"`
	Status        string   `json:"status"`
}

// AgentHealthResponse provides health metrics
type AgentHealthResponse struct {
	AgentID       string     `json:"agent_id"`
	Status        string     `json:"status"`
	LastSeen      *time.Time `json:"last_seen"`
	CPUUsage      *float64   `json:"cpu_usage"`
	MemoryUsageMB *int       `json:"memory_usage_mb"`
	Uptime        int64      `json:"uptime_seconds"`
	IsHealthy     bool       `json:"is_healthy"`
	Issues        []string   `json:"issues,omitempty"`
}

// AgentListResponse wraps agent list with pagination
type AgentListResponse struct {
	Agents []Agent `json:"agents"`
	Total  int     `json:"total"`
	Page   int     `json:"page"`
	Limit  int     `json:"limit"`
}
