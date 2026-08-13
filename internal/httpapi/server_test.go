package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	database "github.com/rigasyahrul/personal-agent/internal/db"
)

func TestHealthSetupAndEmptyHome(t *testing.T) {
	dir := t.TempDir()
	d, err := database.Open(context.Background(), filepath.Join(dir, "db", "a.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	h := New(ServerDeps{
		DB:             d,
		DataDir:        dir,
		Clock:          &clock.FakeClock{T: time.Unix(0, 0)},
		BootstrapToken: "x",
	})
	for _, path := range []string{"/health", "/api/v1/setup/status", "/api/v1/home"} {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		if got := w.Header().Get("Content-Type"); got != "application/json" {
			t.Fatalf("%s content type = %q", path, got)
		}
		var v map[string]any
		if err := json.NewDecoder(w.Body).Decode(&v); err != nil {
			t.Fatalf("%s not JSON: %v", path, err)
		}
	}
}

func TestHealthReportsUnwritableStorage(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing")
	d, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "db.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	h := New(ServerDeps{DB: d, DataDir: dir, Clock: &clock.FakeClock{T: time.Unix(0, 0)}})
	w := httptest.NewRecorder()

	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
	var body map[string]bool
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["ok"] || body["storage_writable"] {
		t.Fatalf("body = %#v", body)
	}
}
