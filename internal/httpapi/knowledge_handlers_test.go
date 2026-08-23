package httpapi

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/auth"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/store"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func knowledgeAPIServer(t *testing.T) (http.Handler, *sql.DB, string, string, []*http.Cookie, *clock.FakeClock) {
	t.Helper()
	db, dir := testutil.TempDB(t)
	now := time.Date(2026, 8, 22, 18, 0, 0, 0, time.UTC)
	stamp := now.Format(time.RFC3339Nano)
	if _, err := db.Exec(`INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,'x',?,?)`, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO auth_sessions(token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,?)`,
		auth.TokenHash("token"), "csrf", now.Add(time.Hour).Format(time.RFC3339Nano), stamp); err != nil {
		t.Fatal(err)
	}
	clk := &clock.FakeClock{T: now}
	h := New(ServerDeps{DB: db, DataDir: dir, Clock: clk})
	cookies := []*http.Cookie{
		{Name: "pa_session", Value: "token"},
		{Name: "pa_csrf", Value: "csrf"},
	}
	created := apiRequest(t, h, http.MethodPost, "/api/v1/projects", map[string]any{
		"name":     "Knowledge",
		"vault_id": nil,
	}, cookies, "csrf")
	if created.Code != http.StatusCreated {
		t.Fatalf("create project = %d %s", created.Code, created.Body.String())
	}
	var proj ProjectDTO
	if err := json.NewDecoder(created.Body).Decode(&proj); err != nil {
		t.Fatal(err)
	}
	if proj.ID == "" {
		t.Fatal("empty project id")
	}
	return h, db, dir, proj.ID, cookies, clk
}

func TestKnowledgeReadMemoryPath(t *testing.T) {
	h, _, dataDir, pid, cookies, _ := knowledgeAPIServer(t)
	want := "# Memory detail\n\nRemember the rooted opener.\n"
	path := filepath.Join(layout.MemoryDir(layout.ProjectRoot(dataDir, "", pid)), "20260822-1800-rooted.md")
	if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}

	got := apiRequest(t, h, http.MethodGet, "/api/v1/projects/"+pid+"/knowledge/read?path=memory/20260822-1800-rooted.md", nil, cookies, "")
	if got.Code != http.StatusOK {
		t.Fatalf("GET knowledge/read memory = %d %s", got.Code, got.Body.String())
	}
	var body struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(got.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Path != "memory/20260822-1800-rooted.md" {
		t.Fatalf("path = %q", body.Path)
	}
	if body.Content != want {
		t.Fatalf("content = %q want %q", body.Content, want)
	}
}

func TestKnowledgeBacklinksJSONUsesKnowledgeID(t *testing.T) {
	h, db, _, pid, cookies, clk := knowledgeAPIServer(t)
	ks := &store.KnowledgeStore{DB: db, Clock: clk}
	ctx := t.Context()

	target, err := ks.UpsertFromContent(ctx, store.UpsertKnowledgeInput{
		Kind:         domain.KnowledgeKindMemoryDetail,
		ProjectID:    pid,
		RelativePath: "memory/b.md",
		Content:      []byte("Target B\n"),
		Status:       "ready",
	})
	if err != nil {
		t.Fatal(err)
	}
	from, err := ks.UpsertFromContent(ctx, store.UpsertKnowledgeInput{
		Kind:         domain.KnowledgeKindMemoryDetail,
		ProjectID:    pid,
		RelativePath: "memory/a.md",
		Content:      []byte("---\ntitle: Note A\n---\nSee [[memory/b]]\n"),
		Status:       "ready",
	})
	if err != nil {
		t.Fatal(err)
	}

	got := apiRequest(t, h, http.MethodGet, "/api/v1/projects/"+pid+"/knowledge/backlinks?knowledge_id="+from.ID, nil, cookies, "")
	if got.Code != http.StatusOK {
		t.Fatalf("backlinks by knowledge_id of linker = %d %s", got.Code, got.Body.String())
	}

	got = apiRequest(t, h, http.MethodGet, "/api/v1/projects/"+pid+"/knowledge/backlinks?path=memory/b.md", nil, cookies, "")
	if got.Code != http.StatusOK {
		t.Fatalf("backlinks by path = %d %s", got.Code, got.Body.String())
	}
	var byPath struct {
		Items []struct {
			KnowledgeID string `json:"knowledge_id"`
			NoteID      string `json:"note_id"`
			Path        string `json:"path"`
			Title       string `json:"title"`
		} `json:"items"`
	}
	if err := json.NewDecoder(got.Body).Decode(&byPath); err != nil {
		t.Fatal(err)
	}
	if len(byPath.Items) != 1 {
		t.Fatalf("items = %#v, want 1", byPath.Items)
	}
	if byPath.Items[0].KnowledgeID != from.ID {
		t.Fatalf("knowledge_id = %q want %q", byPath.Items[0].KnowledgeID, from.ID)
	}
	if byPath.Items[0].NoteID != "" {
		t.Fatalf("ambiguous note_id leaked: %q", byPath.Items[0].NoteID)
	}
	if byPath.Items[0].Path != "memory/a.md" || byPath.Items[0].Title != "Note A" {
		t.Fatalf("item = %+v", byPath.Items[0])
	}

	got = apiRequest(t, h, http.MethodGet, "/api/v1/projects/"+pid+"/knowledge/backlinks?knowledge_id="+target.ID, nil, cookies, "")
	if got.Code != http.StatusOK {
		t.Fatalf("backlinks by knowledge_id = %d %s", got.Code, got.Body.String())
	}
	var byID struct {
		Items []struct {
			KnowledgeID string `json:"knowledge_id"`
			NoteID      string `json:"note_id"`
		} `json:"items"`
	}
	if err := json.NewDecoder(got.Body).Decode(&byID); err != nil {
		t.Fatal(err)
	}
	if len(byID.Items) != 1 || byID.Items[0].KnowledgeID != from.ID || byID.Items[0].NoteID != "" {
		t.Fatalf("by knowledge_id items = %#v", byID.Items)
	}
}

func TestKnowledgeNoteBacklinksResolvesSourceMirror(t *testing.T) {
	h, db, dataDir, pid, cookies, clk := knowledgeAPIServer(t)
	sourceRel := "guide/intro.md"
	body := []byte("# Intro\n")
	if err := os.MkdirAll(filepath.Join(layout.SourceDir(layout.ProjectRoot(dataDir, "", pid)), "guide"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(layout.SourceDir(layout.ProjectRoot(dataDir, "", pid)), sourceRel), body, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO notes(id,project_id,relative_path,content_sha256,byte_size,status,revision,created_at,updated_at)
		VALUES('note-src',?,'guide/intro.md','abc',7,'ready',1,'x','x')`, pid); err != nil {
		t.Fatal(err)
	}

	ks := &store.KnowledgeStore{DB: db, Clock: clk}
	ctx := t.Context()
	target, err := ks.UpsertFromContent(ctx, store.UpsertKnowledgeInput{
		Kind:         domain.KnowledgeKindSource,
		ProjectID:    pid,
		RelativePath: "source/guide/intro.md",
		Content:      body,
		Status:       "ready",
		SourceNoteID: "note-src",
	})
	if err != nil {
		t.Fatal(err)
	}
	from, err := ks.UpsertFromContent(ctx, store.UpsertKnowledgeInput{
		Kind:         domain.KnowledgeKindMemoryDetail,
		ProjectID:    pid,
		RelativePath: "memory/pointer.md",
		Content:      []byte("See [[source/guide/intro|Intro]]\n"),
		Status:       "ready",
	})
	if err != nil {
		t.Fatal(err)
	}
	if target.ID == "note-src" {
		t.Fatal("knowledge id must not equal v1 notes.id")
	}

	got := apiRequest(t, h, http.MethodGet, "/api/v1/projects/"+pid+"/notes/note-src/backlinks", nil, cookies, "")
	if got.Code != http.StatusOK {
		t.Fatalf("notes backlinks = %d %s", got.Code, got.Body.String())
	}
	var bodyJSON struct {
		Items []struct {
			KnowledgeID string `json:"knowledge_id"`
			NoteID      string `json:"note_id"`
			Path        string `json:"path"`
		} `json:"items"`
	}
	if err := json.NewDecoder(got.Body).Decode(&bodyJSON); err != nil {
		t.Fatal(err)
	}
	if len(bodyJSON.Items) != 1 || bodyJSON.Items[0].KnowledgeID != from.ID || bodyJSON.Items[0].NoteID != "" || bodyJSON.Items[0].Path != "memory/pointer.md" {
		t.Fatalf("notes backlinks items = %#v", bodyJSON.Items)
	}

	if _, err := db.Exec(`INSERT INTO notes(id,project_id,relative_path,status,revision,created_at,updated_at)
		VALUES('no-mirror',?,'orphan.md','ready',1,'x','x')`, pid); err != nil {
		t.Fatal(err)
	}
	missing := apiRequest(t, h, http.MethodGet, "/api/v1/projects/"+pid+"/notes/no-mirror/backlinks", nil, cookies, "")
	if missing.Code != http.StatusNotFound {
		t.Fatalf("no mirror = %d %s", missing.Code, missing.Body.String())
	}
}

func TestKnowledgeTreeSourceAndMemoryExcludesAgentsAndSessions(t *testing.T) {
	h, _, dataDir, pid, cookies, _ := knowledgeAPIServer(t)
	root := layout.ProjectRoot(dataDir, "", pid)
	if err := os.WriteFile(filepath.Join(layout.SourceDir(root), "alpha.md"), []byte("A\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(layout.MemoryDir(root), "beta.md"), []byte("B\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "sessions", "s1"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sessions", "s1", "secret.md"), []byte("nope\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".agents", "skills", "compounding", "hidden.md"), []byte("nope\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got := apiRequest(t, h, http.MethodGet, "/api/v1/projects/"+pid+"/knowledge/tree", nil, cookies, "")
	if got.Code != http.StatusOK {
		t.Fatalf("tree = %d %s", got.Code, got.Body.String())
	}
	var body struct {
		Entries []struct {
			Path string `json:"path"`
			Kind string `json:"kind"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(got.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, e := range body.Entries {
		seen[e.Path] = e.Kind
		if strings.Contains(e.Path, ".agents") || strings.HasPrefix(e.Path, "sessions") || strings.Contains(e.Path, "/sessions/") {
			t.Fatalf("tree leaked excluded path %q", e.Path)
		}
	}
	if seen["source"] != "directory" || seen["source/alpha.md"] != "file" {
		t.Fatalf("missing source entries: %#v", seen)
	}
	if seen["memory"] != "directory" || seen["memory/beta.md"] != "file" || seen["memory/lessons.md"] != "file" {
		t.Fatalf("missing memory entries: %#v", seen)
	}
	if _, ok := seen["AGENTS.md"]; ok {
		t.Fatalf("tree should be source+memory only, got AGENTS.md")
	}
}

func TestKnowledgeReadRejectsTraversalAndAgents(t *testing.T) {
	h, _, _, pid, cookies, _ := knowledgeAPIServer(t)
	for _, p := range []string{"../escape.md", "memory/../escape.md", ".agents/skills/x.md", "sessions/s1/x.md"} {
		got := apiRequest(t, h, http.MethodGet, "/api/v1/projects/"+pid+"/knowledge/read?path="+p, nil, cookies, "")
		if got.Code != http.StatusBadRequest {
			t.Errorf("read %q = %d %s, want 400", p, got.Code, got.Body.String())
		}
		got = apiRequest(t, h, http.MethodGet, "/api/v1/projects/"+pid+"/knowledge/backlinks?path="+p, nil, cookies, "")
		if got.Code != http.StatusBadRequest {
			t.Errorf("backlinks %q = %d %s, want 400", p, got.Code, got.Body.String())
		}
	}
}

// Break this would catch: search matching both notes, or JSON using note_id
// instead of knowledge_id.
func TestProjectSearchReturnsOneHitAndKnowledgeID(t *testing.T) {
	h, db, _, pid, cookies, clk := knowledgeAPIServer(t)
	ks := &store.KnowledgeStore{DB: db, Clock: clk}
	ctx := t.Context()

	match, err := ks.UpsertFromContent(ctx, store.UpsertKnowledgeInput{
		Kind:         domain.KnowledgeKindMemoryDetail,
		ProjectID:    pid,
		RelativePath: "memory/match.md",
		Content:      []byte("---\ntitle: Match Note\n---\nBody has alphaquark only here.\n"),
		Status:       "ready",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ks.UpsertFromContent(ctx, store.UpsertKnowledgeInput{
		Kind:         domain.KnowledgeKindMemoryDetail,
		ProjectID:    pid,
		RelativePath: "memory/other.md",
		Content:      []byte("---\ntitle: Other Note\n---\nBody has betazebra instead.\n"),
		Status:       "ready",
	}); err != nil {
		t.Fatal(err)
	}

	got := apiRequest(t, h, http.MethodGet, "/api/v1/projects/"+pid+"/search?q=alphaquark&limit=20", nil, cookies, "")
	if got.Code != http.StatusOK {
		t.Fatalf("search = %d %s", got.Code, got.Body.String())
	}
	var body struct {
		Hits []struct {
			KnowledgeID  string `json:"knowledge_id"`
			NoteID       string `json:"note_id"`
			Path         string `json:"path"`
			Title        string `json:"title"`
			Snippet      string `json:"snippet"`
			Kind         string `json:"kind"`
			SourceNoteID string `json:"source_note_id"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(got.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Hits) != 1 {
		t.Fatalf("hits = %#v, want 1", body.Hits)
	}
	hit := body.Hits[0]
	if hit.KnowledgeID != match.ID {
		t.Fatalf("knowledge_id = %q want %q", hit.KnowledgeID, match.ID)
	}
	if hit.NoteID != "" {
		t.Fatalf("ambiguous note_id leaked: %q", hit.NoteID)
	}
	if hit.Path != "memory/match.md" || hit.Title != "Match Note" || hit.Kind != string(domain.KnowledgeKindMemoryDetail) {
		t.Fatalf("hit = %+v", hit)
	}
	if !strings.Contains(hit.Snippet, "alphaquark") {
		t.Fatalf("snippet %q missing alphaquark", hit.Snippet)
	}
}

// Break this would catch: raw FTS5 syntax in q becoming a 500.
func TestProjectSearchFTSSpecialCharsDoNot500(t *testing.T) {
	h, db, _, pid, cookies, clk := knowledgeAPIServer(t)
	ks := &store.KnowledgeStore{DB: db, Clock: clk}
	if _, err := ks.UpsertFromContent(t.Context(), store.UpsertKnowledgeInput{
		Kind:         domain.KnowledgeKindMemoryDetail,
		ProjectID:    pid,
		RelativePath: "memory/safe.md",
		Content:      []byte("needleword sits in the body.\n"),
		Status:       "ready",
	}); err != nil {
		t.Fatal(err)
	}

	for _, q := range []string{`"`, `*`, `AND`, `OR`, `NOT`, `"foo*bar"`, `foo AND bar`, `NEEDLEWORD*`, `"needleword"`} {
		path := "/api/v1/projects/" + pid + "/search?q=" + url.QueryEscape(q)
		got := apiRequest(t, h, http.MethodGet, path, nil, cookies, "")
		if got.Code == http.StatusInternalServerError {
			t.Fatalf("query %q = 500 %s", q, got.Body.String())
		}
		if got.Code != http.StatusOK {
			t.Fatalf("query %q = %d %s, want 200", q, got.Code, got.Body.String())
		}
		var body struct {
			Hits []json.RawMessage `json:"hits"`
		}
		if err := json.NewDecoder(got.Body).Decode(&body); err != nil {
			t.Fatalf("query %q json: %v body=%s", q, err, got.Body.String())
		}
		if body.Hits == nil {
			t.Fatalf("query %q hits is null, want array", q)
		}
	}
}

func TestKnowledgeRoutesRequireAuth(t *testing.T) {
	h, _, _, pid, _, _ := knowledgeAPIServer(t)
	for _, path := range []string{
		"/api/v1/projects/" + pid + "/knowledge/read?path=memory/lessons.md",
		"/api/v1/projects/" + pid + "/knowledge/tree",
		"/api/v1/projects/" + pid + "/knowledge/backlinks?path=memory/lessons.md",
		"/api/v1/projects/" + pid + "/notes/note-src/backlinks",
		"/api/v1/projects/" + pid + "/search?q=alphaquark",
	} {
		got := apiRequest(t, h, http.MethodGet, path, nil, nil, "")
		if got.Code != http.StatusUnauthorized {
			t.Errorf("anonymous %s = %d", path, got.Code)
		}
	}
}
