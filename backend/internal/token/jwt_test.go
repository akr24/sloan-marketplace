package token

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const secret = "0123456789abcdef0123456789abcdef"

func newIssuer(t *testing.T) *Issuer {
	t.Helper()
	iss, err := New(secret, time.Hour, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	return iss
}

func TestSessionRoundTrip(t *testing.T) {
	iss := newIssuer(t)
	raw, err := iss.IssueSession(42)
	if err != nil {
		t.Fatal(err)
	}
	id, err := iss.ParseSession(raw)
	if err != nil || id != 42 {
		t.Fatalf("got %d, %v", id, err)
	}
}

func TestSignupRoundTrip(t *testing.T) {
	iss := newIssuer(t)
	raw, _ := iss.IssueSignup("tim@mit.edu")
	email, err := iss.ParseSignup(raw)
	if err != nil || email != "tim@mit.edu" {
		t.Fatalf("got %q, %v", email, err)
	}
}

func TestAudiencesAreNotInterchangeable(t *testing.T) {
	iss := newIssuer(t)
	session, _ := iss.IssueSession(1)
	signup, _ := iss.IssueSignup("tim@mit.edu")
	if _, err := iss.ParseSignup(session); !errors.Is(err, ErrInvalid) {
		t.Errorf("session token accepted as signup: %v", err)
	}
	if _, err := iss.ParseSession(signup); !errors.Is(err, ErrInvalid) {
		t.Errorf("signup token accepted as session: %v", err)
	}
}

func TestRejectsShortSecret(t *testing.T) {
	if _, err := New("short", time.Hour, time.Hour); err == nil {
		t.Fatal("want error for short secret")
	}
}

func TestRejectsExpired(t *testing.T) {
	iss := newIssuer(t)
	start := time.Now()
	iss.now = func() time.Time { return start }
	raw, _ := iss.IssueSession(1)

	iss.now = func() time.Time { return start.Add(2 * time.Hour) }
	if _, err := iss.ParseSession(raw); !errors.Is(err, ErrInvalid) {
		t.Fatalf("want ErrInvalid, got %v", err)
	}
}

func TestRejectsTamperedAndForeign(t *testing.T) {
	iss := newIssuer(t)
	other, _ := New(strings.Repeat("x", 32), time.Hour, time.Hour)
	raw, _ := other.IssueSession(1)

	for _, bad := range []string{"", "not.a.jwt", raw, raw[:len(raw)-2] + "zz"} {
		if _, err := iss.ParseSession(bad); err == nil {
			t.Errorf("ParseSession(%q) should fail", bad)
		}
	}
}
