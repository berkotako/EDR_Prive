// Privé Platform API
// REST API for DLP policy management, agent configuration, and query interface
// Built with Gin framework for high performance

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/sentinel-enterprise/platform/api/internal/handlers"
	"github.com/sentinel-enterprise/platform/api/internal/models"
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
	dbURL := getEnv("DATABASE_URL", "")

	// Initialize database connection (PostgreSQL for metadata)
	// TODO: Initialize actual database
	log.Info("Database connection initialized (skeleton)")

	// Initialize Gin router
	router := setupRouter()

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

func setupRouter() *gin.Engine {
	router := gin.Default()

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"version": apiVersion,
			"time":    time.Now().UTC(),
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// DLP Policy Management
		dlp := v1.Group("/dlp")
		{
			dlp.GET("/policies", handlers.ListDLPPolicies)
			dlp.GET("/policies/:id", handlers.GetDLPPolicy)
			dlp.POST("/policies", handlers.CreateDLPPolicy)
			dlp.PUT("/policies/:id", handlers.UpdateDLPPolicy)
			dlp.DELETE("/policies/:id", handlers.DeleteDLPPolicy)

			// Fingerprint management
			dlp.POST("/policies/:id/fingerprints", handlers.AddFingerprints)
			dlp.DELETE("/policies/:id/fingerprints/:fingerprint_id", handlers.DeleteFingerprint)

			// Policy testing
			dlp.POST("/test", handlers.TestDLPPolicy)
		}

		// Agent Management
		agents := v1.Group("/agents")
		{
			agents.GET("", handlers.ListAgents)
			agents.GET("/:id", handlers.GetAgent)
			agents.PUT("/:id", handlers.UpdateAgent)
			agents.DELETE("/:id", handlers.DeleteAgent)

			// Agent configuration
			agents.GET("/:id/config", handlers.GetAgentConfig)
			agents.PUT("/:id/config", handlers.UpdateAgentConfig)
		}

		// Telemetry Query Interface
		telemetry := v1.Group("/telemetry")
		{
			telemetry.POST("/query", handlers.QueryEvents)
			telemetry.GET("/events/:id", handlers.GetEvent)
			telemetry.GET("/statistics", handlers.GetStatistics)
		}

		// MITRE ATT&CK Framework
		mitre := v1.Group("/mitre")
		{
			mitre.GET("/tactics", handlers.ListMITRETactics)
			mitre.GET("/techniques", handlers.ListMITRETechniques)
			mitre.GET("/coverage", handlers.GetMITRECoverage)
		}

		// Alerting Rules
		alerts := v1.Group("/alerts")
		{
			alerts.GET("/rules", handlers.ListAlertRules)
			alerts.POST("/rules", handlers.CreateAlertRule)
			alerts.PUT("/rules/:id", handlers.UpdateAlertRule)
			alerts.DELETE("/rules/:id", handlers.DeleteAlertRule)
		}

		// License Management
		licenses := v1.Group("/licenses")
		{
			licenses.GET("", handlers.ListLicenses)
			licenses.GET("/:id", handlers.GetLicense)
			licenses.POST("", handlers.CreateLicense)
			licenses.POST("/validate", handlers.ValidateLicense)
			licenses.POST("/trial", handlers.GenerateTrialLicense)
			licenses.DELETE("/:id", handlers.RevokeLicense)
			licenses.GET("/:id/usage", handlers.GetLicenseUsage)
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
