package publish_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/publish"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func fixture(t *testing.T) (string, *sql.DB, *clock.FakeClock) {
	t.Helper()
	db, d := testutil.TempDB(t)
	now := time.Date(2026, 8, 13, 1, 2, 3, 0, time.UTC)
	_, err := db.Exec(`INSERT INTO vaults(id,name,created_at,updated_at) VALUES('v1','V','x','x'); INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES('p1','v1','P','x','x')`)
	if err != nil {
		t.Fatal(err)
	}
	if err = layout.EnsureProjectDirs(d, "v1", "p1"); err != nil {
		t.Fatal(err)
	}
	return d, db, &clock.FakeClock{T: now}
}

func TestConcurrentSameKeyRetriesConverge(t *testing.T) {
	d, db, c := fixture(t)
	m := publish.Machine{DB: db, DataDir: d, Clock: c}
	const callers = 16
	start := make(chan struct{})
	type result struct {
		status, note string
		err          error
	}
	results := make(chan result, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			in := input()
			in.ReviewMode = "whole"
			s, n, err := m.Run(context.Background(), in)
			results <- result{s, n, err}
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	for got := range results {
		if got.err != nil || got.status != "completed" || got.note != "n1" {
			t.Errorf("result=%+v", got)
		}
	}
	for table, where := range map[string]string{"direct_ops": "request_key='key1'", "notes": "id='n1'"} {
		var n int
		if err := db.QueryRow("SELECT count(*) FROM " + table + " WHERE " + where).Scan(&n); err != nil || n != 1 {
			t.Fatalf("%s=%d err=%v", table, n, err)
		}
	}
	var reviews int
	if err := db.QueryRow(`SELECT count(*) FROM review_items WHERE note_id='n1'`).Scan(&reviews); err != nil || reviews != 1 {
		t.Fatalf("reviews=%d err=%v", reviews, err)
	}
	b, err := os.ReadFile(filepath.Join(layout.SourceDir(layout.ProjectRoot(d, "v1", "p1")), "guide", "one.md"))
	if err != nil || string(b) != "# One\n" {
		t.Fatalf("body=%q err=%v", b, err)
	}
}

func TestWholeReviewItemStartsWithExactDurableState(t *testing.T) {
	d, db, c := fixture(t)
	in := input()
	in.ReviewMode = domain.ReviewMode("whole")
	if _, _, err := (&publish.Machine{DB: db, DataDir: d, Clock: c}).Run(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	var kind, sha, prompt, due, status, scheduler string
	var revision, stage, reps, lapses int
	var rowVersion int64
	var interval, ease float64
	var lastReviewed sql.NullString
	err := db.QueryRow(`SELECT kind,source_sha256,source_revision,prompt,stage,interval_days,ease_factor,reps,lapses,due_at,last_reviewed_at,row_version,status,scheduler_version FROM review_items WHERE note_id=?`, in.NoteID).
		Scan(&kind, &sha, &revision, &prompt, &stage, &interval, &ease, &reps, &lapses, &due, &lastReviewed, &rowVersion, &status, &scheduler)
	if err != nil {
		t.Fatal(err)
	}
	wantDue := c.Now().UTC().Format(time.RFC3339Nano)
	if kind != "whole" || sha == "" || revision != 1 || prompt != "Review this note" || stage != 0 || interval != 0 || ease != 2.5 || reps != 0 || lapses != 0 || due != wantDue || lastReviewed.Valid || rowVersion != 0 || status != "active" || scheduler != "sm2-lite-v1" {
		t.Fatalf("whole state: kind=%q sha=%q revision=%d prompt=%q stage=%d interval=%v ease=%v reps=%d lapses=%d due=%q last=%v version=%d status=%q scheduler=%q", kind, sha, revision, prompt, stage, interval, ease, reps, lapses, due, lastReviewed, rowVersion, status, scheduler)
	}
}

func TestWholeReviewConflictMustMatchExactReusableRow(t *testing.T) {
	d, db, c := completedFixture(t, "whole")
	if _, err := db.Exec(`UPDATE review_items SET scheduler_version='other' WHERE note_id='n1'; UPDATE direct_ops SET status='finalized' WHERE id='op1'`); err != nil {
		t.Fatal(err)
	}
	err := (&publish.Machine{DB: db, DataDir: d, Clock: c}).RecoverAll(context.Background())
	if err == nil {
		t.Fatal("mismatched active whole row was reused")
	}
	assertOpStatus(t, db, "op1", "finalized")
}
func input() publish.PublishInput {
	return publish.PublishInput{OpID: "op1", RequestKey: "key1", RequestFingerprint: "fp1", Kind: "direct", Body: []byte("# One\n"), TargetProjectID: "p1", TargetRelPath: "guide/one.md", ReviewMode: domain.ReviewMode("none"), NoteID: "n1"}
}

func TestDirectCreateIsIdempotentAndNeverOverwrites(t *testing.T) {
	d, db, c := fixture(t)
	m := publish.Machine{DB: db, DataDir: d, Clock: c}
	in := input()
	status, note, err := m.Run(context.Background(), in)
	if err != nil || status != "completed" || note != "n1" {
		t.Fatalf("%s %s %v", status, note, err)
	}
	status, note, err = m.Run(context.Background(), in)
	if err != nil || status != "completed" || note != "n1" {
		t.Fatalf("retry %s %s %v", status, note, err)
	}
	different := in
	different.OpID = "op2"
	different.NoteID = "n2"
	different.RequestFingerprint = "other"
	if _, _, err = m.Run(context.Background(), different); !errors.Is(err, publish.ErrConflict) {
		t.Fatalf("fingerprint: %v", err)
	}
	collision := in
	collision.OpID = "op3"
	collision.NoteID = "n3"
	collision.RequestKey = "key3"
	collision.RequestFingerprint = "fp3"
	if _, _, err = m.Run(context.Background(), collision); !errors.Is(err, publish.ErrConflict) {
		t.Fatalf("collision: %v", err)
	}
	b, _ := os.ReadFile(filepath.Join(layout.SourceDir(layout.ProjectRoot(d, "v1", "p1")), "guide", "one.md"))
	if string(b) != "# One\n" {
		t.Fatalf("bytes %q", b)
	}
}

func TestDirectValidation(t *testing.T) {
	d, db, c := fixture(t)
	m := publish.Machine{DB: db, DataDir: d, Clock: c}
	var cases []publish.PublishInput
	for _, p := range []string{"../x.md", "x.MD", "x.txt", "memory/x.md", "soul/x.md"} {
		in := input()
		in.OpID += p
		in.RequestKey += p
		in.RequestFingerprint += p
		in.NoteID += p
		in.TargetRelPath = p
		cases = append(cases, in)
	}
	big := input()
	big.Body = []byte(strings.Repeat("x", (1<<20)+1))
	big.OpID = "big"
	big.RequestKey = "big"
	big.RequestFingerprint = "big"
	big.NoteID = "big"
	cases = append(cases, big)
	for _, in := range cases {
		if _, _, err := m.Run(context.Background(), in); !errors.Is(err, publish.ErrInvalid) {
			t.Errorf("accepted %#v: %v", in, err)
		}
	}
}

func promoteFixture(t *testing.T) (string, *sql.DB, *clock.FakeClock, publish.PublishInput) {
	t.Helper()
	d, db, c := fixture(t)
	_, err := db.Exec(`INSERT INTO sessions(id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at) VALUES('s1','project','v1','p1','active','p','m','{}','{}','S','x','x')`)
	if err != nil {
		t.Fatal(err)
	}
	workspace := layout.SessionWorkspace(d, "project", "v1", "p1", "s1")
	if err = os.MkdirAll(workspace, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(workspace, "draft.md"), []byte("# Frozen\n"), 0600); err != nil {
		t.Fatal(err)
	}
	in := publish.PublishInput{OpID: "promote-op", RequestKey: "promote-key", RequestFingerprint: "promote-fp", Kind: "promote", SessionID: "s1", WorkspacePath: "draft.md", TargetProjectID: "p1", TargetRelPath: "guides/frozen.md", ReviewMode: "whole", NoteID: "promote-note"}
	return d, db, c, in
}

func TestPromotePublishesOnceAndRejectsChangedFingerprint(t *testing.T) {
	d, db, c, in := promoteFixture(t)
	m := publish.Machine{DB: db, DataDir: d, Clock: c}
	status, note, err := m.Run(context.Background(), in)
	if err != nil || status != "completed" || note != in.NoteID {
		t.Fatalf("first run=(%q,%q,%v)", status, note, err)
	}
	body, err := os.ReadFile(filepath.Join(layout.SourceDir(layout.ProjectRoot(d, "v1", "p1")), "guides", "frozen.md"))
	if err != nil || string(body) != "# Frozen\n" {
		t.Fatalf("body=%q err=%v", body, err)
	}
	status, note, err = m.Run(context.Background(), in)
	if err != nil || status != "completed" || note != in.NoteID {
		t.Fatalf("retry=(%q,%q,%v)", status, note, err)
	}
	for table, where := range map[string]string{"promote_ops": "id='promote-op'", "notes": "id='promote-note'", "review_items": "note_id='promote-note'"} {
		var count int
		if err = db.QueryRow("SELECT count(*) FROM " + table + " WHERE " + where).Scan(&count); err != nil || count != 1 {
			t.Fatalf("%s count=%d err=%v", table, count, err)
		}
	}
	changed := in
	changed.RequestFingerprint = "changed"
	_, _, err = m.Run(context.Background(), changed)
	var conflict *publish.ConflictError
	if !errors.As(err, &conflict) || conflict.Code != "idempotency_key_reused" {
		t.Fatalf("changed fingerprint error=%v", err)
	}
}

func TestConcurrentPromoteSameKeyRetriesAcrossMachinesConverge(t *testing.T) {
	d, db, c, base := promoteFixture(t)
	const callers = 16
	start := make(chan struct{})
	type result struct {
		status, note string
		err          error
	}
	results := make(chan result, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			in := base
			in.OpID = fmt.Sprintf("promote-op-%d", i)
			in.NoteID = fmt.Sprintf("promote-note-%d", i)
			m := publish.Machine{DB: db, DataDir: d, Clock: c}
			<-start
			status, note, err := m.Run(context.Background(), in)
			results <- result{status: status, note: note, err: err}
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)

	var originalNote string
	for got := range results {
		if got.err != nil || got.status != "completed" {
			t.Errorf("result=%+v", got)
		}
		if originalNote == "" {
			originalNote = got.note
		} else if got.note != originalNote {
			t.Errorf("note=%q, want original %q", got.note, originalNote)
		}
	}
	for table, where := range map[string]string{"promote_ops": "request_key='promote-key'", "notes": "relative_path='guides/frozen.md'", "review_items": "note_id='" + originalNote + "'"} {
		var count int
		if err := db.QueryRow("SELECT count(*) FROM " + table + " WHERE " + where).Scan(&count); err != nil || count != 1 {
			t.Errorf("%s count=%d err=%v", table, count, err)
		}
	}
}

func TestPromoteRejectsAnotherProjectAndExistingDestination(t *testing.T) {
	d, db, c, in := promoteFixture(t)
	m := publish.Machine{DB: db, DataDir: d, Clock: c}
	_, err := db.Exec(`INSERT INTO projects(id,name,created_at,updated_at) VALUES('p2','Other','x','x')`)
	if err != nil {
		t.Fatal(err)
	}
	other := in
	other.TargetProjectID = "p2"
	if _, _, err = m.Run(context.Background(), other); err == nil || !strings.Contains(err.Error(), "session project is the only promote target") {
		t.Fatalf("cross-project error=%v", err)
	}
	destination := filepath.Join(layout.SourceDir(layout.ProjectRoot(d, "v1", "p1")), "guides", "frozen.md")
	if err = os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(destination, []byte("keep me"), 0600); err != nil {
		t.Fatal(err)
	}
	_, _, err = m.Run(context.Background(), in)
	var conflict *publish.ConflictError
	if !errors.As(err, &conflict) || conflict.Code != "destination_exists" {
		t.Fatalf("destination error=%v", err)
	}
	if body, readErr := os.ReadFile(destination); readErr != nil || string(body) != "keep me" {
		t.Fatalf("destination=%q err=%v", body, readErr)
	}
}

func TestReviewModesAndRecoveryDeduplicate(t *testing.T) {
	for _, mode := range []domain.ReviewMode{"whole", "bites"} {
		t.Run(string(mode), func(t *testing.T) {
			d, db, c := fixture(t)
			m := publish.Machine{DB: db, DataDir: d, Clock: c}
			in := input()
			in.ReviewMode = mode
			if _, _, err := m.Run(context.Background(), in); err != nil {
				t.Fatal(err)
			}
			// Replay both durable boundaries rather than recovering a completed no-op.
			if _, err := db.Exec(`UPDATE direct_ops SET status='finalized' WHERE id='op1'`); err != nil {
				t.Fatal(err)
			}
			if mode == "bites" {
				if _, err := db.Exec(`UPDATE review_pending SET status='completed' WHERE note_id='n1'`); err != nil {
					t.Fatal(err)
				}
				if _, err := db.Exec(`UPDATE direct_ops SET status='finalized' WHERE id='op1'`); err != nil {
					t.Fatal(err)
				}
				if err := m.RecoverAll(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			if err := m.RecoverAll(context.Background()); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`UPDATE direct_ops SET status='review_enqueued' WHERE id='op1'`); err != nil {
				t.Fatal(err)
			}
			if err := m.RecoverAll(context.Background()); err != nil {
				t.Fatal(err)
			}
			table := "review_items"
			if mode == "bites" {
				table = "review_pending"
			}
			var n int
			if err := db.QueryRow("SELECT count(*) FROM " + table + " WHERE note_id='n1'").Scan(&n); err != nil || n != 1 {
				t.Fatalf("count=%d err=%v", n, err)
			}
			var status string
			if err := db.QueryRow(`SELECT status FROM direct_ops WHERE id='op1'`).Scan(&status); err != nil || status != "completed" {
				t.Fatalf("status=%q err=%v", status, err)
			}
		})
	}
}

func TestRecoverAfterPublishedFileBeforeFinalization(t *testing.T) {
	d, db, c := fixture(t)
	m := publish.Machine{DB: db, DataDir: d, Clock: c}
	if _, _, err := m.Run(context.Background(), input()); err != nil {
		t.Fatal(err)
	}
	_, _ = db.Exec(`UPDATE notes SET status='pending',content_sha256=NULL,byte_size=NULL,revision=0 WHERE id='n1'; UPDATE direct_ops SET status='published_fs' WHERE id='op1'`)
	if err := m.RecoverAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	var ns, os string
	if err := db.QueryRow(`SELECT n.status,o.status FROM notes n JOIN direct_ops o ON o.note_id=n.id WHERE n.id='n1'`).Scan(&ns, &os); err != nil || ns != "ready" || os != "completed" {
		t.Fatalf("%s %s %v", ns, os, err)
	}
}

func preparePromoteRecovery(t *testing.T, status string, mode domain.ReviewMode) (string, *sql.DB, publish.Machine, publish.PublishInput, string) {
	t.Helper()
	d, db, c, in := promoteFixture(t)
	in.ReviewMode = mode
	m := publish.Machine{DB: db, DataDir: d, Clock: c}
	if _, _, err := m.Run(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(layout.SourceDir(layout.ProjectRoot(d, "v1", "p1")), filepath.FromSlash(in.TargetRelPath))
	if status != "review_enqueued" {
		if _, err := db.Exec(`DELETE FROM review_items WHERE note_id=?; DELETE FROM review_pending WHERE note_id=?`, in.NoteID, in.NoteID); err != nil {
			t.Fatal(err)
		}
	}
	switch status {
	case "accepted", "frozen":
		if _, err := db.Exec(`DELETE FROM notes WHERE id=?`, in.NoteID); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(destination); err != nil {
			t.Fatal(err)
		}
	case "path_reserved", "published_fs":
		if _, err := db.Exec(`UPDATE notes SET status='pending',content_sha256=NULL,byte_size=NULL,revision=0 WHERE id=?`, in.NoteID); err != nil {
			t.Fatal(err)
		}
		if status == "path_reserved" {
			if err := os.Remove(destination); err != nil {
				t.Fatal(err)
			}
		}
	case "finalized", "review_enqueued":
		// The completed baseline already has the required ready Note and file.
	default:
		t.Fatalf("unsupported status %q", status)
	}
	if status != "accepted" {
		workspace := filepath.Join(layout.SessionWorkspace(d, "project", "v1", "p1", "s1"), "draft.md")
		if err := os.WriteFile(workspace, []byte("changed workspace"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`UPDATE promote_ops SET status=? WHERE id=?`, status, in.OpID); err != nil {
		t.Fatal(err)
	}
	return d, db, m, in, destination
}

func TestRecoverAllPromoteCrashStateMatrix(t *testing.T) {
	for _, status := range []string{"accepted", "frozen", "path_reserved", "published_fs", "finalized", "review_enqueued"} {
		t.Run(status, func(t *testing.T) {
			_, db, m, in, destination := preparePromoteRecovery(t, status, "whole")
			if err := m.RecoverAll(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err := m.RecoverAll(context.Background()); err != nil {
				t.Fatal(err)
			}
			var opStatus, noteStatus string
			if err := db.QueryRow(`SELECT o.status,n.status FROM promote_ops o JOIN notes n ON n.id=o.note_id WHERE o.id=?`, in.OpID).Scan(&opStatus, &noteStatus); err != nil {
				t.Fatal(err)
			}
			if opStatus != "completed" || noteStatus != "ready" {
				t.Fatalf("op=%q note=%q", opStatus, noteStatus)
			}
			body, err := os.ReadFile(destination)
			if err != nil || string(body) != "# Frozen\n" {
				t.Fatalf("destination=%q err=%v", body, err)
			}
			var reviews int
			if err := db.QueryRow(`SELECT count(*) FROM review_items WHERE note_id=? AND kind='whole'`, in.NoteID).Scan(&reviews); err != nil || reviews != 1 {
				t.Fatalf("whole reviews=%d err=%v", reviews, err)
			}
		})
	}
}

func TestRecoverAllPromoteBitesDeduplicatesPendingGeneration(t *testing.T) {
	for _, status := range []string{"finalized", "review_enqueued"} {
		t.Run(status, func(t *testing.T) {
			_, db, m, in, _ := preparePromoteRecovery(t, status, "bites")
			if err := m.RecoverAll(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err := m.RecoverAll(context.Background()); err != nil {
				t.Fatal(err)
			}
			var opStatus, noteStatus string
			if err := db.QueryRow(`SELECT o.status,n.status FROM promote_ops o JOIN notes n ON n.id=o.note_id WHERE o.id=?`, in.OpID).Scan(&opStatus, &noteStatus); err != nil || opStatus != "completed" || noteStatus != "ready" {
				t.Fatalf("op=%q note=%q err=%v", opStatus, noteStatus, err)
			}
			var pending int
			if err := db.QueryRow(`SELECT count(*) FROM review_pending WHERE note_id=? AND generator_version='bites-v1' AND status='pending'`, in.NoteID).Scan(&pending); err != nil || pending != 1 {
				t.Fatalf("pending generations=%d err=%v", pending, err)
			}
		})
	}
}

func TestRecoverAllPromoteRejectsInvalidReviewEnqueuedState(t *testing.T) {
	for _, tc := range []struct {
		name   string
		update string
	}{
		{name: "wrong_project_id", update: `UPDATE review_items SET project_id='p2' WHERE note_id=?`},
		{name: "wrong_source_revision", update: `UPDATE review_items SET source_revision=2 WHERE note_id=?`},
		{name: "non_active_status", update: `UPDATE review_items SET status='suspended' WHERE note_id=?`},
		{name: "wrong_scheduler_version", update: `UPDATE review_items SET scheduler_version='other' WHERE note_id=?`},
	} {
		t.Run("whole/"+tc.name, func(t *testing.T) {
			_, db, m, in, _ := preparePromoteRecovery(t, "review_enqueued", "whole")
			if _, err := db.Exec(`INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES('p2','v1','Other','x','x')`); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(tc.update, in.NoteID); err != nil {
				t.Fatal(err)
			}
			if err := m.RecoverAll(context.Background()); err == nil {
				t.Fatal("expected malformed whole review state to fail recovery")
			}
			var status string
			if err := db.QueryRow(`SELECT status FROM promote_ops WHERE id=?`, in.OpID).Scan(&status); err != nil || status != "review_enqueued" {
				t.Fatalf("status=%q err=%v", status, err)
			}
		})
	}
	for _, mode := range []domain.ReviewMode{"whole", "bites"} {
		t.Run(string(mode)+"/missing", func(t *testing.T) {
			_, db, m, in, _ := preparePromoteRecovery(t, "review_enqueued", mode)
			if _, err := db.Exec(`DELETE FROM review_items WHERE note_id=?; DELETE FROM review_pending WHERE note_id=?`, in.NoteID, in.NoteID); err != nil {
				t.Fatal(err)
			}
			if err := m.RecoverAll(context.Background()); err == nil {
				t.Fatal("expected invalid review_enqueued state to fail recovery")
			}
			var status string
			if err := db.QueryRow(`SELECT status FROM promote_ops WHERE id=?`, in.OpID).Scan(&status); err != nil || status != "review_enqueued" {
				t.Fatalf("status=%q err=%v", status, err)
			}
		})
	}
}

func TestRecoverAllPromotePublishedDestinationReconciliation(t *testing.T) {
	for _, status := range []string{"published_fs", "finalized", "review_enqueued"} {
		for _, damage := range []string{"missing", "hash_mismatch", "size_mismatch"} {
			t.Run(status+"/"+damage, func(t *testing.T) {
				_, db, m, in, destination := preparePromoteRecovery(t, status, "whole")
				switch damage {
				case "missing":
					if err := os.Remove(destination); err != nil {
						t.Fatal(err)
					}
				case "hash_mismatch":
					if err := os.WriteFile(destination, []byte("# Change\n"), 0600); err != nil {
						t.Fatal(err)
					}
				case "size_mismatch":
					if err := os.WriteFile(destination, []byte("tampered and longer"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				if err := m.RecoverAll(context.Background()); err == nil {
					t.Fatal("expected reconciliation failure")
				}
				var got string
				if err := db.QueryRow(`SELECT status FROM promote_ops WHERE id=?`, in.OpID).Scan(&got); err != nil || got != status {
					t.Fatalf("status=%q want=%q err=%v", got, status, err)
				}
			})
		}
	}
}

func TestAcceptedRecoveryMissingStageFailsAndContinues(t *testing.T) {
	d, db, c := fixture(t)
	now := c.Now().Format(time.RFC3339Nano)
	for _, id := range []string{"missing", "next"} {
		_, err := db.Exec(`INSERT INTO direct_ops(id,request_key,request_fingerprint,target_project_id,target_relative_path,review_mode,note_id,status,created_at,updated_at) VALUES(?,?,?,?,?,'none',?,'accepted',?,?)`, id, id, id, "p1", id+".md", "n"+id, now, now)
		if err != nil {
			t.Fatal(err)
		}
	}
	stage := filepath.Join(d, "staging", "direct", "next")
	if err := os.MkdirAll(stage, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "body.md"), []byte("next"), 0600); err != nil {
		t.Fatal(err)
	}
	m := publish.Machine{DB: db, DataDir: d, Clock: c}
	if err := m.RecoverAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	var failedStatus, failedError, nextStatus string
	if err := db.QueryRow(`SELECT status,error FROM direct_ops WHERE id='missing'`).Scan(&failedStatus, &failedError); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT status FROM direct_ops WHERE id='next'`).Scan(&nextStatus); err != nil {
		t.Fatal(err)
	}
	if failedStatus != "failed" || failedError == "" || nextStatus != "completed" {
		t.Fatalf("missing=(%s,%q) next=%s", failedStatus, failedError, nextStatus)
	}
}

func TestRecoveryStateMatrix(t *testing.T) {
	t.Run("accepted valid stage and frozen valid immutable stage complete", func(t *testing.T) {
		for _, state := range []string{"accepted", "frozen"} {
			t.Run(state, func(t *testing.T) {
				d, db, c := fixture(t)
				now := c.Now().Format(time.RFC3339Nano)
				sha := ""
				size := any(nil)
				if state == "frozen" {
					sha = "230d8358dc8e8890b4c58deeb62912ee2f20357ae92a5cc861b98e68fe31acb5"
					size = 4
				}
				_, err := db.Exec(`INSERT INTO direct_ops(id,request_key,request_fingerprint,target_project_id,target_relative_path,review_mode,note_id,frozen_sha256,frozen_size,status,created_at,updated_at) VALUES('op1','key1','fp1','p1','one.md','none','n1',?,?,?, ?,?)`, sha, size, state, now, now)
				if err != nil {
					t.Fatal(err)
				}
				stage := filepath.Join(d, "staging", "direct", "op1")
				if err = os.MkdirAll(stage, 0700); err != nil {
					t.Fatal(err)
				}
				if err = os.WriteFile(filepath.Join(stage, "body.md"), []byte("body"), 0600); err != nil {
					t.Fatal(err)
				}
				m := publish.Machine{DB: db, DataDir: d, Clock: c}
				if err = m.RecoverAll(context.Background()); err != nil {
					t.Fatal(err)
				}
				assertOpStatus(t, db, "op1", "completed")
			})
		}
	})

	t.Run("path reserved matching destination without stage completes", func(t *testing.T) {
		d, db, c := completedFixture(t, "none")
		if _, err := db.Exec(`UPDATE notes SET status='pending',content_sha256=NULL,byte_size=NULL,revision=0; UPDATE direct_ops SET status='path_reserved'`); err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(filepath.Join(d, "staging")); err != nil {
			t.Fatal(err)
		}
		if err := (&publish.Machine{DB: db, DataDir: d, Clock: c}).RecoverAll(context.Background()); err != nil {
			t.Fatal(err)
		}
		assertOpStatus(t, db, "op1", "completed")
	})

	for _, state := range []string{"path_reserved", "published_fs"} {
		for _, artifact := range []string{"missing", "mismatch", "symlink", "symlink_parent", "directory", "fifo"} {
			if state == "path_reserved" && artifact == "missing" {
				continue
			}
			t.Run(state+"_"+artifact+" fails artifact and continues", func(t *testing.T) {
				d, db, c := completedFixture(t, "none")
				dst := filepath.Join(layout.SourceDir(layout.ProjectRoot(d, "v1", "p1")), "guide", "one.md")
				if err := os.Remove(dst); err != nil {
					t.Fatal(err)
				}
				switch artifact {
				case "mismatch":
					if err := os.WriteFile(dst, []byte("original"), 0600); err != nil {
						t.Fatal(err)
					}
				case "symlink":
					if err := os.Symlink("elsewhere", dst); err != nil {
						t.Fatal(err)
					}
				case "symlink_parent":
					if err := os.Remove(filepath.Dir(dst)); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink("elsewhere", filepath.Dir(dst)); err != nil {
						t.Fatal(err)
					}
				case "directory":
					if err := os.Mkdir(dst, 0700); err != nil {
						t.Fatal(err)
					}
				case "fifo":
					if err := syscall.Mkfifo(dst, 0600); err != nil {
						t.Fatal(err)
					}
				}
				if _, err := db.Exec(`UPDATE notes SET status='pending',content_sha256=NULL,byte_size=NULL,revision=0 WHERE id='n1'; UPDATE direct_ops SET status=? WHERE id='op1'`, state); err != nil {
					t.Fatal(err)
				}
				seedAccepted(t, db, d, c, "next")
				m := publish.Machine{DB: db, DataDir: d, Clock: c}
				if err := m.RecoverAll(context.Background()); err != nil {
					t.Fatalf("startup wedged: %v", err)
				}
				assertOpStatus(t, db, "op1", "failed")
				assertOpStatus(t, db, "next", "completed")
				var noteStatus string
				if err := db.QueryRow(`SELECT status FROM notes WHERE id='n1'`).Scan(&noteStatus); err != nil || noteStatus != "failed" {
					t.Fatalf("note=%q err=%v", noteStatus, err)
				}
				if artifact == "mismatch" {
					b, err := os.ReadFile(dst)
					if err != nil || string(b) != "original" {
						t.Fatalf("artifact changed: %q %v", b, err)
					}
				}
			})
		}
	}

	t.Run("published fs valid completes and terminal failed is skipped", func(t *testing.T) {
		d, db, c := completedFixture(t, "none")
		if _, err := db.Exec(`UPDATE notes SET status='pending',content_sha256=NULL,byte_size=NULL,revision=0; UPDATE direct_ops SET status='published_fs'`); err != nil {
			t.Fatal(err)
		}
		if err := (&publish.Machine{DB: db, DataDir: d, Clock: c}).RecoverAll(context.Background()); err != nil {
			t.Fatal(err)
		}
		assertOpStatus(t, db, "op1", "completed")
		if _, err := db.Exec(`UPDATE direct_ops SET status='failed',error='keep me' WHERE id='op1'`); err != nil {
			t.Fatal(err)
		}
		if err := (&publish.Machine{DB: db, DataDir: d, Clock: c}).RecoverAll(context.Background()); err != nil {
			t.Fatal(err)
		}
		var got string
		if err := db.QueryRow(`SELECT error FROM direct_ops WHERE id='op1'`).Scan(&got); err != nil || got != "keep me" {
			t.Fatalf("error=%q err=%v", got, err)
		}
	})
}

func TestAcceptedRecoveryUnsafeStagingFailsAndContinues(t *testing.T) {
	for _, unsafe := range []string{"component", "final"} {
		t.Run(unsafe, func(t *testing.T) {
			d, db, c := fixture(t)
			seedAccepted(t, db, d, c, "bad")
			seedAccepted(t, db, d, c, "next")
			bad := filepath.Join(d, "staging", "direct", "bad")
			if err := os.Remove(filepath.Join(bad, "body.md")); err != nil {
				t.Fatal(err)
			}
			if unsafe == "final" {
				if err := os.Symlink("target", filepath.Join(bad, "body.md")); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.RemoveAll(bad); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(d, "elsewhere"), bad); err != nil {
					t.Fatal(err)
				}
			}
			if err := (&publish.Machine{DB: db, DataDir: d, Clock: c}).RecoverAll(context.Background()); err != nil {
				t.Fatal(err)
			}
			assertOpStatus(t, db, "bad", "failed")
			assertOpStatus(t, db, "next", "completed")
		})
	}
}

func TestDatabaseFailureIsNotConflictOrTerminalized(t *testing.T) {
	d, db, c := fixture(t)
	seedAccepted(t, db, d, c, "blocked")
	if _, err := db.Exec(`CREATE TRIGGER fail_direct_update BEFORE UPDATE ON direct_ops BEGIN SELECT RAISE(ABORT,'injected database failure'); END`); err != nil {
		t.Fatal(err)
	}
	m := publish.Machine{DB: db, DataDir: d, Clock: c}
	err := m.RecoverAll(context.Background())
	if err == nil || errors.Is(err, publish.ErrConflict) {
		t.Fatalf("got %v", err)
	}
	assertOpStatus(t, db, "blocked", "accepted")
}

func completedFixture(t *testing.T, mode domain.ReviewMode) (string, *sql.DB, *clock.FakeClock) {
	t.Helper()
	d, db, c := fixture(t)
	in := input()
	in.ReviewMode = mode
	if _, _, err := (&publish.Machine{DB: db, DataDir: d, Clock: c}).Run(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	return d, db, c
}
func seedAccepted(t *testing.T, db *sql.DB, d string, c *clock.FakeClock, id string) {
	t.Helper()
	now := c.Now().Format(time.RFC3339Nano)
	if _, err := db.Exec(`INSERT INTO direct_ops(id,request_key,request_fingerprint,target_project_id,target_relative_path,review_mode,note_id,status,created_at,updated_at) VALUES(?,?,?,?,?,'none',?,'accepted',?,?)`, id, id, id, "p1", id+".md", "n"+id, now, now); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(d, "staging", "direct", id)
	if err := os.MkdirAll(stage, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stage, "body.md"), []byte(id), 0600); err != nil {
		t.Fatal(err)
	}
}
func assertOpStatus(t *testing.T, db *sql.DB, id, want string) {
	t.Helper()
	var got string
	if err := db.QueryRow(`SELECT status FROM direct_ops WHERE id=?`, id).Scan(&got); err != nil || got != want {
		t.Fatalf("op %s status=%q want=%q err=%v", id, got, want, err)
	}
}

func TestUnknownProjectAndUnsafeOpIDCreateNoArtifacts(t *testing.T) {
	d, db, c := fixture(t)
	m := publish.Machine{DB: db, DataDir: d, Clock: c}
	for _, mutate := range []func(*publish.PublishInput){
		func(in *publish.PublishInput) { in.TargetProjectID = "missing" },
		func(in *publish.PublishInput) { in.OpID = "../escape" },
	} {
		in := input()
		mutate(&in)
		if _, _, err := m.Run(context.Background(), in); err == nil {
			t.Fatal("expected rejection")
		}
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM direct_ops`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("ops=%d err=%v", count, err)
	}
	if _, err := os.Stat(filepath.Join(d, "staging")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("staging artifact: %v", err)
	}
}
