// License Management API Handlers

package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/sentinel-enterprise/platform/license/models"
	"github.com/sentinel-enterprise/platform/license/service"
)

// LicenseHandler handles license-related requests
type LicenseHandler struct {
	service *service.LicenseService
}

// NewLicenseHandler creates a new license handler
func NewLicenseHandler(service *service.LicenseService) *LicenseHandler {
	return &LicenseHandler{
		service: service,
	}
}

// CreateLicense generates a new license key
func (h *LicenseHandler) CreateLicense(c *gin.Context) {
	var req models.CreateLicenseRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "License service not available"})
		return
	}

	license, err := h.service.CreateLicense(req)
	if err != nil {
		log.Errorf("Failed to create license: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Infof("Created license for %s (%s tier)", req.CustomerEmail, req.Tier)

	c.JSON(http.StatusCreated, gin.H{
		"license": license,
		"message": "License created successfully",
	})
}

// ValidateLicense checks if a license key is valid
func (h *LicenseHandler) ValidateLicense(c *gin.Context) {
	var req models.ValidateLicenseRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "License service not available"})
		return
	}

	response, err := h.service.ValidateLicense(req.LicenseKey, req.AgentID)
	if err != nil {
		log.Errorf("Failed to validate license: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !response.Valid {
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

// ListLicenses retrieves all licenses
func (h *LicenseHandler) ListLicenses(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "License service not available"})
		return
	}

	// Parse pagination parameters
	limit := 50
	offset := 0

	licenses, total, err := h.service.ListLicenses(limit, offset)
	if err != nil {
		log.Errorf("Failed to list licenses: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"licenses": licenses,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetLicense retrieves a specific license
func (h *LicenseHandler) GetLicense(c *gin.Context) {
	licenseID := c.Param("id")

	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "License service not available"})
		return
	}

	license, err := h.service.GetLicense(licenseID)
	if err != nil {
		log.Errorf("Failed to get license: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "License not found"})
		return
	}

	c.JSON(http.StatusOK, license)
}

// RevokeLicense deactivates a license
func (h *LicenseHandler) RevokeLicense(c *gin.Context) {
	licenseID := c.Param("id")

	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "License service not available"})
		return
	}

	err := h.service.RevokeLicense(licenseID, "Revoked via API")
	if err != nil {
		log.Errorf("Failed to revoke license: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Warnf("Revoked license: %s", licenseID)

	c.JSON(http.StatusOK, gin.H{
		"message": "License revoked successfully",
	})
}

// GenerateTrialLicense creates a trial license
func (h *LicenseHandler) GenerateTrialLicense(c *gin.Context) {
	type TrialRequest struct {
		Email string `json:"email" binding:"required,email"`
		Name  string `json:"name" binding:"required"`
	}

	var req TrialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "License service not available"})
		return
	}

	license, err := h.service.GenerateTrialLicense(req.Email, req.Name)
	if err != nil {
		log.Errorf("Failed to generate trial license: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"license": license,
		"message": "14-day trial license created",
	})
}

// GetLicenseUsage returns usage statistics
func (h *LicenseHandler) GetLicenseUsage(c *gin.Context) {
	licenseID := c.Param("id")

	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "License service not available"})
		return
	}

	usage, err := h.service.GetLicenseUsage(licenseID)
	if err != nil {
		log.Errorf("Failed to get license usage: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, usage)
}
