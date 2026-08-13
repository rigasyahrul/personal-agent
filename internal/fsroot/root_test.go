package fsroot_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/rigasyahrul/personal-agent/internal/fsroot"
)

func TestRootRejectsSymlinksAndNonRegularFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/note.md", []byte("ok"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("note.md", dir+"/link.md"); err != nil {
		t.Fatal(err)
	}
	r, err := fsroot.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if _, err = r.ReadFile("note.md", 1024); err != nil {
		t.Fatal(err)
	}
	if _, err = r.ReadFile("link.md", 1024); !errors.Is(err, fsroot.ErrUnsafe) {
		t.Fatalf("symlink read: %v", err)
	}
	if err = r.MkdirAll("link.md/child", 0700); !errors.Is(err, fsroot.ErrUnsafe) {
		t.Fatalf("symlink mkdir: %v", err)
	}
	if _, err = r.ReadFile("../note.md", 1024); !errors.Is(err, fsroot.ErrInvalidPath) {
		t.Fatalf("invalid path: %v", err)
	}
}

func TestOpenRejectsSymlinkedAbsoluteParent(t *testing.T) {
	parent := t.TempDir()
	real := filepath.Join(parent, "real")
	if err := os.MkdirAll(filepath.Join(real, "root"), 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(parent, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if _, err := fsroot.Open(filepath.Join(link, "root")); !errors.Is(err, fsroot.ErrUnsafe) {
		t.Fatalf("Open through symlinked parent: %v", err)
	}
}

func TestReadFileRejectsSymlinkedParent(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "note.md"), []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "linked")); err != nil {
		t.Fatal(err)
	}
	r, err := fsroot.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if _, err := r.ReadFile("linked/note.md", 1024); !errors.Is(err, fsroot.ErrUnsafe) {
		t.Fatalf("ReadFile through symlinked parent: %v", err)
	}
}

func TestWalkRejectsInvalidLogicalEntryPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, `bad\name`), 0700); err != nil {
		t.Fatal(err)
	}
	r, err := fsroot.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	called := false
	err = r.Walk(func(string, os.FileInfo) error { called = true; return nil })
	if !errors.Is(err, fsroot.ErrInvalidPath) || called {
		t.Fatalf("Walk invalid entry: err=%v called=%v", err, called)
	}
}

func TestReadFileRejectsBodyOverLimit(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "large.md"), make([]byte, 11), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := fsroot.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if _, err := r.ReadFile("large.md", 10); !errors.Is(err, fsroot.ErrUnsafe) {
		t.Fatalf("oversized ReadFile: %v", err)
	}
}

func TestWriteFileNoReplaceRejectsCollisionAndSymlinkParent(t *testing.T) {
	dir := t.TempDir()
	r, err := fsroot.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := r.WriteFileNoReplace("docs/note.md", []byte("one"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteFileNoReplace("docs/note.md", []byte("two"), 0600); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("collision: %v", err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "docs", "note.md"))
	if string(b) != "one" {
		t.Fatalf("overwritten: %q", b)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteFileNoReplace("link/note.md", []byte("x"), 0600); !errors.Is(err, fsroot.ErrUnsafe) {
		t.Fatalf("symlink: %v", err)
	}
}
