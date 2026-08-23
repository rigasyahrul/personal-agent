package httpapi

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/agent"
	"github.com/rigasyahrul/personal-agent/internal/backup"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/compound"
	"github.com/rigasyahrul/personal-agent/internal/config"
	"github.com/rigasyahrul/personal-agent/internal/publish"
	"github.com/rigasyahrul/personal-agent/internal/review"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

type ServerDeps struct {
	DB             *sql.DB
	DataDir        string
	Clock          clock.Clock
	BootstrapToken string
	SecureCookies  bool
	Static         http.FileSystem
	UI             http.Handler
	Publish        *publish.Machine
	Models         []config.ModelRef
	Provider       agent.Provider
	Backup         *backup.Service
	Barrier        *backup.Barrier
	// BackupSinkConfigured is a non-secret boolean for Settings UI.
	BackupSinkConfigured bool
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
	SettingsRoutes(mux, deps.DB, deps.Clock, deps.Backup, deps.BackupSinkConfigured)
	ProjectRoutes(mux, deps.DB, deps.DataDir, deps.Clock, deps.Barrier)
	NoteRoutes(mux, deps.DB, deps.DataDir, deps.Clock, deps.Publish, deps.Barrier)
	now := func() time.Time { return deps.Clock.Now().UTC() }
	if deps.Publish != nil && deps.Publish.SessionLocks == nil {
		deps.Publish.SessionLocks = store.NewSessionLocks()
	}
	var sessionLocks *store.SessionLocks
	if deps.Publish != nil {
		sessionLocks = deps.Publish.SessionLocks
	} else {
		sessionLocks = store.NewSessionLocks()
	}
	sessions := &store.SessionStore{DB: deps.DB, DataDir: deps.DataDir, Now: now, Models: deps.Models, Barrier: deps.Barrier, Locks: sessionLocks}
	messages := &store.MessageStore{DB: deps.DB, Now: now}
	runs := &store.RunStore{DB: deps.DB, Now: now}
	compounds := &store.CompoundStore{DB: deps.DB, Clock: deps.Clock, Barrier: deps.Barrier}
	runner := &agent.Runner{DB: deps.DB, DataDir: deps.DataDir, Provider: deps.Provider, Messages: messages, Runs: runs, Sessions: sessions, Compound: compounds, Clock: deps.Clock, Barrier: deps.Barrier}
	sh := &sessionHandlers{sessions: sessions, models: deps.Models}
	ch := &chatHandlers{sessions: sessions, messages: messages, runs: runs, runner: runner, dataDir: deps.DataDir}
	auth := func(next http.Handler) http.Handler { return requireAuthAt(deps.DB, now, next) }
	mutation := func(next http.Handler) http.Handler { return securedMutation(deps.DB, now, next) }
	InstructionRoutes(mux, deps.DB, deps.DataDir, deps.Clock, deps.Barrier, auth, mutation)
	MemoryRoutes(mux, deps.DB, deps.DataDir, deps.Clock, auth)
	rh := reviewHandlers{db: deps.DB, queue: review.Queue{DB: deps.DB, Clock: deps.Clock}, store: store.ReviewStore{DB: deps.DB, Clock: deps.Clock}}
	ph := promoteHandlers{db: deps.DB, machine: deps.Publish, sessions: sessions}
	bh := backupHandlers{service: deps.Backup}
	// Backup POST must NOT go through Barrier.Mutate middleware — Service.Run takes Snapshot.
	mux.Handle("GET /api/v1/backups", auth(http.HandlerFunc(bh.list)))
	mux.Handle("POST /api/v1/backups", mutation(http.HandlerFunc(bh.create)))
	mux.Handle("POST /api/v1/sessions/{id}/promote", mutation(http.HandlerFunc(ph.create)))
	mux.Handle("GET /api/v1/operations/{id}", auth(http.HandlerFunc(ph.status)))
	mux.Handle("GET /api/v1/review/queue", auth(http.HandlerFunc(rh.queueDue)))
	mux.Handle("POST /api/v1/review/items/{id}/rate", mutation(http.HandlerFunc(rh.rate)))
	mux.Handle("POST /api/v1/review/items/{id}/suspend", mutation(http.HandlerFunc(rh.suspend)))
	mux.Handle("POST /api/v1/review/pending/{id}/retry", mutation(http.HandlerFunc(rh.retry)))
	mux.Handle("GET /api/v1/models", auth(http.HandlerFunc(sh.modelsList)))
	mux.Handle("GET /api/v1/projects/{id}/sessions", auth(http.HandlerFunc(sh.projectSessions)))
	mux.Handle("POST /api/v1/projects/{id}/sessions", mutation(http.HandlerFunc(sh.projectSessions)))
	mux.Handle("GET /api/v1/sessions/{id}", auth(http.HandlerFunc(sh.session)))
	mux.Handle("DELETE /api/v1/sessions/{id}", mutation(http.HandlerFunc(sh.session)))
	mux.Handle("GET /api/v1/sessions/{id}/messages", auth(http.HandlerFunc(ch.messagesRoute)))
	mux.Handle("POST /api/v1/sessions/{id}/messages", mutation(http.HandlerFunc(ch.messagesRoute)))
	co := &compoundHandlers{
		sessions:  sessions,
		compound:  compounds,
		publisher: &compound.Publisher{DataDir: deps.DataDir, DB: deps.DB, Clock: deps.Clock, Barrier: deps.Barrier},
		runner:    runner,
		clock:     deps.Clock,
	}
	mux.Handle("POST /api/v1/sessions/{id}/compound", mutation(http.HandlerFunc(co.create)))
	mux.Handle("GET /api/v1/sessions/{id}/compound/{proposal_id}", auth(http.HandlerFunc(co.get)))
	mux.Handle("POST /api/v1/sessions/{id}/compound/{proposal_id}/decide", mutation(http.HandlerFunc(co.decide)))
	mux.Handle("GET /api/v1/sessions/{id}/runs/current", auth(http.HandlerFunc(ch.currentRun)))
	mux.Handle("GET /api/v1/sessions/{id}/workspace/tree", auth(http.HandlerFunc(ch.workspaceTree)))
	mux.Handle("GET /api/v1/sessions/{id}/workspace/file", auth(http.HandlerFunc(ch.workspaceFile)))
	mux.Handle("GET /health", healthHandler(deps.DataDir))
	ui := deps.UI
	if ui == nil {
		ui = http.FileServer(deps.Static)
	}
	if deps.UI != nil || deps.Static != nil {
		mux.Handle("GET /", ui)
	}
	return mux
}
