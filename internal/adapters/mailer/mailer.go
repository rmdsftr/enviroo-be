package mailer

import (
	"enviroo-be/internal/config"
	"fmt"
	"strconv"

	"gopkg.in/gomail.v2"
)

type Params struct {
	To      string
	Subject string
	Body    string
}

type Mailer struct {
	host       string
	port       int
	email      string
	password   string
	senderName string
}

func New(cfg *config.Config) *Mailer {
	port, _ := strconv.Atoi(cfg.SMTPPort)
	return &Mailer{
		host:       cfg.SMTPHost,
		port:       port,
		email:      cfg.SMTPEmail,
		password:   cfg.SMTPPassword,
		senderName: cfg.SMTPSenderName,
	}
}

func (m *Mailer) SendEmail(params Params) error {
	msg := gomail.NewMessage()
	msg.SetHeader("From", fmt.Sprintf("%s <%s>", m.senderName, m.email))
	msg.SetHeader("To", params.To)
	msg.SetHeader("Subject", params.Subject)
	msg.SetBody("text/html", params.Body)

	dialer := gomail.NewDialer(m.host, m.port, m.email, m.password)

	if err := dialer.DialAndSend(msg); err != nil {
		return fmt.Errorf("gagal mengirim email ke %s: %w", params.To, err)
	}

	return nil
}
