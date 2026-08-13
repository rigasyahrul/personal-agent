package store

import (
	"context"
	"database/sql"
)

type PromoteStore struct{ DB *sql.DB }

func (s PromoteStore) ByKey(ctx context.Context, key string) (DirectOperation, error) {
	return s.scan(s.DB.QueryRowContext(ctx, `SELECT id,request_key,request_fingerprint,target_project_id,target_relative_path,review_mode,note_id,status,coalesce(frozen_sha256,''),coalesce(frozen_size,0),coalesce(error,''),session_id,workspace_path FROM promote_ops WHERE request_key=?`, key))
}

func (s PromoteStore) ByID(ctx context.Context, id string) (DirectOperation, error) {
	return s.scan(s.DB.QueryRowContext(ctx, `SELECT id,request_key,request_fingerprint,target_project_id,target_relative_path,review_mode,note_id,status,coalesce(frozen_sha256,''),coalesce(frozen_size,0),coalesce(error,''),session_id,workspace_path FROM promote_ops WHERE id=?`, id))
}

func (s PromoteStore) Active(ctx context.Context) ([]DirectOperation, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id FROM promote_ops WHERE status NOT IN ('completed','failed') ORDER BY created_at,id`)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	var out []DirectOperation
	for _, id := range ids {
		o, err := s.ByID(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}

func (s PromoteStore) scan(row *sql.Row) (o DirectOperation, err error) {
	o.Kind = "promote"
	err = row.Scan(&o.ID, &o.RequestKey, &o.RequestFingerprint, &o.ProjectID, &o.RelativePath, &o.ReviewMode, &o.NoteID, &o.Status, &o.FrozenSHA, &o.FrozenSize, &o.Error, &o.SessionID, &o.WorkspacePath)
	return
}
