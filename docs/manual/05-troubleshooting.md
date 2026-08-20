# 05 — Troubleshooting

Symptoms → likely cause → what to do. For deploy/TLS detail see [`docs/ops/deploy.md`](../ops/deploy.md).

## Cannot open the UI / blank page

| Check | Action |
|-------|--------|
| Process up? | `curl -sf http://127.0.0.1:8080/health` (or your host) |
| Local binary without assets? | Run `make web-build` (Node **22**) so `web/dist` exists; restart Go |
| Docker prod image? | Rebuild image after UI changes (`docker compose … up --build`) — prod has **no** host `web/` mount |
| Dev HMR empty/wrong? | Use `make docker-dev` (not prod compose alone); confirm `PA_UI_DEV_PROXY` is set in the API container |
| Wrong URL | Use the published host/portal URL; sandboxes need Amp **portal** URLs, not raw sandbox hosts |

## Storage not ready / writes fail

Top bar or `/health` shows storage not writable.

- Data directory missing or not writable by the process user
- Volume not mounted (`pa-data` → `/data` in Compose)
- Disk full or permissions

Fix the path/permissions, restart, re-check `/health`.

## Bootstrap rejected

| Symptom | Cause |
|---------|--------|
| Unauthorized / bad token | `BOOTSTRAP_TOKEN` mismatch (env vs form) |
| Conflict / owner exists | Bootstrap already done — use **Sign in** |
| Token too short | Server requires a strong bootstrap secret (≥32 chars recommended) |

After a successful bootstrap, do not expect a second setup screen.

## Login fails after bootstrap

- Wrong password (no recovery in-app)
- Cookies blocked; third-party cookie issues on odd embeddings
- `PA_SECURE_COOKIES=true` on plain HTTP → cookie never sticks — use HTTPS or set `false` only on trusted localhost
- Clock skew rare; still verify host time if sessions die instantly

## Sidebar missing / stuck on auth

- Not logged in → complete bootstrap or sign-in
- Hard-refresh after deploy; clear site data if an old service worker or stale shell is cached (normally none for this app)

## Empty model list / cannot create session

- Set `PA_MODELS` (e.g. `openai:gpt-4o-mini`) and **restart**
- UI copy on Sessions explains missing models

## Chat does not reply / run stuck

- `OPENAI_API_KEY` (and optional `OPENAI_BASE_URL`) for live providers
- Provider outage or bad base URL
- **Busy** (409): another run holds the session or a global single-flight key — wait or use one tab
- Watch session status on the chat page; refresh if the UI lost a poll (draft in the composer should survive polling)

## Composer loses focus while typing

That is a **bug** relative to the product invariant (poll must not rebuild a focused composer). Note the browser and steps; developers should keep `SessionChat.focus.test.ts` green and avoid full-shell `innerHTML`-style rebuilds on poll.

## Promote / save to source fails

- Destination must end in `.md` and must **not** already exist (no-clobber)
- Path traversal or invalid segments rejected
- Workspace tools only exist if the session was created with **Allow workspace files**
- Failed card generation may show a retry affordance — use it rather than double-promoting

## Notes publish conflict

Same no-clobber rules as promote: choose another path or remove/rename the existing source file through the product/API.

## Review empty when you expect cards

- Nothing due yet (caught up)
- Wrong **scope**: global vs vault vs project
- Items suspended
- Publish/promote never created review items (mode **none**)

## Backup now disabled or errors

- Another backup already running — wait
- Storage not writable
- Disk full under `PA_DATA_DIR`
- S3 errors alone should not erase a successful **local** backup; check last-backup detail text

## Vault / project confusion

| Expectation | Product rule |
|-------------|--------------|
| “Move project into a vault later” | **Not supported** in v1 — `vault_id` is fixed at **create** |
| Unfiled project on vault grid | Unfiled only on **global** Projects; enter vault to create vaulted projects |
| Global Sessions list | Disabled — open a project (or vault sessions → project) |

## Docker: UI changes not visible

- **Prod compose:** image-baked `web/dist` — rebuild image
- **Dev:** `make docker-dev` with live mounts + Vite; do not expect host edits on prod-only compose
- Confirm which process listens on 8080 before claiming a fix

## Amp orb specifics

- Prefer portal URLs from `amp orb services ensure` / `amp orb portal`
- Dev data often under `.amp/state/…` — different from host `./data`
- Long-lived servers should use orb supervised services, not ad-hoc background shells

## Still stuck

1. Capture `/health` JSON  
2. Note exact UI route (hash) and HTTP status from failed actions  
3. Check process logs (Docker: `docker compose logs`)  
4. Confirm env: `PA_DATA_DIR`, `PA_SECURE_COOKIES`, `PA_MODELS`, keys  
5. See [06 — Reference](06-reference.md)

## Next

→ [06 — Reference](06-reference.md)
