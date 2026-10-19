// Package email delivers one-time sign-in codes.
//
// BrevoMailer uses Brevo's HTTPS API (300/day free) and works from hosts
// that block SMTP. SMTPMailer works with any SMTP relay, such as a Gmail
// account with an app password. LogMailer prints the code to the server log
// and is what local development uses when neither is configured.
package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
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

	// Hosts that block outbound SMTP (Railway's trial plan does) drop the
	// connection silently, so bound the attempt rather than hang the request.
	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

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
		return fmt.Errorf("email: send to %s: timed out after %s (is outbound SMTP blocked?)", to, sendTimeout)
	}
}

const sendTimeout = 15 * time.Second

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

// BrevoMailer sends through Brevo's HTTP API (free tier: 300 emails/day).
// It exists because some hosts, Railway's Trial and Hobby plans among them,
// block outbound SMTP ports entirely, while HTTPS always gets through.
type BrevoMailer struct {
	APIKey string
	From   string // "Name <addr>" or just "addr"; the address must be a verified sender in Brevo
	HTTP   *http.Client
}

func (m *BrevoMailer) SendCode(ctx context.Context, to, code string) error {
	name, addr := splitAddress(m.From)
	if name == "" {
		name = "Sloan Marketplace" // Brevo requires a sender name
	}
	payload, _ := json.Marshal(map[string]any{
		"sender":      map[string]string{"name": name, "email": addr},
		"to":          []map[string]string{{"email": to}},
		"subject":     fmt.Sprintf("Your Sloan Marketplace code: %s", code),
		"textContent": fmt.Sprintf("Your sign-in code is %s\n\nIt expires in 10 minutes. If you didn't ask for it, you can ignore this email.\n", code),
	})
	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.brevo.com/v3/smtp/email", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("api-key", m.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := m.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("email: brevo send to %s: %w", to, err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return fmt.Errorf("email: brevo send to %s: %s: %s", to, res.Status, msg)
	}
	return nil
}

// splitAddress turns "Name <a@b>" into ("Name", "a@b"); a bare address
// yields an empty name.
func splitAddress(from string) (name, addr string) {
	addr = envelopeAddress(from)
	if i := strings.LastIndex(from, "<"); i >= 0 {
		name = strings.TrimSpace(from[:i])
	}
	return name, addr
}
