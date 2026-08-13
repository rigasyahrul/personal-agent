package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/auth"
	"github.com/rigasyahrul/personal-agent/internal/backup"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func TestBackupsRequireAuthAndPostRequiresCSRF(t *testing.T) {
	s := newBackupTestServer(t)
	for _, tc := range []struct {
		method string
		want   int
	}{
		{http.MethodGet, http.StatusUnauthorized},
		{http.MethodPost, http.StatusUnauthorized},
	} {
		r := httptest.NewRequest(tc.method, "/api/v1/backups", nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatalf("%s got %d", tc.method, w.Code)
		}
	}
	r := authenticatedBackupRequest(t, "POST", "/api/v1/backups", strings.NewReader(`{}`), false)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("got %d", w.Code)
	}
}

func TestBackupNowThenListAndSettingsStatus(t *testing.T) {
	s, dataDir := newBackupTestServerWithDir(t)
	source := layout.SourceDir(layout.ProjectRoot(dataDir, "", "p1"))
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "known.md"), []byte("# n\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := authenticatedBackupRequest(t, "POST", "/api/v1/backups", strings.NewReader(`{}`), true)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusCreated || !strings.Contains(w.Body.String(), `"status":"succeeded"`) {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	t.Cleanup(func() {
		_ = filepath.WalkDir(filepath.Join(dataDir, "backups"), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				_ = os.Chmod(path, 0o700)
			} else {
				_ = os.Chmod(path, 0o600)
			}
			return nil
		})
	})
	for _, path := range []string{"/api/v1/backups", "/api/v1/settings"} {
		r = authenticatedBackupRequest(t, "GET", path, nil, false)
		w = httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"last_success"`) {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
	}
}

func newBackupTestServer(t *testing.T) http.Handler {
	t.Helper()
	h, _ := newBackupTestServerWithDir(t)
	return h
}

func newBackupTestServerWithDir(t *testing.T) (http.Handler, string) {
	t.Helper()
	db, dataDir := testutil.TempDB(t)
	now := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)
	if _, err := db.Exec("INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,'hash','x','x')"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO auth_sessions(token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,?)",
		auth.TokenHash("session"), "csrf", now.Add(time.Hour).Format(time.RFC3339Nano), "x"); err != nil {
		t.Fatal(err)
	}
	barrier := &backup.Barrier{}
	svc := backup.NewService(db, dataDir, barrier, &clock.FakeClock{T: now}, nil)
	h := New(ServerDeps{
		DB: db, DataDir: dataDir, Clock: &clock.FakeClock{T: now},
		Backup: svc, Barrier: barrier, BackupSinkConfigured: false,
	})
	return h, dataDir
}

func authenticatedBackupRequest(t *testing.T, method, path string, body *strings.Reader, csrf bool) *http.Request {
	t.Helper()
	var r *http.Request
	if body == nil {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, body)
	}
	r.AddCookie(&http.Cookie{Name: "pa_session", Value: "session"})
	r.AddCookie(&http.Cookie{Name: "pa_csrf", Value: "csrf"})
	if csrf {
		r.Header.Set("X-CSRF-Token", "csrf")
	}
	if method == http.MethodPost || method == http.MethodPut {
		r.Header.Set("Content-Type", "application/json")
	}
	return r
}
