# 04 — Settings and backup

Route: `#settings` (also `#/settings`).

## Settings screen

Read-only display of:

- **Timezone** (IANA string used for review “today” and daily backup clock)  
- **Default provider / model** (if set)  

### Backup section

| Control | Meaning |
|---------|---------|
| Last successful backup time | From the last succeeded run |
| Last failure | Shown when newer than last success |
| Remote sink configured | **yes/no** — bucket env present (secrets never shown) |
| **Schedule** | **Off** or **Daily** |
| **Backup now** | Starts a backup run immediately |

**Daily** fires in-process around **local 03:00** in the owner timezone (UTC if unset). Missed ticks catch up at most once after startup.

Changing schedule is saved immediately when you change the dropdown.

## What a backup is

A **directory bundle** under:

```text
$PA_DATA_DIR/backups/local/{run_id}/
  manifest.json
  database.sqlite
  files/**      # project sources + workspaces mirror
  staging/**    # optional
```

Bundles are **sealed** (not owner-writable). The app does **not** require a `.tar.gz`.

| Mode | Success means |
|------|----------------|
| Local only (no bucket) | Directory bundle written and recorded |
| Bucket configured | Same tree uploaded under `backups/{run_id}/` as well |

Ambient cloud credentials are **ignored** when no backup bucket is configured, so local-only mode still starts cleanly.

Env names and restore procedure: [`docs/ops/backup-restore.md`](../ops/backup-restore.md).

## Practical habits

1. After important learning sessions, click **Backup now** and confirm success in Settings.  
2. Before upgrades or risky experiments, backup first (see deploy upgrade sequence).  
3. Before you trust backups operationally, run the **restore drill** once from the ops doc (stop writers → copy bundle → verify checksums → restore to empty data dir → health + open a known note).  
4. Store `manifest_hash` / run id for the drill record.

## Models and chat credentials

Models are **not** edited as free text in Settings for session create — they come from server env:

| Variable | Role |
|----------|------|
| `PA_MODELS` | `provider:model_id,...` list |
| `OPENAI_API_KEY` | Chat / bites |
| `OPENAI_BASE_URL` | Optional compatible base URL |

Restart the process after changing env. See [06 — Reference](06-reference.md).

## Next

→ [05 — Troubleshooting](05-troubleshooting.md)
