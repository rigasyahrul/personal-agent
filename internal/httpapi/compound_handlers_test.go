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
	Scope      string `json:"scope"`
	ProjectID  string `json:"project_id"`
	VaultID    string `json:"vault_id"`
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

func TestCompoundDecideReDrivesPublishWhenApprovedUnfinished(t *testing.T) {
	f := newCompoundHTTPFixture(t)
	body := "# Agents\n\nrule: decide-retry-recovery\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"
	pending := f.postPending(t, "rk-decide-recover", body)
	decided := time.Unix(2000, 0).UTC().Format(time.RFC3339Nano)
	if _, err := f.db.Exec(`UPDATE compound_proposals SET status='approved', decided_at=?, finished_at=NULL WHERE id=?`, decided, pending.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(f.agentsPath()); err == nil {
		t.Fatal("AGENTS.md already present before decide retry")
	}

	res := apiRequest(t, f.h, "POST", "/api/v1/sessions/"+f.sess.ID+"/compound/"+pending.ID+"/decide", map[string]any{
		"request_key": "rk-decide-recover",
		"decision":    "approve",
	}, f.cookies, "csrf")
	if res.Code != http.StatusOK {
		t.Fatalf("decide retry = %d %s", res.Code, res.Body.String())
	}
	got := decodeCompoundProposal(t, res.Body.Bytes())
	if got.Status != "approved" || got.FinishedAt == nil {
		t.Fatalf("decide retry proposal = %+v", got)
	}
	raw, err := os.ReadFile(f.agentsPath())
	if err != nil {
		t.Fatalf("AGENTS.md missing after decide retry: %v", err)
	}
	if string(raw) != body {
		t.Fatalf("AGENTS.md after decide retry = %q, want %q", raw, body)
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

func insertScopedSession(t *testing.T, db *sql.DB, id, home, vaultID string) {
	t.Helper()
	ts := time.Unix(1000, 0).UTC().Format(time.RFC3339Nano)
	var err error
	switch home {
	case "vault":
		_, err = db.Exec(`INSERT INTO sessions(id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at)
			VALUES(?,?,?,NULL,'active','openai','m','{}','{}',?,?,?)`, id, home, vaultID, id, ts, ts)
	case "global":
		_, err = db.Exec(`INSERT INTO sessions(id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at)
			VALUES(?,?,NULL,NULL,'active','openai','m','{}','{}',?,?,?)`, id, home, id, ts, ts)
	default:
		t.Fatalf("unsupported home %q", home)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func compoundMemoryItems(detailPath, detail, lessons string) []map[string]string {
	return []map[string]string{
		{"kind": "memory_detail", "path": detailPath, "action": "create", "content": detail, "content_sha256": shaHexTest(detail)},
		{"kind": "lessons_index_row", "path": "memory/lessons.md", "action": "update", "content": lessons, "content_sha256": shaHexTest(lessons)},
	}
}

func mustNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Fatalf("path must not exist: %s", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}

func TestCompoundPOSTVaultRejectsAgentsPatch(t *testing.T) {
	h, db, _, _, cookies := chatAPIServer(t, nil)
	insertScopedSession(t, db, "sess-vault", "vault", "v")
	body := "# Agents\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"
	res := apiRequest(t, h, "POST", "/api/v1/sessions/sess-vault/compound", map[string]any{
		"request_key": "rk-vault-agents",
		"items": []map[string]string{{
			"kind":           "agents_patch",
			"path":           "AGENTS.md",
			"action":         "update",
			"content":        body,
			"content_sha256": shaHexTest(body),
		}},
	}, cookies, "csrf")
	if res.Code != http.StatusBadRequest {
		t.Fatalf("vault agents_patch = %d %s want 400", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "invalid_items") {
		t.Fatalf("body = %s, want invalid_items", res.Body.String())
	}
}

func TestCompoundPOSTVaultAcceptsMemoryDetail(t *testing.T) {
	h, db, dataDir, pid, cookies := chatAPIServer(t, nil)
	insertScopedSession(t, db, "sess-vault-mem", "vault", "v")
	detailPath := "memory/20260822-1200-vault-http.md"
	detail := "# Vault http lesson\n"
	lessons := "- [[memory/20260822-1200-vault-http]]\n"
	res := apiRequest(t, h, "POST", "/api/v1/sessions/sess-vault-mem/compound", map[string]any{
		"request_key": "rk-vault-mem",
		"items":       compoundMemoryItems(detailPath, detail, lessons),
	}, cookies, "csrf")
	if res.Code != http.StatusOK {
		t.Fatalf("vault memory POST = %d %s want 200", res.Code, res.Body.String())
	}
	got := decodeCompoundProposal(t, res.Body.Bytes())
	if got.Status != "pending" || got.Scope != "vault" || got.VaultID != "v" || got.ProjectID != "" {
		t.Fatalf("vault proposal = %+v", got)
	}

	decide := apiRequest(t, h, "POST", "/api/v1/sessions/sess-vault-mem/compound/"+got.ID+"/decide", map[string]any{
		"request_key": "rk-vault-mem",
		"decision":    "approve",
	}, cookies, "csrf")
	if decide.Code != http.StatusOK {
		t.Fatalf("vault decide = %d %s", decide.Code, decide.Body.String())
	}
	vaultFile := filepath.Join(layout.VaultRoot(dataDir, "v"), filepath.FromSlash(detailPath))
	raw, err := os.ReadFile(vaultFile)
	if err != nil {
		t.Fatalf("vault memory missing: %v", err)
	}
	if string(raw) != detail {
		t.Fatalf("vault memory = %q", raw)
	}
	mustNotExist(t, filepath.Join(layout.ProjectRoot(dataDir, "v", pid), filepath.FromSlash(detailPath)))
	mustNotExist(t, filepath.Join(layout.GlobalRoot(dataDir), filepath.FromSlash(detailPath)))
}

func TestCompoundPOSTGlobalWritesGlobalOnly(t *testing.T) {
	h, db, dataDir, pid, cookies := chatAPIServer(t, nil)
	insertScopedSession(t, db, "sess-global", "global", "")
	body := "# Global agents\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"
	res := apiRequest(t, h, "POST", "/api/v1/sessions/sess-global/compound", map[string]any{
		"request_key": "rk-global-agents",
		"items": []map[string]string{{
			"kind":           "agents_patch",
			"path":           "AGENTS.md",
			"action":         "update",
			"content":        body,
			"content_sha256": shaHexTest(body),
		}},
	}, cookies, "csrf")
	if res.Code != http.StatusOK {
		t.Fatalf("global agents POST = %d %s want 200", res.Code, res.Body.String())
	}
	got := decodeCompoundProposal(t, res.Body.Bytes())
	if got.Status != "pending" || got.Scope != "global" || got.ProjectID != "" || got.VaultID != "" {
		t.Fatalf("global proposal = %+v", got)
	}

	decide := apiRequest(t, h, "POST", "/api/v1/sessions/sess-global/compound/"+got.ID+"/decide", map[string]any{
		"request_key": "rk-global-agents",
		"decision":    "approve",
	}, cookies, "csrf")
	if decide.Code != http.StatusOK {
		t.Fatalf("global decide = %d %s", decide.Code, decide.Body.String())
	}
	finished := decodeCompoundProposal(t, decide.Body.Bytes())
	if finished.Status != "approved" {
		t.Fatalf("global approve status = %+v", finished)
	}
	raw, err := os.ReadFile(filepath.Join(layout.GlobalRoot(dataDir), "AGENTS.md"))
	if err != nil {
		t.Fatalf("global AGENTS.md missing: %v", err)
	}
	if string(raw) != body {
		t.Fatalf("global AGENTS.md = %q", raw)
	}
	mustNotExist(t, filepath.Join(layout.ProjectRoot(dataDir, "v", pid), "AGENTS.md"))
	mustNotExist(t, filepath.Join(layout.VaultRoot(dataDir, "v"), "AGENTS.md"))
}

func TestCompoundPOSTProjectMemoryStaysInProjectRoot(t *testing.T) {
	f := newCompoundHTTPFixture(t)
	detailPath := "memory/20260822-1200-project-http.md"
	detail := "# Project http lesson\n"
	lessons := "- [[memory/20260822-1200-project-http]]\n"
	res := apiRequest(t, f.h, "POST", "/api/v1/sessions/"+f.sess.ID+"/compound", map[string]any{
		"request_key": "rk-proj-mem",
		"items":       compoundMemoryItems(detailPath, detail, lessons),
	}, f.cookies, "csrf")
	if res.Code != http.StatusOK {
		t.Fatalf("project memory POST = %d %s", res.Code, res.Body.String())
	}
	got := decodeCompoundProposal(t, res.Body.Bytes())
	if got.Scope != "project" || got.ProjectID != f.pid || got.VaultID != "v" {
		t.Fatalf("project proposal = %+v", got)
	}

	decide := apiRequest(t, f.h, "POST", "/api/v1/sessions/"+f.sess.ID+"/compound/"+got.ID+"/decide", map[string]any{
		"request_key": "rk-proj-mem",
		"decision":    "approve",
	}, f.cookies, "csrf")
	if decide.Code != http.StatusOK {
		t.Fatalf("project decide = %d %s", decide.Code, decide.Body.String())
	}
	projectFile := filepath.Join(layout.ProjectRoot(f.dataDir, "v", f.pid), filepath.FromSlash(detailPath))
	raw, err := os.ReadFile(projectFile)
	if err != nil {
		t.Fatalf("project memory missing: %v", err)
	}
	if string(raw) != detail {
		t.Fatalf("project memory = %q", raw)
	}
	mustNotExist(t, filepath.Join(layout.VaultRoot(f.dataDir, "v"), filepath.FromSlash(detailPath)))
	mustNotExist(t, filepath.Join(layout.GlobalRoot(f.dataDir), filepath.FromSlash(detailPath)))
}

func TestCompoundApproveEndToEnd(t *testing.T) {
	h, _, dataDir, _, cookies := chatAPIServer(t, nil)

	createdProject := apiRequest(t, h, "POST", "/api/v1/projects", map[string]any{
		"name": "e2e-compound", "vault_id": "v",
	}, cookies, "csrf")
	if createdProject.Code != http.StatusCreated {
		t.Fatalf("create project = %d %s", createdProject.Code, createdProject.Body.String())
	}
	var project struct {
		ID      string `json:"id"`
		VaultID string `json:"vault_id"`
	}
	if err := json.Unmarshal(createdProject.Body.Bytes(), &project); err != nil {
		t.Fatal(err)
	}
	if project.ID == "" || project.VaultID != "v" {
		t.Fatalf("project = %+v", project)
	}

	created := apiRequest(t, h, "POST", "/api/v1/projects/"+project.ID+"/sessions", map[string]string{
		"title": "e2e", "provider": "openai", "model_id": "m",
	}, cookies, "csrf")
	if created.Code != http.StatusCreated {
		t.Fatalf("create session = %d %s", created.Code, created.Body.String())
	}
	var sess domain.Session
	if err := json.Unmarshal(created.Body.Bytes(), &sess); err != nil {
		t.Fatal(err)
	}

	const standingRule = "rule: e2e-task-34-standing"
	body := "# Agents\n\n" + standingRule + "\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"
	posted := apiRequest(t, h, "POST", "/api/v1/sessions/"+sess.ID+"/compound", map[string]any{
		"request_key": "rk-e2e-approve",
		"items": []map[string]string{{
			"kind":           "agents_patch",
			"path":           "AGENTS.md",
			"action":         "update",
			"content":        body,
			"content_sha256": shaHexTest(body),
		}},
	}, cookies, "csrf")
	if posted.Code != http.StatusOK {
		t.Fatalf("POST items = %d %s want 200", posted.Code, posted.Body.String())
	}
	pending := decodeCompoundProposal(t, posted.Body.Bytes())
	if pending.ID == "" || pending.Status != "pending" || pending.Scope != "project" || pending.ProjectID != project.ID {
		t.Fatalf("pending proposal = %+v", pending)
	}

	decided := apiRequest(t, h, "POST", "/api/v1/sessions/"+sess.ID+"/compound/"+pending.ID+"/decide", map[string]any{
		"request_key": "rk-e2e-approve",
		"decision":    "approve",
	}, cookies, "csrf")
	if decided.Code != http.StatusOK {
		t.Fatalf("decide approve = %d %s want 200", decided.Code, decided.Body.String())
	}
	got := decodeCompoundProposal(t, decided.Body.Bytes())
	if got.Status != "approved" || got.FinishedAt == nil {
		t.Fatalf("approved proposal = %+v", got)
	}

	raw, err := os.ReadFile(filepath.Join(layout.ProjectRoot(dataDir, project.VaultID, project.ID), "AGENTS.md"))
	if err != nil {
		t.Fatalf("project AGENTS.md missing after approve: %v", err)
	}
	agents := string(raw)
	for _, want := range []string{standingRule, "## Memory", "[[memory/lessons"} {
		if !strings.Contains(agents, want) {
			t.Fatalf("project AGENTS.md missing %q:\n%s", want, agents)
		}
	}
}
