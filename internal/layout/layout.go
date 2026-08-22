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

// defaultAgentsMemoryBlock is the Canonical AGENTS Memory section (exact markdown).
const defaultAgentsMemoryBlock = "## Memory\n" +
	"- Lesson index: [[memory/lessons|lessons.md]] — scan titles when stuck or before reinventing a fix.\n" +
	"- Detail files live under `memory/YYYYMMDD-HHmm-*.md`; open only what the index suggests.\n" +
	"- Prefer codifying durable rules here; keep evidence in memory (compound ≠ diary).\n"

// lessonsScaffold is the thin index seed for memory/lessons.md.
const lessonsScaffold = "# Lessons\n" +
	"\n" +
	"> Thin index only. Detail files: `memory/YYYYMMDD-HHmm-slug.md`.\n" +
	"\n"

// EnsureGlobalKnowledgeDirs seeds GlobalRoot with instructions, memory, and compounding skill.
// Existing files are left unchanged (idempotent).
func EnsureGlobalKnowledgeDirs(dataDir string, skillMarkdown string) error {
	return ensureKnowledgeScope(GlobalRoot(dataDir), skillMarkdown, true)
}

// EnsureVaultKnowledgeDirs seeds VaultRoot with memory + compounding skill only.
// Per Canonical: no vault SOUL/SYSTEM/AGENTS.
func EnsureVaultKnowledgeDirs(dataDir, vaultID string, skillMarkdown string) error {
	return ensureKnowledgeScope(VaultRoot(dataDir, vaultID), skillMarkdown, false)
}

// EnsureProjectKnowledge seeds ProjectRoot with instructions, memory, and compounding skill.
// Call after EnsureProjectDirs. Existing files are left unchanged (idempotent).
func EnsureProjectKnowledge(dataDir, vaultID, projectID string, skillMarkdown string) error {
	return ensureKnowledgeScope(ProjectRoot(dataDir, vaultID, projectID), skillMarkdown, true)
}

// ensureKnowledgeScope creates memory/ and .agents/skills/compounding/, seeds
// skill + lessons when missing, and optionally seeds SOUL/SYSTEM/AGENTS.
func ensureKnowledgeScope(scopeRoot string, skillMarkdown string, withInstructions bool) error {
	if err := os.MkdirAll(MemoryDir(scopeRoot), 0700); err != nil {
		return fmt.Errorf("create memory dir: %w", err)
	}
	skillDir := filepath.Join(AgentsSkillsDir(scopeRoot), "compounding")
	if err := os.MkdirAll(skillDir, 0700); err != nil {
		return fmt.Errorf("create compounding skill dir: %w", err)
	}

	if err := writeFileIfMissing(CompoundingSkillPath(scopeRoot), []byte(skillMarkdown), 0600); err != nil {
		return fmt.Errorf("seed skill: %w", err)
	}
	if err := writeFileIfMissing(LessonsPath(scopeRoot), []byte(lessonsScaffold), 0600); err != nil {
		return fmt.Errorf("seed lessons: %w", err)
	}

	if withInstructions {
		if err := writeFileIfMissing(InstructionPath(scopeRoot, "SOUL.md"), []byte("\n"), 0600); err != nil {
			return fmt.Errorf("seed SOUL.md: %w", err)
		}
		if err := writeFileIfMissing(InstructionPath(scopeRoot, "SYSTEM.md"), []byte("\n"), 0600); err != nil {
			return fmt.Errorf("seed SYSTEM.md: %w", err)
		}
		if err := writeFileIfMissing(InstructionPath(scopeRoot, "AGENTS.md"), []byte(defaultAgentsMemoryBlock), 0600); err != nil {
			return fmt.Errorf("seed AGENTS.md: %w", err)
		}
	}
	return nil
}

// writeFileIfMissing creates path with content only when the file does not exist.
func writeFileIfMissing(path string, content []byte, perm os.FileMode) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	if _, err := f.Write(content); err != nil {
		return err
	}
	return nil
}
