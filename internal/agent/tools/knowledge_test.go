package tools_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rigasyahrul/personal-agent/internal/agent/tools"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/store"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func knowledgeToolHarness(t *testing.T) *store.KnowledgeStore {
	t.Helper()
	db, _ := testutil.TempDB(t)
	if _, err := db.Exec(`
		INSERT INTO vaults(id,name,created_at,updated_at) VALUES('v1','V','x','x');
		INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES('p1','v1','P','x','x');
		INSERT INTO notes(id,project_id,relative_path,status,revision,created_at,updated_at)
			VALUES('note-src','p1','articles/intro.md','ready',1,'x','x');
	`); err != nil {
		t.Fatal(err)
	}
	return &store.KnowledgeStore{DB: db}
}

// Break this would catch: handler not calling SearchProject, or emitting note_id
// / source_note_id as knowledge_id instead of knowledge_notes.id.
func TestSearchProjectReturnsHitsWithKnowledgeID(t *testing.T) {
	s := knowledgeToolHarness(t)
	ctx := context.Background()
	note, err := s.UpsertFromContent(ctx, store.UpsertKnowledgeInput{
		Kind:         domain.KnowledgeKindSource,
		ProjectID:    "p1",
		RelativePath: "source/articles/intro.md",
		Content:      []byte("Body mentions searchneedletoken once.\n"),
		Status:       "ready",
		SourceNoteID: "note-src",
	})
	if err != nil {
		t.Fatal(err)
	}

	h := tools.NewKnowledgeToolHandler(s, "p1")
	raw, err := h.Execute(ctx, "search_project", json.RawMessage(`{"query":"searchneedletoken","limit":5}`))
	if err != nil {
		t.Fatal(err)
	}

	var out struct {
		Hits []struct {
			KnowledgeID  string `json:"knowledge_id"`
			Path         string `json:"path"`
			Title        string `json:"title"`
			Snippet      string `json:"snippet"`
			Kind         string `json:"kind"`
			SourceNoteID string `json:"source_note_id"`
			NoteID       string `json:"note_id"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("result JSON: %v\n%s", err, raw)
	}
	if len(out.Hits) != 1 {
		t.Fatalf("hits = %#v, want 1", out.Hits)
	}
	hit := out.Hits[0]
	if hit.KnowledgeID != note.ID || hit.KnowledgeID == "note-src" {
		t.Fatalf("knowledge_id = %q, want %s (not notes.id)", hit.KnowledgeID, note.ID)
	}
	if hit.NoteID != "" {
		t.Fatalf("hit must not include note_id, got %q in %s", hit.NoteID, raw)
	}
	if hit.Path != "source/articles/intro.md" || hit.Kind != string(domain.KnowledgeKindSource) {
		t.Fatalf("path/kind = %+v", hit)
	}
	if hit.SourceNoteID != "note-src" {
		t.Fatalf("source_note_id = %q, want note-src", hit.SourceNoteID)
	}
	if !strings.Contains(strings.ToLower(hit.Snippet), "searchneedletoken") {
		t.Fatalf("snippet %q missing query", hit.Snippet)
	}
}

func TestKnowledgeRejectsUnknownName(t *testing.T) {
	h := tools.NewKnowledgeToolHandler(knowledgeToolHarness(t), "p1")
	if _, err := h.Execute(context.Background(), "read_file", json.RawMessage(`{"path":"x"}`)); err == nil {
		t.Fatal("workspace name accepted as knowledge tool")
	}
	if _, err := h.Execute(context.Background(), "search_vault", json.RawMessage(`{"query":"x"}`)); err == nil {
		t.Fatal("unknown name accepted")
	}
}
