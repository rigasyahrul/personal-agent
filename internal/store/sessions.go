package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/config"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/ids"
	"github.com/rigasyahrul/personal-agent/internal/layout"
)

const defaultToolGrantsJSON = `{"workspace_files":false}`

type SessionStore struct {
	DB      *sql.DB
	DataDir string
	Now     func() time.Time
	Models  []config.ModelRef
}

type CreateSessionInput struct {
	ProjectID, Title, Provider, ModelID, ModelParametersJSON, ToolGrantsJSON string
}

func (s *SessionStore) CreateProject(ctx context.Context, in CreateSessionInput) (domain.Session, error) {
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
	err = scanSession(s.DB.QueryRowContext(ctx, sessionSelect+` WHERE id=?`, id), &out)
	return out, err
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
