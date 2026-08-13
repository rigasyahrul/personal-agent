package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

type settingsDTO struct {
	Timezone        string `json:"timezone"`
	DefaultProvider string `json:"default_provider"`
	DefaultModelID  string `json:"default_model_id"`
	BackupSchedule  string `json:"backup_schedule"`
}

func SettingsRoutes(mux *http.ServeMux, db *sql.DB, c clock.Clock) {
	s := store.SettingsStore{DB: db}
	authenticated := func(next http.Handler) http.Handler { return requireAuthAt(db, c.Now, next) }
	mux.Handle("GET /api/v1/settings", authenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value, err := s.Get(r.Context())
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		writeSettings(w, value)
	})))
	mux.Handle("PUT /api/v1/settings", authenticated(RequireCSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input settingsDTO
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		value := store.Settings{Timezone: input.Timezone, DefaultProvider: input.DefaultProvider, DefaultModelID: input.DefaultModelID, BackupSchedule: input.BackupSchedule}
		if err := s.Put(r.Context(), value, c.Now()); errors.Is(err, store.ErrInvalidSettings) {
			http.Error(w, "invalid settings", http.StatusBadRequest)
			return
		} else if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		writeSettings(w, value)
	}))))
}

func writeSettings(w http.ResponseWriter, value store.Settings) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(settingsDTO{Timezone: value.Timezone, DefaultProvider: value.DefaultProvider, DefaultModelID: value.DefaultModelID, BackupSchedule: value.BackupSchedule})
}
