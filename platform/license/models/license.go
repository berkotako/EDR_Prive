// License Models for Privé Platform

package models

import "time"

// LicenseTier defines the subscription level
type LicenseTier string

const (
	TierFree       LicenseTier = "free"
	TierPro        LicenseTier = "professional"
	TierEnterprise LicenseTier = "enterprise"
)

// License represents a software license for Privé
type License struct {
	ID              string      `json:"id" db:"id"`
	LicenseKey      string      `json:"license_key" db:"license_key"`
	CustomerEmail   string      `json:"customer_email" db:"customer_email"`
	CustomerName    string      `json:"customer_name" db:"customer_name"`
	CompanyName     string      `json:"company_name" db:"company_name"`
	Tier            LicenseTier `json:"tier" db:"tier"`
	MaxAgents       int         `json:"max_agents" db:"max_agents"`
	MaxUsers        int         `json:"max_users" db:"max_users"`
	Features        []string    `json:"features" db:"-"`
	IssuedAt        time.Time   `json:"issued_at" db:"issued_at"`
	ExpiresAt       *time.Time  `json:"expires_at" db:"expires_at"`
	IsActive        bool        `json:"is_active" db:"is_active"`
	ActivatedAt     *time.Time  `json:"activated_at" db:"activated_at"`
	LastValidatedAt *time.Time  `json:"last_validated_at" db:"last_validated_at"`
	Metadata        string      `json:"metadata" db:"metadata"` // JSON-encoded map
}

// LicenseFeatures defines feature sets per tier
type LicenseFeatures struct {
	EDRMonitoring        bool `json:"edr_monitoring"`
	DLPProtection        bool `json:"dlp_protection"`
	ThreatHunting        bool `json:"threat_hunting"`
	RealTimeAlerting     bool `json:"real_time_alerting"`
	CustomRules          bool `json:"custom_rules"`
	APIAccess            bool `json:"api_access"`
	MultiTenancy         bool `json:"multi_tenancy"`
	AdvancedAnalytics    bool `json:"advanced_analytics"`
	ThreatIntelligence   bool `json:"threat_intelligence"`
	IncidentResponse     bool `json:"incident_response"`
	ComplianceReporting  bool `json:"compliance_reporting"`
	PrioritySupport      bool `json:"priority_support"`
	CustomIntegrations   bool `json:"custom_integrations"`
	MachineLearning      bool `json:"machine_learning"`
}

// GetFeaturesForTier returns the feature set for a license tier
func GetFeaturesForTier(tier LicenseTier) LicenseFeatures {
	switch tier {
	case TierFree:
		return LicenseFeatures{
			EDRMonitoring:     true,
			DLPProtection:     false,
			ThreatHunting:     false,
			RealTimeAlerting:  false,
			CustomRules:       false,
			APIAccess:         false,
			MultiTenancy:      false,
		}
	case TierPro:
		return LicenseFeatures{
			EDRMonitoring:        true,
			DLPProtection:        true,
			ThreatHunting:        true,
			RealTimeAlerting:     true,
			CustomRules:          true,
			APIAccess:            true,
			MultiTenancy:         false,
			AdvancedAnalytics:    true,
			ThreatIntelligence:   true,
			ComplianceReporting:  true,
			PrioritySupport:      false,
		}
	case TierEnterprise:
		return LicenseFeatures{
			EDRMonitoring:        true,
			DLPProtection:        true,
			ThreatHunting:        true,
			RealTimeAlerting:     true,
			CustomRules:          true,
			APIAccess:            true,
			MultiTenancy:         true,
			AdvancedAnalytics:    true,
			ThreatIntelligence:   true,
			IncidentResponse:     true,
			ComplianceReporting:  true,
			PrioritySupport:      true,
			CustomIntegrations:   true,
			MachineLearning:      true,
		}
	default:
		return LicenseFeatures{}
	}
}

// GetLimitsForTier returns resource limits per tier
func GetLimitsForTier(tier LicenseTier) (maxAgents int, maxUsers int) {
	switch tier {
	case TierFree:
		return 5, 1 // 5 agents, 1 user
	case TierPro:
		return 100, 10 // 100 agents, 10 users
	case TierEnterprise:
		return -1, -1 // Unlimited
	default:
		return 0, 0
	}
}

// CreateLicenseRequest is the request body for creating a new license
type CreateLicenseRequest struct {
	CustomerEmail string      `json:"customer_email" binding:"required,email"`
	CustomerName  string      `json:"customer_name" binding:"required"`
	CompanyName   string      `json:"company_name"`
	Tier          LicenseTier `json:"tier" binding:"required"`
	DurationDays  int         `json:"duration_days"` // 0 for perpetual
}

// ValidateLicenseRequest validates a license key
type ValidateLicenseRequest struct {
	LicenseKey string `json:"license_key" binding:"required"`
	AgentID    string `json:"agent_id"`
	Hostname   string `json:"hostname"`
}

// ValidateLicenseResponse returns validation result
type ValidateLicenseResponse struct {
	Valid            bool             `json:"valid"`
	License          *License         `json:"license,omitempty"`
	Features         LicenseFeatures  `json:"features,omitempty"`
	RemainingAgents  int              `json:"remaining_agents,omitempty"`
	ExpiresInDays    int              `json:"expires_in_days,omitempty"`
	Message          string           `json:"message,omitempty"`
}

// LicenseUsage tracks license usage statistics
type LicenseUsage struct {
	LicenseID      string    `json:"license_id" db:"license_id"`
	ActiveAgents   int       `json:"active_agents" db:"active_agents"`
	ActiveUsers    int       `json:"active_users" db:"active_users"`
	EventsIngested int64     `json:"events_ingested" db:"events_ingested"`
	StorageUsedGB  float64   `json:"storage_used_gb" db:"storage_used_gb"`
	LastUpdated    time.Time `json:"last_updated" db:"last_updated"`
}
