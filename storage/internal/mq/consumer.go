package mq

import (
	"context"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stywzn/Go-Cloud-System/storage/pkg/db" // 假设你的 MySQL 连接在这里
)

// QuotaUpgradeEvent 必须与 Interaction 服务中定义的结构体保持完全一致
type QuotaUpgradeEvent struct {
	UserID   int    `json:"user_id"`
	QuotaAdd int64  `json:"quota_add"`
	EventID  string `json:"event_id"`
}

// StartQuotaConsumer 启动后台消费者协程
func StartQuotaConsumer(amqpURL string) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		log.Fatalf("Storage 连接 RabbitMQ 失败: %v", err)
	}
	// 注意：这里不要关闭 conn，因为消费者需要一直运行在后台

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Storage 打开 Channel 失败: %v", err)
	}

	// 声明队列（确保存储服务先启动时队列也存在）
	q, err := ch.QueueDeclare(
		"storage.quota.queue", // 队列名
		true,                  // 持久化
		false,                 // auto-delete
		false,                 // exclusive
		false,                 // no-wait
		nil,                   // args
	)
	if err != nil {
		log.Fatalf("Storage 声明队列失败: %v", err)
	}

	// 开始消费（autoAck 必须设置为 false，我们要手动确认！）
	msgs, err := ch.Consume(
		q.Name, // 队列
		"",     // 消费者标签
		false,  // auto-ack (核心：关闭自动确认)
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatalf("注册消费者失败: %v", err)
	}

	log.Println("[*] Storage 正在后台监听加配额消息...")

	// 开启一个永久运行的 Goroutine 处理消息
	go func() {
		for d := range msgs {
			log.Printf("收到消息: %s", d.Body)

			var event QuotaUpgradeEvent
			if err := json.Unmarshal(d.Body, &event); err != nil {
				log.Printf("消息解析失败，丢弃: %v", err)
				d.Ack(false) // 格式错的消息直接确认掉，防止死循环阻塞
				continue
			}

			// 处理加配额的业务逻辑
			err := processQuotaUpgrade(event)
			if err != nil {
				log.Printf("处理配额增加失败 (UserID: %d): %v", event.UserID, err)
				// 处理失败，拒绝确认并重新入队（生产环境通常配合死信队列 DLQ 使用）
				d.Nack(false, true)
				continue
			}

			// 完美成功，手动发送 ACK 确认！消息正式从 MQ 中删除
			log.Printf("✅ 用户 %d 成功增加配额 %d 字节", event.UserID, event.QuotaAdd)
			d.Ack(false)
		}
	}()
}

// processQuotaUpgrade 核心业务逻辑：落库 + 幂等校验
func processQuotaUpgrade(event QuotaUpgradeEvent) error {
	ctx := context.Background()

	// 开启数据库事务（保证插入流水和更新配额同时成功或同时失败）
	tx := db.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 1. 幂等性校验：尝试插入事件流水
	// 如果这行代码报错（触发唯一索引冲突），说明这条消息已经被处理过了
	err := tx.Exec("INSERT INTO lottery_event_records (event_id, user_id) VALUES (?, ?)", event.EventID, event.UserID).Error
	if err != nil {
		tx.Rollback()
		// 判断是否是 Duplicate Key 冲突 (这里简化处理，认为插入失败就是重复消费)
		log.Printf("⚠️ 消息 %s 已被处理过，触发幂等防重，忽略本次操作", event.EventID)
		return nil // 注意这里返回 nil，让外部发送 ACK，把重复的消息消费掉
	}

	// 2. 核心操作：给用户加上真实配额
	// 假设你 user 表里的字段叫 total_quota
	err = tx.Exec("UPDATE users SET total_quota = total_quota + ? WHERE id = ?", event.QuotaAdd, event.UserID).Error
	if err != nil {
		tx.Rollback()
		return err // 数据库更新失败，返回 err，让外部执行 Nack 重试
	}

	// 提交事务
	return tx.Commit().Error
}
