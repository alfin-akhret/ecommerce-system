package mail

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
