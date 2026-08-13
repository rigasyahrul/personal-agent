package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/ids"
)

type RunStore struct {
	DB  *sql.DB
	Now func() time.Time
}

const runSelect = `SELECT id,session_id,request_key,status,started_at,completed_at,error,created_at FROM agent_runs`

func (s *RunStore) BeginOrGet(ctx context.Context, sessionID, requestKey string) (string, bool, error) {
	if requestKey == "" {
		return "", false, ErrValidation
	}
	var sessionStatus string
	if err := s.DB.QueryRowContext(ctx, `SELECT status FROM sessions WHERE id=?`, sessionID).Scan(&sessionStatus); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, ErrNotFound
		}
		return "", false, err
	}
	if sessionStatus != "active" {
		return "", false, ErrValidation
	}
	if run, err := s.byKey(ctx, sessionID, requestKey); err == nil {
		return run.ID, true, nil
	} else if !errors.Is(err, ErrNotFound) {
		return "", false, err
	}
	if _, err := s.Current(ctx, sessionID); err == nil {
		return "", false, ErrSessionBusy
	} else if !errors.Is(err, ErrNotFound) {
		return "", false, err
	}

	id, now := ids.NewID(), s.Now().UTC()
	_, err := s.DB.ExecContext(ctx, `INSERT INTO agent_runs(id,session_id,request_key,status,created_at) VALUES(?,?,?,'queued',?)`, id, sessionID, requestKey, formatTime(now))
	if err == nil {
		return id, false, nil
	}
	// A competing insert may have won either unique constraint. Query state
	// rather than depending on driver-specific SQLite error text.
	if run, lookupErr := s.byKey(ctx, sessionID, requestKey); lookupErr == nil {
		return run.ID, true, nil
	}
	if _, lookupErr := s.Current(ctx, sessionID); lookupErr == nil {
		return "", false, ErrSessionBusy
	}
	return "", false, err
}

func (s *RunStore) Current(ctx context.Context, sessionID string) (domain.AgentRun, error) {
	return scanRun(s.DB.QueryRowContext(ctx, runSelect+` WHERE session_id=? AND status IN ('queued','running')`, sessionID))
}

func (s *RunStore) ByID(ctx context.Context, runID string) (domain.AgentRun, error) {
	return scanRun(s.DB.QueryRowContext(ctx, runSelect+` WHERE id=?`, runID))
}

func (s *RunStore) byKey(ctx context.Context, sessionID, requestKey string) (domain.AgentRun, error) {
	return scanRun(s.DB.QueryRowContext(ctx, runSelect+` WHERE session_id=? AND request_key=?`, sessionID, requestKey))
}

func (s *RunStore) MarkRunning(ctx context.Context, runID string) error {
	result, err := s.DB.ExecContext(ctx, `UPDATE agent_runs SET status='running',started_at=? WHERE id=? AND status='queued'`, formatTime(s.Now().UTC()), runID)
	return changedOne(result, err)
}

func (s *RunStore) MarkDone(ctx context.Context, runID, status, errMsg string) error {
	if status != domain.AgentRunStatusCompleted && status != domain.AgentRunStatusFailed && status != domain.AgentRunStatusCancelled {
		return ErrValidation
	}
	var message any
	if errMsg != "" {
		message = errMsg
	}
	result, err := s.DB.ExecContext(ctx, `UPDATE agent_runs SET status=?,completed_at=?,error=? WHERE id=? AND status IN ('queued','running')`,
		status, formatTime(s.Now().UTC()), message, runID)
	return changedOne(result, err)
}

func changedOne(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrNotFound
	}
	return nil
}

func scanRun(row scanner) (domain.AgentRun, error) {
	var out domain.AgentRun
	var startedAt, completedAt, errorMessage sql.NullString
	var createdAt string
	if err := row.Scan(&out.ID, &out.SessionID, &out.RequestKey, &out.Status, &startedAt, &completedAt, &errorMessage, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return out, ErrNotFound
		}
		return out, err
	}
	var err error
	if out.CreatedAt, err = parseTime(createdAt); err != nil {
		return out, err
	}
	if startedAt.Valid {
		value, err := parseTime(startedAt.String)
		if err != nil {
			return out, err
		}
		out.StartedAt = &value
	}
	if completedAt.Valid {
		value, err := parseTime(completedAt.String)
		if err != nil {
			return out, err
		}
		out.CompletedAt = &value
	}
	if errorMessage.Valid {
		out.Error = &errorMessage.String
	}
	return out, nil
}
