package publish_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	cases := []publish.PublishInput{input()}
	cases[0].Kind = "promote"
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
