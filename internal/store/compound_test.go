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
