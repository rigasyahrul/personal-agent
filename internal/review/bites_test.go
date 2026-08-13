package review

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

type fakeGenerator struct {
	bites []Bite
	err   error
	call  func()
}

func (f fakeGenerator) Generate(context.Context, string) ([]Bite, error) {
	if f.call != nil {
		f.call()
	}
	return f.bites, f.err
}

func fixture(t *testing.T) (*sql.DB, string, string, string, string) {
	t.Helper()
	db, dir := testutil.TempDB(t)
	body := []byte("# source\n")
	hash := fmt.Sprintf("%x", sha256.Sum256(body))
	now := "2026-08-12T09:00:00Z"
	if _, err := db.Exec(`INSERT INTO projects(id,name,created_at,updated_at) VALUES('p','P',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	root := layout.SourceDir(layout.ProjectRoot(dir, "", "p"))
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "n.md"), body, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO notes(id,project_id,relative_path,content_sha256,byte_size,status,revision,created_at,updated_at) VALUES('n','p','n.md',?,?, 'ready',3,?,?)`, hash, len(body), now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO review_pending(id,note_id,source_sha256,generator_version,status,attempts,created_at,updated_at) VALUES('g','n',?,'bites-v1','pending',0,?,?)`, hash, now, now); err != nil {
		t.Fatal(err)
	}
	return db, dir, "g", "n", hash
}

func TestBiteWorkerFailureKeepsNoteAndRetryCreatesItemsOnce(t *testing.T) {
	db, dir, generation, note, _ := fixture(t)
	now := time.Date(2026, 8, 12, 9, 0, 0, 123, time.UTC)
	w := BiteWorker{DB: db, DataDir: dir, Clock: &clock.FakeClock{T: now}, Generator: fakeGenerator{err: errors.New("provider down")}, Lease: time.Minute}
	did, err := w.LeaseAndRun(context.Background())
	if !did || err == nil || !strings.Contains(err.Error(), "provider down") {
		t.Fatalf("did=%v err=%v", did, err)
	}
	var ns, ps string
	if err := db.QueryRow(`SELECT status FROM notes WHERE id=?`, note).Scan(&ns); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT status FROM review_pending WHERE id=?`, generation).Scan(&ps); err != nil {
		t.Fatal(err)
	}
	if ns != "ready" || ps != "failed" {
		t.Fatalf("note=%s pending=%s", ns, ps)
	}
	if _, err := db.Exec(`UPDATE review_pending SET status='pending',lease_until=NULL,last_error=NULL WHERE id=? AND status='failed'`, generation); err != nil {
		t.Fatal(err)
	}
	w.Generator = fakeGenerator{bites: []Bite{{Prompt: " What? ", Answer: " Answer "}, {Prompt: "Second", Answer: "Two"}}}
	if did, err = w.LeaseAndRun(context.Background()); !did || err != nil {
		t.Fatalf("did=%v err=%v", did, err)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM review_items WHERE generation_id=?`, generation).Scan(&count); err != nil || count != 2 {
		t.Fatalf("count=%d err=%v", count, err)
	}
	if did, err = w.LeaseAndRun(context.Background()); did || err != nil {
		t.Fatalf("duplicate did=%v err=%v", did, err)
	}
}

func TestBiteWorkerLeaseTimingAndOwnership(t *testing.T) {
	t.Run("same-second expired and attempts", func(t *testing.T) {
		db, dir, _, _, _ := fixture(t)
		now := time.Date(2026, 8, 12, 9, 0, 0, 900, time.UTC)
		_, _ = db.Exec(`UPDATE review_pending SET status='leased',lease_until=?`, now.Add(-time.Nanosecond).Format(time.RFC3339Nano))
		w := BiteWorker{DB: db, DataDir: dir, Clock: &clock.FakeClock{T: now}, Generator: fakeGenerator{bites: []Bite{{"p", "a"}}}, Lease: time.Minute}
		if did, err := w.LeaseAndRun(context.Background()); !did || err != nil {
			t.Fatalf("did=%v err=%v", did, err)
		}
		var attempts int
		if err := db.QueryRow(`SELECT attempts FROM review_pending`).Scan(&attempts); err != nil || attempts != 1 {
			t.Fatalf("attempts=%d err=%v", attempts, err)
		}
	})
	t.Run("non-expired skipped", func(t *testing.T) {
		db, dir, _, _, _ := fixture(t)
		now := time.Date(2026, 8, 12, 9, 0, 0, 100, time.UTC)
		_, _ = db.Exec(`UPDATE review_pending SET status='leased',lease_until=?`, now.Add(time.Nanosecond).Format(time.RFC3339Nano))
		w := BiteWorker{DB: db, DataDir: dir, Clock: &clock.FakeClock{T: now}, Generator: fakeGenerator{bites: []Bite{{"p", "a"}}}, Lease: time.Minute}
		if did, err := w.LeaseAndRun(context.Background()); did || err != nil {
			t.Fatalf("did=%v err=%v", did, err)
		}
	})
	t.Run("old worker cannot complete stolen lease", func(t *testing.T) {
		db, dir, _, _, _ := fixture(t)
		c := &clock.FakeClock{T: time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)}
		gen := fakeGenerator{bites: []Bite{{"p", "a"}}, call: func() {
			c.Advance(2 * time.Minute)
			_, _ = db.Exec(`UPDATE review_pending SET lease_until=?,attempts=attempts+1 WHERE id='g'`, c.Now().Add(time.Minute).Format(time.RFC3339Nano))
		}}
		w := BiteWorker{DB: db, DataDir: dir, Clock: c, Generator: gen, Lease: time.Minute}
		if did, err := w.LeaseAndRun(context.Background()); !did || !errors.Is(err, ErrLeaseLost) {
			t.Fatalf("did=%v err=%v", did, err)
		}
		var count int
		_ = db.QueryRow(`SELECT count(*) FROM review_items`).Scan(&count)
		if count != 0 {
			t.Fatalf("items=%d", count)
		}
	})
}

func TestBiteWorkerExactFieldsAndHashMismatch(t *testing.T) {
	db, dir, _, _, _ := fixture(t)
	now := time.Date(2026, 8, 12, 9, 0, 0, 42, time.UTC)
	w := BiteWorker{DB: db, DataDir: dir, Clock: &clock.FakeClock{T: now}, Generator: fakeGenerator{bites: []Bite{{" first ", " one "}, {"second", "two"}}}, Lease: time.Minute}
	if _, err := w.LeaseAndRun(context.Background()); err != nil {
		t.Fatal(err)
	}
	rows, err := db.Query(`SELECT ordinal,project_id,note_id,source_revision,prompt,answer,stage,interval_days,ease_factor,reps,lapses,due_at,last_reviewed_at,row_version,status,scheduler_version FROM review_items ORDER BY ordinal`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for want := 0; rows.Next(); want++ {
		var ordinal, revision, stage, reps, lapses, rowVersion int
		var project, note, prompt, answer, due, status, version string
		var interval, ease float64
		var last sql.NullString
		if err := rows.Scan(&ordinal, &project, &note, &revision, &prompt, &answer, &stage, &interval, &ease, &reps, &lapses, &due, &last, &rowVersion, &status, &version); err != nil {
			t.Fatal(err)
		}
		if ordinal != want || project != "p" || note != "n" || revision != 3 || stage != 0 || interval != 0 || ease != 2.5 || reps != 0 || lapses != 0 || due != now.Format(time.RFC3339Nano) || last.Valid || rowVersion != 0 || status != "active" || version != "sm2-lite-v1" {
			t.Fatalf("bad row ordinal=%d prompt=%q answer=%q", ordinal, prompt, answer)
		}
	}
	db2, dir2, _, _, _ := fixture(t)
	_, _ = db2.Exec(`UPDATE review_pending SET source_sha256='bad'`)
	w.DB = db2
	w.DataDir = dir2
	if did, err := w.LeaseAndRun(context.Background()); !did || err == nil {
		t.Fatalf("did=%v err=%v", did, err)
	}
	var status string
	_ = db2.QueryRow(`SELECT status FROM review_pending`).Scan(&status)
	if status != "failed" {
		t.Fatalf("status=%s", status)
	}
}
