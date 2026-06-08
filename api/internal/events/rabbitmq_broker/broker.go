package rabbitmqbroker

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/alfin-akhret/ecommerce-system/internal/events"
	amqp "github.com/rabbitmq/amqp091-go"
)

const eventExchange = "ecommerce.events"

type RabbitMQBroker struct {
	conn        *amqp.Connection
	publisherCh *amqp.Channel
	consumerChs []*amqp.Channel

	mu sync.Mutex
}

func CreateNewBroker(connString string) *RabbitMQBroker {
	conn, err := amqp.Dial(connString)
	failOnError(err, "Failed to connect to rabbitMQ")

	pubCh, err := conn.Channel()
	failOnError(err, "Failed to open channel")

	b := &RabbitMQBroker{
		conn:        conn,
		publisherCh: pubCh,
	}

	if err := b.declareExchange(pubCh); err != nil {
		failOnError(err, "failed to declare exchange")
	}

	return b
}

func (r *RabbitMQBroker) declareExchange(ch *amqp.Channel) error {
	return ch.ExchangeDeclare(
		eventExchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Printf("%s: %s", msg, err)
	}
}

func (r *RabbitMQBroker) Publish(ctx context.Context, event events.Event, retryCount int) error {

	// publish
	body, err := json.Marshal(event.Payload)
	if err != nil {
		log.Printf("Error parsing event payload: %v", err.Error())
		return err
	}

	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = r.publisherCh.PublishWithContext(
		publishCtx,
		eventExchange, // exchange
		event.Name,    // routing key
		false,         // mandatory
		false,         // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Headers: amqp.Table{
				"x-retry-count": retryCount,
			},
		})
	if err != nil {
		log.Printf(" [x] Sent %v\n", body)
		return err
	}
	return nil
}

func (r *RabbitMQBroker) Subscribe(eventName string, subscriberName string, handler events.Handler) {

	// open new channel for each subscriber
	ch, err := r.conn.Channel()
	failOnError(err, "Failed to open channel")

	r.mu.Lock()
	r.consumerChs = append(r.consumerChs, ch)
	r.mu.Unlock()

	if err := r.declareExchange(ch); err != nil {
		failOnError(err, "failed to declare exchange")
	}

	queueName := subscriberName + "." + eventName
	dlqName := queueName + ".dlq"

	_, err = ch.QueueDeclare(dlqName,
		true, false, false, false, nil)
	failOnError(err, "failed to declare DLQ")

	_, err = ch.QueueDeclare(queueName,
		true, false, false, false,
		amqp.Table{
			amqp.QueueTypeArg:           amqp.QueueTypeQuorum,
			"x-dead-letter-exchange":    "",
			"x-dead-letter-routing-key": dlqName,
		})
	failOnError(err, "failed to declare queue")

	err = ch.QueueBind(
		queueName,
		eventName,
		eventExchange,
		false,
		nil,
	)
	failOnError(err, "failed to bind queue")

	// set Qos
	if err := ch.Qos(1, 0, false); err != nil {
		failOnError(err, "failed to set QoS")
	}

	// consume
	msgs, err := ch.Consume(
		eventName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	failOnError(err, "Failed to register a consumer")

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)

			retryCount := getRetryCount(d.Headers)

			event, err := decodeEvent(eventName, d.Body)
			if err != nil {
				_ = d.Nack(false, false)
				continue
			}

			ctx := context.Background()

			if err := handler(ctx, event); err != nil {
				if retryCount >= 3 {
					_ = d.Nack(false, false)
					continue
				}

				if err := r.Publish(ctx, event, retryCount+1); err != nil {
					_ = d.Nack(false, false)
					continue
				}

				_ = d.Ack(false)
				continue
			}

			_ = d.Ack(false)
		}
	}()
}

func (r *RabbitMQBroker) Close() error {
	for _, ch := range r.consumerChs {
		if err := ch.Close(); err != nil {
			log.Printf("failed to close consumer channerl: %v", err)
		}
	}

	if r.publisherCh != nil {
		if err := r.publisherCh.Close(); err != nil {
			log.Printf("failed to close publisher channel: %v", err)
		}
	}

	if r.conn != nil {
		return r.conn.Close()
	}

	return nil
}
