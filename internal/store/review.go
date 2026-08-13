package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/ids"
	"github.com/rigasyahrul/personal-agent/internal/review"
)

type RatedItem struct {
	ID               string     `json:"id"`
	Stage            int        `json:"stage"`
	IntervalDays     float64    `json:"interval_days"`
	EaseFactor       float64    `json:"ease_factor"`
	Reps             int        `json:"reps"`
	Lapses           int        `json:"lapses"`
	DueAt            time.Time  `json:"due_at"`
	LastReviewedAt   *time.Time `json:"last_reviewed_at"`
	RowVersion       int64      `json:"row_version"`
	SchedulerVersion string     `json:"scheduler_version"`
}

type RowVersionConflict struct{ Current int64 }

func (e *RowVersionConflict) Error() string {
	return fmt.Sprintf("review item row version conflict: current version is %d", e.Current)
}

type ReviewStore struct {
	DB    *sql.DB
	Clock clock.Clock
}

type ReviewPending struct {
	ID, NoteID, SourceSHA256, GeneratorVersion, Status string
}

func ReviewPendingForPublication(ctx context.Context, db *sql.DB, noteID, sourceSHA256 string) (ReviewPending, error) {
	var out ReviewPending
	err := db.QueryRowContext(ctx, `SELECT id,note_id,source_sha256,generator_version,status FROM review_pending WHERE note_id=? AND source_sha256=? AND generator_version='bites-v1'`, noteID, sourceSHA256).
		Scan(&out.ID, &out.NoteID, &out.SourceSHA256, &out.GeneratorVersion, &out.Status)
	return out, err
}

func RetryReviewPending(ctx context.Context, db *sql.DB, id string) error {
	res, err := db.ExecContext(ctx, `UPDATE review_pending SET status='pending',lease_until=NULL,last_error=NULL WHERE id=? AND status='failed'`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		var exists int
		if err := db.QueryRowContext(ctx, `SELECT 1 FROM review_pending WHERE id=?`, id).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		return ErrConflict
	}
	return nil
}

func (s ReviewStore) Suspend(ctx context.Context, id string) error {
	res, err := s.DB.ExecContext(ctx, `UPDATE review_items SET status='suspended' WHERE id=? AND status='active'`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 1 {
		return nil
	}
	var status string
	if err := s.DB.QueryRowContext(ctx, `SELECT status FROM review_items WHERE id=?`, id).Scan(&status); errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if status == "suspended" {
		return nil
	}
	return ErrConflict
}

func (s ReviewStore) Rate(ctx context.Context, itemID, requestKey string, expectedVersion int64, rating domain.Rating, durationMS int64) (RatedItem, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return RatedItem{}, err
	}
	defer tx.Rollback()

	prior, found, err := persistedEventResult(ctx, tx, requestKey)
	if err != nil {
		return RatedItem{}, err
	}
	if found {
		return prior, nil
	}
	if strings.TrimSpace(itemID) == "" || strings.TrimSpace(requestKey) == "" || durationMS < 0 || !validRating(rating) {
		return RatedItem{}, ErrValidation
	}

	current, err := activeRatedItem(ctx, tx, itemID)
	if err != nil {
		return RatedItem{}, err
	}
	if current.RowVersion != expectedVersion {
		return RatedItem{}, &RowVersionConflict{Current: current.RowVersion}
	}

	now := s.Clock.Now().UTC()
	nextState := review.ApplyRating(review.ReviewItemState{
		Stage: current.Stage, IntervalDays: current.IntervalDays, EaseFactor: current.EaseFactor,
		Reps: current.Reps, Lapses: current.Lapses, DueAt: current.DueAt,
	}, rating, now)
	resulting := current
	resulting.Stage = nextState.Stage
	resulting.IntervalDays = nextState.IntervalDays
	resulting.EaseFactor = nextState.EaseFactor
	resulting.Reps = nextState.Reps
	resulting.Lapses = nextState.Lapses
	resulting.DueAt = nextState.DueAt.UTC()
	resulting.LastReviewedAt = &now
	resulting.RowVersion++
	previousJSON, err := json.Marshal(current)
	if err != nil {
		return RatedItem{}, err
	}
	resultingJSON, err := json.Marshal(resulting)
	if err != nil {
		return RatedItem{}, err
	}

	res, err := tx.ExecContext(ctx, `UPDATE review_items SET stage=?,interval_days=?,ease_factor=?,reps=?,lapses=?,due_at=?,last_reviewed_at=?,row_version=? WHERE id=? AND status='active' AND row_version=?`,
		resulting.Stage, resulting.IntervalDays, resulting.EaseFactor, resulting.Reps, resulting.Lapses,
		formatTime(resulting.DueAt), formatTime(now), resulting.RowVersion, itemID, expectedVersion)
	if err != nil {
		return RatedItem{}, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return RatedItem{}, err
	}
	if rows != 1 {
		latest, lookupErr := activeRatedItem(ctx, tx, itemID)
		if lookupErr != nil {
			return RatedItem{}, lookupErr
		}
		return RatedItem{}, &RowVersionConflict{Current: latest.RowVersion}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO review_events(id,review_item_id,request_key,rating,previous_state_json,resulting_state_json,scheduler_version,reviewed_at,duration_ms) VALUES(?,?,?,?,?,?,?,?,?)`,
		ids.NewID(), itemID, requestKey, string(rating), string(previousJSON), string(resultingJSON), current.SchedulerVersion, formatTime(now), durationMS)
	if err != nil {
		return RatedItem{}, err
	}
	if err := tx.Commit(); err != nil {
		return RatedItem{}, err
	}
	return resulting, nil
}

func validRating(rating domain.Rating) bool {
	return rating == domain.RatingAgain || rating == domain.RatingHard || rating == domain.RatingGood || rating == domain.RatingEasy
}

func persistedEventResult(ctx context.Context, tx *sql.Tx, key string) (RatedItem, bool, error) {
	var encoded string
	err := tx.QueryRowContext(ctx, `SELECT resulting_state_json FROM review_events WHERE request_key=?`, key).Scan(&encoded)
	if errors.Is(err, sql.ErrNoRows) {
		return RatedItem{}, false, nil
	}
	if err != nil {
		return RatedItem{}, false, err
	}
	var item RatedItem
	if err := json.Unmarshal([]byte(encoded), &item); err != nil {
		return RatedItem{}, false, err
	}
	return item, true, nil
}

func activeRatedItem(ctx context.Context, tx *sql.Tx, id string) (RatedItem, error) {
	var item RatedItem
	var due string
	var last sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT id,stage,interval_days,ease_factor,reps,lapses,due_at,last_reviewed_at,row_version,scheduler_version FROM review_items WHERE id=? AND status='active'`, id).
		Scan(&item.ID, &item.Stage, &item.IntervalDays, &item.EaseFactor, &item.Reps, &item.Lapses, &due, &last, &item.RowVersion, &item.SchedulerVersion)
	if err != nil {
		return RatedItem{}, err
	}
	item.DueAt, err = parseTime(due)
	if err != nil {
		return RatedItem{}, err
	}
	if last.Valid {
		value, parseErr := parseTime(last.String)
		if parseErr != nil {
			return RatedItem{}, parseErr
		}
		item.LastReviewedAt = &value
	}
	return item, nil
}
