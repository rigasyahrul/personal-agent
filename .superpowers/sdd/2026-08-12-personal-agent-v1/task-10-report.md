# Task 10 Report

## Status

Complete. Vaults and projects persist through dedicated stores, project directories are created in canonical vault-aware locations, null placement is stored as SQL NULL, and project vault placement is immutable.

## Files

- `internal/domain/models.go`: shared `domain.Vault` and `domain.Project` models with repository JSON/time conventions.
- `internal/store/vaults.go`: vault create/list persistence and blank-name validation.
- `internal/store/projects.go`: project create/get/list persistence, validation, nullable placement, and directory creation.
- `internal/store/projects_test.go`: CRUD, validation, directory placement, SQL NULL, and immutability coverage.
- `internal/db/migrations/001_init.sql`: `projects_vault_immutable` trigger in migration 001.

## TDD Evidence

### RED

`go test ./internal/store -run 'Test(VaultAndProjectCRUD|StoresRejectBlankNamesAndUnknownVault|ProjectWithoutVaultPersistsNullPlacement)' -v`

Failed at compile time as expected because `NewVaultStore`, `NewProjectStore`, and `ErrInvalid` were undefined.

### GREEN

The same focused command passed all three Task 10 tests. `go test ./...` subsequently passed every package.

## Commit

`0548e55eaad6ecdc78ead0079b34448a13f62135` — `feat: persist vaults and projects`

## Self-review

- Confirmed tests use `testutil.TempDB`, which opens the full canonical database file path.
- Confirmed names are trimmed and blank names are rejected.
- Confirmed unknown non-empty vault IDs return `ErrInvalid`.
- Confirmed empty vault placement is passed through the existing nullable helper and verified as SQL NULL.
- Confirmed timestamps are UTC RFC3339Nano in SQLite and shared models use `time.Time` plus snake-case JSON tags.
- Confirmed the migration trigger rejects changes between null and non-null placements as well as between vault IDs.
- Ran `gofmt -d` and `git diff --check`; both were clean before commit.

## Concerns

None. As required, the immutable trigger is part of migration 001 and therefore applies to newly initialized databases; no later migration was added for already-migrated databases.

## Fix round 1

### Test files

- `internal/store/projects_test.go`: canonical validation sentinel assertions, missing-project mapping, and real filesystem directory-failure rollback coverage.

### RED

Command:

`go test ./internal/store -run 'Test(StoresRejectBlankNamesAndUnknownVault|ProjectGetMissingReturnsNotFound|ProjectCreateRollsBackWhenDirectoryCreationFails)' -v`

The package failed to compile because the required canonical `store.ErrValidation` and `store.ErrNotFound` sentinels were undefined. This exposed the task-local sentinel and absent not-found contract. The rollback test uses a regular file as the project store's data directory so `os.MkdirAll` fails without mocks; the existing transaction rollback behavior already satisfied that case once the package compiled.

### GREEN

Commands:

- `go test ./internal/store -run 'Test(StoresRejectBlankNamesAndUnknownVault|ProjectGetMissingReturnsNotFound|ProjectCreateRollsBackWhenDirectoryCreationFails)' -v`
- `go test ./internal/store`
- `go test ./...`
- `test -z "$(gofmt -d internal/store/errors.go internal/store/vaults.go internal/store/projects.go internal/store/projects_test.go)"`
- `git diff --check`

All three focused tests passed, the complete `internal/store` package passed, all repository packages passed, and formatting/diff checks produced no output.

### Fix commit

This report is included in the dedicated `fix: use canonical project store errors` commit.

### Self-review

- Validation now returns the shared `ErrValidation` sentinel for blank vault/project names and unknown vault placement.
- `ProjectStore.Get` maps only `sql.ErrNoRows` to `ErrNotFound` and preserves other errors.
- The filesystem test uses no mocks and verifies zero project rows remain after directory creation fails.
- Scope is limited to Task 10 store code/tests, the shared store sentinel owner, and this report; the ledger and board were not edited.
