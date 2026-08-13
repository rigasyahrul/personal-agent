package store_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

func TestRunStoreBeginOrGetIsIdempotentAndSerializesActiveRuns(t *testing.T) {
	ss, sessionID := seedAgentSession(t)
	runs := &store.RunStore{DB: ss.DB, Now: time.Now}
	one, existing, err := runs.BeginOrGet(context.Background(), sessionID, "key-1")
	if err != nil || existing || one == "" {
		t.Fatalf("first = %q, %v, %v", one, existing, err)
	}
	same, existing, err := runs.BeginOrGet(context.Background(), sessionID, "key-1")
	if err != nil || !existing || same != one {
		t.Fatalf("retry = %q, %v, %v", same, existing, err)
	}
	if _, _, err := runs.BeginOrGet(context.Background(), sessionID, "key-2"); !errors.Is(err, store.ErrSessionBusy) {
		t.Fatalf("different key error = %v", err)
	}
	current, err := runs.Current(context.Background(), sessionID)
	if err != nil || current.ID != one || current.Status != domain.AgentRunStatusQueued {
		t.Fatalf("current = %#v, %v", current, err)
	}
}

func TestRunStoreTransitionsAndAllowsNextRunAfterTerminal(t *testing.T) {
	ss, sessionID := seedAgentSession(t)
	now := time.Date(2026, 8, 12, 13, 0, 0, 0, time.UTC)
	runs := &store.RunStore{DB: ss.DB, Now: func() time.Time { return now }}
	runID, _, err := runs.BeginOrGet(context.Background(), sessionID, "key-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := runs.MarkRunning(context.Background(), runID); err != nil {
		t.Fatal(err)
	}
	running, err := runs.ByID(context.Background(), runID)
	if err != nil || running.Status != domain.AgentRunStatusRunning || running.StartedAt == nil || !running.StartedAt.Equal(now) {
		t.Fatalf("running = %#v, %v", running, err)
	}
	if err := runs.MarkDone(context.Background(), runID, domain.AgentRunStatusFailed, "provider failed"); err != nil {
		t.Fatal(err)
	}
	done, err := runs.ByID(context.Background(), runID)
	if err != nil || done.Status != domain.AgentRunStatusFailed || done.CompletedAt == nil || done.Error == nil || *done.Error != "provider failed" {
		t.Fatalf("done = %#v, %v", done, err)
	}
	if _, err := runs.Current(context.Background(), sessionID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Current error = %v", err)
	}
	if next, existing, err := runs.BeginOrGet(context.Background(), sessionID, "key-2"); err != nil || existing || next == runID {
		t.Fatalf("next = %q, %v, %v", next, existing, err)
	}
}

func TestRunStoreRejectsMissingAndTerminalSessions(t *testing.T) {
	ss, sessionID := seedAgentSession(t)
	runs := &store.RunStore{DB: ss.DB, Now: time.Now}
	if _, _, err := runs.BeginOrGet(context.Background(), "missing", "key"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing error = %v", err)
	}
	if _, err := ss.DB.Exec(`UPDATE sessions SET status='terminal' WHERE id=?`, sessionID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runs.BeginOrGet(context.Background(), sessionID, "key"); !errors.Is(err, store.ErrSessionTerminal) {
		t.Fatalf("terminal error = %v", err)
	}
}

func TestRunStoreRetrySurvivesSessionTerminalization(t *testing.T) {
	ss, sessionID := seedAgentSession(t)
	runs := &store.RunStore{DB: ss.DB, Now: time.Now}
	runID, existing, err := runs.BeginOrGet(context.Background(), sessionID, "key-1")
	if err != nil || existing {
		t.Fatalf("first = %q, %v, %v", runID, existing, err)
	}
	if _, err := ss.DB.Exec(`UPDATE sessions SET status='terminal' WHERE id=?`, sessionID); err != nil {
		t.Fatal(err)
	}
	same, existing, err := runs.BeginOrGet(context.Background(), sessionID, "key-1")
	if err != nil || !existing || same != runID {
		t.Fatalf("retry = %q, %v, %v; want original run %q", same, existing, err, runID)
	}
	if _, _, err := runs.BeginOrGet(context.Background(), sessionID, "key-2"); !errors.Is(err, store.ErrSessionTerminal) {
		t.Fatalf("different key error = %v", err)
	}
}

func TestRunStoreConcurrentBeginOrGet(t *testing.T) {
	const count = 20
	t.Run("same key", func(t *testing.T) {
		ss, sessionID := seedAgentSession(t)
		runs := &store.RunStore{DB: ss.DB, Now: time.Now}
		type result struct {
			id       string
			existing bool
			err      error
		}
		results := make(chan result, count)
		var wg sync.WaitGroup
		for range count {
			wg.Add(1)
			go func() {
				defer wg.Done()
				id, existing, err := runs.BeginOrGet(context.Background(), sessionID, "same-key")
				results <- result{id: id, existing: existing, err: err}
			}()
		}
		wg.Wait()
		close(results)
		var id string
		created := 0
		for result := range results {
			if result.err != nil {
				t.Fatal(result.err)
			}
			if id == "" {
				id = result.id
			} else if result.id != id {
				t.Fatalf("run ID = %q, want %q", result.id, id)
			}
			if !result.existing {
				created++
			}
		}
		if created != 1 {
			t.Fatalf("created count = %d, want 1", created)
		}
	})

	t.Run("different keys", func(t *testing.T) {
		ss, sessionID := seedAgentSession(t)
		runs := &store.RunStore{DB: ss.DB, Now: time.Now}
		errs := make(chan error, count)
		var wg sync.WaitGroup
		for i := range count {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _, err := runs.BeginOrGet(context.Background(), sessionID, fmt.Sprintf("key-%d", i))
				errs <- err
			}()
		}
		wg.Wait()
		close(errs)
		created, busy := 0, 0
		for err := range errs {
			switch {
			case err == nil:
				created++
			case errors.Is(err, store.ErrSessionBusy):
				busy++
			default:
				t.Fatalf("unexpected error: %v", err)
			}
		}
		if created != 1 || busy != count-1 {
			t.Fatalf("created, busy = %d, %d; want 1, %d", created, busy, count-1)
		}
	})
}
