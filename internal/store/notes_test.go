package store_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/paths"
	"github.com/rigasyahrul/personal-agent/internal/store"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func TestNoteTreeAndIntegrity(t *testing.T) {
	db, d := testutil.TempDB(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','P','x','x')`); err != nil {
		t.Fatal(err)
	}
	source := layout.SourceDir(layout.ProjectRoot(d, "", "p1"))
	if err := os.MkdirAll(filepath.Join(source, "guide"), 0700); err != nil {
		t.Fatal(err)
	}
	body := []byte("# Safe\n")
	path := filepath.Join(source, "guide", "safe.md")
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	if _, err := db.Exec(`INSERT INTO notes(id,project_id,relative_path,content_sha256,byte_size,status,revision,created_at,updated_at) VALUES('n1','p1','guide/safe.md',?,?,'ready',1,'x','x')`, fmt.Sprintf("%x", sum), len(body)); err != nil {
		t.Fatal(err)
	}
	s := store.NewNoteStore(db, d)
	tree, err := s.Tree(ctx, "p1")
	if err != nil || len(tree) != 2 || tree[1].NoteID != "n1" {
		t.Fatalf("%+v %v", tree, err)
	}
	doc, err := s.Get(ctx, "n1")
	if err != nil || string(doc.Body) != string(body) {
		t.Fatalf("%+v %v", doc, err)
	}
	if err := os.WriteFile(path, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err = s.Get(ctx, "n1")
	if !errors.Is(err, store.ErrIntegrity) {
		t.Fatalf("got %v", err)
	}
	if _, err = s.Get(ctx, "missing"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
}

func TestNoteTreeRejectsUnsafeAndUnindexedNodes(t *testing.T) {
	db, d := testutil.TempDB(t)
	if _, err := db.Exec(`INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','P','x','x')`); err != nil {
		t.Fatal(err)
	}
	source := layout.SourceDir(layout.ProjectRoot(d, "", "p1"))
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "rogue.md"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.NewNoteStore(db, d).Tree(context.Background(), "p1"); !errors.Is(err, store.ErrIntegrity) {
		t.Fatalf("unindexed: %v", err)
	}
}

func TestNoteGetRejectsSymlinkedNoteParent(t *testing.T) {
	db, d := testutil.TempDB(t)
	if _, err := db.Exec(`INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','P','x','x')`); err != nil {
		t.Fatal(err)
	}
	real := t.TempDir()
	if err := os.MkdirAll(real, 0700); err != nil {
		t.Fatal(err)
	}
	body := []byte("safe")
	if err := os.WriteFile(filepath.Join(real, "note.md"), body, 0600); err != nil {
		t.Fatal(err)
	}
	source := layout.SourceDir(layout.ProjectRoot(d, "", "p1"))
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, filepath.Join(source, "guide")); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	if _, err := db.Exec(`INSERT INTO notes(id,project_id,relative_path,content_sha256,byte_size,status,revision,created_at,updated_at) VALUES('n1','p1','guide/note.md',?,?,'ready',1,'x','x')`, fmt.Sprintf("%x", sum), len(body)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.NewNoteStore(db, d).Get(context.Background(), "n1"); !errors.Is(err, store.ErrIntegrity) {
		t.Fatalf("symlinked note parent: %v", err)
	}
}

func TestNoteGetRejectsInvalidSizesAndActualOversize(t *testing.T) {
	for _, tc := range []struct {
		name string
		size int64
		body []byte
	}{
		{name: "negative metadata", size: -1, body: []byte("x")},
		{name: "oversized metadata", size: paths.MaxMarkdownBytes + 1, body: []byte("x")},
		{name: "actual oversized body", size: paths.MaxMarkdownBytes, body: make([]byte, paths.MaxMarkdownBytes+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, d := testutil.TempDB(t)
			if tc.size < 0 {
				if _, err := db.Exec(`PRAGMA ignore_check_constraints=ON`); err != nil {
					t.Fatal(err)
				}
			}
			_, _ = db.Exec(`INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','P','x','x')`)
			source := layout.SourceDir(layout.ProjectRoot(d, "", "p1"))
			if err := os.MkdirAll(source, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(source, "note.md"), tc.body, 0600); err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(tc.body)
			_, _ = db.Exec(`INSERT INTO notes(id,project_id,relative_path,content_sha256,byte_size,status,revision,created_at,updated_at) VALUES('n1','p1','note.md',?,?,'ready',1,'x','x')`, fmt.Sprintf("%x", sum), tc.size)
			if _, err := store.NewNoteStore(db, d).Get(context.Background(), "n1"); !errors.Is(err, store.ErrIntegrity) {
				t.Fatalf("Get: %v", err)
			}
		})
	}
}

func TestNoteTreeRejectsInvalidWalkedFolderName(t *testing.T) {
	db, d := testutil.TempDB(t)
	if _, err := db.Exec(`INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','P','x','x')`); err != nil {
		t.Fatal(err)
	}
	source := layout.SourceDir(layout.ProjectRoot(d, "", "p1"))
	if err := os.MkdirAll(filepath.Join(source, `bad\folder`), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := store.NewNoteStore(db, d).Tree(context.Background(), "p1"); !errors.Is(err, store.ErrIntegrity) {
		t.Fatalf("invalid walked folder: %v", err)
	}
}
