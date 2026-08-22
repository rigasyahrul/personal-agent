package store_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

func TestReadLessonsIndex_MissingFileReturnsEmpty(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	got, err := store.ReadLessonsIndex(root)
	if err != nil {
		t.Fatalf("ReadLessonsIndex missing: err = %v, want nil", err)
	}
	if got != "" {
		t.Fatalf("ReadLessonsIndex missing: got %q, want \"\"", got)
	}
}

func TestReadLessonsIndex_PresentContentReturned(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	want := "# Lessons\n\n- remember the path\n"
	if err := os.MkdirAll(filepath.Join(root, "memory"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "memory", "lessons.md"), []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := store.ReadLessonsIndex(root)
	if err != nil {
		t.Fatalf("ReadLessonsIndex: %v", err)
	}
	if got != want {
		t.Fatalf("ReadLessonsIndex content = %q, want %q", got, want)
	}
}

func TestReadLessonsIndex_PathUsesMemoryLessonsUnderScopeRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	// Write only via layout.LessonsPath so the helper must resolve the same path.
	path := layout.LessonsPath(root)
	if !strings.HasSuffix(filepath.ToSlash(path), "/memory/lessons.md") {
		t.Fatalf("layout.LessonsPath = %q, want .../memory/lessons.md", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	marker := "path-check-marker\n"
	if err := os.WriteFile(path, []byte(marker), 0o600); err != nil {
		t.Fatal(err)
	}
	// Decoy at wrong location must not be read.
	if err := os.WriteFile(filepath.Join(root, "lessons.md"), []byte("wrong-root\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := store.ReadLessonsIndex(root)
	if err != nil {
		t.Fatalf("ReadLessonsIndex: %v", err)
	}
	if got != marker {
		t.Fatalf("ReadLessonsIndex = %q, want content from memory/lessons.md under scope root", got)
	}
}

func TestReadLessonsIndex_EmptyFileReturnsEmpty(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	path := layout.LessonsPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte{}, 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := store.ReadLessonsIndex(root)
	if err != nil {
		t.Fatalf("ReadLessonsIndex empty: err = %v, want nil", err)
	}
	if got != "" {
		t.Fatalf("ReadLessonsIndex empty: got %q, want \"\"", got)
	}
}
