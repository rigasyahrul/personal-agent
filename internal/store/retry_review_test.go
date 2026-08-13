package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rigasyahrul/personal-agent/internal/store"
)

func TestRetryReviewPendingOnlyRetriesFailedAndRetainsAttempts(t *testing.T) {
	s, _ := reviewStoreFixture(t)
	now := "2026-08-12T09:00:00Z"
	if _, err := s.DB.Exec(`INSERT INTO review_pending(id,note_id,source_sha256,generator_version,status,attempts,lease_until,last_error,created_at,updated_at) VALUES('pending','n1','hash','bites-v1','failed',4,?,'down',?,?)`, now, now, now); err != nil {
		t.Fatal(err)
	}
	if err := store.RetryReviewPending(context.Background(), s.DB, "pending"); err != nil {
		t.Fatal(err)
	}
	var status string
	var attempts int
	var lease, last any
	if err := s.DB.QueryRow(`SELECT status,attempts,lease_until,last_error FROM review_pending WHERE id='pending'`).Scan(&status, &attempts, &lease, &last); err != nil {
		t.Fatal(err)
	}
	if status != "pending" || attempts != 4 || lease != nil || last != nil {
		t.Fatalf("status=%s attempts=%d lease=%v last=%v", status, attempts, lease, last)
	}
	if err := store.RetryReviewPending(context.Background(), s.DB, "pending"); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}
