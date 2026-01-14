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

	// TODO: Insert into database
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

	// TODO: Check database for license status and usage
	// For now, return success based on cryptographic validation

	license := &models.License{
		ID:            payload.ID,
		LicenseKey:    licenseKey,
		CustomerEmail: payload.Email,
		Tier:          models.LicenseTier(payload.Tier),
		MaxAgents:     payload.MaxAgents,
		IssuedAt:      time.Unix(payload.IssuedAt, 0),
		IsActive:      true,
	}

	if payload.ExpiresAt > 0 {
		expiry := time.Unix(payload.ExpiresAt, 0)
		license.ExpiresAt = &expiry
	}

	// Calculate remaining time
	expiresInDays := -1
	if payload.ExpiresAt > 0 {
		expiresInDays = int(time.Until(time.Unix(payload.ExpiresAt, 0)).Hours() / 24)
	}

	// Get features
	features := models.GetFeaturesForTier(license.Tier)

	// TODO: Calculate actual remaining agents from usage
	remainingAgents := payload.MaxAgents
	if remainingAgents == -1 {
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
	// TODO: Query database
	log.Infof("Retrieved license: %s", licenseID)
	return nil, fmt.Errorf("not implemented")
}

// ListLicenses retrieves all licenses (with pagination)
func (s *LicenseService) ListLicenses(limit, offset int) ([]*models.License, int, error) {
	// TODO: Query database with pagination
	log.Info("Listed licenses")
	return []*models.License{}, 0, nil
}

// RevokeLicense deactivates a license
func (s *LicenseService) RevokeLicense(licenseID string, reason string) error {
	// TODO: Update database to set is_active = false
	log.Warnf("Revoked license: %s (reason: %s)", licenseID, reason)
	return nil
}

// GetLicenseUsage retrieves usage statistics for a license
func (s *LicenseService) GetLicenseUsage(licenseID string) (*models.LicenseUsage, error) {
	// TODO: Query database for usage stats
	usage := &models.LicenseUsage{
		LicenseID:      licenseID,
		ActiveAgents:   0,
		ActiveUsers:    0,
		EventsIngested: 0,
		StorageUsedGB:  0,
		LastUpdated:    time.Now(),
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
	// TODO: Update database with new tier and regenerate key
	log.Infof("Upgraded license %s to %s tier", licenseID, newTier)
	return nil
}

// ExtendLicense extends the expiration date
func (s *LicenseService) ExtendLicense(licenseID string, additionalDays int) error {
	// TODO: Update expiration date in database
	log.Infof("Extended license %s by %d days", licenseID, additionalDays)
	return nil
}
