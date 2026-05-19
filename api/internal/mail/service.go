package mail

import (
	"context"
	"fmt"

	"github.com/alfin-akhret/ecommerce-system/internal/events"
	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	mailClient "github.com/go-mail/mail/v2"
	"go.uber.org/zap"
)

type Service struct {
	broker events.Broker
	cfg    *EmailConfig
}

func NewService(broker events.Broker, cfg *EmailConfig) *Service {
	return &Service{
		broker: broker,
		cfg:    cfg,
	}
}

func (s *Service) SubscribeTo(topic string) {
	s.broker.Subscribe(topic, func(ctx context.Context, event events.Event) {
		log := helper.LoggerFromCtx(ctx)

		payload := EmailPayload{
			From: s.cfg.DefaultSender,
			To:   "user@anywhere.com",
		}

		switch {
		case topic == "order.created":
			p := event.Payload.(events.OrderCreatedPayload)
			payload.Subject = "Order Created"
			payload.Body = fmt.Sprintf("Your order with ID:%s has been created", p.OrderID)
		case topic == "payment.callback.processed":
			p := event.Payload.(events.PaymentCallbackProcessedPayload)
			payload.Subject = "Payment Status"
			payload.Body = fmt.Sprintf("Your payment status with ID: %s for Order: %s was %s",
				p.PaymentID, p.OrderID, p.Status)
		}

		err := s.sendMail(ctx, payload)
		if err != nil {
			log.Error("Something wrong", zap.String("error", err.Error()))
			return
		}

		log.Info("Email sent", zap.String("to", payload.To), zap.String("Subject", payload.Subject))
	})
}

func (s *Service) sendMail(ctx context.Context, payload EmailPayload) error {
	log := helper.LoggerFromCtx(ctx)

	m := mailClient.NewMessage()

	m.SetHeader("From", payload.From)
	m.SetHeader("To", payload.To)
	m.SetHeader("Subject", payload.Subject)
	m.SetBody("text/plain", payload.Body)

	d := mailClient.NewDialer(
		s.cfg.SMTPHost,
		s.cfg.SMTPPort,
		"",
		"",
	)

	if err := d.DialAndSend(m); err != nil {
		log.Error("Failed to send email", zap.String("error", err.Error()))
		return err
	}

	return nil

}
