// Deception Technology Handler
// Manages honeypots, honey tokens, and deception-based threat detection

package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"

	"github.com/sentinel-enterprise/platform/api/internal/models"
)

// DeceptionHandler handles deception technology operations
type DeceptionHandler struct {
	db *sql.DB
}

// NewDeceptionHandler creates a new deception handler
func NewDeceptionHandler(db *sql.DB) *DeceptionHandler {
	return &DeceptionHandler{db: db}
}

// CreateHoneypot deploys a new honeypot
func (h *DeceptionHandler) CreateHoneypot(c *gin.Context) {
	var req models.CreateHoneypotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	honeypotID := uuid.New().String()
	configJSON, _ := json.Marshal(req.Configuration)
	metadataJSON, _ := json.Marshal(req.Metadata)

	query := `
		INSERT INTO honeypots (
			id, license_id, name, honeypot_type, status, deployment_mode,
			target_platform, configuration, location, is_active, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, TRUE, $10)
		RETURNING deployed_at, created_at, updated_at
	`

	var deployedAt, createdAt, updatedAt time.Time
	err := h.db.QueryRow(query,
		honeypotID,
		req.LicenseID,
		req.Name,
		req.HoneypotType,
		models.HoneypotStatusActive,
		req.DeploymentMode,
		req.TargetPlatform,
		configJSON,
		req.Location,
		metadataJSON,
	).Scan(&deployedAt, &createdAt, &updatedAt)

	if err != nil {
		log.Errorf("Failed to create honeypot: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create honeypot"})
		return
	}

	honeypot := models.Honeypot{
		ID:              honeypotID,
		LicenseID:       req.LicenseID,
		Name:            req.Name,
		HoneypotType:    req.HoneypotType,
		Status:          models.HoneypotStatusActive,
		DeploymentMode:  req.DeploymentMode,
		TargetPlatform:  req.TargetPlatform,
		Configuration:   req.Configuration,
		Location:        req.Location,
		IsActive:        true,
		InteractionCount: 0,
		DeployedAt:      deployedAt,
		Metadata:        req.Metadata,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}

	c.JSON(http.StatusCreated, honeypot)
}

// ListHoneypots lists all honeypots for a license
func (h *DeceptionHandler) ListHoneypots(c *gin.Context) {
	licenseID := c.Query("license_id")
	status := c.Query("status")

	query := `
		SELECT id, license_id, name, honeypot_type, status, deployment_mode,
		       target_platform, configuration, location, is_active,
		       interaction_count, last_interaction, deployed_at,
		       created_at, updated_at
		FROM honeypots
		WHERE license_id = $1
	`

	args := []interface{}{licenseID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}

	query += " ORDER BY deployed_at DESC"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		log.Errorf("Failed to list honeypots: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list honeypots"})
		return
	}
	defer rows.Close()

	honeypots := []models.Honeypot{}
	for rows.Next() {
		var honeypot models.Honeypot
		var configJSON []byte
		var lastInteraction sql.NullTime

		err := rows.Scan(
			&honeypot.ID,
			&honeypot.LicenseID,
			&honeypot.Name,
			&honeypot.HoneypotType,
			&honeypot.Status,
			&honeypot.DeploymentMode,
			&honeypot.TargetPlatform,
			&configJSON,
			&honeypot.Location,
			&honeypot.IsActive,
			&honeypot.InteractionCount,
			&lastInteraction,
			&honeypot.DeployedAt,
			&honeypot.CreatedAt,
			&honeypot.UpdatedAt,
		)

		if err != nil {
			continue
		}

		json.Unmarshal(configJSON, &honeypot.Configuration)
		if lastInteraction.Valid {
			honeypot.LastInteraction = &lastInteraction.Time
		}

		honeypots = append(honeypots, honeypot)
	}

	c.JSON(http.StatusOK, gin.H{
		"honeypots": honeypots,
		"count":     len(honeypots),
	})
}

// GetHoneypot retrieves a specific honeypot
func (h *DeceptionHandler) GetHoneypot(c *gin.Context) {
	id := c.Param("id")

	query := `
		SELECT id, license_id, name, honeypot_type, status, deployment_mode,
		       target_platform, configuration, location, is_active,
		       interaction_count, last_interaction, deployed_at, metadata,
		       created_at, updated_at
		FROM honeypots
		WHERE id = $1
	`

	var honeypot models.Honeypot
	var configJSON, metadataJSON []byte
	var lastInteraction sql.NullTime

	err := h.db.QueryRow(query, id).Scan(
		&honeypot.ID,
		&honeypot.LicenseID,
		&honeypot.Name,
		&honeypot.HoneypotType,
		&honeypot.Status,
		&honeypot.DeploymentMode,
		&honeypot.TargetPlatform,
		&configJSON,
		&honeypot.Location,
		&honeypot.IsActive,
		&honeypot.InteractionCount,
		&lastInteraction,
		&honeypot.DeployedAt,
		&metadataJSON,
		&honeypot.CreatedAt,
		&honeypot.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Honeypot not found"})
		return
	}

	if err != nil {
		log.Errorf("Failed to get honeypot: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve honeypot"})
		return
	}

	json.Unmarshal(configJSON, &honeypot.Configuration)
	json.Unmarshal(metadataJSON, &honeypot.Metadata)
	if lastInteraction.Valid {
		honeypot.LastInteraction = &lastInteraction.Time
	}

	c.JSON(http.StatusOK, honeypot)
}

// UpdateHoneypot updates a honeypot
func (h *DeceptionHandler) UpdateHoneypot(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateHoneypotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `
		UPDATE honeypots
		SET name = COALESCE($1, name),
		    status = COALESCE($2, status),
		    is_active = COALESCE($3, is_active),
		    updated_at = NOW()
		WHERE id = $4
	`

	result, err := h.db.Exec(query, req.Name, req.Status, req.IsActive, id)
	if err != nil {
		log.Errorf("Failed to update honeypot: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update honeypot"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Honeypot not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Honeypot updated successfully"})
}

// DeleteHoneypot deletes a honeypot
func (h *DeceptionHandler) DeleteHoneypot(c *gin.Context) {
	id := c.Param("id")

	result, err := h.db.Exec("DELETE FROM honeypots WHERE id = $1", id)
	if err != nil {
		log.Errorf("Failed to delete honeypot: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete honeypot"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Honeypot not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Honeypot deleted successfully"})
}

// CreateHoneyToken creates a new honey token
func (h *DeceptionHandler) CreateHoneyToken(c *gin.Context) {
	var req models.CreateHoneyTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tokenID := uuid.New().String()
	tokenValue := h.generateHoneyToken(req.TokenType)

	// Generate callback URL if not provided
	callbackURL := req.CallbackURL
	if callbackURL == "" {
		callbackURL = fmt.Sprintf("https://api.prive-platform.com/v1/deception/callback/%s", tokenID)
	}

	metadataJSON, _ := json.Marshal(req.Metadata)

	query := `
		INSERT INTO honey_tokens (
			id, license_id, name, token_type, token_value, callback_url,
			is_active, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, TRUE, $7)
		RETURNING created_at, updated_at
	`

	var createdAt, updatedAt time.Time
	err := h.db.QueryRow(query,
		tokenID,
		req.LicenseID,
		req.Name,
		req.TokenType,
		tokenValue,
		callbackURL,
		metadataJSON,
	).Scan(&createdAt, &updatedAt)

	if err != nil {
		log.Errorf("Failed to create honey token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create honey token"})
		return
	}

	token := models.HoneyToken{
		ID:          tokenID,
		LicenseID:   req.LicenseID,
		Name:        req.Name,
		TokenType:   req.TokenType,
		TokenValue:  tokenValue,
		CallbackURL: callbackURL,
		IsActive:    true,
		AccessCount: 0,
		Metadata:    req.Metadata,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	c.JSON(http.StatusCreated, token)
}

// ListHoneyTokens lists all honey tokens for a license
func (h *DeceptionHandler) ListHoneyTokens(c *gin.Context) {
	licenseID := c.Query("license_id")

	query := `
		SELECT id, license_id, name, token_type, token_value, callback_url,
		       is_active, access_count, last_accessed, created_at, updated_at
		FROM honey_tokens
		WHERE license_id = $1
		ORDER BY created_at DESC
	`

	rows, err := h.db.Query(query, licenseID)
	if err != nil {
		log.Errorf("Failed to list honey tokens: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list tokens"})
		return
	}
	defer rows.Close()

	tokens := []models.HoneyToken{}
	for rows.Next() {
		var token models.HoneyToken
		var lastAccessed sql.NullTime

		err := rows.Scan(
			&token.ID,
			&token.LicenseID,
			&token.Name,
			&token.TokenType,
			&token.TokenValue,
			&token.CallbackURL,
			&token.IsActive,
			&token.AccessCount,
			&lastAccessed,
			&token.CreatedAt,
			&token.UpdatedAt,
		)

		if err != nil {
			continue
		}

		if lastAccessed.Valid {
			token.LastAccessed = &lastAccessed.Time
		}

		tokens = append(tokens, token)
	}

	c.JSON(http.StatusOK, gin.H{
		"tokens": tokens,
		"count":  len(tokens),
	})
}

// RecordDeceptionEvent records an interaction with a deception asset
func (h *DeceptionHandler) RecordDeceptionEvent(c *gin.Context) {
	var event models.DeceptionEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	eventID := uuid.New().String()
	detailsJSON, _ := json.Marshal(event.Details)
	metadataJSON, _ := json.Marshal(event.Metadata)

	query := `
		INSERT INTO deception_events (
			id, license_id, event_type, honeypot_id, honey_token_id,
			source_ip, source_hostname, source_user, interaction_type,
			severity, details, alert_created, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, FALSE, $12)
		RETURNING detected_at
	`

	var detectedAt time.Time
	err := h.db.QueryRow(query,
		eventID,
		event.LicenseID,
		event.EventType,
		event.HoneypotID,
		event.HoneyTokenID,
		event.SourceIP,
		event.SourceHostname,
		event.SourceUser,
		event.InteractionType,
		event.Severity,
		detailsJSON,
		metadataJSON,
	).Scan(&detectedAt)

	if err != nil {
		log.Errorf("Failed to record deception event: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record event"})
		return
	}

	// Update interaction count
	if event.HoneypotID != "" {
		h.db.Exec(`
			UPDATE honeypots
			SET interaction_count = interaction_count + 1,
			    last_interaction = NOW()
			WHERE id = $1
		`, event.HoneypotID)
	}

	if event.HoneyTokenID != "" {
		h.db.Exec(`
			UPDATE honey_tokens
			SET access_count = access_count + 1,
			    last_accessed = NOW()
			WHERE id = $1
		`, event.HoneyTokenID)
	}

	event.ID = eventID
	event.DetectedAt = detectedAt
	event.AlertCreated = false

	c.JSON(http.StatusCreated, event)
}

// ListDeceptionEvents lists deception events
func (h *DeceptionHandler) ListDeceptionEvents(c *gin.Context) {
	licenseID := c.Query("license_id")
	limit := 100

	query := `
		SELECT id, license_id, event_type, honeypot_id, honey_token_id,
		       source_ip, source_hostname, source_user, interaction_type,
		       severity, details, alert_created, alert_id, detected_at
		FROM deception_events
		WHERE license_id = $1
		ORDER BY detected_at DESC
		LIMIT $2
	`

	rows, err := h.db.Query(query, licenseID, limit)
	if err != nil {
		log.Errorf("Failed to list deception events: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list events"})
		return
	}
	defer rows.Close()

	events := []models.DeceptionEvent{}
	for rows.Next() {
		var event models.DeceptionEvent
		var detailsJSON []byte
		var honeypotID, honeyTokenID, sourceHostname, sourceUser, alertID sql.NullString

		err := rows.Scan(
			&event.ID,
			&event.LicenseID,
			&event.EventType,
			&honeypotID,
			&honeyTokenID,
			&event.SourceIP,
			&sourceHostname,
			&sourceUser,
			&event.InteractionType,
			&event.Severity,
			&detailsJSON,
			&event.AlertCreated,
			&alertID,
			&event.DetectedAt,
		)

		if err != nil {
			continue
		}

		if honeypotID.Valid {
			event.HoneypotID = honeypotID.String
		}
		if honeyTokenID.Valid {
			event.HoneyTokenID = honeyTokenID.String
		}
		if sourceHostname.Valid {
			event.SourceHostname = sourceHostname.String
		}
		if sourceUser.Valid {
			event.SourceUser = sourceUser.String
		}
		if alertID.Valid {
			event.AlertID = alertID.String
		}

		json.Unmarshal(detailsJSON, &event.Details)
		events = append(events, event)
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"count":  len(events),
	})
}

// GetDeceptionStatistics retrieves statistics about deception deployments
func (h *DeceptionHandler) GetDeceptionStatistics(c *gin.Context) {
	licenseID := c.Query("license_id")

	stats := models.DeceptionStatistics{
		LicenseID: licenseID,
	}

	// Honeypot statistics
	h.db.QueryRow(`
		SELECT COUNT(*),
		       COUNT(CASE WHEN is_active = TRUE THEN 1 END),
		       COUNT(CASE WHEN status = 'compromised' THEN 1 END)
		FROM honeypots
		WHERE license_id = $1
	`, licenseID).Scan(&stats.TotalHoneypots, &stats.ActiveHoneypots, &stats.CompromisedHoneypots)

	// Honey token statistics
	h.db.QueryRow(`
		SELECT COUNT(*),
		       COUNT(CASE WHEN is_active = TRUE THEN 1 END)
		FROM honey_tokens
		WHERE license_id = $1
	`, licenseID).Scan(&stats.TotalHoneyTokens, &stats.ActiveHoneyTokens)

	// Event statistics
	h.db.QueryRow(`
		SELECT COUNT(*),
		       COUNT(CASE WHEN detected_at > NOW() - INTERVAL '24 hours' THEN 1 END),
		       COUNT(CASE WHEN detected_at > NOW() - INTERVAL '7 days' THEN 1 END),
		       COUNT(DISTINCT source_ip)
		FROM deception_events
		WHERE license_id = $1
	`, licenseID).Scan(&stats.TotalEvents, &stats.Events24h, &stats.Events7d, &stats.UniqueSourceIPs)

	// Calculate threat score (0-100)
	stats.ThreatScore = float64(stats.Events7d) * 2.5
	if stats.ThreatScore > 100 {
		stats.ThreatScore = 100
	}

	c.JSON(http.StatusOK, stats)
}

// ListHoneypotTemplates lists available honeypot templates
func (h *DeceptionHandler) ListHoneypotTemplates(c *gin.Context) {
	// In production, load from database
	templates := []models.HoneypotTemplate{
		{
			ID:             "ssh-linux",
			Name:           "SSH Honeypot (Linux)",
			Description:    "Simulates a Linux SSH server",
			HoneypotType:   models.HoneypotTypeSSH,
			TargetPlatform: "linux",
			DifficultyLevel: "medium",
			Configuration: models.HoneypotConfiguration{
				ListenPort:         22,
				ServiceBanner:      "OpenSSH_7.4 Ubuntu",
				LogAllInteractions: true,
				AlertOnInteraction: true,
			},
			IsPopular:   true,
			UseCount:    156,
			SuccessRate: 0.78,
		},
		{
			ID:             "smb-windows",
			Name:           "SMB File Share (Windows)",
			Description:    "Simulates a Windows file share",
			HoneypotType:   models.HoneypotTypeSMB,
			TargetPlatform: "windows",
			DifficultyLevel: "high",
			Configuration: models.HoneypotConfiguration{
				ListenPort:         445,
				ServiceBanner:      "Windows Server 2019",
				LogAllInteractions: true,
				AlertOnInteraction: true,
			},
			IsPopular:   true,
			UseCount:    203,
			SuccessRate: 0.82,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"templates": templates,
		"count":     len(templates),
	})
}

// Helper functions

func (h *DeceptionHandler) generateHoneyToken(tokenType models.HoneyTokenType) string {
	switch tokenType {
	case models.TokenTypeAWSKey:
		return fmt.Sprintf("AKIA%s", h.randomString(16))
	case models.TokenTypeAPIKey:
		return h.randomString(32)
	case models.TokenTypeDatabaseCreds:
		return fmt.Sprintf("user:honey_%s", h.randomString(12))
	case models.TokenTypeDNSQuery:
		return fmt.Sprintf("%s.canarytoken.com", h.randomString(16))
	default:
		return h.randomString(24)
	}
}

func (h *DeceptionHandler) randomString(length int) string {
	bytes := make([]byte, length/2)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
