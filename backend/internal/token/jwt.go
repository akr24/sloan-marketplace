// Package token issues and verifies the JWTs the API hands out.
//
// There are two kinds, told apart by audience so one can never stand in for
// the other:
//
//   - session tokens identify a signed-in user and carry only the user id;
//   - signup tokens prove an email was just verified by one-time code and
//     carry only that email, so a new user can finish creating their account.
//
// Both are HS256-signed with one server secret.
package token

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	issuer          = "sloan-marketplace"
	audienceSession = "session"
	audienceSignup  = "signup"
)

var ErrInvalid = errors.New("token: invalid")

// Issuer signs and parses tokens with one secret.
type Issuer struct {
	secret     []byte
	sessionTTL time.Duration
	signupTTL  time.Duration
	now        func() time.Time
}

// New requires a secret of at least 32 bytes so HS256 isn't brute-forceable.
func New(secret string, sessionTTL, signupTTL time.Duration) (*Issuer, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("token: JWT_SECRET must be at least 32 bytes, got %d", len(secret))
	}
	return &Issuer{secret: []byte(secret), sessionTTL: sessionTTL, signupTTL: signupTTL, now: time.Now}, nil
}

// IssueSession returns a signed session token for the user id.
func (i *Issuer) IssueSession(userID int64) (string, error) {
	return i.sign(audienceSession, strconv.FormatInt(userID, 10), i.sessionTTL)
}

// ParseSession verifies a session token and returns its user id.
func (i *Issuer) ParseSession(raw string) (int64, error) {
	sub, err := i.parse(raw, audienceSession)
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseInt(sub, 10, 64)
	if err != nil {
		return 0, ErrInvalid
	}
	return id, nil
}

// IssueSignup returns a short-lived token proving email was verified.
func (i *Issuer) IssueSignup(email string) (string, error) {
	return i.sign(audienceSignup, email, i.signupTTL)
}

// ParseSignup verifies a signup token and returns its email.
func (i *Issuer) ParseSignup(raw string) (string, error) {
	return i.parse(raw, audienceSignup)
}

func (i *Issuer) sign(aud, sub string, ttl time.Duration) (string, error) {
	now := i.now()
	claims := jwt.RegisteredClaims{
		Issuer:    issuer,
		Audience:  jwt.ClaimStrings{aud},
		Subject:   sub,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
}

func (i *Issuer) parse(raw, aud string) (string, error) {
	var c jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(raw, &c, func(t *jwt.Token) (any, error) {
		return i.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithAudience(aud),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(i.now),
	)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if c.Subject == "" {
		return "", ErrInvalid
	}
	return c.Subject, nil
}
