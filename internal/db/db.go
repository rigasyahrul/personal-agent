package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Open(ctx context.Context, file string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return nil, err
	}
	d, err := sql.Open("sqlite", file)
	if err != nil {
		return nil, err
	}
	d.SetMaxOpenConns(1)
	closeWithError := func(err error) (*sql.DB, error) {
		_ = d.Close()
		return nil, err
	}

	for _, statement := range []string{
		"PRAGMA foreign_keys=ON",
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err = d.ExecContext(ctx, statement); err != nil {
			return closeWithError(err)
		}
	}
	if _, err = d.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return closeWithError(err)
	}

	var applied int
	if err = d.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations WHERE version='001'`).Scan(&applied); err != nil {
		return closeWithError(err)
	}
	if applied == 0 {
		migration, readErr := migrations.ReadFile("migrations/001_init.sql")
		if readErr != nil {
			return closeWithError(readErr)
		}
		tx, migrationErr := d.BeginTx(ctx, nil)
		if migrationErr == nil {
			_, migrationErr = tx.ExecContext(ctx, string(migration))
		}
		if migrationErr == nil {
			_, migrationErr = tx.ExecContext(ctx, `INSERT INTO schema_migrations VALUES('001',strftime('%Y-%m-%dT%H:%M:%fZ','now'))`)
		}
		if migrationErr == nil {
			migrationErr = tx.Commit()
		} else if tx != nil {
			_ = tx.Rollback()
		}
		if migrationErr != nil {
			return closeWithError(fmt.Errorf("migration 001: %w", migrationErr))
		}
	}

	return d, nil
}
