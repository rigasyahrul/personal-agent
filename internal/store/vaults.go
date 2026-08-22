package store

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/agent/skills"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/ids"
	"github.com/rigasyahrul/personal-agent/internal/layout"
)

type VaultStore struct {
	db      *sql.DB
	dataDir string
	clock   clock.Clock
}

func NewVaultStore(db *sql.DB, dataDir string, c clock.Clock) *VaultStore {
	return &VaultStore{db: db, dataDir: dataDir, clock: c}
}

func (s *VaultStore) Create(ctx context.Context, name string) (domain.Vault, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return domain.Vault{}, ErrValidation
	}
	now := s.clock.Now().UTC()
	v := domain.Vault{ID: ids.NewID(), Name: name, CreatedAt: now, UpdatedAt: now}
	_, err := s.db.ExecContext(ctx, `INSERT INTO vaults(id,name,created_at,updated_at) VALUES(?,?,?,?)`, v.ID, v.Name, formatTime(v.CreatedAt), formatTime(v.UpdatedAt))
	if err != nil {
		return domain.Vault{}, err
	}
	if err := layout.EnsureVaultKnowledgeDirs(s.dataDir, v.ID, skills.DefaultCompoundingSkillMarkdown()); err != nil {
		// Roll back vault row so create is atomic with disk seed.
		_, _ = s.db.ExecContext(ctx, `DELETE FROM vaults WHERE id=?`, v.ID)
		return domain.Vault{}, err
	}
	return v, nil
}

func (s *VaultStore) List(ctx context.Context) ([]domain.Vault, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,created_at,updated_at FROM vaults ORDER BY name,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Vault{}
	for rows.Next() {
		var v domain.Vault
		var createdAt, updatedAt string
		if err := rows.Scan(&v.ID, &v.Name, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if v.CreatedAt, err = parseTime(createdAt); err != nil {
			return nil, err
		}
		if v.UpdatedAt, err = parseTime(updatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }

func parseTime(value string) (time.Time, error) { return time.Parse(time.RFC3339Nano, value) }
