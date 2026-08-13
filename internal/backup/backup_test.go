package backup_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/backup"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	dbopen "github.com/rigasyahrul/personal-agent/internal/db"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	_ "modernc.org/sqlite"
)

func TestRunCreatesConsistentLocalBundleAndSucceededRun(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	db, err := dbopen.Open(ctx, filepath.Join(dataDir, "db", "personal-agent.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','Known','2026-08-12T10:00:00Z','2026-08-12T10:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	source := layout.SourceDir(layout.ProjectRoot(dataDir, "", "p1"))
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "known.md"), []byte("# known note\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	barrier := &backup.Barrier{}
	svc := backup.NewService(db, dataDir, barrier, &clock.FakeClock{T: time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)}, nil)
	run, err := svc.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != "succeeded" || run.LocalPath == "" || run.ManifestHash == "" {
		t.Fatalf("run=%+v", run)
	}
	// Sealed trees are not owner-writable; restore write bits so TempDir cleanup works.
	t.Cleanup(func() { unsealTree(t, run.LocalPath) })
	info, err := os.Stat(run.LocalPath)
	if err != nil || !info.IsDir() {
		t.Fatalf("LocalPath must be directory: path=%q err=%v", run.LocalPath, err)
	}
	if !strings.Contains(filepath.ToSlash(run.LocalPath), "/backups/local/") {
		t.Fatalf("LocalPath not under backups/local: %q", run.LocalPath)
	}

	manifest, files := readBundleDir(t, run.LocalPath)
	if manifest.CutoffAt != "2026-08-12T10:00:00Z" {
		t.Fatalf("cutoff=%q", manifest.CutoffAt)
	}
	for name, want := range manifest.Files {
		sum := sha256.Sum256(files[name])
		if hex.EncodeToString(sum[:]) != want {
			t.Fatalf("checksum mismatch for %s", name)
		}
	}
	mb, err := os.ReadFile(filepath.Join(run.LocalPath, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	gotMH := sha256.Sum256(mb)
	if hex.EncodeToString(gotMH[:]) != run.ManifestHash {
		t.Fatalf("manifest_hash mismatch")
	}

	snapshot := filepath.Join(t.TempDir(), "snapshot.sqlite")
	if err := os.WriteFile(snapshot, files["database.sqlite"], 0o600); err != nil {
		t.Fatal(err)
	}
	restored, err := sql.Open("sqlite", snapshot)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	var name string
	if err := restored.QueryRow(`SELECT name FROM projects WHERE id='p1'`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Known" {
		t.Fatalf("project name=%q", name)
	}
	if string(files["files/global/projects/p1/source/known.md"]) != "# known note\n" {
		t.Fatal("bundle omitted known data")
	}

	// Sealed: owner must not be able to write into the published bundle.
	if info.Mode().Perm()&0o200 != 0 {
		t.Fatalf("bundle dir still owner-writable: %v", info.Mode())
	}
	dbInfo, err := os.Stat(filepath.Join(run.LocalPath, "database.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if dbInfo.Mode().Perm()&0o200 != 0 {
		t.Fatalf("database.sqlite still owner-writable: %v", dbInfo.Mode())
	}
}

func TestBarrierBlocksMutationDuringSnapshot(t *testing.T) {
	b := &backup.Barrier{}
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	go func() {
		_ = b.Snapshot(func() error {
			close(entered)
			<-release
			return nil
		})
	}()
	<-entered
	go func() {
		_ = b.Mutate(func() error {
			close(done)
			return nil
		})
	}()
	select {
	case <-done:
		t.Fatal("mutation crossed snapshot barrier")
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("mutation remained blocked")
	}
}

type testManifest struct {
	Version  int               `json:"version"`
	CutoffAt string            `json:"cutoff_at"`
	Files    map[string]string `json:"files"`
}

func readBundleDir(t *testing.T, dir string) (testManifest, map[string][]byte) {
	t.Helper()
	mb, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m testManifest
	if err := json.Unmarshal(mb, &m); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{"manifest.json": mb}
	for name := range m.Files {
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		files[name] = b
	}
	return m, files
}

func unsealTree(t *testing.T, root string) {
	t.Helper()
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			_ = os.Chmod(path, 0o700)
		} else {
			_ = os.Chmod(path, 0o600)
		}
		return nil
	})
}
