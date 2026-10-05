package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akr24/sloan-marketplace/backend/internal/domain"
)

type EmailCodeRepository struct {
	pool *pgxpool.Pool
}

func NewEmailCodeRepository(pool *pgxpool.Pool) *EmailCodeRepository {
	return &EmailCodeRepository{pool: pool}
}

func (r *EmailCodeRepository) Put(ctx context.Context, c *domain.EmailCode) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO email_codes (email, code_hash, attempts, expires_at, created_at)
		VALUES ($1, $2, 0, $3, $4)
		ON CONFLICT (email) DO UPDATE
		SET code_hash = EXCLUDED.code_hash, attempts = 0,
		    expires_at = EXCLUDED.expires_at, created_at = EXCLUDED.created_at`,
		c.Email, c.CodeHash, c.ExpiresAt, c.CreatedAt)
	return err
}

func (r *EmailCodeRepository) Get(ctx context.Context, email string) (*domain.EmailCode, error) {
	var c domain.EmailCode
	err := r.pool.QueryRow(ctx, `
		SELECT email, code_hash, attempts, expires_at, created_at
		FROM email_codes WHERE email = $1`, email).
		Scan(&c.Email, &c.CodeHash, &c.Attempts, &c.ExpiresAt, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *EmailCodeRepository) IncrementAttempts(ctx context.Context, email string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		UPDATE email_codes SET attempts = attempts + 1
		WHERE email = $1 RETURNING attempts`, email).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, domain.ErrNotFound
	}
	return n, err
}

func (r *EmailCodeRepository) Delete(ctx context.Context, email string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM email_codes WHERE email = $1`, email)
	return err
}
