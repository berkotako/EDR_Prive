// WebSocket Live Updates Models

package models

import "time"

// WSMessageType represents the type of WebSocket message
type WSMessageType string

const (
	// Event types
	WSTypeNewEvent         WSMessageType = "new_event"
	WSTypeNewAlert         WSMessageType = "new_alert"
	WSTypeAgentStatus      WSMessageType = "agent_status_change"
	WSTypeHeartbeat        WSMessageType = "heartbeat"
	WSTypePolicyUpdate     WSMessageType = "policy_update"
	WSTypeSystemNotification WSMessageType = "system_notification"

	// Control messages
	WSTypeSubscribe        WSMessageType = "subscribe"
	WSTypeUnsubscribe      WSMessageType = "unsubscribe"
	WSTypePing             WSMessageType = "ping"
	WSTypePong             WSMessageType = "pong"
	WSTypeError            WSMessageType = "error"
	WSTypeConnected        WSMessageType = "connected"
)

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type      WSMessageType      `json:"type"`
	Timestamp time.Time          `json:"timestamp"`
	Data      interface{}        `json:"data,omitempty"`
	Error     string             `json:"error,omitempty"`
}

// WSSubscription represents a client's subscription preferences
type WSSubscription struct {
	TenantID      string          `json:"tenant_id"`
	EventTypes    []string        `json:"event_types,omitempty"`     // Filter by event type
	Severities    []uint8         `json:"severities,omitempty"`      // Filter by severity
	AgentIDs      []string        `json:"agent_ids,omitempty"`       // Filter by specific agents
	Hostnames     []string        `json:"hostnames,omitempty"`       // Filter by hostname
	AlertOnly     bool            `json:"alert_only"`                // Only send alerts
}

// WSConnectRequest is sent when establishing WebSocket connection
type WSConnectRequest struct {
	TenantID string `json:"tenant_id"`
	Token    string `json:"token,omitempty"` // Auth token
}

// WSEventNotification represents a new event notification
type WSEventNotification struct {
	EventID        string    `json:"event_id"`
	EventType      string    `json:"event_type"`
	Hostname       string    `json:"hostname"`
	Severity       uint8     `json:"severity"`
	MitreTactic    string    `json:"mitre_tactic,omitempty"`
	MitreTechnique string    `json:"mitre_technique,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
	Summary        string    `json:"summary"`
}

// WSAlertNotification represents a new alert notification
type WSAlertNotification struct {
	AlertID     string    `json:"alert_id"`
	RuleName    string    `json:"rule_name"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	EventCount  int       `json:"event_count"`
	Hostname    string    `json:"hostname,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// WSAgentStatusNotification represents agent status change
type WSAgentStatusNotification struct {
	AgentID   string    `json:"agent_id"`
	Hostname  string    `json:"hostname"`
	OldStatus string    `json:"old_status"`
	NewStatus string    `json:"new_status"`
	Timestamp time.Time `json:"timestamp"`
	Reason    string    `json:"reason,omitempty"`
}

// WSStatistics represents real-time statistics update
type WSStatistics struct {
	TotalEvents       int64            `json:"total_events"`
	EventsLast24h     int64            `json:"events_last_24h"`
	EventsLastHour    int64            `json:"events_last_hour"`
	ActiveAlerts      int              `json:"active_alerts"`
	OnlineAgents      int              `json:"online_agents"`
	OfflineAgents     int              `json:"offline_agents"`
	EventsByType      map[string]int64 `json:"events_by_type"`
	EventsBySeverity  map[uint8]int64  `json:"events_by_severity"`
	Timestamp         time.Time        `json:"timestamp"`
}

// WSClient represents a connected WebSocket client
type WSClient struct {
	ID           string
	TenantID     string
	Subscription WSSubscription
	SendChan     chan WSMessage
	Connected    bool
	ConnectedAt  time.Time
	LastPingAt   time.Time
}
