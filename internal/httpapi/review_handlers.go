package httpapi

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/review"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

type reviewHandlers struct {
	db    *sql.DB
	queue review.Queue
	store store.ReviewStore
}

func (h reviewHandlers) queueDue(w http.ResponseWriter, r *http.Request) {
	scope, err := review.ParseScope(r.URL.Query().Get("scope"))
	if err != nil {
		apiError(w, 400, "invalid_scope")
		return
	}
	out, err := h.queue.Due(r.Context(), scope)
	if err != nil {
		internalError(w)
		return
	}
	jsonResponse(w, 200, out)
}
func (h reviewHandlers) rate(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Rating     domain.Rating `json:"rating"`
		RequestKey string        `json:"request_key"`
		RowVersion *int64        `json:"row_version"`
		DurationMS *int64        `json:"duration_ms"`
	}
	if err := decodeStrictJSON(r, &in); err != nil || strings.TrimSpace(in.RequestKey) == "" || in.RowVersion == nil || *in.RowVersion < 0 || in.DurationMS == nil || *in.DurationMS < 0 || !validHTTPRating(in.Rating) {
		apiError(w, 400, "invalid_request")
		return
	}
	out, err := h.store.Rate(r.Context(), r.PathValue("id"), in.RequestKey, *in.RowVersion, in.Rating, *in.DurationMS)
	var conflict *store.RowVersionConflict
	switch {
	case errors.As(err, &conflict):
		apiError(w, 409, "row_version_conflict")
	case errors.Is(err, sql.ErrNoRows) || errors.Is(err, store.ErrNotFound):
		apiError(w, 404, "review_item_not_found")
	case errors.Is(err, store.ErrValidation):
		apiError(w, 400, "invalid_request")
	case err != nil:
		internalError(w)
	default:
		jsonResponse(w, 200, out)
	}
}
func validHTTPRating(v domain.Rating) bool {
	return v == domain.RatingAgain || v == domain.RatingHard || v == domain.RatingGood || v == domain.RatingEasy
}
func (h reviewHandlers) suspend(w http.ResponseWriter, r *http.Request) {
	if err := decodeEmptyJSON(r); err != nil {
		apiError(w, 400, "invalid_request")
		return
	}
	err := h.store.Suspend(r.Context(), r.PathValue("id"))
	switch {
	case errors.Is(err, store.ErrNotFound):
		apiError(w, 404, "review_item_not_found")
	case errors.Is(err, store.ErrConflict):
		apiError(w, 409, "review_item_inactive")
	case err != nil:
		internalError(w)
	default:
		w.WriteHeader(204)
	}
}
func (h reviewHandlers) retry(w http.ResponseWriter, r *http.Request) {
	if err := decodeEmptyJSON(r); err != nil {
		apiError(w, 400, "invalid_request")
		return
	}
	err := store.RetryReviewPending(r.Context(), h.db, r.PathValue("id"))
	switch {
	case errors.Is(err, store.ErrNotFound):
		apiError(w, 404, "review_pending_not_found")
	case errors.Is(err, store.ErrConflict):
		apiError(w, 409, "review_pending_not_failed")
	case err != nil:
		internalError(w)
	default:
		w.WriteHeader(204)
	}
}
func decodeEmptyJSON(r *http.Request) error { var in struct{}; return decodeStrictJSON(r, &in) }
