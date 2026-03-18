package mq

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	// Only keep the Connection as a global variable.
	// TCP connections are safe for concurrent use across goroutines.
	Conn *amqp.Connection
)

// InitRabbitMQ initializes the connection and declares exchanges and queues
func InitRabbitMQ(amqpURL string) {
	var err error
	Conn, err = amqp.Dial(amqpURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	// Open a temporary channel purely for setup
	setupChannel, err := Conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open setup channel: %v", err)
	}
	// Ensure the setup channel is closed after initialization
	defer setupChannel.Close()

	// Declare Dead Letter Exchange
	err = setupChannel.ExchangeDeclare(
		"quota_exchange_dlq", "direct", true, false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare DLX: %v", err)
	}

	// Declare Dead Letter Queue
	dlqQueue, err := setupChannel.QueueDeclare(
		"storage.quota.queue.dlq", true, false, false, false,
		amqp.Table{
			"x-dead-letter-exchange":    "quota_exchange",
			"x-dead-letter-routing-key": "quota.add",
		},
	)
	if err != nil {
		log.Fatalf("Failed to declare DLQ: %v", err)
	}

	// Bind DLQ to DLX
	err = setupChannel.QueueBind(
		dlqQueue.Name, "quota.add.dlq", "quota_exchange_dlq", false, nil,
	)
	if err != nil {
		log.Fatalf("Failed to bind DLQ: %v", err)
	}

	// Declare Main Exchange
	err = setupChannel.ExchangeDeclare(
		"quota_exchange", "direct", true, false, false, false, nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare Main Exchange: %v", err)
	}

	// Declare Main Queue with DLX config
	q, err := setupChannel.QueueDeclare(
		"storage.quota.queue", true, false, false, false,
		amqp.Table{
			"x-dead-letter-exchange":    "quota_exchange_dlq",
			"x-dead-letter-routing-key": "quota.add.dlq",
		},
	)
	if err != nil {
		log.Fatalf("Failed to declare Main Queue: %v", err)
	}

	// Bind Main Queue to Main Exchange
	err = setupChannel.QueueBind(
		q.Name, "quota.add", "quota_exchange", false, nil,
	)
	if err != nil {
		log.Fatalf("Failed to bind Main Queue: %v", err)
	}

	log.Println("RabbitMQ initialized successfully")
}

// GetChannel creates and returns a new channel from the global connection.
// The caller is responsible for deferring channel.Close().
func GetChannel() (*amqp.Channel, error) {
	if Conn == nil || Conn.IsClosed() {
		return nil, amqp.ErrClosed
	}
	return Conn.Channel()
}

// Close gracefully closes the AMQP connection.
// Closing the connection automatically closes all associated channels.
func Close() {
	if Conn != nil && !Conn.IsClosed() {
		Conn.Close()
	}
}
