package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func TestBootstrapIsOneTimeAndValidatesToken(t *testing.T) {
	d, _ := testutil.TempDB(t)
	now := time.Unix(1000, 0).UTC()

	if err := Bootstrap(context.Background(), d, "secret", "wrong", "long-enough-password", now); !errors.Is(err, ErrBootstrapToken) {
		t.Fatalf("wrong token error = %v", err)
	}
	if err := Bootstrap(context.Background(), d, "secret", "secret", "long-enough-password", now); err != nil {
		t.Fatal(err)
	}
	if err := Bootstrap(context.Background(), d, "secret", "secret", "another-long-password", now); !errors.Is(err, ErrBootstrapped) {
		t.Fatalf("second bootstrap error = %v", err)
	}

	var hash, created string
	if err := d.QueryRow("SELECT password_hash, created_at FROM owner WHERE id=1").Scan(&hash, &created); err != nil {
		t.Fatal(err)
	}
	if hash == "long-enough-password" || !CheckPassword(hash, "long-enough-password") {
		t.Fatal("password was not securely hashed")
	}
	if created != now.Format(time.RFC3339Nano) {
		t.Fatalf("created_at = %q", created)
	}
}

func TestValidCSRF(t *testing.T) {
	if !ValidCSRF("same-token", "same-token") {
		t.Fatal("matching tokens rejected")
	}
	for _, pair := range [][2]string{{"", ""}, {"token", ""}, {"token", "wrong"}} {
		if ValidCSRF(pair[0], pair[1]) {
			t.Fatalf("accepted cookie %q header %q", pair[0], pair[1])
		}
	}
}
