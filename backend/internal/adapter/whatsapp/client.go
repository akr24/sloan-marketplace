// Package whatsapp talks to the WhatsApp Business Cloud API (Meta's hosted
// API; free to use with a test number, and service conversations a user
// starts are free).
//
// Client sends text replies. Inbound messages arrive at the webhook in the
// HTTP adapter, which hands them to the bot use case.
package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const graphURL = "https://graph.facebook.com/v21.0"

// Sender delivers a text message to a WhatsApp number (E.164 digits).
type Sender interface {
	SendText(ctx context.Context, to, body string) error
}

// Client is the Cloud API implementation of Sender.
type Client struct {
	PhoneNumberID string
	AccessToken   string
	HTTP          *http.Client
}

func NewClient(phoneNumberID, accessToken string) *Client {
	return &Client{PhoneNumberID: phoneNumberID, AccessToken: accessToken, HTTP: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) SendText(ctx context.Context, to, body string) error {
	payload, _ := json.Marshal(map[string]any{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "text",
		"text":              map[string]any{"body": body, "preview_url": true},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/%s/messages", graphURL, c.PhoneNumberID), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("whatsapp: send: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return fmt.Errorf("whatsapp: send: %s: %s", res.Status, msg)
	}
	return nil
}

// LogSender prints replies to the server log; used when no Cloud API
// credentials are configured, and by the local /api/dev/whatsapp simulator.
type LogSender struct{}

func (LogSender) SendText(_ context.Context, to, body string) error {
	log.Printf("whatsapp (dev): reply to +%s:\n%s", to, body)
	return nil
}
