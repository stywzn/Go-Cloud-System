package mq

import (
	"context"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// DeadLetterMessage structure for DLQ events
type DeadLetterMessage struct {
	QuotaUpgradeEvent QuotaUpgradeEvent `json:"event"`
	FailureReason     string            `json:"failure_reason"`
	RetryCount        int32             `json:"retry_count"`
	LastFailedAt      time.Time         `json:"last_failed_at"`
	TraceID           string            `json:"trace_id"`
}

// QuotaUpgradeEvent payload
type QuotaUpgradeEvent struct {
	UserID   int    `json:"user_id"`
	QuotaAdd int64  `json:"quota_add"`
	EventID  string `json:"event_id"`
}

// StartDLQMonitor starts the background consumer for the Dead Letter Queue
func StartDLQMonitor(ctx context.Context) {
	log.Println("Starting DLQ Monitor...")

	// 1. Get an independent channel for the DLQ consumer
	consumeCh, err := GetChannel()
	if err != nil {
		log.Printf("Failed to get channel for DLQ consumer: %v", err)
		return
	}

	msgs, err := consumeCh.Consume(
		"storage.quota.queue.dlq",
		"dlq-monitor",
		false, // Auto-Ack MUST be false for reliability
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Printf("Failed to register DLQ consumer: %v", err)
		return
	}

	go func() {
		defer consumeCh.Close() // Ensure consumer channel is closed on exit
		for {
			select {
			case <-ctx.Done():
				log.Println("DLQ Monitor shutting down...")
				return
			case msg := <-msgs:
				handleDeadLetter(ctx, msg)
			}
		}
	}()
}

func handleDeadLetter(ctx context.Context, msg amqp.Delivery) {
	traceID := ""
	if headers, ok := msg.Headers["X-Trace-ID"]; ok {
		if traceIDStr, ok := headers.(string); ok {
			traceID = traceIDStr
		}
	}

	var event QuotaUpgradeEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("[TraceID: %s] Failed to parse DLQ message body: %v", traceID, err)
		// Malformed message, reject permanently
		msg.Nack(false, false)
		return
	}

	retryCount := getRetryCount(msg)

	if retryCount < 3 {
		// Process retry asynchronously to avoid blocking the consumer pipeline
		go processRetry(ctx, msg, traceID, retryCount)
	} else {
		// 3. Prevent Data Loss: Max retries reached, MUST persist manually
		persistFailedMessage(event, traceID, retryCount)
		// Safe to Ack only after permanent persistence
		msg.Ack(false)
	}
}

// processRetry handles exponential backoff and republishing
func processRetry(ctx context.Context, msg amqp.Delivery, traceID string, currentRetry int32) {
	// 2. Exponential Backoff: wait 5s, then 10s, then 20s...
	delay := time.Duration(1<<currentRetry) * 5 * time.Second
	log.Printf("[TraceID: %s] Delaying retry #%d for %v...", traceID, currentRetry+1, delay)

	select {
	case <-time.After(delay):
		// Proceed with republish
	case <-ctx.Done():
		// Server shutting down during sleep, return message to DLQ
		msg.Nack(false, true)
		return
	}

	// Get a new temporary channel for publishing
	pubCh, err := GetChannel()
	if err != nil {
		log.Printf("[TraceID: %s] Failed to get publish channel: %v", traceID, err)
		msg.Nack(false, true) // Requeue to DLQ on infrastructure failure
		return
	}
	defer pubCh.Close()

	// Prepare new headers
	headers := amqp.Table{}
	for k, v := range msg.Headers {
		headers[k] = v
	}
	headers["x-retry-count"] = currentRetry + 1
	headers["X-Trace-ID"] = traceID

	err = pubCh.PublishWithContext(ctx,
		"quota_exchange", // Main exchange
		"quota.add",      // Main routing key
		false,
		false,
		amqp.Publishing{
			ContentType:  msg.ContentType,
			Body:         msg.Body,
			DeliveryMode: amqp.Persistent,
			Headers:      headers,
		})

	if err != nil {
		log.Printf("[TraceID: %s] Failed to republish message: %v", traceID, err)
		msg.Nack(false, true) // Requeue to DLQ
		return
	}

	log.Printf("[TraceID: %s] Successfully republished message (Retry: %d)", traceID, currentRetry+1)

	// Only Ack the original DLQ message after successful republish
	msg.Ack(false)
}

// getRetryCount safely extracts the retry count from headers
func getRetryCount(msg amqp.Delivery) int32 {
	if countRaw, ok := msg.Headers["x-retry-count"]; ok {
		switch v := countRaw.(type) {
		case int32:
			return v
		case int64:
			return int32(v)
		case int:
			return int32(v)
		}
	}
	return 0
}

// persistFailedMessage acts as the final safety net for dead messages
func persistFailedMessage(event QuotaUpgradeEvent, traceID string, retryCount int32) {
	// TODO: In a real project, INSERT this into a MySQL table (e.g., failed_events)
	// Example: INSERT INTO failed_events (user_id, event_id, status) VALUES (?, ?, 'FAILED')

	log.Printf("[TraceID: %s] CRITICAL: Message permanently failed after %d retries. Event: %+v. Require manual compensation.",
		traceID, retryCount, event)
}
