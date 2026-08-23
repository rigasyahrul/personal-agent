package store_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/store"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

// Break this would catch: boot-time backfill never upserts a ready notes row
// from disk, so knowledge_notes and knowledge_fts stay empty for existing
// source files (the pre-knowledge-index migration gap).
func TestBackfillReadySourceNotesCreatesKnowledgeAndFTS(t *testing.T) {
	db, dataDir := testutil.TempDB(t)
	ctx := context.Background()
	if _, err := db.Exec(`
		INSERT INTO vaults(id,name,created_at,updated_at) VALUES('v1','V','x','x');
		INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES('p1','v1','P','x','x');
	`); err != nil {
		t.Fatal(err)
	}

	source := layout.SourceDir(layout.ProjectRoot(dataDir, "v1", "p1"))
	if err := os.MkdirAll(filepath.Join(source, "guide"), 0o700); err != nil {
		t.Fatal(err)
	}
	body := []byte("# Safe\n\nBackfill token plumblossom lives here.\n")
	if err := os.WriteFile(filepath.Join(source, "guide", "safe.md"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(body))
	if _, err := db.Exec(`
		INSERT INTO notes(id,project_id,relative_path,content_sha256,byte_size,status,revision,created_at,updated_at)
		VALUES('n1','p1','guide/safe.md',?,?,'ready',1,'x','x')`, sum, len(body)); err != nil {
		t.Fatal(err)
	}

	ks := &store.KnowledgeStore{DB: db, Clock: clock.RealClock{}}
	if err := ks.BackfillReadySourceNotes(ctx, dataDir); err != nil {
		t.Fatal(err)
	}

	note, err := ks.ByScopePath(ctx, "p1", "", false, "source/guide/safe.md")
	if err != nil {
		t.Fatalf("knowledge mirror missing: %v", err)
	}
	if note.Kind != domain.KnowledgeKindSource {
		t.Fatalf("kind = %q, want source", note.Kind)
	}
	if note.SourceNoteID != "n1" {
		t.Fatalf("source_note_id = %q, want n1", note.SourceNoteID)
	}
	if note.ID == "n1" {
		t.Fatal("knowledge_notes.id must not equal notes.id")
	}
	if note.RelativePath != "source/guide/safe.md" || note.ProjectID != "p1" || note.IsGlobal {
		t.Fatalf("mirror = %+v", note)
	}

	ids := ftsNoteIDs(t, db, "plumblossom")
	if len(ids) != 1 || ids[0] != note.ID {
		t.Fatalf("FTS after backfill = %v, want [%s]", ids, note.ID)
	}
}

// Break this would catch: backfill re-inserts a second knowledge_notes row
// (or rewrites source_note_id) when the source mirror already exists.
func TestBackfillReadySourceNotesSkipsExistingMirror(t *testing.T) {
	db, dataDir := testutil.TempDB(t)
	ctx := context.Background()
	if _, err := db.Exec(`
		INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','P','x','x');
	`); err != nil {
		t.Fatal(err)
	}
	source := layout.SourceDir(layout.ProjectRoot(dataDir, "", "p1"))
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatal(err)
	}
	body := []byte("Already mirrored backfilltoken.\n")
	if err := os.WriteFile(filepath.Join(source, "intro.md"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(body))
	if _, err := db.Exec(`
		INSERT INTO notes(id,project_id,relative_path,content_sha256,byte_size,status,revision,created_at,updated_at)
		VALUES('n1','p1','intro.md',?,?,'ready',1,'x','x')`, sum, len(body)); err != nil {
		t.Fatal(err)
	}

	ks := &store.KnowledgeStore{DB: db, Clock: clock.RealClock{}}
	existing, err := ks.UpsertFromContent(ctx, store.UpsertKnowledgeInput{
		Kind:         domain.KnowledgeKindSource,
		ProjectID:    "p1",
		RelativePath: "source/intro.md",
		Content:      body,
		Status:       "ready",
		SourceNoteID: "n1",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := ks.BackfillReadySourceNotes(ctx, dataDir); err != nil {
		t.Fatal(err)
	}

	var n int
	if err := db.QueryRow(`SELECT count(*) FROM knowledge_notes WHERE project_id='p1'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("knowledge_notes count = %d, want 1", n)
	}
	got, err := ks.ByID(ctx, existing.ID)
	if err != nil || got.ID != existing.ID || got.SourceNoteID != "n1" {
		t.Fatalf("existing mirror changed: %+v err=%v", got, err)
	}
}
