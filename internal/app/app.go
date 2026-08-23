package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/agent"
	"github.com/rigasyahrul/personal-agent/internal/agent/skills"
	"github.com/rigasyahrul/personal-agent/internal/backup"
	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/config"
	database "github.com/rigasyahrul/personal-agent/internal/db"
	"github.com/rigasyahrul/personal-agent/internal/httpapi"
	"github.com/rigasyahrul/personal-agent/internal/layout"
	"github.com/rigasyahrul/personal-agent/internal/publish"
	"github.com/rigasyahrul/personal-agent/internal/review"
	"github.com/rigasyahrul/personal-agent/internal/store"
)

// Dependencies permits deterministic injection for acceptance tests and operators.
type Dependencies struct {
	Clock                  clock.Clock
	Provider               agent.Provider
	BiteGenerator          review.BiteGenerator
	ObjectSink             backup.Sink
	AfterPublishTransition func(string) error
	CrashAfter             string
	// DisableBackgroundWorkers skips bite/backup loops (tests drive them explicitly).
	DisableBackgroundWorkers bool
	SecureCookies            *bool
	BootstrapToken           string
	Models                   []config.ModelRef
	Static                   http.FileSystem
}

func DefaultDependencies(cfg config.Config) Dependencies {
	return Dependencies{
		Clock:         clock.RealClock{},
		Provider:      &agent.OpenAICompat{APIKey: cfg.OpenAIAPIKey, BaseURL: cfg.OpenAIBaseURL},
		BiteGenerator: agent.ProviderBiteGenerator{Provider: &agent.OpenAICompat{APIKey: cfg.OpenAIAPIKey, BaseURL: cfg.OpenAIBaseURL}},
		ObjectSink:    nil,
	}
}

type App struct {
	db       *sql.DB
	dataDir  string
	cfg      config.Config
	handler  http.Handler
	cancel   context.CancelFunc
	workers  sync.WaitGroup
	Barrier  *backup.Barrier
	Backup   *backup.Service
	Publish  *publish.Machine
	Bites    *review.BiteWorker
	Locks    *store.SessionLocks
	Provider agent.Provider
	Clock    clock.Clock
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	return NewWithDependencies(ctx, cfg, DefaultDependencies(cfg))
}

func NewWithDependencies(ctx context.Context, cfg config.Config, deps Dependencies) (*App, error) {
	if deps.Clock == nil {
		deps.Clock = clock.RealClock{}
	}
	if deps.BootstrapToken != "" {
		cfg.BootstrapToken = deps.BootstrapToken
	}
	if deps.Models != nil {
		cfg.Models = deps.Models
	}
	secure := cfg.SecureCookies
	if deps.SecureCookies != nil {
		secure = *deps.SecureCookies
	}
	db, err := database.Open(ctx, filepath.Join(cfg.DataDir, "db", "personal-agent.sqlite"))
	if err != nil {
		return nil, err
	}
	if err := layout.EnsureGlobalKnowledgeDirs(cfg.DataDir, skills.DefaultCompoundingSkillMarkdown()); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("seed global knowledge: %w", err)
	}
	barrier := &backup.Barrier{}
	sessionLocks := store.NewSessionLocks()
	machine := &publish.Machine{
		DB:              db,
		DataDir:         cfg.DataDir,
		Clock:           deps.Clock,
		Barrier:         barrier,
		SessionLocks:    sessionLocks,
		AfterTransition: deps.AfterPublishTransition,
		CrashAfter:      deps.CrashAfter,
	}
	if err := machine.RecoverAll(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("recover unfinished publications: %w", err)
	}
	if err := (&store.KnowledgeStore{DB: db, Clock: deps.Clock}).BackfillReadySourceNotes(ctx, cfg.DataDir); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("backfill knowledge notes: %w", err)
	}
	backupSvc := backup.NewService(db, cfg.DataDir, barrier, deps.Clock, deps.ObjectSink)
	// Only load AWS/S3 when a bucket is explicitly configured; ignore ambient creds otherwise.
	if cfg.BackupS3Bucket != "" && deps.ObjectSink == nil {
		sink, sinkErr := backup.NewAWSSinkFromEnv(ctx, cfg.BackupS3Bucket, cfg.BackupS3Region, cfg.BackupS3Endpoint)
		if sinkErr != nil {
			_ = db.Close()
			return nil, fmt.Errorf("configure backup sink: %w", sinkErr)
		}
		backupSvc.Sink = sink
		backupSvc.Bucket = cfg.BackupS3Bucket
	} else if deps.ObjectSink != nil && cfg.BackupS3Bucket != "" {
		backupSvc.Bucket = cfg.BackupS3Bucket
	}
	provider := deps.Provider
	if provider == nil {
		provider = &agent.OpenAICompat{APIKey: cfg.OpenAIAPIKey, BaseURL: cfg.OpenAIBaseURL}
	}
	biteGen := deps.BiteGenerator
	if biteGen == nil {
		biteGen = agent.ProviderBiteGenerator{Provider: provider}
	}
	static := deps.Static
	if static == nil {
		static = http.Dir("web/dist")
	}
	var ui http.Handler
	if rawURL := os.Getenv("PA_UI_DEV_PROXY"); rawURL != "" {
		ui, err = httpapi.NewUIProxy(rawURL)
		if err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	workerCtx, cancelWorkers := context.WithCancel(ctx)
	sinkConfigured := cfg.BackupS3Bucket != "" || deps.ObjectSink != nil
	application := &App{
		db:       db,
		dataDir:  cfg.DataDir,
		cfg:      cfg,
		cancel:   cancelWorkers,
		Barrier:  barrier,
		Backup:   backupSvc,
		Publish:  machine,
		Locks:    sessionLocks,
		Provider: provider,
		Clock:    deps.Clock,
		handler: httpapi.New(httpapi.ServerDeps{
			DB:                   db,
			DataDir:              cfg.DataDir,
			Clock:                deps.Clock,
			BootstrapToken:       cfg.BootstrapToken,
			SecureCookies:        secure,
			Static:               static,
			UI:                   ui,
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
		Clock:     deps.Clock,
		Generator: biteGen,
		Lease:     time.Minute,
		Barrier:   barrier,
	}
	application.Bites = biteWorker
	if !deps.DisableBackgroundWorkers {
		application.workers.Add(1)
		go func() {
			defer application.workers.Done()
			runBiteWorker(workerCtx, biteWorker, time.Second)
		}()
		application.workers.Add(1)
		go func() {
			defer application.workers.Done()
			runDailyBackupScheduler(workerCtx, db, backupSvc, deps.Clock)
		}()
	}
	return application, nil
}

func (a *App) Handler() http.Handler { return a.handler }

func (a *App) DB() *sql.DB { return a.db }

func (a *App) DataDir() string { return a.dataDir }

func (a *App) Recover(ctx context.Context) error {
	if a.Publish == nil {
		return nil
	}
	return a.Publish.RecoverAll(ctx)
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
