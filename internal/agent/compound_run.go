package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

const (
	compoundMarker         = "PA_COMPOUND_V1"
	compoundDefaultUserMsg = "Generate compound proposal items."
)

func isEphemeralSystem(content string) bool {
	return strings.HasPrefix(content, runtimeMarker) || strings.HasPrefix(content, compoundMarker)
}

// StartCompound admits the same one-active-run slot as chat, then runs a
// tools-disabled generation that CreatePending's parsed JSON items.
func (r *Runner) StartCompound(ctx context.Context, sessionID, requestKey, userMessage string) (string, error) {
	if strings.TrimSpace(userMessage) == "" {
		userMessage = compoundDefaultUserMsg
	}
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
	// Detach from the HTTP request context so cancel cannot skip MarkDone
	// (same as chat Start). Generate stays synchronous for the caller.
	r.finishRun(context.Background(), admission.RunID, r.executeCompound)
	return admission.RunID, nil
}

func (r *Runner) executeCompound(ctx context.Context, runID string, session domain.Session) error {
	if r.Compound == nil {
		return errors.New("compound store is unavailable")
	}
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

	messages := make([]ChatMessage, 0, len(history)+2)
	for _, message := range history {
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

	vaultID, projectID := sessionScopeIDs(session)
	sections, err := BuildSessionPrompt(BuildPromptInput{
		DataDir:   r.DataDir,
		Home:      session.Home,
		VaultID:   vaultID,
		ProjectID: projectID,
	})
	if err != nil {
		return fmt.Errorf("build session prompt: %w", err)
	}
	compoundSys, err := r.buildCompoundSystem(session)
	if err != nil {
		return err
	}
	messages = append([]ChatMessage{
		{Role: domain.MessageRoleSystem, Content: joinPromptSections(sections)},
		{Role: domain.MessageRoleSystem, Content: compoundSys},
	}, messages...)

	if r.Provider == nil {
		return fmt.Errorf("provider %q is unavailable", session.Provider)
	}
	// Tools DISABLED for compound generation — even if workspace_files is granted.
	req := ChatRequest{Model: session.ModelID, Messages: messages, Parameters: parameters}

	response, err := r.Provider.Chat(ctx, req)
	if err != nil {
		return err
	}
	if len(response.ToolCalls) != 0 {
		return errors.New("compound generation must not use tools")
	}

	run := runID
	if err := r.Messages.Append(ctx, domain.Message{
		SessionID: session.ID, RunID: &run, Role: domain.MessageRoleAssistant,
		Content: response.Content, Status: domain.MessageStatusComplete, CreatedAt: r.now(),
	}); err != nil {
		return err
	}

	items, err := ParseCompoundItemsFromAssistant(response.Content)
	if err != nil {
		return fmt.Errorf("parse compound items: %w", err)
	}
	stampCompoundItemHashes(items)

	scope, projectID, vaultID, err := compoundScopeFromSession(session)
	if err != nil {
		return err
	}
	runRow, err := r.Runs.ByID(ctx, runID)
	if err != nil {
		return err
	}
	_, err = r.Compound.CreatePending(ctx, store.CreateProposalInput{
		SessionID:  session.ID,
		RequestKey: runRow.RequestKey,
		Scope:      scope,
		ProjectID:  projectID,
		VaultID:    vaultID,
		Items:      items,
		Now:        r.now(),
	})
	if err != nil {
		return fmt.Errorf("create compound proposal: %w", err)
	}
	return nil
}

func (r *Runner) buildCompoundSystem(session domain.Session) (string, error) {
	vaultID, projectID := sessionScopeIDs(session)
	skill, _, err := LoadCompoundingSkill(r.DataDir, session.Home, vaultID, projectID)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(compoundMarker)
	b.WriteByte('\n')
	b.WriteString(skill)
	b.WriteString("\n\n## Output contract\n")
	b.WriteString("Emit only a JSON array of compound items (raw or in a ```json fence).\n")
	b.WriteString("Do not call tools. Do not write AGENTS.md or memory files yourself.\n")
	b.WriteString("The server will set content_sha256; you may omit or invent that field.\n")
	return b.String(), nil
}

func stampCompoundItemHashes(items []store.CompoundItem) {
	for i := range items {
		sum := sha256.Sum256([]byte(items[i].Content))
		items[i].ContentSHA256 = hex.EncodeToString(sum[:])
	}
}

func sessionScopeIDs(session domain.Session) (vaultID, projectID string) {
	if session.VaultID != nil {
		vaultID = *session.VaultID
	}
	if session.ProjectID != nil {
		projectID = *session.ProjectID
	}
	return vaultID, projectID
}

func compoundScopeFromSession(sess domain.Session) (domain.CompoundScope, string, string, error) {
	vaultID, projectID := sessionScopeIDs(sess)
	switch sess.Home {
	case layout.SessionHome("project"):
		return domain.CompoundScopeProject, projectID, vaultID, nil
	case layout.SessionHome("vault"):
		return domain.CompoundScopeVault, "", vaultID, nil
	case layout.SessionHome("global"):
		return domain.CompoundScopeGlobal, "", "", nil
	default:
		return "", "", "", fmt.Errorf("invalid session home %q", sess.Home)
	}
}
