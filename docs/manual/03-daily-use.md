# 03 — Daily use

How to work in the product after login.

## Global desk vs vault

The sidebar switches between two contexts:

| Context | How you get there | What “Projects” means |
|---------|-------------------|------------------------|
| **Global** | Default after login; **Leave vault** | Projects **without** a vault (unfiled) |
| **Vault** | Open a vault from **Vaults** | Only projects in that vault |

Unfiled = empty / missing `vault_id`. Vaulted projects show a **vault name** badge on cards.

## Home (global)

Route: `#/home`

- Due-review summary and unfiled-project count
- Shortcuts: **New project** → Projects grid; **New vault** → Vaults grid
- **Recent** unfiled project cards

## Projects (unfiled)

Route: `#/projects`

1. Search the card grid by name.
2. **Create** with a name only — the project is created **without** a vault (`vault_id: null`).
3. Open a card → **project hub**.

To put work inside a vault, **enter that vault first**, then create a project there (vault is locked on create).

## Vaults

Route: `#/vaults`

1. Search vault cards.
2. **Create vault** with a name.
3. Click a vault (or create) → **vault home** (`#/vaults/{id}`).

### Inside a vault

| Screen | Route | Use |
|--------|-------|-----|
| Vault home | `#/vaults/{id}` | Counts and recent projects in this vault |
| Vault projects | `#/vaults/{id}/projects` | Search + create project **locked** to this vault |
| Vault sessions | `#/vaults/{id}/sessions` | Sessions from all projects in the vault (open one → project sessions) |
| Vault review | `#/vaults/{id}/review` | Due cards only for this vault’s projects |

**Leave vault** returns to `#/home` and restores the global sidebar.

## Project hub

Route: `#/projects/{id}`

- Breadcrumbs (Vaults → vault → project, or Projects → project when unfiled)
- Metrics: notes / sessions / due
- Links: **Notes**, **Sessions**, **Review**

When you open a vaulted project from a deep link, the shell stays in **vault** context once the project loads.

## Notes — source library

Route: `#/projects/{id}/notes` (optional `#/projects/{id}/notes/{noteId}`)

- **Two panes:** tree on the left, reader on the right
- Create folders / Markdown under the project source tree (same rules as before: paths, 1 MiB body, review modes on publish)
- Publish is **no-clobber**; conflicts mean change the path or fix the tree

Open a note from the tree to read it. URLs use **note id**, not the path.

## Sessions — chat

Route: `#/projects/{id}/sessions`

Sessions are always **per-project**. Global sidebar “Sessions” stays disabled until you are in a project (or use vault sessions to jump into one).

### Create a session

Requires configured models (`PA_MODELS`).

1. **Title**
2. **Model** (provider:model from server list)
3. Optional **Allow workspace files** (default off)
4. **New session**

Provider, model, and project placement are **fixed for the life of the session**.

### Chat

Open a session:

- Message list + sticky **Message** composer + **Send**
- Status line shows run state (polling)
- Polling **must not** steal focus from the composer (if you are typing, the draft stays)
- If another run is active under a different key, the API returns **busy** (409) — wait or use one tab

Chat needs a working OpenAI-compatible endpoint when you want model replies.

### Workspace panel

Only if **Allow workspace files** was checked:

- Browse workspace tree and open files
- For a promotable `.md` file, **Save to source** opens promote

### Promote (“Save to source”)

1. Select a workspace `.md`
2. **Save to source**
3. Set **target path** (must end in `.md`)
4. Choose review mode: none / whole / bites
5. **Save**

Watch operation status badges on the chat page (including retry for failed card generation when shown). Destination paths are **no-clobber**. Idempotent retries reuse the same key for the same payload.

## Review — spaced repetition

Entry points:

- Sidebar **Review** (global) → all projects (`#/review?scope=all`)
- Vault **Review** → that vault’s projects only
- Project hub **Review** → that project only

If nothing is due: **Caught up**.

For each card:

- **Whole** — open current note, then rate
- **Bite** — **Reveal answer**, then rate

Ratings: **Again** · **Hard** · **Good** · **Easy** (scheduler `sm2-lite-v1`).  
You can **suspend** an item when the UI offers it.

Retries of the same rating request key do not double-apply events.

## Suggested daily loop

1. Home or a vault — pick work with due count, or open **Review**.
2. Clear due cards.
3. Work in a **session** or add a **source** file under Notes.
4. Promote useful workspace drafts.
5. When the day matters: Settings → **Backup now**.

## Next

→ [04 — Settings and backup](04-settings-and-backup.md)
