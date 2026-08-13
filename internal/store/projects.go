package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/ids"
	"github.com/rigasyahrul/personal-agent/internal/layout"
)

type ProjectStore struct {
	db      *sql.DB
	dataDir string
	clock   clock.Clock
}

func NewProjectStore(db *sql.DB, dataDir string, c clock.Clock) *ProjectStore {
	return &ProjectStore{db: db, dataDir: dataDir, clock: c}
}

func (s *ProjectStore) ReadyNoteCount(ctx context.Context, projectID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notes WHERE project_id=? AND status='ready'`, projectID).Scan(&count)
	return count, err
}

func (s *ProjectStore) Create(ctx context.Context, name, vaultID string) (domain.Project, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Project{}, ErrValidation
	}
	if vaultID != "" {
		var exists bool
		if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM vaults WHERE id=?)`, vaultID).Scan(&exists); err != nil {
			return domain.Project{}, err
		}
		if !exists {
			return domain.Project{}, ErrValidation
		}
	}

	now := s.clock.Now().UTC()
	p := domain.Project{ID: ids.NewID(), VaultID: vaultID, Name: name, CreatedAt: now, UpdatedAt: now}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Project{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `INSERT INTO projects(id,vault_id,name,created_at,updated_at) VALUES(?,?,?,?,?)`, p.ID, nullable(vaultID), p.Name, formatTime(p.CreatedAt), formatTime(p.UpdatedAt)); err != nil {
		return domain.Project{}, err
	}
	if err = layout.EnsureProjectDirs(s.dataDir, vaultID, p.ID); err != nil {
		return domain.Project{}, err
	}
	if err = tx.Commit(); err != nil {
		return domain.Project{}, err
	}
	return p, nil
}

func scanProject(row interface{ Scan(...any) error }) (domain.Project, error) {
	var p domain.Project
	var vaultID sql.NullString
	var createdAt, updatedAt string
	if err := row.Scan(&p.ID, &vaultID, &p.Name, &createdAt, &updatedAt); err != nil {
		return domain.Project{}, err
	}
	p.VaultID = vaultID.String
	var err error
	if p.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.Project{}, err
	}
	if p.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.Project{}, err
	}
	return p, nil
}

func (s *ProjectStore) Get(ctx context.Context, id string) (domain.Project, error) {
	p, err := scanProject(s.db.QueryRowContext(ctx, `SELECT id,vault_id,name,created_at,updated_at FROM projects WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Project{}, ErrNotFound
	}
	return p, err
}

func (s *ProjectStore) List(ctx context.Context) ([]domain.Project, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,vault_id,name,created_at,updated_at FROM projects ORDER BY updated_at DESC,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Project{}
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
