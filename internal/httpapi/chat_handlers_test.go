package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/agent"
	"github.com/rigasyahrul/personal-agent/internal/auth"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/config"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

type failingProvider struct{ calls atomic.Int32 }

func (p *failingProvider) Chat(context.Context, agent.ChatRequest) (agent.ChatResponse, error) {
	p.calls.Add(1)
	return agent.ChatResponse{}, errors.New("secret outage detail")
}

func TestChatAPIProviderFailureIsSafeIdempotentAndHistoryReadable(t *testing.T) {
	p := &failingProvider{}
	h, _, pid, cookies := chatAPIServer(t, p)
	created := apiRequest(t, h, "POST", "/api/v1/projects/"+pid+"/sessions", map[string]string{"title": "x", "provider": "openai", "model_id": "m"}, cookies, "csrf")
	if created.Code != http.StatusCreated {
		t.Fatalf("create=%d %s", created.Code, created.Body.String())
	}
	var s domain.Session
	if err := json.Unmarshal(created.Body.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/sessions/" + s.ID + "/messages"
	body := map[string]string{"content": "hello", "request_key": "same"}
	first := apiRequest(t, h, "POST", path, body, cookies, "csrf")
	second := apiRequest(t, h, "POST", path, body, cookies, "csrf")
	if first.Code != 502 || second.Code != 202 || p.calls.Load() != 1 || strings.Contains(first.Body.String(), "secret") {
		t.Fatalf("responses=%d/%d calls=%d body=%s", first.Code, second.Code, p.calls.Load(), first.Body.String())
	}
	if history := apiRequest(t, h, "GET", path, nil, cookies, ""); history.Code != 200 || !strings.Contains(history.Body.String(), "hello") {
		t.Fatalf("history=%d %s", history.Code, history.Body.String())
	}
}

func TestChatAPIValidationAuthAndRunMappings(t *testing.T) {
	h, db, pid, cookies := chatAPIServer(t, &failingProvider{})
	create := func(title string) domain.Session {
		t.Helper()
		res := apiRequest(t, h, "POST", "/api/v1/projects/"+pid+"/sessions", map[string]string{"title": title, "provider": "openai", "model_id": "m"}, cookies, "csrf")
		if res.Code != http.StatusCreated {
			t.Fatalf("create %s=%d %s", title, res.Code, res.Body.String())
		}
		var s domain.Session
		if err := json.Unmarshal(res.Body.Bytes(), &s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	s := create("active")
	path := "/api/v1/sessions/" + s.ID + "/messages"
	if got := apiRequest(t, h, "POST", path, map[string]string{"content": "x", "request_key": "k"}, nil, "bad").Code; got != http.StatusUnauthorized {
		t.Fatalf("anonymous post=%d", got)
	}
	for _, body := range []string{`{"content":" ","request_key":"k"}`, `{"content":"x","request_key":" \t"}`, `{"content":"x","request_key":"k"}{"content":"y","request_key":"z"}`} {
		if got := rawAPIRequest(t, h, "POST", path, body, cookies, "csrf").Code; got != http.StatusBadRequest {
			t.Fatalf("invalid message %q=%d", body, got)
		}
	}
	if _, err := db.Exec(`INSERT INTO agent_runs(id,session_id,request_key,status,created_at) VALUES('busy-run',?,?,'queued','1970-01-01T00:00:00Z')`, s.ID, "other-key"); err != nil {
		t.Fatal(err)
	}
	if got := apiRequest(t, h, "POST", path, map[string]string{"content": "x", "request_key": "fresh"}, cookies, "csrf").Code; got != http.StatusConflict {
		t.Fatalf("busy=%d", got)
	}

	terminal := create("terminal")
	if got := apiRequest(t, h, "DELETE", "/api/v1/sessions/"+terminal.ID, nil, cookies, "csrf").Code; got != http.StatusNoContent {
		t.Fatalf("delete terminal fixture=%d", got)
	}
	if got := apiRequest(t, h, "POST", "/api/v1/sessions/"+terminal.ID+"/messages", map[string]string{"content": "x", "request_key": "fresh-terminal"}, cookies, "csrf").Code; got != http.StatusConflict {
		t.Fatalf("terminal=%d", got)
	}
	for _, suffix := range []string{"/messages", "/runs/current"} {
		if got := apiRequest(t, h, "GET", "/api/v1/sessions/missing"+suffix, nil, cookies, "").Code; got != http.StatusNotFound {
			t.Fatalf("missing %s=%d", suffix, got)
		}
	}
	if got := apiRequest(t, h, "POST", "/api/v1/sessions/missing/messages", map[string]string{"content": "x", "request_key": "missing"}, cookies, "csrf").Code; got != http.StatusNotFound {
		t.Fatalf("post missing session=%d", got)
	}
	none := create("none")
	current := apiRequest(t, h, "GET", "/api/v1/sessions/"+none.ID+"/runs/current", nil, cookies, "")
	if current.Code != http.StatusNoContent || current.Body.Len() != 0 {
		t.Fatalf("no current=%d body=%q", current.Code, current.Body.String())
	}
}

func chatAPIServer(t *testing.T, provider agent.Provider) (http.Handler, *sql.DB, string, []*http.Cookie) {
	t.Helper()
	// Shared setup variant allows exercising the injected provider.
	db, dir := testutil.TempDB(t)
	now := time.Unix(1000, 0).UTC()
	for _, seed := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,'x',?,?)`, []any{now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)}},
		{`INSERT INTO auth_sessions(token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,?)`, []any{auth.TokenHash("token"), "csrf", now.Add(time.Hour).Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)}},
		{`INSERT INTO vaults(id,name,created_at,updated_at) VALUES('v','v',?,?)`, []any{now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)}},
		{`INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES('p','v','p',?,?)`, []any{now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)}},
	} {
		if _, err := db.Exec(seed.query, seed.args...); err != nil {
			t.Fatal(err)
		}
	}
	return New(ServerDeps{DB: db, DataDir: dir, Clock: &clock.FakeClock{T: now}, Models: []config.ModelRef{{Provider: "openai", ModelID: "m"}}, Provider: provider}), db, "p", []*http.Cookie{{Name: "pa_session", Value: "token"}, {Name: "pa_csrf", Value: "csrf"}}
}
