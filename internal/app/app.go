package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/agent"
	"github.com/rigasyahrul/personal-agent/internal/backup"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/config"
	database "github.com/rigasyahrul/personal-agent/internal/db"
	"github.com/rigasyahrul/personal-agent/internal/httpapi"
	"github.com/rigasyahrul/personal-agent/internal/publish"
	"github.com/rigasyahrul/personal-agent/internal/review"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

type App struct {
	db      *sql.DB
	handler http.Handler
	cancel  context.CancelFunc
	workers sync.WaitGroup
	Barrier *backup.Barrier
	Backup  *backup.Service
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	db, err := database.Open(ctx, filepath.Join(cfg.DataDir, "db", "personal-agent.sqlite"))
	if err != nil {
		return nil, err
	}
	realClock := clock.RealClock{}
	barrier := &backup.Barrier{}
	machine := &publish.Machine{DB: db, DataDir: cfg.DataDir, Clock: realClock, Barrier: barrier}
	if err := machine.RecoverAll(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("recover unfinished publications: %w", err)
	}
	backupSvc := backup.NewService(db, cfg.DataDir, barrier, realClock, nil)
	// Only load AWS/S3 when a bucket is explicitly configured; ignore ambient creds otherwise.
	if cfg.BackupS3Bucket != "" {
		sink, sinkErr := backup.NewAWSSinkFromEnv(ctx, cfg.BackupS3Bucket, cfg.BackupS3Region, cfg.BackupS3Endpoint)
		if sinkErr != nil {
			_ = db.Close()
			return nil, fmt.Errorf("configure backup sink: %w", sinkErr)
		}
		backupSvc.Sink = sink
		backupSvc.Bucket = cfg.BackupS3Bucket
	}
	provider := &agent.OpenAICompat{APIKey: cfg.OpenAIAPIKey, BaseURL: cfg.OpenAIBaseURL}
	workerCtx, cancelWorkers := context.WithCancel(ctx)
	sinkConfigured := cfg.BackupS3Bucket != ""
	application := &App{
		db:      db,
		cancel:  cancelWorkers,
		Barrier: barrier,
		Backup:  backupSvc,
		handler: httpapi.New(httpapi.ServerDeps{
			DB:                   db,
			DataDir:              cfg.DataDir,
			Clock:                realClock,
			BootstrapToken:       cfg.BootstrapToken,
			SecureCookies:        cfg.SecureCookies,
			Static:               http.Dir("web"),
			Publish:              machine,
			Models:               cfg.Models,
			Provider:             provider,
			Backup:               backupSvc,
			Barrier:              barrier,
			BackupSinkConfigured: sinkConfigured,
		}),
	}
	biteWorker := &review.BiteWorker{
		DB:        db,
		DataDir:   cfg.DataDir,
		Clock:     realClock,
		Generator: agent.ProviderBiteGenerator{Provider: provider},
		Lease:     time.Minute,
		Barrier:   barrier,
	}
	application.workers.Add(1)
	go func() {
		defer application.workers.Done()
		runBiteWorker(workerCtx, biteWorker, time.Second)
	}()
	application.workers.Add(1)
	go func() {
		defer application.workers.Done()
		runDailyBackupScheduler(workerCtx, db, backupSvc, realClock)
	}()

	return application, nil
}

func (a *App) Handler() http.Handler {
	return a.handler
}

func (a *App) Close() error {
	a.cancel()
	a.workers.Wait()
	return a.db.Close()
}

func runBiteWorker(ctx context.Context, worker *review.BiteWorker, pollInterval time.Duration) {
	for {
		didWork, _ := worker.LeaseAndRun(ctx)
		if ctx.Err() != nil {
			return
		}
		if didWork {
			continue
		}
		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

// runDailyBackupScheduler fires backup.Service.Run at next local 03:00 when schedule is daily.
// Missed ticks catch up at most once on startup (immediate run if past 03:00 and none today).
func runDailyBackupScheduler(ctx context.Context, db *sql.DB, svc *backup.Service, clk clock.Clock) {
	settings := store.SettingsStore{DB: db}
	var lastRunDay string
	for {
		if ctx.Err() != nil {
			return
		}
		value, err := settings.Get(ctx)
		if err != nil || value.BackupSchedule != "daily" {
			timer := time.NewTimer(time.Minute)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			continue
		}
		loc := time.UTC
		if value.Timezone != "" {
			if l, lerr := time.LoadLocation(value.Timezone); lerr == nil {
				loc = l
			}
		}
		now := clk.Now().In(loc)
		dayKey := now.Format("2006-01-02")
		next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, loc)
		if !now.Before(next) {
			// Past today's 03:00: run once if not already run today, then wait until tomorrow 03:00.
			if lastRunDay != dayKey {
				_, _ = svc.Run(ctx)
				lastRunDay = dayKey
			}
			next = next.Add(24 * time.Hour)
		}
		wait := next.Sub(now)
		if wait < time.Second {
			wait = time.Second
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if ctx.Err() != nil {
				return
			}
			// Re-check schedule still daily.
			value, err = settings.Get(ctx)
			if err == nil && value.BackupSchedule == "daily" {
				_, _ = svc.Run(ctx)
				lastRunDay = clk.Now().In(loc).Format("2006-01-02")
			}
		}
	}
}
