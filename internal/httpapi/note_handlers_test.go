package httpapi

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/auth"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func TestDirectNoteHandlerValidationIdempotencyAndVaultPlacement(t *testing.T) {
	db, dataDir := testutil.TempDB(t)
	now := time.Now().UTC()
	_, _ = db.Exec("INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,'x','x','x')")
	_, _ = db.Exec("INSERT INTO auth_sessions(token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,'x')", auth.TokenHash(testSession), testCSRF, now.Add(time.Hour).Format(time.RFC3339Nano))
	_, _ = db.Exec("INSERT INTO vaults(id,name,created_at,updated_at) VALUES('v1','V','x','x'); INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES('p1','v1','P','x','x')")
	_ = layout.EnsureProjectDirs(dataDir, "v1", "p1")
	h := New(ServerDeps{DB: db, DataDir: dataDir, Clock: &clock.FakeClock{T: now}})
	request := func(key, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/v1/projects/p1/direct-notes", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", key)
		r.Header.Set("X-CSRF-Token", testCSRF)
		r.AddCookie(&http.Cookie{Name: "pa_session", Value: testSession})
		r.AddCookie(&http.Cookie{Name: "pa_csrf", Value: testCSRF})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	good := `{"path":"guide/new.md","review_mode":"none","body":"# New\n"}`
	first := request("key", good)
	if first.Code != 201 {
		t.Fatalf("first=%d %s", first.Code, first.Body.String())
	}
	retry := request("key", good)
	if retry.Code != 200 && retry.Code != 201 {
		t.Fatalf("retry=%d", retry.Code)
	}
	if got := request("key", `{"path":"guide/other.md","review_mode":"none","body":"x"}`).Code; got != 409 {
		t.Fatalf("conflict=%d", got)
	}
	if b, err := os.ReadFile(filepath.Join(layout.SourceDir(layout.ProjectRoot(dataDir, "v1", "p1")), "guide", "new.md")); err != nil || string(b) != "# New\n" {
		t.Fatalf("vault bytes=%q %v", b, err)
	}
	for _, tc := range []struct{ key, body string }{{"", good}, {"x", `{"path":"x.txt","review_mode":"none","body":"x"}`}, {"x", `{"path":"x.md","review_mode":"bad","body":"x"}`}} {
		if got := request(tc.key, tc.body).Code; got != 400 {
			t.Errorf("invalid got %d", got)
		}
	}
}

func TestNoteHandlersAndFolders(t *testing.T) {
	db, dataDir := testutil.TempDB(t)
	now := time.Now().UTC()
	_, _ = db.Exec("INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,'x','x','x')")
	_, _ = db.Exec("INSERT INTO auth_sessions(token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,'x')", auth.TokenHash(testSession), testCSRF, now.Add(time.Hour).Format(time.RFC3339Nano))
	_, _ = db.Exec("INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','P','x','x')")
	source := layout.SourceDir(layout.ProjectRoot(dataDir, "", "p1"))
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	body := []byte("# body\n")
	sum := sha256.Sum256(body)
	if err := os.WriteFile(filepath.Join(source, "note.md"), body, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO notes(id,project_id,relative_path,content_sha256,byte_size,status,revision,created_at,updated_at) VALUES('n1','p1','note.md',?,?,'ready',1,'x','x')`, fmt.Sprintf("%x", sum), len(body)); err != nil {
		t.Fatal(err)
	}
	h := New(ServerDeps{DB: db, DataDir: dataDir, Clock: &clock.FakeClock{T: now}})
	for _, path := range []string{"/api/v1/projects/p1/tree", "/api/v1/notes/n1"} {
		if got := projectAPIRequest(h, "GET", path, "", false, "").Code; got != 401 {
			t.Errorf("anonymous %s = %d", path, got)
		}
	}
	if got := projectAPIRequest(h, "POST", "/api/v1/projects/p1/folders", `{"path":"docs/api"}`, true, testCSRF); got.Code != 201 {
		t.Fatalf("mkdir = %d %s", got.Code, got.Body.String())
	}
	if got := projectAPIRequest(h, "POST", "/api/v1/projects/p1/folders", `{"path":"docs/api"}`, true, testCSRF).Code; got != 409 {
		t.Errorf("existing = %d", got)
	}
	for _, p := range []string{"memory/x", "soul", "docs/x.md", "../escape"} {
		if got := projectAPIRequest(h, "POST", "/api/v1/projects/p1/folders", `{"path":"`+p+`"}`, true, testCSRF).Code; got != 400 {
			t.Errorf("invalid %q = %d", p, got)
		}
	}
	read := projectAPIRequest(h, "GET", "/api/v1/notes/n1", "", true, "")
	if read.Code != 200 {
		t.Fatalf("read = %d %s", read.Code, read.Body.String())
	}
	var decoded map[string]any
	if err := json.NewDecoder(read.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["body"] != "# body\n" {
		t.Fatalf("body = %#v", decoded["body"])
	}
	if err := os.WriteFile(filepath.Join(source, "note.md"), []byte("bad"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := projectAPIRequest(h, "GET", "/api/v1/notes/n1", "", true, ""); got.Code != 409 {
		t.Fatalf("integrity = %d %s", got.Code, got.Body.String())
	}
}
