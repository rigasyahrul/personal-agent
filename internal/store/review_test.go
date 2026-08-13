package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
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
	expectedPrevious := store.RatedItem{
		ID: itemID, Stage: 0, IntervalDays: 0, EaseFactor: 2.5, Reps: 0, Lapses: 0,
		DueAt: s.Clock.Now().UTC(), LastReviewedAt: nil, RowVersion: 0, SchedulerVersion: "sm2-lite-v1",
	}
	got, err := s.Rate(ctx, itemID, "rate-1", 0, domain.RatingGood, 1250)
	if err != nil {
		t.Fatal(err)
	}
	if got.RowVersion != 1 || got.Stage != 1 || got.IntervalDays != 1 || got.LastReviewedAt == nil || !got.LastReviewedAt.Equal(s.Clock.Now()) {
		t.Fatalf("unexpected resulting item: %+v", got)
	}
	var rating, scheduler, reviewedAt, previousJSON, resultingJSON string
	var duration int64
	if err := s.DB.QueryRow(`SELECT rating,scheduler_version,reviewed_at,duration_ms,previous_state_json,resulting_state_json FROM review_events WHERE request_key='rate-1'`).Scan(&rating, &scheduler, &reviewedAt, &duration, &previousJSON, &resultingJSON); err != nil {
		t.Fatal(err)
	}
	if rating != "good" || scheduler != "sm2-lite-v1" || reviewedAt != s.Clock.Now().UTC().Format(time.RFC3339Nano) || duration != 1250 {
		t.Fatalf("event metadata=(%q,%q,%q,%d)", rating, scheduler, reviewedAt, duration)
	}
	var previous, eventResult store.RatedItem
	if err := json.Unmarshal([]byte(previousJSON), &previous); err != nil || !reflect.DeepEqual(previous, expectedPrevious) {
		t.Fatalf("previous=%+v err=%v, want %+v", previous, err, expectedPrevious)
	}
	if err := json.Unmarshal([]byte(resultingJSON), &eventResult); err != nil || !reflect.DeepEqual(eventResult, got) {
		t.Fatalf("event result=%+v err=%v, want %+v", eventResult, err, got)
	}
	var persisted store.RatedItem
	var due, last string
	if err := s.DB.QueryRow(`SELECT id,stage,interval_days,ease_factor,reps,lapses,due_at,last_reviewed_at,row_version,scheduler_version FROM review_items WHERE id=?`, itemID).Scan(&persisted.ID, &persisted.Stage, &persisted.IntervalDays, &persisted.EaseFactor, &persisted.Reps, &persisted.Lapses, &due, &last, &persisted.RowVersion, &persisted.SchedulerVersion); err != nil {
		t.Fatal(err)
	}
	persisted.DueAt, _ = time.Parse(time.RFC3339Nano, due)
	lastTime, _ := time.Parse(time.RFC3339Nano, last)
	persisted.LastReviewedAt = &lastTime
	if !reflect.DeepEqual(persisted, got) {
		t.Fatalf("persisted=%+v, want %+v", persisted, got)
	}

	again, err := s.Rate(ctx, itemID, "rate-1", 0, domain.RatingGood, 1250)
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

func TestRateRejectsRequestKeyReuseForDifferentMutation(t *testing.T) {
	s, itemID := reviewStoreFixture(t)
	ctx := context.Background()
	if _, err := s.Rate(ctx, itemID, "rate-1", 0, domain.RatingGood, 1250); err != nil {
		t.Fatal(err)
	}
	_, err := s.DB.Exec(`INSERT INTO review_items(id,project_id,note_id,kind,source_sha256,source_revision,prompt,stage,due_at,interval_days,ease_factor,reps,lapses,row_version,status,scheduler_version)
		VALUES('item2','p1','n1','whole','sha',2,'Review this note',0,?,0,2.5,0,0,0,'active','sm2-lite-v1')`, s.Clock.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name       string
		item       string
		version    int64
		rating     domain.Rating
		durationMS int64
	}{
		{name: "item", item: "item2", version: 0, rating: domain.RatingGood, durationMS: 1250},
		{name: "version", item: itemID, version: 1, rating: domain.RatingGood, durationMS: 1250},
		{name: "rating", item: itemID, version: 0, rating: domain.RatingEasy, durationMS: 1250},
		{name: "duration", item: itemID, version: 0, rating: domain.RatingGood, durationMS: 1251},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.Rate(ctx, tc.item, "rate-1", tc.version, tc.rating, tc.durationMS)
			if !errors.Is(err, store.ErrConflict) {
				t.Fatalf("err=%v, want request-key conflict", err)
			}
		})
	}

	var events int
	if err := s.DB.QueryRow(`SELECT count(*) FROM review_events`).Scan(&events); err != nil || events != 1 {
		t.Fatalf("events=%d err=%v", events, err)
	}
}

func TestRateRejectsInvalidRatingWithoutPanicking(t *testing.T) {
	s, itemID := reviewStoreFixture(t)
	_, err := s.Rate(context.Background(), itemID, "bad", 0, domain.Rating("surprise"), 1)
	if !errors.Is(err, store.ErrValidation) {
		t.Fatalf("err=%v", err)
	}
	var events int
	if err := s.DB.QueryRow(`SELECT count(*) FROM review_events`).Scan(&events); err != nil || events != 0 {
		t.Fatalf("events=%d err=%v", events, err)
	}
}

func TestRateRejectsNegativeDuration(t *testing.T) {
	s, itemID := reviewStoreFixture(t)
	_, err := s.Rate(context.Background(), itemID, "bad", 0, domain.RatingGood, -1)
	if !errors.Is(err, store.ErrValidation) {
		t.Fatalf("err=%v", err)
	}
}

func TestConcurrentRateSameKeyConverges(t *testing.T) {
	s, itemID := reviewStoreFixture(t)
	start := make(chan struct{})
	results := make(chan store.RatedItem, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			got, err := s.Rate(context.Background(), itemID, "same", 0, domain.RatingGood, 10)
			results <- got
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("raw/concurrent error: %v", err)
		}
	}
	var first *store.RatedItem
	for got := range results {
		if first == nil {
			first = &got
		} else if !reflect.DeepEqual(*first, got) {
			t.Fatalf("results differ: %+v %+v", *first, got)
		}
	}
	var count int
	if err := s.DB.QueryRow(`SELECT count(*) FROM review_events`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("events=%d err=%v", count, err)
	}
}

func TestConcurrentRateDifferentKeysReturnsTypedConflict(t *testing.T) {
	s, itemID := reviewStoreFixture(t)
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, key := range []string{"one", "two"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			<-start
			_, err := s.Rate(context.Background(), itemID, key, 0, domain.RatingGood, 10)
			errs <- err
		}(key)
	}
	close(start)
	wg.Wait()
	close(errs)
	successes, conflicts := 0, 0
	for err := range errs {
		if err == nil {
			successes++
			continue
		}
		var conflict *store.RowVersionConflict
		if errors.As(err, &conflict) && conflict.Current == 1 {
			conflicts++
			continue
		}
		t.Fatalf("raw/untyped error: %v", err)
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
	var count int
	if err := s.DB.QueryRow(`SELECT count(*) FROM review_events`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("events=%d err=%v", count, err)
	}
}
