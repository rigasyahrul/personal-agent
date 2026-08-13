package httpapi

import (
	"database/sql"
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
	ProjectRoutes(mux, deps.DB, deps.DataDir, deps.Clock)
	mux.Handle("GET /health", healthHandler(deps.DataDir))
	if deps.Static != nil {
		mux.Handle("GET /", http.FileServer(deps.Static))
	}
	return mux
}
