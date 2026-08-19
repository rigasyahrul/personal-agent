package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/ids"
)

// ErrRunBusy is returned when a session already has a non-terminal agent run
// under a different request key. It is the same sentinel as ErrSessionBusy.
var ErrRunBusy = ErrSessionBusy

type RunStore struct {
	DB  *sql.DB
	Now func() time.Time
}

// RunAdmission is the result of atomic run + user-message admission.
type RunAdmission struct {
	RunID    string
	Existing bool
}

const runSelect = `SELECT id,session_id,request_key,status,started_at,completed_at,error,created_at FROM agent_runs`

// Admit inserts the user message and a queued run in one IMMEDIATE transaction,
// or returns the existing run for the same request key without appending again.
func (s *RunStore) Admit(ctx context.Context, sessionID, requestKey, userMessage string, now time.Time) (RunAdmission, error) {
	if requestKey == "" || userMessage == "" {
		return RunAdmission{}, ErrValidation
	}
	now = now.UTC()
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return RunAdmission{}, err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return RunAdmission{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), `ROLLBACK`)
		}
	}()

	// Same-key idempotency first.
	var existingID string
	err = conn.QueryRowContext(ctx, `SELECT id FROM agent_runs WHERE session_id=? AND request_key=?`, sessionID, requestKey).Scan(&existingID)
	if err == nil {
		if _, err := conn.ExecContext(ctx, `COMMIT`); err != nil {
			return RunAdmission{}, err
		}
		committed = true
		return RunAdmission{RunID: existingID, Existing: true}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return RunAdmission{}, err
	}

	var status string
	err = conn.QueryRowContext(ctx, `SELECT status FROM sessions WHERE id=?`, sessionID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return RunAdmission{}, ErrNotFound
	}
	if err != nil {
		return RunAdmission{}, err
	}
	if status != "active" {
		return RunAdmission{}, ErrSessionTerminal
	}

	var activeID string
	err = conn.QueryRowContext(ctx, `SELECT id FROM agent_runs WHERE session_id=? AND status IN ('queued','running')`, sessionID).Scan(&activeID)
	if err == nil {
		return RunAdmission{}, ErrRunBusy
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return RunAdmission{}, err
	}

	runID := ids.NewID()
	msgID := ids.NewID()
	var seq int
	if err := conn.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence),0)+1 FROM messages WHERE session_id=?`, sessionID).Scan(&seq); err != nil {
		return RunAdmission{}, err
	}
	// Insert run first (messages.FK → agent_runs), then user message.
	if _, err := conn.ExecContext(ctx, `INSERT INTO agent_runs(id,session_id,request_key,status,created_at)
		VALUES(?,?,?,'queued',?)`, runID, sessionID, requestKey, formatTime(now)); err != nil {
		// Unique race: prefer same-key idempotency, else busy.
		var racedID string
		if lookupErr := conn.QueryRowContext(ctx, `SELECT id FROM agent_runs WHERE session_id=? AND request_key=?`, sessionID, requestKey).Scan(&racedID); lookupErr == nil {
			if _, cErr := conn.ExecContext(ctx, `COMMIT`); cErr != nil {
				return RunAdmission{}, cErr
			}
			committed = true
			return RunAdmission{RunID: racedID, Existing: true}, nil
		}
		return RunAdmission{}, ErrRunBusy
	}
	if _, err := conn.ExecContext(ctx, `INSERT INTO messages
		(id,session_id,run_id,sequence,role,content,tool_calls_json,tool_call_id,status,created_at)
		VALUES(?,?,?,?,?,?,NULL,NULL,?,?)`,
		msgID, sessionID, runID, seq, domain.MessageRoleUser, userMessage, domain.MessageStatusComplete, formatTime(now)); err != nil {
		return RunAdmission{}, err
	}
	if _, err := conn.ExecContext(ctx, `COMMIT`); err != nil {
		return RunAdmission{}, err
	}
	committed = true
	return RunAdmission{RunID: runID, Existing: false}, nil
}

// BeginOrGet admits a run without a user message (legacy/low-level). Prefer Admit.
func (s *RunStore) BeginOrGet(ctx context.Context, sessionID, requestKey string) (string, bool, error) {
	if requestKey == "" {
		return "", false, ErrValidation
	}
	if run, err := s.byKey(ctx, sessionID, requestKey); err == nil {
		return run.ID, true, nil
	} else if !errors.Is(err, ErrNotFound) {
		return "", false, err
	}

	id, now := ids.NewID(), s.Now().UTC()
	result, err := s.DB.ExecContext(ctx, `INSERT INTO agent_runs(id,session_id,request_key,status,created_at)
		SELECT ?,?,?, 'queued',? WHERE EXISTS(SELECT 1 FROM sessions WHERE id=? AND status='active')`,
		id, sessionID, requestKey, formatTime(now), sessionID)
	if err == nil {
		count, rowsErr := result.RowsAffected()
		if rowsErr != nil {
			return "", false, rowsErr
		}
		if count == 1 {
			return id, false, nil
		}
		return "", false, s.sessionAdmissionError(ctx, sessionID)
	}
	// A competing insert may have won either unique constraint. Query durable
	// state rather than depending on driver-specific SQLite error text.
	if run, lookupErr := s.byKey(ctx, sessionID, requestKey); lookupErr == nil {
		return run.ID, true, nil
	}
	if admissionErr := s.sessionAdmissionError(ctx, sessionID); admissionErr != nil {
		return "", false, admissionErr
	}
	if _, lookupErr := s.Current(ctx, sessionID); lookupErr == nil {
		return "", false, ErrSessionBusy
	}
	return "", false, err
}

func (s *RunStore) sessionAdmissionError(ctx context.Context, sessionID string) error {
	var status string
	if err := s.DB.QueryRowContext(ctx, `SELECT status FROM sessions WHERE id=?`, sessionID).Scan(&status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if status != "active" {
		return ErrSessionTerminal
	}
	return nil
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
