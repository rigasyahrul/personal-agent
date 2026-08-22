package agent

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/rigasyahrul/personal-agent/internal/layout"
)

// PromptSection is one assembled slice of the session system prompt.
type PromptSection struct {
	Name      string // "runtime"|"system"|"soul"|"agents"|"lessons"
	Path      string // empty for runtime
	Content   string
	Truncated bool
}

// BuildPromptInput selects scope roots and byte caps for BuildSessionPrompt.
type BuildPromptInput struct {
	DataDir         string
	Home            layout.SessionHome
	VaultID         string // may be empty
	ProjectID       string // may be empty
	MaxPerFileBytes int    // default 32_768
	MaxTotalBytes   int    // default 96_000
}

const (
	defaultMaxPerFileBytes = 32_768
	defaultMaxTotalBytes   = 96_000
	runtimeMarker          = "PA_RUNTIME_V1"
)

// BuildSessionPrompt loads scoped instruction/memory files and returns ordered sections.
// Load order: runtime, SYSTEM.md, SOUL.md, AGENTS.md, lessons.md (skip missing/empty).
// No project←global fallback. Vault sessions use global instructions + vault memory.
// Truncation priority (keep higher): AGENTS > SYSTEM > SOUL > lessons. Runtime always kept.
func BuildSessionPrompt(in BuildPromptInput) ([]PromptSection, error) {
	maxPer := in.MaxPerFileBytes
	if maxPer <= 0 {
		maxPer = defaultMaxPerFileBytes
	}
	maxTotal := in.MaxTotalBytes
	if maxTotal <= 0 {
		maxTotal = defaultMaxTotalBytes
	}

	instrRoot, memRoot, err := scopeRoots(in)
	if err != nil {
		return nil, err
	}

	runtime := PromptSection{
		Name:    "runtime",
		Path:    "",
		Content: buildRuntimeContent(in, instrRoot, memRoot),
	}

	type fileSpec struct {
		name     string
		path     string
		priority int // higher = keep first when over total budget
	}
	// Load order on disk: SYSTEM, SOUL, AGENTS, lessons.
	// Truncation priority: AGENTS(4) > SYSTEM(3) > SOUL(2) > lessons(1).
	specs := []fileSpec{
		{name: "system", path: layout.InstructionPath(instrRoot, "SYSTEM.md"), priority: 3},
		{name: "soul", path: layout.InstructionPath(instrRoot, "SOUL.md"), priority: 2},
		{name: "agents", path: layout.InstructionPath(instrRoot, "AGENTS.md"), priority: 4},
		{name: "lessons", path: layout.LessonsPath(memRoot), priority: 1},
	}

	var loaded []PromptSection
	for _, sp := range specs {
		sec, ok, err := readSection(sp.name, sp.path, maxPer)
		if err != nil {
			return nil, err
		}
		if ok {
			loaded = append(loaded, sec)
		}
	}

	applyTotalBudget(loaded, maxTotal)

	out := make([]PromptSection, 0, 1+len(loaded))
	out = append(out, runtime)
	for _, s := range loaded {
		if s.Content == "" && s.Truncated {
			// Drop sections reduced to empty by total budget.
			continue
		}
		if s.Content == "" {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

func scopeRoots(in BuildPromptInput) (instrRoot, memRoot string, err error) {
	switch in.Home {
	case layout.SessionHome("project"):
		root := layout.ProjectRoot(in.DataDir, in.VaultID, in.ProjectID)
		return root, root, nil
	case layout.SessionHome("vault"):
		return layout.GlobalRoot(in.DataDir), layout.VaultRoot(in.DataDir, in.VaultID), nil
	case layout.SessionHome("global"):
		root := layout.GlobalRoot(in.DataDir)
		return root, root, nil
	default:
		return "", "", fmt.Errorf("invalid session home %q", in.Home)
	}
}

func buildRuntimeContent(in BuildPromptInput, instrRoot, memRoot string) string {
	skillRoot := memRoot // skill root matches memory root per Canonical table
	var b strings.Builder
	b.WriteString(runtimeMarker)
	b.WriteByte('\n')
	b.WriteString("Personal-agent runtime context.\n")
	b.WriteString("Session home: ")
	b.WriteString(string(in.Home))
	b.WriteByte('\n')
	b.WriteString("Tools and safety: use only provided tools; never exfiltrate secrets; respect path sandboxes.\n")
	b.WriteString("Compounding: compound only on explicit user action — do not write lessons or AGENTS changes unprompted.\n")
	b.WriteString("Path roots for this home:\n")
	b.WriteString("- instruction+AGENTS root: ")
	b.WriteString(instrRoot)
	b.WriteByte('\n')
	b.WriteString("- memory root: ")
	b.WriteString(memRoot)
	b.WriteByte('\n')
	b.WriteString("- skill root: ")
	b.WriteString(skillRoot)
	b.WriteByte('\n')
	return b.String()
}

func readSection(name, path string, maxPer int) (PromptSection, bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return PromptSection{}, false, nil
		}
		return PromptSection{}, false, fmt.Errorf("read %s: %w", path, err)
	}
	content := string(raw)
	if strings.TrimSpace(content) == "" {
		return PromptSection{}, false, nil
	}
	sec := PromptSection{Name: name, Path: path, Content: content}
	if len(sec.Content) > maxPer {
		sec.Content = truncateBytes(sec.Content, maxPer)
		sec.Truncated = true
	}
	return sec, true, nil
}

// applyTotalBudget enforces MaxTotalBytes over non-runtime sections.
// Priority keep order: AGENTS > SYSTEM > SOUL > lessons (reduce lower first).
func applyTotalBudget(sections []PromptSection, maxTotal int) {
	total := 0
	for i := range sections {
		total += len(sections[i].Content)
	}
	if total <= maxTotal {
		return
	}

	// Reduce in reverse priority: lessons, soul, system, agents.
	reduceOrder := []string{"lessons", "soul", "system", "agents"}
	for _, name := range reduceOrder {
		if total <= maxTotal {
			break
		}
		idx := -1
		for i := range sections {
			if sections[i].Name == name {
				idx = i
				break
			}
		}
		if idx < 0 {
			continue
		}
		over := total - maxTotal
		cur := len(sections[idx].Content)
		if cur == 0 {
			continue
		}
		if over >= cur {
			total -= cur
			sections[idx].Content = ""
			sections[idx].Truncated = true
			continue
		}
		keep := cur - over
		sections[idx].Content = truncateBytes(sections[idx].Content, keep)
		sections[idx].Truncated = true
		total = maxTotal
	}
}

// truncateBytes cuts s to at most n bytes without splitting a UTF-8 rune.
func truncateBytes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}
