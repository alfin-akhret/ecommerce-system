package mail

import (
	"context"

	"github.com/alfin-akhret/ecommerce-system/internal/events"
	"github.com/alfin-akhret/ecommerce-system/pkg/helper"
	mailClient "github.com/go-mail/mail/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"
)

type Service struct {
	broker    events.Broker
	cfg       *EmailConfig
	repo      *Repository
	inboxRepo events.InboxRepositoryManager
}

func NewService(broker events.Broker,
	cfg *EmailConfig,
	db *pgxpool.Pool,
	inboxRepo events.InboxRepositoryManager) *Service {

	repo := NewEmailRepository(db)
	return &Service{
		broker:    broker,
		cfg:       cfg,
		repo:      repo,
		inboxRepo: inboxRepo,
	}
}

func (s *Service) SubscribeTo(ctx context.Context, topic string) {
	s.broker.Subscribe(ctx, topic, "mail-service", func(ctx context.Context, event events.Event) error {

		inboxEvent := events.InboxEvent{
			ID:        event.ID,
			EventType: topic,
			Payload:   event.RawPayload,
			Status:    events.InboxPending,
			CreatedAt: event.CreatedAt,
		}

		return s.inboxRepo.Save(ctx, inboxEvent)

		/**
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
				case topic == "payment.expired":
					p := event.Payload.(events.PaymentExpiredPayload)
					payload.Subject = "Payment Expired"
					payload.Body = fmt.Sprintf(
						`Your order information:

		Order ID: %s
		Payment ID: %s
		Order Status: CANCELED
		Payment Status: %s
		`,
						p.OrderID,
						p.PaymentID,
						"EXPIRED",
					)
				}

				// idempotency:
				// 1. kalau message sudah pernah sukses diproses, skip dan ack
				if s.repo.IsProcessed(ctx, event.ID) {
					return nil
				}

				// 2. kirim email dulu
				log.Info("Sending email...", zap.String("body", payload.Body))

				err := s.sendMail(ctx, payload)
				if err != nil {
					log.Error("Something wrong", zap.String("error", err.Error()))
					return err
				}

				// 3. Baru tandai processed setelah email sukses
				err = s.repo.InsertProcessedMessage(ctx, event.ID)
				if err != nil {
					if IsDuplicateKeyError(err) {
						log.Error("Duplicate event: Event has been processed before", zap.String("error", err.Error()))
						return nil
					}
					return err
				}

				// kelemahan cara diatas adalah
				// jika email sukses dikirim, lalu service crash sebelum InserProcessedMessage,
				// maka event bisa retry dan akibatnya email akan terkirim dua kali
				// ini hal biasa di sistem event driven disebut dg istilah
				// "at least once + idempotent consumer"
				// tapi ini masih ada solusinya. -> inbox pattern

				return nil
		*/
	})
}

func (s *Service) sendMail(ctx context.Context, payload EmailPayload) error {
	log := helper.LoggerFromCtx(ctx)
	tr := otel.Tracer("mail.service")
	ctx, span := tr.Start(ctx, "mail.service.sendMail")
	defer span.End()

	span.SetAttributes(
		attribute.String("To", payload.To),
		attribute.String("Subject", payload.Subject),
	)

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
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		log.Error("Failed to send email", zap.String("error", err.Error()))
		return err
	}

	log.Info("Email sent", zap.String("to", payload.To), zap.String("Subject", payload.Subject))

	return nil

}
