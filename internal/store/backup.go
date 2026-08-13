package store

import (
	"context"
	"database/sql"

	"github.com/rigasyahrul/personal-agent/internal/domain"
)

func CreateBackupRun(ctx context.Context, db *sql.DB, r domain.BackupRun) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO backup_runs(id,status,cutoff_at,started_at) VALUES(?,?,?,?)`,
		r.ID, r.Status, r.CutoffAt, r.StartedAt,
	)
	return err
}

func CompleteBackupRun(ctx context.Context, db *sql.DB, r domain.BackupRun) error {
	_, err := db.ExecContext(ctx,
		`UPDATE backup_runs SET status=?,local_path=NULLIF(?,''),object_key=NULLIF(?,''),manifest_hash=NULLIF(?,''),completed_at=NULLIF(?,''),error=NULLIF(?,'') WHERE id=?`,
		r.Status, r.LocalPath, r.ObjectKey, r.ManifestHash, r.CompletedAt, r.Error, r.ID,
	)
	return err
}

func ListBackupRuns(ctx context.Context, db *sql.DB) ([]domain.BackupRun, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT id,status,cutoff_at,COALESCE(local_path,''),COALESCE(object_key,''),COALESCE(manifest_hash,''),started_at,COALESCE(completed_at,''),COALESCE(error,'') FROM backup_runs ORDER BY started_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.BackupRun
	for rows.Next() {
		var r domain.BackupRun
		if err := rows.Scan(&r.ID, &r.Status, &r.CutoffAt, &r.LocalPath, &r.ObjectKey, &r.ManifestHash, &r.StartedAt, &r.CompletedAt, &r.Error); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
