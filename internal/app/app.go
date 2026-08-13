package app

import (
	"context"
	"database/sql"
	"net/http"
	"path/filepath"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/config"
	database "github.com/rigasyahrul/personal-agent/internal/db"
	"github.com/rigasyahrul/personal-agent/internal/httpapi"
	"github.com/rigasyahrul/personal-agent/internal/publish"
)

type App struct {
	db      *sql.DB
	handler http.Handler
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	db, err := database.Open(ctx, filepath.Join(cfg.DataDir, "db", "personal-agent.sqlite"))
	if err != nil {
		return nil, err
	}
	machine := &publish.Machine{DB: db, DataDir: cfg.DataDir, Clock: clock.RealClock{}}
	if err := machine.RecoverAll(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &App{
		db: db,
		handler: httpapi.New(httpapi.ServerDeps{
			DB:             db,
			DataDir:        cfg.DataDir,
			Clock:          clock.RealClock{},
			BootstrapToken: cfg.BootstrapToken,
			SecureCookies:  cfg.SecureCookies,
			Static:         http.Dir("web"),
			Publish:        machine,
		}),
	}, nil
}

func (a *App) Handler() http.Handler {
	return a.handler
}

func (a *App) Close() error {
	return a.db.Close()
}
