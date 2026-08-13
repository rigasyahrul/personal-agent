package store

import (
	"context"
	"testing"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/testutil"
)

func TestSettingsGetPutAndValidation(t *testing.T) {
	db, _ := testutil.TempDB(t)
	s := SettingsStore{DB: db}
	got, err := s.Get(context.Background())
	if err != nil || got.Timezone != "UTC" || got.BackupSchedule != "off" {
		t.Fatalf("defaults: %+v, %v", got, err)
	}
	now := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)
	want := Settings{Timezone: "Asia/Jakarta", DefaultProvider: "openai", DefaultModelID: "gpt-5", BackupSchedule: "daily"}
	if err := s.Put(context.Background(), want, now); err != nil {
		t.Fatal(err)
	}
	got, err = s.Get(context.Background())
	if err != nil || got != want {
		t.Fatalf("round trip: %+v, %v", got, err)
	}
	var updated string
	if err := db.QueryRow("SELECT updated_at FROM settings WHERE id=1").Scan(&updated); err != nil || updated != now.Format(time.RFC3339Nano) {
		t.Fatalf("updated_at = %q, err=%v", updated, err)
	}
	for _, invalid := range []Settings{
		{Timezone: "Not/AZone", BackupSchedule: "off"},
		{Timezone: "UTC", BackupSchedule: "weekly"},
	} {
		if err := s.Put(context.Background(), invalid, now); err != ErrInvalidSettings {
			t.Fatalf("Put(%+v) error = %v", invalid, err)
		}
	}
}
