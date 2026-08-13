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

func TestSettingsRoutesAuthCSRFAndUpdate(t *testing.T) {
	db, _ := testutil.TempDB(t)
	now := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)
	if _, err := db.Exec("INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,'hash','x','x')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO auth_sessions(token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,?)", auth.TokenHash("session"), "csrf", now.Add(time.Hour).Format(time.RFC3339Nano), "x"); err != nil {
		t.Fatal(err)
	}
	h := New(ServerDeps{DB: db, DataDir: t.TempDir(), Clock: &clock.FakeClock{T: now}})
	request := func(method, body string, authenticated bool, csrf string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "/api/v1/settings", bytes.NewBufferString(body))
		if authenticated {
			r.AddCookie(&http.Cookie{Name: "pa_session", Value: "session"})
			r.AddCookie(&http.Cookie{Name: "pa_csrf", Value: "csrf"})
		}
		r.Header.Set("X-CSRF-Token", csrf)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if got := request(http.MethodGet, "", false, "").Code; got != http.StatusUnauthorized {
		t.Fatalf("anonymous GET = %d", got)
	}
	if got := request(http.MethodPut, `{}`, false, "").Code; got != http.StatusUnauthorized {
		t.Fatalf("PUT auth before CSRF = %d", got)
	}
	if got := request(http.MethodPut, `{}`, true, "").Code; got != http.StatusForbidden {
		t.Fatalf("missing CSRF = %d", got)
	}
	put := request(http.MethodPut, `{"timezone":"Asia/Jakarta","default_provider":"openai","default_model_id":"gpt-5","backup_schedule":"daily"}`, true, "csrf")
	if put.Code != http.StatusOK {
		t.Fatalf("PUT = %d %s", put.Code, put.Body.String())
	}
	get := request(http.MethodGet, "", true, "")
	var body map[string]any
	if err := json.NewDecoder(get.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if get.Code != http.StatusOK || body["timezone"] != "Asia/Jakarta" || body["backup_schedule"] != "daily" {
		t.Fatalf("GET = %d %#v", get.Code, body)
	}
	if _, ok := body["backup"]; !ok {
		t.Fatalf("settings missing backup summary: %#v", body)
	}
	if got := request(http.MethodPut, `{"timezone":"bad","backup_schedule":"off"}`, true, "csrf").Code; got != http.StatusBadRequest {
		t.Fatalf("invalid PUT = %d", got)
	}
}

func TestSettingsRoutesMapDatabaseFailures(t *testing.T) {
	db, _ := testutil.TempDB(t)
	now := time.Unix(1000, 0).UTC()
	if _, err := db.Exec("INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,'hash','x','x')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO auth_sessions(token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,?)", auth.TokenHash("session"), "csrf", now.Add(time.Hour).Format(time.RFC3339Nano), "x"); err != nil {
		t.Fatal(err)
	}
	h := New(ServerDeps{DB: db, DataDir: t.TempDir(), Clock: &clock.FakeClock{T: now}})
	if _, err := db.Exec("DROP TABLE settings"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ method, body string }{{http.MethodGet, ""}, {http.MethodPut, `{"timezone":"UTC","backup_schedule":"off"}`}} {
		r := httptest.NewRequest(tc.method, "/api/v1/settings", bytes.NewBufferString(tc.body))
		r.AddCookie(&http.Cookie{Name: "pa_session", Value: "session"})
		r.AddCookie(&http.Cookie{Name: "pa_csrf", Value: "csrf"})
		r.Header.Set("X-CSRF-Token", "csrf")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("%s database failure = %d", tc.method, w.Code)
		}
	}
}
