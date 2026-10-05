// Package email delivers one-time sign-in codes.
//
// SMTPMailer works with any free SMTP relay: a Gmail account with an app
// password (500/day), Brevo (300/day free) or similar. LogMailer prints the
// code to the server log and is what local development uses when no SMTP
// host is configured.
package email

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// SMTPMailer sends plain-text mail over STARTTLS with PLAIN auth.
type SMTPMailer struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string // e.g. "Sloan Marketplace <you@gmail.com>"
}

func (m *SMTPMailer) SendCode(ctx context.Context, to, code string) error {
	subject := fmt.Sprintf("Your Sloan Marketplace code: %s", code)
	body := fmt.Sprintf("Your sign-in code is %s\r\n\r\nIt expires in 10 minutes. "+
		"If you didn't ask for it, you can ignore this email.\r\n", code)
	msg := strings.Join([]string{
		"From: " + m.From,
		"To: " + to,
		"Subject: " + subject,
		"Date: " + time.Now().Format(time.RFC1123Z),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	auth := smtp.PlainAuth("", m.Username, m.Password, m.Host)
	errc := make(chan error, 1)
	go func() {
		errc <- smtp.SendMail(net.JoinHostPort(m.Host, m.Port), auth, envelopeAddress(m.From), []string{to}, []byte(msg))
	}()
	select {
	case err := <-errc:
		if err != nil {
			return fmt.Errorf("email: send to %s: %w", to, err)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// envelopeAddress pulls "a@b" out of "Name <a@b>".
func envelopeAddress(from string) string {
	if i := strings.LastIndex(from, "<"); i >= 0 {
		return strings.TrimSuffix(from[i+1:], ">")
	}
	return from
}

// LogMailer writes the code to the server log instead of sending mail.
type LogMailer struct{}

func (LogMailer) SendCode(_ context.Context, to, code string) error {
	log.Printf("email (dev): sign-in code for %s is %s", to, code)
	return nil
}
