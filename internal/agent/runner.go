package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
)

type Runner struct {
	DB       *sql.DB
	DataDir  string
	Provider Provider
	Messages MessageStore
	Runs     RunStore
	Sessions SessionReader
	Clock    clock.Clock
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

	run := runID
	if err := r.Messages.Append(ctx, domain.Message{SessionID: sessionID, RunID: &run, Role: domain.MessageRoleUser,
		Content: userMessage, Status: domain.MessageStatusComplete, CreatedAt: r.now()}); err != nil {
		return fail(err)
	}
	if err := r.Runs.MarkRunning(ctx, runID); err != nil {
		return fail(err)
	}
	session, err := r.Sessions.Get(ctx, sessionID)
	if err != nil {
		return fail(err)
	}
	if err := r.execute(ctx, runID, session); err != nil {
		return fail(err)
	}
	if err := r.Runs.MarkDone(ctx, runID, domain.AgentRunStatusCompleted, ""); err != nil {
		return runID, err
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
	for _, fixed := range []string{"model", "messages", "tools"} {
		delete(parameters, fixed)
	}
	messages := make([]ChatMessage, 0, len(history))
	for _, message := range history {
		messages = append(messages, ChatMessage{Role: message.Role, Content: message.Content})
	}
	if r.Provider == nil {
		return fmt.Errorf("provider %q is unavailable", session.Provider)
	}
	response, err := r.Provider.Chat(ctx, ChatRequest{Model: session.ModelID, Messages: messages, Parameters: parameters})
	if err != nil {
		return err
	}
	run := runID
	return r.Messages.Append(ctx, domain.Message{SessionID: session.ID, RunID: &run, Role: domain.MessageRoleAssistant,
		Content: response.Content, Status: domain.MessageStatusComplete, CreatedAt: r.now()})
}

func (r *Runner) now() time.Time {
	if r.Clock != nil {
		return r.Clock.Now().UTC()
	}
	return time.Now().UTC()
}
