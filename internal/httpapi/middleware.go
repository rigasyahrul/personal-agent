package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/auth"
)

type ownerKey struct{}

func RequireAuth(db *sql.DB, next http.Handler) http.Handler {
	return requireAuthAt(db, func() time.Time { return time.Now().UTC() }, next)
}

func requireAuthAt(db *sql.DB, now func() time.Time, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("pa_session")
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		var encodedExpiry string
		err = db.QueryRowContext(r.Context(), "SELECT expires_at FROM auth_sessions WHERE token_hash=?", auth.TokenHash(cookie.Value)).Scan(&encodedExpiry)
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		expires, err := time.Parse(time.RFC3339Nano, encodedExpiry)
		if err != nil || !expires.After(now().UTC()) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ownerKey{}, true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("pa_csrf")
		if err != nil || !auth.ValidCSRF(cookie.Value, r.Header.Get("X-CSRF-Token")) {
			http.Error(w, "csrf", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
