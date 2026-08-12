package auth

import (
	"encoding/base64"
	"testing"
)

func TestNewSessionTokenIsRandomAndURLSafe(t *testing.T) {
	a, b := NewSessionToken(), NewSessionToken()
	if a == "" || a == b {
		t.Fatalf("NewSessionToken() produced empty or duplicate tokens: %q, %q", a, b)
	}
	raw, err := base64.RawURLEncoding.DecodeString(a)
	if err != nil {
		t.Fatalf("NewSessionToken() = %q, want unpadded URL-safe base64: %v", a, err)
	}
	if len(raw) != 32 {
		t.Fatalf("decoded token length = %d, want 32", len(raw))
	}
}

func TestTokenHashIsDeterministicSHA256(t *testing.T) {
	const want = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got := TokenHash("abc"); got != want {
		t.Fatalf("TokenHash(abc) = %q, want %q", got, want)
	}
	if TokenHash("abc") == "abc" {
		t.Fatal("TokenHash returned the plaintext token")
	}
}
