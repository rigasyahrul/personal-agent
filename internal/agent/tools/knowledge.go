package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rigasyahrul/personal-agent/internal/fsroot"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/paths"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

const (
	ToolSearchProject = "search_project"
	ToolReadKnowledge = "read_knowledge" // {path} scope-root-relative
	ToolListKnowledge = "list_knowledge" // {path?}
)

// ProjectSearcher is the store surface used by search_project.
type ProjectSearcher interface {
	SearchProject(ctx context.Context, projectID, query string, limit int) ([]store.SearchHit, error)
}

// KnowledgeToolHandler executes project knowledge tools.
// Task 65 wires this into Runner; workspace tools stay on Workspace.Execute.
type KnowledgeToolHandler struct {
	Searcher  ProjectSearcher
	ProjectID string
	ScopeRoot string
}

func NewKnowledgeToolHandler(searcher ProjectSearcher, projectID string) *KnowledgeToolHandler {
	return &KnowledgeToolHandler{Searcher: searcher, ProjectID: projectID}
}

func (h *KnowledgeToolHandler) Execute(ctx context.Context, name string, raw json.RawMessage) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	switch name {
	case ToolSearchProject:
		return h.searchProject(ctx, raw)
	case ToolReadKnowledge:
		return h.readKnowledge(raw)
	case ToolListKnowledge:
		return h.listKnowledge(raw)
	default:
		return nil, fmt.Errorf("knowledge tool %q is not allowed", name)
	}
}

type knowledgeHit struct {
	KnowledgeID  string `json:"knowledge_id"`
	Path         string `json:"path"`
	Title        string `json:"title"`
	Snippet      string `json:"snippet"`
	Kind         string `json:"kind"`
	SourceNoteID string `json:"source_note_id,omitempty"`
}

type searchResult struct {
	Hits []knowledgeHit `json:"hits"`
}

func (h *KnowledgeToolHandler) searchProject(ctx context.Context, raw json.RawMessage) (json.RawMessage, error) {
	var a struct {
		Query string `json:"query"`
		Limit *int   `json:"limit"`
	}
	if err := decode(raw, &a); err != nil {
		return nil, err
	}
	if h.Searcher == nil {
		return nil, fmt.Errorf("knowledge searcher is not configured")
	}
	if h.ProjectID == "" {
		return nil, fmt.Errorf("project id required")
	}
	limit := 0
	if a.Limit != nil {
		limit = *a.Limit
	}
	hits, err := h.Searcher.SearchProject(ctx, h.ProjectID, a.Query, limit)
	if err != nil {
		return nil, err
	}
	out := searchResult{Hits: make([]knowledgeHit, 0, len(hits))}
	for _, hit := range hits {
		out.Hits = append(out.Hits, knowledgeHit{
			KnowledgeID:  hit.KnowledgeID,
			Path:         hit.Path,
			Title:        hit.Title,
			Snippet:      hit.Snippet,
			Kind:         string(hit.Kind),
			SourceNoteID: hit.SourceNoteID,
		})
	}
	return json.Marshal(out)
}

type knowledgeReadResult struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func (h *KnowledgeToolHandler) readKnowledge(raw json.RawMessage) (json.RawMessage, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := decode(raw, &a); err != nil {
		return nil, err
	}
	if h.ScopeRoot == "" {
		return nil, fmt.Errorf("knowledge scope root is not configured")
	}
	body, err := readKnowledgeFile(h.ScopeRoot, a.Path)
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(body) {
		return nil, fmt.Errorf("knowledge file is not valid UTF-8")
	}
	return json.Marshal(knowledgeReadResult{Path: a.Path, Content: string(body)})
}

func validateKnowledgeToolPath(rel string) error {
	if err := paths.ValidateKnowledgeRelPath(rel); err != nil {
		return err
	}
	for _, part := range strings.Split(rel, "/") {
		if part == ".agents" || part == "sessions" {
			return fmt.Errorf("reserved knowledge path")
		}
	}
	return nil
}

// readKnowledgeFile uses the knowledge FS strategy: SourceDir for source/**,
// MemoryDir for memory/**, instruction allowlist for AGENTS/SOUL/SYSTEM.
// Never fsroot.Open(scopeRoot)+memory/… — ValidateRelPath rejects that.
func readKnowledgeFile(scopeRoot, rel string) ([]byte, error) {
	if err := validateKnowledgeToolPath(rel); err != nil {
		return nil, err
	}
	switch rel {
	case "AGENTS.md", "SOUL.md", "SYSTEM.md":
		return readInstructionFile(scopeRoot, rel)
	}
	rootName, inner, ok := strings.Cut(rel, "/")
	if !ok || inner == "" {
		return nil, fmt.Errorf("invalid knowledge path")
	}
	var dir string
	switch rootName {
	case "memory":
		dir = layout.MemoryDir(scopeRoot)
	case "source":
		dir = layout.SourceDir(scopeRoot)
	default:
		return nil, fmt.Errorf("invalid knowledge path")
	}
	return readRootedFile(dir, inner)
}

func readInstructionFile(scopeRoot, name string) ([]byte, error) {
	abs := layout.InstructionPath(scopeRoot, name)
	info, err := os.Lstat(abs)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fsroot.ErrUnsafe
	}
	if info.Size() > paths.MaxMarkdownBytes {
		return nil, fsroot.ErrUnsafe
	}
	return os.ReadFile(abs)
}

func readRootedFile(dir, rel string) ([]byte, error) {
	root, err := fsroot.Open(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	defer root.Close()
	body, err := root.ReadFile(rel, paths.MaxMarkdownBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, fs.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	return body, nil
}

type knowledgeListEntry struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
	Size int64  `json:"size,omitempty"`
}

type knowledgeListResult struct {
	Entries []knowledgeListEntry `json:"entries"`
}

func (h *KnowledgeToolHandler) listKnowledge(raw json.RawMessage) (json.RawMessage, error) {
	var a struct {
		Path string `json:"path"`
	}
	if err := decode(raw, &a); err != nil {
		return nil, err
	}
	if h.ScopeRoot == "" {
		return nil, fmt.Errorf("knowledge scope root is not configured")
	}
	entries, err := listKnowledgeDir(h.ScopeRoot, a.Path)
	if err != nil {
		return nil, err
	}
	return json.Marshal(knowledgeListResult{Entries: entries})
}

func validateKnowledgeListPath(rel string) error {
	if rel == "" {
		return nil
	}
	if !utf8.ValidString(rel) || strings.HasPrefix(rel, "/") || filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" || strings.Contains(rel, `\`) {
		return fmt.Errorf("invalid knowledge path")
	}
	if len(rel) > paths.MaxPathBytes {
		return fmt.Errorf("invalid knowledge path")
	}
	parts := strings.Split(rel, "/")
	if len(parts) > paths.MaxDepth {
		return fmt.Errorf("invalid knowledge path")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || part == ".agents" || part == "sessions" || part == "soul" {
			return fmt.Errorf("invalid knowledge path")
		}
		if len(part) > paths.MaxComponentBytes {
			return fmt.Errorf("invalid knowledge path")
		}
		for _, r := range part {
			if r == 0 || unicode.IsControl(r) {
				return fmt.Errorf("invalid knowledge path")
			}
		}
	}
	switch parts[0] {
	case "source", "memory":
		return nil
	default:
		return fmt.Errorf("invalid knowledge path")
	}
}

func listKnowledgeDir(scopeRoot, rel string) ([]knowledgeListEntry, error) {
	if err := validateKnowledgeListPath(rel); err != nil {
		return nil, err
	}
	if rel == "" {
		return listKnowledgeRoots(scopeRoot)
	}
	rootName, inner, _ := strings.Cut(rel, "/")
	var dir string
	switch rootName {
	case "memory":
		dir = layout.MemoryDir(scopeRoot)
	case "source":
		dir = layout.SourceDir(scopeRoot)
	default:
		return nil, fmt.Errorf("invalid knowledge path")
	}
	return listPrefixedDir(dir, rootName, inner)
}

func listKnowledgeRoots(scopeRoot string) ([]knowledgeListEntry, error) {
	var entries []knowledgeListEntry
	for _, prefix := range []string{"source", "memory"} {
		var dir string
		switch prefix {
		case "source":
			dir = layout.SourceDir(scopeRoot)
		case "memory":
			dir = layout.MemoryDir(scopeRoot)
		}
		info, err := os.Lstat(dir)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fsroot.ErrUnsafe
		}
		entries = append(entries, knowledgeListEntry{Path: prefix, Kind: "directory"})
	}
	for _, name := range []string{"AGENTS.md", "SOUL.md", "SYSTEM.md"} {
		info, err := os.Lstat(layout.InstructionPath(scopeRoot, name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fsroot.ErrUnsafe
		}
		entries = append(entries, knowledgeListEntry{Path: name, Kind: "file", Size: info.Size()})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

func listPrefixedDir(dir, prefix, inner string) ([]knowledgeListEntry, error) {
	root, err := fsroot.Open(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if inner == "" {
				return []knowledgeListEntry{}, nil
			}
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	defer root.Close()
	var (
		out     []knowledgeListEntry
		seenDir = inner == ""
	)
	err = root.Walk(func(name string, info fs.FileInfo) error {
		full := prefix + "/" + name
		if excludedKnowledgePath(full) {
			return nil
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return fsroot.ErrUnsafe
		}
		if inner == "" {
			if strings.Contains(name, "/") {
				return nil
			}
			out = append(out, listEntry(full, info))
			return nil
		}
		if name == inner {
			if !info.IsDir() {
				return fmt.Errorf("not a directory")
			}
			seenDir = true
			return nil
		}
		if isDirectChild(inner, name) {
			seenDir = true
			out = append(out, listEntry(full, info))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !seenDir {
		return nil, os.ErrNotExist
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func listEntry(path string, info fs.FileInfo) knowledgeListEntry {
	if info.IsDir() {
		return knowledgeListEntry{Path: path, Kind: "directory"}
	}
	return knowledgeListEntry{Path: path, Kind: "file", Size: info.Size()}
}

func isDirectChild(parent, name string) bool {
	prefix := parent + "/"
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	rest := name[len(prefix):]
	return rest != "" && !strings.Contains(rest, "/")
}

func excludedKnowledgePath(rel string) bool {
	for _, part := range strings.Split(rel, "/") {
		if part == ".agents" || part == "sessions" {
			return true
		}
	}
	return false
}
