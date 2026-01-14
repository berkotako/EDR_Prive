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
