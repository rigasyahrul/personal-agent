package httpapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/auth"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/config"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func TestBootstrapLoginMeLogoutAndCSRF(t *testing.T) {
	d, _ := testutil.TempDB(t)
	fake := &clock.FakeClock{T: time.Unix(1000, 0).UTC()}
	mux := http.NewServeMux()
	AuthRoutes(mux, AuthDeps{DB: d, Clock: fake, BootstrapToken: "secret", SecureCookies: true})
	post := func(path, body string, cookies []*http.Cookie, csrf string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
		r.Header.Set("Content-Type", "application/json")
		if csrf != "" {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		for _, c := range cookies {
			r.AddCookie(c)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}

	status := httptest.NewRecorder()
	mux.ServeHTTP(status, httptest.NewRequest(http.MethodGet, "/api/v1/setup/status", nil))
	if status.Code != http.StatusOK || status.Body.String() != "{\"bootstrapped\":false}\n" {
		t.Fatalf("initial status: %d %q", status.Code, status.Body.String())
	}
	if got := post("/api/v1/setup/bootstrap", `{"token":"secret","password":"short"}`, nil, "").Code; got != http.StatusBadRequest {
		t.Fatalf("short password status = %d", got)
	}
	if got := post("/api/v1/setup/bootstrap", `{"token":"wrong","password":"long-enough-password"}`, nil, "").Code; got != http.StatusForbidden {
		t.Fatalf("bad token status = %d", got)
	}
	if got := post("/api/v1/setup/bootstrap", `{"token":"secret","password":"long-enough-password"}`, nil, "").Code; got != http.StatusCreated {
		t.Fatalf("bootstrap status = %d", got)
	}
	if got := post("/api/v1/setup/bootstrap", `{"token":"secret","password":"another-long-password"}`, nil, "").Code; got != http.StatusConflict {
		t.Fatalf("second bootstrap status = %d", got)
	}
	if got := post("/api/v1/auth/login", `{"password":"wrong-password"}`, nil, "").Code; got != http.StatusUnauthorized {
		t.Fatalf("bad login status = %d", got)
	}
	w := post("/api/v1/auth/login", `{"password":"long-enough-password"}`, nil, "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	var session, csrf *http.Cookie
	for _, c := range cookies {
		switch c.Name {
		case "pa_session":
			session = c
		case "pa_csrf":
			csrf = c
		}
	}
	if session == nil || csrf == nil || !session.HttpOnly || csrf.HttpOnly || !session.Secure || !csrf.Secure || session.SameSite != http.SameSiteLaxMode || csrf.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie attributes: session=%+v csrf=%+v", session, csrf)
	}
	if !session.Expires.Equal(fake.Now().Add(30*24*time.Hour)) || !csrf.Expires.Equal(fake.Now().Add(30*24*time.Hour)) {
		t.Fatalf("cookie expiry: session=%v csrf=%v", session.Expires, csrf.Expires)
	}
	var storedHash string
	if err := d.QueryRow("SELECT token_hash FROM auth_sessions").Scan(&storedHash); err != nil || storedHash != auth.TokenHash(session.Value) || storedHash == session.Value {
		t.Fatalf("stored token hash = %q, err=%v", storedHash, err)
	}

	request := func(method, path string, supplied []*http.Cookie, csrfHeader string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		for _, c := range supplied {
			r.AddCookie(c)
		}
		r.Header.Set("X-CSRF-Token", csrfHeader)
		out := httptest.NewRecorder()
		mux.ServeHTTP(out, r)
		return out
	}
	me := request(http.MethodGet, "/api/v1/auth/me", cookies, "")
	var body map[string]any
	_ = json.NewDecoder(me.Body).Decode(&body)
	if me.Code != http.StatusOK || body["owner"] != true {
		t.Fatalf("me: %d %#v", me.Code, body)
	}
	if got := request(http.MethodPost, "/api/v1/auth/logout", nil, "wrong").Code; got != http.StatusUnauthorized {
		t.Fatalf("logout must authenticate before csrf, got %d", got)
	}
	if got := request(http.MethodPost, "/api/v1/auth/logout", cookies, "wrong").Code; got != http.StatusForbidden {
		t.Fatalf("csrf bypass status = %d", got)
	}
	logout := request(http.MethodPost, "/api/v1/auth/logout", cookies, csrf.Value)
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout status = %d", logout.Code)
	}
	if got := request(http.MethodGet, "/api/v1/auth/me", cookies, "").Code; got != http.StatusUnauthorized {
		t.Fatalf("deleted session accepted: %d", got)
	}
}

func TestAuthRoutesUseInjectedClockForExpiry(t *testing.T) {
	d, _ := testutil.TempDB(t)
	fake := &clock.FakeClock{T: time.Unix(1000, 0).UTC()}
	if _, err := d.Exec("INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,'hash','x','x')"); err != nil {
		t.Fatal(err)
	}
	token := "raw-token"
	if _, err := d.Exec("INSERT INTO auth_sessions(token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,?)", auth.TokenHash(token), "csrf", fake.Now().Add(time.Minute).Format(time.RFC3339Nano), "x"); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	AuthRoutes(mux, AuthDeps{DB: d, Clock: fake})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	r.AddCookie(&http.Cookie{Name: "pa_session", Value: token})
	fake.Advance(2 * time.Minute)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expired session status = %d", w.Code)
	}
}

func TestLoginReturnsInternalServerErrorOnDatabaseFailure(t *testing.T) {
	d, _ := testutil.TempDB(t)
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	AuthRoutes(mux, AuthDeps{DB: d, Clock: &clock.FakeClock{T: time.Unix(1000, 0).UTC()}})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"password":"long-enough-password"}`))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("login database failure status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestRequireAuthReturnsInternalServerErrorOnDatabaseFailure(t *testing.T) {
	d, _ := testutil.TempDB(t)
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler called")
	})
	r := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	r.AddCookie(&http.Cookie{Name: "pa_session", Value: "token"})
	w := httptest.NewRecorder()

	RequireAuth(d, next).ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("session database failure status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// httpFixture wraps the full server for security boundary tests.
type httpFixture struct {
	t              *testing.T
	handler        http.Handler
	db             *sql.DB
	bootstrapToken string
	clock          *clock.FakeClock
}

func newHTTPFixture(t *testing.T) *httpFixture {
	t.Helper()
	db, dir := testutil.TempDB(t)
	now := time.Unix(2000, 0).UTC()
	fake := &clock.FakeClock{T: now}
	token := "dev-bootstrap-token-32chars-min!!"
	h := New(ServerDeps{
		DB:             db,
		DataDir:        dir,
		Clock:          fake,
		BootstrapToken: token,
		SecureCookies:  false,
		Models:         []config.ModelRef{{Provider: "openai", ModelID: "test"}},
	})
	return &httpFixture{t: t, handler: h, db: db, bootstrapToken: token, clock: fake}
}

func (fx *httpFixture) request(method, path, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	return fx.requestWithHeaders(method, path, body, cookies, nil)
}

func (fx *httpFixture) requestWithHeaders(method, path, body string, cookies []*http.Cookie, headers map[string]string) *httptest.ResponseRecorder {
	fx.t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	for _, c := range cookies {
		r.AddCookie(c)
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	fx.handler.ServeHTTP(w, r)
	return w
}

func (fx *httpFixture) login(t *testing.T) (*http.Cookie, *http.Cookie) {
	t.Helper()
	// Ensure owner exists.
	if res := fx.requestWithHeaders("POST", "/api/v1/setup/bootstrap", `{"password":"first secure password"}`, nil, map[string]string{"Authorization": "Bearer " + fx.bootstrapToken}); res.Code != http.StatusCreated && res.Code != http.StatusConflict {
		t.Fatalf("bootstrap status=%d body=%s", res.Code, res.Body.String())
	}
	res := fx.loginPassword("first secure password")
	if res.Code != http.StatusOK && res.Code != http.StatusNoContent {
		t.Fatalf("login status=%d body=%s", res.Code, res.Body.String())
	}
	var session, csrf *http.Cookie
	for _, c := range res.Result().Cookies() {
		switch c.Name {
		case "pa_session":
			session = c
		case "pa_csrf":
			csrf = c
		}
	}
	if session == nil || csrf == nil {
		t.Fatal("missing auth cookies")
	}
	return session, csrf
}

func (fx *httpFixture) loginPassword(password string) *httptest.ResponseRecorder {
	return fx.request("POST", "/api/v1/auth/login", `{"password":`+mustJSON(password)+`}`, nil)
}

func mustJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestUnauthenticatedMutationsReturn401(t *testing.T) {
	fx := newHTTPFixture(t)
	tests := []struct{ method, path, body string }{
		{"PUT", "/api/v1/settings", `{"timezone":"UTC"}`},
		{"POST", "/api/v1/projects", `{"name":"x"}`},
		{"POST", "/api/v1/projects/p1/folders", `{"path":"x"}`},
		{"POST", "/api/v1/projects/p1/direct-notes", `{"path":"x.md","body":"x"}`},
		{"POST", "/api/v1/projects/p1/sessions", `{"title":"x","provider":"openai","model_id":"test"}`},
		{"DELETE", "/api/v1/sessions/s1", ``},
		{"POST", "/api/v1/sessions/s1/messages", `{"content":"x","request_key":"k"}`},
		{"POST", "/api/v1/sessions/s1/promote", `{"path":"x.md","target_path":"notes/x.md","review_mode":"whole","request_key":"k"}`},
		{"POST", "/api/v1/review/items/r1/rate", `{"rating":"good","request_key":"k"}`},
		{"POST", "/api/v1/review/items/r1/suspend", `{}`},
		{"POST", "/api/v1/review/pending/p1/retry", `{}`},
		{"POST", "/api/v1/backups", `{}`},
	}
	for _, tc := range tests {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			res := fx.request(tc.method, tc.path, tc.body, nil)
			if res.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
			}
		})
	}
}

func TestCSRFFailureReturns403(t *testing.T) {
	fx := newHTTPFixture(t)
	session, csrf := fx.login(t)
	// Missing CSRF header.
	res := fx.request("POST", "/api/v1/projects", `{"name":"x"}`, []*http.Cookie{session, csrf})
	if res.Code != http.StatusForbidden {
		t.Fatalf("missing header status=%d body=%s", res.Code, res.Body.String())
	}
	reqCookies := []*http.Cookie{session, {Name: "pa_csrf", Value: "cookie-value"}}
	res = fx.requestWithHeaders("POST", "/api/v1/projects", `{"name":"x"}`, reqCookies, map[string]string{"X-CSRF-Token": "header-value"})
	if res.Code != http.StatusForbidden {
		t.Fatalf("mismatch status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestBootstrapTakeoverBlockedWhenOwnerExists(t *testing.T) {
	fx := newHTTPFixture(t)
	first := fx.requestWithHeaders("POST", "/api/v1/setup/bootstrap", `{"password":"first secure password"}`, nil, map[string]string{"Authorization": "Bearer " + fx.bootstrapToken})
	if first.Code != http.StatusCreated {
		t.Fatalf("first status=%d body=%s", first.Code, first.Body.String())
	}
	second := fx.requestWithHeaders("POST", "/api/v1/setup/bootstrap", `{"password":"attacker password"}`, nil, map[string]string{"Authorization": "Bearer " + fx.bootstrapToken})
	if second.Code != http.StatusConflict {
		t.Fatalf("second status=%d body=%s", second.Code, second.Body.String())
	}
	if fx.loginPassword("first secure password").Code != http.StatusNoContent {
		t.Fatal("original owner password no longer works")
	}
	if fx.loginPassword("attacker password").Code == http.StatusNoContent {
		t.Fatal("takeover password was accepted")
	}
}
