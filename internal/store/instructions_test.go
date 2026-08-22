package store_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/store"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func TestNormalizeInstructionFile(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in       string
		fileName string
		kind     domain.KnowledgeKind
	}{
		{"agents", "AGENTS.md", domain.KnowledgeKindAgents},
		{"soul", "SOUL.md", domain.KnowledgeKindSoul},
		{"system", "SYSTEM.md", domain.KnowledgeKindSystem},
		{"AGENTS.md", "AGENTS.md", domain.KnowledgeKindAgents},
		{"SOUL.md", "SOUL.md", domain.KnowledgeKindSoul},
		{"SYSTEM.md", "SYSTEM.md", domain.KnowledgeKindSystem},
		{"Agents", "AGENTS.md", domain.KnowledgeKindAgents},
	}
	for _, tc := range cases {
		file, kind, err := store.NormalizeInstructionFile(tc.in)
		if err != nil || file != tc.fileName || kind != tc.kind {
			t.Fatalf("NormalizeInstructionFile(%q) = %q %q %v, want %q %q nil",
				tc.in, file, kind, err, tc.fileName, tc.kind)
		}
	}

	for _, bad := range []string{"", "../x", "memory/foo", "memory/lessons.md", "x.md", "readme", "AGENTS.txt"} {
		if _, _, err := store.NormalizeInstructionFile(bad); !errors.Is(err, store.ErrValidation) {
			t.Fatalf("NormalizeInstructionFile(%q) err = %v, want ErrValidation", bad, err)
		}
	}
}

func TestInstructionStorePutGetProject(t *testing.T) {
	db, dataDir := testutil.TempDB(t)
	clk := &clock.FakeClock{T: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)}
	ctx := context.Background()

	if _, err := db.Exec(`INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','P','x','x')`); err != nil {
		t.Fatal(err)
	}
	root := layout.ProjectRoot(dataDir, "", "p1")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}

	s := &store.InstructionStore{DB: db, Clock: clk}
	meta := store.ScopeMeta{
		DataDir:   dataDir,
		Scope:     domain.CompoundScopeProject,
		ProjectID: "p1",
	}
	body := "# Project agents\n"
	note, err := s.Put(ctx, meta, store.InstructionName("agents"), body)
	if err != nil {
		t.Fatal(err)
	}
	if note.Kind != domain.KnowledgeKindAgents || note.RelativePath != "AGENTS.md" || note.ProjectID != "p1" {
		t.Fatalf("note = %+v", note)
	}
	if note.IsGlobal || note.VaultID != "" || note.Status != "ready" || note.ID == "" {
		t.Fatalf("note scope/status = %+v", note)
	}
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte(body)))
	if note.ContentSHA256 != sum || note.ByteSize != int64(len(body)) {
		t.Fatalf("hash/size = %s %d", note.ContentSHA256, note.ByteSize)
	}
	if !note.CreatedAt.Equal(clk.T) || !note.UpdatedAt.Equal(clk.T) {
		t.Fatalf("timestamps = %v %v", note.CreatedAt, note.UpdatedAt)
	}

	// File on disk
	raw, err := os.ReadFile(layout.InstructionPath(root, "AGENTS.md"))
	if err != nil || string(raw) != body {
		t.Fatalf("disk = %q err=%v", raw, err)
	}
	info, err := os.Stat(layout.InstructionPath(root, "AGENTS.md"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("mode = %v err=%v", info.Mode(), err)
	}

	content, got, err := s.Get(ctx, root, store.InstructionName("agents"))
	if err != nil || content != body {
		t.Fatalf("Get content = %q err=%v", content, err)
	}
	if got.RelativePath != "AGENTS.md" || got.Kind != domain.KnowledgeKindAgents {
		t.Fatalf("Get note = %+v", got)
	}

	// knowledge_notes row
	var kind, path, projectID string
	var isGlobal int
	var vault sql.NullString
	err = db.QueryRow(`SELECT kind, relative_path, project_id, vault_id, is_global FROM knowledge_notes WHERE id=?`, note.ID).
		Scan(&kind, &path, &projectID, &vault, &isGlobal)
	if err != nil || kind != "agents" || path != "AGENTS.md" || projectID != "p1" || vault.Valid || isGlobal != 0 {
		t.Fatalf("row kind=%s path=%s pid=%s vault=%v global=%d err=%v", kind, path, projectID, vault, isGlobal, err)
	}
}

func TestInstructionStorePutEmptyContent(t *testing.T) {
	db, dataDir := testutil.TempDB(t)
	clk := &clock.FakeClock{T: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)}
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','P','x','x')`); err != nil {
		t.Fatal(err)
	}
	root := layout.ProjectRoot(dataDir, "", "p1")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	s := &store.InstructionStore{DB: db, Clock: clk}
	meta := store.ScopeMeta{DataDir: dataDir, Scope: domain.CompoundScopeProject, ProjectID: "p1"}
	note, err := s.Put(ctx, meta, "soul", "")
	if err != nil {
		t.Fatal(err)
	}
	if note.ByteSize != 0 || note.ContentSHA256 != fmt.Sprintf("%x", sha256.Sum256(nil)) {
		t.Fatalf("empty note = %+v", note)
	}
	content, _, err := s.Get(ctx, root, "soul")
	if err != nil || content != "" {
		t.Fatalf("Get empty = %q err=%v", content, err)
	}
}

func TestInstructionStorePutRejectsBadName(t *testing.T) {
	db, dataDir := testutil.TempDB(t)
	s := &store.InstructionStore{DB: db, Clock: clock.RealClock{}}
	meta := store.ScopeMeta{DataDir: dataDir, Scope: domain.CompoundScopeProject, ProjectID: "p1"}
	for _, name := range []store.InstructionName{"../x", "memory/foo", "memory/lessons.md"} {
		if _, err := s.Put(context.Background(), meta, name, "x"); !errors.Is(err, store.ErrValidation) {
			t.Fatalf("Put(%q) err = %v, want ErrValidation", name, err)
		}
	}
}

func TestInstructionStorePutRejectsVaultScope(t *testing.T) {
	db, dataDir := testutil.TempDB(t)
	s := &store.InstructionStore{DB: db, Clock: clock.RealClock{}}
	meta := store.ScopeMeta{
		DataDir: dataDir,
		Scope:   domain.CompoundScopeVault,
		VaultID: "v1",
	}
	if _, err := s.Put(context.Background(), meta, "agents", "x"); !errors.Is(err, store.ErrValidation) {
		t.Fatalf("vault Put err = %v, want ErrValidation", err)
	}
}

func TestInstructionStorePutGlobal(t *testing.T) {
	db, dataDir := testutil.TempDB(t)
	clk := &clock.FakeClock{T: time.Date(2026, 8, 22, 15, 0, 0, 0, time.UTC)}
	ctx := context.Background()
	root := layout.GlobalRoot(dataDir)
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	s := &store.InstructionStore{DB: db, Clock: clk}
	meta := store.ScopeMeta{DataDir: dataDir, Scope: domain.CompoundScopeGlobal}
	body := "# Global system\n"
	note, err := s.Put(ctx, meta, "system", body)
	if err != nil {
		t.Fatal(err)
	}
	if !note.IsGlobal || note.ProjectID != "" || note.VaultID != "" || note.Kind != domain.KnowledgeKindSystem {
		t.Fatalf("global note = %+v", note)
	}
	var isGlobal int
	var projectID, vaultID sql.NullString
	err = db.QueryRow(`SELECT is_global, project_id, vault_id FROM knowledge_notes WHERE id=?`, note.ID).
		Scan(&isGlobal, &projectID, &vaultID)
	if err != nil || isGlobal != 1 || projectID.Valid || vaultID.Valid {
		t.Fatalf("row global=%d pid=%v vid=%v err=%v", isGlobal, projectID, vaultID, err)
	}
	content, _, err := s.Get(ctx, root, "system")
	if err != nil || content != body {
		t.Fatalf("Get = %q err=%v", content, err)
	}
}

func TestInstructionStorePutOverwritesExisting(t *testing.T) {
	db, dataDir := testutil.TempDB(t)
	clk := &clock.FakeClock{T: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)}
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','P','x','x')`); err != nil {
		t.Fatal(err)
	}
	root := layout.ProjectRoot(dataDir, "", "p1")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	s := &store.InstructionStore{DB: db, Clock: clk}
	meta := store.ScopeMeta{DataDir: dataDir, Scope: domain.CompoundScopeProject, ProjectID: "p1"}

	first, err := s.Put(ctx, meta, "agents", "v1")
	if err != nil {
		t.Fatal(err)
	}
	clk.Advance(time.Hour)
	second, err := s.Put(ctx, meta, "agents", "v2")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("upsert should keep id: %s vs %s", first.ID, second.ID)
	}
	if second.ContentSHA256 != fmt.Sprintf("%x", sha256.Sum256([]byte("v2"))) {
		t.Fatalf("second hash = %s", second.ContentSHA256)
	}
	raw, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil || string(raw) != "v2" {
		t.Fatalf("disk = %q err=%v", raw, err)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM knowledge_notes WHERE project_id='p1' AND relative_path='AGENTS.md'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("row count = %d err=%v", n, err)
	}
}

func TestInstructionStoreGetMissing(t *testing.T) {
	db, dataDir := testutil.TempDB(t)
	root := layout.ProjectRoot(dataDir, "", "missing")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	s := &store.InstructionStore{DB: db, Clock: clock.RealClock{}}
	_, _, err := s.Get(context.Background(), root, "agents")
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("Get missing = %v, want ErrNotFound", err)
	}
}
