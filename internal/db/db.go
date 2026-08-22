package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

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

	if err = applyPendingMigrations(ctx, d); err != nil {
		return closeWithError(err)
	}

	return d, nil
}

// applyPendingMigrations applies every embedded migrations/*.sql whose leading
// version digits are not yet recorded in schema_migrations, in sorted filename order.
func applyPendingMigrations(ctx context.Context, d *sql.DB) error {
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		version, ok := migrationVersion(name)
		if !ok {
			return fmt.Errorf("migration %s: missing leading version digits", name)
		}
		var applied int
		if err := d.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations WHERE version=?`, version).Scan(&applied); err != nil {
			return err
		}
		if applied > 0 {
			continue
		}

		body, readErr := migrations.ReadFile(path.Join("migrations", name))
		if readErr != nil {
			return readErr
		}

		tx, migrationErr := d.BeginTx(ctx, nil)
		if migrationErr == nil {
			_, migrationErr = tx.ExecContext(ctx, string(body))
		}
		if migrationErr == nil {
			_, migrationErr = tx.ExecContext(ctx, `INSERT INTO schema_migrations VALUES(?,strftime('%Y-%m-%dT%H:%M:%fZ','now'))`, version)
		}
		if migrationErr == nil {
			migrationErr = tx.Commit()
		} else if tx != nil {
			_ = tx.Rollback()
		}
		if migrationErr != nil {
			return fmt.Errorf("migration %s: %w", version, migrationErr)
		}
	}
	return nil
}

// migrationVersion returns the leading digit run of a migration filename (e.g. "001_init.sql" → "001").
func migrationVersion(filename string) (string, bool) {
	base := path.Base(filename)
	i := 0
	for i < len(base) && unicode.IsDigit(rune(base[i])) {
		i++
	}
	if i == 0 {
		return "", false
	}
	return base[:i], true
}
