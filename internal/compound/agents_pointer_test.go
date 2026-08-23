package compound_test

import (
	"strings"
	"testing"

	"github.com/rigasyahrul/personal-agent/internal/compound"
)

// Exact Canonical AGENTS Memory block (must match layout wantAgentsMemoryBlock).
const wantCanonicalMemoryBlock = "## Memory\n" +
	"- Lesson index: [[memory/lessons|lessons.md]] — scan titles when stuck or before reinventing a fix.\n" +
	"- Detail files live under `memory/YYYYMMDD-HHmm-*.md`; open only what the index suggests.\n" +
	"- Prefer codifying durable rules here; keep evidence in memory (compound ≠ diary).\n"

func TestEnsureAgentsMemoryPointer_MissingSectionAppendsCanonical(t *testing.T) {
	in := "# Agents\n\nrules here\n"
	got, err := compound.EnsureAgentsMemoryPointer(in)
	if err != nil {
		t.Fatalf("EnsureAgentsMemoryPointer: %v", err)
	}
	if !strings.HasPrefix(got, in) {
		t.Fatalf("must keep original content as prefix:\n got %q\nwant prefix %q", got, in)
	}
	if !strings.Contains(got, wantCanonicalMemoryBlock) {
		t.Fatalf("missing exact Canonical Memory block:\n%s", got)
	}
	if strings.Count(got, wantCanonicalMemoryBlock) != 1 {
		t.Fatalf("Canonical block should appear once:\n%s", got)
	}
	if err := compound.ValidateAgentsMemoryPointer(got); err != nil {
		t.Fatalf("ensured content must validate: %v", err)
	}
}

func TestEnsureAgentsMemoryPointer_PresentPointerUnchanged(t *testing.T) {
	in := "# X\n\n## Memory\n- [[memory/lessons|lessons.md]] custom note\n"
	got, err := compound.EnsureAgentsMemoryPointer(in)
	if err != nil {
		t.Fatalf("EnsureAgentsMemoryPointer: %v", err)
	}
	if got != in {
		t.Fatalf("present pointer must be unchanged\n got %q\nwant %q", got, in)
	}
}

func TestValidateAgentsMemoryPointer(t *testing.T) {
	ok := "# X\n\n## Memory\n- [[memory/lessons|lessons.md]]\n"
	if err := compound.ValidateAgentsMemoryPointer(ok); err != nil {
		t.Fatalf("ok: %v", err)
	}
	if err := compound.ValidateAgentsMemoryPointer("# X\nno pointer\n"); err == nil {
		t.Fatal("expected missing Memory to fail")
	}
	if err := compound.ValidateAgentsMemoryPointer("## Memory\nno wikilink\n"); err == nil {
		t.Fatal("expected missing lessons wikilink to fail")
	}
}
