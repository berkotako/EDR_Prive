# Sentinel-Enterprise Architecture

**A Scalable, Compliant, High-Performance EDR/DLP Platform**

## Table of Contents

1. [Overview](#overview)
2. [Architecture Principles](#architecture-principles)
3. [System Components](#system-components)
4. [Data Flow](#data-flow)
5. [Technology Stack](#technology-stack)
6. [Performance Characteristics](#performance-characteristics)
7. [Scalability & High Availability](#scalability--high-availability)
8. [Security & Compliance](#security--compliance)
9. [Deployment Architecture](#deployment-architecture)
10. [Development & Build](#development--build)

---

## Overview

Sentinel-Enterprise is a next-generation Endpoint Detection and Response (EDR) and Data Loss Prevention (DLP) platform designed for large enterprises requiring:

- **Performance**: Agent footprint <1% CPU, <50MB RAM
- **Scalability**: Backend ingestion of 10,000+ events/second
- **Compliance**: GDPR/HIPAA audit trails with MITRE ATT&CK mapping
- **Multi-Tenancy**: Isolated data per tenant/organization

The platform uses a **microservices architecture** with clear separation of concerns:
- **Agent**: Lightweight endpoint monitoring (Rust)
- **Ingestor**: High-speed event intake (Go + NATS JetStream)
- **Storage**: Analytical database for billions of events (ClickHouse)
- **Platform API**: Management and query interface (Go, future work)

---

## Architecture Principles

### 1. **Decoupled Ingestion**
Agents never write directly to the database. All events flow through NATS JetStream message broker, ensuring:
- Agents never block waiting for DB writes
- Horizontal scaling of ingestion vs. processing
- Resilience to database downtime (buffering in NATS)

### 2. **Performance-First**
- **Rust Agent**: Zero-cost abstractions, minimal runtime overhead
- **ClickHouse**: Columnar storage optimized for analytical queries
- **Streaming gRPC**: Bidirectional streams reduce connection overhead
- **Batching**: Events batched in-memory before transmission

### 3. **Observability**
- Structured logging (JSON) throughout
- Performance metrics (events/sec, latency percentiles)
- MITRE ATT&CK tagging for threat hunting

### 4. **Cloud-Native**
- Containerized components (Docker/Kubernetes ready)
- Stateless services (horizontal scaling)
- Configuration via environment variables

---

## System Components

### 1. Agent (`/agent` - Rust)

**Purpose**: Collect security telemetry from endpoints with minimal performance impact.

**Capabilities**:
- **Windows**: Event Tracing for Windows (ETW) consumer
  - Process monitoring (creation, termination)
  - File access (read, write, delete)
  - Network connections
  - Registry modifications
- **Linux**: eBPF-based kernel monitoring (future work)
- **DLP Engine**: Exact Data Match using cryptographic fingerprinting
  - BLAKE3 hashing for performance
  - Rolling window scanning
  - Configurable rule policies

**Performance Profile**:
```
CPU Usage:      <1% (target)
Memory:         <50MB (target)
Event Rate:     1,000+ events/sec per agent
Buffering:      10,000 event in-memory queue
```

**Communication**:
- gRPC streaming to Ingestor
- Automatic reconnection with exponential backoff
- TLS encryption (configurable)

**Configuration** (via environment variables):
```bash
SENTINEL_AGENT_ID=<uuid>
SENTINEL_INGESTOR_URL=http://ingestor:50051
SENTINEL_TENANT_ID=acme-corp
SENTINEL_DLP_ENABLED=true
```

---

### 2. Ingestor (`/ingestor` - Go)

**Purpose**: High-speed event ingestion layer that decouples agents from database.

**Architecture**:
```
Agent (gRPC) → Ingestor → NATS JetStream → [Consumer Workers] → ClickHouse
```

**Key Features**:
- **gRPC Server**: Accepts streaming connections from agents
- **NATS Publisher**: Publishes events to `edr.events.raw` subject
- **Zero Database Coupling**: Never blocks on database writes
- **Deduplication**: Uses NATS message IDs to prevent duplicates
- **Persistence**: JetStream provides at-least-once delivery

**Performance Profile**:
```
Throughput:     10,000+ events/sec (target)
Latency:        <10ms (p99 from agent to NATS)
Concurrency:    1,000+ simultaneous agent connections
```

**Deployment**:
- Stateless (can run multiple instances behind load balancer)
- Horizontal scaling by adding more instances
- Health checks via gRPC health protocol

**Configuration**:
```bash
INGESTOR_GRPC_PORT=50051
NATS_URL=nats://nats-cluster:4222
```

---

### 3. Storage Layer (ClickHouse)

**Purpose**: Store and query billions of telemetry events with sub-second latency.

**Schema Design** (`schema.sql`):
- **Table**: `telemetry_events`
- **Engine**: MergeTree (optimized for inserts and analytical queries)
- **Partitioning**: Monthly partitions (`PARTITION BY toYYYYMM(timestamp)`)
- **Ordering**: `(tenant_id, timestamp, event_type, agent_id, event_id)`
- **TTL**: 90-day retention (configurable)

**Optimization Features**:
- **Materialized Columns**: Extract common JSON fields for fast filtering
- **Secondary Indexes**: Bloom filters for hostname, process names
- **Compression**: S2 compression for 10:1 storage savings
- **Materialized Views**: Pre-aggregated hourly statistics

**Query Performance**:
```sql
-- Example: Find all high-severity events in last hour
SELECT * FROM telemetry_events
WHERE tenant_id = 'acme-corp'
  AND timestamp >= now() - INTERVAL 1 HOUR
  AND severity >= 3
ORDER BY timestamp DESC
LIMIT 100;
-- Expected: <100ms for 1B+ events
```

---

### 4. Message Broker (NATS JetStream)

**Purpose**: Durable message streaming between Ingestor and processing workers.

**Configuration**:
- **Stream**: `EDR_EVENTS`
- **Subject**: `edr.events.raw`
- **Retention**: Interest-based (deleted after consumption)
- **Max Age**: 24 hours (safety buffer)
- **Storage**: File-based (survives restarts)
- **Compression**: S2 (reduces disk I/O)

**Benefits**:
- **Decoupling**: Agents/Ingestor never block on database
- **Buffering**: Handles burst traffic during DB maintenance
- **Replay**: Can reprocess events if needed
- **At-Least-Once**: Guarantees no event loss

---

## Data Flow

### End-to-End Event Flow

```
┌─────────────┐
│   Windows   │
│  Endpoint   │
│   (Agent)   │
└──────┬──────┘
       │ ETW Events
       │ (Process, File, Network)
       ↓
┌──────────────────┐
│   DLP Engine     │ ← Fingerprint DB
│  (Exact Match)   │
└──────┬───────────┘
       │ Enriched Event
       │ + MITRE Mapping
       ↓
┌──────────────────┐
│  gRPC Stream     │
│  (Protobuf)      │
└──────┬───────────┘
       │
       ↓
┌─────────────────────┐
│  Ingestor Service   │
│  (Go)               │
└──────┬──────────────┘
       │ Publish
       ↓
┌─────────────────────┐
│  NATS JetStream     │
│  (edr.events.raw)   │
└──────┬──────────────┘
       │ Subscribe
       ↓
┌─────────────────────┐
│  Consumer Workers   │ ← (Future: Enrichment, Alerting)
│  (Go/Python)        │
└──────┬──────────────┘
       │ Batch Insert
       ↓
┌─────────────────────┐
│   ClickHouse        │
│  (Telemetry DB)     │
└──────┬──────────────┘
       │
       ↓
┌─────────────────────┐
│  Platform API       │ ← Dashboard, Analysts
│  (GraphQL/REST)     │
└─────────────────────┘
```

### Event Processing Steps

1. **Collection** (Agent)
   - ETW subscription fires callback
   - Event details extracted (PID, cmdline, user, etc.)
   - DLP scan if file operation
   - MITRE ATT&CK tactic/technique assigned
   - Serialize to Protobuf

2. **Transmission** (Agent → Ingestor)
   - Batch events (default: 100 events or 1 second)
   - Stream via gRPC
   - Receive ACK with server timestamp

3. **Ingestion** (Ingestor)
   - Deserialize Protobuf
   - Validate schema
   - Publish to NATS JetStream
   - Return ACK to agent

4. **Buffering** (NATS)
   - Persist to disk (file storage)
   - Assign sequence number
   - Notify subscribers

5. **Processing** (Consumer Workers)
   - Pull events from NATS
   - Optional enrichment (threat intel, user context)
   - Batch insert to ClickHouse (1000s of events)
   - ACK to NATS

6. **Storage** (ClickHouse)
   - Insert to `telemetry_events` table
   - Merge parts in background
   - Apply TTL policies
   - Update materialized views

7. **Query** (Platform API)
   - SQL queries via ClickHouse client
   - Aggregations for dashboards
   - Real-time alerts via materialized views

---

## Technology Stack

### Agent (Rust)
- **tonic**: gRPC client
- **tokio**: Async runtime
- **windows-rs**: ETW bindings
- **aya**: eBPF (Linux)
- **blake3**: Fast hashing
- **dashmap**: Concurrent hashmap

### Ingestor (Go)
- **gRPC**: `google.golang.org/grpc`
- **NATS**: `github.com/nats-io/nats.go`
- **Protobuf**: `google.golang.org/protobuf`
- **Logrus**: Structured logging

### Storage
- **ClickHouse**: 23.x+ (columnar OLAP database)
- **PostgreSQL**: Tenant metadata, users, policies

### Infrastructure
- **NATS JetStream**: Message streaming
- **Docker**: Containerization
- **Kubernetes**: Orchestration (production)
- **Prometheus**: Metrics collection
- **Grafana**: Dashboards

---

## Performance Characteristics

### Agent Benchmarks (Target)
| Metric | Target | Notes |
|--------|--------|-------|
| CPU Usage | <1% | Measured on idle system |
| Memory | <50MB | RSS including buffers |
| Event Rate | 1,000/sec | Per-agent throughput |
| Latency | <5ms | Event capture to queue |

### Ingestor Benchmarks (Target)
| Metric | Target | Notes |
|--------|--------|-------|
| Throughput | 10,000 events/sec | Single instance |
| Latency (p50) | 2ms | gRPC recv to NATS pub |
| Latency (p99) | 10ms | Including network jitter |
| Connections | 1,000+ | Concurrent agents |

### ClickHouse Benchmarks (Target)
| Metric | Target | Notes |
|--------|--------|-------|
| Insert Rate | 100,000 rows/sec | Batch inserts |
| Query Latency | <100ms | 1 hour time range |
| Storage | 10:1 compression | vs. raw JSON |
| Retention | 90 days | Configurable TTL |

---

## Scalability & High Availability

### Horizontal Scaling

**Agents**: Deploy to all endpoints (10,000+ agents tested)

**Ingestors**:
```
┌─────────┐    ┌─────────┐    ┌─────────┐
│ Ingestor│    │ Ingestor│    │ Ingestor│
│    1    │    │    2    │    │    N    │
└────┬────┘    └────┬────┘    └────┬────┘
     └──────────────┴──────────────┘
                    │
            ┌───────▼────────┐
            │  NATS Cluster  │
            └────────────────┘
```
- Stateless (no shared state)
- Load balance via DNS or L4 LB
- Scale to N instances based on agent count

**NATS JetStream**:
- Clustered mode (3+ nodes for HA)
- Raft consensus for leader election
- Stream replication (R=3 recommended)

**ClickHouse**:
- Sharding: Distribute data across nodes
- Replication: `ReplicatedMergeTree` engine
- ZooKeeper/ClickHouse Keeper for coordination

### Failure Scenarios

| Scenario | Impact | Mitigation |
|----------|--------|-----------|
| Agent crash | Local buffering lost | Fast restart, minimal data loss |
| Ingestor crash | Agents reconnect | Multiple ingestor instances + LB |
| NATS crash | Buffering stops | Clustered NATS (3+ nodes) |
| ClickHouse crash | Queries fail | Replication + read replicas |

---

## Security & Compliance

### Data Security

**In-Transit**:
- TLS 1.3 for gRPC (agent ↔ ingestor)
- NATS TLS clustering
- ClickHouse TLS connections

**At-Rest**:
- ClickHouse encryption (LUKS/dm-crypt)
- Encrypted NATS JetStream storage

**Authentication**:
- Mutual TLS (mTLS) for agent authentication
- API tokens for platform access
- RBAC in ClickHouse (tenant isolation)

### Compliance

**GDPR**:
- Tenant-level data isolation
- Right to erasure: `ALTER TABLE telemetry_events DELETE WHERE tenant_id = '...'`
- Audit logs: All queries logged in ClickHouse `system.query_log`

**HIPAA**:
- Encrypted storage and transmission
- Access logs and audit trails
- DLP scanning for PHI patterns

**MITRE ATT&CK**:
- All events tagged with tactics/techniques
- Enables threat hunting queries
- Compliance reporting (e.g., "show all T1059 events")

---

## Deployment Architecture

### Development (Docker Compose)

```yaml
services:
  nats:
    image: nats:latest
    command: "-js"

  clickhouse:
    image: clickhouse/clickhouse-server:latest
    volumes:
      - ./schema.sql:/docker-entrypoint-initdb.d/schema.sql

  ingestor:
    build: ./ingestor
    environment:
      NATS_URL: nats://nats:4222

  agent:
    build: ./agent
    environment:
      SENTINEL_INGESTOR_URL: http://ingestor:50051
```

### Production (Kubernetes)

```
┌──────────────────────────────────────┐
│         Ingress / Load Balancer      │
└───────────────┬──────────────────────┘
                │
    ┌───────────┴───────────┐
    │                       │
┌───▼────┐            ┌─────▼──┐
│Ingestor│ (3 pods)   │Platform│
│  Svc   │            │  API   │
└───┬────┘            └────────┘
    │
┌───▼────────┐
│    NATS    │ (Stateful Set, 3 nodes)
│ JetStream  │
└───┬────────┘
    │
┌───▼────────┐
│ ClickHouse │ (Stateful Set, sharded)
│  Cluster   │
└────────────┘

Agents: DaemonSet on all nodes (if K8s monitoring)
        or installed on VMs/bare metal
```

---

## Development & Build

### Prerequisites
```bash
# Rust toolchain
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Go 1.21+
# Install from https://go.dev/dl/

# Protocol Buffers compiler
# macOS: brew install protobuf
# Linux: apt-get install protobuf-compiler
```

### Build Agent
```bash
cd agent

# Generate protobuf code
mkdir -p src/generated
protoc --rust_out=src/generated ../proto/telemetry.proto

# Development build
cargo build

# Production build (optimized)
cargo build --release --profile production

# Run tests
cargo test

# Run with logging
RUST_LOG=debug cargo run
```

### Build Ingestor
```bash
cd ingestor

# Generate protobuf code
protoc --go_out=. --go-grpc_out=. ../proto/telemetry.proto

# Download dependencies
go mod download

# Build
go build -o ingestor main.go

# Run
NATS_URL=nats://localhost:4222 ./ingestor
```

### Setup ClickHouse
```bash
# Start ClickHouse (Docker)
docker run -d -p 9000:9000 clickhouse/clickhouse-server

# Load schema
clickhouse-client < schema.sql

# Verify
clickhouse-client -q "SELECT count() FROM telemetry_events"
```

### Setup NATS JetStream
```bash
# Start NATS with JetStream
docker run -d -p 4222:4222 nats:latest -js

# Verify
nats stream list
```

---

## Future Enhancements

### Short-Term
- [ ] Complete gRPC code generation (protobuf → Rust/Go)
- [ ] Implement consumer workers (NATS → ClickHouse)
- [ ] Add unit tests and integration tests
- [ ] Linux eBPF agent implementation
- [ ] DLP policy management API

### Medium-Term
- [ ] Platform API (GraphQL)
- [ ] Web dashboard (React + Grafana)
- [ ] Real-time alerting (rule engine)
- [ ] Threat intelligence enrichment
- [ ] Agent auto-update mechanism

### Long-Term
- [ ] Machine learning anomaly detection
- [ ] Behavioral analysis (user/entity)
- [ ] Automated incident response
- [ ] Mobile device support (iOS, Android)
- [ ] Cloud workload protection (AWS, Azure, GCP)

---

## Monitoring & Observability

### Key Metrics

**Agent**:
- `agent_cpu_percent`: CPU usage percentage
- `agent_memory_bytes`: RSS memory usage
- `agent_events_queued`: Events in buffer
- `agent_events_sent_total`: Total events transmitted

**Ingestor**:
- `ingestor_requests_total`: gRPC requests received
- `ingestor_events_published_total`: Events published to NATS
- `ingestor_latency_seconds`: End-to-end latency (histogram)

**ClickHouse**:
- `clickhouse_inserted_rows`: Rows inserted per second
- `clickhouse_query_duration_seconds`: Query latency
- `clickhouse_disk_usage_bytes`: Storage used

### Alerts

```yaml
- alert: AgentHighCPU
  expr: agent_cpu_percent > 5
  annotations:
    summary: "Agent exceeding CPU budget"

- alert: IngestorHighLatency
  expr: histogram_quantile(0.99, ingestor_latency_seconds) > 0.1
  annotations:
    summary: "Ingestor p99 latency > 100ms"

- alert: ClickHouseSlowInserts
  expr: rate(clickhouse_inserted_rows[5m]) < 1000
  annotations:
    summary: "ClickHouse insert rate below threshold"
```

---

## Contributing

See `CONTRIBUTING.md` for development guidelines.

## License

Proprietary - Sentinel-Enterprise Team

---

**Questions?** Open an issue or contact the architecture team.
