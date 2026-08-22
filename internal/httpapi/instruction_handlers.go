package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

type instructionHandlers struct {
	instructions *store.InstructionStore
	projects     *store.ProjectStore
	dataDir      string
}

// InstructionRoutes mounts project + global instruction GET/PUT under /api/v1.
// PUT uses securedMutation (auth then CSRF), matching other mutation routes.
func InstructionRoutes(mux *http.ServeMux, db *sql.DB, dataDir string, c clock.Clock, barrier store.MutBarrier, auth, mutation func(http.Handler) http.Handler) {
	h := &instructionHandlers{
		instructions: &store.InstructionStore{DB: db, Clock: c, Barrier: barrier},
		projects:     store.NewProjectStore(db, dataDir, c),
		dataDir:      dataDir,
	}
	if auth == nil {
		auth = func(next http.Handler) http.Handler { return requireAuthAt(db, c.Now, next) }
	}
	if mutation == nil {
		now := func() time.Time { return c.Now().UTC() }
		mutation = func(next http.Handler) http.Handler { return securedMutation(db, now, next) }
	}
	mux.Handle("GET /api/v1/projects/{id}/instructions/{name}", auth(http.HandlerFunc(h.getProject)))
	mux.Handle("PUT /api/v1/projects/{id}/instructions/{name}", mutation(http.HandlerFunc(h.putProject)))
	mux.Handle("GET /api/v1/global/instructions/{name}", auth(http.HandlerFunc(h.getGlobal)))
	mux.Handle("PUT /api/v1/global/instructions/{name}", mutation(http.HandlerFunc(h.putGlobal)))
}

type instructionBody struct {
	Content string `json:"content"`
}

type instructionDTO struct {
	Content string               `json:"content"`
	Note    domain.KnowledgeNote `json:"note"`
}

func (h *instructionHandlers) getProject(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	name := r.PathValue("name")
	project, err := h.projects.Get(r.Context(), projectID)
	if writeInstructionStoreError(w, err, true) {
		return
	}
	meta := store.ScopeMeta{
		DataDir:   h.dataDir,
		Scope:     domain.CompoundScopeProject,
		ProjectID: project.ID,
		VaultID:   project.VaultID,
	}
	content, note, err := h.instructions.Get(r.Context(), meta, store.InstructionName(name))
	if writeInstructionStoreError(w, err, true) {
		return
	}
	jsonResponse(w, http.StatusOK, instructionDTO{Content: content, Note: note})
}

func (h *instructionHandlers) putProject(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	name := r.PathValue("name")
	var body instructionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	project, err := h.projects.Get(r.Context(), projectID)
	if writeInstructionStoreError(w, err, true) {
		return
	}
	meta := store.ScopeMeta{
		DataDir:   h.dataDir,
		Scope:     domain.CompoundScopeProject,
		ProjectID: project.ID,
		VaultID:   project.VaultID,
	}
	note, err := h.instructions.Put(r.Context(), meta, store.InstructionName(name), body.Content)
	if writeInstructionStoreError(w, err, false) {
		return
	}
	jsonResponse(w, http.StatusOK, instructionDTO{Content: body.Content, Note: note})
}

func (h *instructionHandlers) getGlobal(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	meta := store.ScopeMeta{
		DataDir: h.dataDir,
		Scope:   domain.CompoundScopeGlobal,
	}
	content, note, err := h.instructions.Get(r.Context(), meta, store.InstructionName(name))
	if writeInstructionStoreError(w, err, true) {
		return
	}
	jsonResponse(w, http.StatusOK, instructionDTO{Content: content, Note: note})
}

func (h *instructionHandlers) putGlobal(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var body instructionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	meta := store.ScopeMeta{
		DataDir: h.dataDir,
		Scope:   domain.CompoundScopeGlobal,
	}
	note, err := h.instructions.Put(r.Context(), meta, store.InstructionName(name), body.Content)
	if writeInstructionStoreError(w, err, false) {
		return
	}
	jsonResponse(w, http.StatusOK, instructionDTO{Content: body.Content, Note: note})
}

func writeInstructionStoreError(w http.ResponseWriter, err error, allowNotFound bool) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, store.ErrValidation):
		http.Error(w, "invalid request", http.StatusBadRequest)
	case allowNotFound && errors.Is(err, store.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	default:
		internalError(w)
	}
	return true
}
