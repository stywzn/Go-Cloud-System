package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stywzn/Go-Cloud-System/interaction/config"
	"github.com/stywzn/Go-Cloud-System/interaction/internal/mq"
	"github.com/stywzn/Go-Cloud-System/pkg/trace"
)

type QuotaUpgradeEvent struct {
	UserID   int    `json:"user_id"`
	QuotaAdd int64  `json:"quota_add"`
	EventID  string `json:"event_id"`
}

// Fixed Lua script: properly checks actual stock value in Redis
const seckillLuaScript = `
local stockKey = KEYS[1]
local winnerSetKey = KEYS[2]
local userID = ARGV[1]

-- 1. Check if user already won (Idempotency)
if redis.call('SISMEMBER', winnerSetKey, userID) == 1 then
    return -1 
end

-- 2. Get actual stock from Redis
local stock = tonumber(redis.call('GET', stockKey) or '0')

-- 3. Check stock and deduct
if stock <= 0 then
    return 0
end

redis.call('DECR', stockKey)
redis.call('SADD', winnerSetKey, userID)
return 1
`

// Note: Removed unused 'var Rdb *redis.Client' as config.Redis is used directly.

func DoSeckill(ctx context.Context, userID int) error {
	traceID := trace.GetTraceIDFromContext(ctx)

	stockKey := "lottery:quota:stock"
	winnerSetKey := "lottery:quota:winners"

	// Execute Lua script atomically
	result, err := config.Redis.Eval(ctx, seckillLuaScript, []string{stockKey, winnerSetKey}, userID).Result()
	if err != nil {
		log.Printf("[TraceID: %s] Redis Eval error: %v", traceID, err)
		return errors.New("system busy, please try again later")
	}

	// Safe type assertion to prevent panic
	resInt, ok := result.(int64)
	if !ok {
		log.Printf("[TraceID: %s] Redis Lua returned unexpected type", traceID)
		return errors.New("internal system error")
	}

	if resInt == -1 {
		return errors.New("you have already won")
	}
	if resInt == 0 {
		return errors.New("out of stock")
	}

	// Prepare MQ message
	event := QuotaUpgradeEvent{
		UserID:   userID,
		QuotaAdd: 4 * 1024 * 1024 * 1024,
		EventID:  fmt.Sprintf("lottery_event_%d_%s", userID, traceID),
	}

	body, _ := json.Marshal(event)

	// In DoSeckill function, before publishing:
	ch, err := mq.GetChannel()
	if err != nil {
		return errors.New("failed to get MQ channel")
	}
	defer ch.Close() // Ensure channel is closed to prevent resource leak

	// Publish to RabbitMQ
	err = ch.PublishWithContext(ctx,
		"quota_exchange",
		"quota.add",
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // Ensure message survives MQ restarts
			Headers: amqp.Table{
				"X-Trace-ID": traceID,
			},
		})

	if err != nil {
		// TODO: Implement reliable message delivery (e.g., Transactional Outbox pattern or local retry)
		// Currently logging as critical. Manual intervention required if this triggers.
		log.Printf("[TraceID: %s] CRITICAL: Deducted stock but failed to publish MQ message: %v", traceID, err)
		return errors.New("failed to dispatch reward, please contact support")
	}

	log.Printf("[TraceID: %s] User %d successfully grabbed quota, message dispatched", traceID, userID)
	return nil
}
