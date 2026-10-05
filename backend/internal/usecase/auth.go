package usecase

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
	"time"

	"github.com/akr24/sloan-marketplace/backend/internal/domain"
)

const (
	// CodeTTL is how long an emailed code stays valid.
	CodeTTL = 10 * time.Minute
	// CodeResendInterval stops an address being flooded with codes.
	CodeResendInterval = 30 * time.Second
	// MaxCodeAttempts locks a code after this many wrong guesses; with six
	// digits that leaves a 1-in-200,000 chance of guessing.
	MaxCodeAttempts = 5
	// MaxNameLen keeps display names to something a UI can render.
	MaxNameLen = 100
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotMITEmail  = errors.New("use your MIT email address (ending in mit.edu)")
	ErrCodeInvalid  = errors.New("that code is wrong or has expired; request a new one")
	ErrTooSoon      = errors.New("a code was just sent; wait a moment before asking again")
)

// TokenIssuer is the slice of the token package the auth use cases need, so
// this layer doesn't depend on a JWT library.
type TokenIssuer interface {
	IssueSession(userID int64) (string, error)
	IssueSignup(email string) (string, error)
	ParseSignup(raw string) (string, error)
}

// Session is what a successful sign-in returns.
type Session struct {
	User  *domain.User `json:"user"`
	Token string       `json:"token"`
}

// VerifyResult is either a session for an existing account, or a signup
// token for a verified email that has no account yet.
type VerifyResult struct {
	Session     *Session `json:"session,omitempty"`
	SignupToken string   `json:"signup_token,omitempty"`
	Email       string   `json:"email"`
}

// Auth holds the passwordless account use cases: request a code, verify it,
// and (first time only) finish signing up with a name and WhatsApp number.
type Auth struct {
	users   domain.UserRepository
	codes   domain.EmailCodeRepository
	mailer  domain.Mailer
	tokens  TokenIssuer
	codeKey []byte
	now     func() time.Time
}

// NewAuth needs codeKey to HMAC stored codes, so a leaked table can't be
// brute-forced offline in a million tries.
func NewAuth(users domain.UserRepository, codes domain.EmailCodeRepository, mailer domain.Mailer, tokens TokenIssuer, codeKey string) *Auth {
	return &Auth{users: users, codes: codes, mailer: mailer, tokens: tokens, codeKey: []byte(codeKey), now: time.Now}
}

// RequestCode emails a fresh six-digit code to an MIT address. It works the
// same whether or not the account exists, so sign-up and sign-in share one flow.
func (a *Auth) RequestCode(ctx context.Context, email string) error {
	email, err := NormaliseMITEmail(email)
	if err != nil {
		return err
	}

	prev, err := a.codes.Get(ctx, email)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	if prev != nil && a.now().Sub(prev.CreatedAt) < CodeResendInterval {
		return ErrTooSoon
	}

	code, err := randomCode()
	if err != nil {
		return err
	}
	now := a.now()
	if err := a.codes.Put(ctx, &domain.EmailCode{
		Email:     email,
		CodeHash:  a.hash(email, code),
		ExpiresAt: now.Add(CodeTTL),
		CreatedAt: now,
	}); err != nil {
		return err
	}
	return a.mailer.SendCode(ctx, email, code)
}

// VerifyCode checks the code. A known email gets a session; a new one gets a
// signup token to pass to CompleteSignup. Codes are single use.
func (a *Auth) VerifyCode(ctx context.Context, email, code string) (*VerifyResult, error) {
	email, err := NormaliseMITEmail(email)
	if err != nil {
		return nil, err
	}
	code = strings.TrimSpace(code)

	stored, err := a.codes.Get(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, ErrCodeInvalid
	}
	if err != nil {
		return nil, err
	}
	if a.now().After(stored.ExpiresAt) || stored.Attempts >= MaxCodeAttempts {
		return nil, ErrCodeInvalid
	}
	if !hmac.Equal([]byte(stored.CodeHash), []byte(a.hash(email, code))) {
		if _, err := a.codes.IncrementAttempts(ctx, email); err != nil {
			return nil, err
		}
		return nil, ErrCodeInvalid
	}
	if err := a.codes.Delete(ctx, email); err != nil {
		return nil, err
	}

	u, err := a.users.FindByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		tok, err := a.tokens.IssueSignup(email)
		if err != nil {
			return nil, err
		}
		return &VerifyResult{SignupToken: tok, Email: email}, nil
	}
	if err != nil {
		return nil, err
	}
	s, err := a.session(u)
	if err != nil {
		return nil, err
	}
	return &VerifyResult{Session: s, Email: email}, nil
}

// SignupInput finishes creating an account for a verified email.
type SignupInput struct {
	SignupToken   string
	Name          string
	WhatsAppPhone string
}

// CompleteSignup creates the account behind a signup token and signs in.
func (a *Auth) CompleteSignup(ctx context.Context, in SignupInput) (*Session, error) {
	email, err := a.tokens.ParseSignup(in.SignupToken)
	if err != nil {
		return nil, fmt.Errorf("%w: verification expired, start again", ErrInvalidInput)
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || len(name) > MaxNameLen {
		return nil, fmt.Errorf("%w: name is required and at most %d characters", ErrInvalidInput, MaxNameLen)
	}
	phone, err := NormalisePhone(in.WhatsAppPhone)
	if err != nil {
		return nil, err
	}

	u := &domain.User{Email: email, Name: name, WhatsAppPhone: phone}
	if err := a.users.Create(ctx, u); err != nil {
		return nil, err
	}
	return a.session(u)
}

// Current loads the account behind a verified session token's user id. A
// deleted account yields ErrUnauthenticated so stale tokens stop working.
func (a *Auth) Current(ctx context.Context, userID int64) (*domain.User, error) {
	u, err := a.users.FindByID(ctx, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, domain.ErrUnauthenticated
	}
	return u, err
}

func (a *Auth) session(u *domain.User) (*Session, error) {
	tok, err := a.tokens.IssueSession(u.ID)
	if err != nil {
		return nil, err
	}
	return &Session{User: u, Token: tok}, nil
}

func (a *Auth) hash(email, code string) string {
	m := hmac.New(sha256.New, a.codeKey)
	m.Write([]byte(email + "\x00" + code))
	return hex.EncodeToString(m.Sum(nil))
}

func randomCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// NormaliseMITEmail lower-cases and validates an address, accepting mit.edu
// and its subdomains (e.g. sloan.mit.edu, alum.mit.edu).
func NormaliseMITEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", fmt.Errorf("%w: invalid email", ErrInvalidInput)
	}
	_, domainPart, _ := strings.Cut(email, "@")
	if domainPart != "mit.edu" && !strings.HasSuffix(domainPart, ".mit.edu") {
		return "", ErrNotMITEmail
	}
	return email, nil
}

// NormalisePhone reduces a phone number to E.164 digits without the plus,
// which is the form wa.me links and the WhatsApp API both use. A bare
// 10-digit number is assumed to be US/Canada.
func NormalisePhone(raw string) (string, error) {
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	if len(digits) == 10 && !strings.HasPrefix(strings.TrimSpace(raw), "+") {
		digits = "1" + digits
	}
	if len(digits) < 8 || len(digits) > 15 || digits[0] == '0' {
		return "", fmt.Errorf("%w: enter your WhatsApp number with country code, e.g. +1 617 555 0123", ErrInvalidInput)
	}
	return digits, nil
}
