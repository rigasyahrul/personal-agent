package httpapi

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/fsroot"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/paths"
	"github.com/rigasyahrul/personal-agent/internal/publish"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

type noteHandlers struct {
	db      *sql.DB
	dataDir string
	notes   *store.NoteStore
	publish *publish.Machine
	barrier store.MutBarrier
}

func NoteRoutes(mux *http.ServeMux, db *sql.DB, dataDir string, c clock.Clock, machine *publish.Machine, barrier store.MutBarrier) {
	h := noteHandlers{db: db, dataDir: dataDir, notes: store.NewNoteStore(db, dataDir), publish: machine, barrier: barrier}
	authenticated := func(next http.Handler) http.Handler { return requireAuthAt(db, c.Now, next) }
	mux.Handle("GET /api/v1/projects/{id}/tree", authenticated(http.HandlerFunc(h.tree)))
	mux.Handle("POST /api/v1/projects/{id}/folders", authenticated(RequireCSRF(http.HandlerFunc(h.createFolder))))
	mux.Handle("GET /api/v1/notes/{id}", authenticated(http.HandlerFunc(h.get)))
	mux.Handle("POST /api/v1/projects/{id}/direct-notes", authenticated(RequireCSRF(http.HandlerFunc(h.direct))))
}

func (h noteHandlers) direct(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("Idempotency-Key")
	if strings.TrimSpace(key) == "" {
		http.Error(w, "invalid request", 400)
		return
	}
	var in struct {
		RelativePath string            `json:"relative_path"`
		ReviewMode   domain.ReviewMode `json:"review_mode"`
		Body         string            `json:"body"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 6*paths.MaxMarkdownBytes+4096)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&in); err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		http.Error(w, "invalid request", 400)
		return
	}
	clean, err := paths.ValidateRelPath(in.RelativePath)
	if err != nil {
		http.Error(w, "invalid request", 400)
		return
	}
	project := r.PathValue("id")
	sum := sha256.Sum256([]byte(project + "\x00" + clean + "\x00" + string(in.ReviewMode) + "\x00" + in.Body))
	existing, lookupErr := (store.DirectStore{DB: h.db}).ByKey(r.Context(), key)
	created := store.IsNoRows(lookupErr)
	opID, noteID := uuid.NewString(), uuid.NewString()
	if lookupErr == nil {
		opID, noteID = existing.ID, existing.NoteID
	} else if !created {
		internalError(w)
		return
	}
	status, note, err := h.publish.Run(r.Context(), publish.PublishInput{OpID: opID, RequestKey: key, RequestFingerprint: fmt.Sprintf("%x", sum), Kind: "direct", Body: []byte(in.Body), TargetProjectID: project, TargetRelPath: clean, ReviewMode: in.ReviewMode, NoteID: noteID})
	if errors.Is(err, publish.ErrInvalid) {
		http.Error(w, "invalid request", 400)
		return
	}
	if errors.Is(err, publish.ErrConflict) {
		http.Error(w, "conflict", 409)
		return
	}
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, store.ErrNotFound) {
		http.Error(w, "project not found", 404)
		return
	}
	if err != nil {
		internalError(w)
		return
	}
	code := http.StatusCreated
	if !created {
		code = http.StatusOK
	}
	jsonResponse(w, code, map[string]string{"operation_id": opID, "status": status, "note_id": note})
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
	mkdir := func() error { return root.MkdirAll(p, 0700) }
	if h.barrier != nil {
		err = h.barrier.Mutate(mkdir)
	} else {
		err = mkdir()
	}
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
