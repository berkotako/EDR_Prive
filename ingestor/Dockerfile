# Multi-stage build for Privé Ingestor

# Stage 1: Build
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git protobuf-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /build/ingestor \
    main.go

# Stage 2: Runtime
FROM alpine:3.19

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 prive && \
    adduser -D -u 1000 -G prive prive

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/ingestor /app/ingestor

# Change ownership
RUN chown -R prive:prive /app

# Switch to non-root user
USER prive

# Expose gRPC port
EXPOSE 50051

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD /app/ingestor --health || exit 1

# Run the ingestor
ENTRYPOINT ["/app/ingestor"]
