package http

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/akr24/sloan-marketplace/backend/internal/usecase"
)

type whatsappHandler struct {
	bot  *usecase.Bot
	cfg  WhatsAppConfig
	seen *recentIDs
}

// verify answers Meta's subscription handshake.
func (h whatsappHandler) verify(c echo.Context) error {
	if h.cfg.VerifyToken == "" ||
		c.QueryParam("hub.mode") != "subscribe" ||
		!hmac.Equal([]byte(c.QueryParam("hub.verify_token")), []byte(h.cfg.VerifyToken)) {
		return echo.NewHTTPError(http.StatusForbidden, "verification failed")
	}
	return c.String(http.StatusOK, c.QueryParam("hub.challenge"))
}

// webhookPayload is the subset of the Cloud API notification we read.
type webhookPayload struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				Messages []struct {
					ID   string `json:"id"`
					From string `json:"from"`
					Type string `json:"type"`
					Text struct {
						Body string `json:"body"`
					} `json:"text"`
				} `json:"messages"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

// receive handles inbound messages. Meta retries anything that doesn't get
// a 200 quickly, so failures are logged rather than returned, and message
// ids are de-duplicated so a retry can't post the same listing twice.
func (h whatsappHandler) receive(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "unreadable body")
	}
	if h.cfg.AppSecret != "" && !validSignature(h.cfg.AppSecret, body, c.Request().Header.Get("X-Hub-Signature-256")) {
		return echo.NewHTTPError(http.StatusUnauthorized, "bad signature")
	}

	var p webhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "malformed JSON")
	}
	for _, entry := range p.Entry {
		for _, change := range entry.Changes {
			for _, m := range change.Value.Messages {
				if !h.seen.add(m.ID) {
					continue
				}
				text := m.Text.Body
				if m.Type != "text" {
					text = "help"
				}
				// Detached from the request so a slow reply doesn't hold
				// Meta's connection open.
				go h.handle(context.WithoutCancel(c.Request().Context()), m.From, text)
			}
		}
	}
	return c.NoContent(http.StatusOK)
}

func (h whatsappHandler) handle(ctx context.Context, from, text string) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	reply, err := h.bot.Handle(ctx, from, text)
	if err != nil {
		log.Printf("whatsapp: handling message from %s: %v", from, err)
		reply = "Sorry, something went wrong on our side. Please try again in a minute."
	}
	if err := h.cfg.Sender.SendText(ctx, from, reply); err != nil {
		log.Printf("whatsapp: %v", err)
	}
}

type simulateRequest struct {
	From string `json:"from"`
	Text string `json:"text"`
}

// simulate runs the bot synchronously and returns its reply, for local
// testing without a Meta app. Only registered when DevTools is on.
func (h whatsappHandler) simulate(c echo.Context) error {
	var req simulateRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "malformed JSON")
	}
	reply, err := h.bot.Handle(c.Request().Context(), req.From, req.Text)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"reply": reply})
}

func validSignature(secret string, body []byte, header string) bool {
	sig, ok := strings.CutPrefix(header, "sha256=")
	if !ok {
		return false
	}
	got, err := hex.DecodeString(sig)
	if err != nil {
		return false
	}
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(body)
	return hmac.Equal(got, m.Sum(nil))
}

// recentIDs remembers the last n message ids.
type recentIDs struct {
	mu    sync.Mutex
	set   map[string]struct{}
	order []string
	n     int
}

func newRecentIDs(n int) *recentIDs {
	return &recentIDs{set: make(map[string]struct{}, n), n: n}
}

// add returns false if id was already seen.
func (r *recentIDs) add(id string) bool {
	if id == "" {
		return true
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.set[id]; ok {
		return false
	}
	r.set[id] = struct{}{}
	r.order = append(r.order, id)
	if len(r.order) > r.n {
		delete(r.set, r.order[0])
		r.order = r.order[1:]
	}
	return true
}
