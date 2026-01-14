// Privé Consumer Worker
// Processes events from NATS JetStream and persists to ClickHouse
// Performance Target: 100,000+ inserts/sec with batching

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
)

const (
	// NATS configuration
	natsSubject      = "edr.events.raw"
	natsConsumerName = "clickhouse-writer"
	natsDurable      = "clickhouse-writer-durable"

	// ClickHouse batching
	batchSize     = 1000  // Events per batch
	batchTimeout  = 5     // Seconds before forcing flush
	maxRetries    = 3     // Retry attempts for failed batches
	workerCount   = 4     // Parallel workers for processing

	// Monitoring
	statsInterval = 30 * time.Second
)

// Event represents the deserialized telemetry event from NATS
type Event struct {
	AgentID         string `json:"agent_id"`
	Timestamp       int64  `json:"timestamp"`
	EventType       string `json:"event_type"`
	MitreTactic     string `json:"mitre_tactic"`
	MitreTechnique  string `json:"mitre_technique"`
	Severity        int32  `json:"severity"`
	Payload         string `json:"payload"`
	TenantID        string `json:"tenant_id"`
	Hostname        string `json:"hostname"`
	OSType          string `json:"os_type"`
}

// Consumer processes events from NATS and writes to ClickHouse
type Consumer struct {
	natsConn         *nats.Conn
	jetStream        nats.JetStreamContext
	clickhouse       driver.Conn
	eventsProcessed  atomic.Uint64
	eventsInserted   atomic.Uint64
	batchesFlushed   atomic.Uint64
	errors           atomic.Uint64
	mu               sync.Mutex
}

// NewConsumer creates a new consumer with NATS and ClickHouse connections
func NewConsumer(natsURL, clickhouseAddr string) (*Consumer, error) {
	log.Infof("Connecting to NATS: %s", natsURL)

	// Connect to NATS
	nc, err := nats.Connect(natsURL,
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Warnf("NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Info("NATS reconnected")
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

	log.Infof("Connecting to ClickHouse: %s", clickhouseAddr)

	// Connect to ClickHouse
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{clickhouseAddr},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout:      time.Second * 10,
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Hour,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	// Test connection
	if err := conn.Ping(context.Background()); err != nil {
		nc.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	log.Info("Connected to ClickHouse successfully")

	return &Consumer{
		natsConn:   nc,
		jetStream:  js,
		clickhouse: conn,
	}, nil
}

// Start begins consuming events from NATS
func (c *Consumer) Start(ctx context.Context) error {
	log.Infof("Starting %d consumer workers...", workerCount)

	// Create JetStream consumer if it doesn't exist
	_, err := c.jetStream.AddConsumer(natsSubject, &nats.ConsumerConfig{
		Durable:       natsDurable,
		FilterSubject: natsSubject,
		DeliverPolicy: nats.DeliverAllPolicy,
		AckPolicy:     nats.AckExplicitPolicy,
		MaxAckPending: batchSize * workerCount * 2,
		AckWait:       time.Minute,
	})
	if err != nil && err != nats.ErrStreamNotFound {
		log.Warnf("Consumer might already exist: %v", err)
	}

	// Start multiple workers for parallel processing
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			c.worker(ctx, workerID)
		}(i)
	}

	// Start statistics reporter
	go c.printStats(ctx)

	// Wait for all workers to finish
	wg.Wait()
	log.Info("All consumer workers stopped")

	return nil
}

// worker processes events in batches
func (c *Consumer) worker(ctx context.Context, workerID int) {
	log.Infof("Worker %d started", workerID)

	// Subscribe to JetStream with pull-based consumer
	sub, err := c.jetStream.PullSubscribe(natsSubject, natsDurable, nats.Bind(natsSubject, natsDurable))
	if err != nil {
		log.Errorf("Worker %d: Failed to subscribe: %v", workerID, err)
		return
	}
	defer sub.Unsubscribe()

	batch := make([]Event, 0, batchSize)
	batchMsgs := make([]*nats.Msg, 0, batchSize)
	batchTimer := time.NewTimer(batchTimeout * time.Second)
	defer batchTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			// Flush remaining events before shutdown
			if len(batch) > 0 {
				if c.flushBatchWithAck(workerID, batch, batchMsgs) {
					batch = batch[:0]
					batchMsgs = batchMsgs[:0]
				}
			}
			log.Infof("Worker %d shutting down", workerID)
			return

		case <-batchTimer.C:
			// Flush on timeout
			if len(batch) > 0 {
				if c.flushBatchWithAck(workerID, batch, batchMsgs) {
					batch = batch[:0]
					batchMsgs = batchMsgs[:0]
				}
			}
			batchTimer.Reset(batchTimeout * time.Second)

		default:
			// Pull messages from NATS
			msgs, err := sub.Fetch(batchSize-len(batch), nats.MaxWait(time.Second))
			if err != nil {
				if err == nats.ErrTimeout {
					continue
				}
				log.Errorf("Worker %d: Fetch error: %v", workerID, err)
				time.Sleep(time.Second)
				continue
			}

			// Process messages
			for _, msg := range msgs {
				var event Event
				if err := json.Unmarshal(msg.Data, &event); err != nil {
					log.Errorf("Worker %d: Failed to unmarshal event: %v", workerID, err)
					msg.Nak()
					c.errors.Add(1)
					continue
				}

				batch = append(batch, event)
				batchMsgs = append(batchMsgs, msg)
				c.eventsProcessed.Add(1)

				// Flush when batch is full
				if len(batch) >= batchSize {
					if c.flushBatchWithAck(workerID, batch, batchMsgs) {
						batch = batch[:0]
						batchMsgs = batchMsgs[:0]
					}
					batchTimer.Reset(batchTimeout * time.Second)
					break
				}
			}
		}
	}
}

// flushBatchWithAck writes a batch of events to ClickHouse and acknowledges NATS messages on success
func (c *Consumer) flushBatchWithAck(workerID int, batch []Event, msgs []*nats.Msg) bool {
	if len(batch) == 0 {
		return true
	}

	start := time.Now()

	// Retry logic
	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Warnf("Worker %d: Retry attempt %d for batch of %d events", workerID, attempt, len(batch))
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		err = c.insertBatch(batch)
		if err == nil {
			break
		}

		log.Errorf("Worker %d: Insert failed (attempt %d): %v", workerID, attempt+1, err)
	}

	if err != nil {
		log.Errorf("Worker %d: Failed to insert batch after %d retries: %v", workerID, maxRetries, err)
		c.errors.Add(uint64(len(batch)))
		// NAK all messages so they can be redelivered
		for _, msg := range msgs {
			msg.Nak()
		}
		return false
	}

	// Success! Acknowledge all messages
	for _, msg := range msgs {
		if err := msg.Ack(); err != nil {
			log.Warnf("Worker %d: Failed to ack message: %v", workerID, err)
		}
	}

	// Update metrics
	c.eventsInserted.Add(uint64(len(batch)))
	c.batchesFlushed.Add(1)

	duration := time.Since(start)
	log.Debugf("Worker %d: Flushed %d events in %v (%.0f events/sec)",
		workerID, len(batch), duration, float64(len(batch))/duration.Seconds())

	return true
}

// insertBatch performs the actual ClickHouse insert
func (c *Consumer) insertBatch(batch []Event) error {
	ctx := context.Background()

	// Prepare batch insert
	insertBatch, err := c.clickhouse.PrepareBatch(ctx, `
		INSERT INTO telemetry_events (
			agent_id, timestamp, event_type, mitre_tactic, mitre_technique,
			severity, payload, tenant_id, hostname, os_type
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	// Map event type strings to enum values
	eventTypeMap := map[string]string{
		"PROCESS_START":      "process_start",
		"PROCESS_TERMINATE":  "process_terminate",
		"FILE_ACCESS":        "file_access",
		"FILE_MODIFY":        "file_modify",
		"FILE_DELETE":        "file_delete",
		"NETWORK_CONN":       "network_conn",
		"REGISTRY_MODIFY":    "registry_modify",
		"DLP_VIOLATION":      "dlp_violation",
		"AUTHENTICATION":     "authentication",
	}

	// Append rows
	for _, event := range batch {
		// Convert timestamp from milliseconds to DateTime64
		timestamp := time.UnixMilli(event.Timestamp)

		// Map event type
		eventType := eventTypeMap[event.EventType]
		if eventType == "" {
			eventType = "unspecified"
		}

		err = insertBatch.Append(
			event.AgentID,
			timestamp,
			eventType,
			event.MitreTactic,
			event.MitreTechnique,
			event.Severity,
			event.Payload,
			event.TenantID,
			event.Hostname,
			event.OSType,
		)
		if err != nil {
			return fmt.Errorf("failed to append row: %w", err)
		}
	}

	// Execute batch insert
	if err := insertBatch.Send(); err != nil {
		return fmt.Errorf("failed to send batch: %w", err)
	}

	return nil
}

// printStats periodically logs performance statistics
func (c *Consumer) printStats(ctx context.Context) {
	ticker := time.NewTicker(statsInterval)
	defer ticker.Stop()

	var lastProcessed, lastInserted, lastBatches uint64
	lastTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			processed := c.eventsProcessed.Load()
			inserted := c.eventsInserted.Load()
			batches := c.batchesFlushed.Load()
			errors := c.errors.Load()
			now := time.Now()
			elapsed := now.Sub(lastTime).Seconds()

			processedPerSec := float64(processed-lastProcessed) / elapsed
			insertedPerSec := float64(inserted-lastInserted) / elapsed
			batchesPerSec := float64(batches-lastBatches) / elapsed

			log.Infof("Performance: %.0f events/sec processed, %.0f events/sec inserted, %.1f batches/sec | Total: %d processed, %d inserted, %d errors",
				processedPerSec, insertedPerSec, batchesPerSec, processed, inserted, errors)

			lastProcessed = processed
			lastInserted = inserted
			lastBatches = batches
			lastTime = now
		}
	}
}

// Close gracefully shuts down the consumer
func (c *Consumer) Close() error {
	log.Info("Closing connections...")

	if c.natsConn != nil {
		c.natsConn.Close()
	}

	if c.clickhouse != nil {
		if err := c.clickhouse.Close(); err != nil {
			log.Errorf("Error closing ClickHouse: %v", err)
		}
	}

	return nil
}

func main() {
	// Configure logging
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)
	log.Info("Privé Consumer Worker starting...")

	// Load configuration
	natsURL := getEnv("NATS_URL", nats.DefaultURL)
	clickhouseAddr := getEnv("CLICKHOUSE_ADDR", "localhost:9000")

	// Create consumer
	consumer, err := NewConsumer(natsURL, clickhouseAddr)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info("Shutdown signal received, stopping consumer...")
		cancel()
	}()

	// Start consuming
	if err := consumer.Start(ctx); err != nil {
		log.Fatalf("Consumer error: %v", err)
	}

	log.Info("Consumer worker stopped gracefully")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
