-- PostgreSQL Database Initialization for Privé Platform
-- This script creates all tables for metadata, licenses, and configuration

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ============================================================================
-- LICENSE MANAGEMENT TABLES
-- ============================================================================

-- Licenses table
CREATE TABLE IF NOT EXISTS licenses (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    license_key       TEXT UNIQUE NOT NULL,
    customer_email    VARCHAR(255) NOT NULL,
    customer_name     VARCHAR(255) NOT NULL,
    company_name      VARCHAR(255),
    tier              VARCHAR(50) NOT NULL CHECK (tier IN ('free', 'professional', 'enterprise')),
    max_agents        INTEGER NOT NULL,
    max_users         INTEGER NOT NULL,
    issued_at         TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at        TIMESTAMP,  -- NULL for perpetual licenses
    is_active         BOOLEAN DEFAULT TRUE,
    activated_at      TIMESTAMP,
    last_validated_at TIMESTAMP,
    metadata          JSONB DEFAULT '{}',
    created_at        TIMESTAMP DEFAULT NOW(),
    updated_at        TIMESTAMP DEFAULT NOW()
);

-- License usage tracking
CREATE TABLE IF NOT EXISTS license_usage (
    license_id       UUID PRIMARY KEY REFERENCES licenses(id) ON DELETE CASCADE,
    active_agents    INTEGER DEFAULT 0,
    active_users     INTEGER DEFAULT 0,
    events_ingested  BIGINT DEFAULT 0,
    storage_used_gb  NUMERIC(10, 2) DEFAULT 0,
    last_updated     TIMESTAMP DEFAULT NOW()
);

-- License activation history
CREATE TABLE IF NOT EXISTS license_activations (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    license_id      UUID REFERENCES licenses(id) ON DELETE CASCADE,
    agent_id        VARCHAR(255),
    hostname        VARCHAR(255),
    ip_address      INET,
    os_type         VARCHAR(50),
    activated_at    TIMESTAMP DEFAULT NOW(),
    deactivated_at  TIMESTAMP
);

-- License audit log
CREATE TABLE IF NOT EXISTS license_audit_log (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    license_id      UUID REFERENCES licenses(id) ON DELETE CASCADE,
    action          VARCHAR(100) NOT NULL,  -- created, validated, revoked, extended, etc.
    performed_by    VARCHAR(255),
    details         JSONB,
    created_at      TIMESTAMP DEFAULT NOW()
);

-- ============================================================================
-- USER MANAGEMENT TABLES
-- ============================================================================

-- Users table (for dashboard access)
CREATE TABLE IF NOT EXISTS users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email           VARCHAR(255) UNIQUE NOT NULL,
    password_hash   TEXT NOT NULL,
    full_name       VARCHAR(255),
    role            VARCHAR(50) NOT NULL CHECK (role IN ('admin', 'analyst', 'viewer')),
    license_id      UUID REFERENCES licenses(id) ON DELETE SET NULL,
    is_active       BOOLEAN DEFAULT TRUE,
    last_login      TIMESTAMP,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

-- User sessions
CREATE TABLE IF NOT EXISTS user_sessions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID REFERENCES users(id) ON DELETE CASCADE,
    token           TEXT UNIQUE NOT NULL,
    ip_address      INET,
    user_agent      TEXT,
    expires_at      TIMESTAMP NOT NULL,
    created_at      TIMESTAMP DEFAULT NOW()
);

-- ============================================================================
-- AGENT MANAGEMENT TABLES
-- ============================================================================

-- Agents table (metadata about deployed agents)
CREATE TABLE IF NOT EXISTS agents (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_id        VARCHAR(255) UNIQUE NOT NULL,
    license_id      UUID REFERENCES licenses(id) ON DELETE SET NULL,
    hostname        VARCHAR(255) NOT NULL,
    ip_address      INET,
    os_type         VARCHAR(50),
    os_version      VARCHAR(100),
    agent_version   VARCHAR(50),
    status          VARCHAR(50) CHECK (status IN ('active', 'inactive', 'offline', 'error')),
    last_seen       TIMESTAMP,
    cpu_usage       NUMERIC(5, 2),
    memory_usage_mb INTEGER,
    events_sent     BIGINT DEFAULT 0,
    config          JSONB DEFAULT '{}',
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

-- ============================================================================
-- DLP POLICY TABLES
-- ============================================================================

-- DLP policies
CREATE TABLE IF NOT EXISTS dlp_policies (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    license_id        UUID REFERENCES licenses(id) ON DELETE CASCADE,
    name              VARCHAR(255) NOT NULL,
    description       TEXT,
    severity          VARCHAR(50) CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    enabled           BOOLEAN DEFAULT TRUE,
    rule_type         VARCHAR(50) CHECK (rule_type IN ('fingerprint', 'regex', 'ml')),
    config            JSONB DEFAULT '{}',
    fingerprint_count INTEGER DEFAULT 0,
    created_at        TIMESTAMP DEFAULT NOW(),
    updated_at        TIMESTAMP DEFAULT NOW()
);

-- DLP fingerprints
CREATE TABLE IF NOT EXISTS dlp_fingerprints (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_id         UUID REFERENCES dlp_policies(id) ON DELETE CASCADE,
    fingerprint_hash  TEXT NOT NULL,
    source            VARCHAR(255),
    created_at        TIMESTAMP DEFAULT NOW()
);

-- ============================================================================
-- ALERT RULES TABLES
-- ============================================================================

-- Alert rules
CREATE TABLE IF NOT EXISTS alert_rules (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    license_id      UUID REFERENCES licenses(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    severity        VARCHAR(50),
    enabled         BOOLEAN DEFAULT TRUE,
    condition       JSONB NOT NULL,  -- Rule condition in JSON format
    actions         JSONB DEFAULT '[]',  -- Actions to take (email, webhook, etc.)
    created_by      UUID REFERENCES users(id),
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

-- Alert instances (fired alerts)
CREATE TABLE IF NOT EXISTS alert_instances (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rule_id         UUID REFERENCES alert_rules(id) ON DELETE CASCADE,
    agent_id        UUID REFERENCES agents(id),
    severity        VARCHAR(50),
    message         TEXT,
    details         JSONB,
    status          VARCHAR(50) CHECK (status IN ('open', 'investigating', 'resolved', 'false_positive')),
    assigned_to     UUID REFERENCES users(id),
    created_at      TIMESTAMP DEFAULT NOW(),
    resolved_at     TIMESTAMP
);

-- ============================================================================
-- MITRE ATT&CK REFERENCE TABLES
-- ============================================================================

-- MITRE tactics
CREATE TABLE IF NOT EXISTS mitre_tactics (
    tactic_id       VARCHAR(50) PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    url             TEXT
);

-- MITRE techniques
CREATE TABLE IF NOT EXISTS mitre_techniques (
    technique_id    VARCHAR(50) PRIMARY KEY,
    tactic_id       VARCHAR(50) REFERENCES mitre_tactics(tactic_id),
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    platforms       TEXT[],
    data_sources    TEXT[],
    url             TEXT
);

-- ============================================================================
-- AI ANALYSIS TABLES
-- ============================================================================

-- AI configuration per tenant
CREATE TABLE IF NOT EXISTS ai_configs (
    license_id       UUID PRIMARY KEY REFERENCES licenses(id) ON DELETE CASCADE,
    provider         VARCHAR(50) CHECK (provider IN ('openai', 'anthropic', 'local')),
    openai_key       TEXT,
    openai_model     VARCHAR(100) DEFAULT 'gpt-4',
    anthropic_key    TEXT,
    anthropic_model  VARCHAR(100) DEFAULT 'claude-3-5-sonnet-20241022',
    max_tokens       INTEGER DEFAULT 4096,
    temperature      NUMERIC(3, 2) DEFAULT 0.3,
    enabled          BOOLEAN DEFAULT TRUE,
    created_at       TIMESTAMP DEFAULT NOW(),
    updated_at       TIMESTAMP DEFAULT NOW()
);

-- AI analysis history
CREATE TABLE IF NOT EXISTS ai_analysis_history (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id       UUID REFERENCES licenses(id) ON DELETE CASCADE,
    analysis_type   VARCHAR(50) NOT NULL,
    provider        VARCHAR(50) NOT NULL,
    summary         TEXT NOT NULL,
    event_count     INTEGER NOT NULL,
    tokens_used     INTEGER DEFAULT 0,
    created_at      TIMESTAMP DEFAULT NOW(),
    created_by      VARCHAR(255)
);

-- ============================================================================
-- COLLABORATIVE THREAT HUNTING TABLES
-- ============================================================================

-- Shared detection rules from the community
CREATE TABLE IF NOT EXISTS shared_rules (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name                  VARCHAR(255) NOT NULL,
    description           TEXT,
    rule_type             VARCHAR(50) CHECK (rule_type IN ('yara', 'sigma', 'custom_query', 'alert_rule')),
    content               TEXT NOT NULL,
    mitre_tactics         TEXT[],
    mitre_techniques      TEXT[],
    author                VARCHAR(255) NOT NULL,  -- Can be anonymized
    submitter_license_id  UUID REFERENCES licenses(id) ON DELETE SET NULL,
    upvote_count          INTEGER DEFAULT 0,
    downvote_count        INTEGER DEFAULT 0,
    download_count        INTEGER DEFAULT 0,
    use_count             INTEGER DEFAULT 0,
    false_positive_rate   NUMERIC(5, 4),  -- e.g., 0.0150 = 1.5%
    is_verified           BOOLEAN DEFAULT FALSE,
    verified_by           VARCHAR(255),
    verified_at           TIMESTAMP,
    tags                  TEXT[],
    created_at            TIMESTAMP DEFAULT NOW(),
    updated_at            TIMESTAMP DEFAULT NOW()
);

-- Shared IOCs (Indicators of Compromise)
CREATE TABLE IF NOT EXISTS shared_iocs (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ioc_type           VARCHAR(50) CHECK (ioc_type IN ('ip', 'domain', 'hash', 'email', 'url', 'file_path', 'registry_key')),
    value              TEXT NOT NULL,
    threat_type        VARCHAR(100),  -- malware, phishing, c2, ransomware, etc.
    malware_family     VARCHAR(100),
    description        TEXT,
    confidence         NUMERIC(3, 2) CHECK (confidence >= 0 AND confidence <= 1),  -- 0.0 to 1.0
    severity           VARCHAR(50) CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    first_seen         TIMESTAMP,
    last_seen          TIMESTAMP,
    report_count       INTEGER DEFAULT 1,
    submitter_license_id UUID REFERENCES licenses(id) ON DELETE SET NULL,
    is_verified        BOOLEAN DEFAULT FALSE,
    verified_by        VARCHAR(255),
    verified_at        TIMESTAMP,
    mitre_techniques   TEXT[],
    tags               TEXT[],
    created_at         TIMESTAMP DEFAULT NOW(),
    updated_at         TIMESTAMP DEFAULT NOW()
);

-- Hunting queries shared by the community
CREATE TABLE IF NOT EXISTS hunting_queries (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name              VARCHAR(255) NOT NULL,
    description       TEXT,
    query             TEXT NOT NULL,
    query_language    VARCHAR(50) CHECK (query_language IN ('kql', 'spl', 'sql', 'custom')),
    category          VARCHAR(100),  -- persistence, lateral_movement, credential_access, etc.
    mitre_techniques  TEXT[],
    author            VARCHAR(255),
    submitter_license_id UUID REFERENCES licenses(id) ON DELETE SET NULL,
    upvote_count      INTEGER DEFAULT 0,
    use_count         INTEGER DEFAULT 0,
    rating            NUMERIC(3, 2) DEFAULT 0.0,  -- Average rating 0-5
    tags              TEXT[],
    created_at        TIMESTAMP DEFAULT NOW(),
    updated_at        TIMESTAMP DEFAULT NOW()
);

-- Rule votes (upvote/downvote tracking)
CREATE TABLE IF NOT EXISTS rule_votes (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rule_id         UUID REFERENCES shared_rules(id) ON DELETE CASCADE,
    license_id      UUID REFERENCES licenses(id) ON DELETE CASCADE,
    vote_type       VARCHAR(20) CHECK (vote_type IN ('upvote', 'downvote')),
    created_at      TIMESTAMP DEFAULT NOW(),
    UNIQUE(rule_id, license_id)  -- One vote per license per rule
);

-- Rule comments
CREATE TABLE IF NOT EXISTS rule_comments (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rule_id         UUID REFERENCES shared_rules(id) ON DELETE CASCADE,
    author          VARCHAR(255) NOT NULL,
    license_id      UUID REFERENCES licenses(id) ON DELETE SET NULL,
    comment         TEXT NOT NULL,
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

-- Rule downloads tracking
CREATE TABLE IF NOT EXISTS rule_downloads (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rule_id         UUID REFERENCES shared_rules(id) ON DELETE CASCADE,
    license_id      UUID REFERENCES licenses(id) ON DELETE CASCADE,
    downloaded_at   TIMESTAMP DEFAULT NOW()
);

-- IOC reports (reporting false positives or confirming accuracy)
CREATE TABLE IF NOT EXISTS ioc_reports (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ioc_id          UUID REFERENCES shared_iocs(id) ON DELETE CASCADE,
    license_id      UUID REFERENCES licenses(id) ON DELETE CASCADE,
    report_type     VARCHAR(50) CHECK (report_type IN ('confirmed', 'false_positive', 'additional_info')),
    comment         TEXT,
    created_at      TIMESTAMP DEFAULT NOW()
);

-- ============================================================================
-- SECURITY DATA LAKE TABLES
-- ============================================================================

-- Data lake configuration for cold storage
CREATE TABLE IF NOT EXISTS data_lake_configs (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    license_id            UUID UNIQUE REFERENCES licenses(id) ON DELETE CASCADE,
    provider              VARCHAR(50) CHECK (provider IN ('s3', 'gcs', 'azure_blob')),
    enabled               BOOLEAN DEFAULT TRUE,
    bucket_name           VARCHAR(255) NOT NULL,
    region                VARCHAR(100),
    access_key            TEXT,  -- Encrypt in production
    secret_key            TEXT,  -- Encrypt in production
    project_id            VARCHAR(255),
    credentials_json      TEXT,  -- Encrypt in production
    hot_storage_days      INTEGER DEFAULT 30,
    warm_storage_days     INTEGER DEFAULT 90,
    cold_storage_days     INTEGER DEFAULT 365,
    delete_after_days     INTEGER DEFAULT 2555,  -- 7 years for compliance
    compliance_mode       BOOLEAN DEFAULT FALSE,
    enable_auto_archive   BOOLEAN DEFAULT TRUE,
    compression_type      VARCHAR(50) DEFAULT 'gzip',
    encryption_enabled    BOOLEAN DEFAULT TRUE,
    metadata              JSONB DEFAULT '{}',
    created_at            TIMESTAMP DEFAULT NOW(),
    updated_at            TIMESTAMP DEFAULT NOW()
);

-- Archive jobs tracking
CREATE TABLE IF NOT EXISTS archive_jobs (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    license_id          UUID REFERENCES licenses(id) ON DELETE CASCADE,
    job_type            VARCHAR(50) CHECK (job_type IN ('archive', 'restore', 'delete')),
    status              VARCHAR(50) CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    start_time          TIMESTAMP NOT NULL,
    end_time            TIMESTAMP,
    events_processed    BIGINT DEFAULT 0,
    bytes_processed     BIGINT DEFAULT 0,
    source_location     TEXT NOT NULL,
    target_location     TEXT,
    error               TEXT,
    progress            NUMERIC(5, 4) DEFAULT 0.0,  -- 0.0 to 1.0
    metadata            JSONB DEFAULT '{}',
    created_at          TIMESTAMP DEFAULT NOW(),
    updated_at          TIMESTAMP DEFAULT NOW()
);

-- Archived datasets catalog
CREATE TABLE IF NOT EXISTS archived_datasets (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    license_id          UUID REFERENCES licenses(id) ON DELETE CASCADE,
    dataset_name        VARCHAR(255) NOT NULL,
    storage_path        TEXT NOT NULL,
    start_date          TIMESTAMP NOT NULL,
    end_date            TIMESTAMP NOT NULL,
    event_count         BIGINT NOT NULL,
    compressed_size     BIGINT NOT NULL,  -- Bytes
    original_size       BIGINT NOT NULL,  -- Bytes
    compression_type    VARCHAR(50),
    is_encrypted        BOOLEAN DEFAULT TRUE,
    checksum            VARCHAR(64),  -- SHA256 hash
    storage_class       VARCHAR(50),  -- STANDARD, GLACIER, etc.
    expires_at          TIMESTAMP,
    metadata            JSONB DEFAULT '{}',
    archived_at         TIMESTAMP DEFAULT NOW()
);

-- Data access logs for compliance
CREATE TABLE IF NOT EXISTS data_access_logs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    license_id      UUID REFERENCES licenses(id) ON DELETE CASCADE,
    user_id         UUID REFERENCES users(id) ON DELETE SET NULL,
    action          VARCHAR(100) NOT NULL,  -- query, download, restore, delete
    dataset_id      UUID REFERENCES archived_datasets(id) ON DELETE SET NULL,
    ip_address      INET,
    user_agent      TEXT,
    query_details   JSONB,
    accessed_at     TIMESTAMP DEFAULT NOW()
);

-- Compliance reports
CREATE TABLE IF NOT EXISTS compliance_reports (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    license_id          UUID REFERENCES licenses(id) ON DELETE CASCADE,
    report_type         VARCHAR(50),  -- gdpr, hipaa, sox, pci_dss
    start_date          TIMESTAMP NOT NULL,
    end_date            TIMESTAMP NOT NULL,
    overall_status      VARCHAR(50) CHECK (overall_status IN ('compliant', 'non_compliant', 'warning')),
    findings            JSONB DEFAULT '[]',
    generated_by        VARCHAR(255),
    generated_at        TIMESTAMP DEFAULT NOW(),
    metadata            JSONB DEFAULT '{}'
);

-- ============================================================================
-- DECEPTION TECHNOLOGY TABLES
-- ============================================================================

-- Honeypots
CREATE TABLE IF NOT EXISTS honeypots (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    license_id          UUID REFERENCES licenses(id) ON DELETE CASCADE,
    name                VARCHAR(255) NOT NULL,
    honeypot_type       VARCHAR(50) CHECK (honeypot_type IN ('ssh', 'smb', 'rdp', 'http', 'database', 'email_account', 'file_share', 'api_endpoint', 'credentials')),
    status              VARCHAR(50) CHECK (status IN ('active', 'inactive', 'compromised', 'deploying', 'error')),
    deployment_mode     VARCHAR(50) NOT NULL,  -- network, endpoint, cloud
    target_platform     VARCHAR(50) NOT NULL,  -- windows, linux, aws, azure
    configuration       JSONB DEFAULT '{}',
    location            TEXT,  -- IP address or endpoint ID
    is_active           BOOLEAN DEFAULT TRUE,
    interaction_count   INTEGER DEFAULT 0,
    last_interaction    TIMESTAMP,
    deployed_at         TIMESTAMP DEFAULT NOW(),
    metadata            JSONB DEFAULT '{}',
    created_at          TIMESTAMP DEFAULT NOW(),
    updated_at          TIMESTAMP DEFAULT NOW()
);

-- Honey tokens
CREATE TABLE IF NOT EXISTS honey_tokens (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    license_id      UUID REFERENCES licenses(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    token_type      VARCHAR(50) CHECK (token_type IN ('aws_key', 'api_key', 'database_creds', 'document_url', 'dns_query', 'email_address', 'web_bug', 'qr_code', 'office_document')),
    token_value     TEXT NOT NULL,
    callback_url    TEXT NOT NULL,
    is_active       BOOLEAN DEFAULT TRUE,
    access_count    INTEGER DEFAULT 0,
    last_accessed   TIMESTAMP,
    metadata        JSONB DEFAULT '{}',
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

-- Deception events
CREATE TABLE IF NOT EXISTS deception_events (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    license_id         UUID REFERENCES licenses(id) ON DELETE CASCADE,
    event_type         VARCHAR(50) CHECK (event_type IN ('honeypot_access', 'honeytoken_access', 'credential_attempt', 'file_access', 'network_scan')),
    honeypot_id        UUID REFERENCES honeypots(id) ON DELETE SET NULL,
    honey_token_id     UUID REFERENCES honey_tokens(id) ON DELETE SET NULL,
    source_ip          INET NOT NULL,
    source_hostname    VARCHAR(255),
    source_user        VARCHAR(255),
    interaction_type   VARCHAR(100) NOT NULL,  -- access, scan, exploit_attempt, credential_use
    severity           VARCHAR(50) CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    details            JSONB DEFAULT '{}',
    alert_created      BOOLEAN DEFAULT FALSE,
    alert_id           UUID REFERENCES alert_instances(id) ON DELETE SET NULL,
    metadata           JSONB DEFAULT '{}',
    detected_at        TIMESTAMP DEFAULT NOW()
);

-- Deception campaigns
CREATE TABLE IF NOT EXISTS deception_campaigns (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    license_id        UUID REFERENCES licenses(id) ON DELETE CASCADE,
    name              VARCHAR(255) NOT NULL,
    description       TEXT,
    status            VARCHAR(50) CHECK (status IN ('active', 'paused', 'completed')),
    honeypot_ids      UUID[],
    honey_token_ids   UUID[],
    start_date        TIMESTAMP NOT NULL,
    end_date          TIMESTAMP,
    event_count       INTEGER DEFAULT 0,
    threat_score      NUMERIC(5, 2) DEFAULT 0.0,
    objectives        TEXT[],
    metadata          JSONB DEFAULT '{}',
    created_at        TIMESTAMP DEFAULT NOW(),
    updated_at        TIMESTAMP DEFAULT NOW()
);

-- ============================================================================
-- NOTIFICATION CHANNELS TABLES
-- ============================================================================

-- Notification channels (Email, Slack, PagerDuty, Webhooks)
CREATE TABLE IF NOT EXISTS notification_channels (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    license_id      UUID REFERENCES licenses(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    type            VARCHAR(50) CHECK (type IN ('email', 'slack', 'pagerduty', 'webhook')),
    enabled         BOOLEAN DEFAULT TRUE,
    config          JSONB DEFAULT '{}',
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW()
);

-- Notification logs (audit trail of sent notifications)
CREATE TABLE IF NOT EXISTS notification_logs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    channel_id      UUID REFERENCES notification_channels(id) ON DELETE SET NULL,
    channel_type    VARCHAR(50),
    subject         TEXT NOT NULL,
    message         TEXT NOT NULL,
    priority        VARCHAR(50) CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    status          VARCHAR(50) CHECK (status IN ('sent', 'failed', 'pending')),
    error           TEXT,
    sent_at         TIMESTAMP DEFAULT NOW(),
    metadata        JSONB DEFAULT '{}'
);

-- ============================================================================
-- INDEXES FOR PERFORMANCE
-- ============================================================================

-- License indexes
CREATE INDEX idx_licenses_email ON licenses(customer_email);
CREATE INDEX idx_licenses_tier ON licenses(tier);
CREATE INDEX idx_licenses_active ON licenses(is_active);
CREATE INDEX idx_licenses_expires_at ON licenses(expires_at);

-- License activation indexes
CREATE INDEX idx_license_activations_license ON license_activations(license_id);
CREATE INDEX idx_license_activations_agent ON license_activations(agent_id);

-- User indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_license ON users(license_id);

-- Agent indexes
CREATE INDEX idx_agents_license ON agents(license_id);
CREATE INDEX idx_agents_status ON agents(status);
CREATE INDEX idx_agents_last_seen ON agents(last_seen);

-- DLP indexes
CREATE INDEX idx_dlp_policies_license ON dlp_policies(license_id);
CREATE INDEX idx_dlp_fingerprints_policy ON dlp_fingerprints(policy_id);
CREATE INDEX idx_dlp_fingerprints_hash ON dlp_fingerprints(fingerprint_hash);

-- Alert indexes
CREATE INDEX idx_alert_rules_license ON alert_rules(license_id);
CREATE INDEX idx_alert_instances_rule ON alert_instances(rule_id);
CREATE INDEX idx_alert_instances_status ON alert_instances(status);
CREATE INDEX idx_alert_instances_created ON alert_instances(created_at DESC);

-- Notification indexes
CREATE INDEX idx_notification_channels_license ON notification_channels(license_id);
CREATE INDEX idx_notification_channels_type ON notification_channels(type);
CREATE INDEX idx_notification_logs_channel ON notification_logs(channel_id);
CREATE INDEX idx_notification_logs_sent_at ON notification_logs(sent_at DESC);

-- AI indexes
CREATE INDEX idx_ai_analysis_tenant ON ai_analysis_history(tenant_id);
CREATE INDEX idx_ai_analysis_created ON ai_analysis_history(created_at DESC);
CREATE INDEX idx_ai_analysis_type ON ai_analysis_history(analysis_type);

-- Collaborative indexes
CREATE INDEX idx_shared_rules_type ON shared_rules(rule_type);
CREATE INDEX idx_shared_rules_verified ON shared_rules(is_verified);
CREATE INDEX idx_shared_rules_author ON shared_rules(author);
CREATE INDEX idx_shared_rules_created ON shared_rules(created_at DESC);
CREATE INDEX idx_shared_rules_upvotes ON shared_rules(upvote_count DESC);
CREATE INDEX idx_shared_rules_downloads ON shared_rules(download_count DESC);

CREATE INDEX idx_shared_iocs_type ON shared_iocs(ioc_type);
CREATE INDEX idx_shared_iocs_value ON shared_iocs(value);
CREATE INDEX idx_shared_iocs_threat_type ON shared_iocs(threat_type);
CREATE INDEX idx_shared_iocs_verified ON shared_iocs(is_verified);
CREATE INDEX idx_shared_iocs_created ON shared_iocs(created_at DESC);

CREATE INDEX idx_hunting_queries_category ON hunting_queries(category);
CREATE INDEX idx_hunting_queries_language ON hunting_queries(query_language);
CREATE INDEX idx_hunting_queries_created ON hunting_queries(created_at DESC);
CREATE INDEX idx_hunting_queries_rating ON hunting_queries(rating DESC);

CREATE INDEX idx_rule_votes_rule ON rule_votes(rule_id);
CREATE INDEX idx_rule_votes_license ON rule_votes(license_id);

CREATE INDEX idx_rule_comments_rule ON rule_comments(rule_id);
CREATE INDEX idx_rule_downloads_rule ON rule_downloads(rule_id);
CREATE INDEX idx_ioc_reports_ioc ON ioc_reports(ioc_id);

-- Data lake indexes
CREATE INDEX idx_data_lake_configs_license ON data_lake_configs(license_id);
CREATE INDEX idx_archive_jobs_license ON archive_jobs(license_id);
CREATE INDEX idx_archive_jobs_status ON archive_jobs(status);
CREATE INDEX idx_archive_jobs_created ON archive_jobs(created_at DESC);
CREATE INDEX idx_archived_datasets_license ON archived_datasets(license_id);
CREATE INDEX idx_archived_datasets_dates ON archived_datasets(start_date, end_date);
CREATE INDEX idx_archived_datasets_archived ON archived_datasets(archived_at DESC);
CREATE INDEX idx_data_access_logs_license ON data_access_logs(license_id);
CREATE INDEX idx_data_access_logs_accessed ON data_access_logs(accessed_at DESC);
CREATE INDEX idx_compliance_reports_license ON compliance_reports(license_id);

-- Deception indexes
CREATE INDEX idx_honeypots_license ON honeypots(license_id);
CREATE INDEX idx_honeypots_type ON honeypots(honeypot_type);
CREATE INDEX idx_honeypots_status ON honeypots(status);
CREATE INDEX idx_honeypots_active ON honeypots(is_active);
CREATE INDEX idx_honey_tokens_license ON honey_tokens(license_id);
CREATE INDEX idx_honey_tokens_type ON honey_tokens(token_type);
CREATE INDEX idx_honey_tokens_active ON honey_tokens(is_active);
CREATE INDEX idx_deception_events_license ON deception_events(license_id);
CREATE INDEX idx_deception_events_type ON deception_events(event_type);
CREATE INDEX idx_deception_events_honeypot ON deception_events(honeypot_id);
CREATE INDEX idx_deception_events_token ON deception_events(honey_token_id);
CREATE INDEX idx_deception_events_source_ip ON deception_events(source_ip);
CREATE INDEX idx_deception_events_detected ON deception_events(detected_at DESC);
CREATE INDEX idx_deception_campaigns_license ON deception_campaigns(license_id);
CREATE INDEX idx_deception_campaigns_status ON deception_campaigns(status);

-- ============================================================================
-- TRIGGERS FOR AUTOMATIC TIMESTAMPS
-- ============================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger to tables
CREATE TRIGGER update_licenses_updated_at BEFORE UPDATE ON licenses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_agents_updated_at BEFORE UPDATE ON agents
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_dlp_policies_updated_at BEFORE UPDATE ON dlp_policies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_alert_rules_updated_at BEFORE UPDATE ON alert_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_notification_channels_updated_at BEFORE UPDATE ON notification_channels
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_ai_configs_updated_at BEFORE UPDATE ON ai_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_shared_rules_updated_at BEFORE UPDATE ON shared_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_shared_iocs_updated_at BEFORE UPDATE ON shared_iocs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_hunting_queries_updated_at BEFORE UPDATE ON hunting_queries
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_rule_comments_updated_at BEFORE UPDATE ON rule_comments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_data_lake_configs_updated_at BEFORE UPDATE ON data_lake_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_archive_jobs_updated_at BEFORE UPDATE ON archive_jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_honeypots_updated_at BEFORE UPDATE ON honeypots
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_honey_tokens_updated_at BEFORE UPDATE ON honey_tokens
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_deception_campaigns_updated_at BEFORE UPDATE ON deception_campaigns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- SEED DATA FOR MITRE ATT&CK FRAMEWORK
-- ============================================================================

-- Insert MITRE tactics
INSERT INTO mitre_tactics (tactic_id, name, description, url) VALUES
    ('TA0001', 'Initial Access', 'Adversary is trying to get into your network', 'https://attack.mitre.org/tactics/TA0001'),
    ('TA0002', 'Execution', 'Adversary is trying to run malicious code', 'https://attack.mitre.org/tactics/TA0002'),
    ('TA0003', 'Persistence', 'Adversary is trying to maintain their foothold', 'https://attack.mitre.org/tactics/TA0003'),
    ('TA0004', 'Privilege Escalation', 'Adversary is trying to gain higher-level permissions', 'https://attack.mitre.org/tactics/TA0004'),
    ('TA0005', 'Defense Evasion', 'Adversary is trying to avoid being detected', 'https://attack.mitre.org/tactics/TA0005'),
    ('TA0006', 'Credential Access', 'Adversary is trying to steal account names and passwords', 'https://attack.mitre.org/tactics/TA0006'),
    ('TA0007', 'Discovery', 'Adversary is trying to figure out your environment', 'https://attack.mitre.org/tactics/TA0007'),
    ('TA0008', 'Lateral Movement', 'Adversary is trying to move through your environment', 'https://attack.mitre.org/tactics/TA0008'),
    ('TA0009', 'Collection', 'Adversary is trying to gather data of interest', 'https://attack.mitre.org/tactics/TA0009'),
    ('TA0011', 'Command and Control', 'Adversary is trying to communicate with compromised systems', 'https://attack.mitre.org/tactics/TA0011'),
    ('TA0010', 'Exfiltration', 'Adversary is trying to steal data', 'https://attack.mitre.org/tactics/TA0010'),
    ('TA0040', 'Impact', 'Adversary is trying to manipulate, interrupt, or destroy your systems and data', 'https://attack.mitre.org/tactics/TA0040')
ON CONFLICT (tactic_id) DO NOTHING;

-- Insert sample MITRE techniques
INSERT INTO mitre_techniques (technique_id, tactic_id, name, description, platforms, url) VALUES
    ('T1059', 'TA0002', 'Command and Scripting Interpreter', 'Adversaries may abuse command and script interpreters', ARRAY['Windows', 'Linux', 'macOS'], 'https://attack.mitre.org/techniques/T1059'),
    ('T1071', 'TA0011', 'Application Layer Protocol', 'Adversaries may communicate using application layer protocols', ARRAY['Windows', 'Linux', 'macOS'], 'https://attack.mitre.org/techniques/T1071'),
    ('T1566', 'TA0001', 'Phishing', 'Adversaries may send phishing messages to gain access', ARRAY['Windows', 'Linux', 'macOS'], 'https://attack.mitre.org/techniques/T1566'),
    ('T1547', 'TA0003', 'Boot or Logon Autostart Execution', 'Adversaries may configure system settings to automatically execute a program', ARRAY['Windows', 'Linux', 'macOS'], 'https://attack.mitre.org/techniques/T1547'),
    ('T1204', 'TA0002', 'User Execution', 'Adversaries may rely upon specific actions by a user to gain execution', ARRAY['Windows', 'Linux', 'macOS'], 'https://attack.mitre.org/techniques/T1204')
ON CONFLICT (technique_id) DO NOTHING;

-- ============================================================================
-- VIEWS FOR COMMON QUERIES
-- ============================================================================

-- Active licenses with usage
CREATE OR REPLACE VIEW active_licenses_with_usage AS
SELECT
    l.id,
    l.license_key,
    l.customer_email,
    l.customer_name,
    l.company_name,
    l.tier,
    l.max_agents,
    l.max_users,
    l.expires_at,
    lu.active_agents,
    lu.active_users,
    lu.events_ingested,
    lu.storage_used_gb,
    CASE
        WHEN l.expires_at IS NULL THEN -1
        WHEN l.expires_at > NOW() THEN EXTRACT(EPOCH FROM (l.expires_at - NOW())) / 86400
        ELSE 0
    END AS days_until_expiry
FROM licenses l
LEFT JOIN license_usage lu ON l.id = lu.license_id
WHERE l.is_active = TRUE;

-- Agent health summary
CREATE OR REPLACE VIEW agent_health_summary AS
SELECT
    l.id AS license_id,
    l.customer_name,
    COUNT(a.id) AS total_agents,
    COUNT(CASE WHEN a.status = 'active' THEN 1 END) AS active_agents,
    COUNT(CASE WHEN a.status = 'offline' THEN 1 END) AS offline_agents,
    COUNT(CASE WHEN a.last_seen < NOW() - INTERVAL '1 hour' THEN 1 END) AS stale_agents
FROM licenses l
LEFT JOIN agents a ON l.id = a.license_id
WHERE l.is_active = TRUE
GROUP BY l.id, l.customer_name;

-- ============================================================================
-- GRANT PERMISSIONS (adjust as needed)
-- ============================================================================

-- Create application user
CREATE USER prive_app WITH PASSWORD 'change_this_password';

-- Grant permissions
GRANT CONNECT ON DATABASE prive TO prive_app;
GRANT USAGE ON SCHEMA public TO prive_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO prive_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO prive_app;

-- Grant permissions on future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO prive_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO prive_app;

-- ============================================================================
-- COMPLETION MESSAGE
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE 'Privé PostgreSQL database initialized successfully!';
    RAISE NOTICE 'Tables created: licenses, license_usage, agents, dlp_policies, alert_rules, and more';
    RAISE NOTICE 'MITRE ATT&CK reference data loaded';
    RAISE NOTICE 'Views created for common queries';
    RAISE NOTICE '';
    RAISE NOTICE 'Next steps:';
    RAISE NOTICE '1. Change the prive_app user password';
    RAISE NOTICE '2. Configure DATABASE_URL in platform API';
    RAISE NOTICE '3. Generate license signing keys';
END $$;
