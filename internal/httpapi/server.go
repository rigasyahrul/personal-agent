package httpapi

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/agent"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/config"
	"github.com/rigasyahrul/personal-agent/internal/publish"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

type ServerDeps struct {
	DB             *sql.DB
	DataDir        string
	Clock          clock.Clock
	BootstrapToken string
	SecureCookies  bool
	Static         http.FileSystem
	Publish        *publish.Machine
	Models         []config.ModelRef
	Provider       agent.Provider
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
	now := func() time.Time { return deps.Clock.Now().UTC() }
	sessions := &store.SessionStore{DB: deps.DB, DataDir: deps.DataDir, Now: now, Models: deps.Models}
	messages := &store.MessageStore{DB: deps.DB, Now: now}
	runs := &store.RunStore{DB: deps.DB, Now: now}
	runner := &agent.Runner{DB: deps.DB, DataDir: deps.DataDir, Provider: deps.Provider, Messages: messages, Runs: runs, Sessions: sessions, Clock: deps.Clock}
	sh := &sessionHandlers{sessions: sessions, models: deps.Models}
	ch := &chatHandlers{sessions: sessions, messages: messages, runs: runs, runner: runner, dataDir: deps.DataDir}
	auth := func(next http.Handler) http.Handler { return requireAuthAt(deps.DB, now, next) }
	mutation := func(next http.Handler) http.Handler { return auth(RequireCSRF(next)) }
	mux.Handle("GET /api/v1/models", auth(http.HandlerFunc(sh.modelsList)))
	mux.Handle("GET /api/v1/projects/{id}/sessions", auth(http.HandlerFunc(sh.projectSessions)))
	mux.Handle("POST /api/v1/projects/{id}/sessions", mutation(http.HandlerFunc(sh.projectSessions)))
	mux.Handle("GET /api/v1/sessions/{id}", auth(http.HandlerFunc(sh.session)))
	mux.Handle("DELETE /api/v1/sessions/{id}", mutation(http.HandlerFunc(sh.session)))
	mux.Handle("GET /api/v1/sessions/{id}/messages", auth(http.HandlerFunc(ch.messagesRoute)))
	mux.Handle("POST /api/v1/sessions/{id}/messages", mutation(http.HandlerFunc(ch.messagesRoute)))
	mux.Handle("GET /api/v1/sessions/{id}/runs/current", auth(http.HandlerFunc(ch.currentRun)))
	mux.Handle("GET /api/v1/sessions/{id}/workspace/tree", auth(http.HandlerFunc(ch.workspaceTree)))
	mux.Handle("GET /api/v1/sessions/{id}/workspace/file", auth(http.HandlerFunc(ch.workspaceFile)))
	mux.Handle("GET /health", healthHandler(deps.DataDir))
	if deps.Static != nil {
		mux.Handle("GET /", http.FileServer(deps.Static))
	}
	return mux
}
