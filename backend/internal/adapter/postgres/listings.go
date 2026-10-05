package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/akr24/sloan-marketplace/backend/internal/domain"
)

type ListingRepository struct {
	pool *pgxpool.Pool
}

func NewListingRepository(pool *pgxpool.Pool) *ListingRepository {
	return &ListingRepository{pool: pool}
}

const listingSelect = `
	SELECT l.id, l.seller_id, u.name, u.whatsapp_phone, l.event_name, l.event_date,
	       l.venue, l.quantity, l.price_cents, l.notes, l.source, l.status,
	       l.created_at, l.updated_at
	FROM listings l JOIN users u ON u.id = l.seller_id`

var sortSQL = map[domain.ListingSort]string{
	domain.SortSoonest:   "l.event_date ASC, l.created_at DESC",
	domain.SortNewest:    "l.created_at DESC",
	domain.SortPriceLow:  "l.price_cents ASC, l.event_date ASC",
	domain.SortPriceHigh: "l.price_cents DESC, l.event_date ASC",
}

func (r *ListingRepository) Create(ctx context.Context, l *domain.Listing) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO listings (seller_id, event_name, event_date, venue, quantity, price_cents, notes, source, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`,
		l.SellerID, l.EventName, l.EventDate, l.Venue, l.Quantity, l.PriceCents, l.Notes, l.Source, l.Status,
	).Scan(&l.ID, &l.CreatedAt, &l.UpdatedAt)
}

func (r *ListingRepository) FindByID(ctx context.Context, id int64) (*domain.Listing, error) {
	rows, err := r.pool.Query(ctx, listingSelect+` WHERE l.id = $1`, id)
	if err != nil {
		return nil, err
	}
	list, err := scanListings(rows)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, domain.ErrNotFound
	}
	return &list[0], nil
}

// Search builds the WHERE clause from whichever filter fields are set. Every
// value goes in as a bind parameter; only the fixed ORDER BY is spliced in.
func (r *ListingRepository) Search(ctx context.Context, f domain.ListingFilter) ([]domain.Listing, error) {
	var where []string
	var args []any
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}

	if !f.IncludeAll {
		add("l.status = $%d", domain.StatusActive)
	}
	if f.SellerID != 0 {
		add("l.seller_id = $%d", f.SellerID)
	}
	if f.Query != "" {
		pattern := "%" + escapeLike(f.Query) + "%"
		add("(l.event_name ILIKE $%[1]d OR l.venue ILIKE $%[1]d OR l.notes ILIKE $%[1]d)", pattern)
	}
	if f.DateFrom != nil {
		add("l.event_date >= $%d", *f.DateFrom)
	}
	if f.DateTo != nil {
		add("l.event_date <= $%d", *f.DateTo)
	}
	if f.MinPriceCents != nil {
		add("l.price_cents >= $%d", *f.MinPriceCents)
	}
	if f.MaxPriceCents != nil {
		add("l.price_cents <= $%d", *f.MaxPriceCents)
	}
	if f.MinQuantity > 0 {
		add("l.quantity >= $%d", f.MinQuantity)
	}

	sql := listingSelect
	if len(where) > 0 {
		sql += " WHERE " + strings.Join(where, " AND ")
	}
	order, ok := sortSQL[f.Sort]
	if !ok {
		order = sortSQL[domain.SortSoonest]
	}
	args = append(args, f.Limit)
	sql += fmt.Sprintf(" ORDER BY %s LIMIT $%d", order, len(args))

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return scanListings(rows)
}

func (r *ListingRepository) SetStatus(ctx context.Context, id int64, s domain.Status) error {
	tag, err := r.pool.Exec(ctx, `UPDATE listings SET status = $2, updated_at = now() WHERE id = $1`, id, s)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanListings(rows pgx.Rows) ([]domain.Listing, error) {
	defer rows.Close()
	list := []domain.Listing{}
	for rows.Next() {
		var l domain.Listing
		if err := rows.Scan(&l.ID, &l.SellerID, &l.SellerName, &l.SellerPhone, &l.EventName, &l.EventDate,
			&l.Venue, &l.Quantity, &l.PriceCents, &l.Notes, &l.Source, &l.Status,
			&l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

// escapeLike stops user input acting as LIKE wildcards.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

