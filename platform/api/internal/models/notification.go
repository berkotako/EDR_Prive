// Notification Channel Models

package models

import "time"

// NotificationChannel represents a configured notification channel
type NotificationChannel struct {
	ID          string                 `json:"id"`
	LicenseID   string                 `json:"license_id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // email, slack, pagerduty, webhook
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// CreateChannelRequest is the request body for creating a notification channel
type CreateChannelRequest struct {
	LicenseID   string                 `json:"license_id" binding:"required"`
	Name        string                 `json:"name" binding:"required"`
	Type        string                 `json:"type" binding:"required"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config" binding:"required"`
}

// UpdateChannelRequest is the request body for updating a notification channel
type UpdateChannelRequest struct {
	Name    *string                 `json:"name"`
	Enabled *bool                   `json:"enabled"`
	Config  *map[string]interface{} `json:"config"`
}

// SendNotificationRequest is the request to send a notification
type SendNotificationRequest struct {
	ChannelID string                 `json:"channel_id" binding:"required"`
	Subject   string                 `json:"subject" binding:"required"`
	Message   string                 `json:"message" binding:"required"`
	Priority  string                 `json:"priority"` // low, medium, high, critical
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationLog represents a sent notification for audit purposes
type NotificationLog struct {
	ID          string                 `json:"id"`
	ChannelID   string                 `json:"channel_id"`
	ChannelType string                 `json:"channel_type"`
	Subject     string                 `json:"subject"`
	Message     string                 `json:"message"`
	Priority    string                 `json:"priority"`
	Status      string                 `json:"status"` // sent, failed, pending
	Error       string                 `json:"error,omitempty"`
	SentAt      time.Time              `json:"sent_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// EmailConfig represents email channel configuration
type EmailConfig struct {
	SMTPHost     string   `json:"smtp_host"`
	SMTPPort     int      `json:"smtp_port"`
	Username     string   `json:"username"`
	Password     string   `json:"password"`
	FromAddress  string   `json:"from_address"`
	FromName     string   `json:"from_name"`
	Recipients   []string `json:"recipients"`
	UseTLS       bool     `json:"use_tls"`
}

// SlackConfig represents Slack webhook configuration
type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel,omitempty"`
	Username   string `json:"username,omitempty"`
	IconEmoji  string `json:"icon_emoji,omitempty"`
}

// PagerDutyConfig represents PagerDuty integration configuration
type PagerDutyConfig struct {
	IntegrationKey string `json:"integration_key"`
	RoutingKey     string `json:"routing_key,omitempty"`
}

// WebhookConfig represents custom webhook configuration
type WebhookConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"` // POST, PUT
	Headers map[string]string `json:"headers,omitempty"`
	Timeout int               `json:"timeout"` // seconds
}

// TestChannelRequest is used to test a notification channel
type TestChannelRequest struct {
	ChannelID string `json:"channel_id" binding:"required"`
}

// TestChannelResponse returns the result of a channel test
type TestChannelResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	Error     string    `json:"error,omitempty"`
	TestedAt  time.Time `json:"tested_at"`
	LatencyMs int64     `json:"latency_ms"`
}
