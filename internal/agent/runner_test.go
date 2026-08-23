package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/store"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func waitRunTerminal(t *testing.T, runner *Runner, runs *store.RunStore, runID string) domain.AgentRun {
	t.Helper()
	runner.Wait()
	run, err := runs.ByID(context.Background(), runID)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	return run
}

type fakeProvider struct {
	calls int
	req   ChatRequest
	err   error
}

type scriptedProvider struct {
	requests  []ChatRequest
	responses []ChatResponse
}

func (p *scriptedProvider) Chat(_ context.Context, req ChatRequest) (ChatResponse, error) {
	p.requests = append(p.requests, req)
	response := p.responses[0]
	p.responses = p.responses[1:]
	return response, nil
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

func toolRunner(t *testing.T, provider Provider, granted bool) (*Runner, string, *store.RunStore, *store.MessageStore) {
	t.Helper()
	runner, sessionID, runs, messages := seededRunner(t, `{}`, provider)
	grants := `{"workspace_files":false}`
	if granted {
		grants = `{"workspace_files":true}`
	}
	if _, err := runner.DB.Exec(`UPDATE sessions SET tool_grants_json=? WHERE id=?`, grants, sessionID); err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(runner.DataDir, "files", "global", "sessions", sessionID)
	if err := os.MkdirAll(workspace, 0700); err != nil {
		t.Fatal(err)
	}
	return runner, sessionID, runs, messages
}

func TestRunnerDoesNotAdvertiseToolsWithoutGrant(t *testing.T) {
	provider := &scriptedProvider{responses: []ChatResponse{{Content: "plain answer"}}}
	runner, sessionID, _, messages := toolRunner(t, provider, false)
	if _, err := runner.Start(context.Background(), sessionID, "request", "write x"); err != nil {
		t.Fatal(err)
	}
	runner.Wait()
	if len(provider.requests) != 1 || len(provider.requests[0].Tools) != 0 {
		t.Fatalf("tools leaked: %#v", provider.requests)
	}
	history, _ := messages.List(context.Background(), sessionID)
	for _, message := range history {
		if message.Role == domain.MessageRoleTool {
			t.Fatalf("tool message persisted: %#v", message)
		}
	}
}

func TestRunnerExecutesRootedToolsAndPreservesProtocol(t *testing.T) {
	call := ToolCall{ID: "call-1", Name: "write_file", Arguments: `{"path":"draft.txt","content":"hello"}`}
	provider := &scriptedProvider{responses: []ChatResponse{{ToolCalls: []ToolCall{call}}, {Content: "saved"}}}
	runner, sessionID, _, messages := toolRunner(t, provider, true)
	if _, err := runner.Start(context.Background(), sessionID, "request", "save it"); err != nil {
		t.Fatal(err)
	}
	runner.Wait()
	if len(provider.requests) != 2 || len(provider.requests[0].Tools) != 4 {
		t.Fatalf("requests = %#v", provider.requests)
	}
	followup := provider.requests[1].Messages
	// system prompt + user + assistant(tool_calls) + tool result
	if len(followup) != 4 || followup[0].Role != domain.MessageRoleSystem ||
		len(followup[2].ToolCalls) != 1 || followup[2].ToolCalls[0] != call || followup[3].ToolCallID != call.ID {
		t.Fatalf("follow-up protocol = %#v", followup)
	}
	history, err := messages.List(context.Background(), sessionID)
	if err != nil || len(history) != 4 || history[1].ToolCallsJSON == nil || history[2].ToolCallID == nil || *history[2].ToolCallID != call.ID {
		t.Fatalf("history = %#v, %v", history, err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(history[2].Content), &result); err != nil || result["changed_path"] != "draft.txt" {
		t.Fatalf("tool result = %#v, %v", result, err)
	}
	got, err := os.ReadFile(filepath.Join(runner.DataDir, "files", "global", "sessions", sessionID, "draft.txt"))
	if err != nil || string(got) != "hello" {
		t.Fatalf("workspace file = %q, %v", got, err)
	}
}

func TestRunnerRejectsHostileAndUnknownToolArgumentsSafely(t *testing.T) {
	for _, call := range []ToolCall{
		{ID: "escape", Name: "read_file", Arguments: `{"path":"../../etc/passwd"}`},
		{ID: "shell", Name: "shell", Arguments: `{"command":"id"}`},
	} {
		t.Run(call.ID, func(t *testing.T) {
			provider := &scriptedProvider{responses: []ChatResponse{{ToolCalls: []ToolCall{call}}, {Content: "done"}}}
			runner, sessionID, _, messages := toolRunner(t, provider, true)
			if _, err := runner.Start(context.Background(), sessionID, call.ID, "try"); err != nil {
				t.Fatal(err)
			}
			runner.Wait()
			history, _ := messages.List(context.Background(), sessionID)
			content := history[2].Content
			if !strings.Contains(content, "error") || strings.Contains(content, runner.DataDir) || strings.Contains(content, "/etc/") {
				t.Fatalf("unsafe tool error %q", content)
			}
		})
	}
}

func TestRunnerRejectsInvalidToolCallIDsBeforePersistenceOrExecution(t *testing.T) {
	for _, tc := range []struct {
		name  string
		calls []ToolCall
	}{
		{name: "empty", calls: []ToolCall{{ID: "", Name: "write_file", Arguments: `{"path":"empty.txt","content":"bad"}`}}},
		{name: "whitespace", calls: []ToolCall{{ID: " \t\n", Name: "write_file", Arguments: `{"path":"whitespace.txt","content":"bad"}`}}},
		{name: "duplicate", calls: []ToolCall{
			{ID: "same", Name: "write_file", Arguments: `{"path":"first.txt","content":"bad"}`},
			{ID: "same", Name: "write_file", Arguments: `{"path":"second.txt","content":"bad"}`},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := &scriptedProvider{responses: []ChatResponse{{ToolCalls: tc.calls}}}
			runner, sessionID, runs, messages := toolRunner(t, provider, true)

			runID, err := runner.Start(context.Background(), sessionID, "invalid-"+tc.name, "try invalid calls")
			if err != nil {
				t.Fatalf("Start admission error = %v", err)
			}
			runner.Wait()
			run, lookupErr := runs.ByID(context.Background(), runID)
			if lookupErr != nil || run.Status != domain.AgentRunStatusFailed {
				t.Fatalf("run = %#v, %v", run, lookupErr)
			}
			if len(provider.requests) != 1 {
				t.Fatalf("provider calls = %d, want 1", len(provider.requests))
			}
			history, listErr := messages.List(context.Background(), sessionID)
			if listErr != nil || len(history) != 1 || history[0].Role != domain.MessageRoleUser {
				t.Fatalf("history = %#v, %v", history, listErr)
			}
			workspace := filepath.Join(runner.DataDir, "files", "global", "sessions", sessionID)
			entries, readErr := os.ReadDir(workspace)
			if readErr != nil || len(entries) != 0 {
				t.Fatalf("workspace entries = %#v, %v", entries, readErr)
			}
		})
	}
}

func TestRunnerToolRoundLimitTerminalizesRun(t *testing.T) {
	responses := make([]ChatResponse, 8)
	for i := range responses {
		responses[i].ToolCalls = []ToolCall{{ID: string(rune('a' + i)), Name: "mkdir", Arguments: `{"path":"x"}`}}
	}
	provider := &scriptedProvider{responses: responses}
	runner, sessionID, runs, _ := toolRunner(t, provider, true)
	runID, err := runner.Start(context.Background(), sessionID, "limit", "loop")
	if err != nil {
		t.Fatalf("Start admission error = %v", err)
	}
	runner.Wait()
	run, lookupErr := runs.ByID(context.Background(), runID)
	if lookupErr != nil || run.Status != domain.AgentRunStatusFailed || len(provider.requests) != 8 {
		t.Fatalf("run/requests = %#v/%d, %v", run, len(provider.requests), lookupErr)
	}
	if run.Error == nil || !strings.Contains(*run.Error, "tool round limit") {
		t.Fatalf("run error = %#v", run.Error)
	}
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
	runner.Wait()
	if provider.calls != 1 || provider.req.Model != "gpt-fixed" || len(provider.req.Tools) != 0 {
		t.Fatalf("provider calls/request = %d, %#v", provider.calls, provider.req)
	}
	if len(provider.req.Messages) < 2 || provider.req.Messages[0].Role != domain.MessageRoleSystem ||
		provider.req.Messages[len(provider.req.Messages)-1].Content != "question" {
		t.Fatalf("provider messages = %#v", provider.req.Messages)
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
	// Injected system prompt is ephemeral — never persisted.
	for _, m := range history {
		if m.Role == domain.MessageRoleSystem {
			t.Fatalf("system prompt must not be persisted: %#v", m)
		}
	}
}

func TestRunnerInjectsScopedPromptSystemMessage(t *testing.T) {
	provider := &fakeProvider{}
	db, _ := testutil.TempDB(t)
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	dataDir := t.TempDir()

	// Project AGENTS.md on disk — must appear in the leading system message.
	projectRoot := filepath.Join(dataDir, "files", "vaults", "v1", "projects", "p1")
	if err := os.MkdirAll(projectRoot, 0700); err != nil {
		t.Fatal(err)
	}
	const agentsBody = "PROJECT_AGENTS_SCOPED_BODY"
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte(agentsBody), 0600); err != nil {
		t.Fatal(err)
	}

	ts := now.Format(time.RFC3339Nano)
	if _, err := db.Exec(`INSERT INTO vaults(id,name,created_at,updated_at) VALUES('v1','Vault',?,?);
		INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES('p1','v1','Project',?,?);
		INSERT INTO sessions
		(id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at)
		VALUES('s-proj','project','v1','p1','active','openai','gpt-fixed','{}','{}','Chat',?,?)`,
		ts, ts, ts, ts, ts, ts); err != nil {
		t.Fatal(err)
	}
	// Stale PA_RUNTIME_V1 system message in history must be stripped (no stacking).
	stale := "PA_RUNTIME_V1\nstale-runtime-blob-should-be-dropped"
	if _, err := db.Exec(`INSERT INTO messages
		(id,session_id,sequence,role,content,status,created_at)
		VALUES('m-stale','s-proj',1,'system',?,'complete',?)`, stale, now.Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}

	runs := &store.RunStore{DB: db, Now: func() time.Time { return now }}
	messages := &store.MessageStore{DB: db, Now: func() time.Time { return now }}
	runner := &Runner{
		DB: db, DataDir: dataDir, Provider: provider, Messages: messages, Runs: runs,
		Sessions: &store.SessionStore{DB: db},
	}

	if _, err := runner.Start(context.Background(), "s-proj", "req-prompt", "hello project"); err != nil {
		t.Fatal(err)
	}
	runner.Wait()

	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}
	msgs := provider.req.Messages
	if len(msgs) < 2 {
		t.Fatalf("want system + history messages, got %#v", msgs)
	}
	if msgs[0].Role != domain.MessageRoleSystem {
		t.Fatalf("messages[0].Role = %q, want system", msgs[0].Role)
	}
	if !strings.HasPrefix(msgs[0].Content, "PA_RUNTIME_V1") {
		t.Fatalf("system content must start with PA_RUNTIME_V1, got %q", msgs[0].Content[:min(60, len(msgs[0].Content))])
	}
	if !strings.Contains(msgs[0].Content, agentsBody) {
		t.Fatalf("system content missing AGENTS body %q; got %q", agentsBody, msgs[0].Content)
	}
	// Exactly one PA_RUNTIME_V1 marker in the full request (stale history stripped).
	joined := ""
	for _, m := range msgs {
		joined += m.Content + "\n"
	}
	if n := strings.Count(joined, "PA_RUNTIME_V1"); n != 1 {
		t.Fatalf("PA_RUNTIME_V1 count = %d, want 1 (no stacking); messages=%#v", n, msgs)
	}
	// User message still present after system prefix.
	last := msgs[len(msgs)-1]
	if last.Role != domain.MessageRoleUser || last.Content != "hello project" {
		t.Fatalf("last message = %#v, want user hello project", last)
	}

	// Ephemeral: store must not gain a new system message from the injection.
	history, err := messages.List(context.Background(), "s-proj")
	if err != nil {
		t.Fatal(err)
	}
	var systemCount int
	for _, m := range history {
		if m.Role == domain.MessageRoleSystem {
			systemCount++
			if m.Content != stale {
				t.Fatalf("unexpected system message persisted: %q", m.Content)
			}
		}
	}
	if systemCount != 1 {
		t.Fatalf("persisted system messages = %d, want only the pre-seeded stale one", systemCount)
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
			if err != nil {
				t.Fatalf("Start admission error = %v", err)
			}
			runner.Wait()
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
	inner         *store.RunStore
	completionErr error
	failureErr    error
	failedCalls   int
}

func (s *completionFailingRuns) Admit(ctx context.Context, sessionID, requestKey, userMessage string, now time.Time) (store.RunAdmission, error) {
	return s.inner.Admit(ctx, sessionID, requestKey, userMessage, now)
}
func (s *completionFailingRuns) BeginOrGet(ctx context.Context, sessionID, requestKey string) (string, bool, error) {
	return s.inner.BeginOrGet(ctx, sessionID, requestKey)
}
func (s *completionFailingRuns) MarkRunning(ctx context.Context, runID string) error {
	return s.inner.MarkRunning(ctx, runID)
}
func (s *completionFailingRuns) ByID(ctx context.Context, runID string) (domain.AgentRun, error) {
	return s.inner.ByID(ctx, runID)
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
	return s.inner.MarkDone(ctx, runID, status, message)
}

func TestRunnerCompletionFailureAttemptsFailureTerminalization(t *testing.T) {
	runner, sessionID, runs, _ := seededRunner(t, `{}`, &fakeProvider{})
	completionErr := errors.New("complete failed")
	wrapped := &completionFailingRuns{inner: runs, completionErr: completionErr}
	runner.Runs = wrapped

	runID, err := runner.Start(context.Background(), sessionID, "request", "question")
	if err != nil {
		t.Fatalf("Start admission error = %v", err)
	}
	runner.Wait()
	if wrapped.failedCalls != 1 {
		t.Fatalf("failed MarkDone calls = %d, want 1", wrapped.failedCalls)
	}
	run, lookupErr := runs.ByID(context.Background(), runID)
	if lookupErr != nil || run.Status != domain.AgentRunStatusFailed {
		t.Fatalf("run = %#v, %v", run, lookupErr)
	}
}

func TestRunnerCompletionFailureJoinsFailureTerminalizationError(t *testing.T) {
	runner, sessionID, runs, _ := seededRunner(t, `{}`, &fakeProvider{})
	completionErr := errors.New("complete failed")
	failureErr := errors.New("failure terminalization failed")
	wrapped := &completionFailingRuns{inner: runs, completionErr: completionErr, failureErr: failureErr}
	runner.Runs = wrapped

	_, err := runner.Start(context.Background(), sessionID, "request", "question")
	if err != nil {
		t.Fatalf("Start admission error = %v", err)
	}
	runner.Wait()
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
	if err != nil {
		t.Fatalf("Start admission error = %v", err)
	}
	runner.Wait()
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
		// User message is written inside store.Admit; Messages is only used after admission.
		{name: "history read", messages: &failingMessages{failAppend: -1, failList: true}},
		{name: "assistant append", messages: &failingMessages{failAppend: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner, sid, runs, _ := seededRunner(t, `{}`, &fakeProvider{})
			runner.Messages = tc.messages
			runID, err := runner.Start(context.Background(), sid, "request", "question")
			if err != nil {
				t.Fatalf("Start admission error = %v", err)
			}
			runner.Wait()
			run, lookupErr := runs.ByID(context.Background(), runID)
			if lookupErr != nil || run.Status != domain.AgentRunStatusFailed {
				t.Fatalf("run = %#v, %v", run, lookupErr)
			}
		})
	}
}

type blockingChatProvider struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (p *blockingChatProvider) Chat(context.Context, ChatRequest) (ChatResponse, error) {
	p.once.Do(func() { close(p.started) })
	<-p.release
	return ChatResponse{Content: "answer"}, nil
}

func TestTwoTabsOneAgentRunDifferentKeys(t *testing.T) {
	provider := &blockingChatProvider{started: make(chan struct{}), release: make(chan struct{})}
	runner, sessionID, runs, _ := seededRunner(t, `{}`, provider)
	db := runner.DB
	type result struct {
		id  string
		err error
	}
	start := make(chan struct{})
	out := make(chan result, 2)
	for _, key := range []string{"tab-a", "tab-b"} {
		key := key
		go func() {
			<-start
			id, err := runner.Start(context.Background(), sessionID, key, "explain this")
			out <- result{id, err}
		}()
	}
	close(start)
	a, b := <-out, <-out
	busy := 0
	started := 0
	for _, got := range []result{a, b} {
		switch {
		case got.err == nil:
			started++
		case errors.Is(got.err, ErrSessionBusy):
			busy++
		default:
			t.Fatalf("unexpected result: id=%q err=%v", got.id, got.err)
		}
	}
	if started != 1 || busy != 1 {
		t.Fatalf("started=%d busy=%d, want 1 and 1", started, busy)
	}
	// Ensure provider is unblocked and run finishes.
	select {
	case <-provider.started:
	case <-time.After(2 * time.Second):
		t.Fatal("provider never started")
	}
	close(provider.release)
	runner.Wait()
	_ = runs
	_ = db
}

func TestTwoTabsOneAgentRunSameKeyIsIdempotent(t *testing.T) {
	provider := &blockingChatProvider{started: make(chan struct{}), release: make(chan struct{})}
	runner, sessionID, _, _ := seededRunner(t, `{}`, provider)
	start := make(chan struct{})
	ids := make(chan string, 2)
	errs := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			id, err := runner.Start(context.Background(), sessionID, "same-key", "explain this")
			ids <- id
			errs <- err
		}()
	}
	close(start)
	id1, id2 := <-ids, <-ids
	if err1, err2 := <-errs, <-errs; err1 != nil || err2 != nil {
		t.Fatalf("errors = %v, %v", err1, err2)
	}
	if id1 == "" || id1 != id2 {
		t.Fatalf("run IDs = %q, %q", id1, id2)
	}
	var runs, userMessages int
	if err := runner.DB.QueryRow(`SELECT count(*) FROM agent_runs WHERE session_id=?`, sessionID).Scan(&runs); err != nil {
		t.Fatal(err)
	}
	if err := runner.DB.QueryRow(`SELECT count(*) FROM messages WHERE session_id=? AND role='user'`, sessionID).Scan(&userMessages); err != nil {
		t.Fatal(err)
	}
	if runs != 1 || userMessages != 1 {
		t.Fatalf("runs=%d user_messages=%d, want 1 and 1", runs, userMessages)
	}
	select {
	case <-provider.started:
	case <-time.After(2 * time.Second):
		t.Fatal("provider never started")
	}
	close(provider.release)
	runner.Wait()
}

func projectKnowledgeRunner(t *testing.T, provider Provider, granted bool) (*Runner, string, *store.RunStore, *store.MessageStore) {
	t.Helper()
	db, _ := testutil.TempDB(t)
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	dataDir := t.TempDir()
	ts := now.Format(time.RFC3339Nano)
	if _, err := db.Exec(`
		INSERT INTO vaults(id,name,created_at,updated_at) VALUES('v1','Vault',?,?);
		INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES('p1','v1','Project',?,?);
		INSERT INTO sessions
		(id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at)
		VALUES('s-proj','project','v1','p1','active','openai','gpt-fixed','{}','{}','Chat',?,?);
		INSERT INTO notes(id,project_id,relative_path,status,revision,created_at,updated_at)
			VALUES('note-src','p1','articles/intro.md','ready',1,?,?);
	`, ts, ts, ts, ts, ts, ts, ts, ts); err != nil {
		t.Fatal(err)
	}
	grants := `{"workspace_files":false}`
	if granted {
		grants = `{"workspace_files":true}`
	}
	if _, err := db.Exec(`UPDATE sessions SET tool_grants_json=? WHERE id='s-proj'`, grants); err != nil {
		t.Fatal(err)
	}

	projectRoot := filepath.Join(dataDir, "files", "vaults", "v1", "projects", "p1")
	if err := os.MkdirAll(projectRoot, 0700); err != nil {
		t.Fatal(err)
	}
	const agentsBody = "PROJECT_AGENTS_KNOWLEDGE_BODY\n"
	if err := os.WriteFile(filepath.Join(projectRoot, "AGENTS.md"), []byte(agentsBody), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := (&store.KnowledgeStore{DB: db}).UpsertFromContent(context.Background(), store.UpsertKnowledgeInput{
		Kind:         domain.KnowledgeKindSource,
		ProjectID:    "p1",
		RelativePath: "source/articles/intro.md",
		Content:      []byte("Body mentions searchneedletoken once.\n"),
		Status:       "ready",
		SourceNoteID: "note-src",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(projectRoot, "sessions", "s-proj"), 0700); err != nil {
		t.Fatal(err)
	}

	runs := &store.RunStore{DB: db, Now: func() time.Time { return now }}
	messages := &store.MessageStore{DB: db, Now: func() time.Time { return now }}
	return &Runner{
		DB: db, DataDir: dataDir, Provider: provider, Messages: messages, Runs: runs,
		Sessions: &store.SessionStore{DB: db},
	}, "s-proj", runs, messages
}

func advertisedToolNames(req ChatRequest) []string {
	names := make([]string, 0, len(req.Tools))
	for _, tool := range req.Tools {
		names = append(names, tool.Name)
	}
	return names
}

func hasToolName(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}

// Break this would catch: knowledge tools gated on workspace_files, dispatched
// through Workspace, or advertising write_knowledge.
func TestRunnerProjectHomeKnowledgeToolsWithoutWorkspaceGrant(t *testing.T) {
	search := ToolCall{ID: "call-search", Name: "search_project", Arguments: `{"query":"searchneedletoken","limit":5}`}
	read := ToolCall{ID: "call-read", Name: "read_knowledge", Arguments: `{"path":"AGENTS.md"}`}
	provider := &scriptedProvider{responses: []ChatResponse{
		{ToolCalls: []ToolCall{search, read}},
		{Content: "done"},
	}}
	runner, sessionID, runs, messages := projectKnowledgeRunner(t, provider, false)

	runID, err := runner.Start(context.Background(), sessionID, "req-knowledge", "search then read")
	if err != nil {
		t.Fatal(err)
	}
	runner.Wait()
	run, lookupErr := runs.ByID(context.Background(), runID)
	if lookupErr != nil || run.Status != domain.AgentRunStatusCompleted {
		t.Fatalf("run = %#v, %v", run, lookupErr)
	}
	if len(provider.requests) != 2 {
		t.Fatalf("provider calls = %d, want 2", len(provider.requests))
	}
	names := advertisedToolNames(provider.requests[0])
	for _, want := range []string{"search_project", "read_knowledge", "list_knowledge"} {
		if !hasToolName(names, want) {
			t.Fatalf("missing knowledge tool %q in %#v", want, names)
		}
	}
	if hasToolName(names, "write_knowledge") {
		t.Fatalf("write_knowledge advertised: %#v", names)
	}
	for _, workspaceName := range []string{"read_file", "write_file", "edit_file", "mkdir"} {
		if hasToolName(names, workspaceName) {
			t.Fatalf("workspace tool %q advertised without grant: %#v", workspaceName, names)
		}
	}

	history, err := messages.List(context.Background(), sessionID)
	if err != nil {
		t.Fatal(err)
	}
	var searchRaw, readRaw string
	for _, message := range history {
		if message.Role != domain.MessageRoleTool || message.ToolCallID == nil {
			continue
		}
		switch *message.ToolCallID {
		case search.ID:
			searchRaw = message.Content
		case read.ID:
			readRaw = message.Content
		}
	}
	if searchRaw == "" || readRaw == "" {
		t.Fatalf("missing tool results; history=%#v", history)
	}
	var searchOut struct {
		Hits []struct {
			KnowledgeID string `json:"knowledge_id"`
			Path        string `json:"path"`
		} `json:"hits"`
	}
	if err := json.Unmarshal([]byte(searchRaw), &searchOut); err != nil {
		t.Fatalf("search result %q: %v", searchRaw, err)
	}
	if len(searchOut.Hits) != 1 || searchOut.Hits[0].Path != "source/articles/intro.md" || searchOut.Hits[0].KnowledgeID == "" {
		t.Fatalf("search hits = %#v from %s", searchOut.Hits, searchRaw)
	}
	var readOut struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(readRaw), &readOut); err != nil {
		t.Fatalf("read result %q: %v", readRaw, err)
	}
	if readOut.Path != "AGENTS.md" || readOut.Content != "PROJECT_AGENTS_KNOWLEDGE_BODY\n" {
		t.Fatalf("read_knowledge = %+v", readOut)
	}
}

// Break this would catch: knowledge tools registered for vault or global homes.
func TestRunnerVaultAndGlobalDoNotRegisterKnowledgeTools(t *testing.T) {
	for _, tc := range []struct {
		name    string
		home    string
		granted bool
	}{
		{name: "global no grant", home: "global", granted: false},
		{name: "global grant", home: "global", granted: true},
		{name: "vault no grant", home: "vault", granted: false},
		{name: "vault grant", home: "vault", granted: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := &scriptedProvider{responses: []ChatResponse{{Content: "plain"}}}
			var runner *Runner
			var sessionID string
			if tc.home == "global" {
				runner, sessionID, _, _ = toolRunner(t, provider, tc.granted)
			} else {
				runner, sessionID, _, _ = vaultToolRunner(t, provider, tc.granted)
			}
			if _, err := runner.Start(context.Background(), sessionID, "req-"+tc.name, "hello"); err != nil {
				t.Fatal(err)
			}
			runner.Wait()
			if len(provider.requests) != 1 {
				t.Fatalf("provider calls = %d", len(provider.requests))
			}
			names := advertisedToolNames(provider.requests[0])
			for _, knowledgeName := range []string{"search_project", "read_knowledge", "list_knowledge", "write_knowledge"} {
				if hasToolName(names, knowledgeName) {
					t.Fatalf("%s advertised knowledge tool %q: %#v", tc.home, knowledgeName, names)
				}
			}
			if tc.granted && len(names) != 4 {
				t.Fatalf("granted %s tools = %#v, want 4 workspace tools", tc.home, names)
			}
			if !tc.granted && len(names) != 0 {
				t.Fatalf("ungranted %s tools leaked: %#v", tc.home, names)
			}
		})
	}
}

func vaultToolRunner(t *testing.T, provider Provider, granted bool) (*Runner, string, *store.RunStore, *store.MessageStore) {
	t.Helper()
	db, _ := testutil.TempDB(t)
	now := time.Date(2026, 8, 12, 12, 0, 0, 0, time.UTC)
	dataDir := t.TempDir()
	ts := now.Format(time.RFC3339Nano)
	if _, err := db.Exec(`
		INSERT INTO vaults(id,name,created_at,updated_at) VALUES('v1','Vault',?,?);
		INSERT INTO sessions
		(id,home,vault_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at)
		VALUES('s-vault','vault','v1','active','openai','gpt-fixed','{}','{}','Chat',?,?);
	`, ts, ts, ts, ts); err != nil {
		t.Fatal(err)
	}
	grants := `{"workspace_files":false}`
	if granted {
		grants = `{"workspace_files":true}`
	}
	if _, err := db.Exec(`UPDATE sessions SET tool_grants_json=? WHERE id='s-vault'`, grants); err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(dataDir, "files", "vaults", "v1", "sessions", "s-vault")
	if err := os.MkdirAll(workspace, 0700); err != nil {
		t.Fatal(err)
	}
	runs := &store.RunStore{DB: db, Now: func() time.Time { return now }}
	messages := &store.MessageStore{DB: db, Now: func() time.Time { return now }}
	return &Runner{
		DB: db, DataDir: dataDir, Provider: provider, Messages: messages, Runs: runs,
		Sessions: &store.SessionStore{DB: db},
	}, "s-vault", runs, messages
}

// Break this would catch: workspace names executed without a grant, or unknown
// names (including write_knowledge) accepted on project home.
func TestRunnerProjectHomeRejectsWorkspaceAndUnknownWithoutGrant(t *testing.T) {
	for _, call := range []ToolCall{
		{ID: "ws", Name: "write_file", Arguments: `{"path":"draft.txt","content":"nope"}`},
		{ID: "unknown", Name: "write_knowledge", Arguments: `{"path":"AGENTS.md","content":"x"}`},
	} {
		t.Run(call.Name, func(t *testing.T) {
			provider := &scriptedProvider{responses: []ChatResponse{{ToolCalls: []ToolCall{call}}, {Content: "done"}}}
			runner, sessionID, runs, messages := projectKnowledgeRunner(t, provider, false)
			runID, err := runner.Start(context.Background(), sessionID, "req-"+call.Name, "try")
			if err != nil {
				t.Fatal(err)
			}
			runner.Wait()
			run, lookupErr := runs.ByID(context.Background(), runID)
			if lookupErr != nil || run.Status != domain.AgentRunStatusCompleted {
				t.Fatalf("run = %#v, %v", run, lookupErr)
			}
			history, _ := messages.List(context.Background(), sessionID)
			var result string
			for _, message := range history {
				if message.Role == domain.MessageRoleTool {
					result = message.Content
				}
			}
			if result == "" || !strings.Contains(result, "error") {
				t.Fatalf("want rejected tool result, got %q history=%#v", result, history)
			}
			if _, err := os.Stat(filepath.Join(runner.DataDir, "files", "vaults", "v1", "projects", "p1", "sessions", sessionID, "draft.txt")); !os.IsNotExist(err) {
				t.Fatalf("workspace file written without grant: %v", err)
			}
			agents, err := os.ReadFile(filepath.Join(runner.DataDir, "files", "vaults", "v1", "projects", "p1", "AGENTS.md"))
			if err != nil || string(agents) != "PROJECT_AGENTS_KNOWLEDGE_BODY\n" {
				t.Fatalf("AGENTS.md mutated: %q, %v", agents, err)
			}
		})
	}
}

// Break this would catch: compound generation inheriting project knowledge tools.
func TestCompoundProjectHomeStaysToolsOff(t *testing.T) {
	payload, err := json.Marshal([]store.CompoundItem{{
		Kind:          store.CompoundKindAgentsPatch,
		Path:          "AGENTS.md",
		Action:        store.CompoundActionUpdate,
		Content:       compoundAgentsBody,
		ContentSHA256: compoundSHA(compoundAgentsBody),
	}})
	if err != nil {
		t.Fatal(err)
	}
	provider := &scriptedProvider{responses: []ChatResponse{{Content: string(payload)}}}
	runner, sessionID, _, _ := projectKnowledgeRunner(t, provider, true)
	runner.Compound = &store.CompoundStore{DB: runner.DB}

	if _, err := runner.StartCompound(context.Background(), sessionID, "rk-proj-tools", "compound"); err != nil {
		t.Fatalf("StartCompound: %v", err)
	}
	if len(provider.requests) != 1 {
		t.Fatalf("provider calls = %d, want 1", len(provider.requests))
	}
	if n := len(provider.requests[0].Tools); n != 0 {
		t.Fatalf("compound registered %d tools: %#v", n, provider.requests[0].Tools)
	}
}
