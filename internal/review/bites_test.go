package review_test

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
	"github.com/rigasyahrul/personal-agent/internal/review"
	"github.com/rigasyahrul/personal-agent/internal/store"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

type fakeGenerator struct {
	bites []review.Bite
	err   error
	call  func()
}

func (f fakeGenerator) Generate(context.Context, string) ([]review.Bite, error) {
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
	w := review.BiteWorker{DB: db, DataDir: dir, Clock: &clock.FakeClock{T: now}, Generator: fakeGenerator{err: errors.New("provider down")}, Lease: time.Minute}
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
	if err := store.RetryReviewPending(context.Background(), db, generation); err != nil {
		t.Fatal(err)
	}
	w.Generator = fakeGenerator{bites: []review.Bite{{Prompt: " What? ", Answer: " Answer "}, {Prompt: "Second", Answer: "Two"}}}
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

func TestBiteWorkerLeasesOldestCreatedJobFirst(t *testing.T) {
	db, dir, _, _, hash := fixture(t)
	if err := os.WriteFile(filepath.Join(layout.SourceDir(layout.ProjectRoot(dir, "", "p")), "n2.md"), []byte("# source\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO notes(id,project_id,relative_path,content_sha256,byte_size,status,revision,created_at,updated_at)
		SELECT 'n2',project_id,'n2.md',content_sha256,byte_size,status,revision,created_at,updated_at FROM notes WHERE id='n';
		UPDATE review_pending SET id='z-old',created_at='2026-08-12T08:00:00Z' WHERE id='g';
		INSERT INTO review_pending(id,note_id,source_sha256,generator_version,status,attempts,created_at,updated_at)
		VALUES('a-new','n2',?,'bites-v1','pending',0,'2026-08-12T10:00:00Z','2026-08-12T10:00:00Z')`, hash); err != nil {
		t.Fatal(err)
	}
	w := review.BiteWorker{DB: db, DataDir: dir, Clock: &clock.FakeClock{T: time.Date(2026, 8, 12, 11, 0, 0, 0, time.UTC)}, Generator: fakeGenerator{bites: []review.Bite{{"p", "a"}}}, Lease: time.Minute}
	if did, err := w.LeaseAndRun(context.Background()); !did || err != nil {
		t.Fatalf("did=%v err=%v", did, err)
	}
	var oldStatus, newStatus string
	if err := db.QueryRow(`SELECT status FROM review_pending WHERE id='z-old'`).Scan(&oldStatus); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT status FROM review_pending WHERE id='a-new'`).Scan(&newStatus); err != nil {
		t.Fatal(err)
	}
	if oldStatus != "completed" || newStatus != "pending" {
		t.Fatalf("old=%s new=%s", oldStatus, newStatus)
	}
}

func TestBiteWorkerConflictingOrdinalRollsBackCompletion(t *testing.T) {
	db, dir, generation, note, hash := fixture(t)
	if _, err := db.Exec(`INSERT INTO review_items(id,project_id,note_id,kind,source_sha256,source_revision,prompt,answer,generation_id,ordinal,stage,due_at,interval_days,ease_factor,reps,lapses,row_version,status,scheduler_version) VALUES('existing','p',?,'bite',?,3,'different','content',?,1,0,'2026-08-12T09:00:00Z',0,2.5,0,0,0,'active','sm2-lite-v1')`, note, hash, generation); err != nil {
		t.Fatal(err)
	}
	w := review.BiteWorker{DB: db, DataDir: dir, Clock: &clock.FakeClock{T: time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)}, Generator: fakeGenerator{bites: []review.Bite{{"first", "one"}, {"second", "two"}}}, Lease: time.Minute}
	if did, err := w.LeaseAndRun(context.Background()); !did || err == nil {
		t.Fatalf("did=%v err=%v", did, err)
	}
	var count int
	var status string
	if err := db.QueryRow(`SELECT count(*) FROM review_items WHERE generation_id=?`, generation).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT status FROM review_pending WHERE id=?`, generation).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if count != 1 || status != "leased" {
		t.Fatalf("items=%d status=%s", count, status)
	}
}

func TestBiteWorkerLeaseTimingAndOwnership(t *testing.T) {
	t.Run("same-second expired and attempts", func(t *testing.T) {
		db, dir, _, _, _ := fixture(t)
		now := time.Date(2026, 8, 12, 9, 0, 0, 900, time.UTC)
		_, _ = db.Exec(`UPDATE review_pending SET status='leased',lease_until=?`, now.Add(-time.Nanosecond).Format(time.RFC3339Nano))
		w := review.BiteWorker{DB: db, DataDir: dir, Clock: &clock.FakeClock{T: now}, Generator: fakeGenerator{bites: []review.Bite{{"p", "a"}}}, Lease: time.Minute}
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
		w := review.BiteWorker{DB: db, DataDir: dir, Clock: &clock.FakeClock{T: now}, Generator: fakeGenerator{bites: []review.Bite{{"p", "a"}}}, Lease: time.Minute}
		if did, err := w.LeaseAndRun(context.Background()); did || err != nil {
			t.Fatalf("did=%v err=%v", did, err)
		}
	})
	t.Run("old worker cannot complete stolen lease", func(t *testing.T) {
		db, dir, _, _, _ := fixture(t)
		c := &clock.FakeClock{T: time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)}
		gen := fakeGenerator{bites: []review.Bite{{"p", "a"}}, call: func() {
			c.Advance(2 * time.Minute)
			_, _ = db.Exec(`UPDATE review_pending SET lease_until=?,attempts=attempts+1 WHERE id='g'`, c.Now().Add(time.Minute).Format(time.RFC3339Nano))
		}}
		w := review.BiteWorker{DB: db, DataDir: dir, Clock: c, Generator: gen, Lease: time.Minute}
		if did, err := w.LeaseAndRun(context.Background()); !did || !errors.Is(err, review.ErrLeaseLost) {
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
	w := review.BiteWorker{DB: db, DataDir: dir, Clock: &clock.FakeClock{T: now}, Generator: fakeGenerator{bites: []review.Bite{{" first ", " one "}, {"second", "two"}}}, Lease: time.Minute}
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
