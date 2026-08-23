package knowledge

import (
	"strings"
	"testing"
)

func TestSplitFrontmatter_MissingReturnsEmptyAndFullBody(t *testing.T) {
	t.Parallel()

	const md = "Just a note\nwith no frontmatter.\n"
	fm, body, err := SplitFrontmatter(md)
	if err != nil {
		t.Fatalf("SplitFrontmatter: %v", err)
	}
	if fm.Title != "" || fm.Date != "" || len(fm.Tags) != 0 || len(fm.CodifiedInto) != 0 || fm.Raw != nil {
		t.Fatalf("missing frontmatter must be empty, got %+v", fm)
	}
	if body != md {
		t.Fatalf("body = %q, want full original markdown", body)
	}
}

func TestSplitFrontmatter_ParsesYAMLFieldsAndBody(t *testing.T) {
	t.Parallel()

	md := "---\n" +
		"title: Hub startSession soft-fail\n" +
		"date: 2026-08-20\n" +
		"tags: [hub, sessions]\n" +
		"codified_into: [AGENTS.md]\n" +
		"---\n" +
		"Lesson body\n"
	fm, body, err := SplitFrontmatter(md)
	if err != nil {
		t.Fatalf("SplitFrontmatter: %v", err)
	}
	if fm.Title != "Hub startSession soft-fail" {
		t.Fatalf("Title = %q, want Hub startSession soft-fail", fm.Title)
	}
	if fm.Date != "2026-08-20" {
		t.Fatalf("Date = %q, want 2026-08-20", fm.Date)
	}
	if len(fm.Tags) != 2 || fm.Tags[0] != "hub" || fm.Tags[1] != "sessions" {
		t.Fatalf("Tags = %#v, want [hub sessions]", fm.Tags)
	}
	if len(fm.CodifiedInto) != 1 || fm.CodifiedInto[0] != "AGENTS.md" {
		t.Fatalf("CodifiedInto = %#v, want [AGENTS.md]", fm.CodifiedInto)
	}
	if fm.Raw == nil {
		t.Fatal("Raw must retain parsed YAML keys")
	}
	if body != "Lesson body\n" {
		t.Fatalf("body = %q, want %q", body, "Lesson body\n")
	}
}

func TestSplitFrontmatter_RejectsOversizeBlock(t *testing.T) {
	t.Parallel()

	yamlBlock := "title: x\npad: " + strings.Repeat("a", 64*1024)
	md := "---\n" + yamlBlock + "\n---\nbody\n"
	if _, _, err := SplitFrontmatter(md); err == nil {
		t.Fatal("oversize frontmatter must error (fail closed)")
	}

	unclosed := "---\n" + strings.Repeat("a", 64*1024+1)
	if _, _, err := SplitFrontmatter(unclosed); err == nil {
		t.Fatal("unclosed oversize frontmatter must error (fail closed)")
	}
}

func TestSplitFrontmatter_InvalidYAMLErrors(t *testing.T) {
	t.Parallel()

	md := "---\n[unclosed\n---\nbody\n"
	if _, _, err := SplitFrontmatter(md); err == nil {
		t.Fatal("invalid YAML frontmatter must error")
	}
}

func TestTitleOrStem_UsesTitleWhenPresent(t *testing.T) {
	t.Parallel()

	got := TitleOrStem(Frontmatter{Title: "Hub startSession soft-fail"}, "memory/x.md")
	if got != "Hub startSession soft-fail" {
		t.Fatalf("TitleOrStem = %q, want title", got)
	}
}

func TestTitleOrStem_FallsBackToPathStem(t *testing.T) {
	t.Parallel()

	got := TitleOrStem(Frontmatter{}, "memory/x.md")
	if got != "x" {
		t.Fatalf("TitleOrStem = %q, want x", got)
	}
}
