// Collaborative Threat Hunting Handlers

package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/sentinel-enterprise/platform/api/internal/models"
)

// CollaborativeHandler handles collaborative threat hunting
type CollaborativeHandler struct {
	db *sql.DB
}

// NewCollaborativeHandler creates a new collaborative handler
func NewCollaborativeHandler(db *sql.DB) *CollaborativeHandler {
	return &CollaborativeHandler{
		db: db,
	}
}

// PublishRule publishes a detection rule to the community
func (h *CollaborativeHandler) PublishRule(c *gin.Context) {
	var req models.PublishRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Anonymize author if requested
	author := "Anonymous"
	if !req.Anonymous {
		// Get organization name from license
		var orgName string
		h.db.QueryRow("SELECT company_name FROM licenses WHERE id = $1", req.LicenseID).Scan(&orgName)
		if orgName != "" {
			author = orgName
		}
	}

	ruleID := uuid.New().String()
	metadataJSON, _ := json.Marshal(req.Metadata)
	tacticsJSON, _ := json.Marshal(req.MITRETactics)
	techniquesJSON, _ := json.Marshal(req.MITRETechniques)
	tagsJSON, _ := json.Marshal(req.Tags)

	query := `
		INSERT INTO shared_rules (id, name, description, rule_type, content, metadata,
		                          mitre_tactics, mitre_techniques, tags, author,
		                          submitted_by_license, submitted_at, updated_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW(), 'approved')
		RETURNING submitted_at
	`

	var submittedAt time.Time
	err := h.db.QueryRow(query,
		ruleID, req.Name, req.Description, req.RuleType, req.Content,
		string(metadataJSON), string(tacticsJSON), string(techniquesJSON),
		string(tagsJSON), author, req.LicenseID,
	).Scan(&submittedAt)

	if err != nil {
		log.Errorf("Failed to publish rule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish rule"})
		return
	}

	log.Infof("Rule published: %s by %s", req.Name, author)

	c.JSON(http.StatusCreated, gin.H{
		"id":           ruleID,
		"submitted_at": submittedAt,
		"message":      "Rule published successfully",
	})
}

// SearchRules searches for community-shared rules
func (h *CollaborativeHandler) SearchRules(c *gin.Context) {
	query := c.DefaultQuery("query", "")
	ruleType := c.DefaultQuery("rule_type", "")
	verifiedOnly := c.DefaultQuery("verified_only", "false") == "true"
	sortBy := c.DefaultQuery("sort_by", "recent")
	limit := 50
	offset := 0

	baseQuery := `
		SELECT id, name, description, rule_type, content, metadata,
		       mitre_tactics, mitre_techniques, tags, author, submitted_at, updated_at,
		       upvote_count, downvote_count, download_count, comment_count,
		       false_positive_rate, effectiveness_score, is_verified
		FROM shared_rules
		WHERE status = 'approved'
	`

	args := []interface{}{}
	argCount := 1

	if query != "" {
		baseQuery += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+query+"%")
		argCount++
	}

	if ruleType != "" {
		baseQuery += fmt.Sprintf(" AND rule_type = $%d", argCount)
		args = append(args, ruleType)
		argCount++
	}

	if verifiedOnly {
		baseQuery += " AND is_verified = TRUE"
	}

	// Add sorting
	switch sortBy {
	case "popular":
		baseQuery += " ORDER BY upvote_count DESC, download_count DESC"
	case "effectiveness":
		baseQuery += " ORDER BY effectiveness_score DESC NULLS LAST"
	default:
		baseQuery += " ORDER BY submitted_at DESC"
	}

	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	rows, err := h.db.Query(baseQuery, args...)
	if err != nil {
		log.Errorf("Failed to search rules: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}
	defer rows.Close()

	rules := make([]models.SharedRule, 0)
	for rows.Next() {
		var rule models.SharedRule
		var metadataJSON, tacticsJSON, techniquesJSON, tagsJSON []byte
		var fpRate, effectScore sql.NullFloat64

		err := rows.Scan(
			&rule.ID, &rule.Name, &rule.Description, &rule.RuleType, &rule.Content,
			&metadataJSON, &tacticsJSON, &techniquesJSON, &tagsJSON,
			&rule.Author, &rule.SubmittedAt, &rule.UpdatedAt,
			&rule.UpvoteCount, &rule.DownvoteCount, &rule.DownloadCount, &rule.CommentCount,
			&fpRate, &effectScore, &rule.IsVerified,
		)

		if err != nil {
			log.Warnf("Failed to scan rule: %v", err)
			continue
		}

		// Parse JSON fields
		json.Unmarshal(metadataJSON, &rule.Metadata)
		json.Unmarshal(tacticsJSON, &rule.MITRETactics)
		json.Unmarshal(techniquesJSON, &rule.MITRETechniques)
		json.Unmarshal(tagsJSON, &rule.Tags)

		if fpRate.Valid {
			rule.FalsePositiveRate = &fpRate.Float64
		}
		if effectScore.Valid {
			rule.EffectivenessScore = &effectScore.Float64
		}

		rules = append(rules, rule)
	}

	c.JSON(http.StatusOK, gin.H{
		"rules": rules,
		"total": len(rules),
	})
}

// GetRule retrieves a specific shared rule
func (h *CollaborativeHandler) GetRule(c *gin.Context) {
	ruleID := c.Param("id")

	query := `
		SELECT id, name, description, rule_type, content, metadata,
		       mitre_tactics, mitre_techniques, tags, author, submitted_at, updated_at,
		       upvote_count, downvote_count, download_count, comment_count,
		       false_positive_rate, effectiveness_score, status, is_verified
		FROM shared_rules
		WHERE id = $1
	`

	var rule models.SharedRule
	var metadataJSON, tacticsJSON, techniquesJSON, tagsJSON []byte
	var fpRate, effectScore sql.NullFloat64

	err := h.db.QueryRow(query, ruleID).Scan(
		&rule.ID, &rule.Name, &rule.Description, &rule.RuleType, &rule.Content,
		&metadataJSON, &tacticsJSON, &techniquesJSON, &tagsJSON,
		&rule.Author, &rule.SubmittedAt, &rule.UpdatedAt,
		&rule.UpvoteCount, &rule.DownvoteCount, &rule.DownloadCount, &rule.CommentCount,
		&fpRate, &effectScore, &rule.Status, &rule.IsVerified,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
			return
		}
		log.Errorf("Failed to get rule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve rule"})
		return
	}

	// Parse JSON fields
	json.Unmarshal(metadataJSON, &rule.Metadata)
	json.Unmarshal(tacticsJSON, &rule.MITRETactics)
	json.Unmarshal(techniquesJSON, &rule.MITRETechniques)
	json.Unmarshal(tagsJSON, &rule.Tags)

	if fpRate.Valid {
		rule.FalsePositiveRate = &fpRate.Float64
	}
	if effectScore.Valid {
		rule.EffectivenessScore = &effectScore.Float64
	}

	c.JSON(http.StatusOK, rule)
}

// VoteRule allows users to upvote or downvote a rule
func (h *CollaborativeHandler) VoteRule(c *gin.Context) {
	var req models.RuleVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already voted
	var existingVote string
	err := h.db.QueryRow(
		"SELECT vote_type FROM rule_votes WHERE rule_id = $1 AND license_id = $2",
		req.RuleID, req.LicenseID,
	).Scan(&existingVote)

	if err == nil {
		// User already voted, update vote
		if existingVote != req.VoteType {
			// Update vote and adjust counts
			_, err = h.db.Exec(
				"UPDATE rule_votes SET vote_type = $1, voted_at = NOW() WHERE rule_id = $2 AND license_id = $3",
				req.VoteType, req.RuleID, req.LicenseID,
			)

			// Adjust counts
			if req.VoteType == "upvote" {
				h.db.Exec("UPDATE shared_rules SET upvote_count = upvote_count + 1, downvote_count = GREATEST(downvote_count - 1, 0) WHERE id = $1", req.RuleID)
			} else {
				h.db.Exec("UPDATE shared_rules SET downvote_count = downvote_count + 1, upvote_count = GREATEST(upvote_count - 1, 0) WHERE id = $1", req.RuleID)
			}
		}
	} else {
		// New vote
		_, err = h.db.Exec(
			"INSERT INTO rule_votes (rule_id, license_id, vote_type, voted_at) VALUES ($1, $2, $3, NOW())",
			req.RuleID, req.LicenseID, req.VoteType,
		)

		// Update count
		if req.VoteType == "upvote" {
			h.db.Exec("UPDATE shared_rules SET upvote_count = upvote_count + 1 WHERE id = $1", req.RuleID)
		} else {
			h.db.Exec("UPDATE shared_rules SET downvote_count = downvote_count + 1 WHERE id = $1", req.RuleID)
		}
	}

	if err != nil {
		log.Errorf("Failed to vote: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record vote"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Vote recorded successfully"})
}

// DownloadRule downloads a rule (tracks downloads)
func (h *CollaborativeHandler) DownloadRule(c *gin.Context) {
	var req models.DownloadRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Increment download count
	_, err := h.db.Exec(
		"UPDATE shared_rules SET download_count = download_count + 1 WHERE id = $1",
		req.RuleID,
	)

	if err != nil {
		log.Errorf("Failed to update download count: %v", err)
	}

	// Track download
	h.db.Exec(
		"INSERT INTO rule_downloads (rule_id, license_id, downloaded_at) VALUES ($1, $2, NOW())",
		req.RuleID, req.LicenseID,
	)

	// Get rule content
	var rule models.SharedRule
	var metadataJSON, tacticsJSON, techniquesJSON, tagsJSON []byte

	query := `
		SELECT id, name, description, rule_type, content, metadata,
		       mitre_tactics, mitre_techniques, tags, author
		FROM shared_rules
		WHERE id = $1
	`

	err = h.db.QueryRow(query, req.RuleID).Scan(
		&rule.ID, &rule.Name, &rule.Description, &rule.RuleType, &rule.Content,
		&metadataJSON, &tacticsJSON, &techniquesJSON, &tagsJSON, &rule.Author,
	)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	json.Unmarshal(metadataJSON, &rule.Metadata)
	json.Unmarshal(tacticsJSON, &rule.MITRETactics)
	json.Unmarshal(techniquesJSON, &rule.MITRETechniques)
	json.Unmarshal(tagsJSON, &rule.Tags)

	c.JSON(http.StatusOK, rule)
}

// GetCommunityStats returns community statistics
func (h *CollaborativeHandler) GetCommunityStats(c *gin.Context) {
	stats := models.CommunityStats{
		RulesByType: make(map[string]int),
		IOCsByType:  make(map[string]int),
	}

	// Total counts
	h.db.QueryRow("SELECT COUNT(*) FROM shared_rules WHERE status = 'approved'").Scan(&stats.TotalRules)
	h.db.QueryRow("SELECT COUNT(*) FROM shared_iocs").Scan(&stats.TotalIOCs)
	h.db.QueryRow("SELECT COUNT(*) FROM hunting_queries WHERE is_public = TRUE").Scan(&stats.TotalQueries)
	h.db.QueryRow("SELECT COUNT(DISTINCT submitted_by_license) FROM shared_rules").Scan(&stats.TotalContributors)

	// Rules by type
	rows, _ := h.db.Query("SELECT rule_type, COUNT(*) FROM shared_rules WHERE status = 'approved' GROUP BY rule_type")
	for rows.Next() {
		var ruleType string
		var count int
		rows.Scan(&ruleType, &count)
		stats.RulesByType[ruleType] = count
	}
	rows.Close()

	// Top contributors
	rows, _ = h.db.Query(`
		SELECT author, COUNT(*) as rule_count, COALESCE(SUM(upvote_count), 0) as total_upvotes
		FROM shared_rules
		WHERE status = 'approved' AND author != 'Anonymous'
		GROUP BY author
		ORDER BY rule_count DESC, total_upvotes DESC
		LIMIT 10
	`)

	stats.TopContributors = make([]models.ContributorStat, 0)
	for rows.Next() {
		var contrib models.ContributorStat
		rows.Scan(&contrib.Author, &contrib.RuleCount, &contrib.TotalUpvotes)
		contrib.ReputationScore = contrib.RuleCount*10 + contrib.TotalUpvotes
		stats.TopContributors = append(stats.TopContributors, contrib)
	}
	rows.Close()

	// Recent activity
	rows, _ = h.db.Query(`
		SELECT id, name, author, submitted_at
		FROM shared_rules
		WHERE status = 'approved'
		ORDER BY submitted_at DESC
		LIMIT 10
	`)

	stats.RecentActivity = make([]models.ActivityItem, 0)
	for rows.Next() {
		var activity models.ActivityItem
		activity.Type = "rule_published"
		rows.Scan(&activity.ItemID, &activity.Title, &activity.Author, &activity.Timestamp)
		stats.RecentActivity = append(stats.RecentActivity, activity)
	}
	rows.Close()

	c.JSON(http.StatusOK, stats)
}

// PublishIOC publishes an IOC to the community
func (h *CollaborativeHandler) PublishIOC(c *gin.Context) {
	var req models.PublishIOCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	submittedBy := "Anonymous"
	if !req.Anonymous {
		var orgName string
		h.db.QueryRow("SELECT company_name FROM licenses WHERE id = $1", req.LicenseID).Scan(&orgName)
		if orgName != "" {
			submittedBy = orgName
		}
	}

	iocID := uuid.New().String()
	tagsJSON, _ := json.Marshal(req.Tags)

	query := `
		INSERT INTO shared_iocs (id, type, value, description, threat_type, confidence, tags,
		                         submitted_by, submitted_by_license, submitted_at, first_seen, last_seen)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW(), NOW())
		RETURNING submitted_at
	`

	var submittedAt time.Time
	err := h.db.QueryRow(query,
		iocID, req.Type, req.Value, req.Description, req.ThreatType,
		req.Confidence, string(tagsJSON), submittedBy, req.LicenseID,
	).Scan(&submittedAt)

	if err != nil {
		// Check if IOC already exists
		if strings.Contains(err.Error(), "duplicate") {
			// Update existing IOC report count
			h.db.Exec("UPDATE shared_iocs SET report_count = report_count + 1, last_seen = NOW() WHERE value = $1 AND type = $2", req.Value, req.Type)
			c.JSON(http.StatusOK, gin.H{"message": "IOC already exists, updated report count"})
			return
		}

		log.Errorf("Failed to publish IOC: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish IOC"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           iocID,
		"submitted_at": submittedAt,
		"message":      "IOC published successfully",
	})
}

// SearchIOCs searches for community-shared IOCs
func (h *CollaborativeHandler) SearchIOCs(c *gin.Context) {
	query := c.DefaultQuery("query", "")
	iocType := c.DefaultQuery("type", "")
	threatType := c.DefaultQuery("threat_type", "")
	verifiedOnly := c.DefaultQuery("verified_only", "false") == "true"
	limit := 50
	offset := 0

	baseQuery := `
		SELECT id, type, value, description, threat_type, confidence, tags,
		       first_seen, last_seen, submitted_by, submitted_at, report_count, is_verified
		FROM shared_iocs
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 1

	if query != "" {
		baseQuery += fmt.Sprintf(" AND (value ILIKE $%d OR description ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+query+"%")
		argCount++
	}

	if iocType != "" {
		baseQuery += fmt.Sprintf(" AND type = $%d", argCount)
		args = append(args, iocType)
		argCount++
	}

	if threatType != "" {
		baseQuery += fmt.Sprintf(" AND threat_type = $%d", argCount)
		args = append(args, threatType)
		argCount++
	}

	if verifiedOnly {
		baseQuery += " AND is_verified = TRUE"
	}

	baseQuery += " ORDER BY report_count DESC, last_seen DESC"
	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	rows, err := h.db.Query(baseQuery, args...)
	if err != nil {
		log.Errorf("Failed to search IOCs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}
	defer rows.Close()

	iocs := make([]models.SharedIOC, 0)
	for rows.Next() {
		var ioc models.SharedIOC
		var tagsJSON []byte

		err := rows.Scan(
			&ioc.ID, &ioc.Type, &ioc.Value, &ioc.Description, &ioc.ThreatType,
			&ioc.Confidence, &tagsJSON, &ioc.FirstSeen, &ioc.LastSeen,
			&ioc.SubmittedBy, &ioc.SubmittedAt, &ioc.ReportCount, &ioc.IsVerified,
		)

		if err != nil {
			log.Warnf("Failed to scan IOC: %v", err)
			continue
		}

		json.Unmarshal(tagsJSON, &ioc.Tags)
		iocs = append(iocs, ioc)
	}

	c.JSON(http.StatusOK, gin.H{
		"iocs":  iocs,
		"total": len(iocs),
	})
}
