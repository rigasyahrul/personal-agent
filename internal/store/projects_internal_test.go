package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func TestProjectCreateCommitFailureCleansOnlyFreshRoot(t *testing.T) {
	db, d := testutil.TempDB(t)
	s := NewProjectStore(db, d, &clock.FakeClock{T: time.Now()})
	s.commit = func(*sql.Tx) error { return errors.New("injected commit failure") }
	p, err := s.Create(context.Background(), "P", "")
	if err == nil {
		t.Fatal("expected commit failure")
	}
	if p.ID != "" {
		t.Fatalf("returned project: %+v", p)
	}
	projectsDir := layout.ProjectRoot(d, "", "")
	entries, readErr := os.ReadDir(projectsDir)
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("fresh project root remains: %v", entries)
	}
}
