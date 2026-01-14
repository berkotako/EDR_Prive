// Sentinel-Enterprise Ingestion Service
// High-performance gRPC server that accepts agent telemetry and publishes to NATS JetStream.
// Performance Target: Handle 10,000+ events/sec with low latency.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// TODO: Import generated protobuf package
	// pb "github.com/sentinel-enterprise/proto/telemetry"
)

const (
	// gRPC server configuration
	defaultGRPCPort = "50051"
	maxMessageSize  = 4 * 1024 * 1024 // 4MB max message size

	// NATS JetStream configuration
	natsSubject   = "edr.events.raw"
	natsStream    = "EDR_EVENTS"
	natsMaxAge    = 24 * time.Hour // Retain events for 24h in stream
	natsMaxBytes  = 10 * 1024 * 1024 * 1024 // 10GB max stream size

	// Performance monitoring
	statsInterval = 30 * time.Second
)

// IngestorService implements the TelemetryService gRPC interface
type IngestorService struct {
	// pb.UnimplementedTelemetryServiceServer
	natsConn      *nats.Conn
	jetStream     nats.JetStreamContext
	eventsHandled atomic.Uint64
	bytesIngested atomic.Uint64
	mu            sync.RWMutex
}

// NewIngestorService creates a new ingestion service with NATS connection
func NewIngestorService(natsURL string) (*IngestorService, error) {
	log.Infof("Connecting to NATS server: %s", natsURL)

	// Connect to NATS with reconnect options
	nc, err := nats.Connect(natsURL,
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Warnf("NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Info("NATS reconnected successfully")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Create or update JetStream stream for event persistence
	streamConfig := &nats.StreamConfig{
		Name:        natsStream,
		Subjects:    []string{natsSubject, "edr.events.>"},
		Retention:   nats.InterestPolicy,
		MaxAge:      natsMaxAge,
		MaxBytes:    natsMaxBytes,
		Storage:     nats.FileStorage,
		Replicas:    1, // TODO: Increase for HA deployments
		Compression: nats.S2Compression, // Enable compression for storage efficiency
	}

	_, err = js.AddStream(streamConfig)
	if err != nil {
		// Stream might already exist, try to update it
		_, err = js.UpdateStream(streamConfig)
		if err != nil {
			nc.Close()
			return nil, fmt.Errorf("failed to configure JetStream: %w", err)
		}
	}

	log.Infof("JetStream stream '%s' configured successfully", natsStream)

	return &IngestorService{
		natsConn:  nc,
		jetStream: js,
	}, nil
}

// StreamEvents handles bidirectional streaming of telemetry events
// This is the high-performance path: agents stream events, we ACK in batches
func (s *IngestorService) StreamEvents(stream interface{}) error {
	// TODO: Replace with actual protobuf stream type
	// stream pb.TelemetryService_StreamEventsServer

	ctx := context.Background() // Replace with stream.Context()
	clientID := uuid.New().String()
	log.Infof("New stream connection established: client_id=%s", clientID)

	eventsReceived := 0
	startTime := time.Now()

	// Mock event receiving loop (replace with actual protobuf deserialization)
	for {
		// In the real implementation:
		// event, err := stream.Recv()
		// if err == io.EOF {
		//     break
		// }
		// if err != nil {
		//     log.Errorf("Stream error for client %s: %v", clientID, err)
		//     return status.Errorf(codes.Internal, "stream read error: %v", err)
		// }

		// For now, simulate event processing
		select {
		case <-ctx.Done():
			log.Infof("Stream context cancelled for client %s", clientID)
			return ctx.Err()
		default:
			// TODO: Process actual event
			// s.publishEvent(event)
			// eventsReceived++

			// Mock: break after simulation
			time.Sleep(100 * time.Millisecond)
			break
		}

		break // Remove this in real implementation
	}

	duration := time.Since(startTime)
	log.Infof("Stream closed: client_id=%s, events=%d, duration=%.2fs",
		clientID, eventsReceived, duration.Seconds())

	return nil
}

// SubmitEvent handles unary event submission (low-volume fallback)
func (s *IngestorService) SubmitEvent(ctx context.Context, event interface{}) (interface{}, error) {
	// TODO: Replace with actual protobuf types
	// event *pb.Event, *pb.EventAck, error

	log.Debugf("Received unary event: agent_id=%s, type=%s",
		"unknown", "unknown") // Replace with event.AgentId, event.EventType

	// Publish to NATS
	if err := s.publishEvent(event); err != nil {
		log.Errorf("Failed to publish event: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to publish event: %v", err)
	}

	// Return acknowledgment
	ack := struct {
		Success         bool
		EventID         string
		ServerTimestamp int64
	}{
		Success:         true,
		EventID:         uuid.New().String(),
		ServerTimestamp: time.Now().UnixMilli(),
	}

	return ack, nil
}

// publishEvent publishes an event to NATS JetStream for async processing
// This decouples ingestion from database writes for maximum throughput
func (s *IngestorService) publishEvent(event interface{}) error {
	// Serialize event to JSON (protobuf -> JSON for flexibility in downstream consumers)
	// In production, you might keep it as protobuf for efficiency
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publish to JetStream with deduplication and persistence
	pubAck, err := s.jetStream.Publish(natsSubject, eventJSON,
		nats.MsgId(uuid.New().String()), // Deduplication
	)
	if err != nil {
		return fmt.Errorf("failed to publish to NATS: %w", err)
	}

	log.Debugf("Event published: stream=%s, seq=%d", pubAck.Stream, pubAck.Sequence)

	// Update metrics
	s.eventsHandled.Add(1)
	s.bytesIngested.Add(uint64(len(eventJSON)))

	return nil
}

// Close gracefully shuts down the service
func (s *IngestorService) Close() error {
	log.Info("Closing NATS connection...")
	s.natsConn.Close()
	return nil
}

// printStats periodically logs performance statistics
func (s *IngestorService) printStats(ctx context.Context) {
	ticker := time.NewTicker(statsInterval)
	defer ticker.Stop()

	var lastEvents, lastBytes uint64
	lastTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			events := s.eventsHandled.Load()
			bytes := s.bytesIngested.Load()
			now := time.Now()
			elapsed := now.Sub(lastTime).Seconds()

			eventsPerSec := float64(events-lastEvents) / elapsed
			mbPerSec := float64(bytes-lastBytes) / elapsed / (1024 * 1024)

			log.Infof("Performance: %.0f events/sec, %.2f MB/sec (total: %d events, %d MB)",
				eventsPerSec, mbPerSec, events, bytes/(1024*1024))

			lastEvents = events
			lastBytes = bytes
			lastTime = now
		}
	}
}

func main() {
	// Configure logging
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)
	log.Info("Sentinel-Enterprise Ingestor starting...")

	// Load configuration from environment
	grpcPort := getEnv("INGESTOR_GRPC_PORT", defaultGRPCPort)
	natsURL := getEnv("NATS_URL", nats.DefaultURL)

	// Create ingestor service
	service, err := NewIngestorService(natsURL)
	if err != nil {
		log.Fatalf("Failed to create ingestor service: %v", err)
	}
	defer service.Close()

	// Start performance monitoring
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go service.printStats(ctx)

	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", grpcPort, err)
	}

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(maxMessageSize),
		grpc.MaxSendMsgSize(maxMessageSize),
	)

	// TODO: Register service with protobuf
	// pb.RegisterTelemetryServiceServer(grpcServer, service)

	log.Infof("Ingestor gRPC server listening on :%s", grpcPort)

	// Graceful shutdown handling
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Info("Shutdown signal received, stopping server...")
		cancel()
		grpcServer.GracefulStop()
	}()

	// Start serving
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}

	log.Info("Ingestor service stopped")
}

// getEnv retrieves an environment variable with a fallback default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
