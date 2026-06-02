package rabbitmqbroker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/alfin-akhret/ecommerce-system/internal/events"
	amqp "github.com/rabbitmq/amqp091-go"
)

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

func queueArgs(eventName string) amqp.Table {
	return amqp.Table{
		amqp.QueueTypeArg:           amqp.QueueTypeQuorum,
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": eventName + ".dlq",
	}
}
