package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/agent/tools"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/fsroot"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

const maxToolRounds = 8

// ErrSessionBusy is returned when a different request key tries to start
// while a non-terminal agent run is already admitted for the session.
var ErrSessionBusy = errors.New("session has an active agent run")

// mutBarrier is implemented by *backup.Barrier (local to avoid import cycles).
type mutBarrier interface {
	Mutate(func() error) error
}

// RunAdmissions is the store surface used by Runner.Start.
type RunAdmissions interface {
	Admit(ctx context.Context, sessionID, requestKey, userMessage string, now time.Time) (store.RunAdmission, error)
	MarkRunning(ctx context.Context, runID string) error
	MarkDone(ctx context.Context, runID, status, errMsg string) error
	ByID(ctx context.Context, runID string) (domain.AgentRun, error)
	// BeginOrGet retained for tests that exercise low-level admission.
	BeginOrGet(ctx context.Context, sessionID, requestKey string) (string, bool, error)
}

type Runner struct {
	DB       *sql.DB
	DataDir  string
	Provider Provider
	Messages MessageStore
	Runs     RunAdmissions
	Sessions SessionReader
	Compound CompoundCreator
	Clock    clock.Clock
	Barrier  mutBarrier

	// bg tracks in-flight execute goroutines so tests can wait for completion.
	bg sync.WaitGroup
}

// CompoundCreator is the store surface used after a compound generation run.
type CompoundCreator interface {
	CreatePending(ctx context.Context, in store.CreateProposalInput) (domain.CompoundProposal, error)
}

func (r *Runner) Start(ctx context.Context, sessionID, requestKey, userMessage string) (string, error) {
	admission, err := r.Runs.Admit(ctx, sessionID, requestKey, userMessage, r.now())
	if err != nil {
		if errors.Is(err, store.ErrRunBusy) || errors.Is(err, store.ErrSessionBusy) {
			return "", ErrSessionBusy
		}
		return "", err
	}
	if admission.Existing {
		return admission.RunID, nil
	}
	r.bg.Add(1)
	go func(runID string) {
		defer r.bg.Done()
		r.finishRun(context.Background(), runID, r.execute)
	}(admission.RunID)
	return admission.RunID, nil
}

// Wait blocks until all background execute goroutines finish. Tests only.
func (r *Runner) Wait() { r.bg.Wait() }

func (r *Runner) finishRun(ctx context.Context, runID string, exec func(context.Context, string, domain.Session) error) {
	fail := func(cause error) {
		_ = r.Runs.MarkDone(ctx, runID, domain.AgentRunStatusFailed, cause.Error())
	}
	run, err := r.Runs.ByID(ctx, runID)
	if err != nil {
		fail(err)
		return
	}
	session, err := r.Sessions.Get(ctx, run.SessionID)
	if err != nil {
		fail(err)
		return
	}
	if err := r.Runs.MarkRunning(ctx, runID); err != nil {
		fail(err)
		return
	}
	if err := exec(ctx, runID, session); err != nil {
		fail(err)
		return
	}
	if err := r.Runs.MarkDone(ctx, runID, domain.AgentRunStatusCompleted, ""); err != nil {
		fail(err)
	}
}

func (r *Runner) execute(ctx context.Context, runID string, session domain.Session) error {
	history, err := r.Messages.List(ctx, session.ID)
	if err != nil {
		return err
	}
	parameters := make(map[string]any)
	if err := json.Unmarshal([]byte(session.ModelParametersJSON), &parameters); err != nil {
		return fmt.Errorf("decode model parameters: %w", err)
	}
	if parameters == nil {
		return errors.New("decode model parameters: expected JSON object")
	}
	for _, fixed := range []string{"model", "messages", "tools"} {
		delete(parameters, fixed)
	}
	messages := make([]ChatMessage, 0, len(history)+1)
	for _, message := range history {
		// Skip prior injected runtime/compound prompts so they never stack across runs.
		if message.Role == domain.MessageRoleSystem && isEphemeralSystem(message.Content) {
			continue
		}
		converted := ChatMessage{Role: message.Role, Content: message.Content}
		if message.ToolCallID != nil {
			converted.ToolCallID = *message.ToolCallID
		}
		if message.ToolCallsJSON != nil {
			if err := json.Unmarshal([]byte(*message.ToolCallsJSON), &converted.ToolCalls); err != nil {
				return fmt.Errorf("decode persisted tool calls: %w", err)
			}
		}
		messages = append(messages, converted)
	}
	// Ephemeral scoped system prompt (not persisted to Messages store).
	vaultID, projectID := "", ""
	if session.VaultID != nil {
		vaultID = *session.VaultID
	}
	if session.ProjectID != nil {
		projectID = *session.ProjectID
	}
	sections, err := BuildSessionPrompt(BuildPromptInput{
		DataDir:   r.DataDir,
		Home:      session.Home,
		VaultID:   vaultID,
		ProjectID: projectID,
	})
	if err != nil {
		return fmt.Errorf("build session prompt: %w", err)
	}
	systemMsg := ChatMessage{
		Role:    domain.MessageRoleSystem,
		Content: joinPromptSections(sections),
	}
	messages = append([]ChatMessage{systemMsg}, messages...)
	if r.Provider == nil {
		return fmt.Errorf("provider %q is unavailable", session.Provider)
	}
	// Future: session.tool_grants search_vault / search_global — not in slice 1
	var grants struct {
		WorkspaceFiles bool `json:"workspace_files"`
	}
	if err := json.Unmarshal([]byte(session.ToolGrantsJSON), &grants); err != nil {
		return fmt.Errorf("decode tool grants: %w", err)
	}
	req := ChatRequest{Model: session.ModelID, Messages: messages, Parameters: parameters}
	var workspace *tools.Workspace
	var root *fsroot.Root
	if grants.WorkspaceFiles {
		vaultID, projectID := "", ""
		if session.VaultID != nil {
			vaultID = *session.VaultID
		}
		if session.ProjectID != nil {
			projectID = *session.ProjectID
		}
		root, err = fsroot.Open(layout.SessionWorkspace(r.DataDir, session.Home, vaultID, projectID, session.ID))
		if err != nil {
			return err
		}
		defer root.Close()
		workspace = tools.NewWorkspace(root)
		workspace.Barrier = r.Barrier
		req.Tools = workspaceToolDefinitions
	}
	run := runID
	for round := 0; round < maxToolRounds; round++ {
		response, err := r.Provider.Chat(ctx, req)
		if err != nil {
			return err
		}
		if len(response.ToolCalls) == 0 {
			return r.Messages.Append(ctx, domain.Message{SessionID: session.ID, RunID: &run, Role: domain.MessageRoleAssistant,
				Content: response.Content, Status: domain.MessageStatusComplete, CreatedAt: r.now()})
		}
		callIDs := make(map[string]struct{}, len(response.ToolCalls))
		for _, call := range response.ToolCalls {
			if strings.TrimSpace(call.ID) == "" {
				return errors.New("provider returned an invalid tool call ID")
			}
			if _, exists := callIDs[call.ID]; exists {
				return errors.New("provider returned duplicate tool call IDs")
			}
			callIDs[call.ID] = struct{}{}
		}
		if workspace == nil {
			return errors.New("provider returned a tool call without a workspace grant")
		}
		encodedCalls, err := json.Marshal(response.ToolCalls)
		if err != nil {
			return err
		}
		callsJSON := string(encodedCalls)
		if err := r.Messages.Append(ctx, domain.Message{SessionID: session.ID, RunID: &run, Role: domain.MessageRoleAssistant,
			Content: response.Content, ToolCallsJSON: &callsJSON, Status: domain.MessageStatusComplete, CreatedAt: r.now()}); err != nil {
			return err
		}
		assistant := ChatMessage{Role: domain.MessageRoleAssistant, Content: response.Content, ToolCalls: response.ToolCalls}
		req.Messages = append(req.Messages, assistant)
		for _, call := range response.ToolCalls {
			result, toolErr := workspace.Execute(ctx, call.Name, json.RawMessage(call.Arguments))
			content := safeToolError()
			if toolErr == nil {
				encoded, marshalErr := json.Marshal(result)
				if marshalErr != nil {
					return marshalErr
				}
				content = string(encoded)
			}
			callID := call.ID
			if err := r.Messages.Append(ctx, domain.Message{SessionID: session.ID, RunID: &run, Role: domain.MessageRoleTool,
				Content: content, ToolCallID: &callID, Status: domain.MessageStatusComplete, CreatedAt: r.now()}); err != nil {
				return err
			}
			req.Messages = append(req.Messages, ChatMessage{Role: domain.MessageRoleTool, Content: content, ToolCallID: call.ID})
		}
	}
	return errors.New("tool round limit exceeded")
}

func safeToolError() string { return `{"error":"workspace tool request rejected"}` }

// joinPromptSections concatenates BuildSessionPrompt sections into one system message body.
// Runtime section content already starts with PA_RUNTIME_V1; other sections get ## {Name} headers.
func joinPromptSections(sections []PromptSection) string {
	var b strings.Builder
	for i, s := range sections {
		if i > 0 {
			b.WriteString("\n\n")
		}
		if s.Name == "runtime" {
			b.WriteString(s.Content)
			continue
		}
		b.WriteString("## ")
		// Title-case common names for readability (AGENTS, SYSTEM, …).
		switch s.Name {
		case "agents":
			b.WriteString("AGENTS")
		case "system":
			b.WriteString("SYSTEM")
		case "soul":
			b.WriteString("SOUL")
		case "lessons":
			b.WriteString("lessons")
		default:
			b.WriteString(s.Name)
		}
		b.WriteByte('\n')
		b.WriteString(s.Content)
	}
	return b.String()
}

func (r *Runner) now() time.Time {
	if r.Clock != nil {
		return r.Clock.Now().UTC()
	}
	return time.Now().UTC()
}
