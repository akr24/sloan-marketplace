package domain

import (
	"context"
	"time"
)

// Source records which channel a listing was posted through.
type Source string

const (
	SourceWeb      Source = "web"
	SourceWhatsApp Source = "whatsapp"
)

// Status is a listing's lifecycle state.
type Status string

const (
	StatusActive    Status = "active"
	StatusSold      Status = "sold"
	StatusWithdrawn Status = "withdrawn"
)

// Valid reports whether s is one of the known statuses.
func (s Status) Valid() bool {
	return s == StatusActive || s == StatusSold || s == StatusWithdrawn
}

// Listing is an offer to sell event tickets. Seller fields are joined in on
// read so the client can render the WhatsApp contact link without a second call.
type Listing struct {
	ID          int64     `json:"id"`
	SellerID    int64     `json:"seller_id"`
	SellerName  string    `json:"seller_name"`
	SellerPhone string    `json:"seller_whatsapp"`
	EventName   string    `json:"event_name"`
	EventDate   time.Time `json:"event_date"`
	Venue       string    `json:"venue"`
	Quantity    int       `json:"quantity"`
	PriceCents  int       `json:"price_cents"`
	Notes       string    `json:"notes"`
	Source      Source    `json:"source"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ListingSort picks the order of search results.
type ListingSort string

const (
	SortSoonest   ListingSort = "soonest"
	SortNewest    ListingSort = "newest"
	SortPriceLow  ListingSort = "price_low"
	SortPriceHigh ListingSort = "price_high"
)

// ListingFilter narrows a search. Zero values mean "no constraint".
type ListingFilter struct {
	Query         string // matched against event name, venue and notes
	DateFrom      *time.Time
	DateTo        *time.Time
	MinPriceCents *int
	MaxPriceCents *int
	MinQuantity   int
	SellerID      int64
	// Status defaults to active; IncludeAll returns every status (used for
	// a seller's own listings).
	IncludeAll bool
	Sort       ListingSort
	Limit      int
}

// ListingRepository is the storage boundary for listings.
type ListingRepository interface {
	Create(ctx context.Context, l *Listing) error
	FindByID(ctx context.Context, id int64) (*Listing, error)
	Search(ctx context.Context, f ListingFilter) ([]Listing, error)
	// Update replaces the seller-editable fields of the listing with l.ID.
	Update(ctx context.Context, l *Listing) error
	SetStatus(ctx context.Context, id int64, s Status) error
}
