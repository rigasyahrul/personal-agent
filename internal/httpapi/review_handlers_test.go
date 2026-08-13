package httpapi

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/auth"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func reviewFixture(t *testing.T) (http.Handler, *clock.FakeClock, *testDB) {
	t.Helper()
	db, dir := testutil.TempDB(t)
	now := time.Date(2026, 8, 13, 12, 0, 0, 500_000_000, time.UTC)
	c := &clock.FakeClock{T: now}
	mustExec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	mustExec("INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,'x','x','x')")
	mustExec("INSERT INTO auth_sessions(token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,'x')", auth.TokenHash(testSession), testCSRF, now.Add(time.Hour).Format(time.RFC3339Nano))
	mustExec("INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','P1','x','x'),('p2','P2','x','x')")
	mustExec("INSERT INTO notes(id,project_id,relative_path,status,revision,created_at,updated_at) VALUES('n1','p1','a.md','ready',1,'x','x'),('n2','p2','b.md','ready',1,'x','x')")
	return New(ServerDeps{DB: db, DataDir: dir, Clock: c}), c, &testDB{exec: mustExec, queryRow: db.QueryRow}
}

type testDB struct {
	exec     func(string, ...any)
	queryRow func(string, ...any) *sql.Row
}

func insertReview(t *testing.T, d *testDB, id, project, note, kind, due, status string) {
	t.Helper()
	var revision int
	if err := d.queryRow("SELECT count(*)+1 FROM review_items WHERE note_id=?", note).Scan(&revision); err != nil {
		t.Fatal(err)
	}
	answer := any(nil)
	generation := any(nil)
	ordinal := any(nil)
	if kind == "bite" {
		answer, generation, ordinal = "answer", "gen", 0
	}
	d.exec(`INSERT INTO review_items(id,project_id,note_id,kind,source_sha256,source_revision,prompt,answer,generation_id,ordinal,due_at,status,scheduler_version) VALUES(?,?,?,?, 'hash',?,?,?,?,?,?,?,'sm2-lite-v1')`, id, project, note, kind, revision, "prompt-"+id, answer, generation, ordinal, due, status)
}

func reviewRequest(h http.Handler, method, path, body string, authenticated, csrf bool) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if authenticated {
		r.AddCookie(&http.Cookie{Name: "pa_session", Value: testSession})
	}
	if csrf {
		r.AddCookie(&http.Cookie{Name: "pa_csrf", Value: testCSRF})
		r.Header.Set("X-CSRF-Token", testCSRF)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestReviewQueueExplicitScopeChronologyLimitAndCaughtUp(t *testing.T) {
	h, c, d := reviewFixture(t)
	insertReview(t, d, "later", "p1", "n1", "whole", "2026-08-13T12:00:00.45Z", "active")
	insertReview(t, d, "earlier", "p1", "n1", "whole", "2026-08-13T12:00:00.4Z", "active")
	insertReview(t, d, "other", "p2", "n2", "whole", c.Now().Add(-time.Hour).Format(time.RFC3339Nano), "active")
	insertReview(t, d, "future", "p1", "n1", "whole", c.Now().Add(time.Hour).Format(time.RFC3339Nano), "active")
	for _, scope := range []string{"", "bogus", "project:"} {
		if got := reviewRequest(h, "GET", "/api/v1/review/queue?scope="+scope, "", true, false).Code; got != 400 {
			t.Errorf("scope %q = %d", scope, got)
		}
	}
	w := reviewRequest(h, "GET", "/api/v1/review/queue?scope=project:p1", "", true, false)
	if w.Code != 200 {
		t.Fatalf("queue=%d %s", w.Code, w.Body.String())
	}
	var got struct {
		Scope    string `json:"scope"`
		CaughtUp bool   `json:"caught_up"`
		Items    []struct {
			ID        string  `json:"id"`
			ProjectID string  `json:"project_id"`
			Answer    *string `json:"answer"`
		} `json:"items"`
	}
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Scope != "project:p1" || got.CaughtUp || len(got.Items) != 2 || got.Items[0].ID != "earlier" || got.Items[1].ID != "later" {
		t.Fatalf("queue=%+v", got)
	}
	if got.Items[0].ProjectID != "p1" || got.Items[0].Answer != nil {
		t.Fatalf("item=%+v", got.Items[0])
	}
	for i := 0; i < 51; i++ {
		insertReview(t, d, fmt.Sprintf("z%02d", i), "p2", "n2", "whole", c.Now().Add(-time.Minute).Format(time.RFC3339Nano), "active")
	}
	w = reviewRequest(h, "GET", "/api/v1/review/queue?scope=project:p2", "", true, false)
	got.Items = nil
	_ = json.NewDecoder(w.Body).Decode(&got)
	if len(got.Items) != 50 || got.CaughtUp {
		t.Fatalf("limited queue len=%d caught=%v", len(got.Items), got.CaughtUp)
	}
}

func TestReviewMutationsValidationIdempotencyAndBoundaries(t *testing.T) {
	h, c, d := reviewFixture(t)
	insertReview(t, d, "item", "p1", "n1", "whole", c.Now().Format(time.RFC3339Nano), "active")
	d.exec(`INSERT INTO review_pending(id,note_id,source_sha256,generator_version,status,attempts,lease_until,last_error,created_at,updated_at) VALUES('failed','n1','h','bites-v1','failed',2,'x','bad','x','x'),('pending','n1','h2','bites-v1','pending',0,NULL,NULL,'x','x')`)
	if got := reviewRequest(h, "GET", "/api/v1/review/queue?scope=all", "", false, false).Code; got != 401 {
		t.Fatalf("anonymous get=%d", got)
	}
	valid := `{"rating":"good","request_key":"key","row_version":1,"duration_ms":50}`
	if got := reviewRequest(h, "POST", "/api/v1/review/items/item/rate", valid, false, true).Code; got != 401 {
		t.Fatalf("anonymous post=%d", got)
	}
	if got := reviewRequest(h, "POST", "/api/v1/review/items/item/rate", valid, true, false).Code; got != 403 {
		t.Fatalf("csrf=%d", got)
	}
	first := reviewRequest(h, "POST", "/api/v1/review/items/item/rate", valid, true, true)
	replay := reviewRequest(h, "POST", "/api/v1/review/items/item/rate", valid, true, true)
	if first.Code != 200 || replay.Code != 200 || first.Body.String() != replay.Body.String() {
		t.Fatalf("rate=%d/%d %q/%q", first.Code, replay.Code, first.Body.String(), replay.Body.String())
	}
	reused := reviewRequest(h, "POST", "/api/v1/review/items/item/rate", `{"rating":"easy","request_key":"key","row_version":1,"duration_ms":50}`, true, true)
	if reused.Code != 409 || !strings.Contains(reused.Body.String(), `"error":"request_key_conflict"`) {
		t.Fatalf("request key reuse=%d %q", reused.Code, reused.Body.String())
	}
	var events int
	if err := d.queryRow("SELECT count(*) FROM review_events").Scan(&events); err != nil || events != 1 {
		t.Fatalf("events=%d err=%v", events, err)
	}
	if got := reviewRequest(h, "POST", "/api/v1/review/items/item/rate", `{"rating":"Good","request_key":"x","row_version":2,"duration_ms":0}`, true, true).Code; got != 400 {
		t.Errorf("rating=%d", got)
	}
	if got := reviewRequest(h, "POST", "/api/v1/review/items/item/rate", `{"rating":"good","request_key":"x","row_version":1,"duration_ms":0}`, true, true).Code; got != 409 {
		t.Errorf("stale=%d", got)
	}
	for i := 0; i < 2; i++ {
		if got := reviewRequest(h, "POST", "/api/v1/review/items/item/suspend", `{}`, true, true).Code; got != 204 {
			t.Fatalf("suspend %d=%d", i, got)
		}
	}
	if got := reviewRequest(h, "POST", "/api/v1/review/items/missing/suspend", `{}`, true, true).Code; got != 404 {
		t.Errorf("missing suspend=%d", got)
	}
	if got := reviewRequest(h, "POST", "/api/v1/review/pending/failed/retry", `{}`, true, true).Code; got != 204 {
		t.Errorf("retry=%d", got)
	}
	if got := reviewRequest(h, "POST", "/api/v1/review/pending/pending/retry", `{}`, true, true).Code; got != 409 {
		t.Errorf("retry conflict=%d", got)
	}
	if got := reviewRequest(h, "POST", "/api/v1/review/pending/missing/retry", `{}`, true, true).Code; got != 404 {
		t.Errorf("retry missing=%d", got)
	}
}
