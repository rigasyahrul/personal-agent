package publish

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/fsroot"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	pathcheck "github.com/rigasyahrul/personal-agent/internal/paths"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

var ErrConflict = errors.New("publication conflict")
var ErrInvalid = errors.New("invalid publication")

type ConflictError struct{ Code string }

func (e *ConflictError) Error() string { return e.Code }
func (e *ConflictError) Unwrap() error { return ErrConflict }

type PublishInput struct {
	OpID, RequestKey, RequestFingerprint string
	Kind                                 string
	SessionID                            string
	WorkspacePath                        string
	Body                                 []byte
	TargetProjectID, TargetRelPath       string
	ReviewMode                           domain.ReviewMode
	NoteID                               string
}
type Machine struct {
	DB      *sql.DB
	DataDir string
	Clock   clock.Clock
	mu      sync.Mutex
}

func validComponent(s string) bool {
	return s != "" && s != "." && s != ".." && !strings.ContainsAny(s, `/\`) && !strings.ContainsRune(s, 0)
}
func validate(in PublishInput) (string, error) {
	if (in.Kind != "direct" && in.Kind != "promote") || !validComponent(in.OpID) || strings.TrimSpace(in.RequestKey) == "" || strings.TrimSpace(in.RequestFingerprint) == "" || strings.TrimSpace(in.NoteID) == "" || strings.TrimSpace(in.TargetProjectID) == "" || (in.Kind == "direct" && len(in.Body) > pathcheck.MaxMarkdownBytes) {
		return "", ErrInvalid
	}
	p, err := pathcheck.ValidateRelPath(in.TargetRelPath)
	if err != nil || strings.Contains(p, `\`) || path.Ext(p) != ".md" {
		return "", ErrInvalid
	}
	top := strings.Split(p, "/")[0]
	if top == "memory" || top == "soul" || (in.ReviewMode != "none" && in.ReviewMode != "whole" && in.ReviewMode != "bites") {
		return "", ErrInvalid
	}
	if in.Kind == "promote" {
		if !validComponent(in.SessionID) {
			return "", ErrInvalid
		}
		if _, err := pathcheck.ValidateRelPath(in.WorkspacePath); err != nil || strings.Contains(in.WorkspacePath, `\`) || path.Ext(in.WorkspacePath) != ".md" {
			return "", ErrInvalid
		}
	}
	return p, nil
}
func (m *Machine) now() string {
	return m.Clock.Now().UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
}
func stageName(kind, id string) string { return "staging/" + kind + "/" + id + "/body.md" }
func digest(b []byte) string           { return fmt.Sprintf("%x", sha256.Sum256(b)) }

func (m *Machine) dataRoot() (*fsroot.Root, error) { return fsroot.Open(m.DataDir) }
func (m *Machine) readStage(kind, id string) ([]byte, error) {
	if !validComponent(id) {
		return nil, ErrInvalid
	}
	r, err := m.dataRoot()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return r.ReadFile(stageName(kind, id), pathcheck.MaxMarkdownBytes)
}
func (m *Machine) writeStage(kind, id string, body []byte) error {
	if !validComponent(id) {
		return ErrInvalid
	}
	r, err := m.dataRoot()
	if err != nil {
		return err
	}
	defer r.Close()
	err = r.WriteFileNoReplace(stageName(kind, id), body, 0600)
	if errors.Is(err, fs.ErrExist) {
		old, readErr := r.ReadFile(stageName(kind, id), pathcheck.MaxMarkdownBytes)
		if readErr == nil && string(old) == string(body) {
			return nil
		}
		return ErrConflict
	}
	return err
}

func (m *Machine) Run(ctx context.Context, in PublishInput) (string, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.run(ctx, in)
}

func opTable(kind string) string {
	if kind == "promote" {
		return "promote_ops"
	}
	return "direct_ops"
}

func (m *Machine) operationByID(ctx context.Context, kind, id string) (store.DirectOperation, error) {
	if kind == "promote" {
		return (store.PromoteStore{DB: m.DB}).ByID(ctx, id)
	}
	return (store.DirectStore{DB: m.DB}).ByID(ctx, id)
}

func (m *Machine) operationAny(ctx context.Context, id string) (store.DirectOperation, error) {
	o, err := (store.DirectStore{DB: m.DB}).ByID(ctx, id)
	if store.IsNoRows(err) {
		return (store.PromoteStore{DB: m.DB}).ByID(ctx, id)
	}
	return o, err
}

func (m *Machine) run(ctx context.Context, in PublishInput) (string, string, error) {
	p, err := validate(in)
	if err != nil {
		return "", in.NoteID, err
	}
	if in.Kind == "promote" {
		return m.runPromote(ctx, in, p)
	}
	// Validate the project and canonical rooted vault before creating any artifact.
	probe := store.DirectOperation{ProjectID: in.TargetProjectID}
	r, err := m.root(ctx, probe)
	if err != nil {
		return "", in.NoteID, err
	}
	_ = r.Close()
	ds := store.DirectStore{DB: m.DB}
	old, err := ds.ByKey(ctx, in.RequestKey)
	if err == nil {
		if old.RequestFingerprint != in.RequestFingerprint {
			return old.Status, old.NoteID, ErrConflict
		}
		if old.Status == "failed" {
			return old.Status, old.NoteID, ErrConflict
		}
		if old.Status == "accepted" {
			if err = m.writeStage("direct", old.ID, in.Body); err != nil {
				return old.Status, old.NoteID, err
			}
		}
		if err = m.resume(ctx, old); err != nil {
			return old.Status, old.NoteID, err
		}
		old, err = ds.ByID(ctx, old.ID)
		if err != nil {
			return "", old.NoteID, err
		}
		return old.Status, old.NoteID, nil
	}
	if !store.IsNoRows(err) {
		return "", in.NoteID, err
	}
	now := m.now()
	_, err = m.DB.ExecContext(ctx, `INSERT INTO direct_ops(id,request_key,request_fingerprint,target_project_id,target_relative_path,review_mode,note_id,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,'accepted',?,?)`, in.OpID, in.RequestKey, in.RequestFingerprint, in.TargetProjectID, p, in.ReviewMode, in.NoteID, now, now)
	if err != nil {
		race, lookupErr := ds.ByKey(ctx, in.RequestKey)
		if lookupErr != nil {
			return "", in.NoteID, err
		}
		if race.RequestFingerprint != in.RequestFingerprint {
			return race.Status, race.NoteID, ErrConflict
		}
		return m.run(ctx, in)
	}
	if err = m.writeStage("direct", in.OpID, in.Body); err != nil {
		return "accepted", in.NoteID, err
	}
	o, err := ds.ByID(ctx, in.OpID)
	if err != nil {
		return "", in.NoteID, err
	}
	if err = m.resume(ctx, o); err != nil {
		return o.Status, o.NoteID, err
	}
	o, err = ds.ByID(ctx, in.OpID)
	if err != nil {
		return "", in.NoteID, err
	}
	return o.Status, o.NoteID, nil
}

func (m *Machine) runPromote(ctx context.Context, in PublishInput, targetPath string) (string, string, error) {
	ps := store.PromoteStore{DB: m.DB}
	old, err := ps.ByKey(ctx, in.RequestKey)
	if err == nil {
		if old.RequestFingerprint != in.RequestFingerprint {
			return old.Status, old.NoteID, &ConflictError{Code: "idempotency_key_reused"}
		}
		if old.Status == "failed" {
			return old.Status, old.NoteID, &ConflictError{Code: "destination_exists"}
		}
		if old.Status == "accepted" {
			body, loadErr := m.loadPromoteBody(ctx, in)
			if loadErr != nil {
				return old.Status, old.NoteID, loadErr
			}
			if err = m.writeStage("promote", old.ID, body); err != nil {
				return old.Status, old.NoteID, err
			}
		}
		if err = m.resume(ctx, old); err != nil {
			return old.Status, old.NoteID, err
		}
		old, err = ps.ByID(ctx, old.ID)
		return old.Status, old.NoteID, err
	}
	if !store.IsNoRows(err) {
		return "", in.NoteID, err
	}
	body, err := m.loadPromoteBody(ctx, in)
	if err != nil {
		return "", in.NoteID, err
	}
	now := m.now()
	_, err = m.DB.ExecContext(ctx, `INSERT INTO promote_ops(id,request_key,request_fingerprint,session_id,workspace_path,target_project_id,target_relative_path,review_mode,note_id,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,'accepted',?,?)`, in.OpID, in.RequestKey, in.RequestFingerprint, in.SessionID, in.WorkspacePath, in.TargetProjectID, targetPath, in.ReviewMode, in.NoteID, now, now)
	if err != nil {
		race, lookupErr := ps.ByKey(ctx, in.RequestKey)
		if lookupErr != nil {
			return "", in.NoteID, err
		}
		if race.RequestFingerprint != in.RequestFingerprint {
			return race.Status, race.NoteID, &ConflictError{Code: "idempotency_key_reused"}
		}
		if race.Status == "accepted" {
			if err = m.writeStage("promote", race.ID, body); err != nil {
				return race.Status, race.NoteID, err
			}
		}
		if err = m.resume(ctx, race); err != nil {
			return race.Status, race.NoteID, err
		}
		race, err = ps.ByID(ctx, race.ID)
		return race.Status, race.NoteID, err
	}
	if err = m.writeStage("promote", in.OpID, body); err != nil {
		return "accepted", in.NoteID, err
	}
	o, err := ps.ByID(ctx, in.OpID)
	if err == nil {
		err = m.resume(ctx, o)
	}
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return o.Status, o.NoteID, &ConflictError{Code: "destination_exists"}
		}
		return o.Status, o.NoteID, err
	}
	o, err = ps.ByID(ctx, in.OpID)
	return o.Status, o.NoteID, err
}

func (m *Machine) loadPromoteBody(ctx context.Context, in PublishInput) ([]byte, error) {
	var home, status, projectID string
	var vault sql.NullString
	if err := m.DB.QueryRowContext(ctx, `SELECT home,status,project_id,vault_id FROM sessions WHERE id=?`, in.SessionID).Scan(&home, &status, &projectID, &vault); err != nil {
		return nil, err
	}
	if status != "active" || home != "project" || projectID != in.TargetProjectID {
		return nil, fmt.Errorf("session project is the only promote target")
	}
	workspace, err := fsroot.Open(layout.SessionWorkspace(m.DataDir, layout.SessionHome(home), vault.String, projectID, in.SessionID))
	if err != nil {
		return nil, err
	}
	defer workspace.Close()
	return workspace.ReadFile(in.WorkspacePath, pathcheck.MaxMarkdownBytes)
}

var publicationRank = map[string]int{
	"accepted": 0, "frozen": 1, "path_reserved": 2, "published_fs": 3,
	"finalized": 4, "review_enqueued": 5, "completed": 6,
}

func compatibleAtLeast(status, target string) bool {
	got, ok := publicationRank[status]
	want, targetOK := publicationRank[target]
	return ok && targetOK && got >= want
}

func (m *Machine) advance(ctx context.Context, id, from, to string) error {
	o, err := m.operationAny(ctx, id)
	if err != nil {
		return err
	}
	res, err := m.DB.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET status=?,updated_at=? WHERE id=? AND status=?`, opTable(o.Kind)), to, m.now(), id, from)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 1 {
		return nil
	}
	o, err = m.operationByID(ctx, o.Kind, id)
	if err != nil {
		return err
	}
	if compatibleAtLeast(o.Status, to) {
		return nil
	}
	return fmt.Errorf("operation transition %s to %s affected %d rows", from, to, n)
}
func (m *Machine) fail(ctx context.Context, o store.DirectOperation, cause error) error {
	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET status='failed',error=?,updated_at=? WHERE id=? AND status NOT IN ('completed','failed')`, opTable(o.Kind)), cause.Error(), m.now(), o.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		var status string
		if err = tx.QueryRowContext(ctx, fmt.Sprintf(`SELECT status FROM %s WHERE id=?`, opTable(o.Kind)), o.ID).Scan(&status); err != nil || status != "failed" {
			return fmt.Errorf("failed transition: %w", err)
		}
	}
	if o.Status == "path_reserved" || o.Status == "published_fs" {
		res, err = tx.ExecContext(ctx, `UPDATE notes SET status='failed',updated_at=? WHERE id=? AND status='pending'`, m.now(), o.NoteID)
		if err != nil {
			return err
		}
		noteRows, rowsErr := res.RowsAffected()
		if rowsErr != nil {
			return rowsErr
		}
		if noteRows == 0 {
			var status string
			if err = tx.QueryRowContext(ctx, `SELECT status FROM notes WHERE id=?`, o.NoteID).Scan(&status); err != nil || status != "failed" {
				return fmt.Errorf("pending note failure reconciliation: %w", err)
			}
		} else if noteRows != 1 {
			return fmt.Errorf("pending note failure affected %d rows", noteRows)
		}
	}
	return tx.Commit()
}

func (m *Machine) resume(ctx context.Context, o store.DirectOperation) error {
	for {
		switch o.Status {
		case "accepted":
			b, err := m.readStage(o.Kind, o.ID)
			if err != nil {
				if x := m.fail(ctx, o, fmt.Errorf("staging unavailable: %w", err)); x != nil {
					return x
				}
				return ErrConflict
			}
			res, err := m.DB.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET status='frozen',frozen_sha256=?,frozen_size=?,updated_at=? WHERE id=? AND status='accepted'`, opTable(o.Kind)), digest(b), len(b), m.now(), o.ID)
			if err != nil {
				return err
			}
			n, rowsErr := res.RowsAffected()
			if rowsErr != nil {
				return rowsErr
			}
			if n > 1 {
				return fmt.Errorf("freeze transition affected %d rows", n)
			}
			if n == 0 {
				if err = m.advance(ctx, o.ID, "accepted", "frozen"); err != nil {
					return err
				}
			}
		case "frozen":
			if err := m.reserve(ctx, o); err != nil {
				return err
			}
		case "path_reserved":
			if err := m.publish(ctx, o); err != nil {
				return err
			}
		case "published_fs":
			if err := m.finalize(ctx, o); err != nil {
				return err
			}
		case "finalized":
			if err := m.enqueue(ctx, o); err != nil {
				return err
			}
		case "review_enqueued":
			if err := m.advance(ctx, o.ID, "review_enqueued", "completed"); err != nil {
				return err
			}
		case "completed":
			return nil
		case "failed":
			return ErrConflict
		default:
			return fmt.Errorf("unknown status %q", o.Status)
		}
		var err error
		o, err = m.operationByID(ctx, o.Kind, o.ID)
		if err != nil {
			return err
		}
	}
}

func (m *Machine) reserve(ctx context.Context, o store.DirectOperation) error {
	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := m.now()
	if o.Kind == "promote" {
		_, err = tx.ExecContext(ctx, `INSERT INTO notes(id,project_id,relative_path,status,origin_session_id,origin_workspace_path,revision,created_at,updated_at) VALUES(?,?,?,'pending',?,?,0,?,?) ON CONFLICT(id) DO NOTHING`, o.NoteID, o.ProjectID, o.RelativePath, o.SessionID, o.WorkspacePath, now, now)
	} else {
		_, err = tx.ExecContext(ctx, `INSERT INTO notes(id,project_id,relative_path,status,revision,created_at,updated_at) VALUES(?,?,?,'pending',0,?,?) ON CONFLICT(id) DO NOTHING`, o.NoteID, o.ProjectID, o.RelativePath, now, now)
	}
	if err != nil {
		var conflictingID string
		lookupErr := tx.QueryRowContext(ctx, `SELECT id FROM notes WHERE project_id=? AND relative_path=?`, o.ProjectID, o.RelativePath).Scan(&conflictingID)
		_ = tx.Rollback()
		if lookupErr == nil && conflictingID != o.NoteID {
			if x := m.fail(ctx, o, fmt.Errorf("destination path reserved by another note")); x != nil {
				return x
			}
			return ErrConflict
		}
		return err
	}
	var id string
	err = tx.QueryRowContext(ctx, `SELECT id FROM notes WHERE project_id=? AND relative_path=?`, o.ProjectID, o.RelativePath).Scan(&id)
	if err != nil || id != o.NoteID {
		_ = tx.Rollback()
		if x := m.fail(ctx, o, fmt.Errorf("destination path reserved by another note")); x != nil {
			return x
		}
		return ErrConflict
	}
	res, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET status='path_reserved',updated_at=? WHERE id=? AND status='frozen'`, opTable(o.Kind)), now, o.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		_ = tx.Rollback()
		current, reloadErr := m.operationByID(ctx, o.Kind, o.ID)
		if reloadErr == nil && compatibleAtLeast(current.Status, "path_reserved") {
			return nil
		}
		return fmt.Errorf("reserve transition affected %d rows: %w", n, reloadErr)
	}
	if n != 1 {
		return fmt.Errorf("reserve transition affected %d rows", n)
	}
	return tx.Commit()
}
func (m *Machine) root(ctx context.Context, o store.DirectOperation) (*fsroot.Root, error) {
	var vault sql.NullString
	if err := m.DB.QueryRowContext(ctx, `SELECT vault_id FROM projects WHERE id=?`, o.ProjectID).Scan(&vault); err != nil {
		return nil, err
	}
	return fsroot.Open(layout.SourceDir(layout.ProjectRoot(m.DataDir, vault.String, o.ProjectID)))
}

type destinationState uint8

const (
	destinationMissing destinationState = iota
	destinationMatches
	destinationMismatch
	destinationUnsafe
)

// inspectDestination keeps database/root lookup failures separate from durable
// source-tree artifacts. Only the latter are safe to terminalize during startup.
func (m *Machine) inspectDestination(ctx context.Context, o store.DirectOperation) (destinationState, error) {
	r, err := m.root(ctx, o)
	if err != nil {
		return destinationMissing, err
	}
	defer r.Close()
	b, err := r.ReadFile(o.RelativePath, pathcheck.MaxMarkdownBytes)
	if errors.Is(err, fs.ErrNotExist) {
		return destinationMissing, nil
	}
	if errors.Is(err, fsroot.ErrUnsafe) || errors.Is(err, fsroot.ErrInvalidPath) {
		return destinationUnsafe, nil
	}
	if err != nil {
		return destinationMissing, err
	}
	if digest(b) == o.FrozenSHA && int64(len(b)) == o.FrozenSize {
		return destinationMatches, nil
	}
	return destinationMismatch, nil
}
func (m *Machine) publish(ctx context.Context, o store.DirectOperation) error {
	state, err := m.inspectDestination(ctx, o)
	if err != nil {
		return err
	}
	if state == destinationMatches {
		return m.advance(ctx, o.ID, "path_reserved", "published_fs")
	}
	if state == destinationMismatch || state == destinationUnsafe {
		cause := fmt.Errorf("destination is an irrecoverable publication artifact")
		if x := m.fail(ctx, o, cause); x != nil {
			return x
		}
		return ErrConflict
	}
	b, err := m.readStage(o.Kind, o.ID)
	if err != nil || digest(b) != o.FrozenSHA || int64(len(b)) != o.FrozenSize {
		cause := fmt.Errorf("frozen staging unavailable or mismatched")
		if x := m.fail(ctx, o, cause); x != nil {
			return x
		}
		return ErrConflict
	}
	r, err := m.root(ctx, o)
	if err != nil {
		return err
	}
	defer r.Close()
	err = r.WriteFileNoReplace(o.RelativePath, b, 0600)
	if errors.Is(err, fs.ErrExist) {
		state, inspectErr := m.inspectDestination(ctx, o)
		if inspectErr != nil {
			return inspectErr
		}
		if state == destinationMatches {
			return m.advance(ctx, o.ID, "path_reserved", "published_fs")
		}
	}
	if errors.Is(err, fs.ErrExist) || errors.Is(err, fsroot.ErrUnsafe) || errors.Is(err, fsroot.ErrInvalidPath) {
		if x := m.fail(ctx, o, fmt.Errorf("destination became an irrecoverable publication artifact: %w", err)); x != nil {
			return x
		}
		return ErrConflict
	}
	if err != nil {
		return err
	}
	return m.advance(ctx, o.ID, "path_reserved", "published_fs")
}
func (m *Machine) finalize(ctx context.Context, o store.DirectOperation) error {
	state, err := m.inspectDestination(ctx, o)
	if err != nil {
		return err
	}
	if state != destinationMatches {
		if x := m.fail(ctx, o, fmt.Errorf("published destination is missing, mismatched, or unsafe")); x != nil {
			return x
		}
		return ErrConflict
	}
	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := m.now()
	res, err := tx.ExecContext(ctx, `UPDATE notes SET content_sha256=?,byte_size=?,status='ready',revision=1,updated_at=? WHERE id=? AND status='pending'`, o.FrozenSHA, o.FrozenSize, now, o.NoteID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		var sha string
		var size int64
		var status string
		if err = tx.QueryRowContext(ctx, `SELECT coalesce(content_sha256,''),coalesce(byte_size,0),status FROM notes WHERE id=?`, o.NoteID).Scan(&sha, &size, &status); err != nil || status != "ready" || sha != o.FrozenSHA || size != o.FrozenSize {
			return fmt.Errorf("note finalization reconciliation failed: %w", err)
		}
	}
	res, err = tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET status='finalized',updated_at=? WHERE id=? AND status='published_fs'`, opTable(o.Kind)), now, o.ID)
	if err != nil {
		return err
	}
	n, err = res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		var status string
		if err = tx.QueryRowContext(ctx, fmt.Sprintf(`SELECT status FROM %s WHERE id=?`, opTable(o.Kind)), o.ID).Scan(&status); err != nil || !compatibleAtLeast(status, "finalized") {
			return fmt.Errorf("finalize transition affected %d rows: %w", n, err)
		}
	} else if n != 1 {
		return fmt.Errorf("finalize transition affected %d rows: %w", n, err)
	}
	return tx.Commit()
}
func (m *Machine) enqueue(ctx context.Context, o store.DirectOperation) error {
	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := m.now()
	if o.ReviewMode == "whole" {
		_, err = tx.ExecContext(ctx, `INSERT INTO review_items(id,project_id,note_id,kind,source_sha256,source_revision,prompt,stage,due_at,interval_days,ease_factor,reps,lapses,row_version,status,scheduler_version) VALUES(?,?,?,'whole',?,1,'Review this note',0,?,0,2.5,0,0,0,'active','sm2-lite-v1') ON CONFLICT(note_id,source_revision) WHERE kind='whole' AND status='active' DO NOTHING`, uuid.NewString(), o.ProjectID, o.NoteID, o.FrozenSHA, now)
	} else if o.ReviewMode == "bites" {
		_, err = tx.ExecContext(ctx, `INSERT INTO review_pending(id,note_id,source_sha256,generator_version,status,attempts,created_at,updated_at) VALUES(?,?,?,'bites-v1','pending',0,?,?) ON CONFLICT(note_id,source_sha256,generator_version) DO NOTHING`, uuid.NewString(), o.NoteID, o.FrozenSHA, now, now)
	}
	if err != nil {
		return err
	}
	if o.ReviewMode == "whole" {
		var count int
		err = tx.QueryRowContext(ctx, `SELECT count(*) FROM review_items WHERE project_id=? AND note_id=? AND kind='whole' AND source_sha256=? AND source_revision=1 AND status='active' AND scheduler_version='sm2-lite-v1'`, o.ProjectID, o.NoteID, o.FrozenSHA).Scan(&count)
		if err != nil || count != 1 {
			return fmt.Errorf("whole review reconciliation failed: count=%d: %w", count, err)
		}
	} else if o.ReviewMode == "bites" {
		var count int
		err = tx.QueryRowContext(ctx, `SELECT count(*) FROM review_pending WHERE note_id=? AND source_sha256=? AND generator_version='bites-v1'`, o.NoteID, o.FrozenSHA).Scan(&count)
		if err != nil || count != 1 {
			return fmt.Errorf("bites review reconciliation failed: count=%d: %w", count, err)
		}
	}
	res, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET status='review_enqueued',updated_at=? WHERE id=? AND status='finalized'`, opTable(o.Kind)), now, o.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		var status string
		if err = tx.QueryRowContext(ctx, fmt.Sprintf(`SELECT status FROM %s WHERE id=?`, opTable(o.Kind)), o.ID).Scan(&status); err != nil || !compatibleAtLeast(status, "review_enqueued") {
			return fmt.Errorf("enqueue transition affected %d rows: %w", n, err)
		}
	} else if n != 1 {
		return fmt.Errorf("enqueue transition affected %d rows: %w", n, err)
	}
	return tx.Commit()
}
