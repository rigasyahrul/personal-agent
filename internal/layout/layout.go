package layout

import (
	"fmt"
	"os"
	"path/filepath"
)

type SessionHome string

var mkdirProjectChild = os.Mkdir

func ProjectRoot(dataDir, vaultID, projectID string) string {
	if vaultID == "" {
		return filepath.Join(dataDir, "files", "global", "projects", projectID)
	}
	return filepath.Join(dataDir, "files", "vaults", vaultID, "projects", projectID)
}

// GlobalRoot is dataDir/files/global.
func GlobalRoot(dataDir string) string {
	return filepath.Join(dataDir, "files", "global")
}

// VaultRoot is dataDir/files/vaults/{vaultID}.
func VaultRoot(dataDir, vaultID string) string {
	return filepath.Join(dataDir, "files", "vaults", vaultID)
}

// InstructionPath joins scopeRoot with name (e.g. SOUL.md, SYSTEM.md, AGENTS.md).
// Callers validate name; this helper does not.
func InstructionPath(scopeRoot, name string) string {
	return filepath.Join(scopeRoot, name)
}

// MemoryDir is scopeRoot/memory.
func MemoryDir(scopeRoot string) string {
	return filepath.Join(scopeRoot, "memory")
}

// LessonsPath is scopeRoot/memory/lessons.md.
func LessonsPath(scopeRoot string) string {
	return filepath.Join(scopeRoot, "memory", "lessons.md")
}

// AgentsSkillsDir is scopeRoot/.agents/skills.
func AgentsSkillsDir(scopeRoot string) string {
	return filepath.Join(scopeRoot, ".agents", "skills")
}

// CompoundingSkillPath is scopeRoot/.agents/skills/compounding/SKILL.md.
func CompoundingSkillPath(scopeRoot string) string {
	return filepath.Join(scopeRoot, ".agents", "skills", "compounding", "SKILL.md")
}

func SourceDir(projectRoot string) string { return filepath.Join(projectRoot, "source") }

func SessionWorkspace(dataDir string, home SessionHome, vaultID, projectID, sessionID string) string {
	switch home {
	case "global":
		return filepath.Join(dataDir, "files", "global", "sessions", sessionID)
	case "vault":
		return filepath.Join(dataDir, "files", "vaults", vaultID, "sessions", sessionID)
	case "project":
		return filepath.Join(ProjectRoot(dataDir, vaultID, projectID), "sessions", sessionID)
	default:
		panic(fmt.Sprintf("invalid session home %q", home))
	}
}

func EnsureProjectDirs(dataDir, vaultID, projectID string) error {
	root := ProjectRoot(dataDir, vaultID, projectID)
	if err := os.MkdirAll(filepath.Dir(root), 0700); err != nil {
		return fmt.Errorf("create project parent: %w", err)
	}
	if err := os.Mkdir(root, 0700); err != nil {
		return fmt.Errorf("create fresh project root: %w", err)
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(root)
		}
	}()
	for _, name := range []string{"source", "memory", "soul"} {
		if err := mkdirProjectChild(filepath.Join(root, name), 0700); err != nil {
			return fmt.Errorf("create project %s: %w", name, err)
		}
	}
	complete = true
	return nil
}
