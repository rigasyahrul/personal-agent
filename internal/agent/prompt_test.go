package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rigasyahrul/personal-agent/internal/layout"
)

func TestBuildSessionPrompt_ProjectOnlyAgentsNoGlobalFallback(t *testing.T) {
	dataDir := t.TempDir()
	// Global AGENTS with distinct body — must NOT appear in project sections.
	globalRoot := layout.GlobalRoot(dataDir)
	if err := os.MkdirAll(globalRoot, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.InstructionPath(globalRoot, "AGENTS.md"), []byte("GLOBAL_AGENTS_BODY"), 0600); err != nil {
		t.Fatal(err)
	}

	projectRoot := layout.ProjectRoot(dataDir, "v1", "p1")
	if err := os.MkdirAll(projectRoot, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.InstructionPath(projectRoot, "AGENTS.md"), []byte("PROJECT_AGENTS_BODY"), 0600); err != nil {
		t.Fatal(err)
	}

	sections, err := BuildSessionPrompt(BuildPromptInput{
		DataDir:   dataDir,
		Home:      layout.SessionHome("project"),
		VaultID:   "v1",
		ProjectID: "p1",
	})
	if err != nil {
		t.Fatal(err)
	}

	names := sectionNames(sections)
	if len(names) != 2 || names[0] != "runtime" || names[1] != "agents" {
		t.Fatalf("want [runtime agents], got %v", names)
	}
	agents := sections[1]
	if agents.Content != "PROJECT_AGENTS_BODY" {
		t.Fatalf("agents content = %q, want PROJECT_AGENTS_BODY", agents.Content)
	}
	if agents.Path != layout.InstructionPath(projectRoot, "AGENTS.md") {
		t.Fatalf("agents path = %q", agents.Path)
	}
	for _, s := range sections {
		if strings.Contains(s.Content, "GLOBAL_AGENTS_BODY") {
			t.Fatalf("project prompt must not include global AGENTS body; section %q has it", s.Name)
		}
	}
	assertRuntimeBasics(t, sections[0], "project")
}

func TestBuildSessionPrompt_VaultUsesGlobalInstructionsAndVaultLessons(t *testing.T) {
	dataDir := t.TempDir()
	globalRoot := layout.GlobalRoot(dataDir)
	if err := os.MkdirAll(globalRoot, 0700); err != nil {
		t.Fatal(err)
	}
	writeAll(t, map[string]string{
		layout.InstructionPath(globalRoot, "SYSTEM.md"): "GLOBAL_SYSTEM",
		layout.InstructionPath(globalRoot, "SOUL.md"):   "GLOBAL_SOUL",
		layout.InstructionPath(globalRoot, "AGENTS.md"): "GLOBAL_AGENTS",
	})

	vaultRoot := layout.VaultRoot(dataDir, "v1")
	if err := os.MkdirAll(layout.MemoryDir(vaultRoot), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.LessonsPath(vaultRoot), []byte("VAULT_LESSONS"), 0600); err != nil {
		t.Fatal(err)
	}

	// Project memory must not leak into vault sessions.
	projectRoot := layout.ProjectRoot(dataDir, "v1", "p1")
	if err := os.MkdirAll(layout.MemoryDir(projectRoot), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.LessonsPath(projectRoot), []byte("PROJECT_LESSONS"), 0600); err != nil {
		t.Fatal(err)
	}

	sections, err := BuildSessionPrompt(BuildPromptInput{
		DataDir: dataDir,
		Home:    layout.SessionHome("vault"),
		VaultID: "v1",
	})
	if err != nil {
		t.Fatal(err)
	}

	byName := mapSections(sections)
	if _, ok := byName["runtime"]; !ok {
		t.Fatal("missing runtime")
	}
	assertSectionContent(t, byName, "system", "GLOBAL_SYSTEM", layout.InstructionPath(globalRoot, "SYSTEM.md"))
	assertSectionContent(t, byName, "soul", "GLOBAL_SOUL", layout.InstructionPath(globalRoot, "SOUL.md"))
	assertSectionContent(t, byName, "agents", "GLOBAL_AGENTS", layout.InstructionPath(globalRoot, "AGENTS.md"))
	assertSectionContent(t, byName, "lessons", "VAULT_LESSONS", layout.LessonsPath(vaultRoot))

	for _, s := range sections {
		if strings.Contains(s.Content, "PROJECT_LESSONS") {
			t.Fatal("vault session must not load project lessons")
		}
	}
	// Order after runtime: system, soul, agents, lessons
	wantOrder := []string{"runtime", "system", "soul", "agents", "lessons"}
	if got := sectionNames(sections); !equalStrings(got, wantOrder) {
		t.Fatalf("order = %v, want %v", got, wantOrder)
	}
	assertRuntimeBasics(t, byName["runtime"], "vault")
}

func TestBuildSessionPrompt_EmptyFilesSkipped(t *testing.T) {
	dataDir := t.TempDir()
	globalRoot := layout.GlobalRoot(dataDir)
	if err := os.MkdirAll(layout.MemoryDir(globalRoot), 0700); err != nil {
		t.Fatal(err)
	}
	writeAll(t, map[string]string{
		layout.InstructionPath(globalRoot, "SYSTEM.md"): "   \n\t  ",
		layout.InstructionPath(globalRoot, "SOUL.md"):   "SOUL_OK",
		layout.InstructionPath(globalRoot, "AGENTS.md"): "",
		layout.LessonsPath(globalRoot):                  "\n",
	})

	sections, err := BuildSessionPrompt(BuildPromptInput{
		DataDir: dataDir,
		Home:    layout.SessionHome("global"),
	})
	if err != nil {
		t.Fatal(err)
	}
	names := sectionNames(sections)
	if !equalStrings(names, []string{"runtime", "soul"}) {
		t.Fatalf("want [runtime soul], got %v", names)
	}
}

func TestBuildSessionPrompt_TruncationSetsFlag(t *testing.T) {
	dataDir := t.TempDir()
	projectRoot := layout.ProjectRoot(dataDir, "", "p1")
	if err := os.MkdirAll(layout.MemoryDir(projectRoot), 0700); err != nil {
		t.Fatal(err)
	}
	// Large lower-priority content; AGENTS small and must survive.
	big := strings.Repeat("L", 200)
	writeAll(t, map[string]string{
		layout.InstructionPath(projectRoot, "AGENTS.md"): "AGENTS_KEEP",
		layout.InstructionPath(projectRoot, "SYSTEM.md"): "SYSTEM_BODY",
		layout.InstructionPath(projectRoot, "SOUL.md"):   "SOUL_BODY",
		layout.LessonsPath(projectRoot):                  big,
	})

	sections, err := BuildSessionPrompt(BuildPromptInput{
		DataDir:         dataDir,
		Home:            layout.SessionHome("project"),
		ProjectID:       "p1",
		MaxPerFileBytes: 50,
		MaxTotalBytes:   80,
	})
	if err != nil {
		t.Fatal(err)
	}

	byName := mapSections(sections)
	if byName["agents"].Content != "AGENTS_KEEP" {
		t.Fatalf("AGENTS must be kept intact, got %q", byName["agents"].Content)
	}
	if byName["agents"].Truncated {
		t.Fatal("AGENTS should not be truncated when it fits")
	}
	// At least one lower-priority section truncated or reduced.
	var anyTrunc bool
	for _, name := range []string{"system", "soul", "lessons"} {
		if s, ok := byName[name]; ok && s.Truncated {
			anyTrunc = true
			break
		}
	}
	// lessons may be dropped entirely if zero budget remains after higher priority.
	if !anyTrunc {
		// If lessons missing entirely due to budget, that also satisfies cap behavior.
		if _, ok := byName["lessons"]; ok {
			t.Fatal("expected Truncated=true on at least one lower-priority section")
		}
	}
	// Total content bytes of non-runtime sections should not exceed MaxTotalBytes.
	total := 0
	for _, s := range sections {
		if s.Name == "runtime" {
			continue
		}
		total += len(s.Content)
	}
	if total > 80 {
		t.Fatalf("total non-runtime bytes %d > MaxTotalBytes 80", total)
	}
}

func TestBuildSessionPrompt_GlobalHomeLoadsGlobalRootOnly(t *testing.T) {
	dataDir := t.TempDir()
	globalRoot := layout.GlobalRoot(dataDir)
	if err := os.MkdirAll(layout.MemoryDir(globalRoot), 0700); err != nil {
		t.Fatal(err)
	}
	writeAll(t, map[string]string{
		layout.InstructionPath(globalRoot, "SYSTEM.md"): "G_SYS",
		layout.InstructionPath(globalRoot, "SOUL.md"):   "G_SOUL",
		layout.InstructionPath(globalRoot, "AGENTS.md"): "G_AGENTS",
		layout.LessonsPath(globalRoot):                  "G_LESSONS",
	})

	// Vault noise must not appear.
	vaultRoot := layout.VaultRoot(dataDir, "v1")
	if err := os.MkdirAll(layout.MemoryDir(vaultRoot), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.LessonsPath(vaultRoot), []byte("V_LESSONS"), 0600); err != nil {
		t.Fatal(err)
	}

	sections, err := BuildSessionPrompt(BuildPromptInput{
		DataDir: dataDir,
		Home:    layout.SessionHome("global"),
	})
	if err != nil {
		t.Fatal(err)
	}
	byName := mapSections(sections)
	assertSectionContent(t, byName, "system", "G_SYS", layout.InstructionPath(globalRoot, "SYSTEM.md"))
	assertSectionContent(t, byName, "soul", "G_SOUL", layout.InstructionPath(globalRoot, "SOUL.md"))
	assertSectionContent(t, byName, "agents", "G_AGENTS", layout.InstructionPath(globalRoot, "AGENTS.md"))
	assertSectionContent(t, byName, "lessons", "G_LESSONS", layout.LessonsPath(globalRoot))
	for _, s := range sections {
		if strings.Contains(s.Content, "V_LESSONS") {
			t.Fatal("global home must not load vault lessons")
		}
	}
	assertRuntimeBasics(t, byName["runtime"], "global")
}

func TestBuildSessionPrompt_PerFileCapTruncates(t *testing.T) {
	dataDir := t.TempDir()
	projectRoot := layout.ProjectRoot(dataDir, "", "p1")
	if err := os.MkdirAll(projectRoot, 0700); err != nil {
		t.Fatal(err)
	}
	body := strings.Repeat("A", 100)
	if err := os.WriteFile(layout.InstructionPath(projectRoot, "AGENTS.md"), []byte(body), 0600); err != nil {
		t.Fatal(err)
	}

	sections, err := BuildSessionPrompt(BuildPromptInput{
		DataDir:         dataDir,
		Home:            layout.SessionHome("project"),
		ProjectID:       "p1",
		MaxPerFileBytes: 40,
		MaxTotalBytes:   10_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	byName := mapSections(sections)
	a := byName["agents"]
	if !a.Truncated {
		t.Fatal("expected Truncated=true when content exceeds MaxPerFileBytes")
	}
	if len(a.Content) != 40 {
		t.Fatalf("content len=%d, want 40", len(a.Content))
	}
}

func writeAll(t *testing.T, files map[string]string) {
	t.Helper()
	for path, body := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func sectionNames(sections []PromptSection) []string {
	out := make([]string, len(sections))
	for i, s := range sections {
		out[i] = s.Name
	}
	return out
}

func mapSections(sections []PromptSection) map[string]PromptSection {
	m := make(map[string]PromptSection, len(sections))
	for _, s := range sections {
		m[s.Name] = s
	}
	return m
}

func assertSectionContent(t *testing.T, byName map[string]PromptSection, name, content, path string) {
	t.Helper()
	s, ok := byName[name]
	if !ok {
		t.Fatalf("missing section %q", name)
	}
	if s.Content != content {
		t.Fatalf("%s content = %q, want %q", name, s.Content, content)
	}
	if s.Path != path {
		t.Fatalf("%s path = %q, want %q", name, s.Path, path)
	}
}

func assertRuntimeBasics(t *testing.T, runtime PromptSection, home string) {
	t.Helper()
	if runtime.Name != "runtime" {
		t.Fatalf("name=%q", runtime.Name)
	}
	if runtime.Path != "" {
		t.Fatalf("runtime Path must be empty, got %q", runtime.Path)
	}
	if !strings.HasPrefix(runtime.Content, "PA_RUNTIME_V1") {
		t.Fatalf("runtime must start with PA_RUNTIME_V1, got %q", runtime.Content[:min(40, len(runtime.Content))])
	}
	if !strings.Contains(runtime.Content, home) {
		t.Fatalf("runtime must mention session home %q", home)
	}
	c := strings.ToLower(runtime.Content)
	if !strings.Contains(c, "tool") {
		t.Fatal("runtime must mention tools")
	}
	if !strings.Contains(c, "safety") {
		t.Fatal("runtime must mention safety")
	}
	if !strings.Contains(c, "compound") {
		t.Fatal("runtime must mention compound / explicit user action")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
