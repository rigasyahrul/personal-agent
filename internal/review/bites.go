package review

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/fsroot"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/paths"
)

type BiteGenerator interface {
	Generate(context.Context, string) ([]Bite, error)
}
type Bite struct{ Prompt, Answer string }

func validateBites(in []Bite) ([]Bite, error) {
	if len(in) < 1 || len(in) > 8 {
		return nil, fmt.Errorf("generator returned %d bites", len(in))
	}
	out := make([]Bite, len(in))
	for i, b := range in {
		b.Prompt = strings.TrimSpace(b.Prompt)
		b.Answer = strings.TrimSpace(b.Answer)
		if b.Prompt == "" || b.Answer == "" {
			return nil, errors.New("bite prompt and answer must be non-empty")
		}
		out[i] = b
	}
	return out, nil
}

var ErrLeaseLost = errors.New("review generation lease lost")

type BiteWorker struct {
	DB        *sql.DB
	DataDir   string
	Clock     clock.Clock
	Generator BiteGenerator
	Lease     time.Duration
}
type leasedJob struct {
	id, noteID, hash, until string
	attempts                int
}

func (w *BiteWorker) LeaseAndRun(ctx context.Context) (bool, error) {
	job, ok, err := w.acquire(ctx)
	if err != nil || !ok {
		return false, err
	}
	note, err := w.readNote(ctx, job.noteID, job.hash)
	if err != nil {
		return true, w.fail(ctx, job, err)
	}
	if w.Generator == nil {
		return true, w.fail(ctx, job, errors.New("bite generator is nil"))
	}
	bites, err := w.Generator.Generate(ctx, string(note.body))
	if err == nil {
		bites, err = validateBites(bites)
	}
	if err != nil {
		return true, w.fail(ctx, job, err)
	}
	if err = w.complete(ctx, job, note, bites); err != nil {
		return true, err
	}
	return true, nil
}

func (w *BiteWorker) acquire(ctx context.Context) (leasedJob, bool, error) {
	tx, err := w.DB.BeginTx(ctx, nil)
	if err != nil {
		return leasedJob{}, false, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT id,note_id,source_sha256,status,lease_until,attempts FROM review_pending WHERE generator_version='bites-v1' AND status IN ('pending','leased') ORDER BY created_at,id`)
	if err != nil {
		return leasedJob{}, false, err
	}
	now := w.Clock.Now().UTC()
	var chosen leasedJob
	var oldStatus string
	var oldLease sql.NullString
	for rows.Next() {
		var j leasedJob
		var status string
		var lease sql.NullString
		if err := rows.Scan(&j.id, &j.noteID, &j.hash, &status, &lease, &j.attempts); err != nil {
			return j, false, err
		}
		eligible := status == "pending"
		if status == "leased" && lease.Valid {
			expiry, e := time.Parse(time.RFC3339Nano, lease.String)
			if e != nil {
				return j, false, fmt.Errorf("parse lease: %w", e)
			}
			eligible = !expiry.After(now)
		}
		if eligible {
			chosen = j
			oldStatus = status
			oldLease = lease
			break
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return chosen, false, err
	}
	if err := rows.Close(); err != nil {
		return chosen, false, err
	}
	if chosen.id == "" {
		return chosen, false, nil
	}
	chosen.attempts++
	chosen.until = now.Add(w.Lease).Format(time.RFC3339Nano)
	var res sql.Result
	if oldStatus == "pending" {
		res, err = tx.ExecContext(ctx, `UPDATE review_pending SET status='leased',attempts=?,lease_until=?,last_error=NULL,updated_at=? WHERE id=? AND status='pending'`, chosen.attempts, chosen.until, now.Format(time.RFC3339Nano), chosen.id)
	} else {
		res, err = tx.ExecContext(ctx, `UPDATE review_pending SET attempts=?,lease_until=?,last_error=NULL,updated_at=? WHERE id=? AND status='leased' AND lease_until=?`, chosen.attempts, chosen.until, now.Format(time.RFC3339Nano), chosen.id, oldLease.String)
	}
	if err != nil {
		return chosen, false, err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return chosen, false, nil
	}
	if err = tx.Commit(); err != nil {
		return chosen, false, err
	}
	return chosen, true, nil
}

type noteSnapshot struct {
	projectID string
	revision  int
	body      []byte
}

func (w *BiteWorker) readNote(ctx context.Context, noteID, jobHash string) (noteSnapshot, error) {
	var n noteSnapshot
	var rel, hash string
	var size int64
	var vault sql.NullString
	err := w.DB.QueryRowContext(ctx, `SELECT n.project_id,n.relative_path,n.content_sha256,n.byte_size,n.revision,p.vault_id FROM notes n JOIN projects p ON p.id=n.project_id WHERE n.id=? AND n.status='ready'`, noteID).Scan(&n.projectID, &rel, &hash, &size, &n.revision, &vault)
	if err != nil {
		return n, fmt.Errorf("read ready note: %w", err)
	}
	if hash != jobHash {
		return n, errors.New("review source hash does not match note")
	}
	if _, err = paths.ValidateRelPath(rel); err != nil || size < 0 || size > paths.MaxMarkdownBytes {
		return n, errors.New("invalid note metadata")
	}
	r, err := fsroot.Open(layout.SourceDir(layout.ProjectRoot(w.DataDir, vault.String, n.projectID)))
	if err != nil {
		return n, err
	}
	defer r.Close()
	n.body, err = r.ReadFile(rel, paths.MaxMarkdownBytes)
	if err != nil {
		return n, err
	}
	actual := fmt.Sprintf("%x", sha256.Sum256(n.body))
	if actual != hash || int64(len(n.body)) != size {
		return n, errors.New("note integrity check failed")
	}
	return n, nil
}

func (w *BiteWorker) owns(ctx context.Context, tx *sql.Tx, j leasedJob) (bool, error) {
	var count int
	err := tx.QueryRowContext(ctx, `SELECT count(*) FROM review_pending WHERE id=? AND status='leased' AND lease_until=? AND attempts=?`, j.id, j.until, j.attempts).Scan(&count)
	return count == 1, err
}
func (w *BiteWorker) fail(ctx context.Context, j leasedJob, cause error) error {
	tx, err := w.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ok, err := w.owns(ctx, tx, j)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: %v", ErrLeaseLost, cause)
	}
	_, err = tx.ExecContext(ctx, `UPDATE review_pending SET status='failed',lease_until=NULL,last_error=?,updated_at=? WHERE id=? AND status='leased' AND lease_until=? AND attempts=?`, cause.Error(), w.Clock.Now().UTC().Format(time.RFC3339Nano), j.id, j.until, j.attempts)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return cause
}
func (w *BiteWorker) complete(ctx context.Context, j leasedJob, n noteSnapshot, bites []Bite) error {
	tx, err := w.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ok, err := w.owns(ctx, tx, j)
	if err != nil {
		return err
	}
	if !ok {
		return ErrLeaseLost
	}
	now := w.Clock.Now().UTC().Format(time.RFC3339Nano)
	for ordinal, b := range bites {
		_, err = tx.ExecContext(ctx, `INSERT INTO review_items(id,project_id,note_id,kind,source_sha256,source_revision,prompt,answer,generation_id,ordinal,stage,due_at,interval_days,ease_factor,reps,lapses,last_reviewed_at,row_version,status,scheduler_version) VALUES(?,?,?,'bite',?,?,?,?,?,?,0,?,0,2.5,0,0,NULL,0,'active','sm2-lite-v1')`, uuid.NewString(), n.projectID, j.noteID, j.hash, n.revision, b.Prompt, b.Answer, j.id, ordinal, now)
		if err != nil {
			return err
		}
	}
	res, err := tx.ExecContext(ctx, `UPDATE review_pending SET status='completed',lease_until=NULL,last_error=NULL,updated_at=? WHERE id=? AND status='leased' AND lease_until=? AND attempts=?`, now, j.id, j.until, j.attempts)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected != 1 {
		return ErrLeaseLost
	}
	return tx.Commit()
}
