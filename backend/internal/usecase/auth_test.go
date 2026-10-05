package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/akr24/sloan-marketplace/backend/internal/domain"
)

func newTestAuth() (*Auth, *fakeMailer, *fakeUsers) {
	users := &fakeUsers{}
	m := &fakeMailer{}
	return NewAuth(users, newFakeCodes(), m, fakeTokens{}, "test-key"), m, users
}

func TestSignupThenSignin(t *testing.T) {
	ctx := context.Background()
	a, m, _ := newTestAuth()

	if err := a.RequestCode(ctx, "  Tim@Sloan.MIT.edu "); err != nil {
		t.Fatal(err)
	}
	res, err := a.VerifyCode(ctx, "tim@sloan.mit.edu", m.last["tim@sloan.mit.edu"])
	if err != nil {
		t.Fatal(err)
	}
	if res.Session != nil || res.SignupToken == "" {
		t.Fatalf("new email should get a signup token, got %+v", res)
	}

	s, err := a.CompleteSignup(ctx, SignupInput{SignupToken: res.SignupToken, Name: "Tim Beaver", WhatsAppPhone: "(617) 555-0123"})
	if err != nil {
		t.Fatal(err)
	}
	if s.User.WhatsAppPhone != "16175550123" {
		t.Errorf("phone normalised to %q", s.User.WhatsAppPhone)
	}

	// Second time round the same email signs straight in.
	a.now = func() time.Time { return time.Now().Add(time.Minute) }
	if err := a.RequestCode(ctx, "tim@sloan.mit.edu"); err != nil {
		t.Fatal(err)
	}
	res, err = a.VerifyCode(ctx, "tim@sloan.mit.edu", m.last["tim@sloan.mit.edu"])
	if err != nil || res.Session == nil {
		t.Fatalf("existing user should get a session: %+v, %v", res, err)
	}
}

func TestRejectsNonMITEmail(t *testing.T) {
	a, _, _ := newTestAuth()
	for _, email := range []string{"tim@gmail.com", "tim@notmit.edu", "tim@mit.edu.evil.com", "not-an-email"} {
		if err := a.RequestCode(context.Background(), email); err == nil {
			t.Errorf("%s should be rejected", email)
		}
	}
}

func TestCodeIsSingleUseAndLocksAfterAttempts(t *testing.T) {
	ctx := context.Background()
	a, m, _ := newTestAuth()
	_ = a.RequestCode(ctx, "a@mit.edu")
	code := m.last["a@mit.edu"]

	if _, err := a.VerifyCode(ctx, "a@mit.edu", code); err != nil {
		t.Fatal(err)
	}
	if _, err := a.VerifyCode(ctx, "a@mit.edu", code); !errors.Is(err, ErrCodeInvalid) {
		t.Fatalf("reused code: want ErrCodeInvalid, got %v", err)
	}

	a.now = func() time.Time { return time.Now().Add(time.Minute) }
	_ = a.RequestCode(ctx, "a@mit.edu")
	code = m.last["a@mit.edu"]
	for range MaxCodeAttempts {
		_, _ = a.VerifyCode(ctx, "a@mit.edu", "000000x")
	}
	if _, err := a.VerifyCode(ctx, "a@mit.edu", code); !errors.Is(err, ErrCodeInvalid) {
		t.Fatalf("locked code: want ErrCodeInvalid, got %v", err)
	}
}

func TestExpiredCode(t *testing.T) {
	ctx := context.Background()
	a, m, _ := newTestAuth()
	_ = a.RequestCode(ctx, "a@mit.edu")
	a.now = func() time.Time { return time.Now().Add(CodeTTL + time.Second) }
	if _, err := a.VerifyCode(ctx, "a@mit.edu", m.last["a@mit.edu"]); !errors.Is(err, ErrCodeInvalid) {
		t.Fatalf("want ErrCodeInvalid, got %v", err)
	}
}

func TestResendThrottle(t *testing.T) {
	ctx := context.Background()
	a, _, _ := newTestAuth()
	_ = a.RequestCode(ctx, "a@mit.edu")
	if err := a.RequestCode(ctx, "a@mit.edu"); !errors.Is(err, ErrTooSoon) {
		t.Fatalf("want ErrTooSoon, got %v", err)
	}
}

func TestPhoneTaken(t *testing.T) {
	ctx := context.Background()
	a, _, users := newTestAuth()
	_ = users.Create(ctx, &domain.User{Email: "x@mit.edu", Name: "X", WhatsAppPhone: "16175550123"})
	_, err := a.CompleteSignup(ctx, SignupInput{SignupToken: "signup:y@mit.edu", Name: "Y", WhatsAppPhone: "+1 617 555 0123"})
	if !errors.Is(err, domain.ErrPhoneTaken) {
		t.Fatalf("want ErrPhoneTaken, got %v", err)
	}
}

func TestNormalisePhone(t *testing.T) {
	cases := map[string]string{
		"617-555-0123":      "16175550123",
		"+1 (617) 555-0123": "16175550123",
		"+44 7911 123456":   "447911123456",
		"+52 55 1234 5678":  "525512345678",
	}
	for in, want := range cases {
		got, err := NormalisePhone(in)
		if err != nil || got != want {
			t.Errorf("NormalisePhone(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "123", "0123456789012", "+1234567890123456"} {
		if _, err := NormalisePhone(bad); err == nil {
			t.Errorf("NormalisePhone(%q) should fail", bad)
		}
	}
}
