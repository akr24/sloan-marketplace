package domain

import (
	"context"
	"time"
)

// EmailCode is a pending one-time sign-in code. Only the hash is stored.
type EmailCode struct {
	Email     string
	CodeHash  string
	Attempts  int
	ExpiresAt time.Time
	CreatedAt time.Time
}

// EmailCodeRepository stores at most one live code per email.
type EmailCodeRepository interface {
	// Put replaces any existing code for the email and resets attempts.
	Put(ctx context.Context, c *EmailCode) error
	Get(ctx context.Context, email string) (*EmailCode, error)
	// IncrementAttempts records a wrong guess and returns the new count.
	IncrementAttempts(ctx context.Context, email string) (int, error)
	Delete(ctx context.Context, email string) error
}

// Mailer delivers a one-time code. Implementations range from SMTP to
// printing on stdout for local development.
type Mailer interface {
	SendCode(ctx context.Context, to, code string) error
}
