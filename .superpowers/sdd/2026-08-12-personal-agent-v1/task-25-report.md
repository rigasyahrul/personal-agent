# Task 25 report

## Status
DONE

## Red
Command: `go test ./internal/publish -run 'TestPromote(PublishesOnce|RejectsAnother)' -v`

Decisive output:
```
internal/publish/machine_test.go:199:24: undefined: publish.ConflictError
internal/publish/machine_test.go:225:24: undefined: publish.ConflictError
FAIL github.com/rigasyahrul/personal-agent/internal/publish [build failed]
```

## Implementation summary
Extended the existing direct publication Machine rather than replacing it. Promote validates active project session ownership, derives its rooted workspace from persisted session scope, freezes a rooted regular Markdown file under `staging/promote`, persists transitions in `promote_ops`, resolves publication placement through the project's nullable vault, reserves a Note with promote origin metadata, publishes no-clobber, finalizes, enqueues whole/bite review, and completes. Added typed conflict codes for promote idempotency-key reuse and destination collision while preserving direct behavior.

## Files changed
- `internal/publish/machine.go`
- `internal/publish/machine_test.go`
- `internal/store/direct.go`
- `internal/store/promote.go`

## Green
- `go test ./internal/publish -v` — PASS (`ok .../internal/publish 0.618s`)
- `go test ./...` — PASS; all repository packages green (`internal/publish 0.732s`, no failures)
- `git diff --check` — PASS, no output

## Commit
`173988c8fccae2dc19410c3d0a16fc7732d7267c`

## Self-review findings
- Confirmed direct operations continue using `direct_ops` and direct staging; all pre-existing direct/recovery tests pass.
- Confirmed promote uses `promote_ops`, persisted session scope, rooted `fsroot` reads, project-derived nullable vault placement, exact durable transition chain, no-clobber destination handling, and Note origin metadata.
- Confirmed changed promote fingerprints return `idempotency_key_reused`; existing destinations return `destination_exists` and retain bytes.
- No out-of-scope HTTP, worker, scheduler, UI, status-board, or recovery changes were added.

## Concerns
None.

## Fix round 1

- Covering test: `internal/publish/machine_test.go` — `TestConcurrentPromoteSameKeyRetriesAcrossMachinesConverge`.
- Red: `go test ./internal/publish -run TestConcurrentPromoteSameKeyRetriesAcrossMachinesConverge -count=1 -v` — FAIL; concurrent callers returned uniqueness/database errors, different losing note IDs, and no finalized Note/review item.
- Change: promote insert conflicts now re-read by request key, return `idempotency_key_reused` for a changed fingerprint, and resume the stored operation for a matching fingerprint using its original IDs. Accepted concurrent retries ensure the original operation's staging file exists before resuming.
- Green: focused test with `-count=10` PASS (`ok .../internal/publish 0.316s`); `go test ./internal/publish -count=1` PASS (`0.781s`); `go test ./... -count=1` PASS; `git diff --check` PASS.
- Commit SHA: the Fix round 1 commit containing this report (exact SHA recorded in the final task result).
- Self-review: verified separate `Machine` instances converge on one persisted operation, Note, and review item; losing input IDs are never used after reconciliation; direct publication behavior and stale failed-status handling were not changed.
