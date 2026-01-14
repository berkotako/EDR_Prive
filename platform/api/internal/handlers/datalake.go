// Security Data Lake Handler
// Manages cold storage of telemetry data in S3/GCS for compliance and long-term retention

package handlers

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/option"

	"github.com/sentinel-enterprise/platform/api/internal/models"
)

// DataLakeHandler handles data lake operations
type DataLakeHandler struct {
	db *sql.DB
}

// NewDataLakeHandler creates a new data lake handler
func NewDataLakeHandler(db *sql.DB) *DataLakeHandler {
	return &DataLakeHandler{db: db}
}

// CreateDataLakeConfig creates a new data lake configuration
func (h *DataLakeHandler) CreateDataLakeConfig(c *gin.Context) {
	var req models.CreateDataLakeConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate provider-specific requirements
	if err := h.validateProviderConfig(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	configID := uuid.New().String()

	// Store configuration (encrypt sensitive data in production)
	query := `
		INSERT INTO data_lake_configs (
			id, license_id, provider, enabled, bucket_name, region,
			access_key, secret_key, project_id, credentials_json,
			hot_storage_days, warm_storage_days, cold_storage_days,
			delete_after_days, compliance_mode, enable_auto_archive,
			compression_type, encryption_enabled, metadata
		) VALUES ($1, $2, $3, TRUE, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING created_at, updated_at
	`

	metadata, _ := json.Marshal(req.Metadata)
	var createdAt, updatedAt time.Time

	err := h.db.QueryRow(query,
		configID,
		req.LicenseID,
		req.Provider,
		req.BucketName,
		req.Region,
		req.AccessKey, // In production, encrypt with KMS
		req.SecretKey, // In production, encrypt with KMS
		req.ProjectID,
		req.CredentialsJSON, // In production, encrypt with KMS
		req.RetentionPolicy.HotStorageDays,
		req.RetentionPolicy.WarmStorageDays,
		req.RetentionPolicy.ColdStorageDays,
		req.RetentionPolicy.DeleteAfterDays,
		req.RetentionPolicy.ComplianceMode,
		req.RetentionPolicy.EnableAutoArchive,
		req.CompressionType,
		req.EncryptionEnabled,
		metadata,
	).Scan(&createdAt, &updatedAt)

	if err != nil {
		log.Errorf("Failed to create data lake config: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create configuration"})
		return
	}

	config := models.DataLakeConfig{
		ID:                configID,
		LicenseID:         req.LicenseID,
		Provider:          req.Provider,
		Enabled:           true,
		BucketName:        req.BucketName,
		Region:            req.Region,
		RetentionPolicy:   req.RetentionPolicy,
		CompressionType:   req.CompressionType,
		EncryptionEnabled: req.EncryptionEnabled,
		Metadata:          req.Metadata,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}

	c.JSON(http.StatusCreated, config)
}

// GetDataLakeConfig retrieves data lake configuration
func (h *DataLakeHandler) GetDataLakeConfig(c *gin.Context) {
	licenseID := c.Param("license_id")

	query := `
		SELECT id, license_id, provider, enabled, bucket_name, region,
		       hot_storage_days, warm_storage_days, cold_storage_days,
		       delete_after_days, compliance_mode, enable_auto_archive,
		       compression_type, encryption_enabled, metadata,
		       created_at, updated_at
		FROM data_lake_configs
		WHERE license_id = $1
	`

	var config models.DataLakeConfig
	var metadataJSON []byte
	var policy models.RetentionPolicy

	err := h.db.QueryRow(query, licenseID).Scan(
		&config.ID,
		&config.LicenseID,
		&config.Provider,
		&config.Enabled,
		&config.BucketName,
		&config.Region,
		&policy.HotStorageDays,
		&policy.WarmStorageDays,
		&policy.ColdStorageDays,
		&policy.DeleteAfterDays,
		&policy.ComplianceMode,
		&policy.EnableAutoArchive,
		&config.CompressionType,
		&config.EncryptionEnabled,
		&metadataJSON,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Configuration not found"})
		return
	}

	if err != nil {
		log.Errorf("Failed to get data lake config: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve configuration"})
		return
	}

	config.RetentionPolicy = policy
	json.Unmarshal(metadataJSON, &config.Metadata)

	c.JSON(http.StatusOK, config)
}

// UpdateDataLakeConfig updates data lake configuration
func (h *DataLakeHandler) UpdateDataLakeConfig(c *gin.Context) {
	licenseID := c.Param("license_id")

	var req models.UpdateDataLakeConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := `
		UPDATE data_lake_configs
		SET enabled = COALESCE($1, enabled),
		    hot_storage_days = COALESCE($2, hot_storage_days),
		    warm_storage_days = COALESCE($3, warm_storage_days),
		    cold_storage_days = COALESCE($4, cold_storage_days),
		    delete_after_days = COALESCE($5, delete_after_days),
		    compression_type = COALESCE($6, compression_type),
		    encryption_enabled = COALESCE($7, encryption_enabled),
		    updated_at = NOW()
		WHERE license_id = $8
	`

	var hotDays, warmDays, coldDays, deleteDays *int
	if req.RetentionPolicy != nil {
		hotDays = &req.RetentionPolicy.HotStorageDays
		warmDays = &req.RetentionPolicy.WarmStorageDays
		coldDays = &req.RetentionPolicy.ColdStorageDays
		deleteDays = &req.RetentionPolicy.DeleteAfterDays
	}

	result, err := h.db.Exec(query,
		req.Enabled,
		hotDays,
		warmDays,
		coldDays,
		deleteDays,
		req.CompressionType,
		req.EncryptionEnabled,
		licenseID,
	)

	if err != nil {
		log.Errorf("Failed to update data lake config: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update configuration"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Configuration not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configuration updated successfully"})
}

// CreateArchiveJob creates a new archive job
func (h *DataLakeHandler) CreateArchiveJob(c *gin.Context) {
	var req models.CreateArchiveJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	jobID := uuid.New().String()

	query := `
		INSERT INTO archive_jobs (
			id, license_id, job_type, status, start_time,
			source_location, target_location, metadata
		) VALUES ($1, $2, $3, $4, NOW(), $5, $6, $7)
		RETURNING created_at
	`

	metadata, _ := json.Marshal(req.Metadata)
	var createdAt time.Time

	sourceLocation := fmt.Sprintf("clickhouse://events/%s/%s",
		req.StartDate.Format("2006-01-02"),
		req.EndDate.Format("2006-01-02"))

	err := h.db.QueryRow(query,
		jobID,
		req.LicenseID,
		req.JobType,
		models.JobStatusPending,
		sourceLocation,
		req.TargetLocation,
		metadata,
	).Scan(&createdAt)

	if err != nil {
		log.Errorf("Failed to create archive job: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create archive job"})
		return
	}

	// In production, trigger background worker to process the job
	go h.processArchiveJob(jobID, req)

	job := models.ArchiveJob{
		ID:              jobID,
		LicenseID:       req.LicenseID,
		JobType:         req.JobType,
		Status:          models.JobStatusPending,
		StartTime:       time.Now(),
		EventsProcessed: 0,
		BytesProcessed:  0,
		SourceLocation:  sourceLocation,
		TargetLocation:  req.TargetLocation,
		Progress:        0.0,
		Metadata:        req.Metadata,
		CreatedAt:       createdAt,
		UpdatedAt:       createdAt,
	}

	c.JSON(http.StatusCreated, job)
}

// GetArchiveJob retrieves an archive job by ID
func (h *DataLakeHandler) GetArchiveJob(c *gin.Context) {
	jobID := c.Param("id")

	query := `
		SELECT id, license_id, job_type, status, start_time, end_time,
		       events_processed, bytes_processed, source_location,
		       target_location, error, progress, metadata,
		       created_at, updated_at
		FROM archive_jobs
		WHERE id = $1
	`

	var job models.ArchiveJob
	var metadataJSON []byte

	err := h.db.QueryRow(query, jobID).Scan(
		&job.ID,
		&job.LicenseID,
		&job.JobType,
		&job.Status,
		&job.StartTime,
		&job.EndTime,
		&job.EventsProcessed,
		&job.BytesProcessed,
		&job.SourceLocation,
		&job.TargetLocation,
		&job.Error,
		&job.Progress,
		&metadataJSON,
		&job.CreatedAt,
		&job.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	if err != nil {
		log.Errorf("Failed to get archive job: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve job"})
		return
	}

	json.Unmarshal(metadataJSON, &job.Metadata)

	c.JSON(http.StatusOK, job)
}

// ListArchiveJobs lists archive jobs for a license
func (h *DataLakeHandler) ListArchiveJobs(c *gin.Context) {
	licenseID := c.Query("license_id")
	status := c.Query("status")

	query := `
		SELECT id, license_id, job_type, status, start_time, end_time,
		       events_processed, bytes_processed, source_location,
		       target_location, error, progress, created_at, updated_at
		FROM archive_jobs
		WHERE license_id = $1
	`

	args := []interface{}{licenseID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		log.Errorf("Failed to list archive jobs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list jobs"})
		return
	}
	defer rows.Close()

	jobs := []models.ArchiveJob{}
	for rows.Next() {
		var job models.ArchiveJob
		err := rows.Scan(
			&job.ID,
			&job.LicenseID,
			&job.JobType,
			&job.Status,
			&job.StartTime,
			&job.EndTime,
			&job.EventsProcessed,
			&job.BytesProcessed,
			&job.SourceLocation,
			&job.TargetLocation,
			&job.Error,
			&job.Progress,
			&job.CreatedAt,
			&job.UpdatedAt,
		)
		if err != nil {
			continue
		}
		jobs = append(jobs, job)
	}

	c.JSON(http.StatusOK, gin.H{
		"jobs":  jobs,
		"count": len(jobs),
	})
}

// ListArchivedDatasets lists archived datasets
func (h *DataLakeHandler) ListArchivedDatasets(c *gin.Context) {
	licenseID := c.Query("license_id")

	query := `
		SELECT id, license_id, dataset_name, storage_path,
		       start_date, end_date, event_count, compressed_size,
		       original_size, compression_type, is_encrypted,
		       checksum, storage_class, expires_at, metadata,
		       archived_at
		FROM archived_datasets
		WHERE license_id = $1
		ORDER BY archived_at DESC
		LIMIT 100
	`

	rows, err := h.db.Query(query, licenseID)
	if err != nil {
		log.Errorf("Failed to list archived datasets: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list datasets"})
		return
	}
	defer rows.Close()

	datasets := []models.ArchivedDataset{}
	for rows.Next() {
		var dataset models.ArchivedDataset
		var metadataJSON []byte

		err := rows.Scan(
			&dataset.ID,
			&dataset.LicenseID,
			&dataset.DatasetName,
			&dataset.StoragePath,
			&dataset.StartDate,
			&dataset.EndDate,
			&dataset.EventCount,
			&dataset.CompressedSize,
			&dataset.OriginalSize,
			&dataset.CompressionType,
			&dataset.IsEncrypted,
			&dataset.Checksum,
			&dataset.StorageClass,
			&dataset.ExpiresAt,
			&metadataJSON,
			&dataset.ArchivedAt,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(metadataJSON, &dataset.Metadata)
		datasets = append(datasets, dataset)
	}

	c.JSON(http.StatusOK, gin.H{
		"datasets": datasets,
		"count":    len(datasets),
	})
}

// QueryArchivedData queries data from archived datasets
func (h *DataLakeHandler) QueryArchivedData(c *gin.Context) {
	var req models.QueryArchivedDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startTime := time.Now()

	// Get relevant datasets
	query := `
		SELECT id, storage_path, compressed_size
		FROM archived_datasets
		WHERE license_id = $1
		  AND start_date <= $2
		  AND end_date >= $3
		ORDER BY start_date
	`

	rows, err := h.db.Query(query, req.LicenseID, req.EndDate, req.StartDate)
	if err != nil {
		log.Errorf("Failed to query datasets: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query datasets"})
		return
	}
	defer rows.Close()

	var datasetPaths []string
	var totalSize int64

	for rows.Next() {
		var id, path string
		var size int64
		if err := rows.Scan(&id, &path, &size); err != nil {
			continue
		}
		datasetPaths = append(datasetPaths, path)
		totalSize += size
	}

	if len(datasetPaths) == 0 {
		c.JSON(http.StatusOK, models.QueryArchivedDataResponse{
			Results:         []map[string]interface{}{},
			TotalEvents:     0,
			DatasetsQueried: 0,
			QueryTimeMs:     time.Since(startTime).Milliseconds(),
		})
		return
	}

	// In production, implement actual querying from S3/GCS
	// This is a placeholder response
	results := []map[string]interface{}{
		{
			"message": "Archived data query not fully implemented",
			"datasets_found": len(datasetPaths),
			"total_size_bytes": totalSize,
		},
	}

	queryTime := time.Since(startTime).Milliseconds()

	response := models.QueryArchivedDataResponse{
		Results:         results,
		TotalEvents:     0,
		DatasetsQueried: len(datasetPaths),
		QueryTimeMs:     queryTime,
		DataScannedGB:   float64(totalSize) / (1024 * 1024 * 1024),
	}

	c.JSON(http.StatusOK, response)
}

// GetDataLakeStatistics retrieves statistics about archived data
func (h *DataLakeHandler) GetDataLakeStatistics(c *gin.Context) {
	licenseID := c.Query("license_id")

	// Get dataset statistics
	query := `
		SELECT COUNT(*),
		       COALESCE(SUM(event_count), 0),
		       COALESCE(SUM(compressed_size), 0),
		       COALESCE(SUM(original_size), 0),
		       MIN(archived_at),
		       MAX(archived_at)
		FROM archived_datasets
		WHERE license_id = $1
	`

	var stats models.DataLakeStatistics
	stats.LicenseID = licenseID

	var oldestArchive, newestArchive sql.NullTime
	err := h.db.QueryRow(query, licenseID).Scan(
		&stats.TotalDatasets,
		&stats.TotalEvents,
		&stats.TotalStorageBytes,
		&stats.TotalOriginalBytes,
		&oldestArchive,
		&newestArchive,
	)

	if err != nil {
		log.Errorf("Failed to get statistics: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve statistics"})
		return
	}

	if oldestArchive.Valid {
		stats.OldestArchive = oldestArchive.Time
	}
	if newestArchive.Valid {
		stats.NewestArchive = newestArchive.Time
	}

	if stats.TotalOriginalBytes > 0 {
		stats.AverageCompression = float64(stats.TotalStorageBytes) / float64(stats.TotalOriginalBytes)
	}

	// Get job statistics
	jobQuery := `
		SELECT
			COUNT(CASE WHEN status = 'pending' THEN 1 END),
			COUNT(CASE WHEN status = 'completed' THEN 1 END),
			COUNT(CASE WHEN status = 'failed' THEN 1 END)
		FROM archive_jobs
		WHERE license_id = $1
	`

	h.db.QueryRow(jobQuery, licenseID).Scan(
		&stats.PendingArchiveJobs,
		&stats.CompletedArchiveJobs,
		&stats.FailedArchiveJobs,
	)

	// Estimate monthly cost (placeholder calculation)
	storageGB := float64(stats.TotalStorageBytes) / (1024 * 1024 * 1024)
	stats.EstimatedMonthlyCost = storageGB * 0.023 // $0.023/GB for S3 Standard

	c.JSON(http.StatusOK, stats)
}

// TestDataLakeConnection tests connectivity to data lake
func (h *DataLakeHandler) TestDataLakeConnection(c *gin.Context) {
	var req models.TestDataLakeConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startTime := time.Now()

	switch req.Provider {
	case models.ProviderS3:
		result := h.testS3Connection(req)
		result.Latency = time.Since(startTime).Milliseconds()
		c.JSON(http.StatusOK, result)
	case models.ProviderGCS:
		result := h.testGCSConnection(req)
		result.Latency = time.Since(startTime).Milliseconds()
		c.JSON(http.StatusOK, result)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported provider"})
	}
}

// Helper functions

func (h *DataLakeHandler) validateProviderConfig(req *models.CreateDataLakeConfigRequest) error {
	switch req.Provider {
	case models.ProviderS3:
		if req.AccessKey == "" || req.SecretKey == "" {
			return fmt.Errorf("access_key and secret_key required for S3")
		}
	case models.ProviderGCS:
		if req.ProjectID == "" || req.CredentialsJSON == "" {
			return fmt.Errorf("project_id and credentials_json required for GCS")
		}
	default:
		return fmt.Errorf("unsupported provider: %s", req.Provider)
	}
	return nil
}

func (h *DataLakeHandler) testS3Connection(req models.TestDataLakeConnectionRequest) models.TestDataLakeConnectionResponse {
	ctx := context.Background()

	// Create AWS config
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(req.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			req.AccessKey,
			req.SecretKey,
			"",
		)),
	)

	if err != nil {
		return models.TestDataLakeConnectionResponse{
			Success:  false,
			Message:  "Failed to create AWS config",
			Error:    err.Error(),
			TestedAt: time.Now(),
		}
	}

	client := s3.NewFromConfig(cfg)

	// Test bucket access
	_, err = client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(req.BucketName),
	})

	if err != nil {
		return models.TestDataLakeConnectionResponse{
			Success:      false,
			Message:      "Failed to access bucket",
			BucketExists: false,
			Error:        err.Error(),
			TestedAt:     time.Now(),
		}
	}

	// Test write permission
	testKey := fmt.Sprintf("_test_%d.txt", time.Now().Unix())
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(req.BucketName),
		Key:    aws.String(testKey),
		Body:   bytes.NewReader([]byte("test")),
	})

	canWrite := err == nil

	// Clean up test file
	if canWrite {
		client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(req.BucketName),
			Key:    aws.String(testKey),
		})
	}

	return models.TestDataLakeConnectionResponse{
		Success:      true,
		Message:      "Successfully connected to S3",
		BucketExists: true,
		CanWrite:     canWrite,
		CanRead:      true,
		TestedAt:     time.Now(),
	}
}

func (h *DataLakeHandler) testGCSConnection(req models.TestDataLakeConnectionRequest) models.TestDataLakeConnectionResponse {
	ctx := context.Background()

	// Create GCS client
	client, err := storage.NewClient(ctx, option.WithCredentialsJSON([]byte(req.CredentialsJSON)))
	if err != nil {
		return models.TestDataLakeConnectionResponse{
			Success:  false,
			Message:  "Failed to create GCS client",
			Error:    err.Error(),
			TestedAt: time.Now(),
		}
	}
	defer client.Close()

	bucket := client.Bucket(req.BucketName)

	// Test bucket access
	_, err = bucket.Attrs(ctx)
	if err != nil {
		return models.TestDataLakeConnectionResponse{
			Success:      false,
			Message:      "Failed to access bucket",
			BucketExists: false,
			Error:        err.Error(),
			TestedAt:     time.Now(),
		}
	}

	// Test write permission
	testKey := fmt.Sprintf("_test_%d.txt", time.Now().Unix())
	writer := bucket.Object(testKey).NewWriter(ctx)
	_, err = writer.Write([]byte("test"))
	writer.Close()

	canWrite := err == nil

	// Clean up test file
	if canWrite {
		bucket.Object(testKey).Delete(ctx)
	}

	return models.TestDataLakeConnectionResponse{
		Success:      true,
		Message:      "Successfully connected to GCS",
		BucketExists: true,
		CanWrite:     canWrite,
		CanRead:      true,
		TestedAt:     time.Now(),
	}
}

func (h *DataLakeHandler) processArchiveJob(jobID string, req models.CreateArchiveJobRequest) {
	// Update job status to running
	h.db.Exec("UPDATE archive_jobs SET status = $1 WHERE id = $2", models.JobStatusRunning, jobID)

	// In production, implement actual archiving logic:
	// 1. Query events from ClickHouse for date range
	// 2. Compress data
	// 3. Calculate checksum
	// 4. Upload to S3/GCS
	// 5. Create archived_dataset record
	// 6. Optionally delete from hot storage

	// Placeholder: mark as completed after 5 seconds
	time.Sleep(5 * time.Second)

	endTime := time.Now()
	h.db.Exec(`
		UPDATE archive_jobs
		SET status = $1, end_time = $2, progress = 1.0, updated_at = NOW()
		WHERE id = $3
	`, models.JobStatusCompleted, endTime, jobID)

	log.Infof("Archive job %s completed", jobID)
}

func compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	_, err := writer.Write(data)
	if err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func decompressData(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}
