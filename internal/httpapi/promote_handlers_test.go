package httpapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/auth"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

type promoteHTTPTest struct {
	h  http.Handler
	db *sql.DB
}

func newPromoteHTTPTest(t *testing.T) promoteHTTPTest {
	t.Helper()
	db, dataDir := testutil.TempDB(t)
	now := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	stamp := now.Format(time.RFC3339Nano)
	statements := []string{
		`INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,'x','x','x')`,
		`INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','P','` + stamp + `','` + stamp + `')`,
		`INSERT INTO sessions(id,home,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at) VALUES('s1','project','p1','active','test','model','{}','{}','S','` + stamp + `','` + stamp + `')`,
	}
	for _, q := range statements {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO auth_sessions(token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,'x')`, auth.TokenHash(testSession), testCSRF, now.Add(time.Hour).Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	workspace := layout.SessionWorkspace(dataDir, "project", "", "p1", "s1")
	if err := os.MkdirAll(layout.SourceDir(layout.ProjectRoot(dataDir, "", "p1")), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(workspace, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "draft.md"), []byte("original source"), 0600); err != nil {
		t.Fatal(err)
	}
	return promoteHTTPTest{New(ServerDeps{DB: db, DataDir: dataDir, Clock: &clock.FakeClock{T: now}}), db}
}

func submitPromote(t *testing.T, f promoteHTTPTest, key, body string) *bytes.Buffer {
	t.Helper()
	r := projectAPIRequestWithHeaders(f.h, http.MethodPost, "/api/v1/sessions/s1/promote", body, key)
	if r.Code != http.StatusAccepted {
		t.Fatalf("promote = %d %s", r.Code, r.Body.String())
	}
	return r.Body
}

func projectAPIRequestWithHeaders(h http.Handler, method, target, body, key string) *responseCapture {
	r := httptestNewRequest(method, target, body)
	r.AddCookie(&http.Cookie{Name: "pa_session", Value: testSession})
	r.AddCookie(&http.Cookie{Name: "pa_csrf", Value: testCSRF})
	r.Header.Set("X-CSRF-Token", testCSRF)
	r.Header.Set("Idempotency-Key", key)
	w := &responseCapture{ResponseRecorder: httptestNewRecorder()}
	h.ServeHTTP(w, r)
	return w
}

// Aliases keep the fixture concise while retaining ordinary httptest behavior.
type responseCapture struct{ *httptest.ResponseRecorder }

var httptestNewRequest = func(method, target, body string) *http.Request {
	return httptest.NewRequest(method, target, bytes.NewBufferString(body))
}
var httptestNewRecorder = httptest.NewRecorder

func TestPromoteEndpointIdempotencyValidationAndStatus(t *testing.T) {
	f := newPromoteHTTPTest(t)
	body := `{"workspace_path":"draft.md","target_relative_path":"saved/draft.md","review_mode":"bites"}`
	first := submitPromote(t, f, " key-1 ", body)
	second := submitPromote(t, f, "key-1", body)
	if first.String() != second.String() {
		t.Fatalf("replay differs:\n%s\n%s", first.String(), second.String())
	}
	var accepted struct {
		OperationID string `json:"operation_id"`
		NoteID      string `json:"note_id"`
		Status      string `json:"status"`
	}
	if err := json.Unmarshal(first.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if accepted.OperationID == "" || accepted.NoteID == "" || accepted.Status != "completed" {
		t.Fatalf("accepted = %#v", accepted)
	}
	var ops, notes int
	_ = f.db.QueryRow(`SELECT count(*) FROM promote_ops`).Scan(&ops)
	_ = f.db.QueryRow(`SELECT count(*) FROM notes`).Scan(&notes)
	if ops != 1 || notes != 1 {
		t.Fatalf("durable counts = %d, %d", ops, notes)
	}

	status := projectAPIRequest(f.h, http.MethodGet, "/api/v1/operations/"+accepted.OperationID, "", true, "")
	if status.Code != 200 || !bytes.Contains(status.Body.Bytes(), []byte(`"badge":"Note saved; cards pending…"`)) || !bytes.Contains(status.Body.Bytes(), []byte(`"retry_cards":false`)) {
		t.Fatalf("status = %d %s", status.Code, status.Body.String())
	}
	if _, err := f.db.Exec(`UPDATE review_pending SET status='failed' WHERE note_id=?`, accepted.NoteID); err != nil {
		t.Fatal(err)
	}
	status = projectAPIRequest(f.h, http.MethodGet, "/api/v1/operations/"+accepted.OperationID, "", true, "")
	if !bytes.Contains(status.Body.Bytes(), []byte(`"badge":"Cards failed — Retry cards"`)) || !bytes.Contains(status.Body.Bytes(), []byte(`"retry_cards":true`)) {
		t.Fatalf("failed badge = %s", status.Body.String())
	}

	changed := projectAPIRequestWithHeaders(f.h, http.MethodPost, "/api/v1/sessions/s1/promote", `{"workspace_path":"draft.md","target_relative_path":"other.md","review_mode":"bites"}`, "key-1")
	if changed.Code != 409 || !bytes.Contains(changed.Body.Bytes(), []byte("idempotency_key_reused")) {
		t.Fatalf("changed = %d %s", changed.Code, changed.Body.String())
	}
}

func TestPromoteEndpointRejectsMalformedRequestsAndRequiresSecurity(t *testing.T) {
	f := newPromoteHTTPTest(t)
	valid := `{"workspace_path":"draft.md","target_relative_path":"x.md","review_mode":"none"}`
	for _, tc := range []struct{ key, body string }{{"", valid}, {"k", `{}`}, {"k", `{"workspace_path":"draft.md","target_relative_path":"x.md","review_mode":"bad"}`}, {"k", `{"workspace_path":"../draft.md","target_relative_path":"x.md","review_mode":"none"}`}, {"k", `{"workspace_path":"draft.md","target_relative_path":"x.txt","review_mode":"none"}`}, {"k", `{"workspace_path":"draft.md","target_relative_path":"x.md","review_mode":"none","extra":1}`}} {
		w := projectAPIRequestWithHeaders(f.h, http.MethodPost, "/api/v1/sessions/s1/promote", tc.body, tc.key)
		if w.Code != 400 {
			t.Errorf("key=%q body=%s => %d %s", tc.key, tc.body, w.Code, w.Body.String())
		}
	}
	if got := projectAPIRequest(f.h, http.MethodPost, "/api/v1/sessions/s1/promote", valid, false, "").Code; got != 401 {
		t.Errorf("unauth = %d", got)
	}
	if got := projectAPIRequest(f.h, http.MethodPost, "/api/v1/sessions/s1/promote", valid, true, "").Code; got != 403 {
		t.Errorf("csrf = %d", got)
	}
	if got := projectAPIRequest(f.h, http.MethodGet, "/api/v1/operations/missing", "", true, "").Code; got != 404 {
		t.Errorf("missing op = %d", got)
	}
	if got := projectAPIRequest(f.h, http.MethodGet, "/api/v1/operations/missing", "", false, "").Code; got != 401 {
		t.Errorf("unauth op = %d", got)
	}
}
