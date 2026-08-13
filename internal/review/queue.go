package review

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/clock"
)

type Scope struct{ Raw, ProjectID string }

func ParseScope(raw string) (Scope, error) {
	if raw == "all" {
		return Scope{Raw: raw}, nil
	}
	if strings.HasPrefix(raw, "project:") && strings.TrimPrefix(raw, "project:") != "" {
		return Scope{Raw: raw, ProjectID: strings.TrimPrefix(raw, "project:")}, nil
	}
	return Scope{}, fmt.Errorf("scope must be all or project:{id}")
}

type QueueItem struct {
	ID               string     `json:"id"`
	ProjectID        string     `json:"project_id"`
	NoteID           string     `json:"note_id"`
	Kind             string     `json:"kind"`
	Prompt           string     `json:"prompt"`
	Answer           *string    `json:"answer"`
	SourceSHA256     string     `json:"source_sha256"`
	SourceRevision   int64      `json:"source_revision"`
	Stage            int        `json:"stage"`
	IntervalDays     float64    `json:"interval_days"`
	EaseFactor       float64    `json:"ease_factor"`
	Reps             int        `json:"reps"`
	Lapses           int        `json:"lapses"`
	LastReviewedAt   *time.Time `json:"last_reviewed_at"`
	RowVersion       int64      `json:"row_version"`
	DueAt            time.Time  `json:"due_at"`
	SchedulerVersion string     `json:"scheduler_version"`
}
type QueueDTO struct {
	Scope    string      `json:"scope"`
	CaughtUp bool        `json:"caught_up"`
	Items    []QueueItem `json:"items"`
}
type Queue struct {
	DB    *sql.DB
	Clock clock.Clock
}

func (q Queue) Due(ctx context.Context, scope Scope) (QueueDTO, error) {
	query := `SELECT id,project_id,note_id,kind,prompt,answer,source_sha256,source_revision,stage,interval_days,ease_factor,reps,lapses,last_reviewed_at,row_version,due_at,scheduler_version FROM review_items WHERE status='active'`
	var args []any
	if scope.ProjectID != "" {
		query += ` AND project_id=?`
		args = append(args, scope.ProjectID)
	}
	rows, err := q.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return QueueDTO{}, err
	}
	defer rows.Close()
	now := q.Clock.Now().UTC()
	due := make([]QueueItem, 0)
	for rows.Next() {
		var item QueueItem
		var answer, last sql.NullString
		var encodedDue string
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.NoteID, &item.Kind, &item.Prompt, &answer, &item.SourceSHA256, &item.SourceRevision, &item.Stage, &item.IntervalDays, &item.EaseFactor, &item.Reps, &item.Lapses, &last, &item.RowVersion, &encodedDue, &item.SchedulerVersion); err != nil {
			return QueueDTO{}, err
		}
		item.DueAt, err = time.Parse(time.RFC3339Nano, encodedDue)
		if err != nil {
			return QueueDTO{}, fmt.Errorf("review item %s due_at: %w", item.ID, err)
		}
		item.DueAt = item.DueAt.UTC()
		if answer.Valid {
			item.Answer = &answer.String
		}
		if last.Valid {
			parsed, e := time.Parse(time.RFC3339Nano, last.String)
			if e != nil {
				return QueueDTO{}, fmt.Errorf("review item %s last_reviewed_at: %w", item.ID, e)
			}
			parsed = parsed.UTC()
			item.LastReviewedAt = &parsed
		}
		if !item.DueAt.After(now) {
			due = append(due, item)
		}
	}
	if err := rows.Err(); err != nil {
		return QueueDTO{}, err
	}
	sort.Slice(due, func(i, j int) bool {
		if due[i].DueAt.Equal(due[j].DueAt) {
			return due[i].ID < due[j].ID
		}
		return due[i].DueAt.Before(due[j].DueAt)
	})
	caught := len(due) == 0
	if len(due) > 50 {
		due = due[:50]
	}
	return QueueDTO{Scope: scope.Raw, CaughtUp: caught, Items: due}, nil
}
