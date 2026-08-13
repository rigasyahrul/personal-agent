package publish

import (
	"context"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func TestAdvanceReconcilesAlreadyAtTargetAndRejectsUnexpectedState(t *testing.T) {
	db, _ := testutil.TempDB(t)
	_, err := db.Exec(`INSERT INTO vaults(id,name,created_at,updated_at) VALUES('v','v','x','x'); INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES('p','v','p','x','x'); INSERT INTO direct_ops(id,request_key,request_fingerprint,target_project_id,target_relative_path,review_mode,note_id,status,created_at,updated_at) VALUES('o','k','f','p','x.md','none','n','published_fs','x','x')`)
	if err != nil {
		t.Fatal(err)
	}
	m := Machine{DB: db, Clock: &clock.FakeClock{T: time.Now()}}
	if err := m.advance(context.Background(), "o", "path_reserved", "published_fs"); err != nil {
		t.Fatalf("already target: %v", err)
	}
	if err := m.advance(context.Background(), "o", "frozen", "path_reserved"); err == nil {
		t.Fatal("unexpected state accepted")
	}
}
