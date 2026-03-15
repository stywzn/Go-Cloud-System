package mq

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	Conn    *amqp.Connection
	Channel *amqp.Channel
)

// InitRabbitMQ 初始化连接并声明交换机与队列
func InitRabbitMQ(amqpURL string) {
	var err error
	Conn, err = amqp.Dial(amqpURL)
	if err != nil {
		log.Fatalf("无法连接到 RabbitMQ: %v", err)
	}

	Channel, err = Conn.Channel()
	if err != nil {
		log.Fatalf("无法打开 RabbitMQ Channel: %v", err)
	}

	// 声明死信交换机
	err = Channel.ExchangeDeclare(
		"quota_exchange_dlq", // 死信交换机名称
		"direct",             // 类型
		true,                 // 是否持久化
		false,                // 是否自动删除
		false,                // 内部使用
		false,                // no-wait
		nil,                  // arguments
	)
	if err != nil {
		log.Fatalf("声明死信Exchange失败: %v", err)
	}

	// 声明死信队列
	dlqQueue, err := Channel.QueueDeclare(
		"storage.quota.queue.dlq", // 死信队列名称
		true,                      // 持久化
		false,                     // auto-delete
		false,                     // exclusive
		false,                     // no-wait
		amqp.Table{
			"x-dead-letter-exchange":    "quota_exchange", // 死信回源交换机
			"x-dead-letter-routing-key": "quota.add",      // 死信回源路由键
		},
	)
	if err != nil {
		log.Fatalf("声明死信Queue失败: %v", err)
	}

	// 绑定死信队列到死信交换机
	err = Channel.QueueBind(
		dlqQueue.Name,        // 队列名
		"quota.add.dlq",      // 死信路由键
		"quota_exchange_dlq", // 死信交换机
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("绑定死信Queue失败: %v", err)
	}

	// 声明主交换机
	err = Channel.ExchangeDeclare(
		"quota_exchange", // 交换机名称
		"direct",         // 类型
		true,             // 是否持久化
		false,            // 是否自动删除
		false,            // 内部使用
		false,            // no-wait
		nil,              // arguments
	)
	if err != nil {
		log.Fatalf("声明 Exchange 失败: %v", err)
	}

	// 声明主队列（带死信配置）
	q, err := Channel.QueueDeclare(
		"storage.quota.queue", // 队列名称
		true,                  // 持久化 (重要：防止 MQ 重启丢消息)
		false,                 // auto-delete
		false,                 // exclusive
		false,                 // no-wait
		amqp.Table{
			"x-dead-letter-exchange":    "quota_exchange_dlq", // 死信交换机
			"x-dead-letter-routing-key": "quota.add.dlq",      // 死信路由键
		},
	)
	if err != nil {
		log.Fatalf("声明 Queue 失败: %v", err)
	}

	// 将队列绑定到交换机
	err = Channel.QueueBind(
		q.Name,           // 队列名
		"quota.add",      // 路由键 (Routing Key)
		"quota_exchange", // 交换机
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("绑定 Queue 失败: %v", err)
	}

	log.Println("RabbitMQ 初始化成功，已就绪！")
}

// Close 关闭连接
func Close() {
	if Channel != nil {
		Channel.Close()
	}
	if Conn != nil {
		Conn.Close()
	}
}
