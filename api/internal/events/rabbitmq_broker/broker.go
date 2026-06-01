package rabbitmqbroker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/alfin-akhret/ecommerce-system/internal/events"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQBroker struct {
	conn  *amqp.Connection
	pubCh *amqp.Channel
}

func CreateNewBroker(connString string) *RabbitMQBroker {
	conn, err := amqp.Dial(connString)
	failOnError(err, "Failed to connect to rabbitMQ")

	pubCh, err := conn.Channel()
	failOnError(err, "Failed to open channel")

	return &RabbitMQBroker{
		conn:  conn,
		pubCh: pubCh, // publisher channel
	}
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func (r *RabbitMQBroker) Publish(ctx context.Context, event events.Event) {

	// create queue
	q, err := r.pubCh.QueueDeclare(
		event.Name,
		true,
		false,
		false,
		false,
		amqp.Table{
			amqp.QueueTypeArg: amqp.QueueTypeQuorum,
		},
	)
	failOnError(err, "failed to declare a queue")
	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// publish
	body, err := json.Marshal(event.Payload)
	if err != nil {
		log.Printf("Error parsing event payload: %v", err.Error())
		return
	}
	err = r.pubCh.PublishWithContext(publishCtx,
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	failOnError(err, "Failed to publish a message")
	log.Printf(" [x] Sent %v\n", body)
}

func (r *RabbitMQBroker) Subscribe(eventName string, handler events.Handler) {

	// open new channel for each subscriber
	ch, err := r.conn.Channel()
	failOnError(err, "Failed to open channel")

	// create queue
	q, err := ch.QueueDeclare(
		eventName,
		true,
		false,
		false,
		false,
		amqp.Table{
			amqp.QueueTypeArg: amqp.QueueTypeQuorum,
		},
	)
	failOnError(err, "failed to declare a queue")

	// consume
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)

			event, err := decodeEvent(eventName, d.Body)
			if err != nil {
				d.Nack(false, false)
				continue
			}

			ctx := context.Background()
			handler(ctx, event)

			d.Ack(false)
		}
	}()

	/*
		log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
		// Create a channel to receive OS signals
		chn := make(chan os.Signal, 1)
		// Notify the channel for SIGINT (CTRL+C) and SIGTERM
		signal.Notify(chn, os.Interrupt, syscall.SIGTERM)
		// Block until a signal is received
		<-chn
		log.Printf("Shutting down gracefully...")
		// Deferred conn.Close() and ch.Close() will execute!
	*/
}

func decodeEvent(eventName string, body []byte) (events.Event, error) {
	event := events.Event{
		Name:      eventName,
		CreatedAt: time.Now(),
	}

	switch eventName {
	case "order.created":
		var payload events.OrderCreatedPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return event, err
		}
		event.Payload = payload

	case "payment.callback.processed":
		var payload events.PaymentCallbackProcessedPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return event, err
		}
		event.Payload = payload

	case "payment.expired":
		var payload events.PaymentExpiredPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return event, err
		}
		event.Payload = payload

	default:
		return event, fmt.Errorf("unknown event name: %s", eventName)
	}

	return event, nil

}
