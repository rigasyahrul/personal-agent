package httpapi

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/auth"
)

type ownerKey struct{}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": code, "message": message})
}

func RequireAuth(db *sql.DB, next http.Handler) http.Handler {
	return requireAuthAt(db, func() time.Time { return time.Now().UTC() }, next)
}

func requireAuthAt(db *sql.DB, now func() time.Time, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("pa_session")
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		var encodedExpiry string
		err = db.QueryRowContext(r.Context(), "SELECT expires_at FROM auth_sessions WHERE token_hash=?", auth.TokenHash(cookie.Value)).Scan(&encodedExpiry)
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "database error")
			return
		}
		expires, err := time.Parse(time.RFC3339Nano, encodedExpiry)
		if err != nil || !expires.After(now().UTC()) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		ctx := context.WithValue(r.Context(), ownerKey{}, true)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequireCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("pa_csrf")
		if err != nil || cookie.Value == "" || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(r.Header.Get("X-CSRF-Token"))) != 1 {
			writeError(w, http.StatusForbidden, "csrf_failed", "CSRF token missing or invalid")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// securedMutation applies auth then CSRF so unauthenticated requests get 401
// before CSRF evaluation (authenticated mismatches get 403).
func securedMutation(db *sql.DB, now func() time.Time, next http.Handler) http.Handler {
	return requireAuthAt(db, now, RequireCSRF(next))
}
