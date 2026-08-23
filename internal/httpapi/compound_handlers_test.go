package httpapi

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/agent"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/layout"
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

type compoundHTTPFixture struct {
	h       http.Handler
	db      *sql.DB
	dataDir string
	pid     string
	cookies []*http.Cookie
	sess    domain.Session
}

type compoundProposalResp struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	SessionID  string `json:"session_id"`
	RequestKey string `json:"request_key"`
	Error      string `json:"error"`
	Items      []struct {
		Kind    string `json:"kind"`
		Path    string `json:"path"`
		Content string `json:"content"`
	} `json:"items"`
	DecidedAt  *time.Time `json:"decided_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

func newCompoundHTTPFixture(t *testing.T) compoundHTTPFixture {
	t.Helper()
	h, db, dir, pid, cookies := chatAPIServer(t, nil)
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
	return compoundHTTPFixture{h: h, db: db, dataDir: dir, pid: pid, cookies: cookies, sess: sess}
}

func (f compoundHTTPFixture) createSession(t *testing.T, title string) domain.Session {
	t.Helper()
	created := apiRequest(t, f.h, "POST", "/api/v1/projects/"+f.pid+"/sessions", map[string]string{
		"title": title, "provider": "openai", "model_id": "m",
	}, f.cookies, "csrf")
	if created.Code != http.StatusCreated {
		t.Fatalf("create session = %d %s", created.Code, created.Body.String())
	}
	var sess domain.Session
	if err := json.Unmarshal(created.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}
	return sess
}

func (f compoundHTTPFixture) postPending(t *testing.T, key, body string) compoundProposalResp {
	t.Helper()
	res := apiRequest(t, f.h, "POST", "/api/v1/sessions/"+f.sess.ID+"/compound", map[string]any{
		"request_key": key,
		"items": []map[string]string{{
			"kind":           "agents_patch",
			"path":           "AGENTS.md",
			"action":         "update",
			"content":        body,
			"content_sha256": shaHexTest(body),
		}},
	}, f.cookies, "csrf")
	if res.Code != http.StatusOK {
		t.Fatalf("POST compound = %d %s", res.Code, res.Body.String())
	}
	return decodeCompoundProposal(t, res.Body.Bytes())
}

func (f compoundHTTPFixture) agentsPath() string {
	return filepath.Join(layout.ProjectRoot(f.dataDir, "v", f.pid), "AGENTS.md")
}

func decodeCompoundProposal(t *testing.T, raw []byte) compoundProposalResp {
	t.Helper()
	var got compoundProposalResp
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	return got
}

func TestCompoundGETProposal(t *testing.T) {
	f := newCompoundHTTPFixture(t)
	body := "# Agents\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"
	pending := f.postPending(t, "rk-get", body)
	path := "/api/v1/sessions/" + f.sess.ID + "/compound/" + pending.ID

	if got := apiRequest(t, f.h, "GET", path, nil, nil, "").Code; got != http.StatusUnauthorized {
		t.Fatalf("anon GET = %d", got)
	}
	res := apiRequest(t, f.h, "GET", path, nil, f.cookies, "")
	if res.Code != http.StatusOK {
		t.Fatalf("GET proposal = %d %s", res.Code, res.Body.String())
	}
	got := decodeCompoundProposal(t, res.Body.Bytes())
	if got.ID != pending.ID || got.Status != "pending" || got.SessionID != f.sess.ID {
		t.Fatalf("GET proposal = %+v", got)
	}
	if len(got.Items) != 1 || got.Items[0].Path != "AGENTS.md" {
		t.Fatalf("items = %+v", got.Items)
	}
}

func TestCompoundDecideReject(t *testing.T) {
	f := newCompoundHTTPFixture(t)
	body := "# Agents\n\nrule: reject-must-not-write\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"
	pending := f.postPending(t, "rk-reject", body)
	path := "/api/v1/sessions/" + f.sess.ID + "/compound/" + pending.ID + "/decide"
	res := apiRequest(t, f.h, "POST", path, map[string]any{
		"request_key": "rk-reject",
		"decision":    "reject",
	}, f.cookies, "csrf")
	if res.Code != http.StatusOK {
		t.Fatalf("decide reject = %d %s", res.Code, res.Body.String())
	}
	got := decodeCompoundProposal(t, res.Body.Bytes())
	if got.Status != "rejected" || got.DecidedAt == nil || got.FinishedAt == nil {
		t.Fatalf("reject proposal = %+v", got)
	}
	if raw, err := os.ReadFile(f.agentsPath()); err == nil && strings.Contains(string(raw), "reject-must-not-write") {
		t.Fatalf("reject wrote AGENTS.md: %s", raw)
	}
}

func TestCompoundDecideApproveWritesAgents(t *testing.T) {
	f := newCompoundHTTPFixture(t)
	body := "# Agents\n\nrule: task-26-approve\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"
	pending := f.postPending(t, "rk-approve", body)
	path := "/api/v1/sessions/" + f.sess.ID + "/compound/" + pending.ID + "/decide"
	res := apiRequest(t, f.h, "POST", path, map[string]any{
		"request_key": "rk-approve",
		"decision":    "approve",
	}, f.cookies, "csrf")
	if res.Code != http.StatusOK {
		t.Fatalf("decide approve = %d %s", res.Code, res.Body.String())
	}
	got := decodeCompoundProposal(t, res.Body.Bytes())
	if got.Status != "approved" || got.DecidedAt == nil || got.FinishedAt == nil {
		t.Fatalf("approve proposal = %+v", got)
	}
	raw, err := os.ReadFile(f.agentsPath())
	if err != nil {
		t.Fatalf("AGENTS.md missing after approve: %v", err)
	}
	if string(raw) != body {
		t.Fatalf("AGENTS.md = %q, want %q", raw, body)
	}

	again := apiRequest(t, f.h, "POST", path, map[string]any{
		"request_key": "rk-approve",
		"decision":    "approve",
	}, f.cookies, "csrf")
	if again.Code != http.StatusOK {
		t.Fatalf("idempotent approve = %d %s", again.Code, again.Body.String())
	}
	repeat := decodeCompoundProposal(t, again.Body.Bytes())
	if repeat.Status != "approved" || repeat.FinishedAt == nil {
		t.Fatalf("idempotent approve = %+v", repeat)
	}
}

func TestCompoundWrongSessionProposal404(t *testing.T) {
	f := newCompoundHTTPFixture(t)
	body := "# Agents\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"
	pending := f.postPending(t, "rk-wrong-sess", body)
	other := f.createSession(t, "other")
	own := apiRequest(t, f.h, "GET", "/api/v1/sessions/"+f.sess.ID+"/compound/"+pending.ID, nil, f.cookies, "")
	if own.Code != http.StatusOK {
		t.Fatalf("GET own session = %d %s want 200 (route must exist)", own.Code, own.Body.String())
	}

	get := apiRequest(t, f.h, "GET", "/api/v1/sessions/"+other.ID+"/compound/"+pending.ID, nil, f.cookies, "")
	if get.Code != http.StatusNotFound {
		t.Fatalf("GET wrong session = %d %s want 404", get.Code, get.Body.String())
	}
	if !strings.Contains(get.Body.String(), "proposal_not_found") || strings.Contains(get.Body.String(), pending.ID) {
		t.Fatalf("GET wrong session body = %s", get.Body.String())
	}

	decide := apiRequest(t, f.h, "POST", "/api/v1/sessions/"+other.ID+"/compound/"+pending.ID+"/decide", map[string]any{
		"request_key": "rk-wrong-sess",
		"decision":    "approve",
	}, f.cookies, "csrf")
	if decide.Code != http.StatusNotFound {
		t.Fatalf("decide wrong session = %d %s want 404", decide.Code, decide.Body.String())
	}
	if !strings.Contains(decide.Body.String(), "proposal_not_found") || strings.Contains(decide.Body.String(), pending.ID) {
		t.Fatalf("decide wrong session body = %s", decide.Body.String())
	}
}

func TestCompoundGETRecoversApprovedUnfinished(t *testing.T) {
	f := newCompoundHTTPFixture(t)
	body := "# Agents\n\nrule: task-26-recovery\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"
	pending := f.postPending(t, "rk-recover", body)
	decided := time.Unix(2000, 0).UTC().Format(time.RFC3339Nano)
	if _, err := f.db.Exec(`UPDATE compound_proposals SET status='approved', decided_at=?, finished_at=NULL WHERE id=?`, decided, pending.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(f.agentsPath()); err == nil {
		t.Fatal("AGENTS.md already present before recovery GET")
	}

	res := apiRequest(t, f.h, "GET", "/api/v1/sessions/"+f.sess.ID+"/compound/"+pending.ID, nil, f.cookies, "")
	if res.Code != http.StatusOK {
		t.Fatalf("recovery GET = %d %s", res.Code, res.Body.String())
	}
	got := decodeCompoundProposal(t, res.Body.Bytes())
	if got.Status != "approved" || got.FinishedAt == nil {
		t.Fatalf("recovery proposal = %+v", got)
	}
	raw, err := os.ReadFile(f.agentsPath())
	if err != nil {
		t.Fatalf("AGENTS.md missing after recovery: %v", err)
	}
	if string(raw) != body {
		t.Fatalf("AGENTS.md after recovery = %q, want %q", raw, body)
	}
}

func TestCompoundDecideRequiresCSRFAndAuth(t *testing.T) {
	f := newCompoundHTTPFixture(t)
	body := "# Agents\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"
	pending := f.postPending(t, "rk-csrf", body)
	path := "/api/v1/sessions/" + f.sess.ID + "/compound/" + pending.ID + "/decide"
	req := map[string]any{"request_key": "rk-csrf", "decision": "reject"}
	if got := apiRequest(t, f.h, "POST", path, req, nil, "csrf").Code; got != http.StatusUnauthorized {
		t.Fatalf("anon decide = %d", got)
	}
	if got := apiRequest(t, f.h, "POST", path, req, f.cookies, "").Code; got != http.StatusForbidden {
		t.Fatalf("no csrf decide = %d", got)
	}
}
