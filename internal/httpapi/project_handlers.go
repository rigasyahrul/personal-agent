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

type ProjectDTO struct {
	ID           string `json:"id"`
	VaultID      string `json:"vault_id,omitempty"`
	VaultName    string `json:"vault_name,omitempty"`
	Name         string `json:"name"`
	NoteCount    int    `json:"note_count"`
	SessionCount int    `json:"session_count"`
	DueCount     int    `json:"due_count"`
}

type projectHandlers struct {
	vaults   *store.VaultStore
	projects *store.ProjectStore
	clock    clock.Clock
}

func ProjectRoutes(mux *http.ServeMux, db *sql.DB, dataDir string, c clock.Clock) {
	h := projectHandlers{
		vaults:   store.NewVaultStore(db, c),
		projects: store.NewProjectStore(db, dataDir, c),
		clock:    c,
	}
	authenticated := func(next http.Handler) http.Handler { return requireAuthAt(db, c.Now, next) }
	mux.Handle("GET /api/v1/vaults", authenticated(http.HandlerFunc(h.listVaults)))
	mux.Handle("POST /api/v1/vaults", authenticated(RequireCSRF(http.HandlerFunc(h.createVault))))
	mux.Handle("GET /api/v1/projects", authenticated(http.HandlerFunc(h.listProjects)))
	mux.Handle("POST /api/v1/projects", authenticated(RequireCSRF(http.HandlerFunc(h.createProject))))
	mux.Handle("GET /api/v1/projects/{id}", authenticated(http.HandlerFunc(h.getProject)))
	mux.Handle("GET /api/v1/home", authenticated(http.HandlerFunc(h.home)))
}

func (h projectHandlers) listVaults(w http.ResponseWriter, r *http.Request) {
	vaults, err := h.vaults.List(r.Context())
	if err != nil {
		internalError(w)
		return
	}
	jsonResponse(w, http.StatusOK, vaults)
}

func (h projectHandlers) createVault(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	vault, err := h.vaults.Create(r.Context(), input.Name)
	if !writeStoreError(w, err, false) {
		jsonResponse(w, http.StatusCreated, vault)
	}
}

func (h projectHandlers) listProjects(w http.ResponseWriter, r *http.Request) {
	projects, err := h.projects.List(r.Context())
	if err != nil {
		internalError(w)
		return
	}
	dtos, err := h.projectDTOs(r, projects)
	if err != nil {
		internalError(w)
		return
	}
	jsonResponse(w, http.StatusOK, dtos)
}

func (h projectHandlers) createProject(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name    string  `json:"name"`
		VaultID *string `json:"vault_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	vaultID := ""
	if input.VaultID != nil {
		vaultID = *input.VaultID
	}
	project, err := h.projects.Create(r.Context(), input.Name, vaultID)
	if writeStoreError(w, err, false) {
		return
	}
	dtos, err := h.projectDTOs(r, []domain.Project{project})
	if err != nil {
		internalError(w)
		return
	}
	jsonResponse(w, http.StatusCreated, dtos[0])
}

func (h projectHandlers) getProject(w http.ResponseWriter, r *http.Request) {
	project, err := h.projects.Get(r.Context(), r.PathValue("id"))
	if writeStoreError(w, err, true) {
		return
	}
	dtos, err := h.projectDTOs(r, []domain.Project{project})
	if err != nil {
		internalError(w)
		return
	}
	jsonResponse(w, http.StatusOK, dtos[0])
}

func (h projectHandlers) home(w http.ResponseWriter, r *http.Request) {
	projects, err := h.projects.List(r.Context())
	if err != nil {
		internalError(w)
		return
	}
	dtos, err := h.projectDTOs(r, projects)
	if err != nil {
		internalError(w)
		return
	}
	jsonResponse(w, http.StatusOK, struct {
		Projects    []ProjectDTO `json:"projects"`
		DueCount    int          `json:"due_count"`
		GeneratedAt time.Time    `json:"generated_at"`
	}{Projects: dtos, GeneratedAt: h.clock.Now().UTC()})
}

func (h projectHandlers) projectDTOs(r *http.Request, projects []domain.Project) ([]ProjectDTO, error) {
	vaults, err := h.vaults.List(r.Context())
	if err != nil {
		return nil, err
	}
	vaultNames := make(map[string]string, len(vaults))
	for _, vault := range vaults {
		vaultNames[vault.ID] = vault.Name
	}
	dtos := make([]ProjectDTO, len(projects))
	for i, project := range projects {
		dtos[i] = ProjectDTO{ID: project.ID, VaultID: project.VaultID, VaultName: vaultNames[project.VaultID], Name: project.Name}
	}
	return dtos, nil
}

func writeStoreError(w http.ResponseWriter, err error, allowNotFound bool) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, store.ErrValidation):
		http.Error(w, "invalid request", http.StatusBadRequest)
	case allowNotFound && errors.Is(err, store.ErrNotFound):
		http.Error(w, "project not found", http.StatusNotFound)
	default:
		internalError(w)
	}
	return true
}

func internalError(w http.ResponseWriter) {
	http.Error(w, "database error", http.StatusInternalServerError)
}

func jsonResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
