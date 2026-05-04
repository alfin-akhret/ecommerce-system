package jobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"go.uber.org/zap"
)

type SendEmailPayload struct {
	OrderID string
	Email   string
}

func SendEmailHandler(ctx context.Context, payload []byte) error {
	logger := helper.LoggerFromCtx(ctx)

	var p SendEmailPayload
	json.Unmarshal(payload, &p)

	logger.Info("Sending email", zap.String("order_id", p.OrderID), zap.String("email", p.Email))
	fmt.Println("sending email for order:", p.OrderID)

	return nil
}
