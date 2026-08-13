package store_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/config"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/store"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func seedProject(t *testing.T, dataDir string) *store.SessionStore {
	t.Helper()
	conn := testutil.OpenDB(t, dataDir)
	now := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	if _, err := conn.Exec(`INSERT INTO vaults(id,name,created_at,updated_at) VALUES('v1','Vault',?,?);
		INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES('p1','v1','Project',?,?)`, now, now, now, now); err != nil {
		t.Fatal(err)
	}
	return &store.SessionStore{
		DB: conn, DataDir: dataDir, Now: func() time.Time { return now },
		Models: []config.ModelRef{{Provider: "openai", ModelID: "gpt-test"}},
	}
}

func TestSessionStoreCreateProjectAndList(t *testing.T) {
	dataDir := t.TempDir()
	ss := seedProject(t, dataDir)

	got, err := ss.CreateProject(context.Background(), store.CreateSessionInput{
		ProjectID: "p1", Title: "Learn", Provider: "openai", ModelID: "gpt-test",
		ModelParametersJSON: `{"temperature":0.2}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Home != layout.SessionHome("project") || got.VaultID == nil || *got.VaultID != "v1" || got.ProjectID == nil || *got.ProjectID != "p1" {
		t.Fatalf("wrong scope: %#v", got)
	}
	if got.Provider != "openai" || got.ModelID != "gpt-test" || got.ModelParametersJSON != `{"temperature":0.2}` || got.ToolGrantsJSON != `{"workspace_files":false}` {
		t.Fatalf("wrong model or grants: %#v", got)
	}
	workspace := layout.SessionWorkspace(dataDir, got.Home, "v1", "p1", got.ID)
	if info, err := os.Stat(workspace); err != nil || !info.IsDir() {
		t.Fatalf("workspace: %v", err)
	}
	listed, err := ss.ListByProject(context.Background(), "p1")
	if err != nil || len(listed) != 1 || listed[0].ID != got.ID {
		t.Fatalf("list: %#v %v", listed, err)
	}
}

type cancelAfterSessionCommitContext struct {
	context.Context
	db   interface{ QueryRow(string, ...any) *sql.Row }
	done chan struct{}
	once sync.Once
}

func (c *cancelAfterSessionCommitContext) Done() <-chan struct{} {
	var count int
	if err := c.db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&count); err == nil && count > 0 {
		c.once.Do(func() { close(c.done) })
	}
	return c.done
}

func (c *cancelAfterSessionCommitContext) Err() error {
	select {
	case <-c.done:
		return context.Canceled
	default:
		return nil
	}
}

func TestSessionStoreCreateProjectReturnsCommittedSessionWhenContextCanceledAfterCommit(t *testing.T) {
	dataDir := t.TempDir()
	ss := seedProject(t, dataDir)
	observer := testutil.OpenDB(t, dataDir)
	ctx := &cancelAfterSessionCommitContext{Context: context.Background(), db: observer, done: make(chan struct{})}

	got, err := ss.CreateProject(ctx, store.CreateSessionInput{
		ProjectID: "p1", Title: "Learn", Provider: "openai", ModelID: "gpt-test", ModelParametersJSON: `{}`,
	})
	if err != nil {
		t.Fatalf("CreateProject returned an error after durable commit: %v", err)
	}
	if got.ID == "" || got.ProjectID == nil || *got.ProjectID != "p1" {
		t.Fatalf("wrong committed session: %#v", got)
	}
}

func TestSessionStoreRejectsMissingProjectWithoutDirectory(t *testing.T) {
	dataDir := t.TempDir()
	ss := &store.SessionStore{DB: testutil.OpenDB(t, dataDir), DataDir: dataDir, Now: time.Now, Models: []config.ModelRef{{Provider: "openai", ModelID: "m"}}}
	_, err := ss.CreateProject(context.Background(), store.CreateSessionInput{ProjectID: "missing", Provider: "openai", ModelID: "m", ModelParametersJSON: `{}`})
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
	entries, readErr := os.ReadDir(filepath.Join(dataDir, "files"))
	if readErr == nil && len(entries) != 0 {
		t.Fatalf("unexpected directories: %v", entries)
	}
}

func TestSessionStoreRejectsUnconfiguredModels(t *testing.T) {
	for _, tc := range []struct {
		name   string
		models []config.ModelRef
	}{
		{name: "empty configuration"},
		{name: "unknown pair", models: []config.ModelRef{{Provider: "openai", ModelID: "allowed"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dataDir := t.TempDir()
			ss := seedProject(t, dataDir)
			ss.Models = tc.models
			_, err := ss.CreateProject(context.Background(), store.CreateSessionInput{ProjectID: "p1", Provider: "other", ModelID: "model", ModelParametersJSON: `{}`})
			if !errors.Is(err, store.ErrValidation) {
				t.Fatalf("error = %v, want ErrValidation", err)
			}
			var count int
			if err := ss.DB.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&count); err != nil || count != 0 {
				t.Fatalf("session count = %d, error = %v", count, err)
			}
		})
	}
}

func TestSessionDeleteRemovesOnlyWorkspace(t *testing.T) {
	dataDir := t.TempDir()
	ss := seedProject(t, dataDir)
	session, err := ss.CreateProject(context.Background(), store.CreateSessionInput{
		ProjectID: "p1", Provider: "openai", ModelID: "gpt-test", ModelParametersJSON: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	workspace := layout.SessionWorkspace(dataDir, session.Home, "v1", "p1", session.ID)
	if err := os.WriteFile(filepath.Join(workspace, "draft.md"), []byte("draft"), 0o600); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(layout.SourceDir(layout.ProjectRoot(dataDir, "v1", "p1")), "kept.md")
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ss.Delete(context.Background(), session.ID); err != nil {
		t.Fatal(err)
	}
	got, err := ss.Get(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "terminal" || got.DeletedAt == nil {
		t.Fatalf("not tombstoned: %#v", got)
	}
	if _, err := os.Stat(workspace); !os.IsNotExist(err) {
		t.Fatalf("workspace remains: %v", err)
	}
	if body, err := os.ReadFile(source); err != nil || string(body) != "source" {
		t.Fatalf("source changed: %q %v", body, err)
	}
	if err := ss.Delete(context.Background(), session.ID); err != nil {
		t.Fatalf("terminal delete should be idempotent: %v", err)
	}
}

func TestSessionDeleteBlocksActiveRunUntilTerminal(t *testing.T) {
	dataDir := t.TempDir()
	ss := seedProject(t, dataDir)
	session, err := ss.CreateProject(context.Background(), store.CreateSessionInput{
		ProjectID: "p1", Provider: "openai", ModelID: "gpt-test", ModelParametersJSON: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	workspace := layout.SessionWorkspace(dataDir, session.Home, "v1", "p1", session.ID)
	if _, err := ss.DB.Exec(`INSERT INTO agent_runs(id,session_id,request_key,status,created_at) VALUES('run',?,?,'running',?)`, session.ID, "request", time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	if err := ss.Delete(context.Background(), session.ID); !errors.Is(err, store.ErrSessionBusy) {
		t.Fatalf("Delete error = %v, want ErrSessionBusy", err)
	}
	got, err := ss.Get(context.Background(), session.ID)
	if err != nil || got.Status != "active" || got.DeletedAt != nil {
		t.Fatalf("session changed while run active: %#v, %v", got, err)
	}
	if info, err := os.Stat(workspace); err != nil || !info.IsDir() {
		t.Fatalf("workspace removed while run active: %v", err)
	}

	if _, err := ss.DB.Exec(`UPDATE agent_runs SET status='completed',completed_at=? WHERE id='run'`, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := ss.Delete(context.Background(), session.ID); err != nil {
		t.Fatalf("Delete after terminal run: %v", err)
	}
	got, err = ss.Get(context.Background(), session.ID)
	if err != nil || got.Status != "terminal" || got.DeletedAt == nil {
		t.Fatalf("session not tombstoned: %#v, %v", got, err)
	}
	if _, err := os.Stat(workspace); !os.IsNotExist(err) {
		t.Fatalf("workspace remains: %v", err)
	}
}

func TestSessionDeleteRetriesWorkspaceCleanupAfterTombstone(t *testing.T) {
	dataDir := t.TempDir()
	ss := seedProject(t, dataDir)
	session, err := ss.CreateProject(context.Background(), store.CreateSessionInput{
		ProjectID: "p1", Provider: "openai", ModelID: "gpt-test", ModelParametersJSON: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	workspace := layout.SessionWorkspace(dataDir, session.Home, "v1", "p1", session.ID)
	projectRoot := layout.ProjectRoot(dataDir, "v1", "p1")
	savedRoot := projectRoot + ".saved"
	if err := os.Rename(projectRoot, savedRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectRoot, []byte("blocks workspace traversal"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := ss.Delete(context.Background(), session.ID); err == nil {
		t.Fatal("Delete succeeded despite failed workspace cleanup")
	}
	got, err := ss.Get(context.Background(), session.ID)
	if err != nil || got.Status != "terminal" || got.DeletedAt == nil {
		t.Fatalf("session was not tombstoned before cleanup: %#v, %v", got, err)
	}
	if err := os.Remove(projectRoot); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(savedRoot, projectRoot); err != nil {
		t.Fatal(err)
	}

	if err := ss.Delete(context.Background(), session.ID); err != nil {
		t.Fatalf("retry Delete: %v", err)
	}
	if _, err := os.Stat(workspace); !os.IsNotExist(err) {
		t.Fatalf("workspace remains after retry: %v", err)
	}
}

func TestSessionDeleteConcurrentCallsConverge(t *testing.T) {
	dataDir := t.TempDir()
	ss := seedProject(t, dataDir)
	session, err := ss.CreateProject(context.Background(), store.CreateSessionInput{
		ProjectID: "p1", Provider: "openai", ModelID: "gpt-test", ModelParametersJSON: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	workspace := layout.SessionWorkspace(dataDir, session.Home, "v1", "p1", session.ID)
	start := make(chan struct{})
	errs := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			errs <- ss.Delete(context.Background(), session.ID)
		}()
	}
	close(start)
	for range 2 {
		if err := <-errs; err != nil {
			t.Errorf("concurrent Delete: %v", err)
		}
	}
	got, err := ss.Get(context.Background(), session.ID)
	if err != nil || got.Status != "terminal" || got.DeletedAt == nil {
		t.Fatalf("session not tombstoned: %#v, %v", got, err)
	}
	if _, err := os.Stat(workspace); !os.IsNotExist(err) {
		t.Fatalf("workspace remains: %v", err)
	}
}
