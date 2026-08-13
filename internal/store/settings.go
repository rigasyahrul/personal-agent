package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

var ErrInvalidSettings = errors.New("invalid settings")

type Settings struct {
	Timezone        string
	DefaultProvider string
	DefaultModelID  string
	BackupSchedule  string
}

type SettingsStore struct{ DB *sql.DB }

func (s SettingsStore) Get(ctx context.Context) (Settings, error) {
	var out Settings
	var provider, model sql.NullString
	err := s.DB.QueryRowContext(ctx, "SELECT timezone,default_provider,default_model_id,backup_schedule FROM settings WHERE id=1").Scan(&out.Timezone, &provider, &model, &out.BackupSchedule)
	out.DefaultProvider, out.DefaultModelID = provider.String, model.String
	return out, err
}

func (s SettingsStore) Put(ctx context.Context, value Settings, now time.Time) error {
	if _, err := time.LoadLocation(value.Timezone); err != nil || value.BackupSchedule != "off" && value.BackupSchedule != "daily" {
		return ErrInvalidSettings
	}
	_, err := s.DB.ExecContext(ctx, "UPDATE settings SET timezone=?,default_provider=?,default_model_id=?,backup_schedule=?,updated_at=? WHERE id=1", value.Timezone, nullable(value.DefaultProvider), nullable(value.DefaultModelID), value.BackupSchedule, now.UTC().Format(time.RFC3339Nano))
	return err
}

func nullable(value string) any {
	if value == "" {
		return nil
	}
	return value
}
