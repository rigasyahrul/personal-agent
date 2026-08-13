package httpapi

import (
	"net/http"

	"github.com/rigasyahrul/personal-agent/internal/backup"
	"github.com/rigasyahrul/personal-agent/internal/domain"
)

type backupHandlers struct {
	service *backup.Service
}

func (h backupHandlers) list(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		internalError(w)
		return
	}
	runs, err := h.service.List(r.Context())
	if err != nil {
		internalError(w)
		return
	}
	if runs == nil {
		runs = []domain.BackupRun{}
	}
	summary := summarizeBackups(runs)
	jsonResponse(w, http.StatusOK, map[string]any{
		"backups":      runs,
		"last_success": summary.LastSuccess,
		"last_failure": summary.LastFailure,
	})
}

func (h backupHandlers) create(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		internalError(w)
		return
	}
	run, err := h.service.Run(r.Context())
	if err != nil {
		// Still return the run body (may be failed) for UI status.
		jsonResponse(w, http.StatusInternalServerError, run)
		return
	}
	jsonResponse(w, http.StatusCreated, run)
}

type backupSummary struct {
	LastSuccess *domain.BackupRun `json:"last_success"`
	LastFailure *domain.BackupRun `json:"last_failure"`
}

func summarizeBackups(runs []domain.BackupRun) backupSummary {
	var s backupSummary
	for i := range runs {
		r := runs[i]
		if r.Status == "succeeded" && s.LastSuccess == nil {
			cp := r
			s.LastSuccess = &cp
		}
		if r.Status == "failed" && s.LastFailure == nil {
			cp := r
			s.LastFailure = &cp
		}
	}
	return s
}
