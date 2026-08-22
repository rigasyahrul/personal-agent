package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

const compoundAgentsBody = "# Agents\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"

func compoundSHA(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func compoundRunner(t *testing.T, provider Provider, workspaceGranted bool) (*Runner, string, *store.RunStore, *store.CompoundStore) {
	t.Helper()
	runner, sessionID, runs, _ := toolRunner(t, provider, workspaceGranted)
	cs := &store.CompoundStore{DB: runner.DB, Clock: &clock.FakeClock{T: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)}}
	runner.Compound = cs
	return runner, sessionID, runs, cs
}

func TestCompoundRunCreatesPendingFromProviderJSON(t *testing.T) {
	payload, err := json.Marshal([]store.CompoundItem{{
		Kind:          store.CompoundKindAgentsPatch,
		Path:          "AGENTS.md",
		Action:        store.CompoundActionUpdate,
		Content:       compoundAgentsBody,
		ContentSHA256: "00", // model-supplied; server must overwrite
	}})
	if err != nil {
		t.Fatal(err)
	}
	provider := &scriptedProvider{responses: []ChatResponse{{Content: "```json\n" + string(payload) + "\n```"}}}
	runner, sessionID, runs, compounds := compoundRunner(t, provider, false)

	runID, err := runner.StartCompound(context.Background(), sessionID, "rk-gen", "summarize this work")
	if err != nil {
		t.Fatalf("StartCompound: %v", err)
	}
	run, lookupErr := runs.ByID(context.Background(), runID)
	if lookupErr != nil || run.Status != domain.AgentRunStatusCompleted {
		t.Fatalf("run = %#v, %v", run, lookupErr)
	}

	got, err := compounds.GetBySessionRequest(context.Background(), sessionID, "rk-gen")
	if err != nil {
		t.Fatalf("proposal: %v", err)
	}
	if got.Status != domain.CompoundStatusPending || got.SessionID != sessionID {
		t.Fatalf("proposal = %+v", got)
	}
	var items []store.CompoundItem
	if err := json.Unmarshal([]byte(got.ItemsJSON), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Kind != store.CompoundKindAgentsPatch || items[0].Content != compoundAgentsBody {
		t.Fatalf("items = %+v", items)
	}
	wantHash := compoundSHA(compoundAgentsBody)
	if items[0].ContentSHA256 != wantHash {
		t.Fatalf("content_sha256 = %q, want server hash %q", items[0].ContentSHA256, wantHash)
	}
	history, err := runner.Messages.List(context.Background(), sessionID)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range history {
		if m.Role == domain.MessageRoleSystem {
			t.Fatalf("ephemeral system must not be persisted: %q", m.Content)
		}
	}
	if _, err := os.Stat(filepath.Join(runner.DataDir, "files", "global", "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("compound wrote AGENTS.md before decide: %v", err)
	}
}

func TestCompoundRunDoesNotRegisterToolsWithWorkspaceGrant(t *testing.T) {
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
	runner, sessionID, _, _ := compoundRunner(t, provider, true)

	if _, err := runner.StartCompound(context.Background(), sessionID, "rk-tools", "compound"); err != nil {
		t.Fatalf("StartCompound: %v", err)
	}
	if len(provider.requests) != 1 {
		t.Fatalf("provider calls = %d, want 1", len(provider.requests))
	}
	if n := len(provider.requests[0].Tools); n != 0 {
		t.Fatalf("compound run registered %d tools: %#v", n, provider.requests[0].Tools)
	}
	var sawCompound bool
	for _, m := range provider.requests[0].Messages {
		if m.Role == domain.MessageRoleSystem && strings.HasPrefix(m.Content, "PA_COMPOUND_V1") {
			sawCompound = true
		}
	}
	if !sawCompound {
		t.Fatalf("missing PA_COMPOUND_V1 system message: %#v", provider.requests[0].Messages)
	}
}
