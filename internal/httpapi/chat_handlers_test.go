package httpapi

import (
	"context"
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
	h, pid, cookies := sessionAPIServer(t, []config.ModelRef{{Provider: "openai", ModelID: "m"}})
	// Rebuild with provider while retaining the helper's database is intentionally avoided by
	// obtaining its concrete dependencies through a dedicated setup below.
	_ = pid
	_ = h
	h, pid, cookies = chatAPIServer(t, p)
	created := apiRequest(t, h, "POST", "/api/v1/projects/"+pid+"/sessions", map[string]string{"provider": "openai", "model_id": "m"}, cookies, "csrf")
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

func chatAPIServer(t *testing.T, provider agent.Provider) (http.Handler, string, []*http.Cookie) {
	t.Helper()
	// Shared setup variant allows exercising the injected provider.
	db, dir := testutil.TempDB(t)
	now := time.Unix(1000, 0).UTC()
	_, _ = db.Exec(`INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,'x',?,?)`, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	_, _ = db.Exec(`INSERT INTO auth_sessions(token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,?)`, auth.TokenHash("token"), "csrf", now.Add(time.Hour).Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	_, _ = db.Exec(`INSERT INTO vaults(id,name,created_at,updated_at) VALUES('v','v',?,?)`, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	_, _ = db.Exec(`INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES('p','v','p',?,?)`, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	return New(ServerDeps{DB: db, DataDir: dir, Clock: &clock.FakeClock{T: now}, Models: []config.ModelRef{{Provider: "openai", ModelID: "m"}}, Provider: provider}), "p", []*http.Cookie{{Name: "pa_session", Value: "token"}, {Name: "pa_csrf", Value: "csrf"}}
}
