# Backup and restore

Local backups are **directory bundles** under `$PA_DATA_DIR/backups/local/{run_id}/`:

```
manifest.json
database.sqlite
files/**          # mirror of $PA_DATA_DIR/files/**
staging/**        # optional mirror of $PA_DATA_DIR/staging/**
```

With no backup bucket configured (`PA_BACKUP_S3_BUCKET` / `PA_S3_BUCKET` unset), a verified local bundle is a successful backup. Ambient AWS credentials are ignored when the bucket is unset. With a bucket configured, success means the same directory tree was uploaded under `backups/{run_id}/` (object keys = prefix + relative path). RPO is the time since the last successful run; for a daily schedule, worst-case RPO is about 24 hours. RTO depends on bundle size and operator download/verification time.

Published local bundles are sealed (not owner-writable). Schedule is `settings.backup_schedule`: `off` | `daily` (default `off`). Daily runs fire in-process around local 03:00 (owner timezone). **Backup now** is available in Settings.

## Restore drill (required before backups are operational)

A successful **restore drill** is required before considering backups operational.

Exact sequence:

1. **Stop writers** — stop the application so no mutations run:
   ```sh
   docker compose -f deploy/docker-compose.yml stop
   # or stop the personal-agent process
   ```
2. **Select the last successful bundle** — from Settings history or:
   ```sh
   ls -lt "$PA_DATA_DIR/backups/local"
   ```
   Prefer the newest run with status `succeeded`. Record `run_id`, path, and `manifest_hash`.
3. **Verify manifest and checksums** — work from a copy, never over the live tree:
   ```sh
   mkdir -p /tmp/pa-restore
   cp -a "$PA_DATA_DIR/backups/local/RUN_ID/." /tmp/pa-restore/
   # For each files[name] in manifest.json: sha256sum must match.
   # SHA-256 of the exact manifest.json bytes must match the recorded manifest_hash.
   sqlite3 /tmp/pa-restore/database.sqlite 'PRAGMA integrity_check;'   # must print: ok
   ```
4. **Restore into an empty data directory**:
   ```sh
   cp -a "$PA_DATA_DIR" "${PA_DATA_DIR}.before-restore"
   rm -rf "$PA_DATA_DIR"
   mkdir -p "$PA_DATA_DIR/db" "$PA_DATA_DIR/files"
   cp /tmp/pa-restore/database.sqlite "$PA_DATA_DIR/db/personal-agent.sqlite"
   cp -a /tmp/pa-restore/files/. "$PA_DATA_DIR/files/" 2>/dev/null || true
   cp -a /tmp/pa-restore/staging/. "$PA_DATA_DIR/staging/" 2>/dev/null || true
   rm -f "$PA_DATA_DIR/db/personal-agent.sqlite-wal" "$PA_DATA_DIR/db/personal-agent.sqlite-shm"
   # Do NOT copy the bundle's own backups/ tree in as live state.
   ```
5. **Start the app** and run health / read / integrity checks:
   ```sh
   docker compose -f deploy/docker-compose.yml start
   curl -sf http://127.0.0.1:8080/health
   ```
   Sign in, open a known ready note, confirm body and review queue. Record drill date, run ID, cutoff, manifest hash, elapsed time, and result.
6. If any check fails, stop the app, restore `${PA_DATA_DIR}.before-restore`, and investigate before deleting either copy.

## Automated restore drill

```sh
go test ./internal/backup -run TestRestoreDrillFindsKnownNote -v
```

This creates a directory bundle, restores it into a fresh data directory layout (`db/personal-agent.sqlite` + `files/**`), opens the restored SQLite database, finds a known ready Note, and verifies its source body's SHA-256. Run it after every bundle-format change and before declaring backups operational.

## Acceptance coverage

Spec §13.10 is covered by:

```sh
go test ./internal/acceptance -run TestAcceptance10BackupRestoreLastBundleSucceeds -v
```
