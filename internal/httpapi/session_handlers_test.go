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
	"github.com/rigasyahrul/personal-agent/internal/config"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/store"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func sessionAPIServer(t *testing.T, models []config.ModelRef) (http.Handler, string, []*http.Cookie) {
	t.Helper()
	db, dir := testutil.TempDB(t)
	now := time.Unix(1000, 0).UTC()
	if _, err := db.Exec(`INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,'x',?,?)`, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO auth_sessions(token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,?)`, auth.TokenHash("token"), "csrf", now.Add(time.Hour).Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO vaults(id,name,created_at,updated_at) VALUES('v','v',?,?)`, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES('p','v','p',?,?)`, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	h := New(ServerDeps{DB: db, DataDir: dir, Clock: &clock.FakeClock{T: now}, Models: models})
	return h, "p", []*http.Cookie{{Name: "pa_session", Value: "token"}, {Name: "pa_csrf", Value: "csrf"}}
}

func apiRequest(t *testing.T, h http.Handler, method, path string, body any, cookies []*http.Cookie, csrf string) *httptest.ResponseRecorder {
	t.Helper()
	var b bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&b).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	r := httptest.NewRequest(method, path, &b)
	for _, c := range cookies {
		r.AddCookie(c)
	}
	if csrf != "" {
		r.Header.Set("X-CSRF-Token", csrf)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func rawAPIRequest(t *testing.T, h http.Handler, method, path, body string, cookies []*http.Cookie, csrf string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	for _, c := range cookies {
		r.AddCookie(c)
	}
	if csrf != "" {
		r.Header.Set("X-CSRF-Token", csrf)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestSessionAPIAuthModelsCreateGetListDeleteAndImmutable(t *testing.T) {
	h, pid, cookies := sessionAPIServer(t, []config.ModelRef{{Provider: "openai", ModelID: "m"}})
	if got := apiRequest(t, h, "GET", "/api/v1/models", nil, nil, "").Code; got != 401 {
		t.Fatalf("anonymous models=%d", got)
	}
	models := apiRequest(t, h, "GET", "/api/v1/models", nil, cookies, "")
	if models.Body.String() != `{"models":[{"provider":"openai","model_id":"m"}]}`+"\n" {
		t.Fatalf("models=%s", models.Body.String())
	}
	path := "/api/v1/projects/" + pid + "/sessions"
	body := map[string]any{"home": "project", "title": "x", "provider": "openai", "model_id": "m"}
	if got := apiRequest(t, h, "POST", path, body, nil, "bad").Code; got != 401 {
		t.Fatalf("anonymous create=%d", got)
	}
	if got := apiRequest(t, h, "POST", path, body, cookies, "bad").Code; got != 403 {
		t.Fatalf("csrf create=%d", got)
	}
	created := apiRequest(t, h, "POST", path, body, cookies, "csrf")
	if created.Code != 201 {
		t.Fatalf("create=%d %s", created.Code, created.Body.String())
	}
	var s domain.Session
	if err := json.Unmarshal(created.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	if s.ModelParametersJSON != "{}" || s.ToolGrantsJSON != `{"workspace_files":false}` {
		t.Fatalf("defaults=%+v", s)
	}
	if got := apiRequest(t, h, "PUT", "/api/v1/sessions/"+s.ID, body, cookies, "csrf").Code; got != 405 {
		t.Fatalf("put=%d", got)
	}
	for _, p := range []string{path, "/api/v1/sessions/" + s.ID} {
		if got := apiRequest(t, h, "GET", p, nil, cookies, "").Code; got != 200 {
			t.Fatalf("get %s=%d", p, got)
		}
	}
	if got := apiRequest(t, h, "DELETE", "/api/v1/sessions/"+s.ID, nil, cookies, "csrf").Code; got != 204 {
		t.Fatalf("delete=%d", got)
	}
	if got := apiRequest(t, h, "GET", "/api/v1/sessions/missing", nil, cookies, "").Code; got != 404 {
		t.Fatalf("missing=%d", got)
	}
}

func TestSessionAPICreateValidation(t *testing.T) {
	h, pid, cookies := sessionAPIServer(t, nil)
	path := "/api/v1/projects/" + pid + "/sessions"
	valid := map[string]any{"provider": "openai", "model_id": "m"}
	if got := apiRequest(t, h, "POST", path, valid, cookies, "csrf").Code; got != 503 {
		t.Fatalf("empty models=%d", got)
	}
	h, pid, cookies = sessionAPIServer(t, []config.ModelRef{{Provider: "openai", ModelID: "m"}})
	path = "/api/v1/projects/" + pid + "/sessions"
	for _, body := range []any{map[string]any{"home": "vault", "title": "x", "provider": "openai", "model_id": "m"}, map[string]any{"title": "x", "provider": "openai", "model_id": "other"}, map[string]any{"title": "x", "provider": "openai", "model_id": "m", "tool_grants": map[string]bool{"other": true}}, map[string]any{"title": " \t", "provider": "openai", "model_id": "m"}, "bad"} {
		if got := apiRequest(t, h, "POST", path, body, cookies, "csrf").Code; got != 400 {
			t.Fatalf("invalid %#v=%d", body, got)
		}
	}
	trailing := `{"title":"valid first object","provider":"openai","model_id":"m"}{"title":"second object","provider":"openai","model_id":"m"}`
	res := rawAPIRequest(t, h, "POST", path, trailing, cookies, "csrf")
	if res.Code != http.StatusBadRequest || res.Body.String() != `{"error":"invalid_body"}`+"\n" {
		t.Fatalf("trailing JSON=%d body=%q", res.Code, res.Body.String())
	}
	if got := apiRequest(t, h, "POST", "/api/v1/projects/missing/sessions", valid, cookies, "csrf").Code; got != 404 {
		t.Fatalf("missing project=%d", got)
	}
	if got := apiRequest(t, h, "GET", "/api/v1/projects/missing/sessions", nil, cookies, "").Code; got != 404 {
		t.Fatalf("list missing project=%d", got)
	}
}

func TestSessionAPIDeleteAuthenticatesBeforeCSRF(t *testing.T) {
	h, _, _ := sessionAPIServer(t, []config.ModelRef{{Provider: "openai", ModelID: "m"}})
	if got := apiRequest(t, h, "DELETE", "/api/v1/sessions/missing", nil, nil, "bad").Code; got != http.StatusUnauthorized {
		t.Fatalf("anonymous delete=%d", got)
	}
}

func TestProjectSessionsDatabaseFailureIsInternalError(t *testing.T) {
	db, dir := testutil.TempDB(t)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	h := &sessionHandlers{sessions: &store.SessionStore{DB: db, DataDir: dir}}
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		r := httptest.NewRequest(method, "/api/v1/projects/p/sessions", nil)
		r.SetPathValue("id", "p")
		w := httptest.NewRecorder()
		h.projectSessions(w, r)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("%s database failure=%d", method, w.Code)
		}
	}
}
