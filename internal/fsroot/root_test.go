package fsroot_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/rigasyahrul/personal-agent/internal/fsroot"
	"golang.org/x/sys/unix"
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

func TestWriteFileNoReplaceConcurrentWriters(t *testing.T) {
	dir := t.TempDir()
	r, err := fsroot.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	const writers = 12
	start := make(chan struct{})
	errs := make(chan error, writers)
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- r.WriteFileNoReplace("deep/note.md", []byte("complete"), 0600)
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	success, exists := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			success++
		case errors.Is(err, fs.ErrExist):
			exists++
		default:
			t.Errorf("write: %v", err)
		}
	}
	if success != 1 || exists != writers-1 {
		t.Fatalf("success=%d exists=%d", success, exists)
	}
	b, err := os.ReadFile(filepath.Join(dir, "deep", "note.md"))
	if err != nil || string(b) != "complete" {
		t.Fatalf("body=%q err=%v", b, err)
	}
	entries, err := os.ReadDir(filepath.Join(dir, "deep"))
	if err != nil || len(entries) != 1 || entries[0].Name() != "note.md" {
		t.Fatalf("entries=%v err=%v", entries, err)
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

func TestRootReadWriteEditMkdirAndTree(t *testing.T) {
	r, err := fsroot.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := r.MkdirAll("drafts/chapter", 0755); err != nil {
		t.Fatal(err)
	}
	if err := r.WriteFileAtomic("drafts/chapter/notes.txt", []byte("alpha beta"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := r.EditFileAtomic("drafts/chapter/notes.txt", "beta", "gamma"); err != nil {
		t.Fatal(err)
	}
	got, err := r.ReadFile("drafts/chapter/notes.txt", 1024)
	if err != nil || string(got) != "alpha gamma" {
		t.Fatalf("got %q, err %v", got, err)
	}
	entries, err := r.Tree()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 || entries[2].Path != "drafts/chapter/notes.txt" || entries[2].Kind != "file" {
		t.Fatalf("unexpected tree: %#v", entries)
	}
}

func TestWriteFileAtomicRejectsSymlinkAndSpecialFile(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(filepath.Join(dir, "pipe"), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := fsroot.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := r.WriteFileAtomic("escape/new", []byte("owned"), 0600); !errors.Is(err, fsroot.ErrUnsafe) {
		t.Fatalf("symlink write: %v", err)
	}
	if err := r.WriteFileAtomic("pipe", []byte("owned"), 0600); !errors.Is(err, fsroot.ErrUnsafe) {
		t.Fatalf("special replacement: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "new")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("outside file changed: %v", err)
	}
}

func TestWriteFileAtomicDoesNotReplaceSpecialFileCreatedDuringCommit(t *testing.T) {
	dir := t.TempDir()
	r, err := fsroot.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	created := make(chan error, 1)
	go func() {
		for {
			entries, err := os.ReadDir(dir)
			if err != nil {
				created <- err
				return
			}
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), ".pa-write-") {
					created <- unix.Mkfifo(filepath.Join(dir, "note.md"), 0600)
					return
				}
			}
			runtime.Gosched()
		}
	}()

	err = r.WriteFileAtomic("note.md", make([]byte, 1<<20), 0600)
	if createErr := <-created; createErr != nil {
		t.Fatal(createErr)
	}
	if !errors.Is(err, fs.ErrExist) && !errors.Is(err, fsroot.ErrUnsafe) {
		t.Fatalf("write error = %v, want collision", err)
	}
	info, statErr := os.Lstat(filepath.Join(dir, "note.md"))
	if statErr != nil {
		t.Fatal(statErr)
	}
	if info.Mode()&os.ModeNamedPipe == 0 {
		t.Fatalf("destination was replaced: mode=%v", info.Mode())
	}
}

func TestWritesStayAnchoredWhenOpenedRootPathIsReplaced(t *testing.T) {
	for _, tc := range []struct {
		name  string
		write func(*fsroot.Root) error
	}{
		{"atomic", func(r *fsroot.Root) error {
			return r.WriteFileAtomic("nested/note.md", []byte("anchored"), 0600)
		}},
		{"no-replace", func(r *fsroot.Root) error {
			return r.WriteFileNoReplace("nested/note.md", []byte("anchored"), 0600)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent := t.TempDir()
			opened := filepath.Join(parent, "root")
			moved := filepath.Join(parent, "moved")
			if err := os.MkdirAll(filepath.Join(opened, "nested"), 0700); err != nil {
				t.Fatal(err)
			}
			r, err := fsroot.Open(opened)
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			if err := os.Rename(opened, moved); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(opened, "nested"), 0700); err != nil {
				t.Fatal(err)
			}

			if err := tc.write(r); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(moved, "nested", "note.md"))
			if err != nil || string(got) != "anchored" {
				t.Fatalf("opened root content = %q, %v", got, err)
			}
			if _, err := os.Stat(filepath.Join(opened, "nested", "note.md")); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("replacement root was modified: %v", err)
			}
		})
	}
}

func TestWriteFileAtomicCommitFailurePreservesOldRegularFileAndCleansTemps(t *testing.T) {
	dir := t.TempDir()
	old := []byte("old bytes stay exactly intact")
	if err := os.WriteFile(filepath.Join(dir, "note.md"), old, 0600); err != nil {
		t.Fatal(err)
	}
	r, err := fsroot.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0700) })

	if err := r.WriteFileAtomic("note.md", []byte("replacement"), 0600); err == nil {
		t.Fatal("write unexpectedly succeeded")
	}
	got, err := os.ReadFile(filepath.Join(dir, "note.md"))
	if err != nil || string(got) != string(old) {
		t.Fatalf("old file = %q, %v", got, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".pa-write-") {
			t.Fatalf("temporary sibling remains: %s", entry.Name())
		}
	}
}

func TestEditFileAtomicRequiresExactlyOneMatch(t *testing.T) {
	r, err := fsroot.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if err := r.WriteFileAtomic("note.txt", []byte("same same"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := r.EditFileAtomic("note.txt", "same", "new"); err == nil {
		t.Fatal("duplicate old text accepted")
	}
}
