package store_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/store"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func shaHex(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func compoundItem(kind, path, action, content string) store.CompoundItem {
	return store.CompoundItem{
		Kind:          kind,
		Path:          path,
		Action:        action,
		Content:       content,
		ContentSHA256: shaHex(content),
	}
}

func memoryPair(detailPath, detail, lessons string) []store.CompoundItem {
	return []store.CompoundItem{
		compoundItem("memory_detail", detailPath, "create", detail),
		compoundItem("lessons_index_row", "memory/lessons.md", "update", lessons),
	}
}

func seedCompoundSession(t *testing.T, home string) (*store.CompoundStore, string) {
	t.Helper()
	db, _ := testutil.TempDB(t)
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	ts := now.Format(time.RFC3339Nano)

	if _, err := db.Exec(`
		INSERT INTO vaults(id,name,created_at,updated_at) VALUES('v1','Vault',?,?);
		INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES('p1','v1','Project',?,?);
	`, ts, ts, ts, ts); err != nil {
		t.Fatal(err)
	}

	var sessionID string
	switch home {
	case "project":
		sessionID = "sess-project"
		if _, err := db.Exec(`INSERT INTO sessions(id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at)
			VALUES(?,?,?,?, 'active','openai','gpt-test','{}','{}','S',?,?)`,
			sessionID, "project", "v1", "p1", ts, ts); err != nil {
			t.Fatal(err)
		}
	case "vault":
		sessionID = "sess-vault"
		if _, err := db.Exec(`INSERT INTO sessions(id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at)
			VALUES(?,?,?,NULL,'active','openai','gpt-test','{}','{}','S',?,?)`,
			sessionID, "vault", "v1", ts, ts); err != nil {
			t.Fatal(err)
		}
	case "global":
		sessionID = "sess-global"
		if _, err := db.Exec(`INSERT INTO sessions(id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at)
			VALUES(?,?,NULL,NULL,'active','openai','gpt-test','{}','{}','S',?,?)`,
			sessionID, "global", ts, ts); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("unknown home %q", home)
	}

	s := &store.CompoundStore{
		DB:    db,
		Clock: &clock.FakeClock{T: now},
	}
	return s, sessionID
}

func TestCreatePending_ProjectAgentsPatchHappy(t *testing.T) {
	s, sessionID := seedCompoundSession(t, "project")
	ctx := context.Background()
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	body := "# Agents\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"
	items := []store.CompoundItem{
		compoundItem("agents_patch", "AGENTS.md", "update", body),
	}

	got, err := s.CreatePending(ctx, store.CreateProposalInput{
		SessionID:  sessionID,
		RequestKey: "rk-1",
		Scope:      domain.CompoundScopeProject,
		ProjectID:  "p1",
		VaultID:    "v1",
		Items:      items,
		Now:        now,
	})
	if err != nil {
		t.Fatalf("CreatePending: %v", err)
	}
	if got.ID == "" || got.SessionID != sessionID || got.RequestKey != "rk-1" {
		t.Fatalf("ids: %+v", got)
	}
	if got.Scope != domain.CompoundScopeProject || got.ProjectID != "p1" || got.VaultID != "v1" {
		t.Fatalf("scope fields: %+v", got)
	}
	if got.Status != domain.CompoundStatusPending {
		t.Fatalf("status = %q", got.Status)
	}
	if !got.CreatedAt.Equal(now) {
		t.Fatalf("created_at = %v want %v", got.CreatedAt, now)
	}
	if got.DecidedAt != nil || got.FinishedAt != nil {
		t.Fatalf("decided/finished should be nil: %+v", got)
	}
	var decoded []store.CompoundItem
	if err := json.Unmarshal([]byte(got.ItemsJSON), &decoded); err != nil {
		t.Fatalf("items_json: %v", err)
	}
	if len(decoded) != 1 || decoded[0].Kind != "agents_patch" || decoded[0].Path != "AGENTS.md" {
		t.Fatalf("items: %+v", decoded)
	}
}

func TestCreatePending_VaultRejectsAgentsPatch(t *testing.T) {
	s, sessionID := seedCompoundSession(t, "vault")
	ctx := context.Background()
	items := []store.CompoundItem{
		compoundItem("agents_patch", "AGENTS.md", "update", "# no\n"),
	}
	_, err := s.CreatePending(ctx, store.CreateProposalInput{
		SessionID:  sessionID,
		RequestKey: "rk-vault",
		Scope:      domain.CompoundScopeVault,
		VaultID:    "v1",
		Items:      items,
		Now:        time.Now().UTC(),
	})
	if !errors.Is(err, store.ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestCreatePending_VaultAcceptsMemoryDetail(t *testing.T) {
	s, sessionID := seedCompoundSession(t, "vault")
	detailPath := "memory/20260822-1200-vault-lesson.md"
	got, err := s.CreatePending(context.Background(), store.CreateProposalInput{
		SessionID:  sessionID,
		RequestKey: "rk-vault-mem",
		Scope:      domain.CompoundScopeVault,
		VaultID:    "v1",
		Items:      memoryPair(detailPath, "# Vault lesson\n", "- [[memory/20260822-1200-vault-lesson]]\n"),
		Now:        time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreatePending vault memory: %v", err)
	}
	if got.Scope != domain.CompoundScopeVault || got.VaultID != "v1" || got.ProjectID != "" {
		t.Fatalf("vault proposal leaked project scope: %+v", got)
	}
	if got.Status != domain.CompoundStatusPending {
		t.Fatalf("status = %q", got.Status)
	}
	var decoded []store.CompoundItem
	if err := json.Unmarshal([]byte(got.ItemsJSON), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 2 || decoded[0].Path != detailPath || decoded[1].Path != "memory/lessons.md" {
		t.Fatalf("items: %+v", decoded)
	}
}

func TestCreatePending_GlobalAgentsAndMemory(t *testing.T) {
	s, sessionID := seedCompoundSession(t, "global")
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	agents := "# Global agents\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"
	agentsGot, err := s.CreatePending(context.Background(), store.CreateProposalInput{
		SessionID:  sessionID,
		RequestKey: "rk-global-agents",
		Scope:      domain.CompoundScopeGlobal,
		Items:      []store.CompoundItem{compoundItem("agents_patch", "AGENTS.md", "update", agents)},
		Now:        now,
	})
	if err != nil {
		t.Fatalf("CreatePending global agents: %v", err)
	}
	if agentsGot.Scope != domain.CompoundScopeGlobal || agentsGot.ProjectID != "" || agentsGot.VaultID != "" {
		t.Fatalf("global agents proposal bound a vault/project: %+v", agentsGot)
	}

	detailPath := "memory/20260822-1200-global-lesson.md"
	memGot, err := s.CreatePending(context.Background(), store.CreateProposalInput{
		SessionID:  sessionID,
		RequestKey: "rk-global-mem",
		Scope:      domain.CompoundScopeGlobal,
		Items:      memoryPair(detailPath, "# Global lesson\n", "- [[memory/20260822-1200-global-lesson]]\n"),
		Now:        now,
	})
	if err != nil {
		t.Fatalf("CreatePending global memory: %v", err)
	}
	if memGot.Scope != domain.CompoundScopeGlobal || memGot.ProjectID != "" || memGot.VaultID != "" {
		t.Fatalf("global memory proposal bound a vault/project: %+v", memGot)
	}
}

func TestCreatePending_ProjectMemoryKeepsProjectScope(t *testing.T) {
	s, sessionID := seedCompoundSession(t, "project")
	detailPath := "memory/20260822-1200-project-lesson.md"
	got, err := s.CreatePending(context.Background(), store.CreateProposalInput{
		SessionID:  sessionID,
		RequestKey: "rk-proj-mem",
		Scope:      domain.CompoundScopeProject,
		ProjectID:  "p1",
		VaultID:    "v1",
		Items:      memoryPair(detailPath, "# Project lesson\n", "- [[memory/20260822-1200-project-lesson]]\n"),
		Now:        time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreatePending project memory: %v", err)
	}
	if got.Scope != domain.CompoundScopeProject || got.ProjectID != "p1" || got.VaultID != "v1" {
		t.Fatalf("project memory must stay bound to project root ids: %+v", got)
	}
}

func TestCreatePending_ProjectRejectsEscapingMemoryPath(t *testing.T) {
	s, sessionID := seedCompoundSession(t, "project")
	for _, path := range []string{
		"memory/../AGENTS.md",
		"/tmp/x.md",
		"memory/../../vaults/v1/memory/20260822-1200-escape.md",
	} {
		_, err := s.CreatePending(context.Background(), store.CreateProposalInput{
			SessionID:  sessionID,
			RequestKey: "rk-escape-" + path,
			Scope:      domain.CompoundScopeProject,
			ProjectID:  "p1",
			VaultID:    "v1",
			Items:      []store.CompoundItem{compoundItem("memory_detail", path, "create", "# no\n")},
			Now:        time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC),
		})
		if !errors.Is(err, store.ErrValidation) {
			t.Fatalf("path %q: err = %v, want ErrValidation", path, err)
		}
	}
}

func TestValidateCompoundItems_MemoryDetailRequiresLessonsIndexRow(t *testing.T) {
	detail := compoundItem("memory_detail", "memory/20260822-1200-example.md", "create", "# Detail\n")
	err := store.ValidateCompoundItems(domain.CompoundScopeProject, []store.CompoundItem{detail})
	if !errors.Is(err, store.ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}

	// With lessons row — ok
	lessons := compoundItem("lessons_index_row", "memory/lessons.md", "update", "- [[memory/20260822-1200-example]]\n")
	if err := store.ValidateCompoundItems(domain.CompoundScopeProject, []store.CompoundItem{detail, lessons}); err != nil {
		t.Fatalf("with lessons row: %v", err)
	}
}

func TestValidateCompoundItems_PathEscapeAndForbidden(t *testing.T) {
	cases := []struct {
		name string
		item store.CompoundItem
	}{
		{"source", compoundItem("memory_detail", "source/guide.md", "create", "x")},
		{"agents_dir", compoundItem("memory_detail", ".agents/skills/x.md", "create", "x")},
		{"soul", compoundItem("agents_patch", "SOUL.md", "update", "x")},
		{"system", compoundItem("agents_patch", "SYSTEM.md", "update", "x")},
		{"dotdot", compoundItem("memory_detail", "memory/../AGENTS.md", "create", "x")},
		{"abs", compoundItem("memory_detail", "/tmp/x.md", "create", "x")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Fix sha after any content
			tc.item.ContentSHA256 = shaHex(tc.item.Content)
			err := store.ValidateCompoundItems(domain.CompoundScopeProject, []store.CompoundItem{tc.item})
			if !errors.Is(err, store.ErrValidation) {
				t.Fatalf("err = %v, want ErrValidation", err)
			}
		})
	}
}

func TestValidateCompoundItems_BadSHA256(t *testing.T) {
	item := store.CompoundItem{
		Kind:          "agents_patch",
		Path:          "AGENTS.md",
		Action:        "update",
		Content:       "# hi\n",
		ContentSHA256: "deadbeef",
	}
	if err := store.ValidateCompoundItems(domain.CompoundScopeProject, []store.CompoundItem{item}); !errors.Is(err, store.ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestCreatePending_IdempotentSameKeyAndItems(t *testing.T) {
	s, sessionID := seedCompoundSession(t, "project")
	ctx := context.Background()
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	items := []store.CompoundItem{
		compoundItem("agents_patch", "AGENTS.md", "update", "# A\n"),
	}
	in := store.CreateProposalInput{
		SessionID:  sessionID,
		RequestKey: "rk-idem",
		Scope:      domain.CompoundScopeProject,
		ProjectID:  "p1",
		VaultID:    "v1",
		Items:      items,
		Now:        now,
	}
	first, err := s.CreatePending(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.CreatePending(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("idempotent id mismatch: %q vs %q", first.ID, second.ID)
	}
	if first.ItemsJSON != second.ItemsJSON {
		t.Fatalf("items_json mismatch")
	}
}

func TestCreatePending_SameKeyDifferentItemsConflict(t *testing.T) {
	s, sessionID := seedCompoundSession(t, "project")
	ctx := context.Background()
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	base := store.CreateProposalInput{
		SessionID:  sessionID,
		RequestKey: "rk-conflict",
		Scope:      domain.CompoundScopeProject,
		ProjectID:  "p1",
		VaultID:    "v1",
		Now:        now,
	}
	first := base
	first.Items = []store.CompoundItem{compoundItem("agents_patch", "AGENTS.md", "update", "# one\n")}
	if _, err := s.CreatePending(ctx, first); err != nil {
		t.Fatal(err)
	}
	second := base
	second.Items = []store.CompoundItem{compoundItem("agents_patch", "AGENTS.md", "update", "# two\n")}
	_, err := s.CreatePending(ctx, second)
	if !errors.Is(err, store.ErrConflict) {
		t.Fatalf("err = %v, want ErrConflict", err)
	}
}

func TestValidateCompoundItems_EmptyRejected(t *testing.T) {
	if err := store.ValidateCompoundItems(domain.CompoundScopeProject, nil); !errors.Is(err, store.ErrValidation) {
		t.Fatalf("nil items: %v", err)
	}
	if err := store.ValidateCompoundItems(domain.CompoundScopeProject, []store.CompoundItem{}); !errors.Is(err, store.ErrValidation) {
		t.Fatalf("empty items: %v", err)
	}
}

func TestValidateCompoundItems_MemoryDetailPathRegex(t *testing.T) {
	lessons := compoundItem("lessons_index_row", "memory/lessons.md", "update", "- row\n")
	badPaths := []string{
		"memory/lessons.md", // wrong kind for this path as detail
		"memory/2026-08-22-bad.md",
		"memory/20260822-1200.md",
		"memory/20260822-1200-Bad.md",
		"memory/subdir/20260822-1200-ok.md",
	}
	for _, p := range badPaths {
		detail := compoundItem("memory_detail", p, "create", "# x\n")
		// When path is lessons.md with kind memory_detail, still fail regex / pairing
		err := store.ValidateCompoundItems(domain.CompoundScopeProject, []store.CompoundItem{detail, lessons})
		if !errors.Is(err, store.ErrValidation) {
			t.Fatalf("path %q: err = %v, want ErrValidation", p, err)
		}
	}

	ok := compoundItem("memory_detail", "memory/20260822-1200-ok-slug.md", "create", "# x\n")
	if err := store.ValidateCompoundItems(domain.CompoundScopeProject, []store.CompoundItem{ok, lessons}); err != nil {
		t.Fatalf("valid detail path: %v", err)
	}
}

func TestValidateCompoundItems_SizeAndCountCaps(t *testing.T) {
	// max 20 items
	items := make([]store.CompoundItem, 21)
	for i := range items {
		body := fmt.Sprintf("# %d\n", i)
		// use agents_patch only for first; but vault-forbidden not relevant
		items[i] = compoundItem("agents_patch", "AGENTS.md", "update", body)
	}
	if err := store.ValidateCompoundItems(domain.CompoundScopeProject, items); !errors.Is(err, store.ErrValidation) {
		t.Fatalf("21 items: %v", err)
	}

	// 256KiB content cap
	big := strings.Repeat("x", 256*1024+1)
	item := compoundItem("agents_patch", "AGENTS.md", "update", big)
	if err := store.ValidateCompoundItems(domain.CompoundScopeProject, []store.CompoundItem{item}); !errors.Is(err, store.ErrValidation) {
		t.Fatalf("oversized: %v", err)
	}
}

func mustCreatePending(t *testing.T, s *store.CompoundStore, sessionID, key, body string) domain.CompoundProposal {
	t.Helper()
	got, err := s.CreatePending(context.Background(), store.CreateProposalInput{
		SessionID:  sessionID,
		RequestKey: key,
		Scope:      domain.CompoundScopeProject,
		ProjectID:  "p1",
		VaultID:    "v1",
		Items:      []store.CompoundItem{compoundItem("agents_patch", "AGENTS.md", "update", body)},
		Now:        time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreatePending: %v", err)
	}
	return got
}

func TestDecide_RejectSetsBothTimestamps(t *testing.T) {
	s, sessionID := seedCompoundSession(t, "project")
	pending := mustCreatePending(t, s, sessionID, "rk-reject", "# pending\n")
	decided := time.Date(2026, 8, 22, 13, 0, 0, 0, time.UTC)

	got, err := s.Decide(context.Background(), store.DecideInput{
		ProposalID: pending.ID,
		Decision:   "reject",
		Now:        decided,
	})
	if err != nil {
		t.Fatalf("Decide reject: %v", err)
	}
	if got.Status != domain.CompoundStatusRejected {
		t.Fatalf("status = %q", got.Status)
	}
	if got.DecidedAt == nil || !got.DecidedAt.Equal(decided) {
		t.Fatalf("decided_at = %v want %v", got.DecidedAt, decided)
	}
	if got.FinishedAt == nil || !got.FinishedAt.Equal(decided) {
		t.Fatalf("finished_at = %v want %v", got.FinishedAt, decided)
	}
}

func TestDecide_ApproveSetsDecidedAtOnly(t *testing.T) {
	s, sessionID := seedCompoundSession(t, "project")
	pending := mustCreatePending(t, s, sessionID, "rk-approve", "# pending\n")
	decided := time.Date(2026, 8, 22, 13, 30, 0, 0, time.UTC)

	got, err := s.Decide(context.Background(), store.DecideInput{
		ProposalID: pending.ID,
		Decision:   "approve",
		Now:        decided,
	})
	if err != nil {
		t.Fatalf("Decide approve: %v", err)
	}
	if got.Status != domain.CompoundStatusApproved {
		t.Fatalf("status = %q", got.Status)
	}
	if got.DecidedAt == nil || !got.DecidedAt.Equal(decided) {
		t.Fatalf("decided_at = %v want %v", got.DecidedAt, decided)
	}
	if got.FinishedAt != nil {
		t.Fatalf("finished_at should be nil until publish, got %v", got.FinishedAt)
	}
}

func TestMarkFinished_SetsFinishedAt(t *testing.T) {
	s, sessionID := seedCompoundSession(t, "project")
	pending := mustCreatePending(t, s, sessionID, "rk-finish", "# pending\n")
	decided := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
	approved, err := s.Decide(context.Background(), store.DecideInput{
		ProposalID: pending.ID,
		Decision:   "approve",
		Now:        decided,
	})
	if err != nil {
		t.Fatal(err)
	}
	finished := decided.Add(time.Minute)
	if err := s.MarkFinished(context.Background(), approved.ID, string(domain.CompoundStatusApproved), "", finished); err != nil {
		t.Fatalf("MarkFinished approved: %v", err)
	}
	got, err := s.Get(context.Background(), approved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.CompoundStatusApproved {
		t.Fatalf("status = %q want approved", got.Status)
	}
	if got.FinishedAt == nil || !got.FinishedAt.Equal(finished) {
		t.Fatalf("finished_at = %v want %v", got.FinishedAt, finished)
	}
	if got.Error != "" {
		t.Fatalf("error = %q", got.Error)
	}

	// Failed path: second proposal.
	pending2 := mustCreatePending(t, s, sessionID, "rk-fail", "# other\n")
	approved2, err := s.Decide(context.Background(), store.DecideInput{
		ProposalID: pending2.ID,
		Decision:   "approve",
		Now:        decided,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkFinished(context.Background(), approved2.ID, string(domain.CompoundStatusFailed), "publish boom", finished); err != nil {
		t.Fatalf("MarkFinished failed: %v", err)
	}
	failed, err := s.Get(context.Background(), approved2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != domain.CompoundStatusFailed {
		t.Fatalf("status = %q want failed", failed.Status)
	}
	if failed.Error != "publish boom" {
		t.Fatalf("error = %q", failed.Error)
	}
	if failed.FinishedAt == nil || !failed.FinishedAt.Equal(finished) {
		t.Fatalf("failed finished_at = %v", failed.FinishedAt)
	}
}

func TestDecide_ApproveRecomputesSHA256(t *testing.T) {
	s, sessionID := seedCompoundSession(t, "project")
	pending := mustCreatePending(t, s, sessionID, "rk-sha", "# old\n")
	edited := store.CompoundItem{
		Kind:          "agents_patch",
		Path:          "AGENTS.md",
		Action:        "update",
		Content:       "# edited body\n",
		ContentSHA256: "deadbeef", // advisory; server must overwrite
	}
	got, err := s.Decide(context.Background(), store.DecideInput{
		ProposalID: pending.ID,
		Decision:   "approve",
		Items:      []store.CompoundItem{edited},
		Now:        time.Date(2026, 8, 22, 15, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}
	var decoded []store.CompoundItem
	if err := json.Unmarshal([]byte(got.ItemsJSON), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 1 || decoded[0].Content != "# edited body\n" {
		t.Fatalf("items: %+v", decoded)
	}
	if decoded[0].ContentSHA256 != shaHex("# edited body\n") {
		t.Fatalf("sha = %q want recomputed", decoded[0].ContentSHA256)
	}
}

func TestDecide_ApproveValidatesFinalItems(t *testing.T) {
	s, sessionID := seedCompoundSession(t, "project")
	pending := mustCreatePending(t, s, sessionID, "rk-bad-edit", "# ok\n")
	_, err := s.Decide(context.Background(), store.DecideInput{
		ProposalID: pending.ID,
		Decision:   "approve",
		Items: []store.CompoundItem{
			compoundItem("memory_detail", "source/owned.md", "create", "# no\n"),
		},
		Now: time.Now().UTC(),
	})
	if !errors.Is(err, store.ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
	got, err := s.Get(context.Background(), pending.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.CompoundStatusPending {
		t.Fatalf("status should stay pending after failed decide, got %q", got.Status)
	}
}

func TestDecide_IdempotentSameDecision(t *testing.T) {
	s, sessionID := seedCompoundSession(t, "project")
	pending := mustCreatePending(t, s, sessionID, "rk-idemp", "# a\n")
	now := time.Date(2026, 8, 22, 16, 0, 0, 0, time.UTC)
	first, err := s.Decide(context.Background(), store.DecideInput{
		ProposalID: pending.ID,
		Decision:   "reject",
		Now:        now,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Decide(context.Background(), store.DecideInput{
		ProposalID: pending.ID,
		Decision:   "reject",
		Now:        now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("idempotent reject: %v", err)
	}
	if first.ID != second.ID || second.Status != domain.CompoundStatusRejected {
		t.Fatalf("second = %+v", second)
	}
	if second.DecidedAt == nil || !second.DecidedAt.Equal(now) {
		t.Fatalf("idempotent must not change decided_at: %v", second.DecidedAt)
	}
}

func TestDecide_ConflictingDecision(t *testing.T) {
	s, sessionID := seedCompoundSession(t, "project")
	pending := mustCreatePending(t, s, sessionID, "rk-conflict-dec", "# a\n")
	if _, err := s.Decide(context.Background(), store.DecideInput{
		ProposalID: pending.ID,
		Decision:   "reject",
		Now:        time.Date(2026, 8, 22, 17, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}
	_, err := s.Decide(context.Background(), store.DecideInput{
		ProposalID: pending.ID,
		Decision:   "approve",
		Now:        time.Date(2026, 8, 22, 17, 5, 0, 0, time.UTC),
	})
	if !errors.Is(err, store.ErrConflict) {
		t.Fatalf("err = %v, want ErrConflict", err)
	}
}

func TestGet_AndGetBySessionRequest(t *testing.T) {
	s, sessionID := seedCompoundSession(t, "project")
	pending := mustCreatePending(t, s, sessionID, "rk-get", "# get\n")
	byID, err := s.Get(context.Background(), pending.ID)
	if err != nil {
		t.Fatal(err)
	}
	if byID.ID != pending.ID || byID.RequestKey != "rk-get" {
		t.Fatalf("Get: %+v", byID)
	}
	byKey, err := s.GetBySessionRequest(context.Background(), sessionID, "rk-get")
	if err != nil {
		t.Fatal(err)
	}
	if byKey.ID != pending.ID {
		t.Fatalf("GetBySessionRequest: %+v", byKey)
	}
	if _, err := s.Get(context.Background(), "missing"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing Get: %v", err)
	}
}
