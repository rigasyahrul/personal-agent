package httpapi

import (
	"database/sql"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/fsroot"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/paths"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

type knowledgeHandlers struct {
	projects  *store.ProjectStore
	knowledge *store.KnowledgeStore
	db        *sql.DB
	dataDir   string
}

// KnowledgeRoutes mounts project knowledge GET endpoints under /api/v1.
// All routes are auth only (not CSRF mutations).
func KnowledgeRoutes(mux *http.ServeMux, db *sql.DB, dataDir string, c clock.Clock, auth func(http.Handler) http.Handler) {
	h := &knowledgeHandlers{
		projects:  store.NewProjectStore(db, dataDir, c),
		knowledge: &store.KnowledgeStore{DB: db, Clock: c},
		db:        db,
		dataDir:   dataDir,
	}
	if auth == nil {
		auth = func(next http.Handler) http.Handler { return requireAuthAt(db, c.Now, next) }
	}
	mux.Handle("GET /api/v1/projects/{id}/knowledge/backlinks", auth(http.HandlerFunc(h.backlinks)))
	mux.Handle("GET /api/v1/projects/{id}/notes/{note_id}/backlinks", auth(http.HandlerFunc(h.noteBacklinks)))
	mux.Handle("GET /api/v1/projects/{id}/knowledge/read", auth(http.HandlerFunc(h.read)))
	mux.Handle("GET /api/v1/projects/{id}/knowledge/tree", auth(http.HandlerFunc(h.tree)))
}

type knowledgeBacklinkDTO struct {
	KnowledgeID string `json:"knowledge_id"`
	Path        string `json:"path"`
	Title       string `json:"title"`
	Snippet     string `json:"snippet"`
}

type knowledgeReadDTO struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type knowledgeTreeEntry struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
	Size int64  `json:"size,omitempty"`
}

func (h *knowledgeHandlers) backlinks(w http.ResponseWriter, r *http.Request) {
	project, err := h.projects.Get(r.Context(), r.PathValue("id"))
	if writeStoreError(w, err, true) {
		return
	}
	q := r.URL.Query()
	path := q.Get("path")
	knowledgeID := q.Get("knowledge_id")
	if (path == "") == (knowledgeID == "") {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	var noteID string
	if knowledgeID != "" {
		note, err := h.knowledge.ByID(r.Context(), knowledgeID)
		if writeKnowledgeError(w, err) {
			return
		}
		if note.ProjectID != project.ID || note.IsGlobal {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		noteID = note.ID
	} else {
		if err := validateKnowledgeHTTPPath(path); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		note, err := h.knowledge.ByScopePath(r.Context(), project.ID, "", false, path)
		if writeKnowledgeError(w, err) {
			return
		}
		noteID = note.ID
	}
	h.writeBacklinks(w, r, noteID)
}

func (h *knowledgeHandlers) noteBacklinks(w http.ResponseWriter, r *http.Request) {
	project, err := h.projects.Get(r.Context(), r.PathValue("id"))
	if writeStoreError(w, err, true) {
		return
	}
	var rel string
	err = h.db.QueryRowContext(r.Context(), `SELECT relative_path FROM notes WHERE id=? AND project_id=?`,
		r.PathValue("note_id"), project.ID).Scan(&rel)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		internalError(w)
		return
	}
	note, err := h.knowledge.ByScopePath(r.Context(), project.ID, "", false, "source/"+rel)
	if writeKnowledgeError(w, err) {
		return
	}
	h.writeBacklinks(w, r, note.ID)
}

func (h *knowledgeHandlers) writeBacklinks(w http.ResponseWriter, r *http.Request, knowledgeID string) {
	links, err := h.knowledge.Backlinks(r.Context(), knowledgeID)
	if writeKnowledgeError(w, err) {
		return
	}
	items := make([]knowledgeBacklinkDTO, 0, len(links))
	for _, bl := range links {
		items = append(items, knowledgeBacklinkDTO{
			KnowledgeID: bl.FromNoteID,
			Path:        bl.FromPath,
			Title:       bl.FromTitle,
			Snippet:     bl.Snippet,
		})
	}
	jsonResponse(w, http.StatusOK, map[string]any{"items": items})
}

func (h *knowledgeHandlers) read(w http.ResponseWriter, r *http.Request) {
	project, err := h.projects.Get(r.Context(), r.PathValue("id"))
	if writeStoreError(w, err, true) {
		return
	}
	rel := r.URL.Query().Get("path")
	if err := validateKnowledgeHTTPPath(rel); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	body, err := readKnowledgeFile(layout.ProjectRoot(h.dataDir, project.VaultID, project.ID), rel)
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, fs.ErrNotExist) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if errors.Is(err, store.ErrValidation) || errors.Is(err, fsroot.ErrInvalidPath) {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err != nil {
		internalError(w)
		return
	}
	if !utf8.Valid(body) {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	jsonResponse(w, http.StatusOK, knowledgeReadDTO{Path: rel, Content: string(body)})
}

func (h *knowledgeHandlers) tree(w http.ResponseWriter, r *http.Request) {
	project, err := h.projects.Get(r.Context(), r.PathValue("id"))
	if writeStoreError(w, err, true) {
		return
	}
	scopeRoot := layout.ProjectRoot(h.dataDir, project.VaultID, project.ID)
	entries, err := listKnowledgeTree(scopeRoot)
	if err != nil {
		internalError(w)
		return
	}
	jsonResponse(w, http.StatusOK, map[string]any{"entries": entries})
}

func writeKnowledgeError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, store.ErrValidation):
		http.Error(w, "invalid request", http.StatusBadRequest)
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	default:
		internalError(w)
	}
	return true
}

func validateKnowledgeHTTPPath(rel string) error {
	if err := paths.ValidateKnowledgeRelPath(rel); err != nil {
		return err
	}
	for _, part := range strings.Split(rel, "/") {
		if part == ".agents" || part == "sessions" {
			return errors.New("reserved knowledge path")
		}
	}
	return nil
}

// readKnowledgeFile reads a scope-root-relative knowledge path.
// source/** → SourceDir + source-rel; memory/** → MemoryDir + memory-rel;
// AGENTS/SOUL/SYSTEM → InstructionPath. Never ValidateRelPath on "memory/…".
func readKnowledgeFile(scopeRoot, rel string) ([]byte, error) {
	if err := validateKnowledgeHTTPPath(rel); err != nil {
		return nil, store.ErrValidation
	}
	switch rel {
	case "AGENTS.md", "SOUL.md", "SYSTEM.md":
		return readInstructionFile(scopeRoot, rel)
	}
	rootName, inner, ok := strings.Cut(rel, "/")
	if !ok || inner == "" {
		return nil, store.ErrValidation
	}
	var dir string
	switch rootName {
	case "memory":
		dir = layout.MemoryDir(scopeRoot)
	case "source":
		dir = layout.SourceDir(scopeRoot)
	default:
		return nil, store.ErrValidation
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
		if isNotExist(err) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	return body, nil
}

func listKnowledgeTree(scopeRoot string) ([]knowledgeTreeEntry, error) {
	var entries []knowledgeTreeEntry
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
		entries = append(entries, knowledgeTreeEntry{Path: prefix, Kind: "directory"})
		kids, err := walkPrefixedTree(dir, prefix)
		if err != nil {
			return nil, err
		}
		entries = append(entries, kids...)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, nil
}

func walkPrefixedTree(dir, prefix string) ([]knowledgeTreeEntry, error) {
	root, err := fsroot.Open(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer root.Close()
	var out []knowledgeTreeEntry
	err = root.Walk(func(name string, info fs.FileInfo) error {
		full := prefix + "/" + name
		if excludedKnowledgePath(full) {
			return nil
		}
		kind := "file"
		size := info.Size()
		if info.IsDir() {
			kind = "directory"
			size = 0
		} else if !info.Mode().IsRegular() {
			return fsroot.ErrUnsafe
		}
		out = append(out, knowledgeTreeEntry{Path: full, Kind: kind, Size: size})
		return nil
	})
	return out, err
}

func excludedKnowledgePath(rel string) bool {
	for _, part := range strings.Split(rel, "/") {
		if part == ".agents" || part == "sessions" {
			return true
		}
	}
	return false
}

func isNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, fs.ErrNotExist)
}
