// Security Data Lake Models
// Provides cold storage for telemetry data in S3/GCS for compliance and long-term retention

package models

import "time"

// DataLakeConfig represents the configuration for cold storage
type DataLakeConfig struct {
	ID                string                 `json:"id"`
	LicenseID         string                 `json:"license_id"`
	Provider          DataLakeProvider       `json:"provider"` // s3, gcs, azure_blob
	Enabled           bool                   `json:"enabled"`
	BucketName        string                 `json:"bucket_name"`
	Region            string                 `json:"region,omitempty"`
	AccessKey         string                 `json:"access_key,omitempty"` // Stored encrypted
	SecretKey         string                 `json:"secret_key,omitempty"` // Stored encrypted
	ProjectID         string                 `json:"project_id,omitempty"` // For GCS
	CredentialsJSON   string                 `json:"credentials_json,omitempty"`
	RetentionPolicy   RetentionPolicy        `json:"retention_policy"`
	CompressionType   string                 `json:"compression_type"` // gzip, zstd, none
	EncryptionEnabled bool                   `json:"encryption_enabled"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// DataLakeProvider represents supported cloud storage providers
type DataLakeProvider string

const (
	ProviderS3        DataLakeProvider = "s3"
	ProviderGCS       DataLakeProvider = "gcs"
	ProviderAzureBlob DataLakeProvider = "azure_blob"
)

// RetentionPolicy defines how long data should be retained
type RetentionPolicy struct {
	HotStorageDays    int  `json:"hot_storage_days"`    // Days in ClickHouse
	WarmStorageDays   int  `json:"warm_storage_days"`   // Days in compressed format
	ColdStorageDays   int  `json:"cold_storage_days"`   // Days in archive storage
	DeleteAfterDays   int  `json:"delete_after_days"`   // Total retention period
	ComplianceMode    bool `json:"compliance_mode"`     // Prevent early deletion
	EnableAutoArchive bool `json:"enable_auto_archive"` // Automatically archive old data
}

// CreateDataLakeConfigRequest is the request to configure data lake
type CreateDataLakeConfigRequest struct {
	LicenseID         string                 `json:"license_id" binding:"required"`
	Provider          DataLakeProvider       `json:"provider" binding:"required"`
	BucketName        string                 `json:"bucket_name" binding:"required"`
	Region            string                 `json:"region"`
	AccessKey         string                 `json:"access_key"`
	SecretKey         string                 `json:"secret_key"`
	ProjectID         string                 `json:"project_id"`
	CredentialsJSON   string                 `json:"credentials_json"`
	RetentionPolicy   RetentionPolicy        `json:"retention_policy" binding:"required"`
	CompressionType   string                 `json:"compression_type"`
	EncryptionEnabled bool                   `json:"encryption_enabled"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// UpdateDataLakeConfigRequest is the request to update data lake configuration
type UpdateDataLakeConfigRequest struct {
	Enabled           *bool            `json:"enabled"`
	RetentionPolicy   *RetentionPolicy `json:"retention_policy"`
	CompressionType   *string          `json:"compression_type"`
	EncryptionEnabled *bool            `json:"encryption_enabled"`
}

// ArchiveJob represents a data archival job
type ArchiveJob struct {
	ID               string           `json:"id"`
	LicenseID        string           `json:"license_id"`
	JobType          ArchiveJobType   `json:"job_type"` // archive, restore, delete
	Status           ArchiveJobStatus `json:"status"`
	StartTime        time.Time        `json:"start_time"`
	EndTime          *time.Time       `json:"end_time,omitempty"`
	EventsProcessed  int64            `json:"events_processed"`
	BytesProcessed   int64            `json:"bytes_processed"`
	SourceLocation   string           `json:"source_location"`
	TargetLocation   string           `json:"target_location"`
	Error            string           `json:"error,omitempty"`
	Progress         float64          `json:"progress"` // 0.0 to 1.0
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// ArchiveJobType defines the type of archive operation
type ArchiveJobType string

const (
	JobTypeArchive ArchiveJobType = "archive"
	JobTypeRestore ArchiveJobType = "restore"
	JobTypeDelete  ArchiveJobType = "delete"
)

// ArchiveJobStatus represents the status of an archive job
type ArchiveJobStatus string

const (
	JobStatusPending    ArchiveJobStatus = "pending"
	JobStatusRunning    ArchiveJobStatus = "running"
	JobStatusCompleted  ArchiveJobStatus = "completed"
	JobStatusFailed     ArchiveJobStatus = "failed"
	JobStatusCancelled  ArchiveJobStatus = "cancelled"
)

// ArchivedDataset represents a collection of archived data
type ArchivedDataset struct {
	ID              string                 `json:"id"`
	LicenseID       string                 `json:"license_id"`
	DatasetName     string                 `json:"dataset_name"`
	StoragePath     string                 `json:"storage_path"`
	StartDate       time.Time              `json:"start_date"`
	EndDate         time.Time              `json:"end_date"`
	EventCount      int64                  `json:"event_count"`
	CompressedSize  int64                  `json:"compressed_size"` // Bytes
	OriginalSize    int64                  `json:"original_size"`   // Bytes
	CompressionType string                 `json:"compression_type"`
	IsEncrypted     bool                   `json:"is_encrypted"`
	Checksum        string                 `json:"checksum"` // SHA256
	StorageClass    string                 `json:"storage_class"` // STANDARD, GLACIER, etc.
	ExpiresAt       *time.Time             `json:"expires_at,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	ArchivedAt      time.Time              `json:"archived_at"`
}

// CreateArchiveJobRequest is the request to create an archive job
type CreateArchiveJobRequest struct {
	LicenseID      string                 `json:"license_id" binding:"required"`
	JobType        ArchiveJobType         `json:"job_type" binding:"required"`
	StartDate      time.Time              `json:"start_date" binding:"required"`
	EndDate        time.Time              `json:"end_date" binding:"required"`
	TargetLocation string                 `json:"target_location"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// QueryArchivedDataRequest is the request to query archived data
type QueryArchivedDataRequest struct {
	LicenseID      string                 `json:"license_id" binding:"required"`
	DatasetIDs     []string               `json:"dataset_ids"`
	StartDate      time.Time              `json:"start_date" binding:"required"`
	EndDate        time.Time              `json:"end_date" binding:"required"`
	Query          string                 `json:"query"` // SQL-like query
	Filters        map[string]interface{} `json:"filters,omitempty"`
	Limit          int                    `json:"limit"`
	IncludeMetrics bool                   `json:"include_metrics"`
}

// QueryArchivedDataResponse is the response from querying archived data
type QueryArchivedDataResponse struct {
	Results         []map[string]interface{} `json:"results"`
	TotalEvents     int64                    `json:"total_events"`
	DatasetsQueried int                      `json:"datasets_queried"`
	QueryTimeMs     int64                    `json:"query_time_ms"`
	DataScannedGB   float64                  `json:"data_scanned_gb"`
	Metrics         *QueryMetrics            `json:"metrics,omitempty"`
}

// QueryMetrics provides detailed query performance metrics
type QueryMetrics struct {
	DownloadTimeMs   int64   `json:"download_time_ms"`
	DecompressionMs  int64   `json:"decompression_time_ms"`
	FilteringMs      int64   `json:"filtering_time_ms"`
	BytesDownloaded  int64   `json:"bytes_downloaded"`
	BytesScanned     int64   `json:"bytes_scanned"`
	CompressionRatio float64 `json:"compression_ratio"`
}

// DataLakeStatistics provides statistics about archived data
type DataLakeStatistics struct {
	LicenseID             string    `json:"license_id"`
	TotalDatasets         int       `json:"total_datasets"`
	TotalEvents           int64     `json:"total_events"`
	TotalStorageBytes     int64     `json:"total_storage_bytes"`
	TotalOriginalBytes    int64     `json:"total_original_bytes"`
	AverageCompression    float64   `json:"average_compression"`
	OldestArchive         time.Time `json:"oldest_archive"`
	NewestArchive         time.Time `json:"newest_archive"`
	PendingArchiveJobs    int       `json:"pending_archive_jobs"`
	CompletedArchiveJobs  int       `json:"completed_archive_jobs"`
	FailedArchiveJobs     int       `json:"failed_archive_jobs"`
	EstimatedMonthlyCost  float64   `json:"estimated_monthly_cost"`
}

// ComplianceReport represents a compliance audit report
type ComplianceReport struct {
	ID                 string                 `json:"id"`
	LicenseID          string                 `json:"license_id"`
	ReportType         string                 `json:"report_type"` // gdpr, hipaa, sox, pci_dss
	StartDate          time.Time              `json:"start_date"`
	EndDate            time.Time              `json:"end_date"`
	DataRetention      string                 `json:"data_retention"`
	EncryptionStatus   string                 `json:"encryption_status"`
	AccessLogs         []AccessLogEntry       `json:"access_logs"`
	DeletionRequests   []DeletionRequest      `json:"deletion_requests,omitempty"`
	Findings           []ComplianceFinding    `json:"findings"`
	OverallStatus      string                 `json:"overall_status"` // compliant, non_compliant, warning
	GeneratedAt        time.Time              `json:"generated_at"`
	GeneratedBy        string                 `json:"generated_by"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// AccessLogEntry represents a data access log for compliance
type AccessLogEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	User        string    `json:"user"`
	Action      string    `json:"action"`
	DatasetID   string    `json:"dataset_id"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent,omitempty"`
}

// DeletionRequest represents a GDPR/privacy deletion request
type DeletionRequest struct {
	RequestID   string    `json:"request_id"`
	DataSubject string    `json:"data_subject"`
	RequestedAt time.Time `json:"requested_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Status      string    `json:"status"`
	RecordsDeleted int64  `json:"records_deleted"`
}

// ComplianceFinding represents an issue found during compliance check
type ComplianceFinding struct {
	Severity    string `json:"severity"` // critical, high, medium, low
	Category    string `json:"category"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

// TestDataLakeConnectionRequest is used to test data lake connectivity
type TestDataLakeConnectionRequest struct {
	Provider        DataLakeProvider `json:"provider" binding:"required"`
	BucketName      string           `json:"bucket_name" binding:"required"`
	Region          string           `json:"region"`
	AccessKey       string           `json:"access_key"`
	SecretKey       string           `json:"secret_key"`
	ProjectID       string           `json:"project_id"`
	CredentialsJSON string           `json:"credentials_json"`
}

// TestDataLakeConnectionResponse returns the result of connection test
type TestDataLakeConnectionResponse struct {
	Success      bool      `json:"success"`
	Message      string    `json:"message"`
	Latency      int64     `json:"latency_ms"`
	BucketExists bool      `json:"bucket_exists"`
	CanWrite     bool      `json:"can_write"`
	CanRead      bool      `json:"can_read"`
	Error        string    `json:"error,omitempty"`
	TestedAt     time.Time `json:"tested_at"`
}
