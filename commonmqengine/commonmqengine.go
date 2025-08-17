package commonmqengine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"sync"

	"github.com/fabioluissilva/microservicetemplate/commonlogger"
	"github.com/rabbitmq/amqp091-go"
	"jmartins.com/messageadapterphotos3/internal/configuration"
)

// Package mqengine provides an interface to interact with RabbitMQ for message queuing.

/* =========================
   Example usage
   ========================= */

// cfg := NewMQConfiguration(
// 	WithCredentials("user", "pass"),
// 	WithHost("rabbit.internal"),
// 	WithPort(5672),
// 	WithVHost("myapp"),
// 	WithQueues(
// 		NewQueue("orders",
// 			WithExchange("orders-ex"),
// 			WithRoutingKey("orders.*"),
// 			WithDurable(true),
// 			WithArgs(map[string]interface{}{
// 				"x-message-ttl": int32(60000),
// 			}),
// 		),
// 		NewQueue("audit",
// 			WithAutoDelete(true),
// 		),
// 	),
// )

type QueueConfiguration struct {
	ExchangeName string
	RoutingKey   string
	Name         string
	Durable      bool
	AutoDelete   bool
	Exclusive    bool
	NoWait       bool
	Args         map[string]interface{}
}

type MQConfiguration struct {
	Username string
	Password string
	MqHost   string
	MqPort   int
	VHost    string
	Queues   []QueueConfiguration
}

/* =========================
   MQ options & constructor
   ========================= */

type MQOption func(*MQConfiguration)

func NewMQConfiguration(opts ...MQOption) *MQConfiguration {
	cfg := &MQConfiguration{
		MqHost: "localhost",
		MqPort: 5672,
		VHost:  "/",
		Queues: []QueueConfiguration{},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

func WithCredentials(username, password string) MQOption {
	return func(c *MQConfiguration) {
		c.Username = username
		c.Password = password
	}
}

func WithHost(host string) MQOption {
	return func(c *MQConfiguration) { c.MqHost = host }
}

func WithPort(port int) MQOption {
	return func(c *MQConfiguration) { c.MqPort = port }
}

func WithVHost(vhost string) MQOption {
	return func(c *MQConfiguration) { c.VHost = vhost }
}

func WithQueue(q QueueConfiguration) MQOption {
	return func(c *MQConfiguration) { c.Queues = append(c.Queues, q) }
}

func WithQueues(queues ...QueueConfiguration) MQOption {
	return func(c *MQConfiguration) { c.Queues = append(c.Queues, queues...) }
}

/* =========================
   Queue options & builder
   ========================= */

type QueueOption func(*QueueConfiguration)

func NewQueue(name string, opts ...QueueOption) QueueConfiguration {
	q := QueueConfiguration{
		Name:       name,
		Durable:    true, // good default
		AutoDelete: false,
		Exclusive:  false,
		NoWait:     false,
		Args:       make(map[string]interface{}),
	}
	for _, opt := range opts {
		opt(&q)
	}
	return q
}

func WithExchange(name string) QueueOption {
	return func(q *QueueConfiguration) { q.ExchangeName = name }
}

func WithRoutingKey(key string) QueueOption {
	return func(q *QueueConfiguration) { q.RoutingKey = key }
}

func WithDurable(b bool) QueueOption {
	return func(q *QueueConfiguration) { q.Durable = b }
}

func WithAutoDelete(b bool) QueueOption {
	return func(q *QueueConfiguration) { q.AutoDelete = b }
}

func WithExclusive(b bool) QueueOption {
	return func(q *QueueConfiguration) { q.Exclusive = b }
}

func WithNoWait(b bool) QueueOption {
	return func(q *QueueConfiguration) { q.NoWait = b }
}

func WithArgs(args map[string]interface{}) QueueOption {
	return func(q *QueueConfiguration) { q.Args = args }
}

var (
	channel  *amqp091.Channel
	conn     *amqp091.Connection
	mu       sync.Mutex
	mqconfig MQConfiguration
)

func GetChannel() *amqp091.Channel {
	mu.Lock()
	defer mu.Unlock()
	return channel
}

func ensureChannel() error {
	var err error
	url, urlObfuscated := buildUrl()

	if conn == nil || conn.IsClosed() {
		commonlogger.Warn("[MQEngine] ensureChannel: connection is not initialized or is closed. Reconnecting to RabbitMQ at URL: " + urlObfuscated)
		conn, err = amqp091.Dial(url)
		if err != nil {
			commonlogger.Error("[MQEngine] ensureChannel: Failed to connect to RabbitMQ", slog.Any("error", err))
			return fmt.Errorf("[MQEngine] ensureChannel: Failed to connect to RabbitMQ: %w", err)
		}
	}

	if channel == nil || channel.IsClosed() {
		channel, err = conn.Channel()
		commonlogger.Warn("[MQEngine] ensureChannel: channel is not open. Opening Channel")
		if err != nil {
			commonlogger.Error("[MQEngine] ensureChannel: Failed to open Channel", slog.Any("error", err))
			return fmt.Errorf("[MQEngine] ensureChannel: Failed to open Channel: %w", err)
		}
	}
	commonlogger.Debug("[MQEngine] ensureChannel: Channel is open and ready to use at url: " + urlObfuscated)
	return nil
}

func buildUrl() (string, string) {
	password := mqconfig.Password
	obfuscatedPassword := password
	if password != "" && len(password) > 4 {
		obfuscatedPassword = password[:4] + "..."
	}
	url := fmt.Sprintf("amqp://%s:%s@%s:%d/%s", mqconfig.Username, mqconfig.Password, mqconfig.MqHost, mqconfig.MqPort, mqconfig.VHost)
	urlObfuscated := fmt.Sprintf("amqp://%s:%s@%s:%d/%s", mqconfig.Username, obfuscatedPassword, mqconfig.MqHost, mqconfig.MqPort, mqconfig.VHost)
	commonlogger.Debug("[MQEngine] Engine: Connection URL: " + urlObfuscated)
	return url, urlObfuscated
}

func ConnectRabbitMQ(ctx context.Context) error {
	mu.Lock()
	defer mu.Unlock()

	commonlogger.Info("[MQEngine] Connecting to RabbitMQ at Host: " + mqconfig.MqHost + " Port: " + fmt.Sprintf("%d", mqconfig.MqPort) + " VHost: " + mqconfig.VHost)
	// Connect to RabbitMQ server

	err := ensureChannel()
	if err != nil {
		commonlogger.Error("[MQEngine] Failed to ensure channel is open", slog.Any("error", err))
		return fmt.Errorf("[MQEngine] Failed to ensure channel is open: %w", err)
	}

	commonlogger.Info("[MQEngine] Declaring Queue: " + configuration.GetConfig().QueueName)
	_, err = channel.QueueDeclare(
		configuration.GetConfig().QueueName, // name
		true,                                // durable
		false,                               // delete when unused
		false,                               // exclusive
		false,                               // no-wait
		nil,                                 // arguments
	)
	if err != nil {
		return fmt.Errorf("[MQEngine] failed to declare queue: %w", err)
	}

	err = channel.QueueBind(
		configuration.GetConfig().QueueName,    // queue name
		configuration.GetConfig().QueueName,    // routing key
		configuration.GetConfig().ExchangeName, // exchange
		false,                                  // no-wait
		nil,                                    // arguments
	)
	if err != nil {
		return fmt.Errorf("[MQEngine] failed to bind queue: %w", err)
	}
	commonlogger.Info("[MQEngine] Queue " + configuration.GetConfig().QueueName + " declared and bound successfully")
	// Declare Retry Queue
	_, err = channel.QueueDeclare(
		configuration.GetConfig().RetryQueueName, // name
		true,                                     // durable
		false,                                    // delete when unused
		false,                                    // exclusive
		false,                                    // no-wait
		amqp091.Table{
			"x-message-ttl":             int32(configuration.GetConfig().RetryTTL),
			"x-dead-letter-exchange":    "",                                  // Default Exchange for direct delivery
			"x-dead-letter-routing-key": configuration.GetConfig().QueueName, // main queue name
		},
	)
	if err != nil {
		return fmt.Errorf("[MQEngine] failed to declare retry queue: %w", err)
	}
	commonlogger.Info("[MQEngine] Retry Queue " + configuration.GetConfig().RetryQueueName + " declared successfully")
	// Declare Retry Queue
	_, err = channel.QueueDeclare(
		configuration.GetConfig().DeadLetterQueueName, // name
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("[MQEngine] failed to declare Dead Letter queue: %w", err)
	}
	commonlogger.Info("[MQEngine] Dead Letter Queue " + configuration.GetConfig().DeadLetterQueueName + " declared successfully")

	return nil
}

func SendMessageToQueue(message string, system string, contenttype string, correlationId string) (string, error) {
	mu.Lock()
	defer mu.Unlock()

	err := ensureChannel()
	if err != nil {
		commonlogger.Error("[MQEngine] Failed to ensure channel is open", slog.Any("error", err))
		return "", fmt.Errorf("[MQEngine] Failed to ensure channel is open: %w", err)
	}

	commonlogger.Info("[MQEngine] Sending message to queue: " + configuration.GetConfig().QueueName)
	// Publish a message to the queue
	err = channel.PublishWithContext(context.Background(),
		configuration.GetConfig().ExchangeName, // exchange
		configuration.GetConfig().QueueName,    // routing key
		false,                                  // mandatory
		false,                                  // immediate
		amqp091.Publishing{
			ContentType:   contenttype,
			Body:          []byte(message),
			CorrelationId: correlationId,
			AppId:         system,
			Headers:       amqp091.Table{},
		})
	if err != nil {
		return "", fmt.Errorf("[MQEngine] failed to publish message: %w", err)
	}
	return message, nil
}

// ConsumeFromQueue reads a message from the RabbitMQ queue
// If autoAck is true, the message will be acknowledged automatically when consumed
// Otherwise, the caller is responsible for acknowledging the message
func ConsumeFromQueue(queueName string, autoAck bool) (<-chan amqp091.Delivery, error) {

	mu.Lock()
	defer mu.Unlock()

	err := ensureChannel()
	if err != nil {
		commonlogger.Error("[MQEngine] Failed to ensure channel is open", slog.Any("error", err))
		return nil, fmt.Errorf("[MQEngine] Failed to ensure channel is open: %w", err)
	}

	commonlogger.Info("[MQEngine] Starting to consume from queue: " + queueName)

	deliveries, err := channel.Consume(
		queueName, // queue name
		"",        // consumer tag (empty string generates a unique tag)
		autoAck,   // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("[MQEngine] failed to register consumer: %w", err)
	}

	commonlogger.Info("[MQEngine] Consumer registered successfully")
	return deliveries, nil
}

func SaveMessageToFile(correlationId string, body string, headers map[string]interface{}) error {
	// Save the message body to <correlationId>.json
	bodyFileName := correlationId + ".json"
	err := os.WriteFile(bodyFileName, []byte(body), 0644)
	if err != nil {
		return fmt.Errorf("failed to write message body to file %s: %w", bodyFileName, err)
	}

	// Save the headers to <correlationId>_headers.json
	headersFileName := correlationId + "_headers.json"
	headersData, err := json.MarshalIndent(headers, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal headers to JSON: %w", err)
	}
	err = os.WriteFile(headersFileName, headersData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write headers to file %s: %w", headersFileName, err)
	}

	return nil
}

func CopyMessageToQueue(message amqp091.Delivery, targetQueue string) error {
	mu.Lock()
	defer mu.Unlock()
	retryTTL := 0

	err := ensureChannel()
	if err != nil {
		commonlogger.Error("[MQEngine] Failed to ensure channel is open", slog.Any("error", err))
		return fmt.Errorf("[MQEngine] Failed to ensure channel is open: %w", err)
	}

	headers := message.Headers

	if headers["X-Retry-Count"] != nil {
		retryCount := headers["X-Retry-Count"].(int32)
		retryCount++
		headers["X-Retry-Count"] = retryCount
	} else {
		headers["X-Retry-Count"] = 1
	}

	if headers["X-Retry-TTL"] != nil {
		retryTTL = headers["X-Retry-TTL"].(int)
	}

	publishing := amqp091.Publishing{
		ContentType:   message.ContentType,
		Body:          message.Body,
		CorrelationId: message.CorrelationId,
		AppId:         message.AppId,
		Headers:       headers,
		ReplyTo:       message.ReplyTo,
		MessageId:     message.MessageId,
		Timestamp:     message.Timestamp,
	}

	// Max Retry Attempts reached, move to Dead Letter Queue and remove timeout
	if retryCount, ok := headers["X-Retry-Count"].(int32); ok && retryCount >= int32(configuration.GetConfig().RetryMaxAttempts+1) {
		commonlogger.Debug("[MQEngine] Max Retry Attempts reached. Moving to Dead Letter Queue.")
		targetQueue = configuration.GetConfig().DeadLetterQueueName
		retryTTL = 0 // No expiration time for dead letter queue
	} else {
		if retryTTL > 0 {
			publishing.Expiration = strconv.Itoa(retryTTL)
		}
	}
	commonlogger.Debug(fmt.Sprintf("[MQEngine] Copying message to queue: %s with headers: %v Retry Count: %v Expiration Time: %s", targetQueue, headers, headers["X-Retry-Count"], publishing.Expiration))

	// Publish the message to the target queue

	err = channel.PublishWithContext(
		context.Background(),
		"", // default exchange to publish to the queue directly
		targetQueue,
		false,
		false,
		publishing,
	)
	if err != nil {
		return fmt.Errorf("[MQEngine] failed to copy message: %w", err)
	}
	return nil
}

func Close() {
	mu.Lock()
	defer mu.Unlock()

	if channel != nil {
		channel.Close()
	}
	if conn != nil {
		conn.Close()
	}
	channel = nil
	conn = nil
}
func IsConnected() bool {
	mu.Lock()
	defer mu.Unlock()

	if channel != nil && conn != nil {
		return true
	}
	return false
}
func GetConnection() *amqp091.Connection {
	mu.Lock()
	defer mu.Unlock()

	if conn != nil {
		return conn
	}
	return nil
}

// IsHealthy checks if the RabbitMQ connection and channel are healthy.
func IsHealthy() bool {
	mu.Lock()
	defer mu.Unlock()

	if conn == nil || channel == nil {
		commonlogger.Error("RabbitMQ connection or channel is not initialized", "package", "mqengine", "service", configuration.GetConfig().MessageAdapterName)
		return false
	}

	if conn.IsClosed() {
		commonlogger.Error("RabbitMQ connection is closed", "package", "mqengine", "service", configuration.GetConfig().MessageAdapterName)
		return false
	}

	return true
}
