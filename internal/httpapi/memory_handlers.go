package httpapi

import (
	"database/sql"
	"net/http"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

type memoryHandlers struct {
	projects *store.ProjectStore
	dataDir  string
}

// MemoryRoutes mounts the thin P1 lessons-index read under /api/v1.
// GET is auth only (not a CSRF mutation).
func MemoryRoutes(mux *http.ServeMux, db *sql.DB, dataDir string, c clock.Clock, auth func(http.Handler) http.Handler) {
	h := &memoryHandlers{
		projects: store.NewProjectStore(db, dataDir, c),
		dataDir:  dataDir,
	}
	if auth == nil {
		auth = func(next http.Handler) http.Handler { return requireAuthAt(db, c.Now, next) }
	}
	mux.Handle("GET /api/v1/projects/{id}/memory/lessons", auth(http.HandlerFunc(h.getLessons)))
}

type memoryLessonsDTO struct {
	Content string `json:"content"`
}

func (h *memoryHandlers) getLessons(w http.ResponseWriter, r *http.Request) {
	project, err := h.projects.Get(r.Context(), r.PathValue("id"))
	if writeStoreError(w, err, true) {
		return
	}
	content, err := store.ReadLessonsIndex(layout.ProjectRoot(h.dataDir, project.VaultID, project.ID))
	if err != nil {
		internalError(w)
		return
	}
	jsonResponse(w, http.StatusOK, memoryLessonsDTO{Content: content})
}
