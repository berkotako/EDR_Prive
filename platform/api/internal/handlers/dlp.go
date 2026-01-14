// DLP Policy Management Handlers with PostgreSQL Integration

package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/sentinel-enterprise/platform/api/internal/models"
)

// DLPHandler handles DLP policy management requests
type DLPHandler struct {
	db *sql.DB
}

// NewDLPHandler creates a new DLP handler
func NewDLPHandler(db *sql.DB) *DLPHandler {
	return &DLPHandler{
		db: db,
	}
}

// ListDLPPolicies retrieves all DLP policies for a tenant
func (h *DLPHandler) ListDLPPolicies(c *gin.Context) {
	licenseID := c.Query("license_id")
	if licenseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "license_id required"})
		return
	}

	query := `
		SELECT id, license_id, name, description, severity, enabled, rule_type,
		       config, fingerprint_count, created_at, updated_at
		FROM dlp_policies
		WHERE license_id = $1
		ORDER BY created_at DESC
	`

	rows, err := h.db.Query(query, licenseID)
	if err != nil {
		log.Errorf("Failed to query DLP policies: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}
	defer rows.Close()

	policies := make([]models.DLPPolicy, 0)
	for rows.Next() {
		var policy models.DLPPolicy
		var configJSON []byte

		err := rows.Scan(
			&policy.ID,
			&policy.TenantID,
			&policy.Name,
			&policy.Description,
			&policy.Severity,
			&policy.Enabled,
			&policy.RuleType,
			&configJSON,
			&policy.FingerprintCount,
			&policy.CreatedAt,
			&policy.UpdatedAt,
		)

		if err != nil {
			log.Warnf("Failed to scan policy: %v", err)
			continue
		}

		// Parse JSON config
		if len(configJSON) > 0 {
			json.Unmarshal(configJSON, &policy.Config)
		}

		policies = append(policies, policy)
	}

	c.JSON(http.StatusOK, gin.H{
		"policies": policies,
		"total":    len(policies),
	})
}

// GetDLPPolicy retrieves a specific DLP policy by ID
func (h *DLPHandler) GetDLPPolicy(c *gin.Context) {
	policyID := c.Param("id")

	query := `
		SELECT id, license_id, name, description, severity, enabled, rule_type,
		       config, fingerprint_count, created_at, updated_at
		FROM dlp_policies
		WHERE id = $1
	`

	var policy models.DLPPolicy
	var configJSON []byte

	err := h.db.QueryRow(query, policyID).Scan(
		&policy.ID,
		&policy.TenantID,
		&policy.Name,
		&policy.Description,
		&policy.Severity,
		&policy.Enabled,
		&policy.RuleType,
		&configJSON,
		&policy.FingerprintCount,
		&policy.CreatedAt,
		&policy.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
			return
		}
		log.Errorf("Failed to query DLP policy: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}

	// Parse JSON config
	if len(configJSON) > 0 {
		json.Unmarshal(configJSON, &policy.Config)
	}

	c.JSON(http.StatusOK, policy)
}

// CreateDLPPolicy creates a new DLP policy
func (h *DLPHandler) CreateDLPPolicy(c *gin.Context) {
	var req models.CreateDLPPolicyRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate request
	if req.Name == "" || req.TenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and tenant_id required"})
		return
	}

	// Validate license exists
	var licenseExists bool
	err := h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM licenses WHERE id = $1 AND is_active = TRUE)", req.TenantID).Scan(&licenseExists)
	if err != nil || !licenseExists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid license_id"})
		return
	}

	// Generate policy ID
	policyID := uuid.New().String()

	// Serialize config to JSON
	configJSON, _ := json.Marshal(req.Config)

	query := `
		INSERT INTO dlp_policies (id, license_id, name, description, severity, enabled, rule_type, config, fingerprint_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	var createdAt, updatedAt time.Time
	err = h.db.QueryRow(query,
		policyID,
		req.TenantID,
		req.Name,
		req.Description,
		req.Severity,
		req.Enabled,
		req.RuleType,
		string(configJSON),
	).Scan(&policyID, &createdAt, &updatedAt)

	if err != nil {
		log.Errorf("Failed to create DLP policy: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create policy"})
		return
	}

	policy := models.DLPPolicy{
		ID:          policyID,
		TenantID:    req.TenantID,
		Name:        req.Name,
		Description: req.Description,
		Severity:    req.Severity,
		Enabled:     req.Enabled,
		RuleType:    req.RuleType,
		Config:      req.Config,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}

	log.Infof("Created DLP policy: %s (%s)", policy.Name, policy.ID)

	c.JSON(http.StatusCreated, policy)
}

// UpdateDLPPolicy updates an existing DLP policy
func (h *DLPHandler) UpdateDLPPolicy(c *gin.Context) {
	policyID := c.Param("id")

	var req models.UpdateDLPPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build dynamic update query
	query := `
		UPDATE dlp_policies
		SET updated_at = NOW()
	`
	args := []interface{}{}
	argCount := 1

	if req.Name != nil {
		query += `, name = $` + string(rune('0'+argCount))
		args = append(args, *req.Name)
		argCount++
	}
	if req.Description != nil {
		query += `, description = $` + string(rune('0'+argCount))
		args = append(args, *req.Description)
		argCount++
	}
	if req.Severity != nil {
		query += `, severity = $` + string(rune('0'+argCount))
		args = append(args, *req.Severity)
		argCount++
	}
	if req.Enabled != nil {
		query += `, enabled = $` + string(rune('0'+argCount))
		args = append(args, *req.Enabled)
		argCount++
	}
	if req.Config != nil {
		configJSON, _ := json.Marshal(req.Config)
		query += `, config = $` + string(rune('0'+argCount))
		args = append(args, string(configJSON))
		argCount++
	}

	query += ` WHERE id = $` + string(rune('0'+argCount))
	args = append(args, policyID)

	result, err := h.db.Exec(query, args...)
	if err != nil {
		log.Errorf("Failed to update DLP policy: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update policy"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
		return
	}

	log.Infof("Updated DLP policy: %s", policyID)

	c.JSON(http.StatusOK, gin.H{
		"id":         policyID,
		"updated_at": time.Now(),
		"message":    "Policy updated successfully",
	})
}

// DeleteDLPPolicy deletes a DLP policy
func (h *DLPHandler) DeleteDLPPolicy(c *gin.Context) {
	policyID := c.Param("id")

	query := `DELETE FROM dlp_policies WHERE id = $1`

	result, err := h.db.Exec(query, policyID)
	if err != nil {
		log.Errorf("Failed to delete DLP policy: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete policy"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
		return
	}

	log.Infof("Deleted DLP policy: %s", policyID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Policy deleted successfully",
	})
}

// AddFingerprints adds fingerprints to a DLP policy
func (h *DLPHandler) AddFingerprints(c *gin.Context) {
	policyID := c.Param("id")

	var req models.AddFingerprintsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Begin transaction
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Insert fingerprints
	insertQuery := `
		INSERT INTO dlp_fingerprints (id, policy_id, fingerprint_hash, source, created_at)
		VALUES ($1, $2, $3, $4, NOW())
	`

	for _, fp := range req.Fingerprints {
		_, err := tx.Exec(insertQuery,
			uuid.New().String(),
			policyID,
			fp.Hash,
			fp.Source,
		)
		if err != nil {
			log.Errorf("Failed to insert fingerprint: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add fingerprints"})
			return
		}
	}

	// Update fingerprint count
	updateQuery := `
		UPDATE dlp_policies
		SET fingerprint_count = fingerprint_count + $1, updated_at = NOW()
		WHERE id = $2
	`
	_, err = tx.Exec(updateQuery, len(req.Fingerprints), policyID)
	if err != nil {
		log.Errorf("Failed to update fingerprint count: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update policy"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	log.Infof("Added %d fingerprints to policy %s", len(req.Fingerprints), policyID)

	c.JSON(http.StatusCreated, gin.H{
		"policy_id": policyID,
		"added":     len(req.Fingerprints),
		"message":   "Fingerprints added successfully",
	})
}

// DeleteFingerprint removes a fingerprint from a policy
func (h *DLPHandler) DeleteFingerprint(c *gin.Context) {
	policyID := c.Param("id")
	fingerprintID := c.Param("fingerprint_id")

	// Begin transaction
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Delete fingerprint
	deleteQuery := `DELETE FROM dlp_fingerprints WHERE id = $1 AND policy_id = $2`
	result, err := tx.Exec(deleteQuery, fingerprintID, policyID)
	if err != nil {
		log.Errorf("Failed to delete fingerprint: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete fingerprint"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fingerprint not found"})
		return
	}

	// Update fingerprint count
	updateQuery := `
		UPDATE dlp_policies
		SET fingerprint_count = fingerprint_count - 1, updated_at = NOW()
		WHERE id = $1
	`
	_, err = tx.Exec(updateQuery, policyID)
	if err != nil {
		log.Errorf("Failed to update fingerprint count: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update policy"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	log.Infof("Deleted fingerprint %s from policy %s", fingerprintID, policyID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Fingerprint deleted successfully",
	})
}

// TestDLPPolicy tests a DLP policy against sample data
func (h *DLPHandler) TestDLPPolicy(c *gin.Context) {
	var req models.TestDLPPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get policy from database
	query := `
		SELECT id, name, severity, rule_type, config
		FROM dlp_policies
		WHERE id = $1
	`

	var policyID, name, severity, ruleType string
	var configJSON []byte

	err := h.db.QueryRow(query, req.PolicyID).Scan(&policyID, &name, &severity, &ruleType, &configJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// For now, return mock results (in production, this would run actual DLP scan)
	// TODO: Integrate with actual DLP engine from agent code
	results := models.TestDLPPolicyResponse{
		Matches: []models.DLPMatch{
			{
				PolicyID:   policyID,
				PolicyName: name,
				Offset:     42,
				Length:     11,
				Confidence: 0.95,
				MatchType:  "exact",
			},
		},
		ScanDurationMs: 15,
		DataSizeBytes:  len(req.TestData),
	}

	c.JSON(http.StatusOK, results)
}
