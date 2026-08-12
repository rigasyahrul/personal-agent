package testutil

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	database "github.com/rigasyahrul/personal-agent/internal/db"
)

func OpenDB(t *testing.T, dataDir string) *sql.DB {
	t.Helper()
	d, err := database.Open(context.Background(), filepath.Join(dataDir, "db", "personal-agent.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func TempDB(t *testing.T) (db *sql.DB, dataDir string) {
	t.Helper()
	dataDir = t.TempDir()
	return OpenDB(t, dataDir), dataDir
}
