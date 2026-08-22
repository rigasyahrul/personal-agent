package skills

import (
	"strings"
	"testing"
)

func TestDefaultCompoundingSkillEmbedded(t *testing.T) {
	s := DefaultCompoundingSkillMarkdown()
	for _, need := range []string{"codify", "lessons.md", "agents_patch", "memory_detail", "diary"} {
		if !strings.Contains(strings.ToLower(s), strings.ToLower(need)) {
			t.Fatalf("missing %q in skill", need)
		}
	}
	if len(s) < 400 {
		t.Fatalf("skill too short: %d", len(s))
	}
	// Additional natural assertions from plan/spec §14
	for _, need := range []string{"lessons_index_row", "Memory"} {
		if !strings.Contains(s, need) {
			t.Fatalf("missing %q in skill", need)
		}
	}
}
