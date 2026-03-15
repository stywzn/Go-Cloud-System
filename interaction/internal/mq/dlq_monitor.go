package mq

import (
	"context"
	"encoding/json"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// DeadLetterMessage 死信消息结构
type DeadLetterMessage struct {
	QuotaUpgradeEvent QuotaUpgradeEvent `json:"event"`
	OriginalMessage   amqp.Delivery     `json:"original_message"`
	FailureReason     string            `json:"failure_reason"`
	RetryCount        int               `json:"retry_count"`
	FirstFailedAt     time.Time         `json:"first_failed_at"`
	LastFailedAt      time.Time         `json:"last_failed_at"`
	TraceID           string            `json:"trace_id"`
}

// QuotaUpgradeEvent 配额升级事件
type QuotaUpgradeEvent struct {
	UserID   int    `json:"user_id"`
	QuotaAdd int64  `json:"quota_add"`
	EventID  string `json:"event_id"`
}

// StartDLQMonitor 启动死信队列监控
func StartDLQMonitor(ctx context.Context) {
	log.Println("启动死信队列监控...")

	// 创建死信队列消费者
	msgs, err := Channel.Consume(
		"storage.quota.queue.dlq", // 队列名称
		"dlq-monitor",             // 消费者标签
		false,                     // 自动确认
		false,                     // 独占
		false,                     // 不等待
		false,                     // 参数
		nil,
	)
	if err != nil {
		log.Printf("死信队列消费者创建失败: %v", err)
		return
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("死信队列监控停止")
				return
			case msg := <-msgs:
				handleDeadLetter(ctx, msg)
			}
		}
	}()
}

// handleDeadLetter 处理死信消息
func handleDeadLetter(ctx context.Context, msg amqp.Delivery) {
	traceID := ""
	if headers, ok := msg.Headers["X-Trace-ID"]; ok {
		if traceIDStr, ok := headers.(string); ok {
			traceID = traceIDStr
		}
	}

	log.Printf("[TraceID: %s] 收到死信消息: %s", traceID, string(msg.Body))

	// 解析原始消息
	var event QuotaUpgradeEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("[TraceID: %s] 解析死信消息失败: %v", traceID, err)
		msg.Nack(false, false) // 拒绝消息，不重新入队
		return
	}

	// 记录死信消息信息
	dlqMsg := DeadLetterMessage{
		QuotaUpgradeEvent: event,
		OriginalMessage:   msg,
		FailureReason:     "处理失败或超时",
		LastFailedAt:      time.Now(),
		TraceID:           traceID,
	}

	// 这里可以：
	// 1. 记录到数据库
	// 2. 发送告警通知
	// 3. 手动重试逻辑
	log.Printf("[TraceID: %s] 死信消息详情: %+v", traceID, dlqMsg)

	// 根据业务逻辑决定是否重试
	if shouldRetry(msg) {
		log.Printf("[TraceID: %s] 尝试重新投递消息", traceID)
		republishMessage(ctx, msg, traceID)
	}

	// 确认死信消息处理完成
	msg.Ack(false)
}

// shouldRetry 判断是否应该重试
func shouldRetry(msg amqp.Delivery) bool {
	// 检查重试次数
	if retryCount, ok := msg.Headers["x-retry-count"]; ok {
		if count, ok := retryCount.(int32); ok && count >= 3 {
			return false // 超过最大重试次数
		}
	}
	return true
}

// republishMessage 重新发布消息
func republishMessage(ctx context.Context, originalMsg amqp.Delivery, traceID string) {
	// 增加重试次数
	retryCount := int32(0)
	if count, ok := originalMsg.Headers["x-retry-count"]; ok {
		if c, ok := count.(int32); ok {
			retryCount = c + 1
		}
	}

	// 创建新的消息头
	headers := amqp.Table{}
	for k, v := range originalMsg.Headers {
		headers[k] = v
	}
	headers["x-retry-count"] = retryCount
	headers["X-Trace-ID"] = traceID

	// 重新发布消息
	err := Channel.PublishWithContext(ctx,
		"quota_exchange",
		"quota.add",
		false,
		false,
		amqp.Publishing{
			ContentType:  originalMsg.ContentType,
			Body:         originalMsg.Body,
			DeliveryMode: amqp.Persistent,
			Headers:      headers,
		})

	if err != nil {
		log.Printf("[TraceID: %s] 重试消息发布失败: %v", traceID, err)
	} else {
		log.Printf("[TraceID: %s] 消息重试发布成功 (重试次数: %d)", traceID, retryCount)
	}
}
