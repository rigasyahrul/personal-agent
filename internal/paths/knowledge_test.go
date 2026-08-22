package paths

import (
	"errors"
	"testing"
)

func TestValidateKnowledgeRelPath(t *testing.T) {
	t.Parallel()

	t.Run("allows instruction files", func(t *testing.T) {
		for _, p := range []string{"AGENTS.md", "SOUL.md", "SYSTEM.md"} {
			if err := ValidateKnowledgeRelPath(p); err != nil {
				t.Fatalf("ValidateKnowledgeRelPath(%q) = %v, want nil", p, err)
			}
		}
	})

	t.Run("allows memory and source markdown", func(t *testing.T) {
		for _, p := range []string{
			"memory/lessons.md",
			"memory/20260822-1200-slug.md",
			"source/a.md",
			"source/articles/intro.md",
		} {
			if err := ValidateKnowledgeRelPath(p); err != nil {
				t.Fatalf("ValidateKnowledgeRelPath(%q) = %v, want nil", p, err)
			}
		}
	})

	t.Run("rejects unsafe and reserved paths", func(t *testing.T) {
		bad := []string{
			"",
			"..",
			"../x",
			"/abs.md",
			".agents/x",
			".agents/skills/compounding/SKILL.md",
			"sessions/s1/file.md",
			"soul/x.md",
			"memory",
			"source",
			"memory/../escape.md",
			"notes/a.md",
			"AGENTS",
			"agents.md",
			"memory/not-md.txt",
		}
		for _, p := range bad {
			if err := ValidateKnowledgeRelPath(p); err == nil {
				t.Fatalf("ValidateKnowledgeRelPath(%q) = nil, want error", p)
			}
		}
	})
}

func TestValidateRelPathStillRejectsMemoryLessons(t *testing.T) {
	t.Parallel()
	// Promote/source validator must keep rejecting memory components — never loosen.
	_, err := ValidateRelPath("memory/lessons.md")
	var pe *PathError
	if !errors.As(err, &pe) || pe.Code != "reserved_path" {
		t.Fatalf("ValidateRelPath(memory/lessons.md) = %v, want reserved_path", err)
	}
}
