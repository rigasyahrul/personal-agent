package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/store"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

type fakeProvider struct {
	calls int
	req   ChatRequest
	err   error
}

func (f *fakeProvider) Chat(_ context.Context, req ChatRequest) (ChatResponse, error) {
	f.calls++
	f.req = req
	if f.err != nil {
		return ChatResponse{}, f.err
	}
	return ChatResponse{Content: "answer"}, nil
}

func seededRunner(t *testing.T, parameters string, provider Provider) (*Runner, string, *store.RunStore, *store.MessageStore) {
	t.Helper()
	db, _ := testutil.TempDB(t)
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	if _, err := db.Exec(`INSERT INTO sessions
		(id,home,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at)
		VALUES('s1','global','active','openai','gpt-fixed',?,'{}','Chat',?,?)`, parameters, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	runs := &store.RunStore{DB: db, Now: func() time.Time { return now }}
	messages := &store.MessageStore{DB: db, Now: func() time.Time { return now }}
	return &Runner{DB: db, DataDir: t.TempDir(), Provider: provider, Messages: messages, Runs: runs,
		Sessions: &store.SessionStore{DB: db}}, "s1", runs, messages
}

func TestRunnerStartIsSynchronousIdempotentAndUsesImmutableModel(t *testing.T) {
	provider := &fakeProvider{}
	runner, sessionID, runs, messages := seededRunner(t, `{"temperature":0.2,"model":"overridden","messages":"overridden","tools":[{"name":"bad"}]}`, provider)
	runID, err := runner.Start(context.Background(), sessionID, "request-1", "question")
	if err != nil {
		t.Fatal(err)
	}
	retryID, err := runner.Start(context.Background(), sessionID, "request-1", "ignored retry")
	if err != nil || retryID != runID {
		t.Fatalf("retry = %q, %v", retryID, err)
	}
	if provider.calls != 1 || provider.req.Model != "gpt-fixed" || len(provider.req.Messages) != 1 || provider.req.Messages[0].Content != "question" || len(provider.req.Tools) != 0 {
		t.Fatalf("provider calls/request = %d, %#v", provider.calls, provider.req)
	}
	if provider.req.Parameters["temperature"] != 0.2 {
		t.Fatalf("parameters = %#v", provider.req.Parameters)
	}
	for _, fixed := range []string{"model", "messages", "tools"} {
		if _, exists := provider.req.Parameters[fixed]; exists {
			t.Fatalf("fixed field %q remains in parameters", fixed)
		}
	}
	run, err := runs.ByID(context.Background(), runID)
	if err != nil || run.Status != domain.AgentRunStatusCompleted {
		t.Fatalf("run = %#v, %v", run, err)
	}
	history, err := messages.List(context.Background(), sessionID)
	if err != nil || len(history) != 2 || history[0].Role != domain.MessageRoleUser || history[1].Content != "answer" {
		t.Fatalf("history = %#v, %v", history, err)
	}
}

func TestRunnerFailuresTerminalizeAdmittedRun(t *testing.T) {
	for _, tc := range []struct {
		name       string
		parameters string
		provider   Provider
	}{
		{name: "bad model JSON", parameters: `[]`, provider: &fakeProvider{}},
		{name: "null model JSON", parameters: `null`, provider: &fakeProvider{}},
		{name: "nil provider", parameters: `{}`},
		{name: "provider", parameters: `{}`, provider: &fakeProvider{err: errors.New("offline")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner, sessionID, runs, messages := seededRunner(t, tc.parameters, tc.provider)
			runID, err := runner.Start(context.Background(), sessionID, "request", "saved question")
			if err == nil {
				t.Fatal("Start succeeded")
			}
			run, lookupErr := runs.ByID(context.Background(), runID)
			if lookupErr != nil || run.Status != domain.AgentRunStatusFailed || run.Error == nil {
				t.Fatalf("run = %#v, %v", run, lookupErr)
			}
			history, listErr := messages.List(context.Background(), sessionID)
			if listErr != nil || len(history) != 1 || history[0].Content != "saved question" {
				t.Fatalf("history = %#v, %v", history, listErr)
			}
		})
	}
}

type completionFailingRuns struct {
	RunStore
	completionErr error
	failureErr    error
	failedCalls   int
}

func (s *completionFailingRuns) MarkDone(ctx context.Context, runID, status, message string) error {
	if status == domain.AgentRunStatusCompleted {
		return s.completionErr
	}
	if status == domain.AgentRunStatusFailed {
		s.failedCalls++
		if s.failureErr != nil {
			return s.failureErr
		}
	}
	return s.RunStore.MarkDone(ctx, runID, status, message)
}

func TestRunnerCompletionFailureAttemptsFailureTerminalization(t *testing.T) {
	runner, sessionID, runs, _ := seededRunner(t, `{}`, &fakeProvider{})
	completionErr := errors.New("complete failed")
	wrapped := &completionFailingRuns{RunStore: runner.Runs, completionErr: completionErr}
	runner.Runs = wrapped

	runID, err := runner.Start(context.Background(), sessionID, "request", "question")
	if !errors.Is(err, completionErr) {
		t.Fatalf("Start error = %v, want completion error", err)
	}
	if wrapped.failedCalls != 1 {
		t.Fatalf("failed MarkDone calls = %d, want 1", wrapped.failedCalls)
	}
	run, lookupErr := runs.ByID(context.Background(), runID)
	if lookupErr != nil || run.Status != domain.AgentRunStatusFailed {
		t.Fatalf("run = %#v, %v", run, lookupErr)
	}
}

func TestRunnerCompletionFailureJoinsFailureTerminalizationError(t *testing.T) {
	runner, sessionID, _, _ := seededRunner(t, `{}`, &fakeProvider{})
	completionErr := errors.New("complete failed")
	failureErr := errors.New("failure terminalization failed")
	wrapped := &completionFailingRuns{RunStore: runner.Runs, completionErr: completionErr, failureErr: failureErr}
	runner.Runs = wrapped

	_, err := runner.Start(context.Background(), sessionID, "request", "question")
	if !errors.Is(err, completionErr) || !errors.Is(err, failureErr) {
		t.Fatalf("Start error = %v, want joined completion and terminalization errors", err)
	}
	if wrapped.failedCalls != 1 {
		t.Fatalf("failed MarkDone calls = %d, want 1", wrapped.failedCalls)
	}
}

type failingSessionReader struct{ err error }

func (s failingSessionReader) Get(context.Context, string) (domain.Session, error) {
	return domain.Session{}, s.err
}

func TestRunnerSessionReadFailurePrecedesMessagesAndProvider(t *testing.T) {
	provider := &fakeProvider{}
	runner, sessionID, runs, _ := seededRunner(t, `{}`, provider)
	messages := &failingMessages{failAppend: -1}
	runner.Messages = messages
	runner.Sessions = failingSessionReader{err: errors.New("session read failed")}

	runID, err := runner.Start(context.Background(), sessionID, "request", "question")
	if err == nil {
		t.Fatal("Start succeeded")
	}
	if len(messages.items) != 0 || provider.calls != 0 {
		t.Fatalf("messages/provider calls = %d/%d, want 0/0", len(messages.items), provider.calls)
	}
	run, lookupErr := runs.ByID(context.Background(), runID)
	if lookupErr != nil || run.Status != domain.AgentRunStatusFailed {
		t.Fatalf("run = %#v, %v", run, lookupErr)
	}
}

type failingMessages struct {
	items      []domain.Message
	failList   bool
	failAppend int
}

func (s *failingMessages) List(context.Context, string) ([]domain.Message, error) {
	if s.failList {
		return nil, errors.New("list failed")
	}
	return s.items, nil
}
func (s *failingMessages) Append(_ context.Context, msg domain.Message) error {
	s.failAppend--
	if s.failAppend == 0 {
		return errors.New("append failed")
	}
	s.items = append(s.items, msg)
	return nil
}

func TestRunnerReadAndAppendFailuresTerminalizeRun(t *testing.T) {
	for _, tc := range []struct {
		name     string
		messages *failingMessages
	}{
		{name: "user append", messages: &failingMessages{failAppend: 1}},
		{name: "history read", messages: &failingMessages{failAppend: -1, failList: true}},
		{name: "assistant append", messages: &failingMessages{failAppend: 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner, sid, runs, _ := seededRunner(t, `{}`, &fakeProvider{})
			runner.Messages = tc.messages
			runID, err := runner.Start(context.Background(), sid, "request", "question")
			if err == nil {
				t.Fatal("Start succeeded")
			}
			run, lookupErr := runs.ByID(context.Background(), runID)
			if lookupErr != nil || run.Status != domain.AgentRunStatusFailed {
				t.Fatalf("run = %#v, %v", run, lookupErr)
			}
		})
	}
}
