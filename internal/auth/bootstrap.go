package auth

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"time"
)

var (
	ErrBootstrapped   = errors.New("owner already bootstrapped")
	ErrBootstrapToken = errors.New("invalid bootstrap token")
)

func Bootstrap(ctx context.Context, db *sql.DB, configured, provided, password string, now time.Time) error {
	var count int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM owner").Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return ErrBootstrapped
	}
	if configured == "" || len(configured) != len(provided) || subtle.ConstantTimeCompare([]byte(configured), []byte(provided)) != 1 {
		return ErrBootstrapToken
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	stamp := now.UTC().Format(time.RFC3339Nano)
	if _, err := db.ExecContext(ctx, "INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,?,?,?)", hash, stamp, stamp); err != nil {
		// The singleton owner constraint makes a concurrent winner equivalent to
		// an already-completed bootstrap.
		if scanErr := db.QueryRowContext(ctx, "SELECT count(*) FROM owner").Scan(&count); scanErr == nil && count != 0 {
			return ErrBootstrapped
		}
		return err
	}
	return nil
}
