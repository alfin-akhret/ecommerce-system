package rabbitmqbroker

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQBroker struct {
	host string
}

func CreateNewBroker(host string) *RabbitMQBroker {
	return &RabbitMQBroker{
		host: host,
	}
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

type Publisher struct {
	topic string
}

func CreateNewPublisher(topic string) *Publisher {
	p := &Publisher{topic: topic}
	return p
}

func (p *Publisher) Publish() {
	// open connection
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	failOnError(err, "Failed to connect to rabbitMQ")
	defer conn.Close()

	// open channel
	ch, err := conn.Channel()
	failOnError(err, "Failed to open channel")
	defer ch.Close()

	// create queue
	q, err := ch.QueueDeclare(
		p.topic,
		true,
		false,
		false,
		false,
		amqp.Table{
			amqp.QueueTypeArg: amqp.QueueTypeQuorum,
		},
	)
	failOnError(err, "failed to declare a queue")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// publish
	body := "Hello World!"
	err = ch.PublishWithContext(ctx,
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(body),
		})
	failOnError(err, "Failed to publish a message")
	log.Printf(" [x] Sent %s\n", body)
}

type Consumer struct {
	topic string
}

func CreateNewConsumer(topic string) *Consumer {
	c := &Consumer{topic: topic}
	return c
}

func (c *Consumer) Consume() {
	// open connection
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	failOnError(err, "Failed to connect to rabbitMQ")
	defer conn.Close()

	// open channel
	ch, err := conn.Channel()
	failOnError(err, "Failed to open channel")
	defer ch.Close()

	// create queue
	q, err := ch.QueueDeclare(
		c.topic,
		true,
		false,
		false,
		false,
		amqp.Table{
			amqp.QueueTypeArg: amqp.QueueTypeQuorum,
		},
	)
	failOnError(err, "failed to declare a queue")

	// publish
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	// Create a channel to receive OS signals
	chn := make(chan os.Signal, 1)
	// Notify the channel for SIGINT (CTRL+C) and SIGTERM
	signal.Notify(chn, os.Interrupt, syscall.SIGTERM)
	// Block until a signal is received
	<-chn
	log.Printf("Shutting down gracefully...")
	// Deferred conn.Close() and ch.Close() will execute!
}
