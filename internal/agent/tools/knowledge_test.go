package tools_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rigasyahrul/personal-agent/internal/agent/tools"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/layout"
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

// Break this would catch: read_knowledge not registered, or instruction files
// opened via fsroot.Open(projectRoot) without the allowlist.
func TestReadKnowledgeReturnsAGENTS(t *testing.T) {
	root := t.TempDir()
	want := "## Memory\n- Lesson index lives here.\n"
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}
	h := tools.NewKnowledgeToolHandler(nil, "p1")
	h.ScopeRoot = root
	raw, err := h.Execute(context.Background(), tools.ToolReadKnowledge, json.RawMessage(`{"path":"AGENTS.md"}`))
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("result JSON: %v\n%s", err, raw)
	}
	if out.Path != "AGENTS.md" || out.Content != want {
		t.Fatalf("got %+v, want path=AGENTS.md content=%q", out, want)
	}
}

// Break this would catch: path validation skipped so ../ escapes the scope.
func TestReadKnowledgeRejectsTraversal(t *testing.T) {
	h := tools.NewKnowledgeToolHandler(nil, "p1")
	h.ScopeRoot = t.TempDir()
	if _, err := h.Execute(context.Background(), tools.ToolReadKnowledge, json.RawMessage(`{"path":"../secret.md"}`)); err == nil {
		t.Fatal("accepted ../ traversal")
	}
}

// Break this would catch: opening via fsroot.Open(projectRoot)+memory/…
// (ValidateRelPath rejects) instead of MemoryDir as the sub-root.
func TestReadKnowledgeReadsMemoryViaMemoryDir(t *testing.T) {
	root := t.TempDir()
	want := "# Lessons\n\n- Rooted memory read.\n"
	if err := os.MkdirAll(layout.MemoryDir(root), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(layout.MemoryDir(root), "lessons.md"), []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}
	h := tools.NewKnowledgeToolHandler(nil, "p1")
	h.ScopeRoot = root
	raw, err := h.Execute(context.Background(), tools.ToolReadKnowledge, json.RawMessage(`{"path":"memory/lessons.md"}`))
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("result JSON: %v\n%s", err, raw)
	}
	if out.Path != "memory/lessons.md" || out.Content != want {
		t.Fatalf("got %+v, want memory/lessons.md %q", out, want)
	}
}

func TestReadKnowledgeRejectsAgentsAndSessions(t *testing.T) {
	h := tools.NewKnowledgeToolHandler(nil, "p1")
	h.ScopeRoot = t.TempDir()
	for _, p := range []string{".agents/skills/compounding/SKILL.md", "sessions/s1/file.md", "source/.agents/x.md"} {
		if _, err := h.Execute(context.Background(), tools.ToolReadKnowledge, json.RawMessage(`{"path":`+jsonString(p)+`}`)); err == nil {
			t.Fatalf("accepted reserved path %q", p)
		}
	}
}

// Break this would catch: listing the project root via fsroot.Open(scopeRoot)
// (leaks .agents/sessions) or omitting source/memory children.
func TestListKnowledgeListsEntriesOmitsAgentsAndSessions(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(layout.SourceDir(root), "guide"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(layout.SourceDir(root), "guide", "intro.md"), []byte("# Intro\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(layout.MemoryDir(root), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(layout.MemoryDir(root), "lessons.md"), []byte("# Lessons\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".agents", "skills"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".agents", "skills", "hidden.md"), []byte("nope\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "sessions", "s1"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sessions", "s1", "secret.md"), []byte("nope\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := tools.NewKnowledgeToolHandler(nil, "p1")
	h.ScopeRoot = root
	raw, err := h.Execute(context.Background(), tools.ToolListKnowledge, json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Entries []struct {
			Path string `json:"path"`
			Kind string `json:"kind"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("result JSON: %v\n%s", err, raw)
	}
	seen := map[string]string{}
	for _, e := range out.Entries {
		seen[e.Path] = e.Kind
		if strings.Contains(e.Path, ".agents") || strings.HasPrefix(e.Path, "sessions") || strings.Contains(e.Path, "/sessions/") {
			t.Fatalf("list leaked excluded path %q", e.Path)
		}
	}
	if seen["source"] != "directory" || seen["memory"] != "directory" {
		t.Fatalf("missing knowledge roots: %#v", seen)
	}

	raw, err = h.Execute(context.Background(), tools.ToolListKnowledge, json.RawMessage(`{"path":"source"}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("source list JSON: %v\n%s", err, raw)
	}
	seen = map[string]string{}
	for _, e := range out.Entries {
		seen[e.Path] = e.Kind
	}
	if seen["source/guide"] != "directory" {
		t.Fatalf("source children = %#v, want source/guide directory", seen)
	}
}

// Break this would catch: list_knowledge skipping path validation.
func TestListKnowledgeRejectsTraversal(t *testing.T) {
	h := tools.NewKnowledgeToolHandler(nil, "p1")
	h.ScopeRoot = t.TempDir()
	if _, err := h.Execute(context.Background(), tools.ToolListKnowledge, json.RawMessage(`{"path":"../"}`)); err == nil {
		t.Fatal("accepted ../ list")
	}
}

func jsonString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(b)
}
