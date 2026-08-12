# Phase 1 Task 5 Report

## Status

Implemented owner bootstrap and browser authentication/CSRF APIs on `impl/v1-p1-skeleton` in the single Task 5 commit `feat: add owner bootstrap and browser auth` (the commit containing this report).

## RED evidence

Command:

```text
/tmp/go1.24.6/go/bin/go test ./internal/httpapi -run TestBootstrapLoginMeLogoutAndCSRF -v
```

Expected and observed failure (`exitCode: 1`): `AuthRoutes` and `AuthDeps` were undefined in `internal/httpapi/auth_handlers_test.go`; the package failed to build because the requested API did not exist.

## GREEN evidence

Focused command:

```text
/tmp/go1.24.6/go/bin/go test ./internal/auth ./internal/httpapi -v
```

Observed: all bootstrap, CSRF, password/session, route integration, and injected-clock expiry tests passed (`exitCode: 0`).

Final command after gofmt:

```text
/tmp/go1.24.6/go/bin/go test ./...
```

Observed: every package passed (`exitCode: 0`), including `internal/auth` and `internal/httpapi`. `git diff --check` also completed with no errors before commit.

## Security and behavior checks

- Bootstrap requires the configured non-empty token, compares it in constant time, hashes the owner password with the existing Argon2id implementation, and rejects every later bootstrap with HTTP 409.
- Session cookies are `HttpOnly`, `Secure` according to configuration, `SameSite=Lax`, path `/`, and expire after 30 days. CSRF cookies are intentionally browser-readable, use `Secure` according to configuration, `SameSite=Lax`, path `/`, and the same 30-day expiry.
- Only SHA-256 session token hashes are persisted; raw session tokens remain in cookies. Logout deletes the hashed session and expires both cookies.
- CSRF double-submit comparison is non-empty and constant-time. Logout authentication wraps CSRF validation, producing 401 before 403 for anonymous requests.
- Auth routes use the injected `clock.Clock` for deterministic creation and expiry checks. The required exported `RequireAuth` contract remains available with real UTC time, while `AuthRoutes` uses the internal clock-aware helper.
- Stable tested statuses: setup status 200; malformed/short bootstrap 400; invalid bootstrap token 403; bootstrap 201; repeat bootstrap 409; invalid login 401; login/logout 204; authenticated me 200; missing/expired session 401; bad CSRF 403.

## Files

- `internal/auth/bootstrap.go`
- `internal/auth/bootstrap_test.go`
- `internal/auth/csrf.go`
- `internal/httpapi/auth_handlers.go`
- `internal/httpapi/auth_handlers_test.go`
- `internal/httpapi/middleware.go`
- `.superpowers/sdd/2026-08-12-personal-agent-v1/task-5-report.md`

## Self-review

Reviewed the complete diff for contract scope, handler ordering, error mappings, token storage, cookie attributes, deterministic clock use, and accidental Task 6/7 work. The implementation is limited to Task 5 plus this required report. One clock reading is used per login so database and cookie expiry timestamps cannot diverge.

## Concerns

None. Task 6 still needs to wire `AuthRoutes` into the application, as intentionally excluded here.
