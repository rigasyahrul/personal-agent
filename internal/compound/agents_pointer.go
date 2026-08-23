package compound

import (
	"fmt"
	"strings"

	"github.com/rigasyahrul/personal-agent/internal/store"
)

// AgentsMemoryMarker is the required AGENTS.md section heading.
const AgentsMemoryMarker = "## Memory"

// canonicalAgentsMemoryBlock is the Canonical AGENTS Memory section (exact
// markdown; must match layout.defaultAgentsMemoryBlock / layout tests).
const canonicalAgentsMemoryBlock = "## Memory\n" +
	"- Lesson index: [[memory/lessons|lessons.md]] — scan titles when stuck or before reinventing a fix.\n" +
	"- Detail files live under `memory/YYYYMMDD-HHmm-*.md`; open only what the index suggests.\n" +
	"- Prefer codifying durable rules here; keep evidence in memory (compound ≠ diary).\n"

// EnsureAgentsMemoryPointer appends the Canonical Memory block when the
// ## Memory section is missing. An existing Memory section is left unchanged.
func EnsureAgentsMemoryPointer(content string) (string, error) {
	if strings.Contains(content, AgentsMemoryMarker) {
		return content, nil
	}
	if content == "" {
		return canonicalAgentsMemoryBlock, nil
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + "\n" + canonicalAgentsMemoryBlock, nil
}

// ValidateAgentsMemoryPointer requires the Canonical Memory section + lessons wikilink.
func ValidateAgentsMemoryPointer(content string) error {
	if !strings.Contains(content, AgentsMemoryMarker) {
		return fmt.Errorf("%w: AGENTS.md must keep a %s section", store.ErrValidation, AgentsMemoryMarker)
	}
	if !strings.Contains(content, "[[memory/lessons") {
		return fmt.Errorf("%w: AGENTS.md must keep [[memory/lessons pointer", store.ErrValidation)
	}
	return nil
}
