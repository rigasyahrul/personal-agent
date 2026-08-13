# Task 16 Report

## Status

Implemented DB-enforced session scope/model immutability, session retrieval, and tombstone-first workspace deletion.

## Commit

- `186d671e0f8cca3ad1d56175d00e7dd305d4c2b2` — `feat: enforce session lifecycle invariants`

## RED evidence

Command:

```text
go test ./internal/db ./internal/store -run 'TestSession(ScopeAndImmutableModel|DeleteRemovesOnlyWorkspace)' -v
```

Observed: DB invariant test passed against the pre-existing Phase 1 guards; store package failed to compile because `SessionStore.Get` and `SessionStore.Delete` were undefined. This was the expected missing Task 16 behavior. The first GREEN attempt also exposed a test-fixture setup error (missing source directory), which was corrected without changing production behavior.

## GREEN evidence

Focused command passed:

```text
go test ./internal/db ./internal/store -run 'TestSession(ScopeAndImmutableModel|DeleteRemovesOnlyWorkspace)' -v
```

Package command passed:

```text
go test ./internal/db ./internal/store
```

Full command passed:

```text
go test ./...
```

`git diff --check` also passed before commit.

## Files changed

- `internal/db/migrations/001_init.sql`
- `internal/db/migrate_test.go`
- `internal/store/errors.go`
- `internal/store/sessions.go`
- `internal/store/sessions_test.go`

## Decisions

- Retained migration 001's existing scope shape and amended the Phase 1 triggers to the canonical Task 16 trigger form/name and error text.
- Used SQLite `IS` for null-safe project-vault equality.
- `Get` maps `sql.ErrNoRows` to `ErrNotFound` and reuses `sessionSelect`/`scanSession`.
- `Delete` commits terminal status and timestamps before deleting only the workspace path derived from immutable session scope.
- Terminal deletion is idempotent, as permitted by the brief.
- Added `ErrSessionBusy` only for the specified conditional active-to-terminal update; no Task 17 run coordination was added.

## Self-review

- Confirmed direct inserts reject absent/mismatched project scope and direct model updates fail.
- Confirmed deletion leaves the session retrievable as terminal with `deleted_at` populated.
- Confirmed only the session workspace is removed and project source content remains unchanged.
- Confirmed existing Task 15 create/list/model-validation tests remain enabled and pass.
- Confirmed no Task 17+ behavior or status-board edits were included.

## Concerns

None. As explicitly allowed, a repeated delete of an already-terminal session returns success without retrying workspace cleanup if a prior post-commit filesystem removal failed.

## Fix Round 1

### Tests and RED evidence

- Added `TestSessionDeleteRetriesWorkspaceCleanupAfterTombstone` and `TestSessionDeleteConcurrentCallsConverge` in `internal/store/sessions_test.go`.
- `go test ./internal/store -run 'TestSessionDelete(RetriesWorkspaceCleanupAfterTombstone|ConcurrentCallsConverge)$' -count=20 -v` failed all 20 cleanup-retry runs because the restored workspace remained after the terminal retry; all 20 synchronized concurrent-delete runs passed.

### Fix and GREEN evidence

- Terminal `Delete` calls now commit their read transaction and retry removal of the workspace derived from the stored immutable scope.
- The same focused command passed both tests 20/20 times.
- `go test ./internal/store`, `go test ./internal/db`, and `git diff --check` passed.

### Commit

- This fix-round commit (`fix: make session deletion outcome idempotent`); its SHA is reported in git history and the task result because a commit cannot embed its own content-derived SHA.

### Self-review

- Preserved tombstone-before-cleanup, unknown-session `ErrNotFound`, derived workspace selection, and source preservation.
- Reproduced cleanup failure deterministically with a non-directory path component, verified the tombstone before restoring the filesystem, and verified retry cleanup.
- The synchronized concurrent regression passed repeatedly without production changes for locking or `ErrSessionBusy`; the existing single-connection store serializes the calls and both converge.
- Did not change nullable project-vault semantics or add Task 17+ behavior.
