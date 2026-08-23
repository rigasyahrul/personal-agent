package knowledge

import (
	"strings"
	"testing"
)

func TestParseWikilinks_MemoryAliasAppendsMd(t *testing.T) {
	t.Parallel()

	got := ParseWikilinks("See [[memory/a|Title]] for the lesson.")
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(got), got)
	}
	if got[0].RawTarget != "memory/a" {
		t.Fatalf("RawTarget = %q, want memory/a", got[0].RawTarget)
	}
	if got[0].Alias != "Title" {
		t.Fatalf("Alias = %q, want Title", got[0].Alias)
	}
	if got[0].NormalizedPath != "memory/a.md" {
		t.Fatalf("NormalizedPath = %q, want memory/a.md", got[0].NormalizedPath)
	}
}

func TestParseWikilinks_KeepsExistingMdSuffix(t *testing.T) {
	t.Parallel()

	got := ParseWikilinks("[[source/x.md]]")
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %#v", len(got), got)
	}
	if got[0].RawTarget != "source/x.md" {
		t.Fatalf("RawTarget = %q, want source/x.md", got[0].RawTarget)
	}
	if got[0].Alias != "" {
		t.Fatalf("Alias = %q, want empty", got[0].Alias)
	}
	if got[0].NormalizedPath != "source/x.md" {
		t.Fatalf("NormalizedPath = %q, want source/x.md (do not strip .md)", got[0].NormalizedPath)
	}
}

func TestParseWikilinks_BareInstructionStems(t *testing.T) {
	t.Parallel()

	cases := []struct {
		body, want string
	}{
		{"[[AGENTS]]", "AGENTS.md"},
		{"[[SOUL]]", "SOUL.md"},
		{"[[SYSTEM]]", "SYSTEM.md"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			got := ParseWikilinks(tc.body)
			if len(got) != 1 {
				t.Fatalf("len = %d, want 1: %#v", len(got), got)
			}
			if got[0].NormalizedPath != tc.want {
				t.Fatalf("NormalizedPath = %q, want %q", got[0].NormalizedPath, tc.want)
			}
			if got[0].Alias != "" {
				t.Fatalf("Alias = %q, want empty", got[0].Alias)
			}
		})
	}
}

func TestParseWikilinks_SkipsInvalidTargets(t *testing.T) {
	t.Parallel()

	body := "keep [[memory/a|Title]] skip [[../x]] keep [[source/x.md]] skip [[/abs]] skip [[]]"
	got := ParseWikilinks(body)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 valid links: %#v", len(got), got)
	}
	if got[0].NormalizedPath != "memory/a.md" || got[0].Alias != "Title" {
		t.Fatalf("first = %+v, want memory/a.md alias Title", got[0])
	}
	if got[1].NormalizedPath != "source/x.md" {
		t.Fatalf("second = %+v, want source/x.md", got[1])
	}
}

func TestParseWikilinks_AliasDoesNotAffectJoinKey(t *testing.T) {
	t.Parallel()

	plain := ParseWikilinks("[[memory/a]]")
	aliased := ParseWikilinks("[[memory/a|Totally Different Title]]")
	if len(plain) != 1 || len(aliased) != 1 {
		t.Fatalf("plain=%#v aliased=%#v", plain, aliased)
	}
	if plain[0].NormalizedPath != aliased[0].NormalizedPath {
		t.Fatalf("join key changed by alias: %q vs %q", plain[0].NormalizedPath, aliased[0].NormalizedPath)
	}
	if aliased[0].NormalizedPath != "memory/a.md" {
		t.Fatalf("NormalizedPath = %q, want memory/a.md", aliased[0].NormalizedPath)
	}
}

func TestNormalizeWikilinkTarget_AppendsMdAndTrims(t *testing.T) {
	t.Parallel()

	got, err := NormalizeWikilinkTarget("  memory/a  ")
	if err != nil {
		t.Fatalf("NormalizeWikilinkTarget: %v", err)
	}
	if got != "memory/a.md" {
		t.Fatalf("got %q, want memory/a.md", got)
	}
}

func TestNormalizeWikilinkTarget_RejectsHostile(t *testing.T) {
	t.Parallel()

	cases := []string{
		"../x",
		"memory/../x",
		"/abs",
		"",
		"   ",
		"mem\x00ory/a",
	}
	for _, in := range cases {
		t.Run(strings.ReplaceAll(in, "\x00", "NUL"), func(t *testing.T) {
			t.Parallel()
			if _, err := NormalizeWikilinkTarget(in); err == nil {
				t.Fatalf("NormalizeWikilinkTarget(%q) = nil, want error", in)
			}
		})
	}
}
