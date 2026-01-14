.PHONY: help proto build test docker-up docker-down clean agent ingestor consumer

# Default target
help:
	@echo "Privé - EDR/DLP Platform"
	@echo ""
	@echo "Available targets:"
	@echo "  proto        - Generate protobuf code for all components"
	@echo "  build        - Build all components"
	@echo "  test         - Run all tests"
	@echo "  docker-up    - Start infrastructure with Docker Compose"
	@echo "  docker-down  - Stop infrastructure"
	@echo "  agent        - Build Rust agent"
	@echo "  ingestor     - Build Go ingestor"
	@echo "  consumer     - Build Go consumer"
	@echo "  clean        - Clean build artifacts"

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@chmod +x scripts/generate-proto.sh
	@./scripts/generate-proto.sh

# Build all components
build: proto agent ingestor consumer
	@echo "All components built successfully"

# Build Rust agent
agent:
	@echo "Building Rust agent..."
	@cd agent && cargo build --release --profile production

# Build Go ingestor
ingestor:
	@echo "Building Go ingestor..."
	@cd ingestor && go mod tidy && go build -o bin/ingestor main.go

# Build Go consumer
consumer:
	@echo "Building Go consumer..."
	@cd consumer && go mod tidy && go build -o bin/consumer main.go

# Run tests
test:
	@echo "Running Rust tests..."
	@cd agent && cargo test
	@echo "Running Go tests (ingestor)..."
	@cd ingestor && go test ./...
	@echo "Running Go tests (consumer)..."
	@cd consumer && go test ./...

# Start infrastructure
docker-up:
	@echo "Starting Privé infrastructure..."
	@docker-compose up -d
	@echo "Waiting for services to be healthy..."
	@sleep 5
	@docker-compose ps

# Stop infrastructure
docker-down:
	@echo "Stopping Privé infrastructure..."
	@docker-compose down

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@cd agent && cargo clean
	@rm -rf ingestor/bin ingestor/internal/pb
	@rm -rf consumer/bin consumer/internal/pb
	@rm -rf agent/src/generated/*.rs
	@echo "Clean complete"

# Development environment setup
dev-setup:
	@echo "Setting up development environment..."
	@echo "Installing Rust dependencies..."
	@cd agent && cargo fetch
	@echo "Installing Go dependencies..."
	@cd ingestor && go mod download
	@cd consumer && go mod download
	@echo "Development environment ready"

# Run agent locally (for testing)
run-agent:
	@cd agent && RUST_LOG=debug SENTINEL_INGESTOR_URL=http://localhost:50051 cargo run

# Run ingestor locally
run-ingestor:
	@cd ingestor && NATS_URL=nats://localhost:4222 go run main.go

# Run consumer locally
run-consumer:
	@cd consumer && NATS_URL=nats://localhost:4222 CLICKHOUSE_ADDR=localhost:9000 go run main.go
