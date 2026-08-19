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
	ErrOwnerExists    = ErrBootstrapped // plan alias
	ErrBootstrapToken = errors.New("invalid bootstrap token")
)

func Bootstrap(ctx context.Context, db *sql.DB, configured, provided, password string, now time.Time) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var count int
	if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM owner").Scan(&count); err != nil {
		return err
	}
	// Owner-exists check before token validation so a second attempt never
	// reveals whether the supplied bootstrap token was correct.
	if count != 0 {
		return ErrOwnerExists
	}
	if configured == "" || len(configured) != len(provided) || subtle.ConstantTimeCompare([]byte(configured), []byte(provided)) != 1 {
		return ErrBootstrapToken
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	stamp := now.UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, "INSERT INTO owner(id,password_hash,created_at,updated_at) VALUES(1,?,?,?)", hash, stamp, stamp); err != nil {
		// Singleton owner constraint: concurrent winner is already bootstrapped.
		var again int
		if scanErr := tx.QueryRowContext(ctx, "SELECT count(*) FROM owner").Scan(&again); scanErr == nil && again != 0 {
			return ErrOwnerExists
		}
		return err
	}
	return tx.Commit()
}
