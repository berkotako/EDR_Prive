// Sentinel-Enterprise EDR/DLP Agent
// Performance Target: <1% CPU, <50MB RAM
// Supports: Windows (ETW) and Linux (eBPF)

use anyhow::Result;
use std::sync::Arc;
use tokio::sync::mpsc;
use tracing::{info, error};
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

mod etw;
mod dlp;
mod telemetry;
mod config;

use crate::config::AgentConfig;
use crate::telemetry::{TelemetryClient, Event};

const EVENT_BUFFER_SIZE: usize = 10000;
const BATCH_SIZE: usize = 100;

#[tokio::main]
async fn main() -> Result<()> {
    // Initialize lightweight logging (filter by RUST_LOG env var)
    tracing_subscriber::registry()
        .with(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "sentinel_agent=info".into()),
        )
        .with(tracing_subscriber::fmt::layer())
        .init();

    info!("Sentinel-Enterprise Agent v{} starting...", env!("CARGO_PKG_VERSION"));

    // Load configuration (agent_id, server endpoint, tenant_id)
    let config = AgentConfig::load()?;
    info!("Agent ID: {}", config.agent_id);
    info!("Ingestor endpoint: {}", config.ingestor_url);

    // Create high-throughput channel for event batching
    let (event_tx, event_rx) = mpsc::channel::<Event>(EVENT_BUFFER_SIZE);

    // Initialize DLP engine with fingerprint hashset
    let dlp_engine = Arc::new(dlp::DlpEngine::new());
    info!("DLP engine initialized with {} fingerprints", dlp_engine.fingerprint_count());

    // Start telemetry client (gRPC stream to ingestor)
    let telemetry_client = TelemetryClient::new(config.clone()).await?;
    let telemetry_handle = tokio::spawn(async move {
        if let Err(e) = telemetry_client.run(event_rx).await {
            error!("Telemetry client error: {}", e);
        }
    });

    // Platform-specific event collection
    #[cfg(target_os = "windows")]
    {
        info!("Starting Windows ETW consumer...");
        let etw_tx = event_tx.clone();
        let etw_config = config.clone();
        let dlp_ref = dlp_engine.clone();

        tokio::task::spawn_blocking(move || {
            if let Err(e) = etw::start_consumer(etw_tx, etw_config, dlp_ref) {
                error!("ETW consumer error: {}", e);
            }
        });
    }

    #[cfg(target_os = "linux")]
    {
        info!("Starting Linux eBPF collectors...");
        let ebpf_tx = event_tx.clone();
        let ebpf_config = config.clone();
        let dlp_ref = dlp_engine.clone();

        tokio::spawn(async move {
            if let Err(e) = ebpf::start_collectors(ebpf_tx, ebpf_config, dlp_ref).await {
                error!("eBPF collector error: {}", e);
            }
        });
    }

    info!("Agent fully operational. Monitoring system events...");

    // Wait for telemetry client (runs indefinitely)
    telemetry_handle.await?;

    Ok(())
}
