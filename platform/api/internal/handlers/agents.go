// Agent Management Handlers with PostgreSQL Integration

package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/sentinel-enterprise/platform/api/internal/models"
)

// AgentHandler handles agent management requests
type AgentHandler struct {
	db *sql.DB
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(db *sql.DB) *AgentHandler {
	return &AgentHandler{
		db: db,
	}
}

// ListAgents retrieves all agents for a tenant with optional filtering and pagination
func (h *AgentHandler) ListAgents(c *gin.Context) {
	licenseID := c.Query("license_id")
	if licenseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "license_id required"})
		return
	}

	// Pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	// Optional filters
	status := c.Query("status")
	osType := c.Query("os_type")

	// Build query with filters
	query := `
		SELECT id, agent_id, license_id, hostname, ip_address, os_type, os_version,
		       agent_version, status, last_seen, cpu_usage, memory_usage_mb,
		       events_sent, config, created_at, updated_at
		FROM agents
		WHERE license_id = $1
	`
	args := []interface{}{licenseID}
	argCount := 2

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	if osType != "" {
		query += fmt.Sprintf(" AND os_type = $%d", argCount)
		args = append(args, osType)
		argCount++
	}

	query += " ORDER BY last_seen DESC NULLS LAST"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		log.Errorf("Failed to query agents: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}
	defer rows.Close()

	agents := make([]models.Agent, 0)
	for rows.Next() {
		var agent models.Agent
		var configJSON []byte
		var ipAddress, osType, osVersion, agentVersion sql.NullString
		var lastSeen sql.NullTime
		var cpuUsage sql.NullFloat64
		var memoryUsage sql.NullInt64

		err := rows.Scan(
			&agent.ID,
			&agent.AgentID,
			&agent.LicenseID,
			&agent.Hostname,
			&ipAddress,
			&osType,
			&osVersion,
			&agentVersion,
			&agent.Status,
			&lastSeen,
			&cpuUsage,
			&memoryUsage,
			&agent.EventsSent,
			&configJSON,
			&agent.CreatedAt,
			&agent.UpdatedAt,
		)

		if err != nil {
			log.Warnf("Failed to scan agent: %v", err)
			continue
		}

		// Handle NULL fields
		if ipAddress.Valid {
			agent.IPAddress = ipAddress.String
		}
		if osType.Valid {
			agent.OSType = osType.String
		}
		if osVersion.Valid {
			agent.OSVersion = osVersion.String
		}
		if agentVersion.Valid {
			agent.AgentVersion = agentVersion.String
		}
		if lastSeen.Valid {
			agent.LastSeen = &lastSeen.Time
		}
		if cpuUsage.Valid {
			agent.CPUUsage = &cpuUsage.Float64
		}
		if memoryUsage.Valid {
			memMB := int(memoryUsage.Int64)
			agent.MemoryUsageMB = &memMB
		}

		// Parse JSON config
		if len(configJSON) > 0 {
			json.Unmarshal(configJSON, &agent.Config)
		}

		agents = append(agents, agent)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM agents WHERE license_id = $1"
	countArgs := []interface{}{licenseID}
	if status != "" {
		countQuery += " AND status = $2"
		countArgs = append(countArgs, status)
	}

	var total int
	h.db.QueryRow(countQuery, countArgs...).Scan(&total)

	c.JSON(http.StatusOK, models.AgentListResponse{
		Agents: agents,
		Total:  total,
		Page:   page,
		Limit:  limit,
	})
}

// GetAgent retrieves a specific agent by ID
func (h *AgentHandler) GetAgent(c *gin.Context) {
	agentID := c.Param("id")

	query := `
		SELECT id, agent_id, license_id, hostname, ip_address, os_type, os_version,
		       agent_version, status, last_seen, cpu_usage, memory_usage_mb,
		       events_sent, config, created_at, updated_at
		FROM agents
		WHERE id = $1
	`

	var agent models.Agent
	var configJSON []byte
	var ipAddress, osType, osVersion, agentVersion sql.NullString
	var lastSeen sql.NullTime
	var cpuUsage sql.NullFloat64
	var memoryUsage sql.NullInt64

	err := h.db.QueryRow(query, agentID).Scan(
		&agent.ID,
		&agent.AgentID,
		&agent.LicenseID,
		&agent.Hostname,
		&ipAddress,
		&osType,
		&osVersion,
		&agentVersion,
		&agent.Status,
		&lastSeen,
		&cpuUsage,
		&memoryUsage,
		&agent.EventsSent,
		&configJSON,
		&agent.CreatedAt,
		&agent.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
			return
		}
		log.Errorf("Failed to query agent: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}

	// Handle NULL fields
	if ipAddress.Valid {
		agent.IPAddress = ipAddress.String
	}
	if osType.Valid {
		agent.OSType = osType.String
	}
	if osVersion.Valid {
		agent.OSVersion = osVersion.String
	}
	if agentVersion.Valid {
		agent.AgentVersion = agentVersion.String
	}
	if lastSeen.Valid {
		agent.LastSeen = &lastSeen.Time
	}
	if cpuUsage.Valid {
		agent.CPUUsage = &cpuUsage.Float64
	}
	if memoryUsage.Valid {
		memMB := int(memoryUsage.Int64)
		agent.MemoryUsageMB = &memMB
	}

	// Parse JSON config
	if len(configJSON) > 0 {
		json.Unmarshal(configJSON, &agent.Config)
	}

	c.JSON(http.StatusOK, agent)
}

// UpdateAgent updates agent metadata
func (h *AgentHandler) UpdateAgent(c *gin.Context) {
	agentID := c.Param("id")

	var req models.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build dynamic update query
	query := `UPDATE agents SET updated_at = NOW()`
	args := []interface{}{}
	argCount := 1

	if req.Hostname != nil {
		query += fmt.Sprintf(", hostname = $%d", argCount)
		args = append(args, *req.Hostname)
		argCount++
	}
	if req.IPAddress != nil {
		query += fmt.Sprintf(", ip_address = $%d", argCount)
		args = append(args, *req.IPAddress)
		argCount++
	}
	if req.OSVersion != nil {
		query += fmt.Sprintf(", os_version = $%d", argCount)
		args = append(args, *req.OSVersion)
		argCount++
	}
	if req.AgentVersion != nil {
		query += fmt.Sprintf(", agent_version = $%d", argCount)
		args = append(args, *req.AgentVersion)
		argCount++
	}
	if req.Status != nil {
		query += fmt.Sprintf(", status = $%d", argCount)
		args = append(args, *req.Status)
		argCount++
	}
	if req.CPUUsage != nil {
		query += fmt.Sprintf(", cpu_usage = $%d", argCount)
		args = append(args, *req.CPUUsage)
		argCount++
	}
	if req.MemoryUsageMB != nil {
		query += fmt.Sprintf(", memory_usage_mb = $%d", argCount)
		args = append(args, *req.MemoryUsageMB)
		argCount++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, agentID)

	result, err := h.db.Exec(query, args...)
	if err != nil {
		log.Errorf("Failed to update agent: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agent"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	log.Infof("Updated agent: %s", agentID)

	c.JSON(http.StatusOK, gin.H{
		"id":         agentID,
		"updated_at": time.Now(),
		"message":    "Agent updated successfully",
	})
}

// DeleteAgent removes an agent (decommission)
func (h *AgentHandler) DeleteAgent(c *gin.Context) {
	agentID := c.Param("id")

	query := `DELETE FROM agents WHERE id = $1`

	result, err := h.db.Exec(query, agentID)
	if err != nil {
		log.Errorf("Failed to delete agent: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete agent"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	log.Infof("Deleted agent: %s", agentID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Agent deleted successfully",
	})
}

// GetAgentConfig retrieves agent configuration
func (h *AgentHandler) GetAgentConfig(c *gin.Context) {
	agentID := c.Param("id")

	query := `SELECT config FROM agents WHERE id = $1`

	var configJSON []byte
	err := h.db.QueryRow(query, agentID).Scan(&configJSON)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
			return
		}
		log.Errorf("Failed to query agent config: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}

	var config map[string]interface{}
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &config); err != nil {
			log.Errorf("Failed to unmarshal config: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid configuration"})
			return
		}
	} else {
		config = make(map[string]interface{})
	}

	c.JSON(http.StatusOK, gin.H{
		"agent_id": agentID,
		"config":   config,
	})
}

// UpdateAgentConfig updates agent configuration
func (h *AgentHandler) UpdateAgentConfig(c *gin.Context) {
	agentID := c.Param("id")

	var req models.UpdateAgentConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Serialize config to JSON
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid configuration format"})
		return
	}

	query := `
		UPDATE agents
		SET config = $1, updated_at = NOW()
		WHERE id = $2
	`

	result, err := h.db.Exec(query, string(configJSON), agentID)
	if err != nil {
		log.Errorf("Failed to update agent config: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update configuration"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	log.Infof("Updated agent config: %s", agentID)

	c.JSON(http.StatusOK, gin.H{
		"agent_id": agentID,
		"message":  "Configuration updated successfully",
	})
}

// GetAgentHealth retrieves agent health metrics
func (h *AgentHandler) GetAgentHealth(c *gin.Context) {
	agentID := c.Param("id")

	query := `
		SELECT agent_id, status, last_seen, cpu_usage, memory_usage_mb, created_at
		FROM agents
		WHERE id = $1
	`

	var health models.AgentHealthResponse
	var lastSeen sql.NullTime
	var cpuUsage sql.NullFloat64
	var memoryUsage sql.NullInt64
	var createdAt time.Time

	err := h.db.QueryRow(query, agentID).Scan(
		&health.AgentID,
		&health.Status,
		&lastSeen,
		&cpuUsage,
		&memoryUsage,
		&createdAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
			return
		}
		log.Errorf("Failed to query agent health: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}

	// Handle NULL fields
	if lastSeen.Valid {
		health.LastSeen = &lastSeen.Time
	}
	if cpuUsage.Valid {
		health.CPUUsage = &cpuUsage.Float64
	}
	if memoryUsage.Valid {
		memMB := int(memoryUsage.Int64)
		health.MemoryUsageMB = &memMB
	}

	// Calculate uptime
	health.Uptime = int64(time.Since(createdAt).Seconds())

	// Determine health status
	health.IsHealthy = true
	health.Issues = make([]string, 0)

	// Check if agent is offline (no heartbeat in 5 minutes)
	if lastSeen.Valid {
		timeSinceLastSeen := time.Since(lastSeen.Time)
		if timeSinceLastSeen > 5*time.Minute {
			health.IsHealthy = false
			health.Issues = append(health.Issues, fmt.Sprintf("No heartbeat for %d minutes", int(timeSinceLastSeen.Minutes())))
		}
	} else {
		health.IsHealthy = false
		health.Issues = append(health.Issues, "Never received heartbeat")
	}

	// Check CPU usage
	if cpuUsage.Valid && cpuUsage.Float64 > 5.0 {
		health.Issues = append(health.Issues, fmt.Sprintf("High CPU usage: %.2f%%", cpuUsage.Float64))
	}

	// Check memory usage
	if memoryUsage.Valid && memoryUsage.Int64 > 100 {
		health.Issues = append(health.Issues, fmt.Sprintf("High memory usage: %d MB", memoryUsage.Int64))
	}

	// Check status
	if health.Status == "error" || health.Status == "offline" {
		health.IsHealthy = false
	}

	c.JSON(http.StatusOK, health)
}

// RegisterAgent handles new agent registration
func (h *AgentHandler) RegisterAgent(c *gin.Context) {
	var req models.AgentRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate license key and get license_id
	var licenseID string
	var isActive bool
	err := h.db.QueryRow(
		"SELECT id, is_active FROM licenses WHERE license_key = $1",
		req.LicenseKey,
	).Scan(&licenseID, &isActive)

	if err != nil || !isActive {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or inactive license key"})
		return
	}

	// Check if agent already exists
	var existingID string
	err = h.db.QueryRow("SELECT id FROM agents WHERE agent_id = $1", req.AgentID).Scan(&existingID)

	if err == nil {
		// Agent exists, update it
		query := `
			UPDATE agents
			SET license_id = $1, hostname = $2, ip_address = $3, os_type = $4,
			    os_version = $5, agent_version = $6, status = 'active',
			    last_seen = NOW(), updated_at = NOW()
			WHERE agent_id = $7
			RETURNING id
		`

		err = h.db.QueryRow(query,
			licenseID, req.Hostname, req.IPAddress, req.OSType,
			req.OSVersion, req.AgentVersion, req.AgentID,
		).Scan(&existingID)

		if err != nil {
			log.Errorf("Failed to update agent: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register agent"})
			return
		}

		log.Infof("Agent re-registered: %s", req.AgentID)
		c.JSON(http.StatusOK, gin.H{
			"id":       existingID,
			"agent_id": req.AgentID,
			"message":  "Agent re-registered successfully",
		})
		return
	}

	// New agent, insert it
	id := uuid.New().String()
	query := `
		INSERT INTO agents (id, agent_id, license_id, hostname, ip_address, os_type,
		                    os_version, agent_version, status, last_seen, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'active', NOW(), NOW(), NOW())
		RETURNING id, created_at
	`

	var createdAt time.Time
	err = h.db.QueryRow(query,
		id, req.AgentID, licenseID, req.Hostname, req.IPAddress,
		req.OSType, req.OSVersion, req.AgentVersion,
	).Scan(&id, &createdAt)

	if err != nil {
		log.Errorf("Failed to register agent: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register agent"})
		return
	}

	log.Infof("New agent registered: %s (%s)", req.Hostname, req.AgentID)

	c.JSON(http.StatusCreated, gin.H{
		"id":         id,
		"agent_id":   req.AgentID,
		"created_at": createdAt,
		"message":    "Agent registered successfully",
	})
}

// ProcessHeartbeat handles agent heartbeat updates
func (h *AgentHandler) ProcessHeartbeat(c *gin.Context) {
	var req models.AgentHeartbeat
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `
		UPDATE agents
		SET last_seen = NOW(), cpu_usage = $1, memory_usage_mb = $2,
		    events_sent = $3, status = $4, updated_at = NOW()
		WHERE agent_id = $5
	`

	result, err := h.db.Exec(query,
		req.CPUUsage, req.MemoryUsageMB, req.EventsSent,
		req.Status, req.AgentID,
	)

	if err != nil {
		log.Errorf("Failed to process heartbeat: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process heartbeat"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"agent_id": req.AgentID,
		"message":  "Heartbeat processed",
	})
}
