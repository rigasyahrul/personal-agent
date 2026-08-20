# 04 — Settings and backup

Settings live at **`#/settings`** (sidebar **Settings** in both global and vault context). Legacy hash `#settings` still opens the same screen.

## What the Settings page shows

| Area | Meaning |
|------|---------|
| **Timezone** | Display-only IANA zone from the server (`PA_TIMEZONE` / process default). Change via env + restart — not an in-app edit field. |
| **Backup schedule** | **Off** or **Daily**. Daily uses the server’s configured local hour (`PA_BACKUP_HOUR`, default 3). |
| **Last backup** | Status from the last run (ok / failed / never), with detail when available. |
| **Backup now** | Starts a local sealed backup under the data directory. Button disables while a backup is already running. |

Schedule changes call the settings API and refresh the page state. Only one backup job runs at a time (409 if already in progress).

## What a backup contains

A **sealed directory bundle** under the data root (typically `data/backups/…`):

- SQLite database copy used by the app
- Project source trees and session workspaces as stored on disk
- Manifest metadata for the run

Exact on-disk layout is an implementation detail; treat the whole bundle as the unit of restore, not individual files inside it.

## Optional S3 upload

If you configure S3-compatible env vars (`PA_S3_*` / related keys — see [06 — Reference](06-reference.md) and [`docs/ops/deploy.md`](../ops/deploy.md)):

- After a successful **local** backup, the server may upload the bundle
- Missing or misconfigured S3 does **not** block local backup success; upload failure is reported separately in backup status when applicable
- S3 is **backup transport only** — the app never requires a bucket for day-to-day use

## Operator habits

1. After important work: **Backup now** and wait for success.
2. Prefer **Daily** schedule for always-on hosts (set hour via env).
3. Keep copies of sealed bundles off the primary host (download or S3).
4. Test restore on a **separate** data directory before you need it in anger.
5. Never point two live processes at the same `PA_DATA_DIR`.

## Restore (high level)

Restore is an **ops** procedure, not a full in-app wizard:

1. Stop the running process (and any second instance).
2. Replace or restore the data directory contents from a known-good sealed backup using your host’s procedure (see deploy docs if provided).
3. Start again; confirm `/health` and login.
4. Spot-check projects, notes, and a session.

If your deploy docs describe a specific restore script or volume steps, follow those over improvising file copies while the process is up.

## Security notes

- Session cookie is **HttpOnly**; with `PA_SECURE_COOKIES=true` it is HTTPS-only
- Owner password is not recoverable from the UI — use offline secrets
- Bootstrap token is one-time; after bootstrap it should not remain in casual shell history on shared machines

## Next

→ [05 — Troubleshooting](05-troubleshooting.md)
