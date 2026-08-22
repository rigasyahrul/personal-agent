package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

type compoundHandlers struct {
	sessions *store.SessionStore
	compound *store.CompoundStore
	clock    clock.Clock
}

type compoundCreateRequest struct {
	RequestKey  string               `json:"request_key"`
	UserContext string               `json:"user_context"`
	Items       []store.CompoundItem `json:"items"`
}

type compoundProposalDTO struct {
	ID         string               `json:"id"`
	SessionID  string               `json:"session_id"`
	Scope      string               `json:"scope"`
	ProjectID  string               `json:"project_id"`
	VaultID    string               `json:"vault_id"`
	Status     string               `json:"status"`
	RequestKey string               `json:"request_key"`
	Items      []store.CompoundItem `json:"items"`
	Error      string               `json:"error,omitempty"`
	CreatedAt  time.Time            `json:"created_at"`
	DecidedAt  *time.Time           `json:"decided_at,omitempty"`
	FinishedAt *time.Time           `json:"finished_at,omitempty"`
}

func (h *compoundHandlers) create(w http.ResponseWriter, r *http.Request) {
	sid := r.PathValue("id")
	sess, ok := h.loadMutableSession(w, r, sid)
	if !ok {
		return
	}
	var in compoundCreateRequest
	if decodeStrictJSON(r, &in) != nil || strings.TrimSpace(in.RequestKey) == "" {
		apiError(w, http.StatusBadRequest, "invalid_body")
		return
	}
	if len(in.Items) == 0 {
		apiError(w, http.StatusNotImplemented, "compound_generate_not_implemented")
		return
	}

	scope, projectID, vaultID, err := compoundScopeFromSession(sess)
	if err != nil {
		apiError(w, http.StatusBadRequest, "invalid_session_home")
		return
	}
	now := time.Now().UTC()
	if h.clock != nil {
		now = h.clock.Now().UTC()
	}
	got, err := h.compound.CreatePending(r.Context(), store.CreateProposalInput{
		SessionID:  sess.ID,
		RequestKey: strings.TrimSpace(in.RequestKey),
		Scope:      scope,
		ProjectID:  projectID,
		VaultID:    vaultID,
		Items:      in.Items,
		Now:        now,
	})
	if errors.Is(err, store.ErrValidation) {
		apiError(w, http.StatusBadRequest, "invalid_items")
		return
	}
	if errors.Is(err, store.ErrConflict) {
		apiError(w, http.StatusConflict, "conflict")
		return
	}
	if err != nil {
		internalError(w)
		return
	}
	dto, err := proposalDTO(got)
	if err != nil {
		internalError(w)
		return
	}
	jsonResponse(w, http.StatusOK, dto)
}

func (h *compoundHandlers) loadMutableSession(w http.ResponseWriter, r *http.Request, id string) (domain.Session, bool) {
	sess, err := h.sessions.Get(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		apiError(w, http.StatusNotFound, "session_not_found")
		return domain.Session{}, false
	}
	if err != nil {
		internalError(w)
		return domain.Session{}, false
	}
	if sess.Status != "active" || sess.DeletedAt != nil {
		apiError(w, http.StatusForbidden, "session_terminal")
		return domain.Session{}, false
	}
	return sess, true
}

func compoundScopeFromSession(sess domain.Session) (domain.CompoundScope, string, string, error) {
	switch sess.Home {
	case layout.SessionHome("project"):
		pid, vid := deref(sess.ProjectID), deref(sess.VaultID)
		return domain.CompoundScopeProject, pid, vid, nil
	case layout.SessionHome("vault"):
		return domain.CompoundScopeVault, "", deref(sess.VaultID), nil
	case layout.SessionHome("global"):
		return domain.CompoundScopeGlobal, "", "", nil
	default:
		return "", "", "", errors.New("invalid home")
	}
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func proposalDTO(p domain.CompoundProposal) (compoundProposalDTO, error) {
	var items []store.CompoundItem
	if strings.TrimSpace(p.ItemsJSON) != "" {
		if err := json.Unmarshal([]byte(p.ItemsJSON), &items); err != nil {
			return compoundProposalDTO{}, err
		}
	}
	return compoundProposalDTO{
		ID:         p.ID,
		SessionID:  p.SessionID,
		Scope:      string(p.Scope),
		ProjectID:  p.ProjectID,
		VaultID:    p.VaultID,
		Status:     string(p.Status),
		RequestKey: p.RequestKey,
		Items:      items,
		Error:      p.Error,
		CreatedAt:  p.CreatedAt,
		DecidedAt:  p.DecidedAt,
		FinishedAt: p.FinishedAt,
	}, nil
}
