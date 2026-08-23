package store_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/store"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func knowledgeHarness(t *testing.T) (*store.KnowledgeStore, *sql.DB, *clock.FakeClock) {
	t.Helper()
	db, _ := testutil.TempDB(t)
	clk := &clock.FakeClock{T: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)}
	if _, err := db.Exec(`
		INSERT INTO vaults(id,name,created_at,updated_at) VALUES('v1','V','x','x');
		INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES('p1','v1','P','x','x');
		INSERT INTO notes(id,project_id,relative_path,status,revision,created_at,updated_at)
			VALUES('note-src','p1','articles/intro.md','ready',1,'x','x');
	`); err != nil {
		t.Fatal(err)
	}
	return &store.KnowledgeStore{DB: db, Clock: clk}, db, clk
}

func projectUpsert(kind domain.KnowledgeKind, rel string, content []byte) store.UpsertKnowledgeInput {
	return store.UpsertKnowledgeInput{
		Kind:         kind,
		ProjectID:    "p1",
		RelativePath: rel,
		Content:      content,
		Status:       "ready",
	}
}

type linkRow struct {
	fromID, raw, toPath string
	toNote              sql.NullString
}

func loadLinksFrom(t *testing.T, db *sql.DB, fromID string) []linkRow {
	t.Helper()
	rows, err := db.Query(`SELECT from_note_id, raw_target, to_path, to_note_id FROM note_links WHERE from_note_id=? ORDER BY to_path, raw_target`, fromID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []linkRow
	for rows.Next() {
		var r linkRow
		if err := rows.Scan(&r.fromID, &r.raw, &r.toPath, &r.toNote); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestKnowledgeUpsertAThenBResolvesInboundLink(t *testing.T) {
	s, db, clk := knowledgeHarness(t)
	ctx := context.Background()

	bodyA := []byte("---\ntitle: Note A\n---\nSee [[memory/b]]\n")
	a, err := s.UpsertFromContent(ctx, projectUpsert(domain.KnowledgeKindMemoryDetail, "memory/a.md", bodyA))
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == "" || a.ID == "note-src" {
		t.Fatalf("knowledge id must be an independent UUID, got %q", a.ID)
	}
	if a.Title != "Note A" {
		t.Fatalf("title = %q, want Note A", a.Title)
	}
	if a.Kind != domain.KnowledgeKindMemoryDetail || a.ProjectID != "p1" || a.VaultID != "" || a.IsGlobal {
		t.Fatalf("scope = %+v", a)
	}
	if a.RelativePath != "memory/a.md" || a.Status != "ready" {
		t.Fatalf("path/status = %+v", a)
	}
	wantHash := fmt.Sprintf("%x", sha256.Sum256(bodyA))
	if a.ContentSHA256 != wantHash || a.ByteSize != int64(len(bodyA)) {
		t.Fatalf("hash/size = %s %d", a.ContentSHA256, a.ByteSize)
	}
	if !a.CreatedAt.Equal(clk.T) || !a.UpdatedAt.Equal(clk.T) {
		t.Fatalf("timestamps = %v %v", a.CreatedAt, a.UpdatedAt)
	}
	var fm map[string]any
	if err := json.Unmarshal([]byte(a.FrontmatterJSON), &fm); err != nil {
		t.Fatalf("frontmatter_json %q: %v", a.FrontmatterJSON, err)
	}
	if fm["title"] != "Note A" {
		t.Fatalf("frontmatter title = %#v", fm["title"])
	}

	got, err := s.ByID(ctx, a.ID)
	if err != nil || got.ID != a.ID || got.Title != "Note A" {
		t.Fatalf("ByID = %+v err=%v", got, err)
	}
	byPath, err := s.ByScopePath(ctx, "p1", "", false, "memory/a.md")
	if err != nil || byPath.ID != a.ID {
		t.Fatalf("ByScopePath = %+v err=%v", byPath, err)
	}

	links := loadLinksFrom(t, db, a.ID)
	if len(links) != 1 {
		t.Fatalf("links = %#v, want 1 edge", links)
	}
	if links[0].fromID != a.ID || links[0].raw != "memory/b" || links[0].toPath != "memory/b.md" {
		t.Fatalf("edge = %#v, want from A raw=memory/b to_path=memory/b.md", links[0])
	}
	if links[0].toNote.Valid {
		t.Fatalf("to_note_id should be unset before B exists, got %q", links[0].toNote.String)
	}

	bodyB := []byte("Target B\n")
	b, err := s.UpsertFromContent(ctx, projectUpsert(domain.KnowledgeKindMemoryDetail, "memory/b.md", bodyB))
	if err != nil {
		t.Fatal(err)
	}
	if b.ID == a.ID || b.Title != "b" {
		t.Fatalf("B = %+v", b)
	}

	links = loadLinksFrom(t, db, a.ID)
	if len(links) != 1 || !links[0].toNote.Valid || links[0].toNote.String != b.ID {
		t.Fatalf("after B upsert inbound to_note_id = %#v, want %s", links, b.ID)
	}
}

func TestKnowledgeUpsertBThenAResolvesOutboundLink(t *testing.T) {
	s, db, _ := knowledgeHarness(t)
	ctx := context.Background()

	b, err := s.UpsertFromContent(ctx, projectUpsert(domain.KnowledgeKindMemoryDetail, "memory/b.md", []byte("B\n")))
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.UpsertFromContent(ctx, projectUpsert(domain.KnowledgeKindMemoryDetail, "memory/a.md", []byte("See [[memory/b]]\n")))
	if err != nil {
		t.Fatal(err)
	}

	links := loadLinksFrom(t, db, a.ID)
	if len(links) != 1 {
		t.Fatalf("links = %#v", links)
	}
	if links[0].toPath != "memory/b.md" || !links[0].toNote.Valid || links[0].toNote.String != b.ID {
		t.Fatalf("outbound resolve = %#v, want to_note_id=%s", links[0], b.ID)
	}
}

func TestKnowledgeUpsertKeepsIDAndReplacesLinks(t *testing.T) {
	s, db, clk := knowledgeHarness(t)
	ctx := context.Background()

	first, err := s.UpsertFromContent(ctx, projectUpsert(domain.KnowledgeKindMemoryDetail, "memory/a.md", []byte("See [[memory/b]]\n")))
	if err != nil {
		t.Fatal(err)
	}
	clk.Advance(time.Minute)
	second, err := s.UpsertFromContent(ctx, projectUpsert(domain.KnowledgeKindMemoryDetail, "memory/a.md", []byte("Now [[source/x.md]]\n")))
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("upsert must keep id: first=%s second=%s", first.ID, second.ID)
	}
	if !second.CreatedAt.Equal(first.CreatedAt) {
		t.Fatalf("created_at changed: %v → %v", first.CreatedAt, second.CreatedAt)
	}
	if !second.UpdatedAt.After(first.UpdatedAt) {
		t.Fatalf("updated_at not advanced: %v → %v", first.UpdatedAt, second.UpdatedAt)
	}

	links := loadLinksFrom(t, db, first.ID)
	if len(links) != 1 || links[0].raw != "source/x.md" || links[0].toPath != "source/x.md" {
		t.Fatalf("replaced links = %#v", links)
	}
}

func TestKnowledgeUpsertSourceNoteIDIsIndependent(t *testing.T) {
	s, _, _ := knowledgeHarness(t)
	ctx := context.Background()

	in := projectUpsert(domain.KnowledgeKindSource, "source/articles/intro.md", []byte("# Intro\n"))
	in.SourceNoteID = "note-src"
	note, err := s.UpsertFromContent(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if note.SourceNoteID != "note-src" {
		t.Fatalf("source_note_id = %q", note.SourceNoteID)
	}
	if note.ID == "note-src" {
		t.Fatal("knowledge_notes.id must not equal notes.id")
	}
	// No frontmatter → TitleOrStem uses the path stem, not a markdown heading.
	if note.Title != "intro" {
		t.Fatalf("title = %q, want intro (stem)", note.Title)
	}
}

func TestKnowledgeUpsertRejectsNonKnowledgeRelPath(t *testing.T) {
	s, _, _ := knowledgeHarness(t)
	ctx := context.Background()

	// v1 notes-relative path — ValidateRelPath would allow this; knowledge must not.
	_, err := s.UpsertFromContent(ctx, projectUpsert(domain.KnowledgeKindSource, "articles/intro.md", []byte("x")))
	if !errors.Is(err, store.ErrValidation) {
		t.Fatalf("notes-relative path err = %v, want ErrValidation", err)
	}

	_, err = s.UpsertFromContent(ctx, projectUpsert(domain.KnowledgeKindMemoryDetail, "memory/../escape.md", []byte("x")))
	if !errors.Is(err, store.ErrValidation) {
		t.Fatalf("traversal err = %v, want ErrValidation", err)
	}
}

func TestKnowledgeUpsertRejectsInvalidScope(t *testing.T) {
	s, _, _ := knowledgeHarness(t)
	ctx := context.Background()
	body := []byte("x\n")

	cases := []store.UpsertKnowledgeInput{
		{Kind: domain.KnowledgeKindAgents, RelativePath: "AGENTS.md", Content: body, IsGlobal: true, ProjectID: "p1"},
		{Kind: domain.KnowledgeKindMemoryDetail, RelativePath: "memory/a.md", Content: body, ProjectID: "p1", VaultID: "v1"},
		{Kind: domain.KnowledgeKindMemoryDetail, RelativePath: "memory/a.md", Content: body},
	}
	for i, in := range cases {
		if _, err := s.UpsertFromContent(ctx, in); !errors.Is(err, store.ErrValidation) {
			t.Fatalf("case %d err = %v, want ErrValidation", i, err)
		}
	}
}

func TestKnowledgeScopeIsolationDoesNotResolveAcrossProjects(t *testing.T) {
	s, db, _ := knowledgeHarness(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO projects(id,name,created_at,updated_at) VALUES('p2','P2','x','x')`); err != nil {
		t.Fatal(err)
	}

	b, err := s.UpsertFromContent(ctx, projectUpsert(domain.KnowledgeKindMemoryDetail, "memory/b.md", []byte("B\n")))
	if err != nil {
		t.Fatal(err)
	}
	other := store.UpsertKnowledgeInput{
		Kind:         domain.KnowledgeKindMemoryDetail,
		ProjectID:    "p2",
		RelativePath: "memory/a.md",
		Content:      []byte("See [[memory/b]]\n"),
		Status:       "ready",
	}
	a, err := s.UpsertFromContent(ctx, other)
	if err != nil {
		t.Fatal(err)
	}
	links := loadLinksFrom(t, db, a.ID)
	if len(links) != 1 || links[0].toPath != "memory/b.md" {
		t.Fatalf("links = %#v", links)
	}
	if links[0].toNote.Valid {
		t.Fatalf("cross-project resolve filled to_note_id=%q (B=%s)", links[0].toNote.String, b.ID)
	}
}

func TestKnowledgeByIDAndByScopePathNotFound(t *testing.T) {
	s, _, _ := knowledgeHarness(t)
	ctx := context.Background()
	if _, err := s.ByID(ctx, "missing"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("ByID missing: %v", err)
	}
	if _, err := s.ByScopePath(ctx, "p1", "", false, "memory/missing.md"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("ByScopePath missing: %v", err)
	}
}

func TestKnowledgeBacklinksIncludesLinker(t *testing.T) {
	s, _, _ := knowledgeHarness(t)
	ctx := context.Background()

	b, err := s.UpsertFromContent(ctx, projectUpsert(domain.KnowledgeKindMemoryDetail, "memory/b.md", []byte("B\n")))
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.UpsertFromContent(ctx, projectUpsert(domain.KnowledgeKindMemoryDetail, "memory/a.md", []byte("---\ntitle: Note A\n---\nSee [[memory/b]]\n")))
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.Backlinks(ctx, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("Backlinks(B) = %#v, want 1", got)
	}
	if got[0].FromNoteID != a.ID || got[0].FromPath != "memory/a.md" || got[0].FromTitle != "Note A" {
		t.Fatalf("backlink = %#v, want FromNoteID=%s FromPath=memory/a.md FromTitle=Note A", got[0], a.ID)
	}
}

func TestKnowledgeDeleteLinksFrom(t *testing.T) {
	s, db, _ := knowledgeHarness(t)
	ctx := context.Background()
	a, err := s.UpsertFromContent(ctx, projectUpsert(domain.KnowledgeKindMemoryDetail, "memory/a.md", []byte("See [[memory/b]]\n")))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteLinksFrom(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	if links := loadLinksFrom(t, db, a.ID); len(links) != 0 {
		t.Fatalf("links after delete = %#v", links)
	}
}

func TestKnowledgeGlobalAndVaultScopes(t *testing.T) {
	s, _, _ := knowledgeHarness(t)
	ctx := context.Background()

	global, err := s.UpsertFromContent(ctx, store.UpsertKnowledgeInput{
		Kind:         domain.KnowledgeKindAgents,
		IsGlobal:     true,
		RelativePath: "AGENTS.md",
		Content:      []byte("Global agents\n"),
		Status:       "ready",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !global.IsGlobal || global.ProjectID != "" || global.VaultID != "" {
		t.Fatalf("global = %+v", global)
	}
	got, err := s.ByScopePath(ctx, "", "", true, "AGENTS.md")
	if err != nil || got.ID != global.ID {
		t.Fatalf("global ByScopePath = %+v err=%v", got, err)
	}

	vault, err := s.UpsertFromContent(ctx, store.UpsertKnowledgeInput{
		Kind:         domain.KnowledgeKindMemoryIndex,
		VaultID:      "v1",
		RelativePath: "memory/lessons.md",
		Content:      []byte("---\ntitle: Lessons\n---\n"),
		Status:       "ready",
	})
	if err != nil {
		t.Fatal(err)
	}
	if vault.VaultID != "v1" || vault.ProjectID != "" || vault.IsGlobal || vault.Title != "Lessons" {
		t.Fatalf("vault = %+v", vault)
	}
	got, err = s.ByScopePath(ctx, "", "v1", false, "memory/lessons.md")
	if err != nil || got.ID != vault.ID {
		t.Fatalf("vault ByScopePath = %+v err=%v", got, err)
	}
}

// Break this would catch: Backlinks missing a memory detail that wikilinks
// [[source/intro|Intro]] / [[AGENTS]] after those targets exist in the same project,
// including when memory is upserted before the targets.
func TestKnowledgeBacklinksCrossKindMemorySourceAgents(t *testing.T) {
	body, err := os.ReadFile("../knowledge/testdata/sample_memory.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "[[source/intro|Intro]]") || !strings.Contains(string(body), "[[AGENTS]]") {
		t.Fatal("sample_memory.md must contain [[source/intro|Intro]] and [[AGENTS]]")
	}

	const wantTitle = "Hub startSession is not create-and-send atomic"
	orders := [][]string{
		{"memory", "source", "agents"},
		{"memory", "agents", "source"},
		{"source", "memory", "agents"},
		{"source", "agents", "memory"},
		{"agents", "memory", "source"},
		{"agents", "source", "memory"},
	}
	for _, order := range orders {
		t.Run(strings.Join(order, "-"), func(t *testing.T) {
			s, _, _ := knowledgeHarness(t)
			ctx := context.Background()

			var memory, source, agents domain.KnowledgeNote
			for _, name := range order {
				var note domain.KnowledgeNote
				var err error
				switch name {
				case "memory":
					note, err = s.UpsertFromContent(ctx, projectUpsert(domain.KnowledgeKindMemoryDetail, "memory/hub.md", body))
					memory = note
				case "source":
					note, err = s.UpsertFromContent(ctx, projectUpsert(domain.KnowledgeKindSource, "source/intro.md", []byte("# Intro\n")))
					source = note
				case "agents":
					note, err = s.UpsertFromContent(ctx, projectUpsert(domain.KnowledgeKindAgents, "AGENTS.md", []byte("Standing rules\n")))
					agents = note
				default:
					t.Fatalf("unknown upsert %q", name)
				}
				if err != nil {
					t.Fatalf("upsert %s: %v", name, err)
				}
			}

			assertBacklinkFromMemory := func(targetID, label string) {
				t.Helper()
				got, err := s.Backlinks(ctx, targetID)
				if err != nil {
					t.Fatalf("Backlinks(%s): %v", label, err)
				}
				for _, bl := range got {
					if bl.FromNoteID == memory.ID && bl.FromPath == "memory/hub.md" && bl.FromTitle == wantTitle {
						return
					}
				}
				t.Fatalf("Backlinks(%s) = %#v, want FromNoteID=%s FromPath=memory/hub.md FromTitle=%q", label, got, memory.ID, wantTitle)
			}
			assertBacklinkFromMemory(source.ID, "source")
			assertBacklinkFromMemory(agents.ID, "AGENTS")
		})
	}
}
