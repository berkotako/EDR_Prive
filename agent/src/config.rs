// Agent configuration management
// Loads settings from environment variables or config file.

use anyhow::Result;
use serde::{Deserialize, Serialize};
use uuid::Uuid;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AgentConfig {
    /// Unique agent identifier (UUID v4)
    pub agent_id: String,

    /// Ingestor gRPC endpoint
    pub ingestor_url: String,

    /// Tenant/organization identifier for multi-tenancy
    pub tenant_id: String,

    /// Enable DLP scanning
    pub dlp_enabled: bool,

    /// Event batch size before transmission
    pub batch_size: usize,

    /// Maximum event buffer size (prevents memory exhaustion)
    pub max_buffer_size: usize,
}

impl AgentConfig {
    /// Load configuration from environment variables with defaults.
    pub fn load() -> Result<Self> {
        let agent_id = std::env::var("SENTINEL_AGENT_ID")
            .unwrap_or_else(|_| Uuid::new_v4().to_string());

        let ingestor_url = std::env::var("SENTINEL_INGESTOR_URL")
            .unwrap_or_else(|_| "http://127.0.0.1:50051".to_string());

        let tenant_id = std::env::var("SENTINEL_TENANT_ID")
            .unwrap_or_else(|_| "default".to_string());

        let dlp_enabled = std::env::var("SENTINEL_DLP_ENABLED")
            .unwrap_or_else(|_| "true".to_string())
            .parse::<bool>()
            .unwrap_or(true);

        let batch_size = std::env::var("SENTINEL_BATCH_SIZE")
            .unwrap_or_else(|_| "100".to_string())
            .parse::<usize>()
            .unwrap_or(100);

        let max_buffer_size = std::env::var("SENTINEL_MAX_BUFFER")
            .unwrap_or_else(|_| "10000".to_string())
            .parse::<usize>()
            .unwrap_or(10000);

        Ok(Self {
            agent_id,
            ingestor_url,
            tenant_id,
            dlp_enabled,
            batch_size,
            max_buffer_size,
        })
    }
}
