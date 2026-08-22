package app

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/config"
	database "github.com/rigasyahrul/personal-agent/internal/db"
	"github.com/rigasyahrul/personal-agent/internal/layout"
)

func TestNewRunsBiteWorker(t *testing.T) {
	dataDir := t.TempDir()
	seedPendingBite(t, dataDir)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{
				"role": "assistant", "content": `{"bites":[{"prompt":"What?","answer":"This."}]}`,
			}}},
		})
	}))
	defer provider.Close()

	application, err := New(context.Background(), config.Config{DataDir: dataDir, OpenAIBaseURL: provider.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()

	deadline := time.Now().Add(2 * time.Second)
	for {
		var status, lastError string
		var items int
		err := application.db.QueryRow(`SELECT status,coalesce(last_error,'') FROM review_pending WHERE id='pending'`).Scan(&status, &lastError)
		if err == nil {
			err = application.db.QueryRow(`SELECT count(*) FROM review_items WHERE generation_id='pending'`).Scan(&items)
		}
		if err == nil && status == "completed" && items == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("bite worker did not complete: status=%q items=%d last_error=%q err=%v", status, items, lastError, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestCloseCancelsActiveBiteWorker(t *testing.T) {
	dataDir := t.TempDir()
	seedPendingBite(t, dataDir)
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	provider := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		once.Do(func() { close(started) })
		<-release
	}))
	defer provider.Close()

	application, err := New(context.Background(), config.Config{DataDir: dataDir, OpenAIBaseURL: provider.URL})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("bite worker did not call provider")
	}
	closed := make(chan error, 1)
	go func() { closed <- application.Close() }()
	select {
	case err := <-closed:
		close(release)
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		close(release)
		t.Fatal("Close did not stop the bite worker")
	}
}

func seedPendingBite(t *testing.T, dataDir string) {
	t.Helper()
	if err := layout.EnsureProjectDirs(dataDir, "", "p1"); err != nil {
		t.Fatal(err)
	}
	body := []byte("# Note\n\nThis is the answer.\n")
	if err := os.WriteFile(filepath.Join(layout.SourceDir(layout.ProjectRoot(dataDir, "", "p1")), "note.md"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(body))
	db, err := database.Open(context.Background(), filepath.Join(dataDir, "db", "personal-agent.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = db.Exec(`INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','Project',?,?)`, now, now)
	if err == nil {
		_, err = db.Exec(`INSERT INTO notes(id,project_id,relative_path,content_sha256,byte_size,status,revision,created_at,updated_at) VALUES('n1','p1','note.md',?,?,'ready',1,?,?)`, hash, len(body), now, now)
	}
	if err == nil {
		_, err = db.Exec(`INSERT INTO review_pending(id,note_id,source_sha256,generator_version,status,attempts,created_at,updated_at) VALUES('pending','n1',?,'bites-v1','pending',0,?,?)`, hash, now, now)
	}
	if err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultStaticDirectoryIsViteDist(t *testing.T) {
	body, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `http.Dir("web/dist")`) {
		t.Fatal("default static directory must be web/dist")
	}
}

func TestNewWithDependenciesSeedsGlobalKnowledge(t *testing.T) {
	dataDir := t.TempDir()
	application, err := NewWithDependencies(context.Background(), config.Config{DataDir: dataDir}, Dependencies{
		DisableBackgroundWorkers: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close()

	root := layout.GlobalRoot(dataDir)
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(layout.CompoundingSkillPath(root)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(layout.LessonsPath(root)); err != nil {
		t.Fatal(err)
	}
}
