package compound_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
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

type knowledgeRow struct {
	id, kind, rel, projectID, vaultID string
	isGlobal                          int
}

func loadKnowledgeByPath(t *testing.T, db *sql.DB, rel string) knowledgeRow {
	t.Helper()
	var r knowledgeRow
	var project, vault sql.NullString
	err := db.QueryRow(`SELECT id, kind, relative_path, coalesce(project_id,''), coalesce(vault_id,''), is_global FROM knowledge_notes WHERE relative_path=?`, rel).
		Scan(&r.id, &r.kind, &r.rel, &project, &vault, &r.isGlobal)
	if err != nil {
		t.Fatalf("knowledge_notes %s: %v", rel, err)
	}
	r.projectID = project.String
	r.vaultID = vault.String
	return r
}

type publishedLink struct {
	raw, toPath, toNote string
}

func loadPublishedLinks(t *testing.T, db *sql.DB, fromID string) []publishedLink {
	t.Helper()
	rows, err := db.Query(`SELECT raw_target, to_path, coalesce(to_note_id,'') FROM note_links WHERE from_note_id=? ORDER BY to_path`, fromID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []publishedLink
	for rows.Next() {
		var l publishedLink
		if err := rows.Scan(&l.raw, &l.toPath, &l.toNote); err != nil {
			t.Fatal(err)
		}
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func approveAndPublish(t *testing.T, pub *compound.Publisher, cs *store.CompoundStore, in store.CreateProposalInput, decideAt time.Time) {
	t.Helper()
	pending, err := cs.CreatePending(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	approved, err := cs.Decide(context.Background(), store.DecideInput{
		ProposalID: pending.ID,
		Decision:   "approve",
		Now:        decideAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := pub.PublishApproved(context.Background(), approved); err != nil {
		t.Fatalf("PublishApproved: %v", err)
	}
}

func TestPublishApproved_ReindexesKindsPathsAndWikilinks(t *testing.T) {
	pub, cs, sessionID, _, projectID := seedProjectPublisher(t)
	var vaultID string
	if err := pub.DB.QueryRow(`SELECT vault_id FROM sessions WHERE id=?`, sessionID).Scan(&vaultID); err != nil {
		t.Fatal(err)
	}
	agents := "# Agents\n\nrule: always test first\n\n## Memory\n- Lesson index: [[memory/lessons|lessons.md]] — scan titles.\n"
	detailPath := "memory/20260822-1200-first-lesson.md"
	detail := "# First lesson\n\nSee [[AGENTS]] and [[memory/lessons]].\n"
	lessons := "- [[memory/20260822-1200-first-lesson|First lesson]] — do the thing\n"
	approveAndPublish(t, pub, cs, store.CreateProposalInput{
		SessionID:  sessionID,
		RequestKey: "rk-reindex",
		Scope:      domain.CompoundScopeProject,
		ProjectID:  projectID,
		VaultID:    vaultID,
		Items: []store.CompoundItem{
			item("agents_patch", "AGENTS.md", "update", agents),
			item("memory_detail", detailPath, "create", detail),
			item("lessons_index_row", "memory/lessons.md", "update", lessons),
		},
		Now: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC),
	}, time.Date(2026, 8, 22, 12, 1, 0, 0, time.UTC))

	agentsNote := loadKnowledgeByPath(t, pub.DB, "AGENTS.md")
	if agentsNote.kind != "agents" || agentsNote.projectID != projectID || agentsNote.vaultID != "" || agentsNote.isGlobal != 0 {
		t.Fatalf("AGENTS.md row = %+v", agentsNote)
	}
	detailNote := loadKnowledgeByPath(t, pub.DB, detailPath)
	if detailNote.kind != "memory_detail" || detailNote.projectID != projectID || detailNote.vaultID != "" {
		t.Fatalf("detail row = %+v", detailNote)
	}
	indexNote := loadKnowledgeByPath(t, pub.DB, "memory/lessons.md")
	if indexNote.kind != "memory_index" || indexNote.projectID != projectID {
		t.Fatalf("lessons row = %+v", indexNote)
	}

	agentsLinks := loadPublishedLinks(t, pub.DB, agentsNote.id)
	if len(agentsLinks) != 1 || agentsLinks[0].raw != "memory/lessons" || agentsLinks[0].toPath != "memory/lessons.md" {
		t.Fatalf("AGENTS links = %#v", agentsLinks)
	}
	if agentsLinks[0].toNote != indexNote.id {
		t.Fatalf("AGENTS to_note_id = %q, want %s", agentsLinks[0].toNote, indexNote.id)
	}

	detailLinks := loadPublishedLinks(t, pub.DB, detailNote.id)
	got := map[string]publishedLink{}
	for _, l := range detailLinks {
		got[l.toPath] = l
	}
	if l, ok := got["AGENTS.md"]; !ok || l.raw != "AGENTS" || l.toNote != agentsNote.id {
		t.Fatalf("detail AGENTS link = %#v", detailLinks)
	}
	if l, ok := got["memory/lessons.md"]; !ok || l.toNote != indexNote.id {
		t.Fatalf("detail lessons link = %#v", detailLinks)
	}
}

func TestPublishApproved_VaultMemoryUsesVaultScope(t *testing.T) {
	db, dataDir := testutil.TempDB(t)
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	c := &clock.FakeClock{T: now}
	ctx := context.Background()
	vs := store.NewVaultStore(db, dataDir, c)
	v, err := vs.Create(ctx, "Vault")
	if err != nil {
		t.Fatal(err)
	}
	ts := now.Format(time.RFC3339Nano)
	sessionID := "sess-vault"
	if _, err := db.Exec(`INSERT INTO sessions(id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at)
		VALUES(?,?,?,NULL,'active','openai','gpt-test','{}','{}','S',?,?)`,
		sessionID, "vault", v.ID, ts, ts); err != nil {
		t.Fatal(err)
	}
	cs := &store.CompoundStore{DB: db, Clock: c}
	pub := &compound.Publisher{DataDir: dataDir, DB: db, Clock: c}

	detailPath := "memory/20260822-1400-vault-lesson.md"
	approveAndPublish(t, pub, cs, store.CreateProposalInput{
		SessionID:  sessionID,
		RequestKey: "rk-vault",
		Scope:      domain.CompoundScopeVault,
		VaultID:    v.ID,
		Items: []store.CompoundItem{
			item("memory_detail", detailPath, "create", "See [[memory/lessons]]\n"),
			item("lessons_index_row", "memory/lessons.md", "update", "- [[memory/20260822-1400-vault-lesson|Vault lesson]]\n"),
		},
		Now: now,
	}, now.Add(time.Minute))

	detailNote := loadKnowledgeByPath(t, db, detailPath)
	if detailNote.kind != "memory_detail" || detailNote.vaultID != v.ID || detailNote.projectID != "" || detailNote.isGlobal != 0 {
		t.Fatalf("vault detail = %+v", detailNote)
	}
	indexNote := loadKnowledgeByPath(t, db, "memory/lessons.md")
	if indexNote.kind != "memory_index" || indexNote.vaultID != v.ID || indexNote.projectID != "" {
		t.Fatalf("vault index = %+v", indexNote)
	}
	links := loadPublishedLinks(t, db, detailNote.id)
	if len(links) != 1 || links[0].toPath != "memory/lessons.md" || links[0].toNote != indexNote.id {
		t.Fatalf("vault links = %#v", links)
	}
}

func TestPublishApproved_GlobalAgentsUsesGlobalScope(t *testing.T) {
	db, dataDir := testutil.TempDB(t)
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	c := &clock.FakeClock{T: now}
	ts := now.Format(time.RFC3339Nano)
	sessionID := "sess-global"
	if _, err := db.Exec(`INSERT INTO sessions(id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at)
		VALUES(?,?,NULL,NULL,'active','openai','gpt-test','{}','{}','S',?,?)`,
		sessionID, "global", ts, ts); err != nil {
		t.Fatal(err)
	}
	cs := &store.CompoundStore{DB: db, Clock: c}
	pub := &compound.Publisher{DataDir: dataDir, DB: db, Clock: c}

	agents := "# Global agents\n\n## Memory\n- Lesson index: [[memory/lessons|lessons.md]] — scan titles.\n"
	approveAndPublish(t, pub, cs, store.CreateProposalInput{
		SessionID:  sessionID,
		RequestKey: "rk-global",
		Scope:      domain.CompoundScopeGlobal,
		Items:      []store.CompoundItem{item("agents_patch", "AGENTS.md", "update", agents)},
		Now:        now,
	}, now.Add(time.Minute))

	note := loadKnowledgeByPath(t, db, "AGENTS.md")
	if note.kind != "agents" || note.projectID != "" || note.vaultID != "" || note.isGlobal != 1 {
		t.Fatalf("global AGENTS = %+v", note)
	}
	links := loadPublishedLinks(t, db, note.id)
	if len(links) != 1 || links[0].raw != "memory/lessons" || links[0].toPath != "memory/lessons.md" {
		t.Fatalf("global links = %#v", links)
	}
}
