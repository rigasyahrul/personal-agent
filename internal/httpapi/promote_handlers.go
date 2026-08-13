package httpapi

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/ids"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/publish"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

type promoteHandlers struct {
	db       *sql.DB
	machine  *publish.Machine
	sessions *store.SessionStore
}

type promoteRequest struct {
	WorkspacePath      string            `json:"workspace_path"`
	TargetRelativePath string            `json:"target_relative_path"`
	ReviewMode         domain.ReviewMode `json:"review_mode"`
}

type promoteFingerprint struct {
	SessionID          string            `json:"session_id"`
	WorkspacePath      string            `json:"workspace_path"`
	TargetProjectID    string            `json:"target_project_id"`
	TargetRelativePath string            `json:"target_relative_path"`
	ReviewMode         domain.ReviewMode `json:"review_mode"`
}

func (h promoteHandlers) create(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"code": "idempotency_key_required"})
		return
	}
	var req promoteRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"code": "invalid_request"})
		return
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"code": "invalid_request"})
		return
	}
	session, err := h.sessions.Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, store.ErrNotFound) {
		jsonResponse(w, http.StatusNotFound, map[string]string{"code": "session_not_found"})
		return
	}
	if err != nil {
		internalError(w)
		return
	}
	if session.Status != "active" {
		jsonResponse(w, http.StatusConflict, map[string]string{"code": "session_terminal"})
		return
	}
	if session.Home != layout.SessionHome("project") || session.ProjectID == nil || strings.TrimSpace(*session.ProjectID) == "" {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"code": "invalid_session_scope"})
		return
	}
	wire := promoteFingerprint{session.ID, req.WorkspacePath, *session.ProjectID, req.TargetRelativePath, req.ReviewMode}
	canonical, err := json.Marshal(wire)
	if err != nil {
		internalError(w)
		return
	}
	sum := sha256.Sum256(canonical)
	in := publish.PublishInput{OpID: ids.NewID(), RequestKey: key, RequestFingerprint: hex.EncodeToString(sum[:]), Kind: "promote", SessionID: session.ID, WorkspacePath: req.WorkspacePath, TargetProjectID: *session.ProjectID, TargetRelPath: req.TargetRelativePath, ReviewMode: req.ReviewMode, NoteID: ids.NewID()}
	_, _, runErr := h.machine.Run(r.Context(), in)
	if errors.Is(runErr, publish.ErrInvalid) {
		jsonResponse(w, http.StatusBadRequest, map[string]string{"code": "invalid_request"})
		return
	}
	if errors.Is(runErr, publish.ErrConflict) {
		code := "publication_conflict"
		var conflict *publish.ConflictError
		if errors.As(runErr, &conflict) {
			code = conflict.Code
		}
		jsonResponse(w, http.StatusConflict, map[string]string{"code": code})
		return
	}
	if errors.Is(runErr, sql.ErrNoRows) {
		jsonResponse(w, http.StatusNotFound, map[string]string{"code": "source_not_found"})
		return
	}
	if runErr != nil {
		internalError(w)
		return
	}
	// Always resolve the winning durable row. Machine instances only serialize
	// themselves, so another process may have won the request-key insert.
	durable, err := (store.PromoteStore{DB: h.db}).ByKey(r.Context(), key)
	if err != nil {
		internalError(w)
		return
	}
	jsonResponse(w, http.StatusAccepted, map[string]string{"operation_id": durable.ID, "note_id": durable.NoteID, "status": durable.Status})
}

type operationStatusDTO struct {
	OperationID       string  `json:"operation_id"`
	NoteID            string  `json:"note_id"`
	PublicationStatus string  `json:"publication_status"`
	NoteStatus        *string `json:"note_status"`
	PendingID         *string `json:"pending_id"`
	PendingStatus     *string `json:"pending_status"`
	Badge             string  `json:"badge"`
	RetryCards        bool    `json:"retry_cards"`
}

func (h promoteHandlers) status(w http.ResponseWriter, r *http.Request) {
	op, err := (store.PromoteStore{DB: h.db}).ByID(r.Context(), r.PathValue("id"))
	if store.IsNoRows(err) {
		op, err = (store.DirectStore{DB: h.db}).ByID(r.Context(), r.PathValue("id"))
	}
	if store.IsNoRows(err) {
		jsonResponse(w, http.StatusNotFound, map[string]string{"code": "operation_not_found"})
		return
	}
	if err != nil {
		internalError(w)
		return
	}
	dto := operationStatusDTO{OperationID: op.ID, NoteID: op.NoteID, PublicationStatus: op.Status}
	if op.Status == "failed" {
		dto.Badge = "Promote failed — Retry"
		jsonResponse(w, http.StatusOK, dto)
		return
	}
	if op.Status != "completed" {
		dto.Badge = "Promoting…"
		jsonResponse(w, http.StatusOK, dto)
		return
	}
	var noteStatus string
	if err := h.db.QueryRowContext(r.Context(), `SELECT status FROM notes WHERE id=?`, op.NoteID).Scan(&noteStatus); err != nil {
		internalError(w)
		return
	}
	dto.NoteStatus = &noteStatus
	if noteStatus != "ready" {
		internalError(w)
		return
	}
	if op.ReviewMode == "bites" {
		pending, err := store.ReviewPendingForPublication(r.Context(), h.db, op.NoteID, op.FrozenSHA)
		if err != nil {
			internalError(w)
			return
		}
		dto.PendingID, dto.PendingStatus = &pending.ID, &pending.Status
		switch pending.Status {
		case "pending", "leased":
			dto.Badge = "Note saved; cards pending…"
		case "failed":
			dto.Badge, dto.RetryCards = "Cards failed — Retry cards", true
		case "completed":
			dto.Badge = "Ready"
		default:
			internalError(w)
			return
		}
	} else {
		dto.Badge = "Ready"
	}
	jsonResponse(w, http.StatusOK, dto)
}
