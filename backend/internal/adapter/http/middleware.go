package http

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/akr24/sloan-marketplace/backend/internal/token"
)

const userIDKey = "auth.user_id"

// requireAuth parses the Bearer session token and stores the user id on the
// context. Handlers read it with currentUserID.
func requireAuth(tokens *token.Issuer) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get(echo.HeaderAuthorization)
			raw, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || raw == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing bearer token")
			}
			id, err := tokens.ParseSession(raw)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}
			c.Set(userIDKey, id)
			return next(c)
		}
	}
}

func currentUserID(c echo.Context) int64 {
	id, _ := c.Get(userIDKey).(int64)
	return id
}
