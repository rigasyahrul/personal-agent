package acceptance

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/rigasyahrul/personal-agent/internal/fsroot"
	"github.com/rigasyahrul/personal-agent/internal/layout"
)

func TestAcceptance01PromoteRetrySameKeyOneNoteOneReviewSet(t *testing.T) {
	h := newHarness(t)
	s := h.projectSession("accept-01")
	h.workspaceFile(s, "lesson.md", "# Lesson")
	a := h.promote(s, "lesson.md", "notes/lesson.md", "whole", "promote-key")
	b := h.promote(s, "lesson.md", "notes/lesson.md", "whole", "promote-key")
	if a.NoteID != b.NoteID {
		t.Fatalf("note IDs differ: %s %s", a.NoteID, b.NoteID)
	}
	h.assertCount("notes", "id=?", 1, a.NoteID)
	h.assertCount("review_items", "note_id=?", 1, a.NoteID)
}

func TestAcceptance02CrashAfterFSPublishRecoveryConverges(t *testing.T) {
	h := newHarness(t)
	s := h.projectSession("accept-02")
	h.workspaceFile(s, "crash.md", "# Durable")
	h.crashAfter("published_fs")
	op := h.promoteExpectInterrupted(s, "crash.md", "notes/crash.md", "whole", "crash-key")
	h.restart()
	h.recover()
	h.assertOperationStatus(op.OpID, "completed")
	h.assertReadyNoteFile(op.NoteID, "# Durable")
}

func TestAcceptance03BiteFailureRetryNoDuplicateNote(t *testing.T) {
	h := newHarness(t)
	h.bites.failNext(errors.New("generator unavailable"))
	n := h.directNote("notes/bites.md", "# Bites", "bites", "bite-key")
	h.runBiteWorker()
	h.assertPendingStatus(n.PendingID, "failed")
	h.retryPending(n.PendingID)
	h.runBiteWorker()
	h.assertCount("notes", "id=?", 1, n.NoteID)
	h.assertCount("review_items", "note_id=?", h.bites.generatedCount(), n.NoteID)
}

func TestAcceptance04InvalidSessionScopeRejectedAPIAndDB(t *testing.T) {
	h := newHarness(t)
	res := h.rawJSON("POST", "/api/v1/projects/"+h.projectID+"/sessions", `{"home":"vault","vault_id":"wrong","title":"bad","provider":"openai","model_id":"test"}`, true)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("API status=%d body=%s", res.Code, res.Body.String())
	}
	if _, err := h.db.Exec(`INSERT INTO sessions(id,home,vault_id,project_id,title,provider,model_id,status,created_at,updated_at,model_parameters_json,tool_grants_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		"bad", "vault", "wrong", h.projectID, "bad", "openai", "test", "active", h.now(), h.now(), `{}`, `{}`); err == nil {
		t.Fatal("database accepted mismatched vault/project scope")
	}
}

func TestAcceptance05TraversalAndSymlinkEscapeRejected(t *testing.T) {
	h := newHarness(t)
	s := h.projectSession("accept-05")
	home, vaultID, projectID := h.sessionMeta(s)
	wsRoot := layout.SessionWorkspace(h.dataDir, layout.SessionHome(home), vaultID, projectID, s)
	root, err := fsroot.Open(wsRoot)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	for _, p := range []string{"../secret.md", "/tmp/secret.md"} {
		if err := root.WriteFileAtomic(p, []byte("stolen"), 0o600); err == nil {
			t.Fatalf("accepted %q", p)
		}
	}
	outside := filepath.Join(t.TempDir(), "secret.md")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := h.workspaceSymlink(s, "escape.md", outside); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if got := h.workspaceRead(s, "escape.md"); got.Code < 400 {
		t.Fatal("followed symlink escape")
	}
}

func TestAcceptance06DestinationExists409NoOverwrite(t *testing.T) {
	h := newHarness(t)
	h.directNote("notes/existing.md", "original", "none", "first-key")
	res := h.directNoteResponse("notes/existing.md", "replacement", "none", "second-key")
	if res.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if got := h.sourceBody("notes/existing.md"); got != "original" {
		t.Fatalf("body=%q", got)
	}
}

func TestAcceptance07SessionDeleteRemovesWorkspaceOnly(t *testing.T) {
	h := newHarness(t)
	s := h.projectSession("accept-07")
	h.workspaceFile(s, "keep.md", "keep")
	n := h.promote(s, "keep.md", "notes/keep.md", "whole", "keep-key")
	h.deleteSession(s)
	h.assertWorkspaceAbsent(s)
	h.assertReadyNoteFile(n.NoteID, "keep")
	h.assertCount("review_items", "note_id=?", 1, n.NoteID)
}

func TestAcceptance08RatingRetrySameKeyOneEvent(t *testing.T) {
	h := newHarness(t)
	n := h.directNote("notes/rate.md", "rate", "whole", "rate-note-key")
	h.rate(n.ReviewItemID, "good", "rating-key")
	h.rate(n.ReviewItemID, "good", "rating-key")
	h.assertCount("review_events", "review_item_id=? AND request_key=?", 1, n.ReviewItemID, "rating-key")
}

func TestAcceptance09TwoTabsOneAgentRun(t *testing.T) {
	h := newHarness(t)
	s := h.projectSession("accept-09")
	h.provider.block()
	a, b := h.parallelMessages(s, "tab-a", "tab-b")
	if !((a.Code == http.StatusAccepted && b.Code == http.StatusConflict) || (b.Code == http.StatusAccepted && a.Code == http.StatusConflict)) {
		t.Fatalf("statuses=%d,%d bodies=%s | %s", a.Code, b.Code, a.Body.String(), b.Body.String())
	}
	h.assertCount("agent_runs", "session_id=? AND status IN ('queued','running')", 1, s)
	h.provider.releaseAll()
}

func TestAcceptance10BackupRestoreLastBundleSucceeds(t *testing.T) {
	h := newHarness(t)
	n := h.directNote("notes/backup.md", "restored", "whole", "backup-note-key")
	bundle := h.backupNow()
	restored := h.restoreBundle(bundle)
	restored.assertReadyNoteFile(n.NoteID, "restored")
	restored.assertManifestChecksums()
}

func TestAcceptance11UnauthenticatedMutationRejected(t *testing.T) {
	h := newHarness(t)
	res := h.rawJSON("POST", "/api/v1/projects", `{"name":"takeover"}`, false)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	h.assertCount("projects", "name=?", 0, "takeover")
}
