// Telemetry Query Handlers with ClickHouse Integration

package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/sentinel-enterprise/platform/api/internal/models"
)

// TelemetryHandler handles telemetry query requests
type TelemetryHandler struct {
	db         *sql.DB            // PostgreSQL for metadata
	clickhouse driver.Conn        // ClickHouse for event data
}

// NewTelemetryHandler creates a new telemetry handler
func NewTelemetryHandler(db *sql.DB) *TelemetryHandler {
	// Initialize ClickHouse connection
	clickhouseAddr := getEnvOrDefault("CLICKHOUSE_ADDR", "localhost:9000")
	ch, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{clickhouseAddr},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		DialTimeout:  5 * time.Second,
	})

	if err != nil {
		log.Errorf("Failed to connect to ClickHouse: %v", err)
		return &TelemetryHandler{db: db, clickhouse: nil}
	}

	if err := ch.Ping(context.Background()); err != nil {
		log.Errorf("ClickHouse ping failed: %v", err)
		return &TelemetryHandler{db: db, clickhouse: nil}
	}

	log.Info("ClickHouse connection established")
	return &TelemetryHandler{
		db:         db,
		clickhouse: ch,
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := getEnv(key, ""); value != "" {
		return value
	}
	return defaultValue
}

// QueryEvents queries telemetry events from ClickHouse with filters
func (h *TelemetryHandler) QueryEvents(c *gin.Context) {
	if h.clickhouse == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ClickHouse connection not available"})
		return
	}

	var req models.QueryEventsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse time range
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time format, use RFC3339"})
		return
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_time format, use RFC3339"})
		return
	}

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 100
	}
	if req.Limit > 10000 {
		req.Limit = 10000
	}
	if req.OrderBy == "" {
		req.OrderBy = "timestamp"
	}
	if req.OrderDirection == "" {
		req.OrderDirection = "DESC"
	}

	// Build query
	queryStart := time.Now()
	query := `
		SELECT
			event_id, agent_id, tenant_id, timestamp, server_timestamp,
			event_type, mitre_tactic, mitre_technique, severity, hostname, os_type,
			payload, process_name, file_path, dst_ip, dst_port, username, ingestion_date
		FROM telemetry_events
		WHERE tenant_id = ?
		  AND timestamp >= ?
		  AND timestamp <= ?
	`

	args := []interface{}{req.TenantID, startTime, endTime}

	// Add filters
	if len(req.EventTypes) > 0 {
		placeholders := make([]string, len(req.EventTypes))
		for i := range req.EventTypes {
			placeholders[i] = "?"
			args = append(args, req.EventTypes[i])
		}
		query += " AND event_type IN (" + strings.Join(placeholders, ",") + ")"
	}

	if len(req.AgentIDs) > 0 {
		placeholders := make([]string, len(req.AgentIDs))
		for i := range req.AgentIDs {
			placeholders[i] = "?"
			args = append(args, req.AgentIDs[i])
		}
		query += " AND agent_id IN (" + strings.Join(placeholders, ",") + ")"
	}

	if len(req.Hostnames) > 0 {
		placeholders := make([]string, len(req.Hostnames))
		for i := range req.Hostnames {
			placeholders[i] = "?"
			args = append(args, req.Hostnames[i])
		}
		query += " AND hostname IN (" + strings.Join(placeholders, ",") + ")"
	}

	if req.MinSeverity != nil {
		query += " AND severity >= ?"
		args = append(args, *req.MinSeverity)
	}

	if len(req.MitreTactics) > 0 {
		placeholders := make([]string, len(req.MitreTactics))
		for i := range req.MitreTactics {
			placeholders[i] = "?"
			args = append(args, req.MitreTactics[i])
		}
		query += " AND mitre_tactic IN (" + strings.Join(placeholders, ",") + ")"
	}

	if len(req.MitreTechniques) > 0 {
		placeholders := make([]string, len(req.MitreTechniques))
		for i := range req.MitreTechniques {
			placeholders[i] = "?"
			args = append(args, req.MitreTechniques[i])
		}
		query += " AND mitre_technique IN (" + strings.Join(placeholders, ",") + ")"
	}

	if len(req.ProcessNames) > 0 {
		placeholders := make([]string, len(req.ProcessNames))
		for i := range req.ProcessNames {
			placeholders[i] = "?"
			args = append(args, req.ProcessNames[i])
		}
		query += " AND process_name IN (" + strings.Join(placeholders, ",") + ")"
	}

	if req.SearchText != "" {
		query += " AND positionCaseInsensitive(payload, ?) > 0"
		args = append(args, req.SearchText)
	}

	// Add ordering and pagination
	query += fmt.Sprintf(" ORDER BY %s %s LIMIT ? OFFSET ?", req.OrderBy, req.OrderDirection)
	args = append(args, req.Limit, req.Offset)

	// Execute query
	ctx := context.Background()
	rows, err := h.clickhouse.Query(ctx, query, args...)
	if err != nil {
		log.Errorf("Failed to query events: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return
	}
	defer rows.Close()

	events := make([]models.TelemetryEvent, 0)
	for rows.Next() {
		var event models.TelemetryEvent
		var payloadStr string
		var eventID string

		err := rows.Scan(
			&eventID,
			&event.AgentID,
			&event.TenantID,
			&event.Timestamp,
			&event.ServerTimestamp,
			&event.EventType,
			&event.MitreTactic,
			&event.MitreTechnique,
			&event.Severity,
			&event.Hostname,
			&event.OSType,
			&payloadStr,
			&event.ProcessName,
			&event.FilePath,
			&event.DstIP,
			&event.DstPort,
			&event.Username,
			&event.IngestionDate,
		)

		if err != nil {
			log.Warnf("Failed to scan event: %v", err)
			continue
		}

		event.EventID = eventID

		// Parse JSON payload
		if payloadStr != "" {
			var payload map[string]interface{}
			if err := json.Unmarshal([]byte(payloadStr), &payload); err == nil {
				event.Payload = payload
			}
		}

		events = append(events, event)
	}

	// Get total count (for pagination)
	countQuery := "SELECT COUNT(*) FROM telemetry_events WHERE tenant_id = ? AND timestamp >= ? AND timestamp <= ?"
	var total int64
	if err := h.clickhouse.QueryRow(ctx, countQuery, req.TenantID, startTime, endTime).Scan(&total); err != nil {
		total = int64(len(events))
	}

	queryDuration := time.Since(queryStart).Milliseconds()

	c.JSON(http.StatusOK, models.QueryEventsResponse{
		Events:      events,
		Total:       total,
		Limit:       req.Limit,
		Offset:      req.Offset,
		QueryTimeMs: queryDuration,
	})
}

// GetEvent retrieves a single event by ID
func (h *TelemetryHandler) GetEvent(c *gin.Context) {
	if h.clickhouse == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ClickHouse connection not available"})
		return
	}

	eventID := c.Param("id")

	query := `
		SELECT
			event_id, agent_id, tenant_id, timestamp, server_timestamp,
			event_type, mitre_tactic, mitre_technique, severity, hostname, os_type,
			payload, process_name, file_path, dst_ip, dst_port, username, ingestion_date
		FROM telemetry_events
		WHERE event_id = ?
		LIMIT 1
	`

	var event models.TelemetryEvent
	var payloadStr string
	var eventIDStr string

	ctx := context.Background()
	err := h.clickhouse.QueryRow(ctx, query, eventID).Scan(
		&eventIDStr,
		&event.AgentID,
		&event.TenantID,
		&event.Timestamp,
		&event.ServerTimestamp,
		&event.EventType,
		&event.MitreTactic,
		&event.MitreTechnique,
		&event.Severity,
		&event.Hostname,
		&event.OSType,
		&payloadStr,
		&event.ProcessName,
		&event.FilePath,
		&event.DstIP,
		&event.DstPort,
		&event.Username,
		&event.IngestionDate,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	event.EventID = eventIDStr

	// Parse JSON payload
	if payloadStr != "" {
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(payloadStr), &payload); err == nil {
			event.Payload = payload
		}
	}

	c.JSON(http.StatusOK, event)
}

// GetStatistics retrieves aggregate statistics
func (h *TelemetryHandler) GetStatistics(c *gin.Context) {
	if h.clickhouse == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ClickHouse connection not available"})
		return
	}

	tenantID := c.Query("tenant_id")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	if tenantID == "" || startTime == "" || endTime == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id, start_time, and end_time required"})
		return
	}

	start, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time format"})
		return
	}

	end, err := time.Parse(time.RFC3339, endTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_time format"})
		return
	}

	ctx := context.Background()

	// Total events
	var totalEvents int64
	h.clickhouse.QueryRow(ctx,
		"SELECT COUNT(*) FROM telemetry_events WHERE tenant_id = ? AND timestamp >= ? AND timestamp <= ?",
		tenantID, start, end).Scan(&totalEvents)

	// Events by type
	eventsByType := make(map[string]int64)
	rows, _ := h.clickhouse.Query(ctx,
		"SELECT event_type, COUNT(*) as cnt FROM telemetry_events WHERE tenant_id = ? AND timestamp >= ? AND timestamp <= ? GROUP BY event_type",
		tenantID, start, end)
	for rows.Next() {
		var eventType string
		var count int64
		rows.Scan(&eventType, &count)
		eventsByType[eventType] = count
	}
	rows.Close()

	// Events by severity
	eventsBySeverity := make(map[uint8]int64)
	rows, _ = h.clickhouse.Query(ctx,
		"SELECT severity, COUNT(*) as cnt FROM telemetry_events WHERE tenant_id = ? AND timestamp >= ? AND timestamp <= ? GROUP BY severity",
		tenantID, start, end)
	for rows.Next() {
		var severity uint8
		var count int64
		rows.Scan(&severity, &count)
		eventsBySeverity[severity] = count
	}
	rows.Close()

	// Top MITRE tactics
	topTactics := make([]models.MitreStat, 0)
	rows, _ = h.clickhouse.Query(ctx,
		`SELECT mitre_tactic, COUNT(*) as cnt FROM telemetry_events
		WHERE tenant_id = ? AND timestamp >= ? AND timestamp <= ? AND mitre_tactic != ''
		GROUP BY mitre_tactic ORDER BY cnt DESC LIMIT 10`,
		tenantID, start, end)
	for rows.Next() {
		var tactic string
		var count int64
		rows.Scan(&tactic, &count)
		percentage := float64(count) / float64(totalEvents) * 100
		topTactics = append(topTactics, models.MitreStat{
			ID:         tactic,
			EventCount: count,
			Percentage: percentage,
		})
	}
	rows.Close()

	// Unique counts
	var uniqueAgents, uniqueHosts int64
	h.clickhouse.QueryRow(ctx,
		"SELECT uniq(agent_id) FROM telemetry_events WHERE tenant_id = ? AND timestamp >= ? AND timestamp <= ?",
		tenantID, start, end).Scan(&uniqueAgents)
	h.clickhouse.QueryRow(ctx,
		"SELECT uniq(hostname) FROM telemetry_events WHERE tenant_id = ? AND timestamp >= ? AND timestamp <= ?",
		tenantID, start, end).Scan(&uniqueHosts)

	stats := models.Statistics{
		TotalEvents:      totalEvents,
		EventsByType:     eventsByType,
		EventsBySeverity: eventsBySeverity,
		TopMitreTactics:  topTactics,
		UniqueAgents:     uniqueAgents,
		UniqueHosts:      uniqueHosts,
		TimeRange: models.TimeRange{
			Start: start,
			End:   end,
		},
	}

	c.JSON(http.StatusOK, stats)
}

// ListMITRETactics retrieves all MITRE tactics from PostgreSQL
func (h *TelemetryHandler) ListMITRETactics(c *gin.Context) {
	query := `SELECT tactic_id, name, description, url FROM mitre_tactics ORDER BY tactic_id`

	rows, err := h.db.Query(query)
	if err != nil {
		log.Errorf("Failed to query MITRE tactics: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}
	defer rows.Close()

	tactics := make([]models.MITRETactic, 0)
	for rows.Next() {
		var tactic models.MITRETactic
		var description, url sql.NullString

		err := rows.Scan(&tactic.TacticID, &tactic.Name, &description, &url)
		if err != nil {
			log.Warnf("Failed to scan tactic: %v", err)
			continue
		}

		if description.Valid {
			tactic.Description = description.String
		}
		if url.Valid {
			tactic.URL = url.String
		}

		tactics = append(tactics, tactic)
	}

	c.JSON(http.StatusOK, gin.H{
		"tactics": tactics,
		"total":   len(tactics),
	})
}

// ListMITRETechniques retrieves MITRE techniques, optionally filtered by tactic
func (h *TelemetryHandler) ListMITRETechniques(c *gin.Context) {
	tacticID := c.Query("tactic_id")

	query := `
		SELECT technique_id, tactic_id, name, description, platforms, url
		FROM mitre_techniques
	`
	args := []interface{}{}

	if tacticID != "" {
		query += " WHERE tactic_id = $1"
		args = append(args, tacticID)
	}

	query += " ORDER BY technique_id"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		log.Errorf("Failed to query MITRE techniques: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}
	defer rows.Close()

	techniques := make([]models.MITRETechnique, 0)
	for rows.Next() {
		var tech models.MITRETechnique
		var tacticID, description, url sql.NullString
		var platforms interface{}

		err := rows.Scan(&tech.TechniqueID, &tacticID, &tech.Name, &description, &platforms, &url)
		if err != nil {
			log.Warnf("Failed to scan technique: %v", err)
			continue
		}

		if tacticID.Valid {
			tech.TacticID = tacticID.String
		}
		if description.Valid {
			tech.Description = description.String
		}
		if url.Valid {
			tech.URL = url.String
		}

		// Parse platforms array
		if platforms != nil {
			if platArray, ok := platforms.([]interface{}); ok {
				tech.Platforms = make([]string, 0)
				for _, p := range platArray {
					if pStr, ok := p.(string); ok {
						tech.Platforms = append(tech.Platforms, pStr)
					}
				}
			}
		}

		techniques = append(techniques, tech)
	}

	c.JSON(http.StatusOK, gin.H{
		"techniques": techniques,
		"total":      len(techniques),
	})
}

// GetMITRECoverage calculates MITRE ATT&CK detection coverage
func (h *TelemetryHandler) GetMITRECoverage(c *gin.Context) {
	if h.clickhouse == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ClickHouse connection not available"})
		return
	}

	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	// Get total techniques from PostgreSQL
	var totalTechniques int
	h.db.QueryRow("SELECT COUNT(*) FROM mitre_techniques").Scan(&totalTechniques)

	// Get detected techniques from ClickHouse
	ctx := context.Background()
	rows, err := h.clickhouse.Query(ctx,
		`SELECT mitre_technique, COUNT(*) as cnt, min(timestamp) as first_seen, max(timestamp) as last_seen
		FROM telemetry_events
		WHERE tenant_id = ? AND mitre_technique != ''
		GROUP BY mitre_technique`,
		tenantID)

	if err != nil {
		log.Errorf("Failed to query coverage: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Query failed"})
		return
	}
	defer rows.Close()

	detectedTechniques := make([]models.DetectedTechnique, 0)
	for rows.Next() {
		var tech models.DetectedTechnique
		rows.Scan(&tech.TechniqueID, &tech.EventCount, &tech.FirstSeen, &tech.LastSeen)
		detectedTechniques = append(detectedTechniques, tech)
	}

	coverage := models.MITRECoverage{
		TenantID:           tenantID,
		TotalTechniques:    totalTechniques,
		DetectedCount:      len(detectedTechniques),
		CoveragePercent:    float64(len(detectedTechniques)) / float64(totalTechniques) * 100,
		DetectedTechniques: detectedTechniques,
	}

	c.JSON(http.StatusOK, coverage)
}

// Alert Rules Management

// ListAlertRules retrieves all alert rules for a tenant
func (h *TelemetryHandler) ListAlertRules(c *gin.Context) {
	licenseID := c.Query("license_id")
	if licenseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "license_id required"})
		return
	}

	query := `
		SELECT id, license_id, name, description, severity, enabled, condition, actions, created_at, updated_at
		FROM alert_rules
		WHERE license_id = $1
		ORDER BY created_at DESC
	`

	rows, err := h.db.Query(query, licenseID)
	if err != nil {
		log.Errorf("Failed to query alert rules: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}
	defer rows.Close()

	rules := make([]models.AlertRule, 0)
	for rows.Next() {
		var rule models.AlertRule
		var conditionJSON, actionsJSON []byte
		var description sql.NullString

		err := rows.Scan(
			&rule.ID, &rule.LicenseID, &rule.Name, &description, &rule.Severity,
			&rule.Enabled, &conditionJSON, &actionsJSON, &rule.CreatedAt, &rule.UpdatedAt,
		)

		if err != nil {
			log.Warnf("Failed to scan rule: %v", err)
			continue
		}

		if description.Valid {
			rule.Description = description.String
		}

		// Parse JSON fields
		if len(conditionJSON) > 0 {
			json.Unmarshal(conditionJSON, &rule.Condition)
		}
		if len(actionsJSON) > 0 {
			json.Unmarshal(actionsJSON, &rule.Actions)
		}

		rules = append(rules, rule)
	}

	c.JSON(http.StatusOK, gin.H{
		"rules": rules,
		"total": len(rules),
	})
}

// CreateAlertRule creates a new alert rule
func (h *TelemetryHandler) CreateAlertRule(c *gin.Context) {
	var req models.CreateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ruleID := uuid.New().String()
	conditionJSON, _ := json.Marshal(req.Condition)
	actionsJSON, _ := json.Marshal(req.Actions)

	query := `
		INSERT INTO alert_rules (id, license_id, name, description, severity, enabled, condition, actions, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING created_at, updated_at
	`

	var createdAt, updatedAt time.Time
	err := h.db.QueryRow(query,
		ruleID, req.LicenseID, req.Name, req.Description, req.Severity,
		req.Enabled, string(conditionJSON), string(actionsJSON),
	).Scan(&createdAt, &updatedAt)

	if err != nil {
		log.Errorf("Failed to create alert rule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create rule"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         ruleID,
		"created_at": createdAt,
		"message":    "Alert rule created successfully",
	})
}

// UpdateAlertRule updates an existing alert rule
func (h *TelemetryHandler) UpdateAlertRule(c *gin.Context) {
	ruleID := c.Param("id")

	var req models.UpdateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build dynamic update query (similar to DLP handler)
	query := "UPDATE alert_rules SET updated_at = NOW()"
	args := []interface{}{}
	argCount := 1

	if req.Name != nil {
		query += fmt.Sprintf(", name = $%d", argCount)
		args = append(args, *req.Name)
		argCount++
	}
	if req.Description != nil {
		query += fmt.Sprintf(", description = $%d", argCount)
		args = append(args, *req.Description)
		argCount++
	}
	if req.Severity != nil {
		query += fmt.Sprintf(", severity = $%d", argCount)
		args = append(args, *req.Severity)
		argCount++
	}
	if req.Enabled != nil {
		query += fmt.Sprintf(", enabled = $%d", argCount)
		args = append(args, *req.Enabled)
		argCount++
	}
	if req.Condition != nil {
		conditionJSON, _ := json.Marshal(*req.Condition)
		query += fmt.Sprintf(", condition = $%d", argCount)
		args = append(args, string(conditionJSON))
		argCount++
	}
	if req.Actions != nil {
		actionsJSON, _ := json.Marshal(*req.Actions)
		query += fmt.Sprintf(", actions = $%d", argCount)
		args = append(args, string(actionsJSON))
		argCount++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, ruleID)

	result, err := h.db.Exec(query, args...)
	if err != nil {
		log.Errorf("Failed to update alert rule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update rule"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      ruleID,
		"message": "Alert rule updated successfully",
	})
}

// DeleteAlertRule deletes an alert rule
func (h *TelemetryHandler) DeleteAlertRule(c *gin.Context) {
	ruleID := c.Param("id")

	result, err := h.db.Exec("DELETE FROM alert_rules WHERE id = $1", ruleID)
	if err != nil {
		log.Errorf("Failed to delete alert rule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete rule"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Alert rule deleted successfully"})
}
