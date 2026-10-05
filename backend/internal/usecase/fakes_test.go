package usecase

import (
	"context"
	"sort"
	"sync"

	"github.com/akr24/sloan-marketplace/backend/internal/domain"
)

// In-memory repositories so use cases can be tested without Postgres.

type fakeUsers struct {
	mu   sync.Mutex
	rows []*domain.User
}

func (f *fakeUsers) Create(_ context.Context, u *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.rows {
		if r.Email == u.Email {
			return domain.ErrEmailTaken
		}
		if r.WhatsAppPhone == u.WhatsAppPhone {
			return domain.ErrPhoneTaken
		}
	}
	u.ID = int64(len(f.rows) + 1)
	cp := *u
	f.rows = append(f.rows, &cp)
	return nil
}

func (f *fakeUsers) find(match func(*domain.User) bool) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.rows {
		if match(r) {
			cp := *r
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeUsers) FindByID(_ context.Context, id int64) (*domain.User, error) {
	return f.find(func(u *domain.User) bool { return u.ID == id })
}

func (f *fakeUsers) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	return f.find(func(u *domain.User) bool { return u.Email == email })
}

func (f *fakeUsers) FindByPhone(_ context.Context, phone string) (*domain.User, error) {
	return f.find(func(u *domain.User) bool { return u.WhatsAppPhone == phone })
}

type fakeCodes struct {
	rows map[string]*domain.EmailCode
}

func newFakeCodes() *fakeCodes { return &fakeCodes{rows: map[string]*domain.EmailCode{}} }

func (f *fakeCodes) Put(_ context.Context, c *domain.EmailCode) error {
	cp := *c
	cp.Attempts = 0
	f.rows[c.Email] = &cp
	return nil
}

func (f *fakeCodes) Get(_ context.Context, email string) (*domain.EmailCode, error) {
	c, ok := f.rows[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *c
	return &cp, nil
}

func (f *fakeCodes) IncrementAttempts(_ context.Context, email string) (int, error) {
	c, ok := f.rows[email]
	if !ok {
		return 0, domain.ErrNotFound
	}
	c.Attempts++
	return c.Attempts, nil
}

func (f *fakeCodes) Delete(_ context.Context, email string) error {
	delete(f.rows, email)
	return nil
}

// fakeMailer remembers the last code sent to each address.
type fakeMailer struct{ last map[string]string }

func (m *fakeMailer) SendCode(_ context.Context, to, code string) error {
	if m.last == nil {
		m.last = map[string]string{}
	}
	m.last[to] = code
	return nil
}

type fakeListings struct {
	users *fakeUsers
	rows  []*domain.Listing
}

func (f *fakeListings) Create(_ context.Context, l *domain.Listing) error {
	l.ID = int64(len(f.rows) + 1)
	cp := *l
	f.rows = append(f.rows, &cp)
	return nil
}

func (f *fakeListings) FindByID(ctx context.Context, id int64) (*domain.Listing, error) {
	for _, r := range f.rows {
		if r.ID == id {
			cp := *r
			if u, err := f.users.FindByID(ctx, r.SellerID); err == nil {
				cp.SellerName, cp.SellerPhone = u.Name, u.WhatsAppPhone
			}
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeListings) Search(_ context.Context, flt domain.ListingFilter) ([]domain.Listing, error) {
	out := []domain.Listing{}
	for _, r := range f.rows {
		if !flt.IncludeAll && r.Status != domain.StatusActive {
			continue
		}
		if flt.SellerID != 0 && r.SellerID != flt.SellerID {
			continue
		}
		if flt.DateFrom != nil && r.EventDate.Before(*flt.DateFrom) {
			continue
		}
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}

func (f *fakeListings) SetStatus(_ context.Context, id int64, s domain.Status) error {
	for _, r := range f.rows {
		if r.ID == id {
			r.Status = s
			return nil
		}
	}
	return domain.ErrNotFound
}

// fakeTokens issues readable tokens so tests can inspect them.
type fakeTokens struct{}

func (fakeTokens) IssueSession(id int64) (string, error) { return "session", nil }
func (fakeTokens) IssueSignup(email string) (string, error) {
	return "signup:" + email, nil
}
func (fakeTokens) ParseSignup(raw string) (string, error) {
	if len(raw) > 7 && raw[:7] == "signup:" {
		return raw[7:], nil
	}
	return "", ErrInvalidInput
}
