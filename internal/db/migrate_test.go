package db

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenMigratesAllTablesAndWAL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db", "personal-agent.sqlite")
	d, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })

	var mode string
	if err := d.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" {
		t.Fatalf("journal mode = %q, err = %v", mode, err)
	}

	tables := []string{
		"owner", "settings", "auth_sessions", "vaults", "projects", "sessions",
		"agent_runs", "messages", "notes", "promote_ops", "direct_ops",
		"review_pending", "review_items", "review_events", "backup_runs",
		"knowledge_notes", "compound_proposals", "note_links", "knowledge_fts",
	}
	for _, name := range tables {
		var n int
		if err := d.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&n); err != nil || n != 1 {
			t.Fatalf("table %s: count = %d, err = %v", name, n, err)
		}
	}
	var indexes int
	if err := d.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='index' AND name='agent_runs_one_active'").Scan(&indexes); err != nil || indexes != 1 {
		t.Fatalf("agent_runs_one_active: count = %d, err = %v", indexes, err)
	}
}

func TestOpenAppliesKnowledgeMigration002(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db", "personal-agent.sqlite")
	d, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })

	for _, name := range []string{"knowledge_notes", "compound_proposals", "note_links", "knowledge_fts"} {
		var n int
		if err := d.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&n); err != nil || n != 1 {
			t.Fatalf("table %s: count = %d, err = %v", name, n, err)
		}
	}

	var versions int
	if err := d.QueryRow(`SELECT count(*) FROM schema_migrations WHERE version IN ('001','002')`).Scan(&versions); err != nil || versions != 2 {
		t.Fatalf("schema_migrations 001+002 count = %d, err = %v", versions, err)
	}
}

func TestOpenMigrationIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db", "personal-agent.sqlite")
	for i := 0; i < 2; i++ {
		d, err := Open(context.Background(), path)
		if err != nil {
			t.Fatalf("open %d: %v", i+1, err)
		}
		if err := d.Close(); err != nil {
			t.Fatal(err)
		}
	}

	d, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	var n int
	if err := d.QueryRow("SELECT count(*) FROM schema_migrations WHERE version='001'").Scan(&n); err != nil || n != 1 {
		t.Fatalf("migration 001 records = %d, err = %v", n, err)
	}
	if err := d.QueryRow("SELECT count(*) FROM schema_migrations WHERE version='002'").Scan(&n); err != nil || n != 1 {
		t.Fatalf("migration 002 records = %d, err = %v", n, err)
	}
	// Ensure knowledge tables still present after second open
	var tables int
	if err := d.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name IN ('knowledge_notes','compound_proposals','note_links','knowledge_fts')`).Scan(&tables); err != nil || tables != 4 {
		t.Fatalf("knowledge tables after idempotent open = %d, err = %v", tables, err)
	}
}

func TestKnowledgeNotesScopeAndPartialUnique(t *testing.T) {
	d, err := Open(context.Background(), filepath.Join(t.TempDir(), "db", "personal-agent.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	if _, err := d.Exec(`INSERT INTO vaults(id,name,created_at,updated_at) VALUES('v','V',?,?);
		INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES('p','v','P',?,?)`, now, now, now, now); err != nil {
		t.Fatal(err)
	}

	// Two project-scoped rows with NULL vault_id / different paths — partial unique allows
	for i, path := range []string{"source/a.md", "memory/lessons.md"} {
		id := "kn" + string(rune('1'+i))
		_, err := d.Exec(`INSERT INTO knowledge_notes(
			id, kind, project_id, vault_id, is_global, relative_path, status, created_at, updated_at
		) VALUES(?,?,?,?,0,?,?,?,?)`, id, "source", "p", nil, path, "ready", now, now)
		if err != nil {
			t.Fatalf("insert knowledge_notes %s: %v", path, err)
		}
	}

	// Bad scope: is_global=1 with project_id set
	_, err = d.Exec(`INSERT INTO knowledge_notes(
		id, kind, project_id, vault_id, is_global, relative_path, status, created_at, updated_at
	) VALUES('bad','agents','p',NULL,1,'AGENTS.md','ready',?,?)`, now, now)
	if err == nil {
		t.Fatal("scope CHECK accepted is_global=1 with project_id")
	}

	// Duplicate project path rejected by partial unique
	_, err = d.Exec(`INSERT INTO knowledge_notes(
		id, kind, project_id, vault_id, is_global, relative_path, status, created_at, updated_at
	) VALUES('dup','source','p',NULL,0,'source/a.md','ready',?,?)`, now, now)
	if err == nil {
		t.Fatal("duplicate project relative_path accepted")
	}
}

func TestSessionScopeCheck(t *testing.T) {
	d, err := Open(context.Background(), filepath.Join(t.TempDir(), "db", "personal-agent.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	_, err = d.Exec(`INSERT INTO sessions(id,home,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at) VALUES('s','project','active','p','m','{}','{}','t','x','x')`)
	if err == nil {
		t.Fatal("invalid project scope accepted")
	}
}

func TestSessionScopeAndImmutableModel(t *testing.T) {
	d, err := Open(context.Background(), filepath.Join(t.TempDir(), "db", "personal-agent.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	now := time.Now().UTC()
	if _, err := d.Exec(`INSERT INTO vaults(id,name,created_at,updated_at) VALUES('v','V',?,?);
		INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES('p','v','P',?,?)`, now, now, now, now); err != nil {
		t.Fatal(err)
	}
	bad := []string{
		`INSERT INTO sessions(id,home,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at) VALUES('s1','project','active','p','m','{}','{}','',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		`INSERT INTO sessions(id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at) VALUES('s2','project','wrong','p','active','p','m','{}','{}','',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
	}
	for _, query := range bad {
		if _, err := d.Exec(query); err == nil {
			t.Fatalf("accepted invalid scope: %s", query)
		}
	}
	if _, err := d.Exec(`INSERT INTO sessions(id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at) VALUES('ok','project','v','p','active','p','m','{}','{}','',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`UPDATE sessions SET model_id='other' WHERE id='ok'`); err == nil {
		t.Fatal("model mutation accepted")
	}
}

func TestSettingsBackupScheduleDefaultsOff(t *testing.T) {
	d, err := Open(context.Background(), filepath.Join(t.TempDir(), "db", "personal-agent.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	var schedule string
	if err := d.QueryRow("SELECT backup_schedule FROM settings WHERE id=1").Scan(&schedule); err != nil {
		t.Fatal(err)
	}
	if schedule != "off" {
		t.Fatalf("backup_schedule = %q, want off", schedule)
	}
}
