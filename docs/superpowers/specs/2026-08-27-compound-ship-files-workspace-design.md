# Design: Compound as Ship + Files rail + Workspace notes

**Status:** Draft — awaiting user review  
**Date:** 2026-08-27  
**Stack:** Existing Svelte 5 + TypeScript + Vite SPA under `web/`; Go session-file / promote APIs. Session file tools are **always on** (not a user grant).  
**Route:** `#/projects/:id` (hub + embedded session). Compound control lives on `SessionChat` wherever that component mounts.

**Related:**

- `docs/superpowers/specs/2026-08-21-project-rail-icon-chrome-design.md` (rail chrome; this spec **adds Files** for session writes, **renames** project-notes tab to **Workspace**, **reorders** left icons)
- `docs/superpowers/specs/2026-08-22-obsidian-memory-knowledge-design.md` (human-gated compound API — **not** used by the button after this change)
- `frontend-ui-craft` — screen spec below is the craft freeze; browser vibe-pass required on UI
- Standing rules: polled SPA composer focus; hub startSession not create∧send atomic; rail Files ≠ session-only routing; creates use modals (promote stays a `<dialog>`)

**Approach (approved in design dialogue):** Amp **Ship**-style Compound (canned chat send) + rail **Files** for session-written docs + copy into **Workspace** (project notes) + in-chat file cards + success toast. Rejected: transcript-dump `user_context`; silent review-card generate as the button UX; Amp “Changes” / “Drafts” naming; move-on-save; writing `AGENTS.md` / `memory/**` from this button; keeping code name `workspace_*` for session scratch.

---

## 0. Vocabulary (do not mix)

Product names and code names must match this table so the next agent does not treat session scratch as project Workspace.

| Product UI | Disk | Code / API (after this change) | Today (rename away) |
|------------|------|--------------------------------|---------------------|
| **Files** | `sessions/{session_id}/**` | session files: `SessionFile`, `GET /api/v1/sessions/{id}/files/tree`, `GET /api/v1/sessions/{id}/files?path=` | `workspace_*`, `/workspace/tree`, `/workspace/file`, `source: 'workspace'` |
| **Workspace** | project `source/**` | project notes: existing `listProjectNotes` / `getProjectNote` | rail tab label **Files** |
| **Config** | `SOUL.md` `SYSTEM.md` `AGENTS.md` | instruction APIs | unchanged |

- Tool names `write_file` / `edit_file` / `mkdir` stay (they write **session files**).
- Open-file meta: `source: 'session-file'` (not `'workspace'`). Project notes stay `source: 'project-note'`.
- Grant key: `session_files`. Stored `workspace_files: false` still means **on** (always-on; no UI).
- Promote body field: `session_path` (not `workspace_path`).
- Rail tab ids: `files` \| `workspace` \| `config`. **Migrate** stored `pa.projectRail.tab=files` → `workspace` (old `files` meant project notes).

Do **not** leave aliases named `workspace` for session scratch.

---

## 1. Why

Clicking **Compound** today does not compound. It posts the last six messages as `user_context`, and `Runs.Admit` inserts that dump as a normal user chat row. The model answers like ChatGPT. The user expected Amp **Ship**: one click sends a **fixed instruction**, then the agent works in the thread.

A normal chat turn cannot write project-durable `AGENTS.md` / `memory/**` (`write_knowledge` is not in slice 1). Agent `write_file` lands in `sessions/{id}/`. There is no grant UI, but many sessions (including **ivory bold**) were created with `workspace_files: false`, so the agent cannot create files at all.

Session writes are invisible: no in-chat card, and Files on the rail is project notes (empty: “No project files available.”). Product **Workspace** is the project library; product **Files** is this session’s writes.

---

## 2. Goals and non-goals

### Goals

1. **Compound** sends one canned user message via `sendMessage`. No transcript dump. No `POST /compound` from this button. No review card on this path.
2. **Every session can write session files.** No grant checkbox. Ivory bold and Test 1 included.
3. Right rail left-cluster order: **Files · Workspace · Config**. Right cluster unchanged (Expand · Collapse).
4. **Files** lists this session’s written docs. Click opens the file. Hover **⋯** → **Save to Workspace?** → existing promote modal. Save is a **copy**; the session file stays. After success, stay on Files and toast **“Saved to Workspace.”** Workspace list refreshes immediately.
5. When the agent writes a file, a **file card** appears in the chat **and** the path appears under rail Files.
6. Small reusable **Toast** (success first) using `--success` / `--success-soft`.
7. Rename session-scratch identifiers off `workspace_*` (API paths, types, grant, promote field, `source`).

### Non-goals

- Deleting the compound proposal API / `CompoundReviewCard`
- Agent writes to `AGENTS.md` / `memory/**` / `write_knowledge`
- Auto-compound or a compound inbox
- Workspace (project notes) tree/search redesign
- Toast queue, undo, error/warn variants
- Standalone session route getting this rail (hub rail only)
- Dark mode
- Promote destination stays `source/**` (not memory)

---

## 3. Surfaces

| Surface | Owner | Change |
|---------|--------|--------|
| Compound button | `SessionChat.svelte` | `sendMessage` with locked prompt; drop `createCompound` / review-card from the button |
| Session-file grant | Runner + create + `ProjectSessionsPage` | Always on. Remove “Allow workspace files”. Create `session_files: true`. Runtime tools on even if stored `workspace_files: false` |
| Session-file API | `httpapi` + `web/src/lib/api` | New `/files/tree` and `/files?path=`; delete old `/workspace/*` routes from handlers |
| Rail tabs | `project-rail-prefs.ts`, `ProjectRail.svelte` | Tabs `files` \| `workspace` \| `config`; left order Files, Workspace, Config |
| Files panel | `ProjectRail.svelte` | Session-file tree; ⋯ promote |
| Workspace panel | `ProjectRail.svelte` | Project notes only (former Files list) |
| Chat thread | `SessionChat.svelte` | File card per tool `changed_path` |
| Promote | `PromoteDialog` | `session_path`; on success refresh Workspace data, keep Files selected, toast |
| Toast | new `Toast.svelte` + `app.css` | Success tone only |

Default hub enter: **Config**. Invalid stored tab → `config`. Old stored `files` → **workspace**.

---

## 4. Compound (Ship-style)

### Click

Disabled while send-in-flight or run `queued`/`running`. New `request_key`. Does **not** call `api.createCompound`.

```text
Compound this conversation. Extract only non-obvious learnings. Write durable notes as session files — they stay under Files until I save them to Workspace. Prefer a short standing-rule note over a diary. If nothing reusable was learned, say so and stop. Do not dump the transcript.
```

That string is the user message. Normal chat run. `write_file` / `edit_file` / `mkdir` always available.

### After send

- Do not put the prompt in the composer.
- Poller shows the user row + run + any file cards.
- Send failure: same alert as Send.
- Composer form identity unchanged (focus gate).

### Old pipeline

`POST /compound`, `StartCompound`, review card, decide/publish stay in the repo. This button does not call them.

---

## 5. Files vs Workspace

| Tab | Data | Writers |
|-----|------|---------|
| **Files** | Session-file tree for the **open session** | Agent `write_file` / `edit_file` / `mkdir` |
| **Workspace** | Project notes (`listProjectNotes`) | Human create + **copy** from Files via promote |

No open session, or no session files: Files shows **“No files in this session.”** Do not use another session’s files. Do not hide Files because of a stored grant.

Click a session file → `onOpenFile` with `source: 'session-file'`. Directories not openable.

### Overflow (⋯)

- Hover/focus on a **file** row (not directories).
- **Save to Workspace?**
- Existing promote `<dialog>`.
- **Copy:** session file remains; new `source/**` note created.
- Success: stay on **Files**; refresh project-notes list; toast **“Saved to Workspace.”**
- Failure: dialog error; no toast; both lists unchanged.

Deleting the session does not delete the Workspace note.

---

## 6. In-chat file cards

Tool messages already carry `changed_path` (field or JSON `content`). SessionChat does not render them as files today.

**Rule:** every successful session-file write shows:

1. A **file card** in the thread (basename + path; click opens that session file).
2. The same path in rail **Files** after the next tree load (poll or invalidate on `changed_path`).

Card is not a transcript dump. Not a raw `role=other` tool JSON blob.

If a card exists and Files is missing that path → bug.  
If Files has a new path and the writing turn has no card → bug.

`mkdir` may omit a card (directory). `write_file` / `edit_file` must show a card.

---

## 7. Toast

| Token | Value |
|-------|--------|
| Background | `var(--success-soft)` |
| Text | `var(--success)` |
| Border | same family as `.health-pill[data-tone='ok']` (`#bbf7d0`) |
| Radius | `var(--radius-sm)` |
| Copy | `Saved to Workspace.` |
| A11y | `role="status"` `aria-live="polite"` |
| Lifetime | ~3s; one at a time (replace, do not queue) |

Over `.project-workspace` (bottom). No emoji. No undo.

---

## 8. Screen spec (craft freeze)

**Goal:** Compound feels like Amp Ship. Agent writes are visible in the thread and under Files. Save to Workspace is explicit.

| State | What you see |
|-------|----------------|
| Idle session | Compound enabled. Rail default Config (first visit). |
| Run in flight | Compound disabled; “Wait for the current run”. |
| Files empty | “No files in this session.” |
| Agent wrote a file | File card in chat + row under Files |
| Files populated | Rows; hover/focus shows ⋯ |
| Promote success | Modal closes; still Files; green toast; Workspace list has the copy |
| Promote fail | Modal stays; error in dialog |
| Compound click | User bubble with the locked prompt; run starts |

**Primary actions:** Compound; Save to Workspace (Files ⋯).  
**Out of scope visually:** review card, Changes/Drafts naming, Workspace search.

---

## 9. Accessibility

- Files / Workspace / Config: `role="tab"`; labels match product names.
- ⋯: name **Save options for {basename}**; item **Save to Workspace**.
- File card: button/link named with the file basename; Enter opens.
- Keyboard: Tab to ⋯; Escape closes menu.
- Toast is status-only.

---

## 10. Testing

### Automated

- Compound → `sendMessage` with locked prompt; no `createCompound`; no transcript `user_context`.
- Compound disabled while sending / queued / running.
- Composer identity across Compound + poll.
- Create session sends `session_files: true` (or equivalent always-on); checkbox gone.
- Stored `workspace_files: false` still gets session-file tools.
- No remaining `/workspace/tree` or `/workspace/file` client calls; types are not named `WorkspaceFile` for session scratch.
- Rail order: Files, Workspace, Config. Stored old `files` reads as Workspace. Unknown → config.
- Tool `changed_path` renders a file card; Files tree includes that path.
- ⋯ promote copy; stay on Files; Workspace list updates; toast **Saved to Workspace.**
- Promote failure: no toast; session file remains.
- Toast success tokens. `showModal` polyfill when opening promote.

### Browser vibe-pass (hard)

`make docker-dev` → `#/projects/eea7b476-e0b7-44d8-89e2-d3fafab87172` + **ivory bold**.

Check: Compound canned prompt; Files · Workspace · Config; write appears in chat + Files; save copy stays on Files; toast; Workspace then has the note. Report URL + what you saw.

---

## 11. Implementation sketch (not a plan)

1. Rename session-scratch API/types off `workspace_*`.
2. Always-on session-file tools; drop grant checkbox.
3. Toast + rail Files / Workspace + ⋯ promote.
4. In-chat file cards from `changed_path`.
5. Compound → `sendMessage(locked prompt)`.
6. Tests + vibe-pass.

---

## 12. Spec self-review

- No TBD placeholders.
- Product **Files** vs **Workspace** vs code names are in §0; old `workspace_*` for scratch is forbidden.
- Button does not write `AGENTS.md` / `memory/**`.
- Save is copy; stay on Files; toast names Workspace.
- localStorage old `files` → `workspace` so we do not strand users on an empty session Files tab.
