package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/rigasyahrul/personal-agent/internal/agent"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

type chatHandlers struct {
	sessions *store.SessionStore
	messages *store.MessageStore
	runs     *store.RunStore
	runner   *agent.Runner
}

func (h *chatHandlers) messagesRoute(w http.ResponseWriter, r *http.Request) {
	sid := r.PathValue("id")
	if r.Method == http.MethodGet {
		if _, err := h.sessions.Get(r.Context(), sid); errors.Is(err, store.ErrNotFound) {
			apiError(w, 404, "session_not_found")
			return
		} else if err != nil {
			internalError(w)
			return
		}
		out, err := h.messages.List(r.Context(), sid)
		if err != nil {
			internalError(w)
			return
		}
		jsonResponse(w, 200, out)
		return
	}
	var in struct {
		Content    string `json:"content"`
		RequestKey string `json:"request_key"`
	}
	if decodeStrictJSON(r, &in) != nil || strings.TrimSpace(in.Content) == "" || strings.TrimSpace(in.RequestKey) == "" {
		apiError(w, 400, "invalid_message")
		return
	}
	runID, err := h.runner.Start(r.Context(), sid, in.RequestKey, in.Content)
	if errors.Is(err, store.ErrSessionBusy) {
		apiError(w, 409, "session_busy")
		return
	}
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrSessionTerminal) {
		if errors.Is(err, store.ErrSessionTerminal) {
			apiError(w, http.StatusConflict, "session_terminal")
			return
		}
		apiError(w, http.StatusNotFound, "session_not_found")
		return
	}
	if err != nil {
		jsonResponse(w, 502, map[string]string{"run_id": runID, "error": "provider_unavailable"})
		return
	}
	jsonResponse(w, 202, map[string]string{"run_id": runID})
}

func (h *chatHandlers) currentRun(w http.ResponseWriter, r *http.Request) {
	sid := r.PathValue("id")
	if _, err := h.sessions.Get(r.Context(), sid); errors.Is(err, store.ErrNotFound) {
		apiError(w, 404, "session_not_found")
		return
	} else if err != nil {
		internalError(w)
		return
	}
	out, err := h.runs.Current(r.Context(), sid)
	if errors.Is(err, store.ErrNotFound) {
		w.WriteHeader(204)
		return
	}
	if err != nil {
		internalError(w)
		return
	}
	jsonResponse(w, 200, out)
}
