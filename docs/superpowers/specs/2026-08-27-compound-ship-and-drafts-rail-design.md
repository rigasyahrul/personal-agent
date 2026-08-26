# Design: Compound as Ship + Drafts rail

**Status:** Draft — awaiting user review  
**Date:** 2026-08-27  
**Stack:** Existing Svelte 5 + TypeScript + Vite SPA under `web/`; Go session/workspace/promote APIs unchanged except hub session grant default.  
**Route:** `#/projects/:id` (hub + embedded session). Compound control lives on `SessionChat` wherever that component mounts.

**Related:**

- `docs/superpowers/specs/2026-08-21-project-rail-icon-chrome-design.md` (rail chrome; this spec **adds Drafts** and **reorders** left icons)
- `docs/superpowers/specs/2026-08-22-obsidian-memory-knowledge-design.md` (human-gated compound API — **not** used by the button after this change)
- `frontend-ui-craft` — screen spec below is the craft freeze; browser vibe-pass required on UI
- Standing rules: polled SPA composer focus; hub startSession not create∧send atomic; rail Files ≠ workspace-only; creates use modals (promote stays a `<dialog>`)

**Approach (approved in design dialogue):** Amp **Ship**-style Compound (canned chat send) + new rail tab **Drafts** for session-written files + copy-to-Files via existing promote + success toast. Rejected: transcript-dump `user_context`; silent review-card generate as the button UX; Amp “Changes” naming; move-on-save; writing `AGENTS.md` / `memory/**` from this button.

---

## 1. Why

Clicking **Compound** today does not compound. It posts the last six messages as `user_context`, and `Runs.Admit` inserts that dump as a normal user chat row. The model answers like ChatGPT. The user expected Amp **Ship**: one click sends a **fixed instruction**, then the agent works in the thread.

A normal chat turn cannot write project-durable `AGENTS.md` / `memory/**` (`write_knowledge` is not in slice 1). Session `write_file` lands in `sessions/{id}/`, not the project. Hub sessions currently start with `workspace_files: false`, so the agent cannot create those files at all.

Amp **Changes** is the wrong metaphor (git diffs of files that already *are* the project). Session writes are **drafts** until the user copies them into project notes.

---

## 2. Goals and non-goals

### Goals

1. **Compound** sends one canned user message via `sendMessage` (same admission as Send). No transcript dump. No `POST /compound` from this button. No review card on this path.
2. Hub-created sessions grant **`workspace_files: true`** so the agent can write session files.
3. Right rail left-cluster order: **Drafts · Files · Config**. Right cluster unchanged (Expand · Collapse).
4. **Drafts** lists this session’s workspace files only. Click opens the file. Hover **⋯** → **Save to project notes?** → existing promote modal. Save is a **copy**; the draft stays. After success, stay on Drafts and show toast **“Saved to Files.”** Files list refreshes immediately so the copy is there when the user opens Files.
5. Small reusable **Toast** (success first) using existing `--success` / `--success-soft` tokens.

### Non-goals

- Deleting or rewriting the compound proposal API, publisher, or `CompoundReviewCard` (leave unused by this button)
- Agent writes to `AGENTS.md` / `memory/**` / `write_knowledge`
- Auto-compound, synthesize-memory, or a compound inbox page
- Files tree/search redesign (still deferred)
- Toast queue, undo, error/warn toast variants
- Standalone session route getting a Drafts rail (hub rail only)
- Dark mode
- Changing promote destination (`source/**` project notes, not memory)

---

## 3. Surfaces

| Surface | Owner | Change |
|---------|--------|--------|
| Compound button | `SessionChat.svelte` | `onclick` → `sendMessage` with locked prompt; drop `startCompound` / `createCompound` / review-card wiring from the button |
| Hub create session | `ProjectHubPage.svelte` | `tool_grants.workspace_files: true` (new sessions only; existing rows stay as stored). `ProjectSessionsPage` checkbox unchanged. |
| Rail tabs | `project-rail-prefs.ts`, `ProjectRail.svelte` | Tab type adds `drafts`; left order Drafts, Files, Config; new panel |
| Drafts rows | `ProjectRail.svelte` | Workspace tree for the open session; ⋯ menu; promote |
| Promote | existing `PromoteDialog` | Unchanged contract; on success: refresh Files data, keep Drafts selected, show toast |
| Toast | new `web/src/components/Toast.svelte` + tokens in `app.css` | Success tone only |
| Files panel | `ProjectRail.svelte` | Project notes only (no Workspace subsection — that list moves to Drafts) |

Default hub enter (no saved tab): still **Config**. Invalid stored tab → `config`. Persist `pa.projectRail.tab` including `drafts`.

---

## 4. Compound (Ship-style)

### Click

Same gates as today: disabled while send-in-flight or run `queued`/`running`. Uses a new `request_key`. Does **not** call `api.createCompound`.

Body:

```text
Compound this conversation. Extract only non-obvious learnings. Write durable notes as session files — they stay drafts until I save them to Files. Prefer a short standing-rule note over a diary. If nothing reusable was learned, say so and stop. Do not dump the transcript.
```

That string is the user message. It appears in the thread as the user. The agent run is a normal chat run (tools per session grants). With hub `workspace_files: true`, `write_file` / `edit_file` / `mkdir` are available.

### After send

- Composer unchanged (do not put the prompt in the draft field).
- Existing poller shows the new user row + run.
- If send fails: same chat error alert as Send; prompt is not left as a half-typed draft (it was never in the composer).
- Composer focus / form identity: existing SessionChat focus rules still apply (do not remount the form).

### Old pipeline

`POST /api/v1/sessions/{id}/compound`, `StartCompound`, review card, decide/publish stay in the repo. This spec does not call them from the UI. Do not spend this pass deleting them.

---

## 5. Drafts vs Files

| Tab | Data | Writers |
|-----|------|---------|
| **Drafts** | `GET` workspace tree for the **open session** | Agent workspace tools; user does not create drafts from the rail in this pass |
| **Files** | Project notes (`listProjectNotes`) | Human create (existing) + **copy** from Drafts via promote |

No open session, or workspace grant false: Drafts shows **“No drafts in this session.”** Do not fall back to another session’s files.

Click a draft file → existing hub `onOpenFile` with `source: 'workspace'`. Directories are not openable.

### Overflow (⋯)

- Visible on hover/focus of a **file** row (not directories).
- Menu item: **Save to project notes?**
- Opens the existing promote `<dialog>` (not an inline form).
- Confirm → promote API (existing). **Copy:** session file remains; a new `source/**` note is created.
- Success: stay on **Drafts**; refresh project-notes list in the background; toast **“Saved to Files.”**
- Failure: dialog error stays; no toast; draft unchanged; Files list unchanged.

Deleting the session later does not delete the saved project note.

---

## 6. Toast

New presentational component, not an npm library.

| Token | Value |
|-------|--------|
| Background | `var(--success-soft)` (`#f0fdf4`) |
| Text | `var(--success)` (`#15803d`) |
| Border | same family as `.health-pill[data-tone='ok']` (`#bbf7d0`) |
| Radius | `var(--radius-sm)` |
| Copy | `Saved to Files.` |
| A11y | `role="status"` `aria-live="polite"` |
| Lifetime | ~3s then dismiss; one toast at a time (replace, do not queue) |

Place over the project workspace (bottom-center or bottom-right of `.project-workspace`). No emoji. No undo. No error/warn variants in this pass.

---

## 7. Screen spec (craft freeze)

**Goal:** Compound feels like Amp Ship; session writes are obvious under Drafts; save-to-project is explicit and confirmed.

| State | What you see |
|-------|----------------|
| Idle session | Compound enabled. Rail default Config (first visit). |
| Run in flight | Compound disabled; title “Wait for the current run”. |
| Drafts empty | “No drafts in this session.” |
| Drafts populated | File rows; hover/focus shows ⋯ |
| Promote open | Existing modal |
| Promote success | Modal closes; still Drafts; green toast “Saved to Files.”; Files list contains the new note |
| Promote fail | Modal stays; error in dialog |
| Compound click | User bubble with the locked prompt; run starts |

**Primary actions:** Compound (chat header); Save to project notes (Drafts ⋯).  
**Out of scope visually:** review card, Changes naming, Files search.

---

## 8. Accessibility

- Drafts / Files / Config: `role="tab"` / `tablist` / `tabpanel` as today; labels **Drafts**, **Files**, **Config**.
- ⋯ control: `type="button"`, accessible name **Draft actions** or **Save options for {basename}**; menu item name **Save to project notes**.
- Keyboard: Tab to ⋯, Enter/Space opens menu; Escape closes.
- Toast is status-only (not a focus trap).

---

## 9. Testing

### Automated

- Compound click calls `sendMessage` once with the locked prompt + `request_key`; does **not** call `createCompound`; does not pass a transcript `user_context`.
- Compound disabled while sending / run queued / running (existing gates).
- Composer node identity unchanged across Compound send + poll (`SessionChat.focus` gate).
- Hub `createProjectSession` (and hub startSession body) sends `workspace_files: true`.
- Rail left icon order: Drafts, Files, Config. Default tab Config. `drafts` persists and reads back; unknown tab → config.
- Drafts lists workspace files only when a session is open and grant is on; empty copy otherwise.
- Files panel does not render the old Workspace subsection.
- ⋯ → promote; on success Files list includes the new note **without** switching tab; toast text present.
- Promote failure: no toast; Drafts still has the file.
- Toast uses success tokens / required class (baseline or component test).
- `showModal` polyfill in any test that opens promote (existing Modal rule).

### Browser vibe-pass (hard)

`make docker-dev` → `#/projects/{id}` with an open session.

Check: Compound sends the canned prompt (not a transcript); Drafts / Files / Config order; save-to-notes copy stays on Drafts; toast color + copy; Files then shows the note. Report URL + what you saw. Blocked ≠ passed.

---

## 10. Implementation sketch (not a plan)

1. Toast component + success tokens.
2. Rail tab type + icon order + Drafts panel; move workspace list out of Files.
3. Hub `workspace_files: true`; ⋯ → promote; refresh notes + toast.
4. Replace Compound `startCompound` with `sendMessage(locked prompt)`.
5. Tests + vibe-pass.

---

## 11. Spec self-review

- No TBD / TODO placeholders.
- Button path and durable-memory API do not contradict: button does not write `AGENTS.md` / `memory/**`.
- Save is copy; Files updates immediately; rail stays on Drafts — consistent.
- Scope is one implementation plan (UI + hub grant). Backend compound stack is explicitly out of delete-scope.
