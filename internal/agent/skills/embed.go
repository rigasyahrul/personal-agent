package skills

import _ "embed"

//go:embed compounding/SKILL.md
var defaultCompoundingSkill string

// DefaultCompoundingSkillMarkdown returns the embedded default compounding skill body.
// Seeded into each scope's .agents/skills/compounding/SKILL.md on create; also used
// as runtime fallback when the on-disk skill is missing.
func DefaultCompoundingSkillMarkdown() string {
	return defaultCompoundingSkill
}
