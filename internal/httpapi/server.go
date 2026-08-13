package httpapi

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/rigasyahrul/personal-agent/internal/clock"
)

type ServerDeps struct {
	DB             *sql.DB
	DataDir        string
	Clock          clock.Clock
	BootstrapToken string
	SecureCookies  bool
	Static         http.FileSystem
}

func New(deps ServerDeps) http.Handler {
	mux := http.NewServeMux()
	AuthRoutes(mux, AuthDeps{
		DB:             deps.DB,
		Clock:          deps.Clock,
		BootstrapToken: deps.BootstrapToken,
		SecureCookies:  deps.SecureCookies,
	})
	SettingsRoutes(mux, deps.DB, deps.Clock)
	mux.Handle("GET /health", healthHandler(deps.DataDir))
	mux.HandleFunc("GET /api/v1/home", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"projects":     []any{},
			"due_count":    0,
			"last_project": nil,
		})
	})
	if deps.Static != nil {
		mux.Handle("GET /", http.FileServer(deps.Static))
	}
	return mux
}
