package rabbitmqbroker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/alfin-akhret/ecommerce-system/internal/events"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQBroker struct {
	conn        *amqp.Connection
	publisherCh *amqp.Channel
	consumerChs []*amqp.Channel
}

func CreateNewBroker(connString string) *RabbitMQBroker {
	conn, err := amqp.Dial(connString)
	failOnError(err, "Failed to connect to rabbitMQ")

	pubCh, err := conn.Channel()
	failOnError(err, "Failed to open channel")

	return &RabbitMQBroker{
		conn:        conn,
		publisherCh: pubCh,
	}
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func (r *RabbitMQBroker) Publish(ctx context.Context, event events.Event) {

	// create main queue
	q, err := r.publisherCh.QueueDeclare(
		event.Name,
		true,
		false,
		false,
		false,
		queueArgs(event.Name),
	)
	failOnError(err, "failed to declare a queue")

	// create DLQ queue
	_, err = r.publisherCh.QueueDeclare(
		event.Name+".dlq",
		true,
		false,
		false,
		false,
		nil,
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
	err = r.publisherCh.PublishWithContext(publishCtx,
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
		})
	failOnError(err, "Failed to publish a message")
	log.Printf(" [x] Sent %v\n", body)
}

func (r *RabbitMQBroker) Subscribe(eventName string, handler events.Handler) {

	// open new channel for each subscriber
	ch, err := r.conn.Channel()
	failOnError(err, "Failed to open channel")

	r.consumerChs = append(r.consumerChs, ch)

	// create queue
	q, err := ch.QueueDeclare(
		eventName,
		true,
		false,
		false,
		false,
		queueArgs(eventName),
	)
	failOnError(err, "failed to declare a queue")

	// set Qos
	if err := ch.Qos(1, 0, false); err != nil {
		failOnError(err, "failed to set QoS")
	}

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
				if nackErr := d.Nack(false, false); nackErr != nil {
					log.Printf("Failed to NACK event: %v", nackErr.Error())
				}
				log.Printf("Failed to decode event: %v", err.Error())
				continue
			}

			ctx := context.Background()
			err = handler(ctx, event)
			if err != nil {
				// jika handler error dan NACK berhasil, maka message akan masuk lagi ke queue
				// dan akan di retry
				if nackErr := d.Nack(false, false); nackErr != nil {
					// jika NACK error, print log
					log.Printf("Failed to NACK event: %v", nackErr.Error())
				}
				log.Printf("Handler failed: %v", err.Error())
				continue
			}

			if ackErr := d.Ack(false); ackErr != nil {
				log.Printf("Failed to ACK event: %v", ackErr.Error())
				continue
			}
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
