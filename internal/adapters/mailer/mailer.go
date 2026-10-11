package utils

import (
	"enviroo-be/internal/config"
	"fmt"
	"strconv"

	"gopkg.in/gomail.v2"
)

// EmailParams berisi parameter yang diperlukan untuk mengirim email.
// Cukup isi To, Subject, dan Body (HTML string) saat memanggil dari controller.
type EmailParams struct {
	To      string // Alamat email tujuan
	Subject string // Subjek email
	Body    string // Isi email dalam format HTML
}

// Mailer menyimpan konfigurasi SMTP yang sudah di-load dari config.
type Mailer struct {
	host       string
	port       int
	email      string
	password   string
	senderName string
}

// NewMailer membuat instance Mailer dari Config.
// Panggil sekali di main.go, lalu pass ke controller yang membutuhkan.
func NewMailer(cfg *config.Config) *Mailer {
	port, _ := strconv.Atoi(cfg.SMTPPort)
	return &Mailer{
		host:       cfg.SMTPHost,
		port:       port,
		email:      cfg.SMTPEmail,
		password:   cfg.SMTPPassword,
		senderName: cfg.SMTPSenderName,
	}
}

// SendEmail mengirim email menggunakan konfigurasi SMTP yang sudah tersimpan.
// Contoh penggunaan di controller:
//
//	err := mailer.SendEmail(utils.EmailParams{
//	    To:      "user@example.com",
//	    Subject: "Verifikasi Akun Enviroo",
//	    Body:    "<h1>Halo!</h1><p>Klik link berikut untuk verifikasi...</p>",
//	})
func (m *Mailer) SendEmail(params EmailParams) error {
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
