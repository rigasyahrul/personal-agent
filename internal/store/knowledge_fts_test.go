package store_test

import (
	"context"
	"database/sql"
	"errors"
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
