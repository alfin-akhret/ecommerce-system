package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	"go.uber.org/zap"
)

type SendEmailPayload struct {
	OrderID string
	Email   string
}

func SendEmailHandler(ctx context.Context, payload []byte) error {
	logger := helper.LoggerFromCtx(ctx)

	// for testing
	delay := time.Duration(20) * time.Second
	time.Sleep(delay)
	return errors.New("something wrong")

	var p SendEmailPayload
	json.Unmarshal(payload, &p)

	logger.Info("Sending email", zap.String("order_id", p.OrderID), zap.String("email", p.Email))
	fmt.Println("sending email for order:", p.OrderID)

	return nil
}
