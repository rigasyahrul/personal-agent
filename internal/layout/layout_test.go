package layout

import (
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
