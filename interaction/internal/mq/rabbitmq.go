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

	// 声明一个 Direct 类型的交换机
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

	// 声明队列
	q, err := Channel.QueueDeclare(
		"storage.quota.queue", // 队列名称
		true,                  // 持久化 (重要：防止 MQ 重启丢消息)
		false,                 // auto-delete
		false,                 // exclusive
		false,                 // no-wait
		nil,                   // args
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
