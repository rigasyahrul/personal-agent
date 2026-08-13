package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/domain"
	"github.com/rigasyahrul/personal-agent/internal/ids"
)

type MessageStore struct {
	DB  *sql.DB
	Now func() time.Time
}

func (s *MessageStore) Append(ctx context.Context, message domain.Message) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if message.Sequence == 0 {
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(sequence),0)+1 FROM messages WHERE session_id=?`, message.SessionID).Scan(&message.Sequence); err != nil {
			return err
		}
	}
	if message.ID == "" {
		message.ID = ids.NewID()
	}
	if message.CreatedAt.IsZero() {
		message.CreatedAt = s.Now().UTC()
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO messages
		(id,session_id,run_id,sequence,role,content,tool_calls_json,tool_call_id,status,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?)`, message.ID, message.SessionID, message.RunID, message.Sequence,
		message.Role, message.Content, message.ToolCallsJSON, message.ToolCallID, message.Status, formatTime(message.CreatedAt))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *MessageStore) List(ctx context.Context, sessionID string) ([]domain.Message, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id,session_id,run_id,sequence,role,content,tool_calls_json,tool_call_id,status,created_at
		FROM messages WHERE session_id=? ORDER BY sequence`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Message{}
	for rows.Next() {
		var message domain.Message
		var runID, calls, callID sql.NullString
		var createdAt string
		if err := rows.Scan(&message.ID, &message.SessionID, &runID, &message.Sequence, &message.Role, &message.Content,
			&calls, &callID, &message.Status, &createdAt); err != nil {
			return nil, err
		}
		if runID.Valid {
			message.RunID = &runID.String
		}
		if calls.Valid {
			message.ToolCallsJSON = &calls.String
		}
		if callID.Valid {
			message.ToolCallID = &callID.String
		}
		if message.CreatedAt, err = parseTime(createdAt); err != nil {
			return nil, err
		}
		out = append(out, message)
	}
	return out, rows.Err()
}
