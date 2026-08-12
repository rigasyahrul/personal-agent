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
