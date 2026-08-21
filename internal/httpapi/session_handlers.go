package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/rigasyahrul/personal-agent/internal/config"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

type sessionHandlers struct {
	sessions *store.SessionStore
	models   []config.ModelRef
}

type sessionCreateRequest struct {
	Home            string          `json:"home"`
	Title           string          `json:"title"`
	Provider        string          `json:"provider"`
	ModelID         string          `json:"model_id"`
	ModelParameters map[string]any  `json:"model_parameters"`
	ToolGrants      map[string]bool `json:"tool_grants"`
}

func (h *sessionHandlers) modelsList(w http.ResponseWriter, _ *http.Request) {
	models := h.models
	out := make([]struct {
		Provider string `json:"provider"`
		ModelID  string `json:"model_id"`
	}, 0, len(models))
	for _, model := range models {
		out = append(out, struct {
			Provider string `json:"provider"`
			ModelID  string `json:"model_id"`
		}{model.Provider, model.ModelID})
	}
	jsonResponse(w, http.StatusOK, struct {
		Models any `json:"models"`
	}{out})
}

func (h *sessionHandlers) projectSessions(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	if err := h.projectExists(r, projectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			apiError(w, http.StatusNotFound, "project_not_found")
			return
		}
		internalError(w)
		return
	}
	if r.Method == http.MethodGet {
		out, err := h.sessions.ListByProject(r.Context(), projectID)
		if err != nil {
			internalError(w)
			return
		}
		jsonResponse(w, http.StatusOK, out)
		return
	}
	if len(h.models) == 0 {
		apiError(w, http.StatusServiceUnavailable, "no_models_configured")
		return
	}
	var in sessionCreateRequest
	if err := decodeStrictJSON(r, &in); err != nil {
		apiError(w, http.StatusBadRequest, "invalid_body")
		return
	}
	if strings.TrimSpace(in.Title) == "" {
		apiError(w, http.StatusBadRequest, "invalid_title")
		return
	}
	if in.Home != "" && in.Home != "project" {
		apiError(w, http.StatusBadRequest, "invalid_scope")
		return
	}
	if !configuredModel(h.models, in.Provider, in.ModelID) {
		apiError(w, http.StatusBadRequest, "unknown_model")
		return
	}
	for grant := range in.ToolGrants {
		if grant != "workspace_files" {
			apiError(w, http.StatusBadRequest, "invalid_tool_grants")
			return
		}
	}
	if in.ModelParameters == nil {
		in.ModelParameters = map[string]any{}
	}
	if in.ToolGrants == nil {
		in.ToolGrants = map[string]bool{"workspace_files": false}
	}
	parameters, err := json.Marshal(in.ModelParameters)
	if err != nil {
		apiError(w, 400, "invalid_model_parameters")
		return
	}
	grants, _ := json.Marshal(in.ToolGrants)
	out, err := h.sessions.CreateProject(r.Context(), store.CreateSessionInput{ProjectID: projectID, Title: in.Title, Provider: in.Provider, ModelID: in.ModelID, ModelParametersJSON: string(parameters), ToolGrantsJSON: string(grants)})
	if errors.Is(err, store.ErrNotFound) {
		apiError(w, 404, "project_not_found")
		return
	}
	if errors.Is(err, store.ErrValidation) {
		apiError(w, 400, "unknown_model")
		return
	}
	if err != nil {
		internalError(w)
		return
	}
	jsonResponse(w, http.StatusCreated, out)
}

func (h *sessionHandlers) projectExists(r *http.Request, projectID string) error {
	var exists int
	return h.sessions.DB.QueryRowContext(r.Context(), `SELECT 1 FROM projects WHERE id=?`, projectID).Scan(&exists)
}

func decodeStrictJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func (h *sessionHandlers) session(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if r.Method == http.MethodDelete {
		err := h.sessions.Delete(r.Context(), id)
		if errors.Is(err, store.ErrNotFound) {
			apiError(w, 404, "session_not_found")
			return
		}
		if errors.Is(err, store.ErrSessionBusy) {
			apiError(w, 409, "session_busy")
			return
		}
		if err != nil {
			internalError(w)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method == http.MethodPatch {
		var in struct {
			Title string `json:"title"`
		}
		if err := decodeStrictJSON(r, &in); err != nil {
			apiError(w, http.StatusBadRequest, "invalid_body")
			return
		}
		if strings.TrimSpace(in.Title) == "" {
			apiError(w, http.StatusBadRequest, "invalid_title")
			return
		}
		out, err := h.sessions.RenameTitle(r.Context(), id, in.Title)
		if errors.Is(err, store.ErrNotFound) {
			apiError(w, http.StatusNotFound, "session_not_found")
			return
		}
		if errors.Is(err, store.ErrValidation) {
			apiError(w, http.StatusBadRequest, "invalid_title")
			return
		}
		if err != nil {
			internalError(w)
			return
		}
		jsonResponse(w, http.StatusOK, out)
		return
	}
	out, err := h.sessions.Get(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		apiError(w, 404, "session_not_found")
		return
	}
	if err != nil {
		internalError(w)
		return
	}
	jsonResponse(w, 200, out)
}

func configuredModel(models []config.ModelRef, provider, modelID string) bool {
	for _, m := range models {
		if m.Provider == provider && m.ModelID == modelID {
			return true
		}
	}
	return false
}
func apiError(w http.ResponseWriter, status int, code string) {
	jsonResponse(w, status, map[string]string{"error": code})
}
