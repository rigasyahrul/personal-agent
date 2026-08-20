# 01 — Overview

## What Personal Agent is

A **browser UI + single Go API** you run yourself. One owner account. All durable state lives under one data directory (`PA_DATA_DIR`, default `./data`): SQLite, project source files, session workspaces, and local backup bundles.

The UI is a **Svelte 5** single-page app (hash routes). Production serves built assets from `web/dist`; day-to-day coding uses Vite HMR under `make docker-dev` (see the root README and [`docs/ops/deploy.md`](../ops/deploy.md)).

Optional **OpenAI-compatible** HTTP is used for chat and bite generation. Optional **S3-compatible** storage is for **backup upload only** — the app works fully without a bucket.

## Mental model

```text
Global desk
  ├── Home (dashboard)
  ├── Projects without a vault (“unfiled”)
  ├── Vaults (named containers)
  │     └── Projects locked to that vault
  │           └── Source notes · Sessions · Review
  └── Global Review · Settings

Backup (snapshot of DB + files)
```

| Concept | Meaning |
|---------|---------|
| **Vault** | Optional named container for related projects. First-class nav. Enter a vault to scope the sidebar. |
| **Unfiled project** | Project with no vault (`vault_id` empty). Lives on the **global** Projects grid. |
| **Project** | A learning container: notes, sessions, scoped review. `vault_id` is fixed at create time. |
| **Source note** | Durable Markdown under the project source tree |
| **Session** | One model, optional workspace file tools, chat history (always **per-project**) |
| **Promote** | Copy a workspace `.md` into project source (no clobber) |
| **Review** | Spaced repetition (`sm2-lite-v1`): Again / Hard / Good / Easy — global, vault, or project scope |
| **Backup** | Sealed directory bundle of DB + files (+ optional S3) |

## What you will do most days

1. Work from the **global desk** or **enter a vault**.
2. Open a **project** (hub → Notes / Sessions / Review).
3. Either **chat** in a session (and promote good drafts) or manage **source** notes.
4. Clear **due review** (sidebar Review, or project / vault Review).
5. Occasionally **Backup now** (or leave Daily schedule on).

## Non-goals (v1)

Do not expect:

- Multi-user or multi-tenant auth
- Arbitrary shell / host filesystem for the agent
- Full-text search across the library
- Safe public HTTP without HTTPS
- A separate mobile client

Vaults **are** first-class in the UI (create, search, enter/leave). They are not a multi-tenant permission system.

## Navigation in the app

After login you get a **collapsible left sidebar** (state remembered in the browser) and a top bar with storage health.

### Global sidebar

| Item | What it does |
|------|----------------|
| **Home** | Dashboard: due review summary, recent unfiled projects, shortcuts |
| **Projects** | Searchable grid of **unfiled** projects + create (no vault) |
| **Sessions** | Disabled until you open a project (sessions are always per-project) |
| **Vaults** | Searchable vault cards + create; open a vault to enter vault context |
| **Review** | Due cards across all projects (`#/review?scope=all`) |
| **Settings** | Timezone display, backup schedule, Backup now |

### Vault sidebar (after you enter a vault)

| Item | What it does |
|------|----------------|
| **Leave vault** | Back to the global desk (`#/home`) |
| **Home** | Vault dashboard (counts + recent projects in this vault) |
| **Projects** | Projects in this vault + create (vault locked on create) |
| **Sessions** | Sessions aggregated from this vault’s projects |
| **Review** | Due cards for projects in this vault only |
| **Settings** | Same settings as global |

Project cards that belong to a vault show a **vault name** badge.

### Hash routes (examples)

| Route | Screen |
|-------|--------|
| `#/home` | Global home dashboard |
| `#/projects` | Unfiled projects grid |
| `#/vaults` | Vaults grid |
| `#/vaults/{vaultId}` | Vault home |
| `#/vaults/{vaultId}/projects` | Vault projects |
| `#/vaults/{vaultId}/sessions` | Vault sessions (aggregate) |
| `#/vaults/{vaultId}/review` | Vault review queue |
| `#/projects/{id}` | Project hub |
| `#/projects/{id}/notes` | Notes (two-pane) |
| `#/projects/{id}/notes/{noteId}` | Notes with a note selected |
| `#/projects/{id}/sessions` | Project sessions / chat |
| `#/projects/{id}/review` | Project review |
| `#/review?scope=all` | Global review |
| `#/settings` | Settings (legacy `#settings` still works) |

## Next

→ [02 — Install and first run](02-install-and-first-run.md)
