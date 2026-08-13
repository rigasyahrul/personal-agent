# Task 21 Report

## Status

DONE

## RED

- Command: `go test ./internal/fsroot ./internal/agent/tools -v`
- Decisive failure: `*fsroot.Root` had no `WriteFileAtomic`, `EditFileAtomic`, or `Tree` methods, and `internal/agent/tools` had no production Go files. The command exited 1.

## GREEN

- `go test ./internal/fsroot ./internal/agent/tools -v` — PASS (10 fsroot tests and 3 workspace-tool tests).
- `go test ./...` — PASS for all repository packages.
- `git diff --check` — PASS.

## Commit

- `19dc3a934d6ae5340522cbd2079b0009fd37d9fd` (`feat: add rooted workspace file tools`)

## Files Changed

- `internal/fsroot/root.go`
- `internal/fsroot/root_test.go`
- `internal/agent/tools/workspace.go`
- `internal/agent/tools/workspace_test.go`

## Self-review / Spec Notes

- Extended the existing Phase 2 rooted implementation rather than replacing it; `WriteFileNoReplace`, `Walk`, and their regression tests remain intact.
- Added same-parent temporary-file replacement with sync, strict rooted path checks, no-follow parent lookup, regular-file-only replacement, the 1 MiB body cap, exact-once edits, and sorted type-neutral tree entries.
- Workspace dispatch exposes only `read_file`, `write_file`, `edit_file`, and `mkdir`; JSON rejects unknown fields and trailing values, and context cancellation is checked before execution. No shell tool is exposed.
- Tests cover text files, traversal, symlink escape, special nodes, oversized content, exact edits, malformed tool arguments, unknown tools, cancellation, and existing Phase 2 behavior.

## Concerns

None.

## Fix Round 1

### Status

DONE

### Findings and Resolutions

1. Reworked filesystem ownership and file creation, inspection, cleanup, and directory syncing around Go 1.24 `os.Root`; removed `x/sys/unix` and the duplicate Linux root descriptor. Phase 2 `WriteFileNoReplace` and `Walk` behavior remains covered.
2. Added atomic no-replace hard-link commit semantics for a destination absent at commit time, so a special node created after temporary-file creation is preserved rather than replaced. Existing regular files retain same-parent atomic rename replacement.
3. Added a focused commit-race regression that creates a FIFO after observing the temporary sibling and verifies the failed commit preserves it and cleans up.

### RED

- Command: `go test ./internal/fsroot -run TestWriteFileAtomicDoesNotReplaceSpecialFileCreatedDuringCommit -count=1 -v`
- Decisive output: `root_test.go:265: write error = <nil>, want collision`; test failed and command exited 1 because `Renameat` replaced the FIFO created during the commit window.

### GREEN

- `go test ./internal/fsroot -run TestWriteFileAtomicDoesNotReplaceSpecialFileCreatedDuringCommit -count=10 -v` — PASS (10/10).
- `go test ./internal/fsroot ./internal/agent/tools -v` — PASS (11 fsroot tests and 3 workspace-tool tests).
- `git diff --check` — PASS.

### Commit

- `a6c6fbf7140e27554a60397a1bc74b1ab69eb5f7` (`fix: harden rooted atomic workspace writes`)

### Files Changed

- `internal/fsroot/root.go`
- `internal/fsroot/root_test.go`
- `.superpowers/sdd/2026-08-12-personal-agent-v1/task-21-report.md`

### Concerns

- Go 1.24 does not provide `os.Root.Rename` or `os.Root.Link`; the commit step therefore uses standard-library `os.Rename`/`os.Link` on paths whose parent was first opened and validated through `os.Root`. All available Go 1.24 file lifecycle operations remain rooted, and no platform-specific syscall is used.
