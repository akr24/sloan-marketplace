// Package http is the Echo-based HTTP adapter. It translates requests into
// use-case calls and use-case errors into status codes; no business logic
// lives here.
package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"github.com/akr24/sloan-marketplace/backend/internal/adapter/whatsapp"
	"github.com/akr24/sloan-marketplace/backend/internal/domain"
	"github.com/akr24/sloan-marketplace/backend/internal/token"
	"github.com/akr24/sloan-marketplace/backend/internal/usecase"
)

// Pinger is the slice of the database pool the health check needs.
type Pinger interface {
	Ping(ctx context.Context) error
}

// WhatsAppConfig holds the webhook secrets from the Meta app dashboard.
type WhatsAppConfig struct {
	// VerifyToken is the string you type into the dashboard when
	// subscribing the webhook; Meta echoes it back on the GET handshake.
	VerifyToken string
	// AppSecret signs every webhook POST (X-Hub-Signature-256). When empty,
	// signatures aren't checked, which is only acceptable locally.
	AppSecret string
	Sender    whatsapp.Sender
}

// Deps are the collaborators the router wires handlers to.
type Deps struct {
	DB         Pinger
	Auth       *usecase.Auth
	Listings   *usecase.Listings
	Bot        *usecase.Bot
	Tokens     *token.Issuer
	WhatsApp   WhatsAppConfig
	CORSOrigin string
	// DevTools enables POST /api/dev/whatsapp, a bot simulator that skips
	// Meta entirely. Never enable it in production.
	DevTools bool
}

// New builds the Echo instance with middleware and routes registered.
func New(d Deps) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = errorHandler(e)

	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())
	e.Use(middleware.BodyLimit("64K"))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{d.CORSOrigin},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.PATCH, echo.DELETE, echo.OPTIONS},
		AllowHeaders: []string{echo.HeaderContentType, echo.HeaderAuthorization},
	}))

	e.GET("/healthz", healthHandler(d.DB))

	api := e.Group("/api")

	// Code requests send email, so they get a per-IP limit on top of the
	// per-address resend interval in the use case. Generous, because a whole
	// classroom can share one campus Wi-Fi address. The burst must be set
	// explicitly: Echo derives it from the rate, which rounds to 0 here.
	authLimit := middleware.RateLimiter(middleware.NewRateLimiterMemoryStoreWithConfig(
		middleware.RateLimiterMemoryStoreConfig{Rate: rate.Limit(30.0 / 60), Burst: 20, ExpiresIn: 10 * time.Minute},
	))
	auth := api.Group("/auth")
	h := authHandler{d.Auth}
	auth.POST("/request-code", h.requestCode, authLimit)
	auth.POST("/verify-code", h.verifyCode, authLimit)
	auth.POST("/complete-signup", h.completeSignup)
	auth.GET("/me", h.me, requireAuth(d.Tokens))

	// Listings are members-only: they carry sellers' WhatsApp numbers.
	listings := api.Group("/listings", requireAuth(d.Tokens))
	l := listingsHandler{d.Listings}
	listings.GET("", l.search)
	listings.POST("", l.create)
	listings.GET("/mine", l.mine)
	listings.GET("/:id", l.get)
	listings.PATCH("/:id/status", l.setStatus)

	wa := whatsappHandler{bot: d.Bot, cfg: d.WhatsApp, seen: newRecentIDs(1000)}
	e.GET("/webhooks/whatsapp", wa.verify)
	e.POST("/webhooks/whatsapp", wa.receive)
	if d.DevTools {
		api.POST("/dev/whatsapp", wa.simulate)
	}

	return e
}

func healthHandler(db Pinger) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := db.Ping(c.Request().Context()); err != nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "database unreachable")
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	}
}

// errorHandler maps domain and use-case errors that handlers return as-is
// onto status codes, then defers to Echo's default handler.
func errorHandler(e *echo.Echo) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		var he *echo.HTTPError
		if !errors.As(err, &he) {
			switch {
			case errors.Is(err, usecase.ErrInvalidInput):
				err = echo.NewHTTPError(http.StatusBadRequest, trimInvalid(err))
			case errors.Is(err, usecase.ErrNotMITEmail):
				err = echo.NewHTTPError(http.StatusBadRequest, err.Error())
			case errors.Is(err, usecase.ErrCodeInvalid):
				err = echo.NewHTTPError(http.StatusUnauthorized, err.Error())
			case errors.Is(err, usecase.ErrTooSoon):
				err = echo.NewHTTPError(http.StatusTooManyRequests, err.Error())
			case errors.Is(err, domain.ErrEmailTaken), errors.Is(err, domain.ErrPhoneTaken):
				err = echo.NewHTTPError(http.StatusConflict, err.Error())
			case errors.Is(err, domain.ErrNotFound):
				err = echo.NewHTTPError(http.StatusNotFound, "not found")
			case errors.Is(err, domain.ErrUnauthenticated):
				err = echo.NewHTTPError(http.StatusUnauthorized, "account no longer exists")
			}
		}
		e.DefaultHTTPErrorHandler(err, c)
	}
}

// trimInvalid drops the "invalid input: " prefix so forms show just the reason.
func trimInvalid(err error) string {
	msg := err.Error()
	prefix := usecase.ErrInvalidInput.Error() + ": "
	if len(msg) > len(prefix) && msg[:len(prefix)] == prefix {
		return msg[len(prefix):]
	}
	return msg
}
