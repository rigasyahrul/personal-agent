# 03 — Daily use

How to work in the product after login.

## Home — projects

**Home** lists project cards (notes / sessions / due counts).

### Create a project

1. **New project**  
2. **Name** (required)  
3. Optional **Vault** (label only in v1 — no vault browser)  
4. **Create project** → opens the project overview  

Reserved project-related path names: do not rely on `memory` or `soul` as promotion destinations (rejected as path components).

### Open a project

Click the card. Overview shows:

- Tabs: **Notes** · **Sessions** · **Review**  
- Counts and shortcuts: **New source file**, **Open sessions**, **Review due**

## Notes — source library

Route: `#/projects/{id}/notes`

- Left: **source tree** (folders and notes)  
- **New folder** — relative path (e.g. `guide/examples`)  
- **New Markdown file** — relative path ending in `.md`, body (max **1 MiB** UTF-8), review mode:
  - **None** — publish only  
  - **Whole note** — enqueue whole-note review  
  - **Bites** — enqueue bite generation (needs model/API)  
- **Publish file** — direct create into source (idempotent retry if the same form is resent)

Open a note from the tree to read its body. Paths are project-relative; URLs use **note id**, not the path.

If publish returns a conflict, the path may already exist or integrity checks failed — change the path or fix the tree; the app does not overwrite existing source files.

## Sessions — chat

Route: `#/projects/{id}/sessions`

### Create a session

Requires configured models (`PA_MODELS`).

1. **Title**  
2. **Model** (provider:model from server list)  
3. Optional **Allow workspace files** (default off)  
4. **Create session**  

Provider, model, and project placement are **fixed for the life of the session**.

### Chat

Open a session:

- Message list + **Send**  
- Status line shows run state (polling)  
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

Watch operation status badges on the chat page (including retry for failed card generation when shown). Destination paths are **no-clobber**.

### Delete a session

Use the product/API delete when available from your workflow; deletion marks the session terminal and removes the **workspace** tree. Source notes already promoted stay in the project library.

## Review — spaced repetition

Entry points:

- Nav **Review** → all projects (`scope=all`)  
- Project **Review** → that project only  

If nothing is due: **Caught up**.

For each card:

- **Whole** — open current note, then rate  
- **Bite** — **Reveal answer**, then rate  

Ratings: **Again** · **Hard** · **Good** · **Easy** (scheduler `sm2-lite-v1`).  
You can **suspend** an item when the UI offers it.

Retries of the same rating request key do not double-apply events.

## Suggested daily loop

1. Home — pick a project with due count, or **Review** globally.  
2. Clear due cards.  
3. Work in a **session** or add a **source** file under Notes.  
4. Promote useful workspace drafts.  
5. When the day matters: Settings → **Backup now**.  

## Next

→ [04 — Settings and backup](04-settings-and-backup.md)
