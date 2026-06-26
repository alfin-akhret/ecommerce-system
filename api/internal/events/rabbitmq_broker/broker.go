package rabbitmqbroker

import (
	"context"
	"encoding/json"
	"errors"
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
	confirmCh   chan amqp.Confirmation
	consumerChs []*amqp.Channel

	publishMu sync.Mutex
	mu        sync.Mutex
}

func CreateNewBroker(connString string) *RabbitMQBroker {
	conn, err := amqp.Dial(connString)
	failOnError(err, "Failed to connect to rabbitMQ")

	pubCh, err := conn.Channel()
	failOnError(err, "Failed to open channel")

	// aktifkan Publisher confirm mode
	if err := pubCh.Confirm(false); err != nil {
		failOnError(err, "failed to enable publisher confirm")
	}

	// inisialisasi confirmation channel
	confirmCh := pubCh.NotifyPublish(
		make(chan amqp.Confirmation, 1),
	)

	b := &RabbitMQBroker{
		conn:        conn,
		publisherCh: pubCh,
		confirmCh:   confirmCh,
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

func (r *RabbitMQBroker) Publish(ctx context.Context, event events.Event, retryCount int) (error, string) {
	// biar ga race condition antar goroutine
	r.publishMu.Lock()
	defer r.publishMu.Unlock()

	// publish
	body, err := json.Marshal(event)
	if err != nil {
		log.Printf("Error parsing event payload: %v", err.Error())
		return err, ""
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
			Timestamp:    event.CreatedAt, // biar createdAt event sinkron di semua part (producer, publisher, consumer)
			MessageId:    event.ID,
			Headers: amqp.Table{
				"x-retry-count": retryCount,
			},
		})
	if err != nil {
		log.Printf("{ERROR} %s\n", err.Error())
		return err, ""
	}

	select {
	case confirm, ok := <-r.confirmCh:
		if !ok {
			return errors.New("publisher confirm channel closed"), ""
		}
		if confirm.Ack {
			return nil, "ACK"
		}
		return nil, "NACK"

	case <-time.After(5 * time.Second):
		return errors.New("publisher confirm timeout..."), ""
	}
}

func (r *RabbitMQBroker) Subscribe(
	ctx context.Context,
	eventName string,
	subscriberName string,
	handler events.Handler) {

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
		queueName, // queue
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

			event, err := decodeEvent(d.Body)
			if err != nil {
				log.Printf("failed decoding event, retrying events=%s retry=%d, err=%v",
					event.Name, retryCount+1, err)
				_ = d.Nack(false, false)
				continue
			}

			if err := handler(ctx, event); err != nil {
				if retryCount >= maxRetry {
					_ = d.Nack(false, false)
					continue
				}

				delay := retryDelay(retryCount + 1)
				log.Printf("handler failed, retrying events=%s retry=%d delay=%s err=%v",
					event.Name, retryCount+1, delay, err)

				select {
				case <-time.After(delay):
				case <-ctx.Done():
					_ = d.Nack(false, false)
					return // context sudah cancel, consumer sebaiknya berhenti semua
				}

				if err, _ := r.Publish(ctx, event, retryCount+1); err != nil {
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
