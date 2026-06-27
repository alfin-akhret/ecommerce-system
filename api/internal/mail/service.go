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
			EventType: event.Name,
			Payload:   event.RawPayload,
			CreatedAt: event.CreatedAt,
			Status:    events.InboxPending,
		}

		return s.inboxRepo.Save(ctx, inboxEvent)
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

	m.SetHeader("From", s.cfg.DefaultSender)
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
