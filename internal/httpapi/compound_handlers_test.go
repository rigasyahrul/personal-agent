package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/rigasyahrul/personal-agent/internal/agent"
	"github.com/rigasyahrul/personal-agent/internal/domain"
)

func shaHexTest(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func TestCompoundPOSTItemsCreatesPending(t *testing.T) {
	h, _, _, pid, cookies := chatAPIServer(t, nil)
	created := apiRequest(t, h, "POST", "/api/v1/projects/"+pid+"/sessions", map[string]string{
		"title": "c", "provider": "openai", "model_id": "m",
	}, cookies, "csrf")
	if created.Code != http.StatusCreated {
		t.Fatalf("create session = %d %s", created.Code, created.Body.String())
	}
	var sess domain.Session
	if err := json.Unmarshal(created.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}

	body := "# Agents\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"
	req := map[string]any{
		"request_key": "rk-items",
		"items": []map[string]string{{
			"kind":           "agents_patch",
			"path":           "AGENTS.md",
			"action":         "update",
			"content":        body,
			"content_sha256": shaHexTest(body),
		}},
	}
	res := apiRequest(t, h, "POST", "/api/v1/sessions/"+sess.ID+"/compound", req, cookies, "csrf")
	if res.Code != http.StatusOK {
		t.Fatalf("POST compound = %d %s", res.Code, res.Body.String())
	}
	var got struct {
		ID         string `json:"id"`
		Status     string `json:"status"`
		SessionID  string `json:"session_id"`
		Scope      string `json:"scope"`
		ProjectID  string `json:"project_id"`
		RequestKey string `json:"request_key"`
		Items      []struct {
			Kind string `json:"kind"`
			Path string `json:"path"`
		} `json:"items"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID == "" || got.Status != "pending" || got.SessionID != sess.ID {
		t.Fatalf("proposal = %+v", got)
	}
	if got.Scope != "project" || got.ProjectID != pid || got.RequestKey != "rk-items" {
		t.Fatalf("scope fields = %+v", got)
	}
	if len(got.Items) != 1 || got.Items[0].Kind != "agents_patch" {
		t.Fatalf("items = %+v", got.Items)
	}
}

func TestCompoundPOSTRequiresCSRFAndAuth(t *testing.T) {
	h, _, _, pid, cookies := chatAPIServer(t, nil)
	created := apiRequest(t, h, "POST", "/api/v1/projects/"+pid+"/sessions", map[string]string{
		"title": "c", "provider": "openai", "model_id": "m",
	}, cookies, "csrf")
	var sess domain.Session
	if err := json.Unmarshal(created.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/sessions/" + sess.ID + "/compound"
	body := map[string]any{"request_key": "rk", "items": []map[string]string{}}
	if got := apiRequest(t, h, "POST", path, body, nil, "csrf").Code; got != http.StatusUnauthorized {
		t.Fatalf("anon = %d", got)
	}
	if got := apiRequest(t, h, "POST", path, body, cookies, "").Code; got != http.StatusForbidden {
		t.Fatalf("no csrf = %d", got)
	}
}

type compoundJSONProvider struct {
	content string
	req     agent.ChatRequest
	calls   int
}

func (p *compoundJSONProvider) Chat(_ context.Context, req agent.ChatRequest) (agent.ChatResponse, error) {
	p.calls++
	p.req = req
	return agent.ChatResponse{Content: p.content}, nil
}

func compoundAgentsJSON(wrongHash string) string {
	body := "# Agents\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"
	raw, _ := json.Marshal([]map[string]string{{
		"kind":           "agents_patch",
		"path":           "AGENTS.md",
		"action":         "update",
		"content":        body,
		"content_sha256": wrongHash,
	}})
	return string(raw)
}

func TestCompoundPOSTOmittedItemsGeneratesProposal(t *testing.T) {
	provider := &compoundJSONProvider{content: "```json\n" + compoundAgentsJSON("00") + "\n```"}
	h, _, _, pid, cookies := chatAPIServer(t, provider)
	created := apiRequest(t, h, "POST", "/api/v1/projects/"+pid+"/sessions", map[string]string{
		"title": "c", "provider": "openai", "model_id": "m",
	}, cookies, "csrf")
	var sess domain.Session
	if err := json.Unmarshal(created.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}
	res := apiRequest(t, h, "POST", "/api/v1/sessions/"+sess.ID+"/compound", map[string]any{
		"request_key":  "rk-gen",
		"user_context": "summarize",
	}, cookies, "csrf")
	if res.Code != http.StatusOK {
		t.Fatalf("omitted items = %d %s want 200", res.Code, res.Body.String())
	}
	var got struct {
		ID        string `json:"id"`
		Status    string `json:"status"`
		SessionID string `json:"session_id"`
		Items     []struct {
			Kind          string `json:"kind"`
			Path          string `json:"path"`
			Content       string `json:"content"`
			ContentSHA256 string `json:"content_sha256"`
		} `json:"items"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID == "" || got.Status != "pending" || got.SessionID != sess.ID {
		t.Fatalf("proposal = %+v", got)
	}
	if len(got.Items) != 1 || got.Items[0].Kind != "agents_patch" || got.Items[0].Path != "AGENTS.md" {
		t.Fatalf("items = %+v", got.Items)
	}
	wantHash := shaHexTest("# Agents\n\n## Memory\n- [[memory/lessons|lessons.md]]\n")
	if got.Items[0].ContentSHA256 != wantHash {
		t.Fatalf("hash = %q, want server sha256 %q", got.Items[0].ContentSHA256, wantHash)
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}
	if len(provider.req.Tools) != 0 {
		t.Fatalf("http generate registered tools: %#v", provider.req.Tools)
	}
}

func TestCompoundPOSTGenerateBusySessionConflict(t *testing.T) {
	h, db, _, pid, cookies := chatAPIServer(t, &compoundJSONProvider{content: compoundAgentsJSON("00")})
	created := apiRequest(t, h, "POST", "/api/v1/projects/"+pid+"/sessions", map[string]string{
		"title": "c", "provider": "openai", "model_id": "m",
	}, cookies, "csrf")
	var sess domain.Session
	if err := json.Unmarshal(created.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO agent_runs(id,session_id,request_key,status,created_at) VALUES('busy-run',?,?,'queued','1970-01-01T00:00:00Z')`, sess.ID, "chat-key"); err != nil {
		t.Fatal(err)
	}
	res := apiRequest(t, h, "POST", "/api/v1/sessions/"+sess.ID+"/compound", map[string]any{
		"request_key":  "rk-busy",
		"user_context": "summarize",
	}, cookies, "csrf")
	if res.Code != http.StatusConflict {
		t.Fatalf("busy generate = %d %s want 409", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "session_busy") {
		t.Fatalf("body = %s, want session_busy", res.Body.String())
	}
}

func TestCompoundPOSTItemsDoesNotStartAgentRun(t *testing.T) {
	provider := &compoundJSONProvider{content: compoundAgentsJSON("00")}
	h, _, _, pid, cookies := chatAPIServer(t, provider)
	created := apiRequest(t, h, "POST", "/api/v1/projects/"+pid+"/sessions", map[string]string{
		"title": "c", "provider": "openai", "model_id": "m",
	}, cookies, "csrf")
	var sess domain.Session
	if err := json.Unmarshal(created.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}
	body := "# Agents\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"
	res := apiRequest(t, h, "POST", "/api/v1/sessions/"+sess.ID+"/compound", map[string]any{
		"request_key": "rk-items-no-run",
		"items": []map[string]string{{
			"kind":           "agents_patch",
			"path":           "AGENTS.md",
			"action":         "update",
			"content":        body,
			"content_sha256": shaHexTest(body),
		}},
	}, cookies, "csrf")
	if res.Code != http.StatusOK {
		t.Fatalf("items POST = %d %s", res.Code, res.Body.String())
	}
	if provider.calls != 0 {
		t.Fatalf("items path started agent run; provider calls = %d", provider.calls)
	}
}

func TestCompoundPOSTIgnoresClientScope(t *testing.T) {
	h, _, _, pid, cookies := chatAPIServer(t, nil)
	created := apiRequest(t, h, "POST", "/api/v1/projects/"+pid+"/sessions", map[string]string{
		"title": "c", "provider": "openai", "model_id": "m",
	}, cookies, "csrf")
	var sess domain.Session
	if err := json.Unmarshal(created.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}
	body := "# Agents\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"
	res := apiRequest(t, h, "POST", "/api/v1/sessions/"+sess.ID+"/compound", map[string]any{
		"request_key": "rk-scope",
		"scope":       "global",
		"project_id":  "evil",
		"items": []map[string]string{{
			"kind":           "agents_patch",
			"path":           "AGENTS.md",
			"action":         "update",
			"content":        body,
			"content_sha256": shaHexTest(body),
		}},
	}, cookies, "csrf")
	// Unknown fields rejected (cannot bind client scope).
	if res.Code != http.StatusBadRequest {
		t.Fatalf("client scope = %d %s want 400", res.Code, res.Body.String())
	}
}

func TestCompoundPOSTMissingSession(t *testing.T) {
	h, _, _, _, cookies := chatAPIServer(t, nil)
	body := "# Agents\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"
	res := apiRequest(t, h, "POST", "/api/v1/sessions/missing/compound", map[string]any{
		"request_key": "rk-miss",
		"items": []map[string]string{{
			"kind":           "agents_patch",
			"path":           "AGENTS.md",
			"action":         "update",
			"content":        body,
			"content_sha256": shaHexTest(body),
		}},
	}, cookies, "csrf")
	if res.Code != http.StatusNotFound {
		t.Fatalf("missing = %d %s", res.Code, res.Body.String())
	}
}
