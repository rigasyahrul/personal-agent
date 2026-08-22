package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rigasyahrul/personal-agent/internal/agent/skills"
	"github.com/rigasyahrul/personal-agent/internal/layout"
)

func TestLoadCompoundingSkill_MissingUsesEmbedded(t *testing.T) {
	dataDir := t.TempDir()
	md, src, err := LoadCompoundingSkill(dataDir, layout.SessionHome("global"), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if src != "embedded" {
		t.Fatalf("source = %q want embedded", src)
	}
	want := skills.DefaultCompoundingSkillMarkdown()
	if md != want {
		t.Fatalf("markdown mismatch (len %d vs %d)", len(md), len(want))
	}
}

func TestLoadCompoundingSkill_EmptyFileUsesEmbedded(t *testing.T) {
	dataDir := t.TempDir()
	root := layout.GlobalRoot(dataDir)
	path := layout.CompoundingSkillPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("   \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	md, src, err := LoadCompoundingSkill(dataDir, layout.SessionHome("global"), "", "")
	if err != nil {
		t.Fatal(err)
	}
	if src != "embedded" {
		t.Fatalf("source = %q want embedded", src)
	}
	if md != skills.DefaultCompoundingSkillMarkdown() {
		t.Fatal("expected embedded markdown for empty file")
	}
}

func TestLoadCompoundingSkill_ProjectReadsDisk(t *testing.T) {
	dataDir := t.TempDir()
	root := layout.ProjectRoot(dataDir, "v1", "p1")
	path := layout.CompoundingSkillPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	body := "# Custom compounding\n\nOnly this project.\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	md, src, err := LoadCompoundingSkill(dataDir, layout.SessionHome("project"), "v1", "p1")
	if err != nil {
		t.Fatal(err)
	}
	if src != path {
		t.Fatalf("source = %q want %q", src, path)
	}
	if md != body {
		t.Fatalf("got %q", md)
	}
}

func TestLoadCompoundingSkill_VaultUsesVaultRootNotGlobal(t *testing.T) {
	dataDir := t.TempDir()
	globalPath := layout.CompoundingSkillPath(layout.GlobalRoot(dataDir))
	vaultPath := layout.CompoundingSkillPath(layout.VaultRoot(dataDir, "v1"))
	if err := os.MkdirAll(filepath.Dir(globalPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(vaultPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(globalPath, []byte("GLOBAL SKILL\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(vaultPath, []byte("VAULT SKILL\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	md, src, err := LoadCompoundingSkill(dataDir, layout.SessionHome("vault"), "v1", "")
	if err != nil {
		t.Fatal(err)
	}
	if src != vaultPath {
		t.Fatalf("source = %q want vault path %q", src, vaultPath)
	}
	if !strings.Contains(md, "VAULT SKILL") {
		t.Fatalf("got %q", md)
	}
}

func TestLoadCompoundingSkill_InvalidHome(t *testing.T) {
	_, _, err := LoadCompoundingSkill(t.TempDir(), layout.SessionHome("nope"), "", "")
	if err == nil {
		t.Fatal("expected error")
	}
}
