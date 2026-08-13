package layout

import (
	"fmt"
	"os"
	"path/filepath"
)

type SessionHome string

func ProjectRoot(dataDir, vaultID, projectID string) string {
	if vaultID == "" {
		return filepath.Join(dataDir, "files", "global", "projects", projectID)
	}
	return filepath.Join(dataDir, "files", "vaults", vaultID, "projects", projectID)
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
	for _, name := range []string{"source", "memory", "soul"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0700); err != nil {
			return fmt.Errorf("create project %s: %w", name, err)
		}
	}
	return nil
}
