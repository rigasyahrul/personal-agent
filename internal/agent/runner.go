package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/agent/tools"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/fsroot"
	"github.com/rigasyahrul/personal-agent/internal/layout"
)

const maxToolRounds = 8

// mutBarrier is implemented by *backup.Barrier (local to avoid import cycles).
type mutBarrier interface {
	Mutate(func() error) error
}

type Runner struct {
	DB       *sql.DB
	DataDir  string
	Provider Provider
	Messages MessageStore
	Runs     RunStore
	Sessions SessionReader
	Clock    clock.Clock
	Barrier  mutBarrier
}

func (r *Runner) Start(ctx context.Context, sessionID, requestKey, userMessage string) (runID string, err error) {
	runID, existing, err := r.Runs.BeginOrGet(ctx, sessionID, requestKey)
	if err != nil || existing {
		return runID, err
	}
	fail := func(cause error) (string, error) {
		if doneErr := r.Runs.MarkDone(ctx, runID, domain.AgentRunStatusFailed, cause.Error()); doneErr != nil {
			return runID, errors.Join(cause, fmt.Errorf("mark run failed: %w", doneErr))
		}
		return runID, cause
	}

	session, err := r.Sessions.Get(ctx, sessionID)
	if err != nil {
		return fail(err)
	}
	run := runID
	if err := r.Messages.Append(ctx, domain.Message{SessionID: sessionID, RunID: &run, Role: domain.MessageRoleUser,
		Content: userMessage, Status: domain.MessageStatusComplete, CreatedAt: r.now()}); err != nil {
		return fail(err)
	}
	if err := r.Runs.MarkRunning(ctx, runID); err != nil {
		return fail(err)
	}
	if err := r.execute(ctx, runID, session); err != nil {
		return fail(err)
	}
	if err := r.Runs.MarkDone(ctx, runID, domain.AgentRunStatusCompleted, ""); err != nil {
		return fail(err)
	}
	return runID, nil
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
	messages := make([]ChatMessage, 0, len(history))
	for _, message := range history {
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
	if r.Provider == nil {
		return fmt.Errorf("provider %q is unavailable", session.Provider)
	}
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

func (r *Runner) now() time.Time {
	if r.Clock != nil {
		return r.Clock.Now().UTC()
	}
	return time.Now().UTC()
}
