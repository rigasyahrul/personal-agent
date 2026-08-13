package layout

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectLayoutAndCreation(t *testing.T) {
	d := t.TempDir()
	if got, want := ProjectRoot(d, "", "p1"), filepath.Join(d, "files", "global", "projects", "p1"); got != want {
		t.Fatalf("root=%q want %q", got, want)
	}
	if got, want := ProjectRoot(d, "v1", "p1"), filepath.Join(d, "files", "vaults", "v1", "projects", "p1"); got != want {
		t.Fatalf("vault root=%q want %q", got, want)
	}
	if got := SourceDir(ProjectRoot(d, "", "p1")); got != filepath.Join(d, "files", "global", "projects", "p1", "source") {
		t.Fatal(got)
	}
	if got := SessionWorkspace(d, SessionHome("project"), "v1", "p1", "s1"); got != filepath.Join(d, "files", "vaults", "v1", "projects", "p1", "sessions", "s1") {
		t.Fatal(got)
	}
	if err := EnsureProjectDirs(d, "v1", "p1"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"source", "memory", "soul"} {
		if st, err := os.Stat(filepath.Join(ProjectRoot(d, "v1", "p1"), name)); err != nil || !st.IsDir() {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestEnsureProjectDirsCleansPartialFreshRootAndPreservesExistingRoot(t *testing.T) {
	d := t.TempDir()
	root := ProjectRoot(d, "", "partial")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "memory"), []byte("block"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureProjectDirs(d, "", "partial"); err == nil {
		t.Fatal("expected preexisting root rejection")
	}
	if b, err := os.ReadFile(filepath.Join(root, "memory")); err != nil || string(b) != "block" {
		t.Fatalf("preexisting changed: %q %v", b, err)
	}

	blockedParent := filepath.Join(d, "blocked")
	if err := os.WriteFile(blockedParent, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureProjectDirs(blockedParent, "", "p"); err == nil {
		t.Fatal("expected parent failure")
	}
	if _, err := os.Stat(ProjectRoot(blockedParent, "", "p")); !errors.Is(err, os.ErrNotExist) && err == nil {
		t.Fatalf("partial root remains: %v", err)
	}
}

func TestEnsureProjectDirsCleansFreshRootAfterChildFailure(t *testing.T) {
	d := t.TempDir()
	original := mkdirProjectChild
	calls := 0
	mkdirProjectChild = func(name string, perm os.FileMode) error {
		calls++
		if calls == 2 {
			return errors.New("injected child failure")
		}
		return os.Mkdir(name, perm)
	}
	t.Cleanup(func() { mkdirProjectChild = original })
	root := ProjectRoot(d, "", "p")
	if err := EnsureProjectDirs(d, "", "p"); err == nil {
		t.Fatal("expected child failure")
	}
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial root remains: %v", err)
	}
}

func TestSessionWorkspaceAllHomes(t *testing.T) {
	d := t.TempDir()
	cases := map[SessionHome]string{
		"global":  filepath.Join(d, "files", "global", "sessions", "s"),
		"vault":   filepath.Join(d, "files", "vaults", "v", "sessions", "s"),
		"project": filepath.Join(d, "files", "vaults", "v", "projects", "p", "sessions", "s"),
	}
	for home, want := range cases {
		if got := SessionWorkspace(d, home, "v", "p", "s"); got != want {
			t.Errorf("%s: %q want %q", home, got, want)
		}
	}
}
