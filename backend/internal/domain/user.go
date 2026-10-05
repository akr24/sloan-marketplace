package domain

import (
	"context"
	"errors"
	"time"
)

// User is a verified member of the MIT community. Accounts are only created
// after the email's one-time code has been confirmed, so every row is verified.
type User struct {
	ID            int64     `json:"id"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`
	WhatsAppPhone string    `json:"whatsapp_phone"`
	CreatedAt     time.Time `json:"created_at"`
}

var (
	ErrEmailTaken      = errors.New("email already registered")
	ErrPhoneTaken      = errors.New("WhatsApp number already linked to another account")
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrNotFound        = errors.New("not found")
	ErrForbidden       = errors.New("forbidden")
)

// UserRepository is the storage boundary for accounts.
type UserRepository interface {
	Create(ctx context.Context, u *User) error
	FindByID(ctx context.Context, id int64) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByPhone(ctx context.Context, phone string) (*User, error)
}
