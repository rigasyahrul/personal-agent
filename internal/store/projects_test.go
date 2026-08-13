package store_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/store"
	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func TestVaultAndProjectCRUD(t *testing.T) {
	database, dataDir := testutil.TempDB(t)
	c := &clock.FakeClock{T: time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)}
	ctx := context.Background()

	vs := store.NewVaultStore(database, c)
	v, err := vs.Create(ctx, " Learning ")
	if err != nil {
		t.Fatal(err)
	}
	if v.Name != "Learning" || !v.CreatedAt.Equal(c.T) || !v.UpdatedAt.Equal(c.T) {
		t.Fatalf("vault = %+v", v)
	}
	vaults, err := vs.List(ctx)
	if err != nil || len(vaults) != 1 || vaults[0] != v {
		t.Fatalf("vaults = %+v, err = %v", vaults, err)
	}

	ps := store.NewProjectStore(database, dataDir, c)
	p, err := ps.Create(ctx, " Go ", v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if p.VaultID != v.ID || p.Name != "Go" {
		t.Fatalf("project = %+v", p)
	}
	for _, dir := range []string{"source", "memory", "soul"} {
		if info, err := os.Stat(filepath.Join(layout.ProjectRoot(dataDir, v.ID, p.ID), dir)); err != nil || !info.IsDir() {
			t.Fatalf("project directory %q: %v", dir, err)
		}
	}
	got, err := ps.Get(ctx, p.ID)
	if err != nil || got != p {
		t.Fatalf("Get() = %+v, %v", got, err)
	}
	list, err := ps.List(ctx)
	if err != nil || len(list) != 1 || list[0] != p {
		t.Fatalf("List() = %+v, %v", list, err)
	}
	if _, err := database.Exec(`UPDATE projects SET vault_id=NULL WHERE id=?`, p.ID); err == nil {
		t.Fatal("vault placement must be immutable")
	}
}

func TestStoresRejectBlankNamesAndUnknownVault(t *testing.T) {
	database, dataDir := testutil.TempDB(t)
	c := &clock.FakeClock{T: time.Now()}
	ctx := context.Background()

	if _, err := store.NewVaultStore(database, c).Create(ctx, " \t"); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("blank vault error = %v", err)
	}
	ps := store.NewProjectStore(database, dataDir, c)
	if _, err := ps.Create(ctx, " ", ""); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("blank project error = %v", err)
	}
	if _, err := ps.Create(ctx, "Bad", "missing"); !errors.Is(err, store.ErrInvalid) {
		t.Fatalf("unknown vault error = %v", err)
	}
}

func TestProjectWithoutVaultPersistsNullPlacement(t *testing.T) {
	database, dataDir := testutil.TempDB(t)
	c := &clock.FakeClock{T: time.Now()}
	p, err := store.NewProjectStore(database, dataDir, c).Create(context.Background(), "Global", "")
	if err != nil {
		t.Fatal(err)
	}
	var vaultID sql.NullString
	if err := database.QueryRow(`SELECT vault_id FROM projects WHERE id=?`, p.ID).Scan(&vaultID); err != nil {
		t.Fatal(err)
	}
	if vaultID.Valid || p.VaultID != "" {
		t.Fatalf("vault placement = %#v, project = %+v", vaultID, p)
	}
}
