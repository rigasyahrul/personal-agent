package publish

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

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
}

func validate(in PublishInput) (string, error) {
	if in.Kind != "direct" || strings.TrimSpace(in.OpID) == "" || strings.TrimSpace(in.RequestKey) == "" || strings.TrimSpace(in.RequestFingerprint) == "" || strings.TrimSpace(in.NoteID) == "" || strings.TrimSpace(in.TargetProjectID) == "" || len(in.Body) > pathcheck.MaxMarkdownBytes {
		return "", ErrInvalid
	}
	p, e := pathcheck.ValidateRelPath(in.TargetRelPath)
	if e != nil || strings.Contains(p, `\`) || path.Ext(p) != ".md" {
		return "", ErrInvalid
	}
	top := strings.Split(p, "/")[0]
	if top == "memory" || top == "soul" {
		return "", ErrInvalid
	}
	if in.ReviewMode != "none" && in.ReviewMode != "whole" && in.ReviewMode != "bites" {
		return "", ErrInvalid
	}
	return p, nil
}
func (m *Machine) now() string {
	return m.Clock.Now().UTC().Format("2006-01-02T15:04:05.999999999Z07:00")
}
func (m *Machine) Run(ctx context.Context, in PublishInput) (string, string, error) {
	p, e := validate(in)
	if e != nil {
		return "", in.NoteID, e
	}
	ds := store.DirectStore{DB: m.DB}
	old, e := ds.ByKey(ctx, in.RequestKey)
	if e == nil {
		if old.RequestFingerprint != in.RequestFingerprint {
			return old.Status, old.NoteID, ErrConflict
		}
		if e = m.resume(ctx, old); e != nil {
			return old.Status, old.NoteID, e
		}
		old, _ = ds.ByID(ctx, old.ID)
		return old.Status, old.NoteID, nil
	}
	if !store.IsNoRows(e) {
		return "", in.NoteID, e
	}
	stage := filepath.Join(m.DataDir, "staging", "direct", in.OpID)
	if e = os.MkdirAll(filepath.Dir(stage), 0700); e == nil {
		e = writeStage(stage, in.Body)
	}
	if e != nil {
		return "", in.NoteID, e
	}
	now := m.now()
	_, e = m.DB.ExecContext(ctx, `INSERT INTO direct_ops(id,request_key,request_fingerprint,target_project_id,target_relative_path,review_mode,note_id,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,'accepted',?,?)`, in.OpID, in.RequestKey, in.RequestFingerprint, in.TargetProjectID, p, in.ReviewMode, in.NoteID, now, now)
	if e != nil {
		old, x := ds.ByKey(ctx, in.RequestKey)
		if x == nil && old.RequestFingerprint == in.RequestFingerprint {
			return m.Run(ctx, in)
		}
		return "", in.NoteID, ErrConflict
	}
	sum := fmt.Sprintf("%x", sha256.Sum256(in.Body))
	_, e = m.DB.ExecContext(ctx, `UPDATE direct_ops SET status='frozen',frozen_sha256=?,frozen_size=?,updated_at=? WHERE id=? AND status='accepted'`, sum, len(in.Body), now, in.OpID)
	if e != nil {
		return "", in.NoteID, e
	}
	o, _ := ds.ByID(ctx, in.OpID)
	if e = m.resume(ctx, o); e != nil {
		return o.Status, o.NoteID, e
	}
	o, _ = ds.ByID(ctx, in.OpID)
	return o.Status, o.NoteID, nil
}
func writeStage(name string, b []byte) error {
	f, e := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if errors.Is(e, fs.ErrExist) {
		old, x := os.ReadFile(name)
		if x == nil && string(old) == string(b) {
			return nil
		}
		return ErrConflict
	}
	if e != nil {
		return e
	}
	if _, e = f.Write(b); e == nil {
		e = f.Sync()
	}
	x := f.Close()
	if e == nil {
		e = x
	}
	if e == nil {
		d, x := os.Open(filepath.Dir(name))
		if x != nil {
			return x
		}
		e = d.Sync()
		_ = d.Close()
	}
	return e
}

func (m *Machine) resume(ctx context.Context, o store.DirectOperation) error {
	for {
		switch o.Status {
		case "accepted":
			b, err := os.ReadFile(filepath.Join(m.DataDir, "staging", "direct", o.ID))
			if err != nil {
				return err
			}
			sum := fmt.Sprintf("%x", sha256.Sum256(b))
			if _, err = m.DB.ExecContext(ctx, `UPDATE direct_ops SET status='frozen',frozen_sha256=?,frozen_size=?,updated_at=? WHERE id=? AND status='accepted'`, sum, len(b), m.now(), o.ID); err != nil {
				return err
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
			_, err := m.DB.ExecContext(ctx, `UPDATE direct_ops SET status='completed',updated_at=? WHERE id=? AND status='review_enqueued'`, m.now(), o.ID)
			if err != nil {
				return err
			}
		case "completed":
			return nil
		case "failed":
			return errors.New("publication failed")
		default:
			return fmt.Errorf("unknown status %q", o.Status)
		}
		var e error
		o, e = (store.DirectStore{DB: m.DB}).ByID(ctx, o.ID)
		if e != nil {
			return e
		}
	}
}
func (m *Machine) reserve(ctx context.Context, o store.DirectOperation) error {
	tx, e := m.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	now := m.now()
	_, e = tx.ExecContext(ctx, `INSERT INTO notes(id,project_id,relative_path,status,revision,created_at,updated_at) VALUES(?,?,?,'pending',0,?,?) ON CONFLICT(id) DO NOTHING`, o.NoteID, o.ProjectID, o.RelativePath, now, now)
	if e != nil {
		return ErrConflict
	}
	var id string
	if e = tx.QueryRowContext(ctx, `SELECT id FROM notes WHERE project_id=? AND relative_path=?`, o.ProjectID, o.RelativePath).Scan(&id); e != nil || id != o.NoteID {
		return ErrConflict
	}
	_, e = tx.ExecContext(ctx, `UPDATE direct_ops SET status='path_reserved',updated_at=? WHERE id=? AND status='frozen'`, now, o.ID)
	if e != nil {
		return e
	}
	return tx.Commit()
}
func (m *Machine) root(ctx context.Context, o store.DirectOperation) (*fsroot.Root, error) {
	var vault sql.NullString
	if e := m.DB.QueryRowContext(ctx, `SELECT vault_id FROM projects WHERE id=?`, o.ProjectID).Scan(&vault); e != nil {
		return nil, e
	}
	return fsroot.Open(layout.SourceDir(layout.ProjectRoot(m.DataDir, vault.String, o.ProjectID)))
}
func (m *Machine) publish(ctx context.Context, o store.DirectOperation) error {
	b, e := os.ReadFile(filepath.Join(m.DataDir, "staging", "direct", o.ID))
	if e != nil {
		return e
	}
	if fmt.Sprintf("%x", sha256.Sum256(b)) != o.FrozenSHA || int64(len(b)) != o.FrozenSize {
		return errors.New("frozen bytes mismatch")
	}
	r, e := m.root(ctx, o)
	if e != nil {
		return e
	}
	defer r.Close()
	e = r.WriteFileNoReplace(o.RelativePath, b, 0600)
	if errors.Is(e, fs.ErrExist) {
		old, x := r.ReadFile(o.RelativePath, pathcheck.MaxMarkdownBytes)
		if x == nil && fmt.Sprintf("%x", sha256.Sum256(old)) == o.FrozenSHA {
			return m.setStatus(ctx, o.ID, "published_fs", "path_reserved")
		}
		return ErrConflict
	}
	if e != nil {
		return e
	}
	return m.setStatus(ctx, o.ID, "published_fs", "path_reserved")
}
func (m *Machine) setStatus(ctx context.Context, id, to, from string) error {
	_, e := m.DB.ExecContext(ctx, `UPDATE direct_ops SET status=?,updated_at=? WHERE id=? AND status=?`, to, m.now(), id, from)
	return e
}
func (m *Machine) finalize(ctx context.Context, o store.DirectOperation) error {
	r, e := m.root(ctx, o)
	if e != nil {
		return e
	}
	b, e := r.ReadFile(o.RelativePath, pathcheck.MaxMarkdownBytes)
	r.Close()
	if e != nil || fmt.Sprintf("%x", sha256.Sum256(b)) != o.FrozenSHA {
		return errors.New("published bytes mismatch")
	}
	tx, e := m.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	now := m.now()
	if _, e = tx.ExecContext(ctx, `UPDATE notes SET content_sha256=?,byte_size=?,status='ready',revision=1,updated_at=? WHERE id=?`, o.FrozenSHA, o.FrozenSize, now, o.NoteID); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, `UPDATE direct_ops SET status='finalized',updated_at=? WHERE id=? AND status='published_fs'`, now, o.ID); e != nil {
		return e
	}
	return tx.Commit()
}
func (m *Machine) enqueue(ctx context.Context, o store.DirectOperation) error {
	tx, e := m.DB.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	now := m.now()
	if o.ReviewMode == "whole" {
		_, e = tx.ExecContext(ctx, `INSERT INTO review_items(id,project_id,note_id,kind,source_sha256,source_revision,prompt,stage,due_at,interval_days,ease_factor,reps,lapses,row_version,status,scheduler_version) VALUES(?,?,?,'whole',?,1,'Review this note',0,?,0,2.5,0,0,1,'active','sm2-lite-v1') ON CONFLICT DO NOTHING`, uuid.NewString(), o.ProjectID, o.NoteID, o.FrozenSHA, now)
	} else if o.ReviewMode == "bites" {
		_, e = tx.ExecContext(ctx, `INSERT INTO review_pending(id,note_id,source_sha256,generator_version,status,attempts,created_at,updated_at) VALUES(?,?,?,'bites-v1','pending',0,?,?) ON CONFLICT DO NOTHING`, uuid.NewString(), o.NoteID, o.FrozenSHA, now, now)
	}
	if e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, `UPDATE direct_ops SET status='review_enqueued',updated_at=? WHERE id=? AND status='finalized'`, now, o.ID); e != nil {
		return e
	}
	return tx.Commit()
}
