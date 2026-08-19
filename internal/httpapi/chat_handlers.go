package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/rigasyahrul/personal-agent/internal/agent"
	"github.com/rigasyahrul/personal-agent/internal/fsroot"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/paths"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

type chatHandlers struct {
	sessions *store.SessionStore
	messages *store.MessageStore
	runs     *store.RunStore
	runner   *agent.Runner
	dataDir  string
}

var errWorkspaceFilesDisabled = errors.New("workspace files disabled")

type workspaceEntry struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
	Size int64  `json:"size"`
}

func (h *chatHandlers) workspaceRoot(r *http.Request) (*fsroot.Root, error) {
	session, err := h.sessions.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		return nil, err
	}
	var grants struct {
		WorkspaceFiles bool `json:"workspace_files"`
	}
	if err := json.Unmarshal([]byte(session.ToolGrantsJSON), &grants); err != nil {
		return nil, err
	}
	if !grants.WorkspaceFiles {
		return nil, errWorkspaceFilesDisabled
	}
	vaultID, projectID := "", ""
	if session.VaultID != nil {
		vaultID = *session.VaultID
	}
	if session.ProjectID != nil {
		projectID = *session.ProjectID
	}
	return fsroot.Open(layout.SessionWorkspace(h.dataDir, session.Home, vaultID, projectID, session.ID))
}

func (h *chatHandlers) workspaceTree(w http.ResponseWriter, r *http.Request) {
	root, err := h.workspaceRoot(r)
	if errors.Is(err, store.ErrNotFound) {
		apiError(w, http.StatusNotFound, "session_not_found")
		return
	}
	if errors.Is(err, errWorkspaceFilesDisabled) {
		apiError(w, http.StatusForbidden, "workspace_files_disabled")
		return
	}
	if err != nil {
		internalError(w)
		return
	}
	defer root.Close()
	entries, err := root.Tree()
	if err != nil {
		apiError(w, http.StatusBadRequest, "unsafe_workspace_tree")
		return
	}
	visible := make([]workspaceEntry, 0, len(entries))
	for _, entry := range entries {
		hiddenTemp := false
		for _, component := range strings.Split(entry.Path, "/") {
			if strings.HasPrefix(component, ".pa-write-") {
				hiddenTemp = true
				break
			}
		}
		if !hiddenTemp {
			visible = append(visible, workspaceEntry{
				Path: entry.Path,
				Kind: entry.Kind,
				Size: entry.Size,
			})
		}
	}
	jsonResponse(w, http.StatusOK, map[string]any{"entries": visible})
}

func (h *chatHandlers) workspaceFile(w http.ResponseWriter, r *http.Request) {
	name, err := paths.ValidateRelPath(r.URL.Query().Get("path"))
	if err != nil {
		apiError(w, http.StatusBadRequest, "invalid_path")
		return
	}
	root, err := h.workspaceRoot(r)
	if errors.Is(err, store.ErrNotFound) {
		apiError(w, http.StatusNotFound, "session_not_found")
		return
	}
	if errors.Is(err, errWorkspaceFilesDisabled) {
		apiError(w, http.StatusForbidden, "workspace_files_disabled")
		return
	}
	if err != nil {
		internalError(w)
		return
	}
	defer root.Close()
	body, err := root.ReadFile(name, paths.MaxMarkdownBytes)
	if err != nil || !utf8.Valid(body) {
		apiError(w, http.StatusBadRequest, "workspace_file_unreadable")
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"path": name, "content": string(body)})
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
	if errors.Is(err, agent.ErrSessionBusy) || errors.Is(err, store.ErrSessionBusy) {
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
		// Admission itself failed before a run was created.
		internalError(w)
		return
	}
	// Start is asynchronous: provider failures terminalize the run in the background.
	// Clients observe status via /runs/current and message history.
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
