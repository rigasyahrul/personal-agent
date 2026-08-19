package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/ids"
	"github.com/rigasyahrul/personal-agent/internal/store"
	"modernc.org/sqlite"
)

// Barrier serializes durable mutations vs exclusive backup snapshots.
// Mutate takes a shared lock (many concurrent mutators).
// Snapshot takes the exclusive lock (blocks new and waits for in-flight Mutate).
type Barrier struct{ mu sync.RWMutex }

func (b *Barrier) Mutate(fn func() error) error {
	if b == nil {
		return fn()
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	return fn()
}

func (b *Barrier) Snapshot(fn func() error) error {
	if b == nil {
		return fn()
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return fn()
}

// Sink uploads a completed local bundle directory. nil Sink => local-only success.
type Sink interface {
	// Upload mirrors the directory tree to object storage under objectPrefix/.
	// Every file under localDir is uploaded; keys are objectPrefix + relative path.
	Upload(ctx context.Context, localDir string, objectPrefix string) error
}

// Uploader is an alias retained for plan compatibility.
type Uploader = Sink

type Service struct {
	DB      *sql.DB
	DataDir string
	Barrier *Barrier
	Clock   clock.Clock
	Sink    Sink
	// Bucket, when non-empty, requires Sink.Upload before marking succeeded.
	Bucket string
}

func NewService(db *sql.DB, dataDir string, barrier *Barrier, c clock.Clock, sink Sink) *Service {
	return &Service{DB: db, DataDir: dataDir, Barrier: barrier, Clock: c, Sink: sink}
}

func (s *Service) List(ctx context.Context) ([]domain.BackupRun, error) {
	return store.ListBackupRuns(ctx, s.DB)
}

type manifest struct {
	Version  int               `json:"version"`
	CutoffAt string            `json:"cutoff_at"`
	Files    map[string]string `json:"files"`
}

func (s *Service) Run(ctx context.Context) (run domain.BackupRun, err error) {
	now := s.Clock.Now().UTC()
	run = domain.BackupRun{
		ID:        ids.NewID(),
		Status:    "running",
		CutoffAt:  now.Format(time.RFC3339),
		StartedAt: now.Format(time.RFC3339),
	}
	if err = store.CreateBackupRun(ctx, s.DB, run); err != nil {
		return run, err
	}
	defer func() {
		if err != nil {
			run.Status = "failed"
			run.Error = err.Error()
			run.CompletedAt = s.Clock.Now().UTC().Format(time.RFC3339)
			_ = store.CompleteBackupRun(context.Background(), s.DB, run)
		}
	}()

	err = s.Barrier.Snapshot(func() error {
		var e error
		run.LocalPath, run.ManifestHash, e = s.local(ctx, run)
		return e
	})
	if err != nil {
		return run, err
	}

	if s.Bucket != "" {
		if s.Sink == nil {
			err = fmt.Errorf("backup bucket configured without sink")
			return run, err
		}
		run.ObjectKey = "backups/" + filepath.Base(run.LocalPath)
		if err = s.Sink.Upload(ctx, run.LocalPath, run.ObjectKey); err != nil {
			return run, err
		}
	}

	run.Status = "succeeded"
	run.CompletedAt = s.Clock.Now().UTC().Format(time.RFC3339)
	// Detach from request ctx: local (and optional upload) already durable; a
	// canceled client must not flip a successful snapshot to failed via defer.
	if err = store.CompleteBackupRun(context.Background(), s.DB, run); err != nil {
		return run, err
	}
	return run, nil
}

func (s *Service) local(ctx context.Context, run domain.BackupRun) (string, string, error) {
	root := filepath.Join(s.DataDir, "backups", "local")
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", "", err
	}
	final := filepath.Join(root, run.ID)
	work := final + ".tmp"
	_ = os.RemoveAll(work)
	if err := os.MkdirAll(work, 0o700); err != nil {
		return "", "", err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(work)
		}
	}()

	dbPath := filepath.Join(work, "database.sqlite")
	if err := s.sqliteBackup(ctx, dbPath); err != nil {
		return "", "", err
	}

	entries := map[string]string{"database.sqlite": dbPath}
	if err := collectTree(s.DataDir, "files", entries); err != nil {
		return "", "", err
	}
	if err := collectTree(s.DataDir, "staging", entries); err != nil {
		return "", "", err
	}

	// Materialize payload files into the work directory (db already there).
	for name, src := range entries {
		if name == "database.sqlite" {
			continue
		}
		dst := filepath.Join(work, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			return "", "", err
		}
		if err := copyFile(src, dst); err != nil {
			return "", "", err
		}
		entries[name] = dst
	}

	sums := make(map[string]string, len(entries))
	names := make([]string, 0, len(entries))
	for n := range entries {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		b, err := os.ReadFile(entries[name])
		if err != nil {
			return "", "", err
		}
		sum := sha256.Sum256(b)
		sums[name] = hex.EncodeToString(sum[:])
	}

	m := manifest{Version: 1, CutoffAt: run.CutoffAt, Files: sums}
	mb, err := json.Marshal(m)
	if err != nil {
		return "", "", err
	}
	// Stable trailing newline for operators reading the file.
	if len(mb) == 0 || mb[len(mb)-1] != '\n' {
		mb = append(mb, '\n')
	}
	if err := os.WriteFile(filepath.Join(work, "manifest.json"), mb, 0o600); err != nil {
		return "", "", err
	}

	// Seal only after rename. Darwin rejects rename/RemoveAll of a directory
	// whose own write bits were cleared (Linux allows same-parent rename).
	if err := os.Rename(work, final); err != nil {
		return "", "", err
	}
	cleanup = false
	if err := sealTree(final); err != nil {
		return "", "", err
	}
	mh := sha256.Sum256(mb)
	return final, hex.EncodeToString(mh[:]), nil
}

func (s *Service) sqliteBackup(ctx context.Context, dstPath string) error {
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Raw(func(raw any) error {
		type backupAPI interface {
			NewBackup(string) (*sqlite.Backup, error)
		}
		c, ok := raw.(backupAPI)
		if !ok {
			return fmt.Errorf("sqlite connection lacks backup API")
		}
		// modernc NewBackup expects a URI; plain path works as file: path.
		b, err := c.NewBackup(dstPath)
		if err != nil {
			return err
		}
		defer b.Finish()
		for {
			more, err := b.Step(-1)
			if err != nil {
				return err
			}
			if !more {
				return nil
			}
		}
	})
}

func collectTree(dataDir, top string, entries map[string]string) error {
	root := filepath.Join(dataDir, top)
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", root)
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dataDir, path)
		if err != nil {
			return err
		}
		slash := filepath.ToSlash(rel)
		if !strings.HasPrefix(slash, top+"/") && slash != top {
			return nil
		}
		entries[slash] = path
		return nil
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func sealTree(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		mode := info.Mode()
		if d.IsDir() {
			return os.Chmod(path, mode.Perm()&^0o222|0o111)
		}
		return os.Chmod(path, mode.Perm()&^0o222)
	})
}
