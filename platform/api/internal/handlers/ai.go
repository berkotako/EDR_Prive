// AI-Powered Threat Analysis Handlers with OpenAI and Anthropic Integration

package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/sentinel-enterprise/platform/api/internal/models"
)

// AIHandler handles AI-powered threat analysis
type AIHandler struct {
	db         *sql.DB
	clickhouse driver.Conn
}

// NewAIHandler creates a new AI handler
func NewAIHandler(db *sql.DB, ch driver.Conn) *AIHandler {
	return &AIHandler{
		db:         db,
		clickhouse: ch,
	}
}

// GenerateThreatSummary generates AI-powered analysis of security events
func (h *AIHandler) GenerateThreatSummary(c *gin.Context) {
	var req models.GenerateSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get AI configuration for tenant
	config, err := h.getAIConfig(req.TenantID)
	if err != nil || !config.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AI analysis not configured or disabled for this tenant"})
		return
	}

	// Use requested provider or default from config
	provider := req.Provider
	if provider == "" {
		provider = config.Provider
	}

	startTime := time.Now()

	// Fetch events based on request
	events, err := h.fetchEventsForAnalysis(req)
	if err != nil {
		log.Errorf("Failed to fetch events: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events"})
		return
	}

	if len(events) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No events found for analysis"})
		return
	}

	// Generate analysis using selected LLM provider
	var summary *models.ThreatSummary
	switch provider {
	case models.ProviderOpenAI:
		summary, err = h.analyzeWithOpenAI(config, req, events)
	case models.ProviderAnthropic:
		summary, err = h.analyzeWithAnthropic(config, req, events)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported AI provider"})
		return
	}

	if err != nil {
		log.Errorf("AI analysis failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Analysis failed: %v", err)})
		return
	}

	// Complete summary metadata
	summary.ID = uuid.New().String()
	summary.TenantID = req.TenantID
	summary.AnalysisType = req.AnalysisType
	summary.Provider = provider
	summary.EventCount = len(events)
	summary.GeneratedAt = time.Now()
	summary.ProcessingTimeMs = time.Since(startTime).Milliseconds()

	// Store analysis in history
	h.storeAnalysisHistory(summary)

	c.JSON(http.StatusOK, summary)
}

// GetAIConfig retrieves AI configuration for a tenant
func (h *AIHandler) GetAIConfig(c *gin.Context) {
	licenseID := c.Query("license_id")
	if licenseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "license_id required"})
		return
	}

	config, err := h.getAIConfig(licenseID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "AI configuration not found"})
		return
	}

	// Mask sensitive keys
	if config.OpenAIKey != "" {
		config.OpenAIKey = "sk-" + strings.Repeat("*", 40)
	}
	if config.AnthropicKey != "" {
		config.AnthropicKey = "sk-ant-" + strings.Repeat("*", 40)
	}

	c.JSON(http.StatusOK, config)
}

// UpdateAIConfig updates AI configuration
func (h *AIHandler) UpdateAIConfig(c *gin.Context) {
	var req models.UpdateAIConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if config exists
	var exists bool
	h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM ai_configs WHERE license_id = $1)", req.LicenseID).Scan(&exists)

	if !exists {
		// Insert new config
		query := `
			INSERT INTO ai_configs (license_id, provider, openai_key, openai_model, anthropic_key, anthropic_model, max_tokens, temperature, enabled, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		`
		provider := models.ProviderOpenAI
		if req.Provider != nil {
			provider = *req.Provider
		}

		_, err := h.db.Exec(query,
			req.LicenseID,
			provider,
			getStringValue(req.OpenAIKey, ""),
			getStringValue(req.OpenAIModel, "gpt-4"),
			getStringValue(req.AnthropicKey, ""),
			getStringValue(req.AnthropicModel, "claude-3-5-sonnet-20241022"),
			getIntValue(req.MaxTokens, 4096),
			getFloat64Value(req.Temperature, 0.3),
			getBoolValue(req.Enabled, true),
		)
		if err != nil {
			log.Errorf("Failed to create AI config: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create config"})
			return
		}
	} else {
		// Update existing config
		query := "UPDATE ai_configs SET updated_at = NOW()"
		args := []interface{}{}
		argCount := 1

		if req.Provider != nil {
			query += fmt.Sprintf(", provider = $%d", argCount)
			args = append(args, *req.Provider)
			argCount++
		}
		if req.OpenAIKey != nil {
			query += fmt.Sprintf(", openai_key = $%d", argCount)
			args = append(args, *req.OpenAIKey)
			argCount++
		}
		if req.OpenAIModel != nil {
			query += fmt.Sprintf(", openai_model = $%d", argCount)
			args = append(args, *req.OpenAIModel)
			argCount++
		}
		if req.AnthropicKey != nil {
			query += fmt.Sprintf(", anthropic_key = $%d", argCount)
			args = append(args, *req.AnthropicKey)
			argCount++
		}
		if req.AnthropicModel != nil {
			query += fmt.Sprintf(", anthropic_model = $%d", argCount)
			args = append(args, *req.AnthropicModel)
			argCount++
		}
		if req.MaxTokens != nil {
			query += fmt.Sprintf(", max_tokens = $%d", argCount)
			args = append(args, *req.MaxTokens)
			argCount++
		}
		if req.Temperature != nil {
			query += fmt.Sprintf(", temperature = $%d", argCount)
			args = append(args, *req.Temperature)
			argCount++
		}
		if req.Enabled != nil {
			query += fmt.Sprintf(", enabled = $%d", argCount)
			args = append(args, *req.Enabled)
			argCount++
		}

		query += fmt.Sprintf(" WHERE license_id = $%d", argCount)
		args = append(args, req.LicenseID)

		_, err := h.db.Exec(query, args...)
		if err != nil {
			log.Errorf("Failed to update AI config: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "AI configuration updated successfully"})
}

// ListAnalysisHistory lists past AI analyses
func (h *AIHandler) ListAnalysisHistory(c *gin.Context) {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	query := `
		SELECT id, tenant_id, analysis_type, provider, summary, event_count, tokens_used, created_at, created_by
		FROM ai_analysis_history
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT 50
	`

	rows, err := h.db.Query(query, tenantID)
	if err != nil {
		log.Errorf("Failed to query analysis history: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query failed"})
		return
	}
	defer rows.Close()

	history := make([]models.AIAnalysisHistory, 0)
	for rows.Next() {
		var item models.AIAnalysisHistory
		var createdBy sql.NullString

		err := rows.Scan(
			&item.ID, &item.TenantID, &item.AnalysisType, &item.Provider,
			&item.Summary, &item.EventCount, &item.TokensUsed, &item.CreatedAt, &createdBy,
		)

		if err != nil {
			log.Warnf("Failed to scan history item: %v", err)
			continue
		}

		if createdBy.Valid {
			item.CreatedBy = createdBy.String
		}

		history = append(history, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"total":   len(history),
	})
}

// Private helper methods

func (h *AIHandler) getAIConfig(licenseID string) (*models.AIConfig, error) {
	config := &models.AIConfig{}

	query := `
		SELECT provider, openai_key, openai_model, anthropic_key, anthropic_model,
		       max_tokens, temperature, enabled
		FROM ai_configs
		WHERE license_id = $1
	`

	var openAIKey, openAIModel, anthropicKey, anthropicModel sql.NullString

	err := h.db.QueryRow(query, licenseID).Scan(
		&config.Provider, &openAIKey, &openAIModel, &anthropicKey, &anthropicModel,
		&config.MaxTokens, &config.Temperature, &config.Enabled,
	)

	if err != nil {
		return nil, err
	}

	if openAIKey.Valid {
		config.OpenAIKey = openAIKey.String
	}
	if openAIModel.Valid {
		config.OpenAIModel = openAIModel.String
	}
	if anthropicKey.Valid {
		config.AnthropicKey = anthropicKey.String
	}
	if anthropicModel.Valid {
		config.AnthropicModel = anthropicModel.String
	}

	return config, nil
}

func (h *AIHandler) fetchEventsForAnalysis(req models.GenerateSummaryRequest) ([]models.TelemetryEvent, error) {
	if h.clickhouse == nil {
		return nil, fmt.Errorf("clickhouse connection not available")
	}

	ctx := context.Background()
	query := `
		SELECT event_id, agent_id, timestamp, event_type, mitre_tactic, mitre_technique,
		       severity, hostname, os_type, payload, process_name, file_path, dst_ip, username
		FROM telemetry_events
		WHERE tenant_id = ?
	`
	args := []interface{}{req.TenantID}

	// Filter by event IDs if provided
	if len(req.EventIDs) > 0 {
		placeholders := make([]string, len(req.EventIDs))
		for i := range req.EventIDs {
			placeholders[i] = "?"
			args = append(args, req.EventIDs[i])
		}
		query += " AND event_id IN (" + strings.Join(placeholders, ",") + ")"
	}

	// Filter by time range if provided
	if req.TimeRange != nil {
		query += " AND timestamp >= ? AND timestamp <= ?"
		args = append(args, req.TimeRange.Start, req.TimeRange.End)
	}

	query += " ORDER BY timestamp ASC LIMIT 1000"

	rows, err := h.clickhouse.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]models.TelemetryEvent, 0)
	for rows.Next() {
		var event models.TelemetryEvent
		var payloadStr string
		var eventID string

		err := rows.Scan(
			&eventID, &event.AgentID, &event.Timestamp, &event.EventType,
			&event.MitreTactic, &event.MitreTechnique, &event.Severity,
			&event.Hostname, &event.OSType, &payloadStr, &event.ProcessName,
			&event.FilePath, &event.DstIP, &event.Username,
		)

		if err != nil {
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

	return events, nil
}

func (h *AIHandler) analyzeWithOpenAI(config *models.AIConfig, req models.GenerateSummaryRequest, events []models.TelemetryEvent) (*models.ThreatSummary, error) {
	// Build prompt
	prompt := h.buildAnalysisPrompt(req.AnalysisType, events, req.CustomPrompt)

	// Call OpenAI API
	requestBody := map[string]interface{}{
		"model": config.OpenAIModel,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a cybersecurity expert analyzing security events for an EDR/DLP platform. Provide detailed, actionable analysis with specific recommendations.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens":  config.MaxTokens,
		"temperature": config.Temperature,
	}

	jsonData, _ := json.Marshal(requestBody)

	httpReq, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+config.OpenAIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai API returned status %d", resp.StatusCode)
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	// Parse the AI response
	summary := h.parseAIResponse(apiResp.Choices[0].Message.Content, req.AnalysisType, events)
	summary.TokensUsed = apiResp.Usage.TotalTokens

	return summary, nil
}

func (h *AIHandler) analyzeWithAnthropic(config *models.AIConfig, req models.GenerateSummaryRequest, events []models.TelemetryEvent) (*models.ThreatSummary, error) {
	// Build prompt
	prompt := h.buildAnalysisPrompt(req.AnalysisType, events, req.CustomPrompt)

	// Call Anthropic API
	requestBody := map[string]interface{}{
		"model":      config.AnthropicModel,
		"max_tokens": config.MaxTokens,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"system":      "You are a cybersecurity expert analyzing security events for an EDR/DLP platform. Provide detailed, actionable analysis with specific recommendations.",
		"temperature": config.Temperature,
	}

	jsonData, _ := json.Marshal(requestBody)

	httpReq, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", config.AnthropicKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic API returned status %d", resp.StatusCode)
	}

	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("no response from Anthropic")
	}

	// Parse the AI response
	summary := h.parseAIResponse(apiResp.Content[0].Text, req.AnalysisType, events)
	summary.TokensUsed = apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens

	return summary, nil
}

func (h *AIHandler) buildAnalysisPrompt(analysisType models.AnalysisType, events []models.TelemetryEvent, customPrompt string) string {
	// Build event context
	eventsJSON, _ := json.MarshalIndent(events, "", "  ")

	basePrompt := fmt.Sprintf(`Analyze the following %d security events and provide a comprehensive %s.

Events:
%s

`, len(events), analysisType, string(eventsJSON))

	switch analysisType {
	case models.AnalysisIncidentSummary:
		basePrompt += `Provide:
1. Executive Summary (2-3 sentences)
2. Key Findings (bullet points)
3. MITRE ATT&CK techniques observed
4. Risk assessment
5. Immediate recommendations`

	case models.AnalysisAttackChain:
		basePrompt += `Reconstruct the attack chain:
1. Initial access method
2. Execution timeline
3. Persistence mechanisms
4. Privilege escalation attempts
5. Lateral movement
6. Data collection/exfiltration
7. Overall narrative`

	case models.AnalysisRemediationPlan:
		basePrompt += `Create detailed remediation plan:
1. Immediate containment steps
2. Investigation actions
3. Eradication procedures
4. Recovery steps
5. Long-term prevention measures
Include specific commands where applicable.`

	case models.AnalysisRootCause:
		basePrompt += `Determine root cause:
1. Initial vulnerability or weakness
2. How the attacker exploited it
3. Why detection/prevention failed
4. Contributing factors
5. Lessons learned`

	case models.AnalysisRiskAssessment:
		basePrompt += `Assess risk:
1. Overall risk score (0-10)
2. Likelihood of similar attacks
3. Potential impact
4. Current exposure
5. Risk factors breakdown`
	}

	if customPrompt != "" {
		basePrompt += "\n\nAdditional context:\n" + customPrompt
	}

	basePrompt += "\n\nProvide analysis in a structured format with clear sections."

	return basePrompt
}

func (h *AIHandler) parseAIResponse(content string, analysisType models.AnalysisType, events []models.TelemetryEvent) *models.ThreatSummary {
	// Extract key findings (lines starting with - or •)
	keyFindings := make([]string, 0)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "•") || strings.HasPrefix(trimmed, "*") {
			finding := strings.TrimLeft(trimmed, "-•* ")
			if finding != "" {
				keyFindings = append(keyFindings, finding)
			}
		}
	}

	// Extract recommendations
	recommendations := make([]string, 0)
	inRecommendations := false
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "recommendation") {
			inRecommendations = true
			continue
		}
		if inRecommendations {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "•") || strings.HasPrefix(trimmed, "*") {
				rec := strings.TrimLeft(trimmed, "-•* ")
				if rec != "" {
					recommendations = append(recommendations, rec)
				}
			}
		}
	}

	// Calculate time range from events
	var timeRange models.TimeRange
	if len(events) > 0 {
		timeRange.Start = events[0].Timestamp
		timeRange.End = events[len(events)-1].Timestamp
	}

	return &models.ThreatSummary{
		Summary:          content,
		KeyFindings:      keyFindings,
		Recommendations:  recommendations,
		TimeRange:        timeRange,
	}
}

func (h *AIHandler) storeAnalysisHistory(summary *models.ThreatSummary) {
	query := `
		INSERT INTO ai_analysis_history (id, tenant_id, analysis_type, provider, summary, event_count, tokens_used, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := h.db.Exec(query,
		summary.ID, summary.TenantID, summary.AnalysisType, summary.Provider,
		summary.Summary, summary.EventCount, summary.TokensUsed, summary.GeneratedAt,
	)

	if err != nil {
		log.Errorf("Failed to store analysis history: %v", err)
	}
}

// Helper functions for pointer values
func getStringValue(p *string, def string) string {
	if p != nil {
		return *p
	}
	return def
}

func getIntValue(p *int, def int) int {
	if p != nil {
		return *p
	}
	return def
}

func getFloat64Value(p *float64, def float64) float64 {
	if p != nil {
		return *p
	}
	return def
}

func getBoolValue(p *bool, def bool) bool {
	if p != nil {
		return *p
	}
	return def
}
