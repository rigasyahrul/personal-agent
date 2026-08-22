package layout

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectLayoutAndCreation(t *testing.T) {
	d := t.TempDir()
	if got, want := ProjectRoot(d, "", "p1"), filepath.Join(d, "files", "global", "projects", "p1"); got != want {
		t.Fatalf("root=%q want %q", got, want)
	}
	if got, want := ProjectRoot(d, "v1", "p1"), filepath.Join(d, "files", "vaults", "v1", "projects", "p1"); got != want {
		t.Fatalf("vault root=%q want %q", got, want)
	}
	if got := SourceDir(ProjectRoot(d, "", "p1")); got != filepath.Join(d, "files", "global", "projects", "p1", "source") {
		t.Fatal(got)
	}
	if got := SessionWorkspace(d, SessionHome("project"), "v1", "p1", "s1"); got != filepath.Join(d, "files", "vaults", "v1", "projects", "p1", "sessions", "s1") {
		t.Fatal(got)
	}
	if err := EnsureProjectDirs(d, "v1", "p1"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"source", "memory", "soul"} {
		if st, err := os.Stat(filepath.Join(ProjectRoot(d, "v1", "p1"), name)); err != nil || !st.IsDir() {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestEnsureProjectDirsCleansPartialFreshRootAndPreservesExistingRoot(t *testing.T) {
	d := t.TempDir()
	root := ProjectRoot(d, "", "partial")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "memory"), []byte("block"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureProjectDirs(d, "", "partial"); err == nil {
		t.Fatal("expected preexisting root rejection")
	}
	if b, err := os.ReadFile(filepath.Join(root, "memory")); err != nil || string(b) != "block" {
		t.Fatalf("preexisting changed: %q %v", b, err)
	}

	blockedParent := filepath.Join(d, "blocked")
	if err := os.WriteFile(blockedParent, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureProjectDirs(blockedParent, "", "p"); err == nil {
		t.Fatal("expected parent failure")
	}
	if _, err := os.Stat(ProjectRoot(blockedParent, "", "p")); !errors.Is(err, os.ErrNotExist) && err == nil {
		t.Fatalf("partial root remains: %v", err)
	}
}

func TestEnsureProjectDirsCleansFreshRootAfterChildFailure(t *testing.T) {
	d := t.TempDir()
	original := mkdirProjectChild
	calls := 0
	mkdirProjectChild = func(name string, perm os.FileMode) error {
		calls++
		if calls == 2 {
			return errors.New("injected child failure")
		}
		return os.Mkdir(name, perm)
	}
	t.Cleanup(func() { mkdirProjectChild = original })
	root := ProjectRoot(d, "", "p")
	if err := EnsureProjectDirs(d, "", "p"); err == nil {
		t.Fatal("expected child failure")
	}
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial root remains: %v", err)
	}
}

func TestSessionWorkspaceAllHomes(t *testing.T) {
	d := t.TempDir()
	cases := map[SessionHome]string{
		"global":  filepath.Join(d, "files", "global", "sessions", "s"),
		"vault":   filepath.Join(d, "files", "vaults", "v", "sessions", "s"),
		"project": filepath.Join(d, "files", "vaults", "v", "projects", "p", "sessions", "s"),
	}
	for home, want := range cases {
		if got := SessionWorkspace(d, home, "v", "p", "s"); got != want {
			t.Errorf("%s: %q want %q", home, got, want)
		}
	}
}

func TestKnowledgePaths(t *testing.T) {
	g := GlobalRoot("/data")
	if g != filepath.Join("/data", "files", "global") {
		t.Fatalf("global: %s", g)
	}
	v := VaultRoot("/data", "v1")
	if v != filepath.Join("/data", "files", "vaults", "v1") {
		t.Fatalf("vault: %s", v)
	}
	p := ProjectRoot("/data", "v1", "p1")
	if InstructionPath(p, "AGENTS.md") != filepath.Join(p, "AGENTS.md") {
		t.Fatal("agents path")
	}
	if LessonsPath(p) != filepath.Join(p, "memory", "lessons.md") {
		t.Fatal("lessons")
	}
	if CompoundingSkillPath(g) != filepath.Join(g, ".agents", "skills", "compounding", "SKILL.md") {
		t.Fatal("skill")
	}
	if MemoryDir(p) != filepath.Join(p, "memory") {
		t.Fatal("memory dir")
	}
	if AgentsSkillsDir(g) != filepath.Join(g, ".agents", "skills") {
		t.Fatal("agents skills dir")
	}
}

// Exact Canonical AGENTS Memory block (must match plan contracts).
const wantAgentsMemoryBlock = "## Memory\n" +
	"- Lesson index: [[memory/lessons|lessons.md]] — scan titles when stuck or before reinventing a fix.\n" +
	"- Detail files live under `memory/YYYYMMDD-HHmm-*.md`; open only what the index suggests.\n" +
	"- Prefer codifying durable rules here; keep evidence in memory (compound ≠ diary).\n"

// Exact lessons scaffold from Task 3 seed rules.
const wantLessonsScaffold = "# Lessons\n" +
	"\n" +
	"> Thin index only. Detail files: `memory/YYYYMMDD-HHmm-slug.md`.\n" +
	"\n"

func TestEnsureProjectKnowledgeCreatesSeeds(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureProjectDirs(dir, "", "p1"); err != nil {
		t.Fatal(err)
	}
	skill := "# skill\n"
	if err := EnsureProjectKnowledge(dir, "", "p1", skill); err != nil {
		t.Fatal(err)
	}
	root := ProjectRoot(dir, "", "p1")

	agents, err := os.ReadFile(InstructionPath(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(agents) != wantAgentsMemoryBlock {
		t.Fatalf("AGENTS.md = %q want %q", agents, wantAgentsMemoryBlock)
	}

	lessons, err := os.ReadFile(LessonsPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if string(lessons) != wantLessonsScaffold {
		t.Fatalf("lessons.md = %q want %q", lessons, wantLessonsScaffold)
	}

	skillBody, err := os.ReadFile(CompoundingSkillPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if string(skillBody) != skill {
		t.Fatalf("skill = %q want %q", skillBody, skill)
	}

	for _, name := range []string{"SOUL.md", "SYSTEM.md"} {
		p := InstructionPath(root, name)
		st, err := os.Stat(p)
		if err != nil {
			t.Fatalf("%s missing: %v", name, err)
		}
		if st.IsDir() {
			t.Fatalf("%s is dir", name)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		// empty or single newline
		if len(b) > 1 || (len(b) == 1 && b[0] != '\n') {
			t.Fatalf("%s content unexpected: %q", name, b)
		}
	}

	for _, d := range []string{MemoryDir(root), filepath.Join(AgentsSkillsDir(root), "compounding")} {
		st, err := os.Stat(d)
		if err != nil || !st.IsDir() {
			t.Fatalf("dir %s: %v", d, err)
		}
	}
}

func TestEnsureProjectKnowledgeIdempotent(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureProjectDirs(dir, "", "p1"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureProjectKnowledge(dir, "", "p1", "# skill\n"); err != nil {
		t.Fatal(err)
	}
	agents := InstructionPath(ProjectRoot(dir, "", "p1"), "AGENTS.md")
	if err := os.WriteFile(agents, []byte("custom\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureProjectKnowledge(dir, "", "p1", "# skill\n"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(agents)
	if string(b) != "custom\n" {
		t.Fatalf("overwrote agents: %q", b)
	}
}

func TestEnsureProjectKnowledgeIdempotentSkillAndLessons(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureProjectDirs(dir, "", "p1"); err != nil {
		t.Fatal(err)
	}
	root := ProjectRoot(dir, "", "p1")
	if err := EnsureProjectKnowledge(dir, "", "p1", "# skill\n"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(CompoundingSkillPath(root), []byte("edited-skill\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(LessonsPath(root), []byte("edited-lessons\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureProjectKnowledge(dir, "", "p1", "# other-skill\n"); err != nil {
		t.Fatal(err)
	}
	skill, _ := os.ReadFile(CompoundingSkillPath(root))
	if string(skill) != "edited-skill\n" {
		t.Fatalf("overwrote skill: %q", skill)
	}
	lessons, _ := os.ReadFile(LessonsPath(root))
	if string(lessons) != "edited-lessons\n" {
		t.Fatalf("overwrote lessons: %q", lessons)
	}
}

func TestEnsureGlobalKnowledgeDirsSeeds(t *testing.T) {
	dir := t.TempDir()
	skill := "# global-skill\n"
	if err := EnsureGlobalKnowledgeDirs(dir, skill); err != nil {
		t.Fatal(err)
	}
	root := GlobalRoot(dir)

	agents, err := os.ReadFile(InstructionPath(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(agents) != wantAgentsMemoryBlock {
		t.Fatalf("AGENTS.md = %q want %q", agents, wantAgentsMemoryBlock)
	}
	lessons, err := os.ReadFile(LessonsPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if string(lessons) != wantLessonsScaffold {
		t.Fatalf("lessons.md = %q", lessons)
	}
	skillBody, err := os.ReadFile(CompoundingSkillPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if string(skillBody) != skill {
		t.Fatalf("skill = %q", skillBody)
	}
	for _, name := range []string{"SOUL.md", "SYSTEM.md"} {
		if _, err := os.Stat(InstructionPath(root, name)); err != nil {
			t.Fatalf("%s missing: %v", name, err)
		}
	}

	// idempotency
	if err := os.WriteFile(InstructionPath(root, "AGENTS.md"), []byte("g-custom\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureGlobalKnowledgeDirs(dir, "# other\n"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(InstructionPath(root, "AGENTS.md"))
	if string(b) != "g-custom\n" {
		t.Fatalf("overwrote global agents: %q", b)
	}
}

func TestEnsureVaultKnowledgeDirsSeedsMemoryAndSkillOnly(t *testing.T) {
	dir := t.TempDir()
	skill := "# vault-skill\n"
	if err := EnsureVaultKnowledgeDirs(dir, "v1", skill); err != nil {
		t.Fatal(err)
	}
	root := VaultRoot(dir, "v1")

	lessons, err := os.ReadFile(LessonsPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if string(lessons) != wantLessonsScaffold {
		t.Fatalf("lessons.md = %q", lessons)
	}
	skillBody, err := os.ReadFile(CompoundingSkillPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if string(skillBody) != skill {
		t.Fatalf("skill = %q", skillBody)
	}

	// Canonical: vault has memory+skill only — NO SOUL/SYSTEM/AGENTS
	for _, name := range []string{"AGENTS.md", "SOUL.md", "SYSTEM.md"} {
		p := InstructionPath(root, name)
		if _, err := os.Stat(p); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("vault must not have %s: err=%v", name, err)
		}
	}

	// idempotency for skill and lessons
	if err := os.WriteFile(CompoundingSkillPath(root), []byte("v-edited\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(LessonsPath(root), []byte("v-lessons\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureVaultKnowledgeDirs(dir, "v1", "# other\n"); err != nil {
		t.Fatal(err)
	}
	skillBody, _ = os.ReadFile(CompoundingSkillPath(root))
	if string(skillBody) != "v-edited\n" {
		t.Fatalf("overwrote vault skill: %q", skillBody)
	}
	lessons, _ = os.ReadFile(LessonsPath(root))
	if string(lessons) != "v-lessons\n" {
		t.Fatalf("overwrote vault lessons: %q", lessons)
	}
}
