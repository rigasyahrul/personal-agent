package agent

import (
	"fmt"
	"os"
	"strings"

	"github.com/rigasyahrul/personal-agent/internal/agent/skills"
	"github.com/rigasyahrul/personal-agent/internal/layout"
)

// LoadCompoundingSkill returns the compounding skill markdown for the session scope.
// On missing or empty file it returns the embedded default and source "embedded".
func LoadCompoundingSkill(dataDir string, home layout.SessionHome, vaultID, projectID string) (string, string, error) {
	skillRoot, err := compoundingSkillRoot(dataDir, home, vaultID, projectID)
	if err != nil {
		return "", "", err
	}
	path := layout.CompoundingSkillPath(skillRoot)
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return skills.DefaultCompoundingSkillMarkdown(), "embedded", nil
		}
		return "", "", err
	}
	if strings.TrimSpace(string(body)) == "" {
		return skills.DefaultCompoundingSkillMarkdown(), "embedded", nil
	}
	return string(body), path, nil
}

func compoundingSkillRoot(dataDir string, home layout.SessionHome, vaultID, projectID string) (string, error) {
	// Skill root matches memory root (Canonical table).
	switch home {
	case layout.SessionHome("project"):
		return layout.ProjectRoot(dataDir, vaultID, projectID), nil
	case layout.SessionHome("vault"):
		return layout.VaultRoot(dataDir, vaultID), nil
	case layout.SessionHome("global"):
		return layout.GlobalRoot(dataDir), nil
	default:
		return "", fmt.Errorf("invalid session home %q", home)
	}
}
