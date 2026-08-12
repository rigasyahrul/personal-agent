package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenDBUsesCanonicalPath(t *testing.T) {
	dataDir := t.TempDir()
	d := OpenDB(t, dataDir)
	if err := d.Ping(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "db", "personal-agent.sqlite")); err != nil {
		t.Fatal(err)
	}
}

func TestTempDBReturnsDataDir(t *testing.T) {
	d, dataDir := TempDB(t)
	if d == nil || dataDir == "" {
		t.Fatal("TempDB returned an empty result")
	}
	if _, err := os.Stat(filepath.Join(dataDir, "db", "personal-agent.sqlite")); err != nil {
		t.Fatal(err)
	}
}
