package rabbitmqbroker

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/alfin-akhret/ecommerce-system/internal/events"
	amqp "github.com/rabbitmq/amqp091-go"
)

func decodeEvent(eventName string, body []byte, messageID string) (events.Event, error) {
	event := events.Event{
		ID:        messageID,
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

func getRetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}

	v, ok := headers["x-retry-count"]
	if !ok {
		return 0
	}

	switch val := v.(type) {
	case int:
		return val
	case int32:
		return int(val)
	case int64:
		return int(val)
	default:
		return 0
	}
}

func queueArgs(eventName string) amqp.Table {
	return amqp.Table{
		amqp.QueueTypeArg:           amqp.QueueTypeQuorum,
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": eventName + ".dlq",
	}
}

const (
	maxRetry       = 3
	baseRetryDelay = 1 * time.Second
	maxRetryDelay  = 30 * time.Second
)

func retryDelay(retryCount int) time.Duration {
	if retryCount < 1 {
		return baseRetryDelay
	}

	delay := baseRetryDelay * time.Duration(1<<(retryCount-1))
	if delay > maxRetryDelay {
		return maxRetryDelay
	}

	return delay
}
