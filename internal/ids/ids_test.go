package ids

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewIDIsUUID4(t *testing.T) {
	u, err := uuid.Parse(NewID())
	if err != nil {
		t.Fatal(err)
	}
	if u.Version() != 4 {
		t.Fatalf("version = %v, want 4", u.Version())
	}
}
