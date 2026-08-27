# Design: Composer `@` workspace file mentions

**Status:** Draft — awaiting user review  
**Date:** 2026-08-27  
**Scope:** Open-session **Reply…** composer (`SessionChat`) only  
**Approach:** 1 — overlay list on the existing textarea (not contenteditable)

**Related:**

- `web/src/components/sessions/SessionChat.svelte` — composer; Enter sends; poll-safe form ancestry
- `GET /api/v1/sessions/{id}/workspace/tree` — same tree as the Files rail / `SessionFilesBar`
- Agent `read_file` when `workspace_files` is granted — no new send payload
- `frontend-ui-craft` — screen spec below is the craft freeze; browser vibe-pass required on UI
- Standing rules: polled SPA never replaces the focused composer; UI tokens in `web/src/app.css`

---

## 1. Why

Users already type `@standing-rule.md` in chat so the agent can `read_file` that workspace path. The composer is a plain textarea: there is no suggestion list, so you must remember names. This pass adds a **code-editor file picker** on `@` without attaching bytes or inventing a new mention protocol.

---

## 2. Goals / non-goals

### Goals

1. Typing `@` (start of draft or after whitespace) opens a file suggestion list **above** the composer card.
2. List is **filename-first** (`standing-rule.md`); nested files show a dim parent path (`notes/`).
3. Choosing a row inserts `@relative/path` plus a trailing space (token **B**).
4. Keyboard matches an editor: arrows, Enter/Tab insert, Escape dismisses. Enter with the list **closed** still sends.
5. Focus stays in the textarea. Composer form node identity is unchanged across poll and file-tab switches.
6. Same file set as session workspace tree (files only). Grant off → `@` is plain text.

### Non-goals

- Hub start composer (`How can I help you today?`)
- Project notes, knowledge, `memory/**`, instruction files
- Contenteditable chips / rich mentions
- Attaching file contents on send (token **A** only: path text)
- New HTTP API
- Creating files from the picker
- Dark mode

---

## 3. Screen spec (craft freeze)

**Goal:** VS Code-style file list while composing, without looking like a dashboard dropdown.

**Surface:** Sticky session composer card on `#/projects/:id/sessions` (and hub-embedded `SessionChat` if present). Overlay is **inside** `.session-composer__card`, anchored to the top of the card (`bottom: 100%`), not a page-level modal.

**States**

| State | UI |
|-------|----|
| Idle | Unchanged Reply… textarea |
| Open, loading | List shell, one-line “Loading files…” |
| Open, populated | Up to **10** rows; one highlighted |
| Open, empty match | “No matching files” |
| Open, error | One-line error; composer still usable |
| Grant off / `foo@bar` | No list |

**Primary action:** pick a file (click or Enter/Tab).  
**Out of scope visually:** icons-as-emoji, indigo menus, heavy shadow/glass, second search field.

**Row (populated):** primary = basename; secondary = parent directory or empty for root files. Highlighted row uses `.mention-option--active` (`--accent-soft` / `--accent`) — do not reuse `.tree-item--active` (rail collision). Row min-height **36–40px**. No bullet/emoji prefixes.

---

## 4. Mention token

An **active mention** exists when the cursor sits in a run:

- starts with `@`
- `@` is at index `0` or the previous character is whitespace (` `, `\n`, `\t`)
- continues through non-whitespace until the first whitespace or end of string
- cursor index is at or after that `@` and at or before the end of the run

Query = characters **after** `@` up to the end of the run (not only up to the cursor). Replacing uses the **whole** run so `@stand|ing-rule.md` still swaps the full token.

`user@host` / `foo@bar` does **not** open the list.

Insert replaces the whole run with `@` + workspace-relative path + one trailing space. Example: `@stand` + pick `notes/standing-rule.md` → `@notes/standing-rule.md `.

---

## 5. Filter and rank

Source: `workspaceTree` entries with `kind === 'file'` (directories never listed).

Filter: case-insensitive; query may match basename or full relative path (`filterEntriesByQuery` or equivalent). Empty query (bare `@`) = all files, ranked, capped.

Rank (stable):

1. Basename starts with query
2. Basename contains query
3. Path contains query
4. `path.localeCompare`

Cap **10** after rank. Do not paginate in v1.

---

## 6. Architecture

Keep the existing `<textarea>`. Do **not** switch to contenteditable.

| Unit | Responsibility |
|------|----------------|
| `web/src/lib/mention-files.ts` | Pure: `activeMention(text, cursor)`, `rankWorkspaceFiles(entries, query)`, `insertMention(text, mention, path)`. No DOM, no API. |
| Overlay markup | Listbox inside the composer card; shown when an active mention exists **and** workspace files are granted. |
| `SessionChat.svelte` | Detect mention on `input` + caret (`selectionStart`); load/cache tree; keyboard; insert via bind `draft`. |

**Load:** first time the picker opens for this session, `api.workspaceTree(sessionId)`. Cache in component state. Refresh when tool messages introduce new `changed_path` values (same trigger as `SessionFilesBar`). Do not fetch on every keystroke.

**Keyboard** (when list is open):

| Key | Action |
|-----|--------|
| ArrowDown / ArrowUp | Move highlight; wrap |
| Enter / Tab | If a highlighted (or first) file row exists: insert it; `preventDefault` — **do not send**. If the list is open but empty/loading/error: `preventDefault` — **do not send**, do not insert. |
| Escape | Close list; leave typed text |
| other | Normal typing; list stays open while the mention stays active |

Respect `isComposing` (IME): do not treat Enter as insert or send while composing.

When the list is closed, `onComposerKeydown` is unchanged (Enter sends, Shift+Enter newline).

**Mouse:** mousedown on a row selects (use `mousedown` + preventDefault so the textarea does not blur).

**a11y:** textarea `aria-autocomplete="list"`, `aria-expanded`, `aria-controls`, `aria-activedescendant` when open. List `role="listbox"`; rows `role="option"` + `aria-selected`. Do not move focus into the list.

**Composer stability:** overlay may mount/unmount. The `<form class="session-composer">` and textarea must not. Existing `SessionChat.focus.test.ts` remains a gate.

---

## 7. Data and errors

- **Grant off** (`workspaceEnabled(session)` false): no fetch, no overlay.
- **Tree error** (403, network, 500): overlay shows a single error line; `draft` / send unchanged.
- **Empty workspace:** “No matching files”.
- **Send:** still `POST .../messages` with `{ content, request_key }`. Content is the string after insert. The model uses `read_file` on the path. No attachment field.

No backend change in this pass.

---

## 8. Tokens

Add composer-scoped classes in `web/src/app.css`, e.g. `.session-composer__mentions`, `.mention-option`, `.mention-option__name`, `.mention-option__path`. Reuse `--panel`, `--border`, `--muted`, `--accent-soft`. Panel + border, light shadow consistent with `.session-composer__card`, not a new elevation language.

Do **not** introduce `bg-indigo-600`, emoji icons, or a second floating search input.

---

## 9. Tests

**Unit (`mention-files.test.ts`):**

- Active mention at start, after space/newline; `foo@bar` inactive; cursor mid-token uses whole run
- Rank: starts-with basename before path substring
- Insert replaces whole token and adds trailing space; preserves text before/after

**SessionChat:**

- `@` with grant + files → listbox with basename
- Filter `@stand` → matching row
- Enter on open list inserts `@path ` and does **not** call `sendMessage`
- Enter with list closed still sends
- Escape closes list, keeps `@query`
- Grant off → no listbox
- Tree reject → error line, textarea identity unchanged
- Existing focus/poll tests still pass

**CSS:** `styles-baseline` (or equivalent) asserts mention overlay classes exist and do not use indigo/scaffold soup.

**Browser vibe-pass (HARD):** open the real session URL, type `@`, confirm list above composer, pick a file, confirm inserted path, send still works. Report URL + viewport + what was seen.

---

## 10. Implementation notes

- Node `>=22 <23` on `PATH` before `make web-test`.
- Local UI loop: `make docker-dev` (no `web/dist` rebuild). Worktree vibe-pass must not assume laptop `:8080` is this checkout.
- If a later Files/Workspace rename lands (`/workspace/tree` → session-files), this picker follows that API; do not invent a parallel client.
