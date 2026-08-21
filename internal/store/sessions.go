package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/config"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/ids"
	"github.com/rigasyahrul/personal-agent/internal/layout"
)

const defaultToolGrantsJSON = `{"workspace_files":false}`

// SessionLocks serializes per-session mutations (promote and delete).
type SessionLocks struct {
	mu   sync.Mutex
	byID map[string]*sessionLock
}

type sessionLock struct {
	mu   sync.Mutex
	refs int
}

func NewSessionLocks() *SessionLocks {
	return &SessionLocks{byID: make(map[string]*sessionLock)}
}

// Lock acquires the keyed session lock and returns an unlock function.
func (s *SessionLocks) Lock(id string) func() {
	s.mu.Lock()
	l := s.byID[id]
	if l == nil {
		l = &sessionLock{}
		s.byID[id] = l
	}
	l.refs++
	s.mu.Unlock()
	l.mu.Lock()
	return func() {
		l.mu.Unlock()
		s.mu.Lock()
		l.refs--
		if l.refs == 0 {
			delete(s.byID, id)
		}
		s.mu.Unlock()
	}
}

type SessionStore struct {
	DB      *sql.DB
	DataDir string
	Now     func() time.Time
	Models  []config.ModelRef
	Barrier MutBarrier
	Locks   *SessionLocks
}

type CreateSessionInput struct {
	ProjectID, Title, Provider, ModelID, ModelParametersJSON, ToolGrantsJSON string
}

func (s *SessionStore) CreateProject(ctx context.Context, in CreateSessionInput) (domain.Session, error) {
	var out domain.Session
	err := s.withBarrier(func() error {
		var e error
		out, e = s.createProject(ctx, in)
		return e
	})
	return out, err
}

func (s *SessionStore) withBarrier(fn func() error) error {
	if s.Barrier == nil {
		return fn()
	}
	return s.Barrier.Mutate(fn)
}

func (s *SessionStore) createProject(ctx context.Context, in CreateSessionInput) (domain.Session, error) {
	var out domain.Session
	if !s.modelConfigured(in.Provider, in.ModelID) {
		return out, ErrValidation
	}

	var vaultID sql.NullString
	if err := s.DB.QueryRowContext(ctx, `SELECT vault_id FROM projects WHERE id=?`, in.ProjectID).Scan(&vaultID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return out, ErrNotFound
		}
		return out, err
	}
	if in.ToolGrantsJSON == "" {
		in.ToolGrantsJSON = defaultToolGrantsJSON
	}

	now := s.Now().UTC()
	id := ids.NewID()
	projectID := in.ProjectID
	out = domain.Session{
		ID: id, Home: layout.SessionHome("project"), ProjectID: &projectID, Status: "active",
		Provider: in.Provider, ModelID: in.ModelID, ModelParametersJSON: in.ModelParametersJSON,
		ToolGrantsJSON: in.ToolGrantsJSON, Title: in.Title, CreatedAt: now, UpdatedAt: now,
	}
	if vaultID.Valid {
		vault := vaultID.String
		out.VaultID = &vault
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `INSERT INTO sessions
		(id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at)
		VALUES(?,'project',?,?,'active',?,?,?,?,?,?,?)`, id, nullable(vaultID.String), in.ProjectID,
		in.Provider, in.ModelID, in.ModelParametersJSON, in.ToolGrantsJSON, in.Title, formatTime(now), formatTime(now)); err != nil {
		return out, err
	}

	workspace := layout.SessionWorkspace(s.DataDir, layout.SessionHome("project"), vaultID.String, in.ProjectID, id)
	if err = os.MkdirAll(workspace, 0o700); err != nil {
		return out, err
	}
	if err = tx.Commit(); err != nil {
		_ = os.RemoveAll(workspace)
		return out, err
	}
	return out, nil
}

func (s *SessionStore) modelConfigured(provider, modelID string) bool {
	for _, model := range s.Models {
		if model.Provider == provider && model.ModelID == modelID {
			return true
		}
	}
	return false
}

const sessionSelect = `SELECT id,home,vault_id,project_id,status,provider,model_id,model_parameters_json,tool_grants_json,title,created_at,updated_at,deleted_at FROM sessions`

func (s *SessionStore) Get(ctx context.Context, id string) (domain.Session, error) {
	var out domain.Session
	err := scanSession(s.DB.QueryRowContext(ctx, sessionSelect+` WHERE id=?`, id), &out)
	if errors.Is(err, sql.ErrNoRows) {
		return out, ErrNotFound
	}
	return out, err
}

func (s *SessionStore) Delete(ctx context.Context, id string) error {
	if s.Locks != nil {
		unlock := s.Locks.Lock(id)
		defer unlock()
	}
	return s.withBarrier(func() error { return s.delete(ctx, id) })
}

// RenameTitle updates an active session's title.
func (s *SessionStore) RenameTitle(ctx context.Context, id, title string) (domain.Session, error) {
	var out domain.Session
	title = strings.TrimSpace(title)
	if title == "" {
		return out, ErrValidation
	}
	err := s.withBarrier(func() error {
		var e error
		out, e = s.renameTitle(ctx, id, title)
		return e
	})
	return out, err
}

func (s *SessionStore) renameTitle(ctx context.Context, id, title string) (domain.Session, error) {
	now := s.Now().UTC()
	result, err := s.DB.ExecContext(ctx,
		`UPDATE sessions SET title=?, updated_at=? WHERE id=? AND status='active' AND deleted_at IS NULL`,
		title, formatTime(now), id)
	if err != nil {
		return domain.Session{}, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return domain.Session{}, err
	}
	if rows != 1 {
		// Distinguish missing vs terminal/busy
		_, getErr := s.Get(ctx, id)
		if errors.Is(getErr, ErrNotFound) {
			return domain.Session{}, ErrNotFound
		}
		if getErr != nil {
			return domain.Session{}, getErr
		}
		return domain.Session{}, ErrNotFound
	}
	return s.Get(ctx, id)
}

func (s *SessionStore) delete(ctx context.Context, id string) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var home layout.SessionHome
	var vaultID, projectID sql.NullString
	var status string
	if err := tx.QueryRowContext(ctx, `SELECT home,vault_id,project_id,status FROM sessions WHERE id=?`, id).Scan(&home, &vaultID, &projectID, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	workspace := layout.SessionWorkspace(s.DataDir, home, nullableText(vaultID), nullableText(projectID), id)
	if status == "terminal" {
		if err := tx.Commit(); err != nil {
			return err
		}
		return os.RemoveAll(workspace)
	}
	var activeRun bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM agent_runs WHERE session_id=? AND status IN ('queued','running'))`, id).Scan(&activeRun); err != nil {
		return err
	}
	if activeRun {
		return ErrSessionBusy
	}

	now := s.Now().UTC()
	result, err := tx.ExecContext(ctx, `UPDATE sessions SET status='terminal',deleted_at=?,updated_at=? WHERE id=? AND status='active'`, formatTime(now), formatTime(now), id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrSessionBusy
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	// Tombstone is durable; workspace removal is best-effort under the session lock.
	// Failures leave the terminal tombstone so no new mutation can begin.
	return os.RemoveAll(workspace)
}

func nullableText(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func (s *SessionStore) ListByProject(ctx context.Context, projectID string) ([]domain.Session, error) {
	rows, err := s.DB.QueryContext(ctx, sessionSelect+` WHERE project_id=? ORDER BY created_at DESC,id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Session{}
	for rows.Next() {
		var session domain.Session
		if err := scanSession(rows, &session); err != nil {
			return nil, err
		}
		out = append(out, session)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(...any) error
}

func scanSession(row scanner, out *domain.Session) error {
	var vaultID, projectID, deletedAt sql.NullString
	var createdAt, updatedAt string
	if err := row.Scan(&out.ID, &out.Home, &vaultID, &projectID, &out.Status, &out.Provider, &out.ModelID,
		&out.ModelParametersJSON, &out.ToolGrantsJSON, &out.Title, &createdAt, &updatedAt, &deletedAt); err != nil {
		return err
	}
	if vaultID.Valid {
		out.VaultID = &vaultID.String
	}
	if projectID.Valid {
		out.ProjectID = &projectID.String
	}
	var err error
	if out.CreatedAt, err = parseTime(createdAt); err != nil {
		return err
	}
	if out.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return err
	}
	if deletedAt.Valid {
		value, err := parseTime(deletedAt.String)
		if err != nil {
			return err
		}
		out.DeletedAt = &value
	}
	return nil
}
