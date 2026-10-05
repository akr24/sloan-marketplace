package http

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/akr24/sloan-marketplace/backend/internal/usecase"
)

type authHandler struct {
	auth *usecase.Auth
}

type requestCodeRequest struct {
	Email string `json:"email"`
}

type verifyCodeRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type completeSignupRequest struct {
	SignupToken   string `json:"signup_token"`
	Name          string `json:"name"`
	WhatsAppPhone string `json:"whatsapp_phone"`
}

func (h authHandler) requestCode(c echo.Context) error {
	var req requestCodeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "malformed JSON")
	}
	if err := h.auth.RequestCode(c.Request().Context(), req.Email); err != nil {
		return err
	}
	return c.JSON(http.StatusAccepted, map[string]string{"status": "sent"})
}

// verifyCode returns {session} for an existing account or {signup_token}
// for a new one; the client then calls complete-signup.
func (h authHandler) verifyCode(c echo.Context) error {
	var req verifyCodeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "malformed JSON")
	}
	res, err := h.auth.VerifyCode(c.Request().Context(), req.Email, req.Code)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}

func (h authHandler) completeSignup(c echo.Context) error {
	var req completeSignupRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "malformed JSON")
	}
	s, err := h.auth.CompleteSignup(c.Request().Context(), usecase.SignupInput{
		SignupToken:   req.SignupToken,
		Name:          req.Name,
		WhatsAppPhone: req.WhatsAppPhone,
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, s)
}

// me returns the signed-in user, looked up fresh from the database.
func (h authHandler) me(c echo.Context) error {
	u, err := h.auth.Current(c.Request().Context(), currentUserID(c))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, u)
}
