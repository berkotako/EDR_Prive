-- ClickHouse Schema for Sentinel-Enterprise EDR/DLP Platform
-- Optimized for high-throughput ingestion (10,000+ events/sec) and fast analytical queries
-- Retention: Configurable via TTL (default: 90 days)

-- Drop existing table if re-deploying (CAUTION: data loss)
-- DROP TABLE IF EXISTS telemetry_events;

-- Main telemetry events table using MergeTree engine
-- Partitioned by month for efficient data management and pruning
CREATE TABLE IF NOT EXISTS telemetry_events
(
    -- Event identification and timing
    event_id            UUID DEFAULT generateUUIDv4(),
    agent_id            String,
    tenant_id           String,
    timestamp           DateTime64(3) DEFAULT now64(3), -- Millisecond precision
    server_timestamp    DateTime64(3) DEFAULT now64(3),

    -- Event classification
    event_type          Enum8(
        'unspecified' = 0,
        'process_start' = 1,
        'process_terminate' = 2,
        'file_access' = 3,
        'file_modify' = 4,
        'file_delete' = 5,
        'network_conn' = 6,
        'registry_modify' = 7,
        'dlp_violation' = 8,
        'authentication' = 9
    ),

    -- MITRE ATT&CK framework mapping for threat hunting
    mitre_tactic        LowCardinality(String),  -- e.g., "TA0002_Execution"
    mitre_technique     LowCardinality(String),  -- e.g., "T1059_Command_and_Scripting"
    severity            UInt8,                   -- 0=info, 1=low, 2=medium, 3=high, 4=critical

    -- System context
    hostname            LowCardinality(String),
    os_type             LowCardinality(String),  -- windows, linux, macos

    -- Event-specific payload (JSON for flexibility)
    -- Schema varies by event_type:
    --   PROCESS_START: {"pid":1234,"ppid":500,"cmdline":"...","user":"...","hash":"..."}
    --   FILE_ACCESS: {"path":"...","operation":"read","hash":"...","size":1024}
    --   NETWORK_CONN: {"src_ip":"...","dst_ip":"...","dst_port":443,"protocol":"tcp"}
    --   DLP_VIOLATION: {"rule_id":"...","matched_pattern":"...","file_path":"..."}
    payload             String,

    -- Extracted fields for fast filtering (materialized from JSON payload)
    -- These are computed on INSERT for better query performance
    process_name        String MATERIALIZED JSONExtractString(payload, 'process_name'),
    file_path           String MATERIALIZED JSONExtractString(payload, 'path'),
    dst_ip              String MATERIALIZED JSONExtractString(payload, 'dst_ip'),
    dst_port            UInt16 MATERIALIZED JSONExtractUInt(payload, 'dst_port'),
    username            String MATERIALIZED JSONExtractString(payload, 'user'),

    -- Indexing metadata
    ingestion_date      Date MATERIALIZED toDate(server_timestamp)
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)  -- Monthly partitions for efficient TTL and queries
ORDER BY (tenant_id, timestamp, event_type, agent_id, event_id)
TTL timestamp + INTERVAL 90 DAY   -- Auto-delete events older than 90 days
SETTINGS
    index_granularity = 8192,               -- Default granularity (good for most workloads)
    ttl_only_drop_parts = 1,                -- Drop entire partitions when TTL expires (faster)
    merge_with_ttl_timeout = 3600,          -- Merge parts with expired TTL hourly
    min_bytes_for_wide_part = 10485760,     -- Use wide format for parts >10MB (better compression)
    min_rows_for_wide_part = 100000;

-- Secondary index for fast MITRE ATT&CK lookups
ALTER TABLE telemetry_events ADD INDEX idx_mitre_tactic mitre_tactic TYPE set(100) GRANULARITY 4;
ALTER TABLE telemetry_events ADD INDEX idx_mitre_technique mitre_technique TYPE set(1000) GRANULARITY 4;

-- Bloom filter index for hostname and process name filtering
ALTER TABLE telemetry_events ADD INDEX idx_hostname hostname TYPE bloom_filter(0.01) GRANULARITY 4;
ALTER TABLE telemetry_events ADD INDEX idx_process process_name TYPE bloom_filter(0.01) GRANULARITY 4;

-- Create materialized view for real-time aggregations (optional - for dashboard performance)
CREATE MATERIALIZED VIEW IF NOT EXISTS events_hourly
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(event_hour)
ORDER BY (tenant_id, event_hour, event_type, hostname)
AS SELECT
    tenant_id,
    toStartOfHour(timestamp) AS event_hour,
    event_type,
    hostname,
    count() AS event_count,
    countIf(severity >= 3) AS high_severity_count
FROM telemetry_events
GROUP BY tenant_id, event_hour, event_type, hostname;

-- Create table for DLP policy fingerprints (used by agent for Exact Data Match)
CREATE TABLE IF NOT EXISTS dlp_fingerprints
(
    fingerprint_id      UUID DEFAULT generateUUIDv4(),
    tenant_id           String,
    rule_id             String,
    rule_name           String,
    severity            UInt8,
    fingerprint_hash    String,  -- BLAKE3 or SHA-256 hash of sensitive data chunk
    created_at          DateTime DEFAULT now(),
    updated_at          DateTime DEFAULT now()
)
ENGINE = ReplacingMergeTree(updated_at)
ORDER BY (tenant_id, fingerprint_hash, rule_id)
SETTINGS index_granularity = 8192;

-- Create index for fast fingerprint lookups
ALTER TABLE dlp_fingerprints ADD INDEX idx_fingerprint_hash fingerprint_hash TYPE bloom_filter(0.001) GRANULARITY 1;

-- Create table for agent metadata and health tracking
CREATE TABLE IF NOT EXISTS agents
(
    agent_id            String,
    tenant_id           String,
    hostname            String,
    os_type             LowCardinality(String),
    agent_version       String,
    last_seen           DateTime DEFAULT now(),
    status              Enum8('active' = 1, 'inactive' = 2, 'offline' = 3),
    cpu_usage           Float32,
    memory_usage_mb     UInt32,
    events_sent         UInt64
)
ENGINE = ReplacingMergeTree(last_seen)
ORDER BY (tenant_id, agent_id)
SETTINGS index_granularity = 1024;

-- Create table for MITRE ATT&CK framework reference (for enrichment)
CREATE TABLE IF NOT EXISTS mitre_attack_reference
(
    tactic_id           String,
    tactic_name         String,
    technique_id        String,
    technique_name      String,
    sub_technique_id    String,
    description         String,
    platforms           Array(String),
    data_sources        Array(String)
)
ENGINE = MergeTree()
ORDER BY (tactic_id, technique_id)
SETTINGS index_granularity = 256;

-- Example query patterns for performance validation:

-- 1. Get all high-severity events for a tenant in the last hour
-- SELECT * FROM telemetry_events
-- WHERE tenant_id = 'acme-corp'
--   AND timestamp >= now() - INTERVAL 1 HOUR
--   AND severity >= 3
-- ORDER BY timestamp DESC
-- LIMIT 100;

-- 2. Find all process execution events matching MITRE T1059 (Command and Scripting)
-- SELECT
--     timestamp,
--     hostname,
--     process_name,
--     JSONExtractString(payload, 'cmdline') AS cmdline
-- FROM telemetry_events
-- WHERE tenant_id = 'acme-corp'
--   AND event_type = 'process_start'
--   AND mitre_technique LIKE 'T1059%'
--   AND timestamp >= now() - INTERVAL 24 HOUR
-- ORDER BY timestamp DESC;

-- 3. Aggregate events by type and severity (for dashboard)
-- SELECT
--     event_type,
--     severity,
--     count() AS event_count,
--     uniq(agent_id) AS affected_agents
-- FROM telemetry_events
-- WHERE tenant_id = 'acme-corp'
--   AND timestamp >= today()
-- GROUP BY event_type, severity
-- ORDER BY event_count DESC;

-- 4. DLP violations report
-- SELECT
--     hostname,
--     JSONExtractString(payload, 'rule_id') AS rule_id,
--     JSONExtractString(payload, 'file_path') AS file_path,
--     count() AS violation_count
-- FROM telemetry_events
-- WHERE tenant_id = 'acme-corp'
--   AND event_type = 'dlp_violation'
--   AND timestamp >= now() - INTERVAL 7 DAY
-- GROUP BY hostname, rule_id, file_path
-- ORDER BY violation_count DESC
-- LIMIT 50;

-- 5. Network connections to suspicious IPs (threat intel enrichment)
-- SELECT
--     timestamp,
--     hostname,
--     dst_ip,
--     dst_port,
--     JSONExtractString(payload, 'process_name') AS process
-- FROM telemetry_events
-- WHERE tenant_id = 'acme-corp'
--   AND event_type = 'network_conn'
--   AND dst_ip IN (
--       -- Replace with actual threat intel feed
--       '192.0.2.1', '198.51.100.1'
--   )
--   AND timestamp >= now() - INTERVAL 24 HOUR
-- ORDER BY timestamp DESC;

-- Performance optimization recommendations:
-- 1. Monitor partition sizes: SELECT partition, sum(rows), formatReadableSize(sum(bytes_on_disk))
--    FROM system.parts WHERE table = 'telemetry_events' GROUP BY partition ORDER BY partition;
--
-- 2. Check merge performance: SELECT * FROM system.merges WHERE table = 'telemetry_events';
--
-- 3. Monitor query performance: SELECT query, query_duration_ms FROM system.query_log
--    WHERE type = 'QueryFinish' AND query LIKE '%telemetry_events%'
--    ORDER BY event_time DESC LIMIT 10;
--
-- 4. Optimize for your specific query patterns by adjusting ORDER BY columns
--
-- 5. Consider replication for HA: ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/telemetry_events', '{replica}')
