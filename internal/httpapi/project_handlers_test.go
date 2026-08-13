package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/auth"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

const (
	testSession = "project-api-session"
	testCSRF    = "project-api-csrf"
)

func newProjectAPITestServer(t *testing.T) (http.Handler, *clock.FakeClock) {
	t.Helper()
	db, _ := testutil.TempDB(t)
	now := time.Date(2026, 8, 13, 12, 30, 0, 0, time.UTC)
	if _, err := db.Exec("INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,'hash','x','x')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO auth_sessions(token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,?)", auth.TokenHash(testSession), testCSRF, now.Add(time.Hour).Format(time.RFC3339Nano), "x"); err != nil {
		t.Fatal(err)
	}
	fake := &clock.FakeClock{T: now}
	return New(ServerDeps{DB: db, DataDir: t.TempDir(), Clock: fake}), fake
}

func projectAPIRequest(handler http.Handler, method, path, body string, authenticated bool, csrf string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	if authenticated {
		r.AddCookie(&http.Cookie{Name: "pa_session", Value: testSession})
		r.AddCookie(&http.Cookie{Name: "pa_csrf", Value: testCSRF})
	}
	r.Header.Set("X-CSRF-Token", csrf)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	return w
}

func TestProjectRoutesRequireAuthenticationAndCSRF(t *testing.T) {
	h, _ := newProjectAPITestServer(t)
	for _, path := range []string{"/api/v1/vaults", "/api/v1/projects", "/api/v1/projects/missing", "/api/v1/home"} {
		if got := projectAPIRequest(h, http.MethodGet, path, "", false, "").Code; got != http.StatusUnauthorized {
			t.Errorf("anonymous GET %s = %d", path, got)
		}
	}
	for _, path := range []string{"/api/v1/vaults", "/api/v1/projects"} {
		if got := projectAPIRequest(h, http.MethodPost, path, `{}`, false, "").Code; got != http.StatusUnauthorized {
			t.Errorf("anonymous POST %s = %d, want auth before CSRF", path, got)
		}
		if got := projectAPIRequest(h, http.MethodPost, path, `{}`, true, "").Code; got != http.StatusForbidden {
			t.Errorf("authenticated POST %s without CSRF = %d", path, got)
		}
	}
}

func TestVaultProjectAndHomeAPIs(t *testing.T) {
	h, fake := newProjectAPITestServer(t)
	unvaultedResponse := projectAPIRequest(h, http.MethodPost, "/api/v1/projects", `{"name":"Inbox","vault_id":null}`, true, testCSRF)
	if unvaultedResponse.Code != http.StatusCreated {
		t.Fatalf("create unvaulted project = %d %s", unvaultedResponse.Code, unvaultedResponse.Body.String())
	}
	var unvaulted map[string]any
	if err := json.NewDecoder(unvaultedResponse.Body).Decode(&unvaulted); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"id", "vault_id", "vault_name", "name", "note_count", "session_count", "due_count"} {
		if _, ok := unvaulted[key]; !ok {
			t.Errorf("unvaulted project response missing %q: %#v", key, unvaulted)
		}
	}
	if unvaulted["vault_id"] != "" || unvaulted["vault_name"] != "" {
		t.Errorf("unvaulted project vault fields = %#v, %#v", unvaulted["vault_id"], unvaulted["vault_name"])
	}

	vaultResponse := projectAPIRequest(h, http.MethodPost, "/api/v1/vaults", `{"name":" Work "}`, true, testCSRF)
	if vaultResponse.Code != http.StatusCreated {
		t.Fatalf("create vault = %d %s", vaultResponse.Code, vaultResponse.Body.String())
	}
	var vault struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(vaultResponse.Body).Decode(&vault); err != nil {
		t.Fatal(err)
	}
	if vault.ID == "" || vault.Name != "Work" {
		t.Fatalf("vault = %#v", vault)
	}

	projectResponse := projectAPIRequest(h, http.MethodPost, "/api/v1/projects", `{"name":"Go","vault_id":"`+vault.ID+`"}`, true, testCSRF)
	if projectResponse.Code != http.StatusCreated {
		t.Fatalf("create project = %d %s", projectResponse.Code, projectResponse.Body.String())
	}
	var project ProjectDTO
	if err := json.NewDecoder(projectResponse.Body).Decode(&project); err != nil {
		t.Fatal(err)
	}
	if project.ID == "" || project.VaultID != vault.ID || project.VaultName != "Work" || project.Name != "Go" || project.NoteCount != 0 || project.SessionCount != 0 || project.DueCount != 0 {
		t.Fatalf("project = %#v", project)
	}

	for _, path := range []string{"/api/v1/projects", "/api/v1/projects/" + project.ID} {
		response := projectAPIRequest(h, http.MethodGet, path, "", true, "")
		if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"vault_name":"Work"`)) || !bytes.Contains(response.Body.Bytes(), []byte(`"note_count":0`)) || !bytes.Contains(response.Body.Bytes(), []byte(`"session_count":0`)) || !bytes.Contains(response.Body.Bytes(), []byte(`"due_count":0`)) {
			t.Errorf("GET %s = %d %s", path, response.Code, response.Body.String())
		}
	}
	vaults := projectAPIRequest(h, http.MethodGet, "/api/v1/vaults", "", true, "")
	if vaults.Code != http.StatusOK || !bytes.Contains(vaults.Body.Bytes(), []byte(`"name":"Work"`)) {
		t.Fatalf("list vaults = %d %s", vaults.Code, vaults.Body.String())
	}
	home := projectAPIRequest(h, http.MethodGet, "/api/v1/home", "", true, "")
	if home.Code != http.StatusOK || !bytes.Contains(home.Body.Bytes(), []byte(`"generated_at":"`+fake.Now().Format(time.RFC3339Nano)+`"`)) || !bytes.Contains(home.Body.Bytes(), []byte(`"vault_name":"Work"`)) {
		t.Fatalf("home = %d %s", home.Code, home.Body.String())
	}
}

func TestProjectAPIErrors(t *testing.T) {
	h, _ := newProjectAPITestServer(t)
	for _, tc := range []struct {
		path string
		body string
	}{
		{"/api/v1/vaults", `{"name":" "}`},
		{"/api/v1/projects", `{"name":" "}`},
		{"/api/v1/projects", `{"name":"Go","vault_id":"missing"}`},
		{"/api/v1/projects", `{`},
	} {
		if got := projectAPIRequest(h, http.MethodPost, tc.path, tc.body, true, testCSRF).Code; got != http.StatusBadRequest {
			t.Errorf("POST %s body %q = %d", tc.path, tc.body, got)
		}
	}
	if got := projectAPIRequest(h, http.MethodGet, "/api/v1/projects/missing", "", true, "").Code; got != http.StatusNotFound {
		t.Fatalf("missing project = %d", got)
	}
}

func TestProjectAPIReturnsInternalServerErrorOnDatabaseFailure(t *testing.T) {
	db, _ := testutil.TempDB(t)
	now := time.Unix(1000, 0).UTC()
	if _, err := db.Exec("INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,'hash','x','x')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO auth_sessions(token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,?)", auth.TokenHash(testSession), testCSRF, now.Add(time.Hour).Format(time.RFC3339Nano), "x"); err != nil {
		t.Fatal(err)
	}
	h := New(ServerDeps{DB: db, DataDir: t.TempDir(), Clock: &clock.FakeClock{T: now}})
	if _, err := db.Exec("DROP TABLE projects"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/v1/projects", "/api/v1/projects/id", "/api/v1/home"} {
		if got := projectAPIRequest(h, http.MethodGet, path, "", true, "").Code; got != http.StatusInternalServerError {
			t.Errorf("database failure GET %s = %d", path, got)
		}
	}
}
