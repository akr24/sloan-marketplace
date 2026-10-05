// Command server is the Sloan Marketplace HTTP API.
//
// It wires the layers together: Postgres adapters implement the domain
// repositories, use cases hold the business rules, and the Echo adapter
// exposes them over HTTP (including the WhatsApp webhook). Nothing else in
// the tree imports cmd/server.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akr24/sloan-marketplace/backend/internal/adapter/email"
	httpadapter "github.com/akr24/sloan-marketplace/backend/internal/adapter/http"
	"github.com/akr24/sloan-marketplace/backend/internal/adapter/postgres"
	"github.com/akr24/sloan-marketplace/backend/internal/adapter/whatsapp"
	"github.com/akr24/sloan-marketplace/backend/internal/domain"
	"github.com/akr24/sloan-marketplace/backend/internal/token"
	"github.com/akr24/sloan-marketplace/backend/internal/usecase"
)

func main() {
	addr := ":" + envOr("PORT", "8080")
	devTools := os.Getenv("APP_ENV") == "dev"

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	secret := os.Getenv("JWT_SECRET")
	tokens, err := token.New(secret, 30*24*time.Hour, 30*time.Minute)
	if err != nil {
		log.Fatal(err)
	}

	users := postgres.NewUserRepository(pool)
	listings := usecase.NewListings(postgres.NewListingRepository(pool))
	appURL := envOr("APP_URL", "http://localhost:5174")

	e := httpadapter.New(httpadapter.Deps{
		DB:       pool,
		Auth:     usecase.NewAuth(users, postgres.NewEmailCodeRepository(pool), mailer(), tokens, secret),
		Listings: listings,
		Bot:      usecase.NewBot(users, listings, appURL),
		Tokens:   tokens,
		WhatsApp: httpadapter.WhatsAppConfig{
			VerifyToken: os.Getenv("WHATSAPP_VERIFY_TOKEN"),
			AppSecret:   os.Getenv("WHATSAPP_APP_SECRET"),
			Sender:      whatsappSender(),
		},
		CORSOrigin: envOr("CORS_ORIGIN", "*"),
		DevTools:   devTools,
	})
	e.Server.ReadHeaderTimeout = 5 * time.Second

	go func() {
		log.Printf("listening on %s", addr)
		if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

// mailer sends real email when SMTP is configured and logs codes otherwise.
func mailer() domain.Mailer {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		log.Print("email: SMTP_HOST not set, sign-in codes will be printed to this log")
		return email.LogMailer{}
	}
	return &email.SMTPMailer{
		Host:     host,
		Port:     envOr("SMTP_PORT", "587"),
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     envOr("SMTP_FROM", os.Getenv("SMTP_USERNAME")),
	}
}

// whatsappSender uses the Cloud API when credentials are set and logs
// replies otherwise.
func whatsappSender() whatsapp.Sender {
	id, tok := os.Getenv("WHATSAPP_PHONE_NUMBER_ID"), os.Getenv("WHATSAPP_ACCESS_TOKEN")
	if id == "" || tok == "" {
		log.Print("whatsapp: Cloud API credentials not set, bot replies will be printed to this log")
		return whatsapp.LogSender{}
	}
	return whatsapp.NewClient(id, tok)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
