package jobs

import (
	"context"
	"encoding/json"
	"fmt"
)

type SendEmailPayload struct {
	OrderID string
	Email   string
}

func SendEmailHandler(ctx context.Context, payload []byte) error {
	var p SendEmailPayload
	json.Unmarshal(payload, &p)

	fmt.Println("sending email for order:", p.OrderID)

	return nil
}
