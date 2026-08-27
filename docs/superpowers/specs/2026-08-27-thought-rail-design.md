# Design: Thought chip + Thoughts rail

**Status:** Approved in design dialogue (2026-08-27). Awaiting user review of this file before planning.  
**Date:** 2026-08-27  
**Stack:** Existing Svelte 5 + TypeScript + Vite SPA under `web/`; existing GET messages + GET current run. **No new HTTP endpoint.**  
**Route:** `#/projects/:id` (hub + embedded `SessionChat`).

**Visual refs (structural fidelity; tokens-only is not done):**

| State | File | Intent |
|-------|------|--------|
| Rest | `docs/superpowers/specs/2026-08-27-thought-rail/1-element-thought.png` | Lightbulb + `Thought for 32s`, muted |
| Hover | `docs/superpowers/specs/2026-08-27-thought-rail/2-element-thought-hover.png` | Panel-chevron icon + same label |
| Open | `docs/superpowers/specs/2026-08-27-thought-rail/3-element-thought-clicked.png` | Thoughts header + X + tool timeline |

Adapt Grok’s **chrome** (chip, hover, header, timeline). Do **not** copy multi-agent stacks (Grok Leader / Agent 2 / Agent 3). This product is one agent.

**Related:**

- `docs/superpowers/specs/2026-08-21-project-rail-icon-chrome-design.md` — Config / Files / Workspace icon rail this feature **swaps over**
- `docs/superpowers/specs/2026-08-27-compound-ship-files-workspace-design.md` — in-chat file cards stay
- `frontend-ui-craft` — vibe-pass vs the three refs above
- Standing rules: polled SPA composer focus; never scan `$HOME`; named `@file` refs live in the repo

**Approach (approved):** Reuse the message poll. Expose `run_id` / `tool_calls_json` on the UI type; add `id` + `started_at` on `RunStatus`. Chip in the thread; click sets the **existing** right rail to Thoughts. Rejected: new `/runs/{id}/thoughts` API; client-only timing guesses; a second rail column; session-tab “Thoughts”.

---

## 1. Why

Tool-call turns persist as assistant rows with empty `content` plus `tool_calls_json`. The thread either hid them or showed a bare **ASSISTANT** label. The user still needs to see **what the agent is doing** while it works, and the tool log after.

---

## 2. Goals and non-goals

### Goals

1. New **Thought chip** in the Agent thread (refs 1–2): live seconds while the run is `queued`/`running`, frozen when complete.
2. Click chip → existing right rail becomes **Thoughts** (ref 3). Config / Files / Workspace chrome **swap out** until X.
3. Timeline is **that run only**, filled live as poll lands tool calls.
4. Empty assistant bubbles and raw tool JSON stay **out** of the thread. File cards for `changed_path` stay.
5. Spec-as-test + browser vibe-pass vs the three named refs.

### Non-goals

- Second rail column beside Config
- Session-wide tool history in one rail
- New HTTP endpoint
- Grok multi-agent grouping / web-search badges / “Show more” agent quotes
- Dark mode
- Standalone session route outside the hub (same `SessionChat` chip is fine if that route exists; rail swap is hub `ProjectRail` only)

---

## 3. Surfaces

| Surface | Behavior |
|---------|----------|
| Thought chip | Row in the Agent thread, after the user bubble, before the assistant reply (or alone while the run has no text yet). Not an assistant bubble. |
| Rest | Lightbulb + `Thought for Ns` (muted). |
| Hover | Panel-chevron icon + same label. |
| Click | Hub right rail → Thoughts for that `run_id`. |
| Thoughts rail | Header **Thoughts** + **X**. Body = timeline for that run. Empty: muted `Working…`. |
| X | Restore previous rail tab (`config` / `files` / `workspace`) and icon toolbar. |
| File cards | Unchanged in the thread. |

**When the chip exists**

- While `current` run is `queued` or `running`: always show a chip for that run (timer live).
- After the run is terminal: keep the chip if the run had **any tool call** or elapsed ≥ 1s. Instant no-tool reply → **no** chip.

One chip per run. Several runs in a session → several chips, each bound to its `run_id`.

---

## 4. Ownership

| Owner | Responsibility |
|-------|----------------|
| `SessionChat` | Group messages by `run_id`; render chips; live elapsed; emit `onOpenThoughts(runId)` on click; keep composer mounted |
| `ProjectHubPage` | Hold `thoughtsRunId`; pass a `ThoughtsView` into `ProjectRail`; remember previous rail tab for X |
| `ProjectRail` | If `thoughtsRunId` set: hide icon tabs + Config/Files/Workspace body; show Thoughts header + timeline. X clears `thoughtsRunId` |
| `app.css` | Chip + Thoughts tokens (`thought-chip`, `thoughts-rail`, timeline rows) — no one-off indigo |

`ProjectRail` does not fetch messages. Hub/SessionChat pass a view model:

```ts
type ThoughtRow = {
  id: string
  verb: string   // Read, List, Write, …
  arg: string    // truncated path or argument
  status?: 'ok' | 'error' | 'pending'
  detail?: string // short error snippet only
}

type ThoughtsView = {
  runId: string
  elapsedSec: number
  live: boolean
  rows: ThoughtRow[]
}
```

---

## 5. Data (existing endpoints)

**`GET /api/v1/sessions/{id}/messages`** already returns `domain.Message` (`run_id`, `tool_calls_json`, `tool_call_id`, `role`, `content`, `created_at`). Frontend `ChatMessage` must include:

- `run_id?: string`
- `tool_calls_json?: string` (parse on the client; OpenAI-style `{ id, name, arguments }[]`)
- `tool_call_id?: string`

No Go DTO change required for messages.

**`GET /api/v1/sessions/{id}/runs/current`** already returns `domain.AgentRun`. Frontend `RunStatus` must include `id`, `status`, `started_at` (RFC3339). No new fields on the server if those JSON keys already exist — confirm in the plan’s first test; if `started_at` is omitted when null, that is fine.

**Grouping**

- Assistant rows with `tool_calls_json` + following `role: tool` rows with matching `tool_call_id` belong to `run_id`.
- Live elapsed: `floor((now - started_at) / 1000)` while status is `queued`/`running`. Freeze when terminal.
- Past runs (no current run): elapsed = last message `created_at` − first message `created_at` for that `run_id` (minimum 1s if any tool exists). Do not invent a runs-list API.

**Verbs** (map tool `name` → short label; unknown name → the raw name, title-cased if it is `snake_case`):

| Tool name | Verb |
|-----------|------|
| `read_file` / `read_knowledge` | Read |
| `list_knowledge` / `list_dir` | List |
| `write_file` / `edit_file` | Write |
| `mkdir` | Mkdir |
| other | raw name |

**Arg:** parse `arguments` JSON for `path` first, else first string value, else `""`. Truncate ~48 chars with ellipsis.

**Status:** matching tool result content with `"error"` → `error` + short snippet; empty/pending while live and no result yet → `pending`; else `ok`. Do not dump JSON in the rail or thread.

---

## 6. Edges

- Poll failure: keep last snapshot (same as chat). Chip/rail do not blank.
- Missing `started_at`: count from that run’s first message `created_at`.
- Tool call with no name and no arg: skip the row.
- Chip click while Thoughts is open: switch timeline to **that** run (one rail).
- New send while Thoughts is open: **stay** on the old run until the new chip is clicked or X.
- X always restores the previous icon tab, never “no rail.”
- Composer ancestry stable across rail swap and poll (existing SessionChat focus gate).

---

## 7. Testing and vibe-pass

- Tool-only run → one chip, zero `data-role="other"` / `ASSISTANT` labels, final assistant text still visible.
- Instant no-tool reply → no chip.
- Running + `started_at` → label `Thought for Ns` from clock; terminal → frozen N.
- Click chip → rail header **Thoughts**; Config/Files/Workspace tabs absent; X restores previous tab; composer node identity unchanged.
- Timeline rows from `tool_calls_json` (verb + truncated path); thread has no tool JSON.
- `styles-baseline` asserts `.thought-chip` / `.thoughts-rail` tokens exist.
- Browser: orbit frost (or a seeded tool run) vs refs 1–2–3. Report URL + viewport + side-by-side. Blocked ≠ passed.

---

## 8. Out of this spec

- Fancy thinking markdown / model “reasoning” channel (we only have tool calls).
- Persisting Thoughts-open in `localStorage`.
- Keyboard shortcut beyond click + X (Escape may close Thoughts the same as X; nice-to-have, not required).
