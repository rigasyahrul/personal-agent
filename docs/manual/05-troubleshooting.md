# 05 — Troubleshooting

Short fixes for common v1 issues. For restore and upgrade, use the ops docs.

## Storage / health

| Symptom | What to check |
|---------|----------------|
| Header: storage **unavailable** | `PA_DATA_DIR` exists and is writable by the process user |
| `/health` not 200 | Process down, wrong `PA_ADDR`, or disk full / permissions |
| Compose data missing after restart | Volume `pa-data` still attached; see deploy volume checks |

```sh
curl -sf http://127.0.0.1:8080/health
```

## Bootstrap and login

| Symptom | Likely cause |
|---------|----------------|
| Bootstrap rejected | Wrong `BOOTSTRAP_TOKEN`, or password &lt; 12 characters |
| Bootstrap **conflict / owner exists** | Already bootstrapped — use **Sign in** |
| Login fails | Wrong password; cookies blocked; `PA_SECURE_COOKIES=true` on plain HTTP |
| Logged out unexpectedly | Session expired or cookies cleared |

**Secure cookies:** use `PA_SECURE_COOKIES=false` only on trusted localhost HTTP. On real HTTPS domains, keep `true`.

## Mutations (create, publish, chat, backup)

Browser mutations need a logged-in session **and** CSRF. The UI handles CSRF automatically when cookies work.

| HTTP / message | Meaning |
|----------------|---------|
| **401** | Not logged in (or session invalid) |
| **403** | CSRF / forbidden |
| **409** busy / session_busy | Another agent run is active on that session — wait or use one tab |
| **409** conflict on publish/promote | Path taken, integrity issue, or idempotent conflict — change path or inspect status |
| **400** invalid | Path/body/model validation failed |

Direct publish requires an **Idempotency-Key** (the UI sets this). Retrying the same form reuses the key so you do not double-create.

## Sessions and models

| Symptom | Fix |
|---------|-----|
| “Configure a model before creating a session” | Set `PA_MODELS` and restart |
| Send does nothing useful / runs fail | Set `OPENAI_API_KEY` (and base URL if needed); check run status line |
| Workspace panel missing | Create session with **Allow workspace files** checked |
| Cannot promote | Select a `.md` workspace file; target path must end in `.md` |

## Review and bites

| Symptom | Fix |
|---------|-----|
| Always “Caught up” | No due items; publish/promote with whole/bites review mode |
| Bites stuck / failed | Model/API errors; use **Retry cards** on the session operation badge when shown |
| Wrong “today” | Owner timezone in settings / DB (IANA); restart workers after change if needed |

## Backup

| Symptom | Fix |
|---------|-----|
| Backup failed in Settings | Read error text; disk space; if sink configured, bucket credentials and network |
| Bundle files not deletable | Sealed on purpose — `chmod -R u+w` only on **copies** for restore drills |
| Need full restore | Stop writers → follow [`docs/ops/backup-restore.md`](../ops/backup-restore.md) exactly |

Do **not** run Backup now while manually copying the live data directory.

## Where is my data?

```text
$PA_DATA_DIR/          # default ./data
  db/personal-agent.sqlite
  files/…              # sources + workspaces
  staging/…
  backups/local/{id}/  # sealed bundles
```

Compose: inside the container `/data` on volume `pa-data`.

## Still stuck

1. Reproduce with `curl` health + login from [`docs/ops/deploy.md`](../ops/deploy.md).  
2. Check process logs (compose logs or terminal).  
3. Confirm Go **1.24+** if building from source.  
4. Design / acceptance intent: design spec §11 security and §13 tests (for developers).  

## Next

→ [06 — Reference](06-reference.md)
