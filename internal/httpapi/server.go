package httpapi

import (
	"database/sql"
	"net/http"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/publish"
)

type ServerDeps struct {
	DB             *sql.DB
	DataDir        string
	Clock          clock.Clock
	BootstrapToken string
	SecureCookies  bool
	Static         http.FileSystem
	Publish        *publish.Machine
}

func New(deps ServerDeps) http.Handler {
	if deps.Publish == nil {
		deps.Publish = &publish.Machine{DB: deps.DB, DataDir: deps.DataDir, Clock: deps.Clock}
	}
	mux := http.NewServeMux()
	AuthRoutes(mux, AuthDeps{
		DB:             deps.DB,
		Clock:          deps.Clock,
		BootstrapToken: deps.BootstrapToken,
		SecureCookies:  deps.SecureCookies,
	})
	SettingsRoutes(mux, deps.DB, deps.Clock)
	ProjectRoutes(mux, deps.DB, deps.DataDir, deps.Clock)
	NoteRoutes(mux, deps.DB, deps.DataDir, deps.Clock, deps.Publish)
	mux.Handle("GET /health", healthHandler(deps.DataDir))
	if deps.Static != nil {
		mux.Handle("GET /", http.FileServer(deps.Static))
	}
	return mux
}
