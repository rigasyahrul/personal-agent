package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/fsroot"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/paths"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

type noteHandlers struct {
	db      *sql.DB
	dataDir string
	notes   *store.NoteStore
}

func NoteRoutes(mux *http.ServeMux, db *sql.DB, dataDir string, c clock.Clock) {
	h := noteHandlers{db: db, dataDir: dataDir, notes: store.NewNoteStore(db, dataDir)}
	authenticated := func(next http.Handler) http.Handler { return requireAuthAt(db, c.Now, next) }
	mux.Handle("GET /api/v1/projects/{id}/tree", authenticated(http.HandlerFunc(h.tree)))
	mux.Handle("POST /api/v1/projects/{id}/folders", authenticated(RequireCSRF(http.HandlerFunc(h.createFolder))))
	mux.Handle("GET /api/v1/notes/{id}", authenticated(http.HandlerFunc(h.get)))
}

func (h noteHandlers) tree(w http.ResponseWriter, r *http.Request) {
	v, err := h.notes.Tree(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "project not found", 404)
		return
	}
	if errors.Is(err, store.ErrIntegrity) {
		jsonResponse(w, 409, map[string]string{"code": "integrity_error"})
		return
	}
	if err != nil {
		internalError(w)
		return
	}
	jsonResponse(w, 200, v)
}
func (h noteHandlers) get(w http.ResponseWriter, r *http.Request) {
	n, err := h.notes.Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "note not found", 404)
		return
	}
	if errors.Is(err, store.ErrIntegrity) {
		jsonResponse(w, 409, map[string]string{"code": "integrity_error"})
		return
	}
	if err != nil {
		internalError(w)
		return
	}
	jsonResponse(w, 200, map[string]any{"id": n.ID, "project_id": n.ProjectID, "relative_path": n.RelativePath, "content_sha256": n.ContentSHA256, "byte_size": n.ByteSize, "revision": n.Revision, "body": string(n.Body)})
}

func (h noteHandlers) createFolder(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	p, err := paths.ValidateRelPath(input.Path)
	parts := strings.Split(p, "/")
	if err != nil || strings.Contains(p, `\`) || strings.EqualFold(path.Ext(parts[len(parts)-1]), ".md") || parts[0] == "memory" || parts[0] == "soul" {
		http.Error(w, "invalid request", 400)
		return
	}
	var vault sql.NullString
	err = h.db.QueryRowContext(r.Context(), `SELECT vault_id FROM projects WHERE id=?`, r.PathValue("id")).Scan(&vault)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "project not found", 404)
		return
	}
	if err != nil {
		internalError(w)
		return
	}
	root, err := fsroot.Open(layout.SourceDir(layout.ProjectRoot(h.dataDir, vault.String, r.PathValue("id"))))
	if err != nil {
		jsonResponse(w, 409, map[string]string{"code": "integrity_error"})
		return
	}
	defer root.Close()
	err = root.MkdirAll(p, 0700)
	if errors.Is(err, fs.ErrExist) {
		http.Error(w, "folder exists", 409)
		return
	}
	if err != nil {
		if errors.Is(err, fsroot.ErrInvalidPath) {
			http.Error(w, "invalid request", 400)
		} else {
			jsonResponse(w, 409, map[string]string{"code": "integrity_error"})
		}
		return
	}
	jsonResponse(w, 201, map[string]string{"path": p})
}
