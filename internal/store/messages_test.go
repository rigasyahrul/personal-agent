package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

func seedAgentSession(t *testing.T) (*store.SessionStore, string) {
	t.Helper()
	dataDir := t.TempDir()
	ss := seedProject(t, dataDir)
	session, err := ss.CreateProject(context.Background(), store.CreateSessionInput{
		ProjectID: "p1", Provider: "openai", ModelID: "gpt-test", ModelParametersJSON: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	return ss, session.ID
}

func TestMessageStoreAppendAllocatesOrderedSequences(t *testing.T) {
	ss, sessionID := seedAgentSession(t)
	now := time.Date(2026, 8, 12, 11, 0, 0, 0, time.UTC)
	messages := &store.MessageStore{DB: ss.DB, Now: func() time.Time { return now }}

	first := domain.Message{SessionID: sessionID, Role: domain.MessageRoleUser, Content: "hello", Status: domain.MessageStatusComplete}
	if err := messages.Append(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	second := domain.Message{SessionID: sessionID, Role: domain.MessageRoleAssistant, Content: "hi", Status: domain.MessageStatusComplete}
	if err := messages.Append(context.Background(), second); err != nil {
		t.Fatal(err)
	}

	got, err := messages.List(context.Background(), sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Sequence != 1 || got[1].Sequence != 2 || got[0].Content != "hello" || got[1].Content != "hi" {
		t.Fatalf("messages = %#v", got)
	}
	for _, message := range got {
		if message.ID == "" || !message.CreatedAt.Equal(now) {
			t.Fatalf("missing generated fields: %#v", message)
		}
	}
}

func TestMessageStorePreservesExplicitFieldsAndRoles(t *testing.T) {
	ss, sessionID := seedAgentSession(t)
	messages := &store.MessageStore{DB: ss.DB, Now: time.Now}
	runID, calls, callID := "run-1", `[{"id":"call-1"}]`, "call-1"
	created := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	if _, err := ss.DB.Exec(`INSERT INTO agent_runs(id,session_id,request_key,status,created_at) VALUES(?,?,?,'completed',?)`, runID, sessionID, "prior", created); err != nil {
		t.Fatal(err)
	}
	input := domain.Message{ID: "message-1", SessionID: sessionID, RunID: &runID, Sequence: 7,
		Role: domain.MessageRoleTool, Content: "result", ToolCallsJSON: &calls, ToolCallID: &callID,
		Status: domain.MessageStatusFailed, CreatedAt: created}
	if err := messages.Append(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	got, err := messages.List(context.Background(), sessionID)
	if err != nil || len(got) != 1 {
		t.Fatalf("List = %#v, %v", got, err)
	}
	if got[0].ID != input.ID || got[0].Sequence != 7 || got[0].Role != domain.MessageRoleTool || got[0].Status != domain.MessageStatusFailed || got[0].RunID == nil || *got[0].RunID != runID || got[0].ToolCallsJSON == nil || *got[0].ToolCallsJSON != calls || got[0].ToolCallID == nil || *got[0].ToolCallID != callID {
		t.Fatalf("message = %#v", got[0])
	}
}
