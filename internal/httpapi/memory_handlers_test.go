package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/auth"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func memoryAPIServer(t *testing.T) (http.Handler, string, string, []*http.Cookie) {
	t.Helper()
	db, dir := testutil.TempDB(t)
	now := time.Date(2026, 8, 22, 16, 0, 0, 0, time.UTC)
	stamp := now.Format(time.RFC3339Nano)
	if _, err := db.Exec(`INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,'x',?,?)`, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO auth_sessions(token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,?)`,
		auth.TokenHash("token"), "csrf", now.Add(time.Hour).Format(time.RFC3339Nano), stamp); err != nil {
		t.Fatal(err)
	}
	h := New(ServerDeps{DB: db, DataDir: dir, Clock: &clock.FakeClock{T: now}})
	cookies := []*http.Cookie{
		{Name: "pa_session", Value: "token"},
		{Name: "pa_csrf", Value: "csrf"},
	}
	created := apiRequest(t, h, http.MethodPost, "/api/v1/projects", map[string]any{
		"name":     "Memory",
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
	return h, dir, proj.ID, cookies
}

func TestMemoryLessonsReturnsFileContents(t *testing.T) {
	h, dataDir, pid, cookies := memoryAPIServer(t)
	want := "# Lessons\n\n- [[memory/20260822-1600-example|example]] one-line summary\n"
	path := layout.LessonsPath(layout.ProjectRoot(dataDir, "", pid))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}

	got := apiRequest(t, h, http.MethodGet, "/api/v1/projects/"+pid+"/memory/lessons", nil, cookies, "")
	if got.Code != http.StatusOK {
		t.Fatalf("GET lessons = %d %s", got.Code, got.Body.String())
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(got.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Content != want {
		t.Fatalf("content = %q want %q", body.Content, want)
	}
}

func TestMemoryLessonsMissingFileEmpty(t *testing.T) {
	h, dataDir, pid, cookies := memoryAPIServer(t)
	path := layout.LessonsPath(layout.ProjectRoot(dataDir, "", pid))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	got := apiRequest(t, h, http.MethodGet, "/api/v1/projects/"+pid+"/memory/lessons", nil, cookies, "")
	if got.Code != http.StatusOK {
		t.Fatalf("GET missing lessons = %d %s", got.Code, got.Body.String())
	}
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(got.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Content != "" {
		t.Fatalf("missing file content = %q want empty", body.Content)
	}
}

func TestMemoryLessonsUnknownProjectNotFound(t *testing.T) {
	h, _, _, cookies := memoryAPIServer(t)
	got := apiRequest(t, h, http.MethodGet, "/api/v1/projects/missing/memory/lessons", nil, cookies, "")
	if got.Code != http.StatusNotFound {
		t.Fatalf("unknown project = %d %s", got.Code, got.Body.String())
	}
	if !bytes.Contains(got.Body.Bytes(), []byte("project not found")) {
		t.Fatalf("unknown project body = %q, want project not found", got.Body.String())
	}
}

func TestMemoryLessonsAnonymousUnauthorized(t *testing.T) {
	h, _, pid, _ := memoryAPIServer(t)
	got := apiRequest(t, h, http.MethodGet, "/api/v1/projects/"+pid+"/memory/lessons", nil, nil, "")
	if got.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous GET = %d want 401", got.Code)
	}
}
