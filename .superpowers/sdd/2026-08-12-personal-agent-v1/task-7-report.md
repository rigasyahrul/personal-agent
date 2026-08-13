# Task 7 report

## Outcome

Implemented the complete first-run/authenticated browser skeleton and settings persistence/API contract. The shell checks setup status, renders bootstrap when needed, renders login after bootstrap when `/auth/me` returns 401, and only then exposes Home and hash-based Settings navigation.

## TDD evidence

RED command (after tests, before implementation):

`PATH=/tmp/go1.24.6/bin:$PATH go test ./internal/store ./internal/httpapi -run 'Test(Settings|Static)' -v`

Expected failures observed: undefined `SettingsStore`, `Settings`, and `ErrInvalidSettings`; settings requests returned 404; static test reported missing `../../web/index.html`. (The orb's system Go 1.19 first rejected the Go 1.24 module, so Go 1.24.6 was used explicitly.)

GREEN focused command:

`PATH=/tmp/go1.24.6/bin:$PATH go test ./internal/store ./internal/httpapi -run 'Test(Settings|Static)' -v`

Result: all settings store, settings API, and static shell tests passed.

Full command:

`PATH=/tmp/go1.24.6/bin:$PATH go test ./...`

Result: all packages passed.

JS syntax command:

`find web/js -name '*.js' -print0 | while IFS= read -r -d '' file; do node --input-type=module --check < "$file"; done`

Result: all modules passed syntax checking. (`node --check file.js` treats `.js` as CommonJS without a package manifest, so module input is supplied on stdin.)

Brief self-check: exactly one `/api/v1/home` reference in `pages/home.js`; setup status, auth/me, and settings references present; no `.amp/services.yaml` created.

## Browser state flow

1. Fetch health independently for the header.
2. Fetch `/api/v1/setup/status`.
3. If not bootstrapped, show token + password form and POST both JSON fields to the existing bootstrap contract.
4. If bootstrapped, call `/api/v1/auth/me`; a preserved 401 displays login.
5. After login, render Home. Authenticated navigation exposes Settings, whose stub loads `/api/v1/settings`.

The API helper supports JSON GET/mutations, sends the CSRF cookie through `X-CSRF-Token` on mutations, and throws `APIError` with HTTP status.

## Settings contract

The singleton store reads/writes timezone, default provider, default model ID, and backup schedule. Writes validate the timezone with the IANA location database and schedule as `off|daily`, and use the injected time for `updated_at`. API JSON exposes exactly those four non-secret fields. GET requires authentication; PUT requires authentication before CSRF. Invalid settings map to 400 and database errors to 500.

## Files

- Added static shell under `web/`, including setup and settings page modules.
- Added `internal/store/settings.go` and store tests.
- Added `internal/httpapi/settings_handlers.go`, API tests, and static test.
- Wired settings routes in `internal/httpapi/server.go`.

## Commit

Exactly one Task 7 commit: `feat: add authenticated browser shell and settings` (SHA reported by the implementing agent after commit).

## Self-review and concerns

Confirmed only Task 7 scopes plus this required report are touched, secrets are not represented in store/API DTOs, auth wraps CSRF, and no Phase 2 behavior or service declaration was added. The settings UI is intentionally read-only for this phase. The orb's preinstalled Go is 1.19; verification used downloaded Go 1.24.6 without modifying repository setup.
