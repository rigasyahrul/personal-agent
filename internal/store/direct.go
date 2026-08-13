package store

import (
	"context"
	"database/sql"
	"errors"
)

type DirectOperation struct {
	ID, RequestKey, RequestFingerprint, ProjectID, RelativePath, ReviewMode, NoteID, Status string
	FrozenSHA                                                                               string
	FrozenSize                                                                              int64
}
type DirectStore struct{ DB *sql.DB }

func (s DirectStore) ByKey(ctx context.Context, key string) (DirectOperation, error) {
	return s.scan(s.DB.QueryRowContext(ctx, `SELECT id,request_key,request_fingerprint,target_project_id,target_relative_path,review_mode,note_id,status,coalesce(frozen_sha256,''),coalesce(frozen_size,0) FROM direct_ops WHERE request_key=?`, key))
}
func (s DirectStore) ByID(ctx context.Context, id string) (DirectOperation, error) {
	return s.scan(s.DB.QueryRowContext(ctx, `SELECT id,request_key,request_fingerprint,target_project_id,target_relative_path,review_mode,note_id,status,coalesce(frozen_sha256,''),coalesce(frozen_size,0) FROM direct_ops WHERE id=?`, id))
}
func (s DirectStore) scan(row *sql.Row) (o DirectOperation, err error) {
	err = row.Scan(&o.ID, &o.RequestKey, &o.RequestFingerprint, &o.ProjectID, &o.RelativePath, &o.ReviewMode, &o.NoteID, &o.Status, &o.FrozenSHA, &o.FrozenSize)
	return
}
func (s DirectStore) Active(ctx context.Context) ([]DirectOperation, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id FROM direct_ops WHERE status NOT IN ('completed','failed') ORDER BY created_at,id`)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	var out []DirectOperation
	for _, id := range ids {
		o, e := s.ByID(ctx, id)
		if e != nil {
			return nil, e
		}
		out = append(out, o)
	}
	return out, nil
}
func IsNoRows(err error) bool { return errors.Is(err, sql.ErrNoRows) }
