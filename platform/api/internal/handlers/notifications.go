// Notification Channel Handlers with Email and Slack Integration

package handlers

import (
	"bytes"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/sentinel-enterprise/platform/api/internal/models"
)

// NotificationHandler handles notification channel management
type NotificationHandler struct {
	db *sql.DB
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(db *sql.DB) *NotificationHandler {
	return &NotificationHandler{
		db: db,
	}
}

// ListChannels retrieves all notification channels for a tenant
func (h *NotificationHandler) ListChannels(c *gin.Context) {
	licenseID := c.Query("license_id")
	if licenseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "license_id required"})
		return
	}

	query := `
		SELECT id, license_id, name, type, enabled, config, created_at, updated_at
		FROM notification_channels
		WHERE license_id = $1
		ORDER BY created_at DESC
	`

	rows, err := h.db.Query(query, licenseID)
	if err != nil {
		log.Errorf("Failed to query notification channels: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}
	defer rows.Close()

	channels := make([]models.NotificationChannel, 0)
	for rows.Next() {
		var channel models.NotificationChannel
		var configJSON []byte

		err := rows.Scan(
			&channel.ID, &channel.LicenseID, &channel.Name, &channel.Type,
			&channel.Enabled, &configJSON, &channel.CreatedAt, &channel.UpdatedAt,
		)

		if err != nil {
			log.Warnf("Failed to scan channel: %v", err)
			continue
		}

		// Parse JSON config (mask sensitive fields)
		if len(configJSON) > 0 {
			var config map[string]interface{}
			json.Unmarshal(configJSON, &config)

			// Mask sensitive fields
			if _, ok := config["password"]; ok {
				config["password"] = "********"
			}
			if _, ok := config["webhook_url"]; ok {
				config["webhook_url"] = maskWebhookURL(config["webhook_url"].(string))
			}
			if _, ok := config["integration_key"]; ok {
				config["integration_key"] = "********"
			}

			channel.Config = config
		}

		channels = append(channels, channel)
	}

	c.JSON(http.StatusOK, gin.H{
		"channels": channels,
		"total":    len(channels),
	})
}

// GetChannel retrieves a specific notification channel
func (h *NotificationHandler) GetChannel(c *gin.Context) {
	channelID := c.Param("id")

	query := `
		SELECT id, license_id, name, type, enabled, config, created_at, updated_at
		FROM notification_channels
		WHERE id = $1
	`

	var channel models.NotificationChannel
	var configJSON []byte

	err := h.db.QueryRow(query, channelID).Scan(
		&channel.ID, &channel.LicenseID, &channel.Name, &channel.Type,
		&channel.Enabled, &configJSON, &channel.CreatedAt, &channel.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
		log.Errorf("Failed to query channel: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}

	// Parse JSON config (mask sensitive fields)
	if len(configJSON) > 0 {
		var config map[string]interface{}
		json.Unmarshal(configJSON, &config)

		// Mask sensitive fields
		if _, ok := config["password"]; ok {
			config["password"] = "********"
		}
		if _, ok := config["webhook_url"]; ok {
			config["webhook_url"] = maskWebhookURL(config["webhook_url"].(string))
		}
		if _, ok := config["integration_key"]; ok {
			config["integration_key"] = "********"
		}

		channel.Config = config
	}

	c.JSON(http.StatusOK, channel)
}

// CreateChannel creates a new notification channel
func (h *NotificationHandler) CreateChannel(c *gin.Context) {
	var req models.CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate channel type
	if !isValidChannelType(req.Type) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel type. Must be: email, slack, pagerduty, or webhook"})
		return
	}

	// Validate configuration based on type
	if err := validateChannelConfig(req.Type, req.Config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	channelID := uuid.New().String()
	configJSON, _ := json.Marshal(req.Config)

	query := `
		INSERT INTO notification_channels (id, license_id, name, type, enabled, config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	var createdAt, updatedAt time.Time
	err := h.db.QueryRow(query,
		channelID, req.LicenseID, req.Name, req.Type, req.Enabled, string(configJSON),
	).Scan(&createdAt, &updatedAt)

	if err != nil {
		log.Errorf("Failed to create notification channel: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create channel"})
		return
	}

	log.Infof("Created notification channel: %s (%s)", req.Name, channelID)

	c.JSON(http.StatusCreated, gin.H{
		"id":         channelID,
		"created_at": createdAt,
		"message":    "Notification channel created successfully",
	})
}

// UpdateChannel updates a notification channel
func (h *NotificationHandler) UpdateChannel(c *gin.Context) {
	channelID := c.Param("id")

	var req models.UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build dynamic update query
	query := "UPDATE notification_channels SET updated_at = NOW()"
	args := []interface{}{}
	argCount := 1

	if req.Name != nil {
		query += fmt.Sprintf(", name = $%d", argCount)
		args = append(args, *req.Name)
		argCount++
	}
	if req.Enabled != nil {
		query += fmt.Sprintf(", enabled = $%d", argCount)
		args = append(args, *req.Enabled)
		argCount++
	}
	if req.Config != nil {
		configJSON, _ := json.Marshal(*req.Config)
		query += fmt.Sprintf(", config = $%d", argCount)
		args = append(args, string(configJSON))
		argCount++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, channelID)

	result, err := h.db.Exec(query, args...)
	if err != nil {
		log.Errorf("Failed to update channel: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update channel"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	log.Infof("Updated notification channel: %s", channelID)

	c.JSON(http.StatusOK, gin.H{
		"id":      channelID,
		"message": "Channel updated successfully",
	})
}

// DeleteChannel deletes a notification channel
func (h *NotificationHandler) DeleteChannel(c *gin.Context) {
	channelID := c.Param("id")

	result, err := h.db.Exec("DELETE FROM notification_channels WHERE id = $1", channelID)
	if err != nil {
		log.Errorf("Failed to delete channel: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete channel"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
		return
	}

	log.Infof("Deleted notification channel: %s", channelID)

	c.JSON(http.StatusOK, gin.H{"message": "Channel deleted successfully"})
}

// SendNotification sends a notification via a configured channel
func (h *NotificationHandler) SendNotification(c *gin.Context) {
	var req models.SendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Retrieve channel configuration
	var channel models.NotificationChannel
	var configJSON []byte

	query := "SELECT id, type, enabled, config FROM notification_channels WHERE id = $1"
	err := h.db.QueryRow(query, req.ChannelID).Scan(
		&channel.ID, &channel.Type, &channel.Enabled, &configJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve channel"})
		return
	}

	if !channel.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Channel is disabled"})
		return
	}

	// Parse config
	json.Unmarshal(configJSON, &channel.Config)

	// Send notification based on channel type
	startTime := time.Now()
	var sendErr error

	switch channel.Type {
	case "email":
		sendErr = h.sendEmail(channel.Config, req.Subject, req.Message)
	case "slack":
		sendErr = h.sendSlack(channel.Config, req.Subject, req.Message, req.Priority)
	case "pagerduty":
		sendErr = h.sendPagerDuty(channel.Config, req.Subject, req.Message, req.Priority)
	case "webhook":
		sendErr = h.sendWebhook(channel.Config, req.Subject, req.Message, req.Metadata)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported channel type"})
		return
	}

	latency := time.Since(startTime).Milliseconds()

	// Log notification
	logID := uuid.New().String()
	status := "sent"
	errorMsg := ""
	if sendErr != nil {
		status = "failed"
		errorMsg = sendErr.Error()
	}

	metadataJSON, _ := json.Marshal(req.Metadata)
	h.db.Exec(`
		INSERT INTO notification_logs (id, channel_id, channel_type, subject, message, priority, status, error, sent_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), $9)
	`, logID, req.ChannelID, channel.Type, req.Subject, req.Message, req.Priority, status, errorMsg, string(metadataJSON))

	if sendErr != nil {
		log.Errorf("Failed to send notification: %v", sendErr)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Failed to send notification",
			"details":    sendErr.Error(),
			"latency_ms": latency,
		})
		return
	}

	log.Infof("Sent notification via %s (latency: %dms)", channel.Type, latency)

	c.JSON(http.StatusOK, gin.H{
		"log_id":     logID,
		"status":     status,
		"latency_ms": latency,
		"message":    "Notification sent successfully",
	})
}

// TestChannel tests a notification channel configuration
func (h *NotificationHandler) TestChannel(c *gin.Context) {
	var req models.TestChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Retrieve channel
	var channel models.NotificationChannel
	var configJSON []byte

	query := "SELECT id, type, config FROM notification_channels WHERE id = $1"
	err := h.db.QueryRow(query, req.ChannelID).Scan(&channel.ID, &channel.Type, &configJSON)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Channel not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve channel"})
		return
	}

	json.Unmarshal(configJSON, &channel.Config)

	// Send test notification
	startTime := time.Now()
	testSubject := "Privé Platform - Test Notification"
	testMessage := fmt.Sprintf("This is a test notification from Privé Platform sent at %s", time.Now().Format(time.RFC3339))

	var sendErr error
	switch channel.Type {
	case "email":
		sendErr = h.sendEmail(channel.Config, testSubject, testMessage)
	case "slack":
		sendErr = h.sendSlack(channel.Config, testSubject, testMessage, "low")
	case "pagerduty":
		sendErr = h.sendPagerDuty(channel.Config, testSubject, testMessage, "low")
	case "webhook":
		sendErr = h.sendWebhook(channel.Config, testSubject, testMessage, map[string]interface{}{"test": true})
	}

	latency := time.Since(startTime).Milliseconds()

	response := models.TestChannelResponse{
		Success:   sendErr == nil,
		TestedAt:  time.Now(),
		LatencyMs: latency,
	}

	if sendErr != nil {
		response.Message = "Test failed"
		response.Error = sendErr.Error()
	} else {
		response.Message = "Test notification sent successfully"
	}

	c.JSON(http.StatusOK, response)
}

// sendEmail sends an email notification
func (h *NotificationHandler) sendEmail(config map[string]interface{}, subject, message string) error {
	var emailConfig models.EmailConfig
	configJSON, _ := json.Marshal(config)
	json.Unmarshal(configJSON, &emailConfig)

	// Validate required fields
	if emailConfig.SMTPHost == "" || emailConfig.FromAddress == "" || len(emailConfig.Recipients) == 0 {
		return fmt.Errorf("invalid email configuration")
	}

	// Build email
	from := emailConfig.FromAddress
	if emailConfig.FromName != "" {
		from = fmt.Sprintf("%s <%s>", emailConfig.FromName, emailConfig.FromAddress)
	}

	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = emailConfig.Recipients[0]
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=\"utf-8\""

	body := ""
	for k, v := range headers {
		body += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	body += "\r\n" + message

	// Send via SMTP
	addr := fmt.Sprintf("%s:%d", emailConfig.SMTPHost, emailConfig.SMTPPort)
	auth := smtp.PlainAuth("", emailConfig.Username, emailConfig.Password, emailConfig.SMTPHost)

	if emailConfig.UseTLS {
		// TLS connection
		tlsConfig := &tls.Config{
			ServerName:         emailConfig.SMTPHost,
			InsecureSkipVerify: false,
		}

		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to dial SMTP server: %w", err)
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, emailConfig.SMTPHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Quit()

		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed: %w", err)
		}

		if err = client.Mail(emailConfig.FromAddress); err != nil {
			return fmt.Errorf("failed to set sender: %w", err)
		}

		for _, recipient := range emailConfig.Recipients {
			if err = client.Rcpt(recipient); err != nil {
				return fmt.Errorf("failed to add recipient: %w", err)
			}
		}

		w, err := client.Data()
		if err != nil {
			return fmt.Errorf("failed to get data writer: %w", err)
		}

		_, err = w.Write([]byte(body))
		if err != nil {
			return fmt.Errorf("failed to write message: %w", err)
		}

		err = w.Close()
		if err != nil {
			return fmt.Errorf("failed to close writer: %w", err)
		}

		return nil
	}

	// Plain SMTP
	return smtp.SendMail(addr, auth, emailConfig.FromAddress, emailConfig.Recipients, []byte(body))
}

// sendSlack sends a Slack webhook notification
func (h *NotificationHandler) sendSlack(config map[string]interface{}, subject, message, priority string) error {
	var slackConfig models.SlackConfig
	configJSON, _ := json.Marshal(config)
	json.Unmarshal(configJSON, &slackConfig)

	if slackConfig.WebhookURL == "" {
		return fmt.Errorf("slack webhook URL not configured")
	}

	// Build Slack message with formatting
	color := "#36a64f" // green
	switch priority {
	case "high":
		color = "#ff9900" // orange
	case "critical":
		color = "#ff0000" // red
	}

	payload := map[string]interface{}{
		"text": subject,
		"attachments": []map[string]interface{}{
			{
				"color": color,
				"text":  message,
				"footer": "Privé Security Platform",
				"ts":    time.Now().Unix(),
			},
		},
	}

	if slackConfig.Channel != "" {
		payload["channel"] = slackConfig.Channel
	}
	if slackConfig.Username != "" {
		payload["username"] = slackConfig.Username
	}
	if slackConfig.IconEmoji != "" {
		payload["icon_emoji"] = slackConfig.IconEmoji
	}

	payloadJSON, _ := json.Marshal(payload)

	resp, err := http.Post(slackConfig.WebhookURL, "application/json", bytes.NewBuffer(payloadJSON))
	if err != nil {
		return fmt.Errorf("failed to send Slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned non-200 status: %d", resp.StatusCode)
	}

	return nil
}

// sendPagerDuty sends a PagerDuty alert
func (h *NotificationHandler) sendPagerDuty(config map[string]interface{}, subject, message, priority string) error {
	var pdConfig models.PagerDutyConfig
	configJSON, _ := json.Marshal(config)
	json.Unmarshal(configJSON, &pdConfig)

	if pdConfig.IntegrationKey == "" {
		return fmt.Errorf("pagerduty integration key not configured")
	}

	severity := "info"
	switch priority {
	case "high":
		severity = "warning"
	case "critical":
		severity = "critical"
	}

	payload := map[string]interface{}{
		"routing_key":  pdConfig.IntegrationKey,
		"event_action": "trigger",
		"payload": map[string]interface{}{
			"summary":   subject,
			"severity":  severity,
			"source":    "prive-platform",
			"timestamp": time.Now().Format(time.RFC3339),
			"custom_details": map[string]string{
				"message": message,
			},
		},
	}

	payloadJSON, _ := json.Marshal(payload)

	resp, err := http.Post("https://events.pagerduty.com/v2/enqueue", "application/json", bytes.NewBuffer(payloadJSON))
	if err != nil {
		return fmt.Errorf("failed to send PagerDuty event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("pagerduty returned non-202 status: %d", resp.StatusCode)
	}

	return nil
}

// sendWebhook sends a custom webhook notification
func (h *NotificationHandler) sendWebhook(config map[string]interface{}, subject, message string, metadata map[string]interface{}) error {
	var webhookConfig models.WebhookConfig
	configJSON, _ := json.Marshal(config)
	json.Unmarshal(configJSON, &webhookConfig)

	if webhookConfig.URL == "" {
		return fmt.Errorf("webhook URL not configured")
	}

	if webhookConfig.Method == "" {
		webhookConfig.Method = "POST"
	}
	if webhookConfig.Timeout == 0 {
		webhookConfig.Timeout = 10
	}

	// Build payload
	payload := map[string]interface{}{
		"subject":   subject,
		"message":   message,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	if metadata != nil {
		payload["metadata"] = metadata
	}

	payloadJSON, _ := json.Marshal(payload)

	req, err := http.NewRequest(webhookConfig.Method, webhookConfig.URL, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Prive-Platform/1.0")

	// Add custom headers
	for k, v := range webhookConfig.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{
		Timeout: time.Duration(webhookConfig.Timeout) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned non-2xx status: %d", resp.StatusCode)
	}

	return nil
}

// Helper functions

func isValidChannelType(channelType string) bool {
	validTypes := map[string]bool{
		"email":     true,
		"slack":     true,
		"pagerduty": true,
		"webhook":   true,
	}
	return validTypes[channelType]
}

func validateChannelConfig(channelType string, config map[string]interface{}) error {
	switch channelType {
	case "email":
		if _, ok := config["smtp_host"]; !ok {
			return fmt.Errorf("smtp_host required for email channel")
		}
		if _, ok := config["from_address"]; !ok {
			return fmt.Errorf("from_address required for email channel")
		}
		if _, ok := config["recipients"]; !ok {
			return fmt.Errorf("recipients required for email channel")
		}
	case "slack":
		if _, ok := config["webhook_url"]; !ok {
			return fmt.Errorf("webhook_url required for Slack channel")
		}
	case "pagerduty":
		if _, ok := config["integration_key"]; !ok {
			return fmt.Errorf("integration_key required for PagerDuty channel")
		}
	case "webhook":
		if _, ok := config["url"]; !ok {
			return fmt.Errorf("url required for webhook channel")
		}
	}
	return nil
}

func maskWebhookURL(url string) string {
	if len(url) < 20 {
		return "********"
	}
	return url[:10] + "********" + url[len(url)-10:]
}
