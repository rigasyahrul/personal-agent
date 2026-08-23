package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/agent"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/compound"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

type compoundHandlers struct {
	sessions  *store.SessionStore
	compound  *store.CompoundStore
	publisher *compound.Publisher
	runner    *agent.Runner
	clock     clock.Clock
}

type compoundCreateRequest struct {
	RequestKey  string               `json:"request_key"`
	UserContext string               `json:"user_context"`
	Items       []store.CompoundItem `json:"items"`
}

type compoundDecideRequest struct {
	RequestKey string               `json:"request_key"`
	Decision   string               `json:"decision"`
	Items      []store.CompoundItem `json:"items"`
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
		h.generate(w, r, sess, in)
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

func (h *compoundHandlers) generate(w http.ResponseWriter, r *http.Request, sess domain.Session, in compoundCreateRequest) {
	if h.runner == nil {
		internalError(w)
		return
	}
	userMsg := strings.TrimSpace(in.UserContext)
	if userMsg == "" {
		userMsg = "Generate compound proposal items."
	}
	_, err := h.runner.StartCompound(r.Context(), sess.ID, strings.TrimSpace(in.RequestKey), userMsg)
	if errors.Is(err, agent.ErrSessionBusy) || errors.Is(err, store.ErrSessionBusy) {
		apiError(w, http.StatusConflict, "session_busy")
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		apiError(w, http.StatusNotFound, "session_not_found")
		return
	}
	if errors.Is(err, store.ErrSessionTerminal) {
		apiError(w, http.StatusForbidden, "session_terminal")
		return
	}
	if err != nil {
		internalError(w)
		return
	}
	got, err := h.compound.GetBySessionRequest(r.Context(), sess.ID, strings.TrimSpace(in.RequestKey))
	if errors.Is(err, store.ErrNotFound) {
		apiError(w, http.StatusBadRequest, "compound_generate_failed")
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

func (h *compoundHandlers) get(w http.ResponseWriter, r *http.Request) {
	sid := r.PathValue("id")
	if _, ok := h.loadMutableSession(w, r, sid); !ok {
		return
	}
	got, ok := h.loadSessionProposal(w, r, sid, r.PathValue("proposal_id"))
	if !ok {
		return
	}
	got, err := h.recoverIfNeeded(r.Context(), got)
	if err != nil {
		internalError(w)
		return
	}
	h.writeProposal(w, got)
}

func (h *compoundHandlers) decide(w http.ResponseWriter, r *http.Request) {
	sid := r.PathValue("id")
	if _, ok := h.loadMutableSession(w, r, sid); !ok {
		return
	}
	var in compoundDecideRequest
	if decodeStrictJSON(r, &in) != nil || strings.TrimSpace(in.RequestKey) == "" {
		apiError(w, http.StatusBadRequest, "invalid_body")
		return
	}
	switch in.Decision {
	case "approve", "reject":
	default:
		apiError(w, http.StatusBadRequest, "invalid_body")
		return
	}
	before, ok := h.loadSessionProposal(w, r, sid, r.PathValue("proposal_id"))
	if !ok {
		return
	}
	now := time.Now().UTC()
	if h.clock != nil {
		now = h.clock.Now().UTC()
	}
	got, err := h.compound.Decide(r.Context(), store.DecideInput{
		ProposalID: before.ID,
		RequestKey: strings.TrimSpace(in.RequestKey),
		Decision:   in.Decision,
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
	if errors.Is(err, store.ErrNotFound) {
		apiError(w, http.StatusNotFound, "proposal_not_found")
		return
	}
	if err != nil {
		internalError(w)
		return
	}
	if in.Decision == "approve" && got.Status == domain.CompoundStatusApproved && got.FinishedAt == nil {
		// Same recovery as GET: re-drive publish after crash between CAS and finish.
		_ = h.publishAndFinish(context.Background(), got)
		reloaded, loadErr := h.compound.Get(r.Context(), got.ID)
		if loadErr != nil {
			internalError(w)
			return
		}
		got = reloaded
	}
	h.writeProposal(w, got)
}

func (h *compoundHandlers) loadSessionProposal(w http.ResponseWriter, r *http.Request, sessionID, proposalID string) (domain.CompoundProposal, bool) {
	got, err := h.compound.Get(r.Context(), proposalID)
	if errors.Is(err, store.ErrNotFound) {
		apiError(w, http.StatusNotFound, "proposal_not_found")
		return domain.CompoundProposal{}, false
	}
	if err != nil {
		internalError(w)
		return domain.CompoundProposal{}, false
	}
	if got.SessionID != sessionID {
		apiError(w, http.StatusNotFound, "proposal_not_found")
		return domain.CompoundProposal{}, false
	}
	return got, true
}

func (h *compoundHandlers) recoverIfNeeded(ctx context.Context, p domain.CompoundProposal) (domain.CompoundProposal, error) {
	if p.Status != domain.CompoundStatusApproved || p.FinishedAt != nil {
		return p, nil
	}
	_ = h.publishAndFinish(ctx, p)
	return h.compound.Get(ctx, p.ID)
}

func (h *compoundHandlers) publishAndFinish(ctx context.Context, p domain.CompoundProposal) error {
	now := time.Now().UTC()
	if h.clock != nil {
		now = h.clock.Now().UTC()
	}
	if h.publisher == nil {
		msg := "publisher unavailable"
		_ = h.compound.MarkFinished(ctx, p.ID, string(domain.CompoundStatusFailed), msg, now)
		return errors.New(msg)
	}
	if err := h.publisher.PublishApproved(ctx, p); err != nil {
		_ = h.compound.MarkFinished(ctx, p.ID, string(domain.CompoundStatusFailed), err.Error(), now)
		return err
	}
	return h.compound.MarkFinished(ctx, p.ID, string(domain.CompoundStatusApproved), "", now)
}

func (h *compoundHandlers) writeProposal(w http.ResponseWriter, p domain.CompoundProposal) {
	dto, err := proposalDTO(p)
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
