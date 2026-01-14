// License Service - Business logic for license management

package service

import (
	"crypto/ed25519"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/sentinel-enterprise/platform/license/crypto"
	"github.com/sentinel-enterprise/platform/license/models"
)

// LicenseService handles license operations
type LicenseService struct {
	db         *sql.DB
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
}

// NewLicenseService creates a new license service
func NewLicenseService(db *sql.DB, privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey) *LicenseService {
	return &LicenseService{
		db:         db,
		privateKey: privateKey,
		publicKey:  publicKey,
	}
}

// CreateLicense generates a new license
func (s *LicenseService) CreateLicense(req models.CreateLicenseRequest) (*models.License, error) {
	// Generate license ID
	licenseID := uuid.New().String()

	// Get tier limits
	maxAgents, maxUsers := models.GetLimitsForTier(req.Tier)

	// Calculate expiration
	var expiresAt *time.Time
	if req.DurationDays > 0 {
		expiry := time.Now().AddDate(0, 0, req.DurationDays)
		expiresAt = &expiry
	}

	// Create cryptographic payload
	payload := crypto.LicensePayload{
		ID:        licenseID,
		Email:     req.CustomerEmail,
		Tier:      string(req.Tier),
		IssuedAt:  time.Now().Unix(),
		MaxAgents: maxAgents,
	}

	if expiresAt != nil {
		payload.ExpiresAt = expiresAt.Unix()
	}

	// Generate signed license key
	licenseKey, err := crypto.GenerateLicenseKey(payload, s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate license key: %w", err)
	}

	// Get features for tier
	features := models.GetFeaturesForTier(req.Tier)
	featuresJSON, _ := json.Marshal(features)

	// Create license record
	license := &models.License{
		ID:            licenseID,
		LicenseKey:    licenseKey,
		CustomerEmail: req.CustomerEmail,
		CustomerName:  req.CustomerName,
		CompanyName:   req.CompanyName,
		Tier:          req.Tier,
		MaxAgents:     maxAgents,
		MaxUsers:      maxUsers,
		IssuedAt:      time.Now(),
		ExpiresAt:     expiresAt,
		IsActive:      true,
		Metadata:      string(featuresJSON),
	}

	// Insert into database
	query := `
		INSERT INTO licenses (
			id, license_key, customer_email, customer_name, company_name,
			tier, max_agents, max_users, issued_at, expires_at, is_active, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err = s.db.Exec(query,
		licenseID,
		licenseKey,
		req.CustomerEmail,
		req.CustomerName,
		req.CompanyName,
		string(req.Tier),
		maxAgents,
		maxUsers,
		license.IssuedAt,
		expiresAt,
		true,
		string(featuresJSON),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to insert license into database: %w", err)
	}

	// Initialize license usage record
	usageQuery := `
		INSERT INTO license_usage (license_id, active_agents, active_users, events_ingested, storage_used_gb)
		VALUES ($1, 0, 0, 0, 0)
	`
	_, err = s.db.Exec(usageQuery, licenseID)
	if err != nil {
		log.Warnf("Failed to initialize license usage record: %v", err)
	}

	log.Infof("Created license: %s for %s (%s tier)", licenseID, req.CustomerEmail, req.Tier)

	return license, nil
}

// ValidateLicense checks if a license key is valid
func (s *LicenseService) ValidateLicense(licenseKey string, agentID string) (*models.ValidateLicenseResponse, error) {
	// Cryptographically validate the key
	payload, err := crypto.ValidateLicenseKey(licenseKey, s.publicKey)
	if err != nil {
		return &models.ValidateLicenseResponse{
			Valid:   false,
			Message: fmt.Sprintf("Invalid license: %v", err),
		}, nil
	}

	// Check database for license status and usage
	var isActive bool
	var dbExpiresAt *time.Time

	query := `
		SELECT is_active, expires_at
		FROM licenses
		WHERE id = $1
	`
	err = s.db.QueryRow(query, payload.ID).Scan(&isActive, &dbExpiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return &models.ValidateLicenseResponse{
				Valid:   false,
				Message: "License not found in database",
			}, nil
		}
		log.Errorf("Database error checking license: %v", err)
		// Continue with cryptographic validation if DB fails
		isActive = true
	}

	if !isActive {
		return &models.ValidateLicenseResponse{
			Valid:   false,
			Message: "License has been revoked",
		}, nil
	}

	license := &models.License{
		ID:            payload.ID,
		LicenseKey:    licenseKey,
		CustomerEmail: payload.Email,
		Tier:          models.LicenseTier(payload.Tier),
		MaxAgents:     payload.MaxAgents,
		IssuedAt:      time.Unix(payload.IssuedAt, 0),
		IsActive:      isActive,
	}

	if payload.ExpiresAt > 0 {
		expiry := time.Unix(payload.ExpiresAt, 0)
		license.ExpiresAt = &expiry
	}

	// Calculate remaining time
	expiresInDays := -1
	if payload.ExpiresAt > 0 {
		expiresInDays = int(time.Until(time.Unix(payload.ExpiresAt, 0)).Hours() / 24)
		if expiresInDays <= 0 {
			return &models.ValidateLicenseResponse{
				Valid:   false,
				Message: "License has expired",
			}, nil
		}
	}

	// Get features
	features := models.GetFeaturesForTier(license.Tier)

	// Calculate actual remaining agents from usage
	var activeAgents int
	usageQuery := `
		SELECT active_agents
		FROM license_usage
		WHERE license_id = $1
	`
	err = s.db.QueryRow(usageQuery, payload.ID).Scan(&activeAgents)
	if err != nil && err != sql.ErrNoRows {
		log.Warnf("Failed to get agent usage: %v", err)
		activeAgents = 0
	}

	remainingAgents := payload.MaxAgents - activeAgents
	if payload.MaxAgents == -1 {
		remainingAgents = 999999 // Unlimited
	}

	response := &models.ValidateLicenseResponse{
		Valid:           true,
		License:         license,
		Features:        features,
		RemainingAgents: remainingAgents,
		ExpiresInDays:   expiresInDays,
		Message:         "License valid",
	}

	// Update last validated timestamp
	now := time.Now()
	license.LastValidatedAt = &now

	log.Infof("License validated: %s (%s tier, agent: %s)", payload.ID, payload.Tier, agentID)

	return response, nil
}

// GetLicense retrieves license by ID
func (s *LicenseService) GetLicense(licenseID string) (*models.License, error) {
	query := `
		SELECT id, license_key, customer_email, customer_name, company_name,
		       tier, max_agents, max_users, issued_at, expires_at, is_active,
		       activated_at, last_validated_at, metadata, created_at, updated_at
		FROM licenses
		WHERE id = $1
	`

	license := &models.License{}
	var expiresAt, activatedAt, lastValidatedAt, updatedAt sql.NullTime

	err := s.db.QueryRow(query, licenseID).Scan(
		&license.ID,
		&license.LicenseKey,
		&license.CustomerEmail,
		&license.CustomerName,
		&license.CompanyName,
		&license.Tier,
		&license.MaxAgents,
		&license.MaxUsers,
		&license.IssuedAt,
		&expiresAt,
		&license.IsActive,
		&activatedAt,
		&lastValidatedAt,
		&license.Metadata,
		&license.CreatedAt,
		&updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("license not found")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Handle nullable timestamps
	if expiresAt.Valid {
		license.ExpiresAt = &expiresAt.Time
	}
	if activatedAt.Valid {
		license.ActivatedAt = &activatedAt.Time
	}
	if lastValidatedAt.Valid {
		license.LastValidatedAt = &lastValidatedAt.Time
	}
	if updatedAt.Valid {
		license.UpdatedAt = &updatedAt.Time
	}

	log.Infof("Retrieved license: %s", licenseID)
	return license, nil
}

// ListLicenses retrieves all licenses (with pagination)
func (s *LicenseService) ListLicenses(limit, offset int) ([]*models.License, int, error) {
	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM licenses`
	err := s.db.QueryRow(countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count licenses: %w", err)
	}

	// Get licenses with pagination
	query := `
		SELECT id, license_key, customer_email, customer_name, company_name,
		       tier, max_agents, max_users, issued_at, expires_at, is_active,
		       activated_at, last_validated_at, metadata, created_at, updated_at
		FROM licenses
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query licenses: %w", err)
	}
	defer rows.Close()

	licenses := make([]*models.License, 0)
	for rows.Next() {
		license := &models.License{}
		var expiresAt, activatedAt, lastValidatedAt, updatedAt sql.NullTime

		err := rows.Scan(
			&license.ID,
			&license.LicenseKey,
			&license.CustomerEmail,
			&license.CustomerName,
			&license.CompanyName,
			&license.Tier,
			&license.MaxAgents,
			&license.MaxUsers,
			&license.IssuedAt,
			&expiresAt,
			&license.IsActive,
			&activatedAt,
			&lastValidatedAt,
			&license.Metadata,
			&license.CreatedAt,
			&updatedAt,
		)

		if err != nil {
			log.Warnf("Failed to scan license: %v", err)
			continue
		}

		// Handle nullable timestamps
		if expiresAt.Valid {
			license.ExpiresAt = &expiresAt.Time
		}
		if activatedAt.Valid {
			license.ActivatedAt = &activatedAt.Time
		}
		if lastValidatedAt.Valid {
			license.LastValidatedAt = &lastValidatedAt.Time
		}
		if updatedAt.Valid {
			license.UpdatedAt = &updatedAt.Time
		}

		licenses = append(licenses, license)
	}

	log.Infof("Listed %d licenses (total: %d)", len(licenses), total)
	return licenses, total, nil
}

// RevokeLicense deactivates a license
func (s *LicenseService) RevokeLicense(licenseID string, reason string) error {
	// Update database to set is_active = false
	query := `
		UPDATE licenses
		SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1
	`

	result, err := s.db.Exec(query, licenseID)
	if err != nil {
		return fmt.Errorf("failed to revoke license: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("license not found")
	}

	// Insert audit log entry
	auditQuery := `
		INSERT INTO license_audit_log (license_id, action, details, created_at)
		VALUES ($1, 'revoked', $2, NOW())
	`
	details := fmt.Sprintf(`{"reason": "%s"}`, reason)
	_, err = s.db.Exec(auditQuery, licenseID, details)
	if err != nil {
		log.Warnf("Failed to insert audit log: %v", err)
	}

	log.Warnf("Revoked license: %s (reason: %s)", licenseID, reason)
	return nil
}

// GetLicenseUsage retrieves usage statistics for a license
func (s *LicenseService) GetLicenseUsage(licenseID string) (*models.LicenseUsage, error) {
	query := `
		SELECT license_id, active_agents, active_users, events_ingested,
		       storage_used_gb, last_updated
		FROM license_usage
		WHERE license_id = $1
	`

	usage := &models.LicenseUsage{}
	err := s.db.QueryRow(query, licenseID).Scan(
		&usage.LicenseID,
		&usage.ActiveAgents,
		&usage.ActiveUsers,
		&usage.EventsIngested,
		&usage.StorageUsedGB,
		&usage.LastUpdated,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Return empty usage if not found
			return &models.LicenseUsage{
				LicenseID:      licenseID,
				ActiveAgents:   0,
				ActiveUsers:    0,
				EventsIngested: 0,
				StorageUsedGB:  0,
				LastUpdated:    time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to get license usage: %w", err)
	}

	return usage, nil
}

// GenerateTrialLicense creates a 14-day trial license
func (s *LicenseService) GenerateTrialLicense(email, name string) (*models.License, error) {
	req := models.CreateLicenseRequest{
		CustomerEmail: email,
		CustomerName:  name,
		Tier:          models.TierPro,
		DurationDays:  14,
	}

	return s.CreateLicense(req)
}

// UpgradeLicense upgrades an existing license to a higher tier
func (s *LicenseService) UpgradeLicense(licenseID string, newTier models.LicenseTier) error {
	// Get new limits for tier
	maxAgents, maxUsers := models.GetLimitsForTier(newTier)

	// Update database with new tier and limits
	query := `
		UPDATE licenses
		SET tier = $1, max_agents = $2, max_users = $3, updated_at = NOW()
		WHERE id = $4 AND is_active = TRUE
	`

	result, err := s.db.Exec(query, string(newTier), maxAgents, maxUsers, licenseID)
	if err != nil {
		return fmt.Errorf("failed to upgrade license: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("license not found or inactive")
	}

	// Insert audit log entry
	auditQuery := `
		INSERT INTO license_audit_log (license_id, action, details, created_at)
		VALUES ($1, 'upgraded', $2, NOW())
	`
	details := fmt.Sprintf(`{"new_tier": "%s", "max_agents": %d, "max_users": %d}`, newTier, maxAgents, maxUsers)
	_, err = s.db.Exec(auditQuery, licenseID, details)
	if err != nil {
		log.Warnf("Failed to insert audit log: %v", err)
	}

	log.Infof("Upgraded license %s to %s tier", licenseID, newTier)
	return nil
}

// ExtendLicense extends the expiration date
func (s *LicenseService) ExtendLicense(licenseID string, additionalDays int) error {
	// Update expiration date in database
	query := `
		UPDATE licenses
		SET expires_at = COALESCE(expires_at, NOW()) + INTERVAL '1 day' * $1,
		    updated_at = NOW()
		WHERE id = $2 AND is_active = TRUE
	`

	result, err := s.db.Exec(query, additionalDays, licenseID)
	if err != nil {
		return fmt.Errorf("failed to extend license: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("license not found or inactive")
	}

	// Insert audit log entry
	auditQuery := `
		INSERT INTO license_audit_log (license_id, action, details, created_at)
		VALUES ($1, 'extended', $2, NOW())
	`
	details := fmt.Sprintf(`{"additional_days": %d}`, additionalDays)
	_, err = s.db.Exec(auditQuery, licenseID, details)
	if err != nil {
		log.Warnf("Failed to insert audit log: %v", err)
	}

	log.Infof("Extended license %s by %d days", licenseID, additionalDays)
	return nil
}
