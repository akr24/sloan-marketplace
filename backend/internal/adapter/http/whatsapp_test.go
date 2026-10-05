package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestValidSignature(t *testing.T) {
	body := []byte(`{"entry":[]}`)
	m := hmac.New(sha256.New, []byte("secret"))
	m.Write(body)
	good := "sha256=" + hex.EncodeToString(m.Sum(nil))

	if !validSignature("secret", body, good) {
		t.Error("good signature rejected")
	}
	for _, bad := range []string{"", good[7:], "sha256=zz", "sha256=" + hex.EncodeToString(make([]byte, 32))} {
		if validSignature("secret", body, bad) {
			t.Errorf("accepted %q", bad)
		}
	}
	if validSignature("other", body, good) {
		t.Error("accepted signature from another secret")
	}
}

func TestRecentIDsDedupesAndForgets(t *testing.T) {
	r := newRecentIDs(2)
	if !r.add("a") || r.add("a") {
		t.Fatal("a should be new once")
	}
	r.add("b")
	r.add("c") // evicts a
	if !r.add("a") {
		t.Error("a should have been forgotten")
	}
}
