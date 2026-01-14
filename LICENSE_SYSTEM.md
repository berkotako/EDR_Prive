# Privé Licensing System

## Overview

Privé includes a comprehensive licensing engine with cryptographic key generation, validation, and tiered feature access control. The system is designed for commercial distribution with support for trials, subscriptions, and perpetual licenses.

---

## 🔑 License Tiers

### Free Tier
- **Agents**: 5 endpoints
- **Users**: 1 user
- **Features**:
  - ✅ EDR Monitoring (basic)
  - ❌ DLP Protection
  - ❌ Threat Hunting
  - ❌ Real-time Alerting
  - ❌ API Access

### Professional Tier
- **Agents**: 100 endpoints
- **Users**: 10 users
- **Features**:
  - ✅ EDR Monitoring
  - ✅ DLP Protection
  - ✅ Threat Hunting
  - ✅ Real-time Alerting
  - ✅ Custom Rules
  - ✅ API Access
  - ✅ Advanced Analytics
  - ✅ Threat Intelligence
  - ✅ Compliance Reporting

### Enterprise Tier
- **Agents**: Unlimited
- **Users**: Unlimited
- **Features**:
  - ✅ All Professional features
  - ✅ Multi-Tenancy
  - ✅ Incident Response
  - ✅ Priority Support (24/7)
  - ✅ Custom Integrations
  - ✅ Machine Learning
  - ✅ Dedicated Account Manager

---

## 🔐 License Key Format

License keys are cryptographically signed using Ed25519:

```
PRIVE-V1-XXXXX-XXXXX-XXXXX-XXXXX-XXXXX-XXXXX
```

**Structure:**
- `PRIVE`: Product identifier
- `V1`: Version number
- `XXXXX...`: Base64-encoded payload and signature

**Payload contains:**
- License ID (UUID)
- Customer email
- Tier (free/professional/enterprise)
- Issue timestamp
- Expiration timestamp (optional)
- Max agents allowed
- Cryptographic signature (Ed25519)

---

## 📡 API Endpoints

### Create License
```http
POST /api/v1/licenses
Content-Type: application/json

{
  "customer_email": "customer@example.com",
  "customer_name": "John Doe",
  "company_name": "Acme Corp",
  "tier": "professional",
  "duration_days": 365
}
```

**Response:**
```json
{
  "license": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "license_key": "PRIVE-V1-ABCDE-12345-FGHIJ-67890...",
    "customer_email": "customer@example.com",
    "tier": "professional",
    "max_agents": 100,
    "max_users": 10,
    "issued_at": "2026-01-14T10:00:00Z",
    "expires_at": "2027-01-14T10:00:00Z",
    "is_active": true
  }
}
```

### Validate License
```http
POST /api/v1/licenses/validate
Content-Type: application/json

{
  "license_key": "PRIVE-V1-ABCDE-12345...",
  "agent_id": "agent-001",
  "hostname": "workstation-42"
}
```

**Response:**
```json
{
  "valid": true,
  "license": { ... },
  "features": {
    "edr_monitoring": true,
    "dlp_protection": true,
    "threat_hunting": true,
    ...
  },
  "remaining_agents": 95,
  "expires_in_days": 350,
  "message": "License valid"
}
```

### Generate Trial License
```http
POST /api/v1/licenses/trial
Content-Type: application/json

{
  "email": "trial@example.com",
  "name": "Trial User"
}
```

**Response:**
```json
{
  "license": {
    "id": "TRIAL-...",
    "license_key": "PRIVE-V1-TRIAL-...",
    "tier": "professional",
    "max_agents": 10,
    "max_users": 3,
    "expires_at": "2026-01-28T10:00:00Z"
  },
  "message": "14-day trial license created"
}
```

### List Licenses (Admin)
```http
GET /api/v1/licenses?limit=50&offset=0
```

### Get License Details
```http
GET /api/v1/licenses/{license_id}
```

### Revoke License
```http
DELETE /api/v1/licenses/{license_id}
```

### Get License Usage
```http
GET /api/v1/licenses/{license_id}/usage
```

**Response:**
```json
{
  "license_id": "...",
  "active_agents": 45,
  "active_users": 8,
  "events_ingested": 12500000,
  "storage_used_gb": 125.5,
  "last_updated": "2026-01-14T10:00:00Z"
}
```

---

## 🔧 Implementation

### Backend (Go)

**License Service:**
```go
import "github.com/sentinel-enterprise/platform/license/service"

// Initialize license service
licenseService := service.NewLicenseService(db, privateKey, publicKey)

// Create license
license, err := licenseService.CreateLicense(req)

// Validate license
response, err := licenseService.ValidateLicense(licenseKey, agentID)
```

**Key Generation:**
```go
import "github.com/sentinel-enterprise/platform/license/crypto"

// Generate key pair (do this once, store securely)
keyPair, err := crypto.GenerateKeyPair()

// Generate license key
payload := crypto.LicensePayload{
    ID:        licenseID,
    Email:     "customer@example.com",
    Tier:      "professional",
    IssuedAt:  time.Now().Unix(),
    ExpiresAt: time.Now().AddDate(1, 0, 0).Unix(),
    MaxAgents: 100,
}

licenseKey, err := crypto.GenerateLicenseKey(payload, keyPair.PrivateKey)
```

**Validation:**
```go
payload, err := crypto.ValidateLicenseKey(licenseKey, keyPair.PublicKey)
if err != nil {
    // Invalid or expired license
}
```

### Agent Integration (Rust)

```rust
// Store license key in agent config
let license_key = "PRIVE-V1-ABCDE-12345...";

// Validate on startup
match validate_license(license_key, agent_id).await {
    Ok(response) if response.valid => {
        // Check features
        if response.features.dlp_protection {
            enable_dlp_engine();
        }
    }
    _ => {
        // License invalid, run in limited mode
        warn!("Invalid license, some features disabled");
    }
}
```

---

## 🎨 Dashboard Integration

The admin dashboard includes a **License Management** section:

### Features:
- **License Overview**
  - Total licenses issued
  - Active vs. expired licenses
  - Revenue tracking
  - Tier distribution

- **Create New License**
  - Customer information form
  - Tier selection
  - Duration configuration
  - Instant key generation

- **License List**
  - Searchable/filterable table
  - Status indicators
  - Expiration warnings
  - Quick actions (extend, revoke, view)

- **Usage Analytics**
  - Agents per license
  - Feature utilization
  - Storage consumption
  - API call metrics

### Dashboard Routes:
```
/admin/licenses          - License overview
/admin/licenses/create   - Create new license
/admin/licenses/:id      - License details
/admin/licenses/:id/edit - Edit license
```

---

## 🔒 Security Considerations

### Private Key Storage
**CRITICAL**: The Ed25519 private key must be stored securely:

```bash
# Generate key pair
go run scripts/generate-keys.go

# Store private key in secure location
export LICENSE_PRIVATE_KEY="base64_encoded_key"

# Or use secrets manager (AWS Secrets Manager, HashiCorp Vault)
```

### Key Rotation
1. Generate new key pair
2. Keep old public key for validation
3. Issue new licenses with new private key
4. Gradually phase out old keys

### Validation Best Practices
- Validate license on agent startup
- Re-validate every 24 hours
- Cache validation results
- Graceful degradation on validation failure

---

## 📊 Database Schema

```sql
CREATE TABLE licenses (
    id                UUID PRIMARY KEY,
    license_key       TEXT UNIQUE NOT NULL,
    customer_email    VARCHAR(255) NOT NULL,
    customer_name     VARCHAR(255) NOT NULL,
    company_name      VARCHAR(255),
    tier              VARCHAR(50) NOT NULL,
    max_agents        INTEGER NOT NULL,
    max_users         INTEGER NOT NULL,
    issued_at         TIMESTAMP NOT NULL,
    expires_at        TIMESTAMP,
    is_active         BOOLEAN DEFAULT TRUE,
    activated_at      TIMESTAMP,
    last_validated_at TIMESTAMP,
    metadata          JSONB,
    created_at        TIMESTAMP DEFAULT NOW(),
    updated_at        TIMESTAMP DEFAULT NOW()
);

CREATE TABLE license_usage (
    license_id       UUID REFERENCES licenses(id),
    active_agents    INTEGER DEFAULT 0,
    active_users     INTEGER DEFAULT 0,
    events_ingested  BIGINT DEFAULT 0,
    storage_used_gb  NUMERIC(10, 2) DEFAULT 0,
    last_updated     TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (license_id)
);

CREATE TABLE license_activations (
    id              UUID PRIMARY KEY,
    license_id      UUID REFERENCES licenses(id),
    agent_id        VARCHAR(255),
    hostname        VARCHAR(255),
    ip_address      INET,
    activated_at    TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_licenses_email ON licenses(customer_email);
CREATE INDEX idx_licenses_tier ON licenses(tier);
CREATE INDEX idx_licenses_active ON licenses(is_active);
CREATE INDEX idx_license_activations_license ON license_activations(license_id);
```

---

## 🚀 Deployment

### Environment Variables
```bash
# Platform API
export API_PORT=8080
export DATABASE_URL=postgresql://user:pass@localhost/prive
export LICENSE_PRIVATE_KEY=base64_encoded_private_key
export LICENSE_PUBLIC_KEY=base64_encoded_public_key

# Optional: License server (separate from main API)
export LICENSE_SERVER_URL=https://license.prive-security.com
```

### Docker Compose
```yaml
services:
  platform-api:
    build: ./platform
    environment:
      LICENSE_PRIVATE_KEY: ${LICENSE_PRIVATE_KEY}
      LICENSE_PUBLIC_KEY: ${LICENSE_PUBLIC_KEY}
    volumes:
      - ./keys:/app/keys:ro  # Mount keys securely
```

---

## 💰 Pricing Strategy

### Suggested Pricing:

**Free Tier**
- $0/month
- 5 agents
- Community support

**Professional**
- $99/agent/year
- 100 agents minimum ($9,900/year)
- Email support

**Enterprise**
- Custom pricing
- Unlimited agents
- 24/7 priority support
- Dedicated account manager
- SLA guarantees

### Trial Strategy:
- 14-day full-featured trial
- No credit card required
- Automatic email reminders (7 days, 1 day before expiry)
- Easy upgrade path

---

## 📧 Email Notifications

Implement automated emails for:
- License issued (with activation instructions)
- License expiring (30, 7, 1 day warnings)
- License expired
- Usage threshold warnings (80%, 90%, 100% of agent limit)
- Upgrade opportunities

---

## 🔄 License Lifecycle

```
┌─────────────┐
│   Created   │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Activated  │◄──────┐
└──────┬──────┘       │
       │              │
       ▼              │
┌─────────────┐       │
│   Active    │       │
└──────┬──────┘       │
       │              │
       ├──────────────┘ Extend/Renew
       │
       ▼
┌─────────────┐
│   Expiring  │ (30 days warning)
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Expired   │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   Revoked   │ (Manual action)
└─────────────┘
```

---

## 🧪 Testing

```bash
# Generate test license
curl -X POST http://localhost:8080/api/v1/licenses \
  -H "Content-Type: application/json" \
  -d '{
    "customer_email": "test@example.com",
    "customer_name": "Test User",
    "tier": "professional",
    "duration_days": 30
  }'

# Validate license
curl -X POST http://localhost:8080/api/v1/licenses/validate \
  -H "Content-Type: application/json" \
  -d '{
    "license_key": "PRIVE-V1-...",
    "agent_id": "test-agent-001"
  }'
```

---

## 📝 License Agreement Template

Include with each license:

```
PRIVÉ SOFTWARE LICENSE AGREEMENT

1. Grant of License
This license grants the customer the right to use Privé software
in accordance with the tier purchased (Free/Professional/Enterprise).

2. Restrictions
- May not reverse engineer or modify the software
- May not exceed licensed agent/user limits
- May not transfer license without written consent

3. Support & Updates
- Professional: Email support, software updates included
- Enterprise: 24/7 priority support, dedicated account manager

4. Termination
License may be terminated for:
- Non-payment
- Violation of terms
- Security concerns

5. Warranty Disclaimer
Software provided "AS IS" without warranty...
```

---

## 🎯 Next Steps

1. **Implement database schema** in PostgreSQL
2. **Create admin dashboard** license management UI
3. **Set up email automation** for license lifecycle events
4. **Generate master key pair** and store securely
5. **Create billing integration** (Stripe, PayPal)
6. **Build self-service portal** for customers
7. **Implement usage tracking** and alerts

---

## 📞 Support

For licensing questions:
- Email: licensing@prive-security.com
- Documentation: https://docs.prive-security.com/licensing
- Support Portal: https://support.prive-security.com
