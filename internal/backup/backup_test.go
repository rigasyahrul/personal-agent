package backup_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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

type memSink struct {
	keys []string
	err  error
}

func (m *memSink) Upload(_ context.Context, localDir, objectPrefix string) error {
	if m.err != nil {
		return m.err
	}
	return filepath.WalkDir(localDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}
		key := strings.TrimSuffix(objectPrefix, "/") + "/" + filepath.ToSlash(rel)
		m.keys = append(m.keys, key)
		return nil
	})
}

func testService(t *testing.T) (*backup.Service, *sql.DB, string) {
	t.Helper()
	dataDir := t.TempDir()
	db, err := dbopen.Open(context.Background(), filepath.Join(dataDir, "db", "personal-agent.sqlite"))
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
	svc := backup.NewService(db, dataDir, &backup.Barrier{}, &clock.FakeClock{T: time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)}, nil)
	// Always restore write bits under backups/ so t.TempDir cleanup succeeds after seals.
	t.Cleanup(func() { unsealTree(t, filepath.Join(dataDir, "backups")) })
	return svc, db, dataDir
}

func assertStoredStatus(t *testing.T, db *sql.DB, id, want string) {
	t.Helper()
	var got string
	if err := db.QueryRow(`SELECT status FROM backup_runs WHERE id=?`, id).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("status=%q want %q", got, want)
	}
}

func TestRunWithoutBucketSucceedsLocallyWithoutUpload(t *testing.T) {
	svc, db, _ := testService(t)
	up := &memSink{}
	svc.Sink = up
	run, err := svc.Run(context.Background())
	if err != nil || run.Status != "succeeded" || len(up.keys) != 0 {
		t.Fatalf("run=%+v keys=%v err=%v", run, up.keys, err)
	}
	t.Cleanup(func() { unsealTree(t, run.LocalPath) })
	assertStoredStatus(t, db, run.ID, "succeeded")
}

func TestConfiguredUploadControlsFinalStatus(t *testing.T) {
	svc, db, _ := testService(t)
	up := &memSink{}
	svc.Sink = up
	svc.Bucket = "archive"
	run, err := svc.Run(context.Background())
	if err != nil || run.Status != "succeeded" || len(up.keys) == 0 || run.ObjectKey == "" {
		t.Fatalf("run=%+v keys=%v err=%v", run, up.keys, err)
	}
	t.Cleanup(func() { unsealTree(t, run.LocalPath) })
	assertStoredStatus(t, db, run.ID, "succeeded")
	joined := strings.Join(up.keys, "\n")
	if !strings.Contains(joined, "manifest.json") || !strings.Contains(joined, "database.sqlite") || !strings.Contains(joined, "files/") {
		t.Fatalf("upload keys missing required members: %v", up.keys)
	}

	svc, db, _ = testService(t)
	svc.Bucket = "archive"
	svc.Sink = &memSink{err: fmt.Errorf("S3 unavailable")}
	run, err = svc.Run(context.Background())
	if err == nil || run.Status != "failed" || run.LocalPath == "" {
		t.Fatalf("run=%+v err=%v", run, err)
	}
	t.Cleanup(func() { unsealTree(t, run.LocalPath) })
	assertStoredStatus(t, db, run.ID, "failed")
}

func TestRunSucceedsWhenContextCanceledAfterLocalBundle(t *testing.T) {
	// After local seal + upload, CompleteBackupRun must not depend on a still-live
	// request context — otherwise client disconnect marks a durable snapshot failed.
	svc, db, _ := testService(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.Bucket = "archive"
	svc.Sink = &cancelAfterUploadSink{cancel: cancel}
	run, err := svc.Run(ctx)
	if run.LocalPath == "" {
		t.Fatalf("expected local bundle: run=%+v err=%v", run, err)
	}
	t.Cleanup(func() { unsealTree(t, run.LocalPath) })
	assertStoredStatus(t, db, run.ID, "succeeded")
	if run.Status != "succeeded" || run.ManifestHash == "" || run.ObjectKey == "" {
		t.Fatalf("run=%+v err=%v", run, err)
	}
	// err may be nil even if ctx is canceled after durable success.
	_ = err
}

// cancelAfterUploadSink cancels the parent context once Upload returns successfully,
// simulating client disconnect after the durable work finished.
type cancelAfterUploadSink struct {
	cancel context.CancelFunc
}

func (c *cancelAfterUploadSink) Upload(ctx context.Context, localDir, objectPrefix string) error {
	// Walk once to ensure the dir is readable (same shape as real sink).
	err := filepath.WalkDir(localDir, func(path string, d os.DirEntry, err error) error {
		return err
	})
	c.cancel()
	return err
}

func TestS3SinkUploadsDirectoryTree(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "database.sqlite"), []byte("db"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "files", "a"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "files", "a", "b.md"), []byte("note"), 0o600); err != nil {
		t.Fatal(err)
	}
	client := &fakeS3{}
	sink := backup.NewS3Sink(client, "bucket")
	if err := sink.Upload(context.Background(), dir, "backups/run1"); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"backups/run1/manifest.json":   true,
		"backups/run1/database.sqlite": true,
		"backups/run1/files/a/b.md":    true,
	}
	if len(client.keys) != len(want) {
		t.Fatalf("keys=%v", client.keys)
	}
	for _, k := range client.keys {
		if !want[k] {
			t.Fatalf("unexpected key %q in %v", k, client.keys)
		}
	}
}

type fakeS3 struct {
	keys []string
}

func (f *fakeS3) PutObject(_ context.Context, bucket, key string, body io.Reader, size int64, _ string) error {
	if bucket != "bucket" || size < 0 {
		return fmt.Errorf("bad put")
	}
	if _, err := io.ReadAll(body); err != nil {
		return err
	}
	f.keys = append(f.keys, key)
	return nil
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

func TestRestoreDrillFindsKnownNote(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	db, err := dbopen.Open(ctx, filepath.Join(dataDir, "db", "personal-agent.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	body := []byte("# restore me\n")
	sum := sha256.Sum256(body)
	source := layout.SourceDir(layout.ProjectRoot(dataDir, "", "p1"))
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "known.md"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO projects(id,name,created_at,updated_at) VALUES('p1','Known','2026-08-12T10:00:00Z','2026-08-12T10:00:00Z');
INSERT INTO notes(id,project_id,relative_path,content_sha256,byte_size,status,revision,created_at,updated_at)
VALUES('n1','p1','known.md',?,?,'ready',1,'2026-08-12T10:00:00Z','2026-08-12T10:00:00Z')`,
		hex.EncodeToString(sum[:]), len(body))
	if err != nil {
		t.Fatal(err)
	}
	svc := backup.NewService(db, dataDir, &backup.Barrier{}, &clock.FakeClock{T: time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)}, nil)
	run, err := svc.Run(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { unsealTree(t, run.LocalPath) })

	restoreDir := t.TempDir()
	restoreBundle(t, run.LocalPath, restoreDir)

	// Canonical restore layout: db/personal-agent.sqlite + files/**
	restored, err := dbopen.Open(ctx, filepath.Join(restoreDir, "db", "personal-agent.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	var path, hash string
	if err := restored.QueryRow(`SELECT relative_path,content_sha256 FROM notes WHERE id='n1' AND status='ready'`).Scan(&path, &hash); err != nil {
		t.Fatal(err)
	}
	restoredBody, err := os.ReadFile(filepath.Join(restoreDir, "files", "global", "projects", "p1", "source", filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	got := sha256.Sum256(restoredBody)
	if hash != hex.EncodeToString(got[:]) || string(restoredBody) != string(body) {
		t.Fatal("restored note failed integrity check")
	}
}

// restoreBundle validates manifest checksums and materializes a fresh PA_DATA_DIR layout.
func restoreBundle(t *testing.T, bundleDir, restoreDir string) {
	t.Helper()
	mb, err := os.ReadFile(filepath.Join(bundleDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m testManifest
	if err := json.Unmarshal(mb, &m); err != nil {
		t.Fatal(err)
	}
	for name, want := range m.Files {
		clean := filepath.Clean("/" + name)
		if strings.Contains(name, `\`) || strings.HasPrefix(name, "/") || strings.HasPrefix(clean, "/..") || name != strings.TrimPrefix(clean, "/") {
			// reject absolute / parent paths (canonical relative only)
			if filepath.IsAbs(name) || strings.HasPrefix(filepath.Clean(name), "..") || strings.Contains(name, "..") {
				t.Fatalf("unsafe manifest name %q", name)
			}
		}
		if filepath.IsAbs(name) || strings.Contains(filepath.ToSlash(name), "..") {
			t.Fatalf("unsafe manifest name %q", name)
		}
		src := filepath.Join(bundleDir, filepath.FromSlash(name))
		// Ensure still under bundleDir
		if !strings.HasPrefix(src, bundleDir) {
			t.Fatalf("path escape %q", name)
		}
		b, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(b)
		if hex.EncodeToString(sum[:]) != want {
			t.Fatalf("checksum mismatch for %s", name)
		}
		var dst string
		switch {
		case name == "database.sqlite":
			dst = filepath.Join(restoreDir, "db", "personal-agent.sqlite")
		case strings.HasPrefix(name, "files/") || strings.HasPrefix(name, "staging/"):
			dst = filepath.Join(restoreDir, filepath.FromSlash(name))
		default:
			t.Fatalf("unexpected payload %q", name)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Drop stale wal/shm if any were copied (should not be in bundle).
	_ = os.Remove(filepath.Join(restoreDir, "db", "personal-agent.sqlite-wal"))
	_ = os.Remove(filepath.Join(restoreDir, "db", "personal-agent.sqlite-shm"))
}
