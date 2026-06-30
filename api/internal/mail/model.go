package mail

import "time"

type EmailPayload struct {
	To      string
	Subject string
	Body    string
}

type EmailConfig struct {
	SMTPHost      string
	SMTPPort      int
	DefaultSender string
}

type SentEmail struct {
	ID        int64
	EventID   string
	Recipient string
	Subject   string
	//ProviderMessageID *string
	SentAt time.Time
}
