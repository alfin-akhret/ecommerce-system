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

	declaredQueues map[string]bool
}

func CreateNewBroker(connString string) *RabbitMQBroker {
	conn, err := amqp.Dial(connString)
	failOnError(err, "Failed to connect to rabbitMQ")

	pubCh, err := conn.Channel()
	failOnError(err, "Failed to open channel")

	return &RabbitMQBroker{
		conn:           conn,
		publisherCh:    pubCh,
		declaredQueues: make(map[string]bool),
	}
}

func (r *RabbitMQBroker) ensureQueue(eventName string) error {
	if r.declaredQueues[eventName] {
		return nil
	}

	_, err := r.publisherCh.QueueDeclare(eventName, true, false, false, false, queueArgs(eventName))
	if err != nil {
		return err
	}

	_, err = r.publisherCh.QueueDeclare(eventName+".dlq", true, false, false, false, nil)
	if err != nil {
		return err
	}

	r.declaredQueues[eventName] = true
	return nil
}

func declareQueue(ch *amqp.Channel, eventName string) error {
	_, err := ch.QueueDeclare(eventName, true, false, false, false, queueArgs(eventName))
	if err != nil {
		return err
	}

	_, err = ch.QueueDeclare(eventName+".dlq", true, false, false, false, nil)
	return err
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Printf("%s: %s", msg, err)
	}
}

func (r *RabbitMQBroker) Publish(ctx context.Context, event events.Event, retryCount int) error {

	if err := r.ensureQueue(event.Name); err != nil {
		log.Printf("failed to declare queue")
		return err
	}

	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// publish
	body, err := json.Marshal(event.Payload)
	if err != nil {
		log.Printf("Error parsing event payload: %v", err.Error())
		return err
	}

	// set retry count on rabbitMQ header
	headers := amqp.Table{
		"x-retry-count": retryCount,
	}
	err = r.publisherCh.PublishWithContext(publishCtx,
		"",         // exchange
		event.Name, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Headers:      headers,
		})
	if err != nil {
		log.Printf(" [x] Sent %v\n", body)
		return err
	}
	return nil
}

func (r *RabbitMQBroker) Subscribe(eventName string, handler events.Handler) {

	// open new channel for each subscriber
	ch, err := r.conn.Channel()
	failOnError(err, "Failed to open channel")

	r.consumerChs = append(r.consumerChs, ch)

	// create queue
	if err := declareQueue(ch, eventName); err != nil {
		log.Printf("failed to declare queue: %v", err)
	}

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

			retryCount := 0

			if d.Headers != nil {
				if v, ok := d.Headers["x-retry-count"]; ok {
					switch val := v.(type) {
					case int32:
						retryCount = int(val)
					case int64:
						retryCount = int(val)
					case int:
						retryCount = val
					}
				}
			}

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
				if retryCount > 3 {
					if nackErr := d.Nack(false, false); nackErr != nil {
						log.Printf("failed to NACK event: %v", nackErr)
					}
					continue
				}

				retryCount += 1

				if err := r.Publish(ctx, event, retryCount); err != nil {
					_ = d.Nack(false, true)
					continue
				}

				_ = d.Ack(false)
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
