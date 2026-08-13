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

## Restore drill

1. In Settings, run **Backup now** and confirm its status is `succeeded`. Record its bundle path (or S3 object prefix) and `manifest_hash`.
2. Stop writes and the application: `docker compose -f deploy/docker-compose.yml stop personal-agent` (use the actual application service name from the Compose file if it differs).
3. Preserve the current volume: `cp -a "$PA_DATA_DIR" "${PA_DATA_DIR}.before-restore"`.
4. Download the selected S3 prefix when applicable (every key under `backups/{run_id}/`). Work from an empty temporary directory, never over the live volume: `mkdir -p /tmp/pa-restore && cp -a "$PA_DATA_DIR/backups/local/RUN_ID/." /tmp/pa-restore/`.
5. Recompute `sha256sum` for every file named by `manifest.json`; compare each result to `files[name]`. Recompute SHA-256 over the exact `manifest.json` bytes and compare it with the recorded `manifest_hash`. Abort on a missing, extra, or mismatched payload. Do not restore `db/` or `backups/` from the live tree into the bundle.
6. Verify the database before replacement: `sqlite3 /tmp/pa-restore/database.sqlite 'PRAGMA integrity_check;'`; the only output must be `ok`.
7. Build a fresh data directory:
   - place `database.sqlite` at `$PA_DATA_DIR/db/personal-agent.sqlite`
   - copy `/tmp/pa-restore/files/**` to `$PA_DATA_DIR/files/**`
   - copy `/tmp/pa-restore/staging/**` to `$PA_DATA_DIR/staging/**` if present
   - remove stale `db/personal-agent.sqlite-wal` or `db/personal-agent.sqlite-shm`
   - do **not** restore the bundle’s own `backups/` directory as live state
8. Start the application: `docker compose -f deploy/docker-compose.yml start personal-agent`.
9. Verify `/health`, sign in, open a known note, and confirm its body renders without an integrity error. Confirm projects, review queue, and the latest durable operation states are present.
10. Record drill date, backup run ID, cutoff, manifest hash, elapsed restore time, and verification result. If any check fails, stop the app, restore `${PA_DATA_DIR}.before-restore`, and investigate before deleting either copy.

## Automated acceptance drill

`go test ./internal/backup -run TestRestoreDrillFindsKnownNote -v` creates a directory bundle, restores it into a fresh data directory layout (`db/personal-agent.sqlite` + `files/**`), opens the restored SQLite database, finds a known ready Note, and verifies its source body's SHA-256. Run it after every bundle-format change.
