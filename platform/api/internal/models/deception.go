// Deception Technology Models
// Provides honeypots, honey tokens, and deception-based threat detection

package models

import "time"

// Honeypot represents a deployed deception asset
type Honeypot struct {
	ID              string                 `json:"id"`
	LicenseID       string                 `json:"license_id"`
	Name            string                 `json:"name"`
	HoneypotType    HoneypotType           `json:"honeypot_type"`
	Status          HoneypotStatus         `json:"status"`
	DeploymentMode  string                 `json:"deployment_mode"` // network, endpoint, cloud
	TargetPlatform  string                 `json:"target_platform"` // windows, linux, aws, azure
	Configuration   HoneypotConfiguration  `json:"configuration"`
	Location        string                 `json:"location"` // IP address or endpoint ID
	IsActive        bool                   `json:"is_active"`
	InteractionCount int                   `json:"interaction_count"`
	LastInteraction *time.Time             `json:"last_interaction,omitempty"`
	DeployedAt      time.Time              `json:"deployed_at"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// HoneypotType defines the type of honeypot
type HoneypotType string

const (
	HoneypotTypeSSH          HoneypotType = "ssh"
	HoneypotTypeSMB          HoneypotType = "smb"
	HoneypotTypeRDP          HoneypotType = "rdp"
	HoneypotTypeHTTP         HoneypotType = "http"
	HoneypotTypeDatabase     HoneypotType = "database"
	HoneypotTypeEmailAccount HoneypotType = "email_account"
	HoneypotTypeFileShare    HoneypotType = "file_share"
	HoneypotTypeAPIEndpoint  HoneypotType = "api_endpoint"
	HoneypotTypeCredentials  HoneypotType = "credentials"
)

// HoneypotStatus represents the status of a honeypot
type HoneypotStatus string

const (
	HoneypotStatusActive     HoneypotStatus = "active"
	HoneypotStatusInactive   HoneypotStatus = "inactive"
	HoneypotStatusCompromised HoneypotStatus = "compromised"
	HoneypotStatusDeploying  HoneypotStatus = "deploying"
	HoneypotStatusError      HoneypotStatus = "error"
)

// HoneypotConfiguration defines honeypot-specific configuration
type HoneypotConfiguration struct {
	ListenPort         int                    `json:"listen_port,omitempty"`
	ServiceBanner      string                 `json:"service_banner,omitempty"`
	AllowedCommands    []string               `json:"allowed_commands,omitempty"`
	FakeFiles          []FakeFile             `json:"fake_files,omitempty"`
	FakeCredentials    []FakeCredential       `json:"fake_credentials,omitempty"`
	ResponseDelay      int                    `json:"response_delay,omitempty"` // milliseconds
	LogAllInteractions bool                   `json:"log_all_interactions"`
	AlertOnInteraction bool                   `json:"alert_on_interaction"`
	CustomConfig       map[string]interface{} `json:"custom_config,omitempty"`
}

// FakeFile represents a fake file used as bait
type FakeFile struct {
	Path        string `json:"path"`
	Name        string `json:"name"`
	Content     string `json:"content,omitempty"`
	ContentType string `json:"content_type"` // text, binary, template
	Size        int64  `json:"size"`
}

// FakeCredential represents fake credentials used as bait
type FakeCredential struct {
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	Domain      string   `json:"domain,omitempty"`
	ServiceType string   `json:"service_type"` // ssh, rdp, database, etc.
	Permissions []string `json:"permissions,omitempty"`
}

// CreateHoneypotRequest is the request to deploy a honeypot
type CreateHoneypotRequest struct {
	LicenseID       string                 `json:"license_id" binding:"required"`
	Name            string                 `json:"name" binding:"required"`
	HoneypotType    HoneypotType           `json:"honeypot_type" binding:"required"`
	DeploymentMode  string                 `json:"deployment_mode" binding:"required"`
	TargetPlatform  string                 `json:"target_platform" binding:"required"`
	Configuration   HoneypotConfiguration  `json:"configuration" binding:"required"`
	Location        string                 `json:"location"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// UpdateHoneypotRequest is the request to update a honeypot
type UpdateHoneypotRequest struct {
	Name          *string                `json:"name"`
	Status        *HoneypotStatus        `json:"status"`
	IsActive      *bool                  `json:"is_active"`
	Configuration *HoneypotConfiguration `json:"configuration"`
}

// HoneyToken represents a canary token for detecting unauthorized access
type HoneyToken struct {
	ID             string                 `json:"id"`
	LicenseID      string                 `json:"license_id"`
	Name           string                 `json:"name"`
	TokenType      HoneyTokenType         `json:"token_type"`
	TokenValue     string                 `json:"token_value"`
	CallbackURL    string                 `json:"callback_url"`
	IsActive       bool                   `json:"is_active"`
	AccessCount    int                    `json:"access_count"`
	LastAccessed   *time.Time             `json:"last_accessed,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// HoneyTokenType defines the type of honey token
type HoneyTokenType string

const (
	TokenTypeAWSKey          HoneyTokenType = "aws_key"
	TokenTypeAPIKey          HoneyTokenType = "api_key"
	TokenTypeDatabaseCreds   HoneyTokenType = "database_creds"
	TokenTypeDocumentURL     HoneyTokenType = "document_url"
	TokenTypeDNSQuery        HoneyTokenType = "dns_query"
	TokenTypeEmailAddress    HoneyTokenType = "email_address"
	TokenTypeWebBug          HoneyTokenType = "web_bug"
	TokenTypeQRCode          HoneyTokenType = "qr_code"
	TokenTypeOfficeDocument  HoneyTokenType = "office_document"
)

// CreateHoneyTokenRequest is the request to create a honey token
type CreateHoneyTokenRequest struct {
	LicenseID   string                 `json:"license_id" binding:"required"`
	Name        string                 `json:"name" binding:"required"`
	TokenType   HoneyTokenType         `json:"token_type" binding:"required"`
	CallbackURL string                 `json:"callback_url,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// UpdateHoneyTokenRequest is the request to update a honey token
type UpdateHoneyTokenRequest struct {
	Name     *string `json:"name"`
	IsActive *bool   `json:"is_active"`
}

// DeceptionEvent represents an interaction with a deception asset
type DeceptionEvent struct {
	ID              string                 `json:"id"`
	LicenseID       string                 `json:"license_id"`
	EventType       DeceptionEventType     `json:"event_type"`
	HoneypotID      string                 `json:"honeypot_id,omitempty"`
	HoneyTokenID    string                 `json:"honey_token_id,omitempty"`
	SourceIP        string                 `json:"source_ip"`
	SourceHostname  string                 `json:"source_hostname,omitempty"`
	SourceUser      string                 `json:"source_user,omitempty"`
	InteractionType string                 `json:"interaction_type"` // access, scan, exploit_attempt, credential_use
	Severity        string                 `json:"severity"`
	Details         DeceptionEventDetails  `json:"details"`
	AlertCreated    bool                   `json:"alert_created"`
	AlertID         string                 `json:"alert_id,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	DetectedAt      time.Time              `json:"detected_at"`
}

// DeceptionEventType defines the type of deception event
type DeceptionEventType string

const (
	EventTypeHoneypotAccess    DeceptionEventType = "honeypot_access"
	EventTypeHoneyTokenAccess  DeceptionEventType = "honeytoken_access"
	EventTypeCredentialAttempt DeceptionEventType = "credential_attempt"
	EventTypeFileAccess        DeceptionEventType = "file_access"
	EventTypeNetworkScan       DeceptionEventType = "network_scan"
)

// DeceptionEventDetails provides detailed information about the event
type DeceptionEventDetails struct {
	Protocol           string            `json:"protocol,omitempty"`
	Port               int               `json:"port,omitempty"`
	Command            string            `json:"command,omitempty"`
	UserAgent          string            `json:"user_agent,omitempty"`
	RequestHeaders     map[string]string `json:"request_headers,omitempty"`
	AuthenticationInfo string            `json:"authentication_info,omitempty"`
	AccessedFile       string            `json:"accessed_file,omitempty"`
	SessionDuration    int64             `json:"session_duration,omitempty"` // milliseconds
	BytesTransferred   int64             `json:"bytes_transferred,omitempty"`
}

// DeceptionCampaign represents a coordinated deception deployment
type DeceptionCampaign struct {
	ID              string                 `json:"id"`
	LicenseID       string                 `json:"license_id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Status          string                 `json:"status"` // active, paused, completed
	HoneypotIDs     []string               `json:"honeypot_ids"`
	HoneyTokenIDs   []string               `json:"honey_token_ids"`
	StartDate       time.Time              `json:"start_date"`
	EndDate         *time.Time             `json:"end_date,omitempty"`
	EventCount      int                    `json:"event_count"`
	ThreatScore     float64                `json:"threat_score"`
	Objectives      []string               `json:"objectives"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// CreateCampaignRequest is the request to create a deception campaign
type CreateCampaignRequest struct {
	LicenseID     string                 `json:"license_id" binding:"required"`
	Name          string                 `json:"name" binding:"required"`
	Description   string                 `json:"description"`
	HoneypotIDs   []string               `json:"honeypot_ids"`
	HoneyTokenIDs []string               `json:"honey_token_ids"`
	Objectives    []string               `json:"objectives"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// UpdateCampaignRequest is the request to update a campaign
type UpdateCampaignRequest struct {
	Name          *string    `json:"name"`
	Description   *string    `json:"description"`
	Status        *string    `json:"status"`
	HoneypotIDs   *[]string  `json:"honeypot_ids"`
	HoneyTokenIDs *[]string  `json:"honey_token_ids"`
	EndDate       *time.Time `json:"end_date"`
}

// DeceptionStatistics provides statistics about deception deployments
type DeceptionStatistics struct {
	LicenseID               string    `json:"license_id"`
	TotalHoneypots          int       `json:"total_honeypots"`
	ActiveHoneypots         int       `json:"active_honeypots"`
	CompromisedHoneypots    int       `json:"compromised_honeypots"`
	TotalHoneyTokens        int       `json:"total_honey_tokens"`
	ActiveHoneyTokens       int       `json:"active_honey_tokens"`
	TotalEvents             int64     `json:"total_events"`
	Events24h               int       `json:"events_24h"`
	Events7d                int       `json:"events_7d"`
	UniqueSourceIPs         int       `json:"unique_source_ips"`
	ThreatScore             float64   `json:"threat_score"`
	MostTargetedHoneypot    string    `json:"most_targeted_honeypot,omitempty"`
	MostAccessedToken       string    `json:"most_accessed_token,omitempty"`
	RecentCompromise        *time.Time `json:"recent_compromise,omitempty"`
	ActiveCampaigns         int       `json:"active_campaigns"`
	TotalCampaigns          int       `json:"total_campaigns"`
}

// DeceptionRecommendation provides AI-powered recommendations for deception strategy
type DeceptionRecommendation struct {
	ID              string                 `json:"id"`
	LicenseID       string                 `json:"license_id"`
	RecommendationType string              `json:"recommendation_type"` // deployment, configuration, response
	Priority        string                 `json:"priority"` // low, medium, high, critical
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Rationale       string                 `json:"rationale"`
	Actions         []RecommendedAction    `json:"actions"`
	BasedOnEvents   []string               `json:"based_on_events,omitempty"`
	Status          string                 `json:"status"` // pending, accepted, rejected, implemented
	GeneratedAt     time.Time              `json:"generated_at"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// RecommendedAction defines a specific action to take
type RecommendedAction struct {
	Action      string                 `json:"action"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// HoneypotTemplate represents a pre-configured honeypot template
type HoneypotTemplate struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	Description     string                `json:"description"`
	HoneypotType    HoneypotType          `json:"honeypot_type"`
	TargetPlatform  string                `json:"target_platform"`
	DifficultyLevel string                `json:"difficulty_level"` // low, medium, high
	Configuration   HoneypotConfiguration `json:"configuration"`
	IsPopular       bool                  `json:"is_popular"`
	UseCount        int                   `json:"use_count"`
	SuccessRate     float64               `json:"success_rate"`
}

// DeceptionPlaybook represents automated response to deception events
type DeceptionPlaybook struct {
	ID              string                 `json:"id"`
	LicenseID       string                 `json:"license_id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Enabled         bool                   `json:"enabled"`
	TriggerConditions map[string]interface{} `json:"trigger_conditions"`
	Actions         []PlaybookAction       `json:"actions"`
	ExecutionCount  int                    `json:"execution_count"`
	LastExecuted    *time.Time             `json:"last_executed,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// PlaybookAction defines an automated action
type PlaybookAction struct {
	ActionType  string                 `json:"action_type"` // block_ip, quarantine_host, send_alert, etc.
	Priority    int                    `json:"priority"`
	Parameters  map[string]interface{} `json:"parameters"`
	Description string                 `json:"description"`
}
