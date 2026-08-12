package auth

import (
	"strings"
	"testing"
)

func TestPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Fatalf("HashPassword() = %q, want locked argon2id parameters", hash)
	}
	if !CheckPassword(hash, "correct horse battery staple") {
		t.Fatal("CheckPassword() rejected the correct password")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("CheckPassword() accepted a wrong password")
	}
}

func TestCheckPasswordRejectsMalformedHashes(t *testing.T) {
	malformed := []string{
		"",
		"not-a-phc-hash",
		"$argon2i$v=19$m=65536,t=3,p=2$c2FsdA$aGFzaA",
		"$argon2id$v=16$m=65536,t=3,p=2$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=1,t=1,p=1$c2FsdA$aGFzaA",
		"$argon2id$v=19$m=65536,t=3,p=2$%%%$aGFzaA",
		"$argon2id$v=19$m=65536,t=3,p=2$c2FsdA$%%%",
		"$argon2id$v=19$m=65536,t=3,p=2$c2FsdA$aGFzaA$extra",
	}
	for _, hash := range malformed {
		if CheckPassword(hash, "password") {
			t.Errorf("CheckPassword(%q, password) = true, want false", hash)
		}
	}
}
