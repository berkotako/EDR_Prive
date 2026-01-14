// License Management API Handlers

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/sentinel-enterprise/platform/license/models"
)

// CreateLicense generates a new license key
func CreateLicense(c *gin.Context) {
	var req models.CreateLicenseRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Call license service
	license := &models.License{
		ID:            "LIC-" + generateID(),
		LicenseKey:    "PRIVE-V1-XXXXX-XXXXX-XXXXX-XXXXX",
		CustomerEmail: req.CustomerEmail,
		CustomerName:  req.CustomerName,
		CompanyName:   req.CompanyName,
		Tier:          req.Tier,
	}

	maxAgents, maxUsers := models.GetLimitsForTier(req.Tier)
	license.MaxAgents = maxAgents
	license.MaxUsers = maxUsers

	log.Infof("Created license for %s (%s tier)", req.CustomerEmail, req.Tier)

	c.JSON(http.StatusCreated, gin.H{
		"license": license,
		"message": "License created successfully",
	})
}

// ValidateLicense checks if a license key is valid
func ValidateLicense(c *gin.Context) {
	var req models.ValidateLicenseRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Call license service for validation
	features := models.GetFeaturesForTier(models.TierPro)

	response := models.ValidateLicenseResponse{
		Valid:           true,
		Features:        features,
		RemainingAgents: 95,
		ExpiresInDays:   365,
		Message:         "License is valid and active",
	}

	c.JSON(http.StatusOK, response)
}

// ListLicenses retrieves all licenses
func ListLicenses(c *gin.Context) {
	// TODO: Implement pagination
	licenses := []models.License{
		{
			ID:            "LIC-001",
			CustomerEmail: "demo@example.com",
			CustomerName:  "Demo Customer",
			CompanyName:   "Acme Corp",
			Tier:          models.TierEnterprise,
			MaxAgents:     -1, // Unlimited
			MaxUsers:      -1,
			IsActive:      true,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"licenses": licenses,
		"total":    len(licenses),
	})
}

// GetLicense retrieves a specific license
func GetLicense(c *gin.Context) {
	licenseID := c.Param("id")

	// TODO: Query database
	license := models.License{
		ID:            licenseID,
		CustomerEmail: "customer@example.com",
		Tier:          models.TierPro,
		IsActive:      true,
	}

	c.JSON(http.StatusOK, license)
}

// RevokeLicense deactivates a license
func RevokeLicense(c *gin.Context) {
	licenseID := c.Param("id")

	// TODO: Call service to revoke
	log.Warnf("Revoked license: %s", licenseID)

	c.JSON(http.StatusOK, gin.H{
		"message": "License revoked successfully",
	})
}

// GenerateTrialLicense creates a trial license
func GenerateTrialLicense(c *gin.Context) {
	type TrialRequest struct {
		Email string `json:"email" binding:"required,email"`
		Name  string `json:"name" binding:"required"`
	}

	var req TrialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Generate trial license
	license := &models.License{
		ID:            "TRIAL-" + generateID(),
		LicenseKey:    "PRIVE-V1-TRIAL-XXXXX-XXXXX",
		CustomerEmail: req.Email,
		CustomerName:  req.Name,
		Tier:          models.TierPro,
		MaxAgents:     10,
		MaxUsers:      3,
		IsActive:      true,
	}

	c.JSON(http.StatusCreated, gin.H{
		"license": license,
		"message": "14-day trial license created",
	})
}

// GetLicenseUsage returns usage statistics
func GetLicenseUsage(c *gin.Context) {
	licenseID := c.Param("id")

	usage := models.LicenseUsage{
		LicenseID:      licenseID,
		ActiveAgents:   45,
		ActiveUsers:    8,
		EventsIngested: 12500000,
		StorageUsedGB:  125.5,
	}

	c.JSON(http.StatusOK, usage)
}

func generateID() string {
	// Simple ID generation (replace with UUID in production)
	return "XXXXX"
}
