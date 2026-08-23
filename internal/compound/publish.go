package compound

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/fsroot"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/paths"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

var wikilinkRE = regexp.MustCompile(`\[\[([^\]|#]+)(?:\|([^\]]+))?\]\]`)
var listItemRE = regexp.MustCompile(`^\s*[-*]\s+`)

// Publisher writes approved compound items to the knowledge filesystem.
type Publisher struct {
	DataDir string
	DB      *sql.DB
	Clock   clock.Clock
	Barrier store.MutBarrier
}

// PublishApproved writes final items for an approved proposal.
// It does not MarkFinished — the caller does that after success/failure.
func (p *Publisher) PublishApproved(ctx context.Context, proposal domain.CompoundProposal) error {
	fn := func() error { return p.publishApproved(ctx, proposal) }
	if p.Barrier != nil {
		return p.Barrier.Mutate(fn)
	}
	return fn()
}

func (p *Publisher) publishApproved(ctx context.Context, proposal domain.CompoundProposal) error {
	if proposal.Status != domain.CompoundStatusApproved {
		return fmt.Errorf("%w: publish requires approved proposal", store.ErrValidation)
	}
	var items []store.CompoundItem
	if err := json.Unmarshal([]byte(proposal.ItemsJSON), &items); err != nil {
		return fmt.Errorf("%w: items_json: %v", store.ErrValidation, err)
	}
	if err := store.ValidateCompoundItems(proposal.Scope, items); err != nil {
		return err
	}

	scopeRoot, err := resolveScopeRoot(p.DataDir, proposal)
	if err != nil {
		return err
	}

	planned, err := p.planWrites(scopeRoot, items)
	if err != nil {
		return err
	}

	applied := make([]appliedWrite, 0, len(planned))
	rollback := func() {
		for i := len(applied) - 1; i >= 0; i-- {
			_ = applied[i].restore()
		}
	}
	for _, w := range planned {
		if err := w.apply(); err != nil {
			rollback()
			return err
		}
		applied = append(applied, w)
	}

	if err := p.upsertKnowledge(ctx, proposal, planned); err != nil {
		rollback()
		return err
	}
	return nil
}

type plannedWrite struct {
	kind    domain.KnowledgeKind
	relPath string // scope-root relative (AGENTS.md or memory/…)
	body    []byte
	title   string
	rootDir string
	rootRel string // path inside fsroot (no reserved memory/ prefix)
	prev    []byte
	existed bool
}

type appliedWrite = plannedWrite

func (w plannedWrite) apply() error {
	if err := os.MkdirAll(w.rootDir, 0o700); err != nil {
		return err
	}
	root, err := fsroot.Open(w.rootDir)
	if err != nil {
		return err
	}
	defer root.Close()
	return root.WriteFileAtomic(w.rootRel, w.body, 0o600)
}

func (w plannedWrite) restore() error {
	if !w.existed {
		return os.Remove(filepath.Join(w.rootDir, filepath.FromSlash(w.rootRel)))
	}
	root, err := fsroot.Open(w.rootDir)
	if err != nil {
		return err
	}
	defer root.Close()
	return root.WriteFileAtomic(w.rootRel, w.prev, 0o600)
}

func (p *Publisher) planWrites(scopeRoot string, items []store.CompoundItem) ([]plannedWrite, error) {
	var out []plannedWrite
	for _, it := range items {
		switch it.Kind {
		case store.CompoundKindAgentsPatch:
			if err := ValidateAgentsMemoryPointer(it.Content); err != nil {
				return nil, err
			}
			w, err := planInstructionWrite(scopeRoot, it)
			if err != nil {
				return nil, err
			}
			out = append(out, w)
		case store.CompoundKindMemoryDetail:
			w, err := planMemoryWrite(scopeRoot, it, domain.KnowledgeKindMemoryDetail)
			if err != nil {
				return nil, err
			}
			out = append(out, w)
		case store.CompoundKindLessonsIndexRow:
			existing, err := store.ReadLessonsIndex(scopeRoot)
			if err != nil {
				return nil, err
			}
			merged := mergeLessons(existing, it.Content)
			it.Content = merged
			w, err := planMemoryWrite(scopeRoot, it, domain.KnowledgeKindMemoryIndex)
			if err != nil {
				return nil, err
			}
			out = append(out, w)
		default:
			return nil, fmt.Errorf("%w: unknown compound kind %q", store.ErrValidation, it.Kind)
		}
	}
	return out, nil
}

func planInstructionWrite(scopeRoot string, it store.CompoundItem) (plannedWrite, error) {
	if it.Path != "AGENTS.md" {
		return plannedWrite{}, fmt.Errorf("%w: agents_patch path must be AGENTS.md", store.ErrValidation)
	}
	if err := paths.ValidateKnowledgeRelPath(it.Path); err != nil {
		return plannedWrite{}, fmt.Errorf("%w: %v", store.ErrValidation, err)
	}
	prev, existed, err := readMaybe(filepath.Join(scopeRoot, "AGENTS.md"))
	if err != nil {
		return plannedWrite{}, err
	}
	return plannedWrite{
		kind:    domain.KnowledgeKindAgents,
		relPath: "AGENTS.md",
		body:    []byte(it.Content),
		title:   firstHeading(it.Content, "AGENTS"),
		rootDir: scopeRoot,
		rootRel: "AGENTS.md",
		prev:    prev,
		existed: existed,
	}, nil
}

func planMemoryWrite(scopeRoot string, it store.CompoundItem, kind domain.KnowledgeKind) (plannedWrite, error) {
	if err := paths.ValidateKnowledgeRelPath(it.Path); err != nil {
		return plannedWrite{}, fmt.Errorf("%w: %v", store.ErrValidation, err)
	}
	rel, ok := strings.CutPrefix(it.Path, "memory/")
	if !ok || rel == "" || strings.Contains(rel, "..") {
		return plannedWrite{}, fmt.Errorf("%w: memory path must be under memory/", store.ErrValidation)
	}
	memRoot := layout.MemoryDir(scopeRoot)
	prev, existed, err := readMaybe(filepath.Join(memRoot, filepath.FromSlash(rel)))
	if err != nil {
		return plannedWrite{}, err
	}
	return plannedWrite{
		kind:    kind,
		relPath: it.Path,
		body:    []byte(it.Content),
		title:   firstHeading(it.Content, strings.TrimSuffix(filepath.Base(rel), ".md")),
		rootDir: memRoot,
		rootRel: rel,
		prev:    prev,
		existed: existed,
	}, nil
}

func readMaybe(abs string) (body []byte, existed bool, err error) {
	b, err := os.ReadFile(abs)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return b, true, nil
}

func resolveScopeRoot(dataDir string, p domain.CompoundProposal) (string, error) {
	switch p.Scope {
	case domain.CompoundScopeProject:
		if p.ProjectID == "" {
			return "", fmt.Errorf("%w: project publish requires project_id", store.ErrValidation)
		}
		return layout.ProjectRoot(dataDir, p.VaultID, p.ProjectID), nil
	case domain.CompoundScopeVault:
		if p.VaultID == "" {
			return "", fmt.Errorf("%w: vault publish requires vault_id", store.ErrValidation)
		}
		return layout.VaultRoot(dataDir, p.VaultID), nil
	case domain.CompoundScopeGlobal:
		return layout.GlobalRoot(dataDir), nil
	default:
		return "", fmt.Errorf("%w: invalid compound scope", store.ErrValidation)
	}
}

func mergeLessons(existing, proposed string) string {
	existRows, prefix, suffix := splitLessonRows(existing)
	propRows, _, _ := splitLessonRows(proposed)
	if len(propRows) == 0 && strings.TrimSpace(proposed) != "" {
		// No list rows — try whole proposed as a single new line if it has a wikilink.
		if path, ok := firstWikilinkPath(proposed); ok {
			propRows = []lessonRow{{Path: path, Line: strings.TrimRight(proposed, "\n")}}
		}
	}

	seen := map[string]int{}
	var kept []lessonRow
	for _, r := range existRows {
		seen[r.Path] = len(kept)
		kept = append(kept, r)
	}
	var prepend []lessonRow
	for _, r := range propRows {
		if i, ok := seen[r.Path]; ok {
			kept[i] = r
			continue
		}
		seen[r.Path] = -1
		prepend = append(prepend, r)
	}

	var b strings.Builder
	b.WriteString(prefix)
	if prefix != "" && !strings.HasSuffix(prefix, "\n") {
		b.WriteByte('\n')
	}
	for _, r := range prepend {
		b.WriteString(r.Line)
		if !strings.HasSuffix(r.Line, "\n") {
			b.WriteByte('\n')
		}
	}
	for _, r := range kept {
		b.WriteString(r.Line)
		if !strings.HasSuffix(r.Line, "\n") {
			b.WriteByte('\n')
		}
	}
	b.WriteString(suffix)
	return b.String()
}

type lessonRow struct {
	Path string
	Line string
}

func splitLessonRows(content string) (rows []lessonRow, prefix, suffix string) {
	if content == "" {
		return nil, "", ""
	}
	lines := strings.SplitAfter(content, "\n")
	// Drop the empty trailing split if content ended with newline — keep line text as-is via SplitAfter.
	var firstIdx, lastIdx = -1, -1
	for i, line := range lines {
		if path, ok := lessonRowPath(line); ok {
			rows = append(rows, lessonRow{Path: path, Line: line})
			if firstIdx < 0 {
				firstIdx = i
			}
			lastIdx = i
		}
	}
	if firstIdx < 0 {
		return nil, content, ""
	}
	prefix = strings.Join(lines[:firstIdx], "")
	if lastIdx+1 < len(lines) {
		// suffix is non-row lines after the last row; skip leftover row lines in the middle (already in rows).
		var suf strings.Builder
		for _, line := range lines[lastIdx+1:] {
			if _, ok := lessonRowPath(line); ok {
				continue
			}
			suf.WriteString(line)
		}
		suffix = suf.String()
	}
	return rows, prefix, suffix
}

func lessonRowPath(line string) (string, bool) {
	if !listItemRE.MatchString(line) {
		return "", false
	}
	return firstWikilinkPath(line)
}

func firstWikilinkPath(s string) (string, bool) {
	m := wikilinkRE.FindStringSubmatch(s)
	if m == nil {
		return "", false
	}
	target := strings.TrimSpace(m[1])
	if target == "" || strings.Contains(target, "..") || strings.HasPrefix(target, "/") {
		return "", false
	}
	switch target {
	case "AGENTS", "SOUL", "SYSTEM":
		target += ".md"
	default:
		if !strings.HasSuffix(target, ".md") && !strings.EqualFold(target, "AGENTS.md") {
			target += ".md"
		}
	}
	return target, true
}

func firstHeading(content, fallback string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return fallback
}

func (p *Publisher) upsertKnowledge(ctx context.Context, proposal domain.CompoundProposal, writes []plannedWrite) error {
	if p.DB == nil {
		return nil
	}
	projectID, vaultID, isGlobal, err := knowledgeScope(proposal)
	if err != nil {
		return err
	}
	ks := &store.KnowledgeStore{DB: p.DB, Clock: p.Clock}
	for _, w := range writes {
		if _, err := ks.UpsertFromContent(ctx, store.UpsertKnowledgeInput{
			Kind:         w.kind,
			ProjectID:    projectID,
			VaultID:      vaultID,
			IsGlobal:     isGlobal,
			RelativePath: w.relPath,
			Content:      w.body,
			Status:       "ready",
		}); err != nil {
			return err
		}
	}
	return nil
}

func knowledgeScope(p domain.CompoundProposal) (projectID, vaultID string, isGlobal bool, err error) {
	switch p.Scope {
	case domain.CompoundScopeProject:
		if p.ProjectID == "" {
			return "", "", false, fmt.Errorf("%w: project knowledge requires project_id", store.ErrValidation)
		}
		return p.ProjectID, "", false, nil
	case domain.CompoundScopeVault:
		if p.VaultID == "" {
			return "", "", false, fmt.Errorf("%w: vault knowledge requires vault_id", store.ErrValidation)
		}
		return "", p.VaultID, false, nil
	case domain.CompoundScopeGlobal:
		return "", "", true, nil
	default:
		return "", "", false, fmt.Errorf("%w: invalid compound scope", store.ErrValidation)
	}
}
