// Privé Platform API
// REST API for DLP policy management, agent configuration, and query interface
// Built with Gin framework for high performance

package main

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/sentinel-enterprise/platform/api/internal/handlers"
	"github.com/sentinel-enterprise/platform/database"
	licenseService "github.com/sentinel-enterprise/platform/license/service"
)

const (
	defaultPort = "8080"
	apiVersion  = "v1"
)

func main() {
	// Configure logging
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)
	log.Info("Privé Platform API starting...")

	// Load configuration
	port := getEnv("API_PORT", defaultPort)

	// Initialize database connection (PostgreSQL for metadata)
	dbConfig := database.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnvInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", "prive_app"),
		Password: getEnv("DB_PASSWORD", "change_this_password"),
		Database: getEnv("DB_NAME", "prive"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	db, err := database.InitDB(dbConfig)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close(db)

	// Initialize ClickHouse connection
	clickhouseAddr := getEnv("CLICKHOUSE_ADDR", "localhost:9000")
	ch, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{clickhouseAddr},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		DialTimeout:  5 * time.Second,
	})

	if err != nil {
		log.Warnf("Failed to connect to ClickHouse: %v. Some features will be limited.", err)
		ch = nil
	} else {
		if err := ch.Ping(context.Background()); err != nil {
			log.Warnf("ClickHouse ping failed: %v. Some features will be limited.", err)
			ch = nil
		} else {
			log.Info("ClickHouse connection established")
		}
	}

	// Initialize license service
	// Note: In production, load keys from secure storage (e.g., AWS KMS, HashiCorp Vault)
	privateKeyPath := getEnv("LICENSE_PRIVATE_KEY_PATH", "")
	publicKeyPath := getEnv("LICENSE_PUBLIC_KEY_PATH", "")

	var licenseService *licenseService.LicenseService
	if privateKeyPath != "" && publicKeyPath != "" {
		privateKey, publicKey, err := loadLicenseKeys(privateKeyPath, publicKeyPath)
		if err != nil {
			log.Warnf("Failed to load license keys: %v. License features will be limited.", err)
		} else {
			licenseService = licenseService.NewLicenseService(db, privateKey, publicKey)
			log.Info("License service initialized successfully")
		}
	} else {
		log.Warn("License key paths not configured. Set LICENSE_PRIVATE_KEY_PATH and LICENSE_PUBLIC_KEY_PATH environment variables.")
	}

	// Initialize WebSocket hub
	handlers.InitWebSocketHub()

	// Initialize Gin router
	router := setupRouter(db, ch, licenseService)

	// Create HTTP server
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%s", port),
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Start server in goroutine
	go func() {
		log.Infof("API server listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Errorf("Server forced to shutdown: %v", err)
	}

	log.Info("Server stopped")
}

func setupRouter(db *sql.DB, ch driver.Conn, licService *licenseService.LicenseService) *gin.Engine {
	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"version": apiVersion,
			"time":    time.Now().UTC(),
		})
	})

	// Initialize handlers with dependencies
	licenseHandler := handlers.NewLicenseHandler(licService)
	dlpHandler := handlers.NewDLPHandler(db)
	agentHandler := handlers.NewAgentHandler(db)
	telemetryHandler := handlers.NewTelemetryHandler(db)
	notificationHandler := handlers.NewNotificationHandler(db)
	aiHandler := handlers.NewAIHandler(db, ch)
	collaborativeHandler := handlers.NewCollaborativeHandler(db)
	dataLakeHandler := handlers.NewDataLakeHandler(db)
	deceptionHandler := handlers.NewDeceptionHandler(db)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// DLP Policy Management
		dlp := v1.Group("/dlp")
		{
			dlp.GET("/policies", dlpHandler.ListDLPPolicies)
			dlp.GET("/policies/:id", dlpHandler.GetDLPPolicy)
			dlp.POST("/policies", dlpHandler.CreateDLPPolicy)
			dlp.PUT("/policies/:id", dlpHandler.UpdateDLPPolicy)
			dlp.DELETE("/policies/:id", dlpHandler.DeleteDLPPolicy)

			// Fingerprint management
			dlp.POST("/policies/:id/fingerprints", dlpHandler.AddFingerprints)
			dlp.DELETE("/policies/:id/fingerprints/:fingerprint_id", dlpHandler.DeleteFingerprint)

			// Policy testing
			dlp.POST("/test", dlpHandler.TestDLPPolicy)
		}

		// Agent Management
		agents := v1.Group("/agents")
		{
			agents.POST("/register", agentHandler.RegisterAgent)
			agents.POST("/heartbeat", agentHandler.ProcessHeartbeat)
			agents.GET("", agentHandler.ListAgents)
			agents.GET("/:id", agentHandler.GetAgent)
			agents.GET("/:id/health", agentHandler.GetAgentHealth)
			agents.PUT("/:id", agentHandler.UpdateAgent)
			agents.DELETE("/:id", agentHandler.DeleteAgent)

			// Agent configuration
			agents.GET("/:id/config", agentHandler.GetAgentConfig)
			agents.PUT("/:id/config", agentHandler.UpdateAgentConfig)
		}

		// Telemetry Query Interface
		telemetry := v1.Group("/telemetry")
		{
			telemetry.POST("/query", telemetryHandler.QueryEvents)
			telemetry.GET("/events/:id", telemetryHandler.GetEvent)
			telemetry.GET("/statistics", telemetryHandler.GetStatistics)
		}

		// MITRE ATT&CK Framework
		mitre := v1.Group("/mitre")
		{
			mitre.GET("/tactics", telemetryHandler.ListMITRETactics)
			mitre.GET("/techniques", telemetryHandler.ListMITRETechniques)
			mitre.GET("/coverage", telemetryHandler.GetMITRECoverage)
		}

		// Alerting Rules
		alerts := v1.Group("/alerts")
		{
			alerts.GET("/rules", telemetryHandler.ListAlertRules)
			alerts.POST("/rules", telemetryHandler.CreateAlertRule)
			alerts.PUT("/rules/:id", telemetryHandler.UpdateAlertRule)
			alerts.DELETE("/rules/:id", telemetryHandler.DeleteAlertRule)
		}

		// License Management
		licenses := v1.Group("/licenses")
		{
			licenses.GET("", licenseHandler.ListLicenses)
			licenses.GET("/:id", licenseHandler.GetLicense)
			licenses.POST("", licenseHandler.CreateLicense)
			licenses.POST("/validate", licenseHandler.ValidateLicense)
			licenses.POST("/trial", licenseHandler.GenerateTrialLicense)
			licenses.DELETE("/:id", licenseHandler.RevokeLicense)
			licenses.GET("/:id/usage", licenseHandler.GetLicenseUsage)
		}

		// Notification Channels
		notifications := v1.Group("/notifications")
		{
			notifications.GET("/channels", notificationHandler.ListChannels)
			notifications.GET("/channels/:id", notificationHandler.GetChannel)
			notifications.POST("/channels", notificationHandler.CreateChannel)
			notifications.PUT("/channels/:id", notificationHandler.UpdateChannel)
			notifications.DELETE("/channels/:id", notificationHandler.DeleteChannel)
			notifications.POST("/send", notificationHandler.SendNotification)
			notifications.POST("/test", notificationHandler.TestChannel)
		}

		// AI-Powered Threat Analysis
		ai := v1.Group("/ai")
		{
			ai.POST("/analyze", aiHandler.GenerateThreatSummary)
			ai.GET("/config", aiHandler.GetAIConfig)
			ai.PUT("/config", aiHandler.UpdateAIConfig)
			ai.GET("/history", aiHandler.ListAnalysisHistory)
		}

		// Collaborative Threat Hunting
		collaborative := v1.Group("/collaborative")
		{
			// Shared Rules
			collaborative.POST("/rules/publish", collaborativeHandler.PublishRule)
			collaborative.GET("/rules/search", collaborativeHandler.SearchRules)
			collaborative.GET("/rules/:id", collaborativeHandler.GetRule)
			collaborative.POST("/rules/:id/vote", collaborativeHandler.VoteRule)
			collaborative.POST("/rules/:id/download", collaborativeHandler.DownloadRule)
			collaborative.POST("/rules/:id/comments", collaborativeHandler.AddComment)
			collaborative.GET("/rules/:id/comments", collaborativeHandler.GetComments)

			// Shared IOCs
			collaborative.POST("/iocs/publish", collaborativeHandler.PublishIOC)
			collaborative.GET("/iocs/search", collaborativeHandler.SearchIOCs)
			collaborative.GET("/iocs/:id", collaborativeHandler.GetIOC)
			collaborative.POST("/iocs/:id/report", collaborativeHandler.ReportIOC)

			// Hunting Queries
			collaborative.POST("/queries/publish", collaborativeHandler.PublishQuery)
			collaborative.GET("/queries/search", collaborativeHandler.SearchQueries)
			collaborative.GET("/queries/:id", collaborativeHandler.GetQuery)

			// Statistics
			collaborative.GET("/stats", collaborativeHandler.GetCommunityStats)
		}

		// Security Data Lake (Cold Storage)
		dataLake := v1.Group("/datalake")
		{
			// Configuration
			dataLake.POST("/config", dataLakeHandler.CreateDataLakeConfig)
			dataLake.GET("/config/:license_id", dataLakeHandler.GetDataLakeConfig)
			dataLake.PUT("/config/:license_id", dataLakeHandler.UpdateDataLakeConfig)
			dataLake.POST("/test", dataLakeHandler.TestDataLakeConnection)

			// Archive Jobs
			dataLake.POST("/jobs", dataLakeHandler.CreateArchiveJob)
			dataLake.GET("/jobs/:id", dataLakeHandler.GetArchiveJob)
			dataLake.GET("/jobs", dataLakeHandler.ListArchiveJobs)

			// Datasets
			dataLake.GET("/datasets", dataLakeHandler.ListArchivedDatasets)
			dataLake.POST("/query", dataLakeHandler.QueryArchivedData)

			// Statistics
			dataLake.GET("/stats", dataLakeHandler.GetDataLakeStatistics)
		}

		// Deception Technology (Honeypots & Honey Tokens)
		deception := v1.Group("/deception")
		{
			// Honeypots
			deception.POST("/honeypots", deceptionHandler.CreateHoneypot)
			deception.GET("/honeypots", deceptionHandler.ListHoneypots)
			deception.GET("/honeypots/:id", deceptionHandler.GetHoneypot)
			deception.PUT("/honeypots/:id", deceptionHandler.UpdateHoneypot)
			deception.DELETE("/honeypots/:id", deceptionHandler.DeleteHoneypot)

			// Honey Tokens
			deception.POST("/tokens", deceptionHandler.CreateHoneyToken)
			deception.GET("/tokens", deceptionHandler.ListHoneyTokens)

			// Events
			deception.POST("/events", deceptionHandler.RecordDeceptionEvent)
			deception.GET("/events", deceptionHandler.ListDeceptionEvents)

			// Statistics & Templates
			deception.GET("/stats", deceptionHandler.GetDeceptionStatistics)
			deception.GET("/templates", deceptionHandler.ListHoneypotTemplates)
		}

		// WebSocket Live Updates
		ws := v1.Group("/ws")
		{
			ws.GET("/connect", handlers.HandleWebSocket)
			ws.GET("/stats", handlers.GetConnectionStats())
			ws.POST("/disconnect/:id", handlers.DisconnectClient)
		}
	}

	return router
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func loadLicenseKeys(privateKeyPath, publicKeyPath string) (privateKey, publicKey []byte, err error) {
	privateKey, err = os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read private key: %w", err)
	}

	// Validate Ed25519 private key size (64 bytes)
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, nil, fmt.Errorf("invalid private key size: expected %d bytes, got %d bytes", ed25519.PrivateKeySize, len(privateKey))
	}

	publicKey, err = os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read public key: %w", err)
	}

	// Validate Ed25519 public key size (32 bytes)
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, nil, fmt.Errorf("invalid public key size: expected %d bytes, got %d bytes", ed25519.PublicKeySize, len(publicKey))
	}

	log.Info("License keys validated successfully")
	return privateKey, publicKey, nil
}
