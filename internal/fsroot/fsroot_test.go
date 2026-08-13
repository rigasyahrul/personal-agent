package fsroot_test

import (
	"errors"
	"os"
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
	if _, err = r.ReadFile("note.md"); err != nil {
		t.Fatal(err)
	}
	if _, err = r.ReadFile("link.md"); !errors.Is(err, fsroot.ErrUnsafe) {
		t.Fatalf("symlink read: %v", err)
	}
	if err = r.MkdirAll("link.md/child", 0700); !errors.Is(err, fsroot.ErrUnsafe) {
		t.Fatalf("symlink mkdir: %v", err)
	}
	if _, err = r.ReadFile("../note.md"); !errors.Is(err, fsroot.ErrInvalidPath) {
		t.Fatalf("invalid path: %v", err)
	}
}
