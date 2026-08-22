package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/auth"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func instructionAPIServer(t *testing.T) (http.Handler, string, []*http.Cookie) {
	t.Helper()
	db, dir := testutil.TempDB(t)
	now := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
	stamp := now.Format(time.RFC3339Nano)
	if _, err := db.Exec(`INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,'x',?,?)`, stamp, stamp); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO auth_sessions(token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,?)`,
		auth.TokenHash("token"), "csrf", now.Add(time.Hour).Format(time.RFC3339Nano), stamp); err != nil {
		t.Fatal(err)
	}
	h := New(ServerDeps{DB: db, DataDir: dir, Clock: &clock.FakeClock{T: now}})
	cookies := []*http.Cookie{
		{Name: "pa_session", Value: "token"},
		{Name: "pa_csrf", Value: "csrf"},
	}

	// Create project via API so layout + knowledge seeds exist.
	created := apiRequest(t, h, http.MethodPost, "/api/v1/projects", map[string]any{
		"name":     "Instr",
		"vault_id": nil,
	}, cookies, "csrf")
	if created.Code != http.StatusCreated {
		t.Fatalf("create project = %d %s", created.Code, created.Body.String())
	}
	var proj ProjectDTO
	if err := json.NewDecoder(created.Body).Decode(&proj); err != nil {
		t.Fatal(err)
	}
	if proj.ID == "" {
		t.Fatal("empty project id")
	}
	return h, proj.ID, cookies
}

type instructionResponse struct {
	Content string               `json:"content"`
	Note    domain.KnowledgeNote `json:"note"`
}

func TestInstructionProjectPutGetRoundTrip(t *testing.T) {
	h, pid, cookies := instructionAPIServer(t)
	body := "# Project agents\nDo the thing.\n"
	putPath := "/api/v1/projects/" + pid + "/instructions/agents"
	put := apiRequest(t, h, http.MethodPut, putPath, map[string]string{"content": body}, cookies, "csrf")
	if put.Code != http.StatusOK && put.Code != http.StatusCreated {
		t.Fatalf("PUT project agents = %d %s", put.Code, put.Body.String())
	}
	var putResp instructionResponse
	if err := json.NewDecoder(put.Body).Decode(&putResp); err != nil {
		t.Fatal(err)
	}
	if putResp.Content != body {
		t.Fatalf("PUT content = %q want %q", putResp.Content, body)
	}
	if putResp.Note.ID == "" || putResp.Note.Kind != domain.KnowledgeKindAgents || putResp.Note.RelativePath != "AGENTS.md" {
		t.Fatalf("PUT note = %+v", putResp.Note)
	}
	if putResp.Note.ProjectID != pid || putResp.Note.IsGlobal {
		t.Fatalf("PUT note scope = %+v", putResp.Note)
	}

	get := apiRequest(t, h, http.MethodGet, putPath, nil, cookies, "")
	if get.Code != http.StatusOK {
		t.Fatalf("GET project agents = %d %s", get.Code, get.Body.String())
	}
	var getResp instructionResponse
	if err := json.NewDecoder(get.Body).Decode(&getResp); err != nil {
		t.Fatal(err)
	}
	if getResp.Content != body {
		t.Fatalf("GET content = %q want %q", getResp.Content, body)
	}
	if getResp.Note.ID != putResp.Note.ID || getResp.Note.Kind != domain.KnowledgeKindAgents {
		t.Fatalf("GET note = %+v want id=%s", getResp.Note, putResp.Note.ID)
	}
}

func TestInstructionGlobalPutGetRoundTrip(t *testing.T) {
	h, _, cookies := instructionAPIServer(t)
	body := "# Global system\nBe careful.\n"
	path := "/api/v1/global/instructions/system"
	put := apiRequest(t, h, http.MethodPut, path, map[string]string{"content": body}, cookies, "csrf")
	if put.Code != http.StatusOK && put.Code != http.StatusCreated {
		t.Fatalf("PUT global system = %d %s", put.Code, put.Body.String())
	}
	var putResp instructionResponse
	if err := json.NewDecoder(put.Body).Decode(&putResp); err != nil {
		t.Fatal(err)
	}
	if putResp.Content != body || !putResp.Note.IsGlobal || putResp.Note.Kind != domain.KnowledgeKindSystem {
		t.Fatalf("PUT resp = %+v", putResp)
	}
	if putResp.Note.RelativePath != "SYSTEM.md" || putResp.Note.ProjectID != "" {
		t.Fatalf("PUT note paths = %+v", putResp.Note)
	}

	get := apiRequest(t, h, http.MethodGet, path, nil, cookies, "")
	if get.Code != http.StatusOK {
		t.Fatalf("GET global system = %d %s", get.Code, get.Body.String())
	}
	var getResp instructionResponse
	if err := json.NewDecoder(get.Body).Decode(&getResp); err != nil {
		t.Fatal(err)
	}
	if getResp.Content != body || getResp.Note.ID != putResp.Note.ID {
		t.Fatalf("GET resp = %+v", getResp)
	}
}

func TestInstructionBadName(t *testing.T) {
	h, pid, cookies := instructionAPIServer(t)
	for _, path := range []string{
		"/api/v1/projects/" + pid + "/instructions/readme",
		"/api/v1/global/instructions/lessons",
		"/api/v1/projects/" + pid + "/instructions/memory",
	} {
		if got := apiRequest(t, h, http.MethodGet, path, nil, cookies, "").Code; got != http.StatusBadRequest {
			t.Errorf("GET bad name %s = %d want 400", path, got)
		}
		if got := apiRequest(t, h, http.MethodPut, path, map[string]string{"content": "x"}, cookies, "csrf").Code; got != http.StatusBadRequest {
			t.Errorf("PUT bad name %s = %d want 400", path, got)
		}
	}
}

func TestInstructionPutRequiresCSRF(t *testing.T) {
	h, pid, cookies := instructionAPIServer(t)
	body := map[string]string{"content": "# x\n"}
	for _, path := range []string{
		"/api/v1/projects/" + pid + "/instructions/soul",
		"/api/v1/global/instructions/agents",
	} {
		// Auth first: no cookies → 401
		if got := apiRequest(t, h, http.MethodPut, path, body, nil, "csrf").Code; got != http.StatusUnauthorized {
			t.Errorf("PUT %s no auth = %d want 401", path, got)
		}
		// Auth without CSRF → 403
		if got := apiRequest(t, h, http.MethodPut, path, body, cookies, "").Code; got != http.StatusForbidden {
			t.Errorf("PUT %s no CSRF = %d want 403", path, got)
		}
		if got := apiRequest(t, h, http.MethodPut, path, body, cookies, "wrong").Code; got != http.StatusForbidden {
			t.Errorf("PUT %s bad CSRF = %d want 403", path, got)
		}
	}
}

func TestInstructionGetRequiresAuth(t *testing.T) {
	h, pid, _ := instructionAPIServer(t)
	for _, path := range []string{
		"/api/v1/projects/" + pid + "/instructions/agents",
		"/api/v1/global/instructions/system",
	} {
		if got := apiRequest(t, h, http.MethodGet, path, nil, nil, "").Code; got != http.StatusUnauthorized {
			t.Errorf("GET %s anonymous = %d want 401", path, got)
		}
	}
}

func TestInstructionGetMissing(t *testing.T) {
	h, _, cookies := instructionAPIServer(t)

	// Missing project → 404
	if got := apiRequest(t, h, http.MethodGet, "/api/v1/projects/does-not-exist/instructions/agents", nil, cookies, "").Code; got != http.StatusNotFound {
		t.Fatalf("GET missing project = %d want 404", got)
	}

	// Global file never written and not seeded by project create → 404
	// (EnsureGlobalKnowledgeDirs is not called by project create alone.)
	if got := apiRequest(t, h, http.MethodGet, "/api/v1/global/instructions/soul", nil, cookies, "").Code; got != http.StatusNotFound {
		t.Fatalf("GET missing global soul = %d want 404", got)
	}
}
