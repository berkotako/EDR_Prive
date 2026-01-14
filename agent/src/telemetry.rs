// Telemetry Client - gRPC streaming to ingestor service
// Batches events and maintains persistent connection for high throughput.

use anyhow::{Result, Context};
use std::time::{SystemTime, UNIX_EPOCH};
use tokio::sync::mpsc;
use tracing::{info, error, debug};
use serde::{Serialize, Deserialize};

use crate::config::AgentConfig;

/// Event types matching the protobuf enum
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub enum EventType {
    Unspecified = 0,
    ProcessStart = 1,
    ProcessTerminate = 2,
    FileAccess = 3,
    FileModify = 4,
    FileDelete = 5,
    NetworkConn = 6,
    RegistryModify = 7,
    DlpViolation = 8,
    Authentication = 9,
}

/// Internal event representation (will be converted to protobuf)
#[derive(Debug, Clone)]
pub struct Event {
    pub agent_id: String,
    pub timestamp: i64,
    pub event_type: EventType,
    pub mitre_tactic: String,
    pub mitre_technique: String,
    pub severity: i32,
    pub payload: String, // JSON-encoded event details
    pub tenant_id: String,
    pub hostname: String,
    pub os_type: String,
}

impl Event {
    pub fn new(
        agent_id: String,
        event_type: EventType,
        mitre_tactic: String,
        payload: String,
    ) -> Self {
        let timestamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_millis() as i64;

        Self {
            agent_id,
            timestamp,
            event_type,
            mitre_tactic,
            mitre_technique: String::new(),
            severity: 1,
            payload,
            tenant_id: String::new(),
            hostname: hostname::get()
                .unwrap_or_default()
                .to_string_lossy()
                .to_string(),
            os_type: std::env::consts::OS.to_string(),
        }
    }
}

pub struct TelemetryClient {
    config: AgentConfig,
}

impl TelemetryClient {
    pub async fn new(config: AgentConfig) -> Result<Self> {
        Ok(Self { config })
    }

    /// Run the telemetry client, receiving events from the channel and streaming to ingestor.
    /// TODO: Implement actual gRPC streaming (requires generated protobuf code).
    pub async fn run(self, mut event_rx: mpsc::Receiver<Event>) -> Result<()> {
        info!("Telemetry client connected to: {}", self.config.ingestor_url);

        // TODO: Establish gRPC stream using tonic
        // let mut client = telemetry_client::TelemetryServiceClient::connect(self.config.ingestor_url).await?;
        // let stream = client.stream_events(...).await?;

        while let Some(event) = event_rx.recv().await {
            debug!(
                "Received event: type={:?}, mitre={}, payload_len={}",
                event.event_type,
                event.mitre_tactic,
                event.payload.len()
            );

            // TODO: Send event via gRPC stream
            // stream.send(event).await?;
        }

        Ok(())
    }
}
