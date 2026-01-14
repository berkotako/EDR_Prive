#!/bin/bash
# Generate Protocol Buffer code for all components

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PROTO_DIR="$PROJECT_ROOT/proto"

echo "🔧 Generating Protocol Buffer code..."
echo

# Check for protoc
if ! command -v protoc &> /dev/null; then
    echo "❌ Error: protoc not found. Please install Protocol Buffers compiler."
    echo "   macOS: brew install protobuf"
    echo "   Ubuntu: apt-get install protobuf-compiler"
    exit 1
fi

# Generate Go code for ingestor
echo "📦 Generating Go code for ingestor..."
cd "$PROJECT_ROOT/ingestor"

if ! command -v protoc-gen-go &> /dev/null; then
    echo "   Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "   Installing protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

mkdir -p internal/pb

protoc \
    --go_out=internal/pb \
    --go_opt=paths=source_relative \
    --go-grpc_out=internal/pb \
    --go-grpc_opt=paths=source_relative \
    -I="$PROTO_DIR" \
    "$PROTO_DIR/telemetry.proto"

echo "   ✓ Generated: ingestor/internal/pb/telemetry.pb.go"
echo "   ✓ Generated: ingestor/internal/pb/telemetry_grpc.pb.go"
echo

# Generate Go code for consumer
echo "📦 Generating Go code for consumer..."
cd "$PROJECT_ROOT/consumer"

mkdir -p internal/pb

protoc \
    --go_out=internal/pb \
    --go_opt=paths=source_relative \
    --go-grpc_out=internal/pb \
    --go-grpc_opt=paths=source_relative \
    -I="$PROTO_DIR" \
    "$PROTO_DIR/telemetry.proto"

echo "   ✓ Generated: consumer/internal/pb/telemetry.pb.go"
echo "   ✓ Generated: consumer/internal/pb/telemetry_grpc.pb.go"
echo

# Generate Rust code for agent
echo "🦀 Generating Rust code for agent..."
cd "$PROJECT_ROOT/agent"

echo "   Running cargo build to trigger build.rs..."
if cargo build --quiet 2>&1 | grep -q "error"; then
    echo "   ⚠️  Build had errors (expected if dependencies not fully resolved)"
else
    echo "   ✓ Generated: agent/src/generated/telemetry.rs"
fi
echo

echo "✅ Protocol Buffer code generation complete!"
echo
echo "Next steps:"
echo "  1. cd ingestor && go mod tidy"
echo "  2. cd consumer && go mod tidy"
echo "  3. cd agent && cargo build"
