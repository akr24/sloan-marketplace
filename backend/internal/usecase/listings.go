package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/akr24/sloan-marketplace/backend/internal/domain"
)

const (
	MaxEventNameLen  = 120
	MaxVenueLen      = 120
	MaxNotesLen      = 1000
	MaxQuantity      = 20
	MaxPriceCents    = 1_000_000 // $10,000 per ticket
	DefaultPageLimit = 50
	MaxPageLimit     = 100
)

// ListingInput is what a seller provides, from either the web or the bot.
type ListingInput struct {
	EventName  string
	EventDate  time.Time
	Venue      string
	Quantity   int
	PriceCents int
	Notes      string
}

// Listings holds the ticket listing use cases.
type Listings struct {
	repo domain.ListingRepository
	now  func() time.Time
}

func NewListings(repo domain.ListingRepository) *Listings {
	return &Listings{repo: repo, now: time.Now}
}

// Create validates and stores a new active listing for the seller.
func (l *Listings) Create(ctx context.Context, sellerID int64, src domain.Source, in ListingInput) (*domain.Listing, error) {
	in, err := l.validate(in)
	if err != nil {
		return nil, err
	}

	listing := &domain.Listing{
		SellerID:   sellerID,
		EventName:  in.EventName,
		EventDate:  in.EventDate,
		Venue:      in.Venue,
		Quantity:   in.Quantity,
		PriceCents: in.PriceCents,
		Notes:      in.Notes,
		Source:     src,
		Status:     domain.StatusActive,
	}
	if err := l.repo.Create(ctx, listing); err != nil {
		return nil, err
	}
	// Re-read so the seller fields are joined in.
	return l.repo.FindByID(ctx, listing.ID)
}

// validate trims and checks the seller-provided fields.
func (l *Listings) validate(in ListingInput) (ListingInput, error) {
	in.EventName = strings.TrimSpace(in.EventName)
	in.Venue = strings.TrimSpace(in.Venue)
	in.Notes = strings.TrimSpace(in.Notes)

	switch {
	case in.EventName == "" || len(in.EventName) > MaxEventNameLen:
		return in, fmt.Errorf("%w: event name is required and at most %d characters", ErrInvalidInput, MaxEventNameLen)
	case in.EventDate.IsZero():
		return in, fmt.Errorf("%w: event date is required", ErrInvalidInput)
	case in.EventDate.Before(l.today()):
		return in, fmt.Errorf("%w: event date is in the past", ErrInvalidInput)
	case len(in.Venue) > MaxVenueLen:
		return in, fmt.Errorf("%w: venue is at most %d characters", ErrInvalidInput, MaxVenueLen)
	case in.Quantity < 1 || in.Quantity > MaxQuantity:
		return in, fmt.Errorf("%w: quantity must be between 1 and %d", ErrInvalidInput, MaxQuantity)
	case in.PriceCents < 0 || in.PriceCents > MaxPriceCents:
		return in, fmt.Errorf("%w: price must be between $0 and $%d", ErrInvalidInput, MaxPriceCents/100)
	case len(in.Notes) > MaxNotesLen:
		return in, fmt.Errorf("%w: notes are at most %d characters", ErrInvalidInput, MaxNotesLen)
	}
	return in, nil
}

// Update replaces a seller's own listing's details. Status is left alone;
// use SetStatus for that.
func (l *Listings) Update(ctx context.Context, sellerID, listingID int64, in ListingInput) (*domain.Listing, error) {
	in, err := l.validate(in)
	if err != nil {
		return nil, err
	}
	listing, err := l.repo.FindByID(ctx, listingID)
	if err != nil {
		return nil, err
	}
	if listing.SellerID != sellerID {
		return nil, errors.Join(domain.ErrNotFound, domain.ErrForbidden)
	}
	listing.EventName = in.EventName
	listing.EventDate = in.EventDate
	listing.Venue = in.Venue
	listing.Quantity = in.Quantity
	listing.PriceCents = in.PriceCents
	listing.Notes = in.Notes
	if err := l.repo.Update(ctx, listing); err != nil {
		return nil, err
	}
	return l.repo.FindByID(ctx, listingID)
}

// InputOf returns the editable fields of an existing listing, so a caller
// can change one field and pass the rest through unchanged.
func InputOf(l *domain.Listing) ListingInput {
	return ListingInput{
		EventName:  l.EventName,
		EventDate:  l.EventDate,
		Venue:      l.Venue,
		Quantity:   l.Quantity,
		PriceCents: l.PriceCents,
		Notes:      l.Notes,
	}
}

// Search returns active listings for upcoming events matching the filter.
func (l *Listings) Search(ctx context.Context, f domain.ListingFilter) ([]domain.Listing, error) {
	f.Query = strings.TrimSpace(f.Query)
	f.IncludeAll = false
	f.SellerID = 0
	if f.DateFrom == nil || f.DateFrom.Before(l.today()) {
		today := l.today()
		f.DateFrom = &today
	}
	switch f.Sort {
	case domain.SortSoonest, domain.SortNewest, domain.SortPriceLow, domain.SortPriceHigh:
	case "":
		f.Sort = domain.SortSoonest
	default:
		return nil, fmt.Errorf("%w: unknown sort %q", ErrInvalidInput, f.Sort)
	}
	if f.Limit <= 0 {
		f.Limit = DefaultPageLimit
	}
	f.Limit = min(f.Limit, MaxPageLimit)
	return l.repo.Search(ctx, f)
}

// Get returns one listing by id.
func (l *Listings) Get(ctx context.Context, id int64) (*domain.Listing, error) {
	return l.repo.FindByID(ctx, id)
}

// Mine returns every listing the seller has posted, any status, newest first.
func (l *Listings) Mine(ctx context.Context, sellerID int64) ([]domain.Listing, error) {
	return l.repo.Search(ctx, domain.ListingFilter{
		SellerID:   sellerID,
		IncludeAll: true,
		Sort:       domain.SortNewest,
		Limit:      MaxPageLimit,
	})
}

// SetStatus lets a seller mark their own listing sold, withdrawn or active again.
func (l *Listings) SetStatus(ctx context.Context, sellerID, listingID int64, s domain.Status) (*domain.Listing, error) {
	if !s.Valid() {
		return nil, fmt.Errorf("%w: status must be active, sold or withdrawn", ErrInvalidInput)
	}
	listing, err := l.repo.FindByID(ctx, listingID)
	if err != nil {
		return nil, err
	}
	if listing.SellerID != sellerID {
		// Don't reveal that someone else's listing exists by id.
		return nil, errors.Join(domain.ErrNotFound, domain.ErrForbidden)
	}
	if err := l.repo.SetStatus(ctx, listingID, s); err != nil {
		return nil, err
	}
	listing.Status = s
	return listing, nil
}

// today is the current Cambridge calendar date as midnight UTC, the same
// shape plain event dates are stored and parsed in.
func (l *Listings) today() time.Time {
	return campusDate(l.now())
}
