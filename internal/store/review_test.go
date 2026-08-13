package store_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/store"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func reviewStoreFixture(t *testing.T) (*store.ReviewStore, string) {
	t.Helper()
	db, _ := testutil.TempDB(t)
	now := time.Date(2026, 8, 12, 9, 0, 0, 123456789, time.UTC)
	_, err := db.Exec(`
		INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','P','x','x');
		INSERT INTO notes(id,project_id,relative_path,status,revision,created_at,updated_at) VALUES('n1','p1','note.md','ready',1,'x','x');
		INSERT INTO review_items(id,project_id,note_id,kind,source_sha256,source_revision,prompt,stage,due_at,interval_days,ease_factor,reps,lapses,row_version,status,scheduler_version)
		VALUES('item1','p1','n1','whole','sha',1,'Review this note',0,?,0,2.5,0,0,0,'active','sm2-lite-v1')`, now.Format(time.RFC3339Nano))
	if err != nil {
		t.Fatal(err)
	}
	return &store.ReviewStore{DB: db, Clock: &clock.FakeClock{T: now}}, "item1"
}

func TestRateIsAtomicIdempotentAndVersioned(t *testing.T) {
	s, itemID := reviewStoreFixture(t)
	ctx := context.Background()
	got, err := s.Rate(ctx, itemID, "rate-1", 0, domain.RatingGood, 1250)
	if err != nil {
		t.Fatal(err)
	}
	if got.RowVersion != 1 || got.Stage != 1 || got.IntervalDays != 1 || got.LastReviewedAt == nil || !got.LastReviewedAt.Equal(s.Clock.Now()) {
		t.Fatalf("unexpected resulting item: %+v", got)
	}

	again, err := s.Rate(ctx, itemID, "rate-1", -99, domain.RatingEasy, 20)
	if err != nil || !reflect.DeepEqual(again, got) {
		t.Fatalf("replay=(%+v, %v), want original %+v", again, err, got)
	}
	var events int
	if err := s.DB.QueryRow(`SELECT count(*) FROM review_events WHERE request_key='rate-1'`).Scan(&events); err != nil || events != 1 {
		t.Fatalf("events=%d err=%v", events, err)
	}

	_, err = s.Rate(ctx, itemID, "rate-2", 0, domain.RatingEasy, 20)
	var conflict *store.RowVersionConflict
	if !errors.As(err, &conflict) || conflict.Current != 1 {
		t.Fatalf("conflict=%#v err=%v", conflict, err)
	}
	var version int64
	if err := s.DB.QueryRow(`SELECT row_version FROM review_items WHERE id=?`, itemID).Scan(&version); err != nil || version != 1 {
		t.Fatalf("version=%d err=%v", version, err)
	}
	if err := s.DB.QueryRow(`SELECT count(*) FROM review_events`).Scan(&events); err != nil || events != 1 {
		t.Fatalf("events after stale rate=%d err=%v", events, err)
	}
}

func TestRateRejectsInvalidExternalInputWithoutPanicking(t *testing.T) {
	s, itemID := reviewStoreFixture(t)
	_, err := s.Rate(context.Background(), itemID, "bad", 0, domain.Rating("surprise"), -1)
	if !errors.Is(err, store.ErrValidation) {
		t.Fatalf("err=%v", err)
	}
	var events int
	if err := s.DB.QueryRow(`SELECT count(*) FROM review_events`).Scan(&events); err != nil || events != 0 {
		t.Fatalf("events=%d err=%v", events, err)
	}
}
