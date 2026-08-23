package store_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

func ftsNoteIDs(t *testing.T, db *sql.DB, token string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT note_id FROM knowledge_fts WHERE knowledge_fts MATCH ?`, token)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return ids
}

// Break this would catch: UpsertFromContent never writing knowledge_fts, so a
// body token cannot be found, or FTS note_id is not knowledge_notes.id.
func TestKnowledgeUpsertReindexesFTS(t *testing.T) {
	s, db, _ := knowledgeHarness(t)
	ctx := context.Background()

	note, err := s.UpsertFromContent(ctx, projectUpsert(
		domain.KnowledgeKindMemoryDetail,
		"memory/fts.md",
		[]byte("---\ntitle: UniqueTitleToken\n---\nBody contains zebraquark.\n"),
	))
	if err != nil {
		t.Fatal(err)
	}

	ids := ftsNoteIDs(t, db, "zebraquark")
	if len(ids) != 1 || ids[0] != note.ID {
		t.Fatalf("FTS body token note_id = %v, want [%s] (knowledge_notes.id)", ids, note.ID)
	}

	ids = ftsNoteIDs(t, db, "UniqueTitleToken")
	if len(ids) != 1 || ids[0] != note.ID {
		t.Fatalf("FTS title token note_id = %v, want [%s]", ids, note.ID)
	}

	updated, err := s.UpsertFromContent(ctx, projectUpsert(
		domain.KnowledgeKindMemoryDetail,
		"memory/fts.md",
		[]byte("Replacement body has wibbleflux.\n"),
	))
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != note.ID {
		t.Fatalf("upsert id changed: %s → %s", note.ID, updated.ID)
	}
	if ids := ftsNoteIDs(t, db, "zebraquark"); len(ids) != 0 {
		t.Fatalf("old FTS token still present: %v", ids)
	}
	ids = ftsNoteIDs(t, db, "wibbleflux")
	if len(ids) != 1 || ids[0] != note.ID {
		t.Fatalf("FTS after reindex = %v, want [%s]", ids, note.ID)
	}
}

// Break this would catch: RemoveFTS leaving the FTS row so the token still matches.
func TestKnowledgeRemoveFTS(t *testing.T) {
	s, db, _ := knowledgeHarness(t)
	ctx := context.Background()

	note, err := s.UpsertFromContent(ctx, projectUpsert(
		domain.KnowledgeKindMemoryDetail,
		"memory/gone.md",
		[]byte("Token plumblossom stays until remove.\n"),
	))
	if err != nil {
		t.Fatal(err)
	}
	if ids := ftsNoteIDs(t, db, "plumblossom"); len(ids) != 1 || ids[0] != note.ID {
		t.Fatalf("precondition FTS = %v, want [%s]", ids, note.ID)
	}

	if err := s.RemoveFTS(ctx, note.ID); err != nil {
		t.Fatal(err)
	}
	if ids := ftsNoteIDs(t, db, "plumblossom"); len(ids) != 0 {
		t.Fatalf("FTS after RemoveFTS = %v, want empty", ids)
	}
}

func TestKnowledgeRemoveFTSRejectsEmptyID(t *testing.T) {
	s, _, _ := knowledgeHarness(t)
	if err := s.RemoveFTS(context.Background(), ""); !errors.Is(err, store.ErrValidation) {
		t.Fatalf("RemoveFTS empty id = %v, want ErrValidation", err)
	}
}

// Break this would catch: SearchProject matching FTS across projects (no project_id join).
func TestSearchProjectIsolatesProjects(t *testing.T) {
	s, db, _ := knowledgeHarness(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES('p2','v1','P2','x','x')`); err != nil {
		t.Fatal(err)
	}

	p1, err := s.UpsertFromContent(ctx, store.UpsertKnowledgeInput{
		Kind:         domain.KnowledgeKindMemoryDetail,
		ProjectID:    "p1",
		RelativePath: "memory/p1.md",
		Content:      []byte("sharedtoken lives in project one.\n"),
		Status:       "ready",
	})
	if err != nil {
		t.Fatal(err)
	}
	p2, err := s.UpsertFromContent(ctx, store.UpsertKnowledgeInput{
		Kind:         domain.KnowledgeKindMemoryDetail,
		ProjectID:    "p2",
		RelativePath: "memory/p2.md",
		Content:      []byte("sharedtoken lives in project two.\n"),
		Status:       "ready",
	})
	if err != nil {
		t.Fatal(err)
	}

	hits, err := s.SearchProject(ctx, "p1", "sharedtoken", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].KnowledgeID != p1.ID || hits[0].Path != "memory/p1.md" {
		t.Fatalf("p1 hits = %+v, want only knowledge_id=%s path=memory/p1.md", hits, p1.ID)
	}

	hits, err = s.SearchProject(ctx, "p2", "sharedtoken", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].KnowledgeID != p2.ID || hits[0].Path != "memory/p2.md" {
		t.Fatalf("p2 hits = %+v, want only knowledge_id=%s path=memory/p2.md", hits, p2.ID)
	}
}

// Break this would catch: empty MATCH leaking a SQL error or returning every FTS row.
func TestSearchProjectEmptyQuery(t *testing.T) {
	s, _, _ := knowledgeHarness(t)
	ctx := context.Background()
	if _, err := s.UpsertFromContent(ctx, projectUpsert(
		domain.KnowledgeKindMemoryDetail,
		"memory/keep.md",
		[]byte("alpha token stays indexed.\n"),
	)); err != nil {
		t.Fatal(err)
	}

	for _, q := range []string{"", "   ", "\t"} {
		hits, err := s.SearchProject(ctx, "p1", q, 20)
		if err != nil {
			t.Fatalf("query %q err = %v, want nil", q, err)
		}
		if len(hits) != 0 {
			t.Fatalf("query %q hits = %+v, want empty", q, hits)
		}
	}
}

// Break this would catch: trusting knowledge_fts without joining knowledge_notes.project_id.
func TestSearchProjectIgnoresOrphanFTS(t *testing.T) {
	s, db, _ := knowledgeHarness(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO knowledge_fts(note_id, title, path, body) VALUES('orphan','x','x','orphantoken')`); err != nil {
		t.Fatal(err)
	}
	hits, err := s.SearchProject(ctx, "p1", "orphantoken", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("orphan FTS hits = %+v, want empty", hits)
	}
}

// Break this would catch: limit<=0 returning 0/all rows, or limit>50 not clamped.
func TestSearchProjectLimitDefaultAndMax(t *testing.T) {
	s, _, _ := knowledgeHarness(t)
	ctx := context.Background()
	const token = "limitword"
	for i := 0; i < 51; i++ {
		_, err := s.UpsertFromContent(ctx, projectUpsert(
			domain.KnowledgeKindMemoryDetail,
			fmt.Sprintf("memory/limit-%02d.md", i),
			[]byte(token+" note.\n"),
		))
		if err != nil {
			t.Fatal(err)
		}
	}

	defaultHits, err := s.SearchProject(ctx, "p1", token, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(defaultHits) != 20 {
		t.Fatalf("limit 0 hits = %d, want default 20", len(defaultHits))
	}

	maxHits, err := s.SearchProject(ctx, "p1", token, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(maxHits) != 50 {
		t.Fatalf("limit 100 hits = %d, want max 50", len(maxHits))
	}

	two, err := s.SearchProject(ctx, "p1", token, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(two) != 2 {
		t.Fatalf("limit 2 hits = %d, want 2", len(two))
	}
}

// Break this would catch: raw FTS5 syntax in q surfacing as a SQL error.
func TestSearchProjectFTSSpecialCharsDoNotError(t *testing.T) {
	s, _, _ := knowledgeHarness(t)
	ctx := context.Background()
	note, err := s.UpsertFromContent(ctx, projectUpsert(
		domain.KnowledgeKindMemoryDetail,
		"memory/safe.md",
		[]byte("needleword sits in the body.\n"),
	))
	if err != nil {
		t.Fatal(err)
	}

	for _, q := range []string{`"`, `*`, `AND`, `OR`, `NOT`, `"foo*bar"`, `foo AND bar`, `NEEDLEWORD*`, `"needleword"`} {
		hits, err := s.SearchProject(ctx, "p1", q, 20)
		if err != nil {
			t.Fatalf("query %q err = %v (must not fail)", q, err)
		}
		_ = hits
	}

	hits, err := s.SearchProject(ctx, "p1", `"needleword"`, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].KnowledgeID != note.ID {
		t.Fatalf("quoted needle hits = %+v, want [%s]", hits, note.ID)
	}
}

// Break this would catch: empty snippet, or dumping the whole body instead of a window.
func TestSearchProjectSnippetAroundFirstMatch(t *testing.T) {
	s, _, _ := knowledgeHarness(t)
	ctx := context.Background()
	prefix := strings.Repeat("alpha ", 40)
	suffix := strings.Repeat(" omega", 40)
	body := prefix + "secretword" + suffix + "\n"
	_, err := s.UpsertFromContent(ctx, projectUpsert(
		domain.KnowledgeKindMemoryDetail,
		"memory/snip.md",
		[]byte(body),
	))
	if err != nil {
		t.Fatal(err)
	}

	hits, err := s.SearchProject(ctx, "p1", "secretword", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("hits = %+v, want 1", hits)
	}
	if !strings.Contains(hits[0].Snippet, "secretword") {
		t.Fatalf("snippet %q missing secretword", hits[0].Snippet)
	}
	if len(hits[0].Snippet) >= len(strings.TrimSpace(body)) {
		t.Fatalf("snippet is full body (%d chars)", len(hits[0].Snippet))
	}
}

// Break this would catch: SearchHit.KnowledgeID = notes.id / source_note_id.
func TestSearchProjectKnowledgeIDIsNotSourceNoteID(t *testing.T) {
	s, _, _ := knowledgeHarness(t)
	ctx := context.Background()
	note, err := s.UpsertFromContent(ctx, store.UpsertKnowledgeInput{
		Kind:         domain.KnowledgeKindSource,
		ProjectID:    "p1",
		RelativePath: "source/articles/intro.md",
		Content:      []byte("sourcebodytoken in the mirror.\n"),
		Status:       "ready",
		SourceNoteID: "note-src",
	})
	if err != nil {
		t.Fatal(err)
	}
	hits, err := s.SearchProject(ctx, "p1", "sourcebodytoken", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("hits = %+v, want 1", hits)
	}
	if hits[0].KnowledgeID != note.ID || hits[0].KnowledgeID == "note-src" {
		t.Fatalf("knowledge_id = %q, want %s (not notes.id)", hits[0].KnowledgeID, note.ID)
	}
	if hits[0].SourceNoteID != "note-src" || hits[0].Kind != domain.KnowledgeKindSource {
		t.Fatalf("hit = %+v, want source_note_id=note-src kind=source", hits[0])
	}
	if hits[0].Path != "source/articles/intro.md" || hits[0].Title == "" {
		t.Fatalf("path/title = %+v", hits[0])
	}
}

// Break this would catch: body-only hits ranking above a title match.
func TestSearchProjectPrefersTitleMatch(t *testing.T) {
	s, _, _ := knowledgeHarness(t)
	ctx := context.Background()
	titleNote, err := s.UpsertFromContent(ctx, projectUpsert(
		domain.KnowledgeKindMemoryDetail,
		"memory/title.md",
		[]byte("---\ntitle: Ranktoken Title\n---\nNo other mention.\n"),
	))
	if err != nil {
		t.Fatal(err)
	}
	bodyNote, err := s.UpsertFromContent(ctx, projectUpsert(
		domain.KnowledgeKindMemoryDetail,
		"memory/body.md",
		[]byte("---\ntitle: Other\n---\nBody mentions ranktoken only here.\n"),
	))
	if err != nil {
		t.Fatal(err)
	}

	hits, err := s.SearchProject(ctx, "p1", "ranktoken", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("hits = %+v, want 2", hits)
	}
	if hits[0].KnowledgeID != titleNote.ID {
		t.Fatalf("first hit = %s (%s), want title note %s (body was %s)",
			hits[0].KnowledgeID, hits[0].Path, titleNote.ID, bodyNote.ID)
	}
}
