# 06 — Reference

Quick tables for operators. Behavior detail lives in the product and API; this page is a field card.

## UI stack (current)

| Piece | Location / note |
|-------|------------------|
| Svelte 5 + TypeScript + Vite + Tailwind | `web/` |
| Production assets | `web/dist` (Go embeds/serves) |
| Dev HMR | `make docker-dev` → Go `:8080` + Vite via `PA_UI_DEV_PROXY` |
| Legacy static UI | Removed (`web-legacy` gone) |
| Frontend tests | `make web-test` (Vitest; **Node 22**) |
| Focus regression | `web/src/**/SessionChat.focus.test.ts` (composer must survive poll) |

## Hash routes

| Pattern | Screen |
|---------|--------|
| `#/home` | Global home |
| `#/projects` | Unfiled projects grid |
| `#/vaults` | Vaults grid |
| `#/vaults/{vaultId}` | Vault home |
| `#/vaults/{vaultId}/projects` | Vault projects |
| `#/vaults/{vaultId}/sessions` | Vault session aggregate |
| `#/vaults/{vaultId}/review` | Vault review |
| `#/projects/{projectId}` | Project hub |
| `#/projects/{projectId}/notes` | Notes list |
| `#/projects/{projectId}/notes/{noteId}` | Notes + selected note |
| `#/projects/{projectId}/sessions` | Project sessions / chat |
| `#/projects/{projectId}/review` | Project review |
| `#/review?scope=all` | Global review |
| `#/settings` | Settings |
| `#settings` | Settings (legacy alias) |

Login and bootstrap are unauthenticated full-page routes (no main chrome).

## Sidebar map

**Global:** Home · Projects (unfiled) · Sessions (disabled) · Vaults · Review · Settings  

**Vault:** Leave vault · Home · Projects · Sessions · Review · Settings  

## Core env vars

| Variable | Role |
|----------|------|
| `PA_DATA_DIR` | Data root (DB, files, backups). Default often `./data` or `/data` in containers |
| `PA_ADDR` | Listen address (e.g. `:8080`) |
| `PA_SECURE_COOKIES` | `true` for HTTPS; `false` only on trusted plain HTTP localhost |
| `BOOTSTRAP_TOKEN` | One-time owner bootstrap secret |
| `PA_MODELS` | Comma-separated `provider:model` list for sessions |
| `OPENAI_API_KEY` | Live chat / generation |
| `OPENAI_BASE_URL` | Optional OpenAI-compatible base URL |
| `PA_TIMEZONE` | IANA zone for display / daily backup hour context |
| `PA_BACKUP_HOUR` | Local hour for Daily backup (default 3) |
| `PA_UI_DEV_PROXY` | Dev only: upstream Vite origin (e.g. `http://127.0.0.1:5173`) so Go proxies non-API GETs |
| `PA_DOMAIN` | Optional public hostname for domain profile / TLS setups |
| S3-related (`PA_S3_*` / as documented in deploy) | Optional backup upload |

Never commit real tokens or keys. Prefer `deploy/.env` (gitignored) for Compose.

## Make targets (common)

| Target | Purpose |
|--------|---------|
| `make help` | Default goal — lists public targets |
| `make web-build` | Vite production build → `web/dist` |
| `make web-test` | Frontend unit tests |
| `make build` | `web-build` + Go binary |
| `make test` / `go test ./...` | Go tests (build `web/dist` first if static tests need assets) |
| `make run` | Run API (expects assets as configured) |
| `make docker-dev` | API + Vite HMR (dev compose override) |
| `make docker-dev-down` | Stop dev stack |

## API surface (owner mental model)

Exact paths and schemas are defined by the Go server. Conceptually:

| Area | Examples of operations |
|------|-------------------------|
| Auth | Bootstrap once; login; session cookie; logout |
| Vaults | List / create / get |
| Projects | List (filter unfiled vs vault); create with optional `vault_id`; get |
| Notes / source | Tree, read, write/publish with review mode |
| Sessions | List by project; create (model + tools flag); messages; run status |
| Workspace | Tree/read when tools allowed; promote `.md` → source |
| Review | Due queue by scope; rate; suspend |
| Settings / backup | Get settings; set schedule; backup now; status |
| Health | Liveness + storage writability |

Idempotency: mutating “do it once” actions often take a client **request key**; retries with the same key should not double-apply.

## Product invariants (v1)

| Rule | Note |
|------|------|
| Single owner | No multi-user ACL |
| `vault_id` immutable after project create | No move between vaults |
| Unfiled = empty `vault_id` | Global Projects grid only |
| Sessions always per-project | No global orphan sessions |
| Source + promote no-clobber | Conflicts are explicit |
| One active backup at a time | 409 if busy |
| One conflicting agent run key | 409 busy |
| Poll must not steal composer focus | Automated UI test |
| Prod image baked | No live `web/` mount on prod compose |

## Docs map

| Doc | Audience |
|-----|----------|
| [README.md](README.md) (this manual) | Owner handbook index |
| [01–06](.) | Day-to-day owner chapters |
| [../ops/deploy.md](../ops/deploy.md) | Deploy, Docker, TLS, HMR |
| Root [README.md](../../README.md) | Repo quick start |
| [../superpowers/](../superpowers/) | Design/plan/status (engineering history) |

## Design history

The Svelte redesign design/plan under `docs/superpowers/specs/` and `plans/` describe how the UI was specified and executed. **Runtime truth** is the code on `main` plus this manual and ops deploy doc. Prefer those over frozen “before” sketches if anything disagrees.

## End of manual

Start over: [01 — Overview](01-overview.md) · Index: [README](README.md)
