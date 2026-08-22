package compound_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/compound"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/store"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func shaHex(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func item(kind, path, action, content string) store.CompoundItem {
	return store.CompoundItem{
		Kind:          kind,
		Path:          path,
		Action:        action,
		Content:       content,
		ContentSHA256: shaHex(content),
	}
}

func seedProjectPublisher(t *testing.T) (*compound.Publisher, *store.CompoundStore, string, string, string) {
	t.Helper()
	db, dataDir := testutil.TempDB(t)
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	c := &clock.FakeClock{T: now}
	ctx := context.Background()

	vs := store.NewVaultStore(db, dataDir, c)
	v, err := vs.Create(ctx, "Vault")
	if err != nil {
		t.Fatal(err)
	}
	ps := store.NewProjectStore(db, dataDir, c)
	p, err := ps.Create(ctx, "Project", v.ID)
	if err != nil {
		t.Fatal(err)
	}
	ts := now.Format(time.RFC3339Nano)
	sessionID := "sess-project"
	if _, err := db.Exec(`INSERT INTO sessions(id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at)
		VALUES(?,?,?,?, 'active','openai','gpt-test','{}','{}','S',?,?)`,
		sessionID, "project", v.ID, p.ID, ts, ts); err != nil {
		t.Fatal(err)
	}

	cs := &store.CompoundStore{DB: db, Clock: c}
	pub := &compound.Publisher{
		DataDir: dataDir,
		DB:      db,
		Clock:   c,
	}
	return pub, cs, sessionID, dataDir, p.ID
}

func TestPublishApproved_AgentsPatchWritesFile(t *testing.T) {
	pub, cs, sessionID, dataDir, projectID := seedProjectPublisher(t)
	// Recreate with real project/vault ids from the seeded session.
	db := pub.DB
	var vaultID string
	if err := db.QueryRow(`SELECT vault_id FROM sessions WHERE id=?`, sessionID).Scan(&vaultID); err != nil {
		t.Fatal(err)
	}
	body := "# Agents\n\nrule: always test first\n\n## Memory\n- Lesson index: [[memory/lessons|lessons.md]] — scan titles.\n"
	pending, err := cs.CreatePending(context.Background(), store.CreateProposalInput{
		SessionID:  sessionID,
		RequestKey: "rk-agents",
		Scope:      domain.CompoundScopeProject,
		ProjectID:  projectID,
		VaultID:    vaultID,
		Items:      []store.CompoundItem{item("agents_patch", "AGENTS.md", "update", body)},
		Now:        time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	approved, err := cs.Decide(context.Background(), store.DecideInput{
		ProposalID: pending.ID,
		Decision:   "approve",
		Now:        time.Date(2026, 8, 22, 12, 1, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := pub.PublishApproved(context.Background(), approved); err != nil {
		t.Fatalf("PublishApproved: %v", err)
	}
	root := layout.ProjectRoot(dataDir, vaultID, projectID)
	got, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != body {
		t.Fatalf("AGENTS.md =\n%s", got)
	}
}

func TestPublishApproved_StripsMemoryBlockError(t *testing.T) {
	pub, cs, sessionID, dataDir, projectID := seedProjectPublisher(t)
	var vaultID string
	if err := pub.DB.QueryRow(`SELECT vault_id FROM sessions WHERE id=?`, sessionID).Scan(&vaultID); err != nil {
		t.Fatal(err)
	}
	root := layout.ProjectRoot(dataDir, vaultID, projectID)
	before, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}

	stripped := "# Agents\n\nno memory section here\n"
	pending, err := cs.CreatePending(context.Background(), store.CreateProposalInput{
		SessionID:  sessionID,
		RequestKey: "rk-strip",
		Scope:      domain.CompoundScopeProject,
		ProjectID:  projectID,
		VaultID:    vaultID,
		Items:      []store.CompoundItem{item("agents_patch", "AGENTS.md", "update", stripped)},
		Now:        time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	approved, err := cs.Decide(context.Background(), store.DecideInput{
		ProposalID: pending.ID,
		Decision:   "approve",
		Now:        time.Date(2026, 8, 22, 12, 1, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := pub.PublishApproved(context.Background(), approved); err == nil {
		t.Fatal("expected error when AGENTS strips Memory pointer")
	}
	after, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("AGENTS.md mutated on failed publish:\n%s", after)
	}
}

func TestPublishApproved_MemoryDetailAndLessonsLand(t *testing.T) {
	pub, cs, sessionID, dataDir, projectID := seedProjectPublisher(t)
	var vaultID string
	if err := pub.DB.QueryRow(`SELECT vault_id FROM sessions WHERE id=?`, sessionID).Scan(&vaultID); err != nil {
		t.Fatal(err)
	}
	detailPath := "memory/20260822-1200-first-lesson.md"
	detail := "# First lesson\n\nDo the thing.\n"
	lessons := "- [[memory/20260822-1200-first-lesson|First lesson]] — do the thing\n"
	pending, err := cs.CreatePending(context.Background(), store.CreateProposalInput{
		SessionID:  sessionID,
		RequestKey: "rk-mem",
		Scope:      domain.CompoundScopeProject,
		ProjectID:  projectID,
		VaultID:    vaultID,
		Items: []store.CompoundItem{
			item("memory_detail", detailPath, "create", detail),
			item("lessons_index_row", "memory/lessons.md", "update", lessons),
		},
		Now: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	approved, err := cs.Decide(context.Background(), store.DecideInput{
		ProposalID: pending.ID,
		Decision:   "approve",
		Now:        time.Date(2026, 8, 22, 12, 1, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := pub.PublishApproved(context.Background(), approved); err != nil {
		t.Fatalf("PublishApproved: %v", err)
	}

	root := layout.ProjectRoot(dataDir, vaultID, projectID)
	gotDetail, err := os.ReadFile(filepath.Join(root, detailPath))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotDetail) != detail {
		t.Fatalf("detail =\n%s", gotDetail)
	}
	gotLessons, err := os.ReadFile(layout.LessonsPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gotLessons), "20260822-1200-first-lesson") {
		t.Fatalf("lessons missing detail row:\n%s", gotLessons)
	}
	// Seed header must survive merge.
	if !strings.Contains(string(gotLessons), "# Lessons") {
		t.Fatalf("lessons header lost:\n%s", gotLessons)
	}
}

func TestPublishApproved_SecondCompoundMergesLessons(t *testing.T) {
	pub, cs, sessionID, dataDir, projectID := seedProjectPublisher(t)
	var vaultID string
	if err := pub.DB.QueryRow(`SELECT vault_id FROM sessions WHERE id=?`, sessionID).Scan(&vaultID); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)

	firstDetail := "memory/20260822-1200-alpha.md"
	pending1, err := cs.CreatePending(context.Background(), store.CreateProposalInput{
		SessionID:  sessionID,
		RequestKey: "rk-a",
		Scope:      domain.CompoundScopeProject,
		ProjectID:  projectID,
		VaultID:    vaultID,
		Items: []store.CompoundItem{
			item("memory_detail", firstDetail, "create", "# Alpha\n"),
			item("lessons_index_row", "memory/lessons.md", "update", "- [[memory/20260822-1200-alpha|Alpha]]\n"),
		},
		Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	a1, err := cs.Decide(context.Background(), store.DecideInput{ProposalID: pending1.ID, Decision: "approve", Now: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if err := pub.PublishApproved(context.Background(), a1); err != nil {
		t.Fatal(err)
	}

	secondDetail := "memory/20260822-1300-beta.md"
	pending2, err := cs.CreatePending(context.Background(), store.CreateProposalInput{
		SessionID:  sessionID,
		RequestKey: "rk-b",
		Scope:      domain.CompoundScopeProject,
		ProjectID:  projectID,
		VaultID:    vaultID,
		Items: []store.CompoundItem{
			item("memory_detail", secondDetail, "create", "# Beta\n"),
			item("lessons_index_row", "memory/lessons.md", "update", "- [[memory/20260822-1300-beta|Beta]]\n"),
		},
		Now: now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	a2, err := cs.Decide(context.Background(), store.DecideInput{ProposalID: pending2.ID, Decision: "approve", Now: now.Add(3 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if err := pub.PublishApproved(context.Background(), a2); err != nil {
		t.Fatal(err)
	}

	root := layout.ProjectRoot(dataDir, vaultID, projectID)
	got, err := os.ReadFile(layout.LessonsPath(root))
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "20260822-1200-alpha") {
		t.Fatalf("lost alpha row:\n%s", text)
	}
	if !strings.Contains(text, "20260822-1300-beta") {
		t.Fatalf("missing beta row:\n%s", text)
	}
}

func TestValidateAgentsMemoryPointer(t *testing.T) {
	ok := "# X\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"
	if err := compound.ValidateAgentsMemoryPointer(ok); err != nil {
		t.Fatalf("ok: %v", err)
	}
	if err := compound.ValidateAgentsMemoryPointer("# X\nno pointer\n"); err == nil {
		t.Fatal("expected missing Memory to fail")
	}
	if err := compound.ValidateAgentsMemoryPointer("## Memory\nno wikilink\n"); err == nil {
		t.Fatal("expected missing lessons wikilink to fail")
	}
}
