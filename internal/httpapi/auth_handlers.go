package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/rigasyahrul/personal-agent/internal/auth"
	"github.com/rigasyahrul/personal-agent/internal/clock"
)

const sessionLifetime = 30 * 24 * time.Hour

type AuthDeps struct {
	DB             *sql.DB
	Clock          clock.Clock
	BootstrapToken string
	SecureCookies  bool
}

func AuthRoutes(mux *http.ServeMux, deps AuthDeps) {
	writeJSON := func(w http.ResponseWriter, value any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(value)
	}
	mux.HandleFunc("GET /api/v1/setup/status", func(w http.ResponseWriter, r *http.Request) {
		var count int
		if err := deps.DB.QueryRowContext(r.Context(), "SELECT count(*) FROM owner").Scan(&count); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]bool{"bootstrapped": count != 0})
	})
	mux.HandleFunc("POST /api/v1/setup/bootstrap", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Token    string `json:"token"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || len(input.Password) < 12 {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid bootstrap request")
			return
		}
		// Accept bootstrap token via Authorization: Bearer or JSON body (UI).
		provided := input.Token
		if authz := r.Header.Get("Authorization"); len(authz) > 7 && (authz[:7] == "Bearer " || authz[:7] == "bearer ") {
			provided = authz[7:]
		}
		err := auth.Bootstrap(r.Context(), deps.DB, deps.BootstrapToken, provided, input.Password, deps.Clock.Now())
		switch {
		case errors.Is(err, auth.ErrOwnerExists), errors.Is(err, auth.ErrBootstrapped):
			// Never reveal whether the bootstrap token was correct after owner exists.
			writeError(w, http.StatusConflict, "owner_exists", "owner already exists")
		case errors.Is(err, auth.ErrBootstrapToken):
			writeError(w, http.StatusForbidden, "invalid_bootstrap_token", "invalid bootstrap token")
		case err != nil:
			writeError(w, http.StatusInternalServerError, "internal_error", "database error")
		default:
			w.WriteHeader(http.StatusCreated)
		}
	})
	mux.HandleFunc("POST /api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		var passwordHash string
		err := deps.DB.QueryRowContext(r.Context(), "SELECT password_hash FROM owner WHERE id=1").Scan(&passwordHash)
		if errors.Is(err, sql.ErrNoRows) || err == nil && !auth.CheckPassword(passwordHash, input.Password) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		sessionToken, csrfToken := auth.NewSessionToken(), auth.NewSessionToken()
		now := deps.Clock.Now().UTC()
		expires := now.Add(sessionLifetime)
		if _, err := deps.DB.ExecContext(r.Context(), "INSERT INTO auth_sessions(token_hash,csrf_token,expires_at,created_at) VALUES(?,?,?,?)", auth.TokenHash(sessionToken), csrfToken, expires.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano)); err != nil {
			http.Error(w, "session error", http.StatusInternalServerError)
			return
		}
		setAuthCookie(w, "pa_session", sessionToken, true, deps.SecureCookies, expires)
		setAuthCookie(w, "pa_csrf", csrfToken, false, deps.SecureCookies, expires)
		w.WriteHeader(http.StatusNoContent)
	})

	authenticated := func(next http.Handler) http.Handler {
		return requireAuthAt(deps.DB, deps.Clock.Now, next)
	}
	mux.Handle("GET /api/v1/auth/me", authenticated(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]bool{"owner": true})
	})))
	// Authentication intentionally wraps CSRF so an anonymous request is 401.
	mux.Handle("POST /api/v1/auth/logout", authenticated(RequireCSRF(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, _ := r.Cookie("pa_session")
		if _, err := deps.DB.ExecContext(r.Context(), "DELETE FROM auth_sessions WHERE token_hash=?", auth.TokenHash(cookie.Value)); err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		clearAuthCookie(w, "pa_session", true, deps.SecureCookies)
		clearAuthCookie(w, "pa_csrf", false, deps.SecureCookies)
		w.WriteHeader(http.StatusNoContent)
	}))))
}

func setAuthCookie(w http.ResponseWriter, name, value string, httpOnly, secure bool, expires time.Time) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: httpOnly, Secure: secure, SameSite: http.SameSiteLaxMode, Expires: expires, MaxAge: int(sessionLifetime.Seconds())})
}

func clearAuthCookie(w http.ResponseWriter, name string, httpOnly, secure bool) {
	http.SetCookie(w, &http.Cookie{Name: name, Path: "/", HttpOnly: httpOnly, Secure: secure, SameSite: http.SameSiteLaxMode, Expires: time.Unix(1, 0).UTC(), MaxAge: -1})
}
