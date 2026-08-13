package store_test

import (
	"context"
	"errors"
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
	if _, _, err := runs.BeginOrGet(context.Background(), sessionID, "key"); !errors.Is(err, store.ErrValidation) {
		t.Fatalf("terminal error = %v", err)
	}
}
