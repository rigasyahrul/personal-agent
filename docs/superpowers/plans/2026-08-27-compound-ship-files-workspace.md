# Compound as Ship + Files + Workspace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Worker dispatch (this repo):** `pi -p --approve` in `.worktrees/<branch>` from origin/main. Review = new `pi -p --tools read,bash`. See `docs/superpowers/PROMPT-pi-master-coordinator.md`.

**Goal:** Compound sends a Ship-style canned chat prompt; agent session writes show as in-chat file cards and under rail **Files**; copy into **Workspace** (project notes) with a success toast; session scratch is not named `workspace_*`.

**Architecture:** Keep disk layout (`sessions/{id}/**` vs `source/**`). Always register session-file tools. Rename public session-file HTTP + web types. Rail tabs become Files / Workspace / Config. Compound is `sendMessage` only.

**Tech Stack:** Go + Svelte 5 + TypeScript + Vitest + Testing Library + `go test`. Node `>=22 <23`.

**Spec:** `docs/superpowers/specs/2026-08-27-compound-ship-files-workspace-design.md`  
**Lock:** `docs/superpowers/plans/2026-08-27-compound-ship-files-workspace-lock.md`

## Global Constraints

- Spec vocabulary §0 is law: product **Files** = session writes; product **Workspace** = project notes `source/**`.
- Do not name session scratch `workspace_*` in new/changed public API, web types, or rail `source`.
- Internal `publish.WorkspacePath` / `layout.SessionWorkspace` may stay this pass (map JSON `session_path` → existing field).
- Compound button must not call `createCompound` / `POST /compound`.
- Do not write `AGENTS.md` / `memory/**` / `write_knowledge`.
- Poll must not remount the composer (`SessionChat.focus.test.ts`).
- Tokens in `app.css` (`btn--*`, `toast toast--success`, `rail-icon`). No `bg-indigo-600`.
- Creates/promote stay `<dialog class="modal">`. Polyfill `showModal` in tests that open it.
- Hub startSession: create then list+open even if first send fails.
- Web tests: put Node 22 first on `PATH` before `npm --prefix web test` / `make web-test`.
- After UI: browser vibe-pass on `http://localhost:8080/#/projects/eea7b476-e0b7-44d8-89e2-d3fafab87172` (ivory bold). `make docker-dev`. Blocked ≠ passed.
- Do not merge/push unless the user asks.

## File map

| Path | Role |
|------|------|
| `internal/agent/runner.go` | Always register session-file tools |
| `internal/httpapi/chat_handlers.go` + `server.go` | `/files/tree`, `/files`; no grant 403 |
| `internal/httpapi/session_handlers.go` | Create grant `session_files: true` |
| `internal/httpapi/promote_handlers.go` | JSON `session_path` |
| `web/src/lib/api/types.ts` + `index.ts` | `SessionFile`, `sessionFilesTree`, `sessionFile` |
| `web/src/lib/session-files.ts` | move/rename from `workspace-tree.ts` |
| `web/src/lib/promote.ts` | `sessionFilesEnabled` always true; payload `session_path` |
| `web/src/lib/project-rail-prefs.ts` | tabs `files` \| `workspace` \| `config`; migrate old `files` → `workspace` |
| `web/src/components/Toast.svelte` | success toast |
| `web/src/components/ProjectRail.svelte` | Files / Workspace / Config |
| `web/src/components/sessions/SessionChat.svelte` | Compound send + file cards |
| `web/src/components/sessions/SessionFileTab.svelte` | `source: 'session-file'` |
| `web/src/routes/ProjectHubPage.svelte` | create `session_files: true`; rail wiring |
| `web/src/routes/ProjectSessionsPage.svelte` | drop checkbox; always `session_files: true` |
| `web/src/app.css` | `.toast`, `.toast--success`, `.file-card` |

## Canonical contracts

### Compound prompt (exact)

```
Compound this conversation. Extract only non-obvious learnings. Write durable notes as session files — they stay under Files until I save them to Workspace. Prefer a short standing-rule note over a diary. If nothing reusable was learned, say so and stop. Do not dump the transcript.
```

### Rail

Left: **Files**, **Workspace**, **Config**. Right: Expand, Collapse. Default tab `config`. Stored `files` (legacy) → `workspace`.

### Session-file HTTP

```
GET /api/v1/sessions/{id}/files/tree
GET /api/v1/sessions/{id}/files?path=
```

Remove handler registration for `/workspace/tree` and `/workspace/file`.

### Promote JSON

```json
{ "session_path": "note.md", "target_relative_path": "note.md", "review_mode": "none" }
```

Unknown field `workspace_path` → 400 (strict decode stays).

---

### Task 1: Always-on session file tools (Go)

**Files:**
- Modify: `internal/agent/runner.go` (grant gate around `workspaceToolDefinitions`)
- Modify: `internal/httpapi/chat_handlers.go` (`workspaceRoot` grant 403)
- Test: `internal/agent/runner_test.go` (add)
- Test: existing chat workspace HTTP tests (update 403 cases)

**Interfaces:**
- Consumes: `session.ToolGrantsJSON` may contain `"workspace_files": false`
- Produces: runner always appends `write_file` / `edit_file` / `mkdir`; HTTP tree/file no longer 403 on false grant

- [ ] **Step 1: Write the failing test**

In `runner_test.go`, copy the existing project-home runner fixture and set grants `{"workspace_files":false}`. Assert the provider request includes `write_file`.

```go
func TestRunnerAlwaysRegistersSessionFileToolsWhenGrantFalse(t *testing.T) {
	// same fixture as other runner tests; ToolGrantsJSON = `{"workspace_files":false}`
	// Start a chat run with a scripted provider
	// require hasToolName(names, "write_file")
}
```

Add/adjust HTTP test: session with `workspace_files: false` → `GET .../workspace/tree` (still old path until Task 2) returns 200, not 403.

- [ ] **Step 2: Run tests — expect FAIL**

```bash
rtk go test ./internal/agent/ -count=1 -run TestRunnerAlwaysRegistersSessionFileToolsWhenGrantFalse
```

Expected: FAIL (tools omitted).

- [ ] **Step 3: Minimal implementation**

In `runner.go`, open `layout.SessionWorkspace(...)` and append `workspaceToolDefinitions` **unconditionally** (still close the root). Stop branching on `grants.WorkspaceFiles`.

In `chat_handlers.go` `workspaceRoot`, delete the `if !grants.WorkspaceFiles { return errWorkspaceFilesDisabled }` check. Keep session lookup + `fsroot.Open`.

- [ ] **Step 4: Run tests — expect PASS**

```bash
rtk go test ./internal/agent/ ./internal/httpapi/ -count=1
```

- [ ] **Step 5: Commit**

```bash
rtk git add internal/agent/runner.go internal/agent/runner_test.go internal/httpapi/chat_handlers.go internal/httpapi/*_test.go
rtk git commit -m "fix(agent): always enable session file tools"
```

---

### Task 2: Rename session-file HTTP + web client

**Files:**
- Modify: `internal/httpapi/server.go` routes
- Modify: `internal/httpapi/chat_handlers.go` (handler names/error codes may stay internally this task if tests target new paths)
- Modify: `web/src/lib/api/types.ts`, `web/src/lib/api/index.ts`, `web/src/lib/api/*.test.ts`
- Create: `web/src/lib/session-files.ts` (move `workspace-tree.ts` helpers; types `SessionFileEntry`)
- Modify: all web imports of `workspaceTree` / `WorkspaceFile` / `workspaceEnabled`

**Interfaces:**
- Produces:

```ts
export interface SessionFile {
  path: string
  kind: 'file' | string
  content?: string
}
export interface SessionFileEntry {
  path: string
  kind: 'file' | 'directory' | string
}
sessionFilesTree(sessionId: string): Promise<{ entries: SessionFileEntry[] }>
sessionFile(sessionId: string, path: string): Promise<SessionFile>
```

- [ ] **Step 1: Failing tests**

Go: request `GET /api/v1/sessions/{id}/files/tree` → 200. Old `/workspace/tree` → 404.

Web `compound`/`api` tests: `sessionFilesTree` hits `/files/tree`.

- [ ] **Step 2: Run — expect FAIL** (new path 404)

```bash
rtk go test ./internal/httpapi/ -count=1 -run FilesTree
```

- [ ] **Step 3: Implement**

Register new routes; remove old workspace route registration. Point web client at new paths. Rename `workspace-tree.ts` → `session-files.ts` (`changedPathsFromMessages` unchanged). Update `SessionFileTab` to load via `sessionFile` when `source === 'session-file'` (accept old `'workspace'` as alias **only** inside the tab for one commit if needed, then delete the alias in the same task).

`workspaceEnabled()` → `sessionFilesEnabled()` always `true` (spec: no grant). Keep function so callers compile.

- [ ] **Step 4: Run web + Go tests**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
rtk go test ./internal/httpapi/ -count=1
npm --prefix web test -- --run session-files api
```

- [ ] **Step 5: Commit**

```bash
rtk git commit -m "refactor(api): session files routes and client names"
```

---

### Task 3: Promote JSON `session_path`

**Files:**
- Modify: `internal/httpapi/promote_handlers.go` (`promoteRequest.SessionPath` `json:"session_path"`; map to `PublishInput.WorkspacePath`)
- Modify: `web/src/lib/api/types.ts` `PromotePayload.session_path`
- Modify: `web/src/lib/promote.ts` + tests
- Modify: promote HTTP tests

**Interfaces:**
- Consumes: Task 2 session file paths
- Produces: POST promote body uses `session_path` only

- [ ] **Step 1: Failing test** — POST with `session_path` → 200/valid; POST with only `workspace_path` → 400 `invalid_request`

- [ ] **Step 2: Run — expect FAIL**

```bash
rtk go test ./internal/httpapi/ -count=1 -run Promote
```

- [ ] **Step 3: Implement** decode `session_path`; fingerprint can keep internal `workspace_path` key **or** switch to `session_path` in the fingerprint struct JSON (prefer `session_path` in the fingerprint so keys stay stable going forward — update tests that hash the fingerprint).

- [ ] **Step 4: Tests PASS**

- [ ] **Step 5: Commit** `feat(api): promote session_path field`

---

### Task 4: Toast

**Files:**
- Create: `web/src/components/Toast.svelte`
- Create: `web/src/components/Toast.test.ts`
- Modify: `web/src/app.css`
- Modify: `web/src/styles-baseline.test.ts` (assert `.toast`, `.toast--success`)

**Interfaces:**
- Produces:

```svelte
<Toast open={boolean} tone="success">Saved to Workspace.</Toast>
```

`role="status"` `aria-live="polite"`. CSS:

```css
.toast { position: absolute; bottom: 16px; left: 50%; transform: translateX(-50%); z-index: 20; padding: 10px 12px; border-radius: var(--radius-sm); font-size: 13px; }
.toast--success { background: var(--success-soft); color: var(--success); border: 1px solid #bbf7d0; }
```

- [ ] **Step 1: Failing test** — render with `open` → text + `.toast--success`; `open={false}` not visible

- [ ] **Step 2: `npm --prefix web test -- --run Toast` expect FAIL**

- [ ] **Step 3: Implement component + CSS**

- [ ] **Step 4: Tests PASS** including styles-baseline

- [ ] **Step 5: Commit** `feat(web): success toast`

---

### Task 5: Rail Files · Workspace · Config

**Files:**
- Modify: `web/src/lib/project-rail-prefs.ts` + tests
- Modify: `web/src/components/ProjectRail.svelte` + `ProjectRail.test.ts`
- Modify: `web/src/routes/ProjectHubPage.svelte` (tab wiring, `source: 'session-file'`)
- Modify: `web/src/components/sessions/SessionFileTab.svelte` (`session-file` vs `project-note`)

**Interfaces:**
- `ProjectRailTab = 'files' | 'workspace' | 'config'`
- `readProjectRailTab`: `'files'` stored **before this change** meant project notes → return `'workspace'`. After this change, writing `'files'` means session Files. Distinguish with a new storage value? **Rule:** bump key to `pa.projectRail.tab.v2` values `files|workspace|config`. Old key `pa.projectRail.tab=files|config` maps: `config`→`config`, `files`→`workspace`, missing→`config`. New writes go to `v2` only.

- [ ] **Step 1: Failing prefs tests**

```ts
expect(readProjectRailTab(storageWith('pa.projectRail.tab', 'files'))).toBe('workspace')
expect(readProjectRailTab(storageWith('pa.projectRail.tab.v2', 'files'))).toBe('files')
```

Rail test: left icons order Files, Workspace, Config. Files empty copy: `No files in this session.` Workspace empty: keep project-notes empty copy.

- [ ] **Step 2: Run — FAIL**

```bash
npm --prefix web test -- --run project-rail
```

- [ ] **Step 3: Implement icon + panels.** Files panel = session tree via `sessionFilesTree` when a session is open. Workspace panel = `listProjectNotes` only (no session-file subsection).

Need a **Files** rail icon. Reuse folder icon for Files; give Workspace a distinct existing glyph (e.g. current files icon stays on Files; Config unchanged). If only two glyphs exist, add one path in `rail-icons.ts` named `workspace`.

- [ ] **Step 4: Tests PASS** (`ProjectRail`, `ProjectHubPage`, `SessionFileTab`)

- [ ] **Step 5: Commit** `feat(web): Files and Workspace rail tabs`

---

### Task 6: Save to Workspace from Files ⋯

**Files:**
- Modify: `ProjectRail.svelte` (⋯ menu)
- Modify: `ProjectHubPage.svelte` (promote dialog + toast + refresh notes)
- Modify: tests

**Interfaces:**
- Menu item name: **Save to Workspace**
- On success: stay on Files; `listProjectNotes` refetch; `<Toast open tone="success">Saved to Workspace.</Toast>` ~3s
- Copy, not move (do not delete session file)

- [ ] **Step 1: Failing test** — click ⋯ → Save to Workspace → `promoteSession` called with `session_path`; tab still Files; toast text present; notes list includes new path (mock)

Polyfill `showModal`/`close` in `beforeEach`.

- [ ] **Step 2: Run — FAIL**

- [ ] **Step 3: Implement** overflow + existing `PromoteDialog`. Parent handles success: toast timer 3000ms; refresh notes.

- [ ] **Step 4: Tests PASS**

- [ ] **Step 5: Commit** `feat(web): copy session file to Workspace`

---

### Task 7: In-chat file cards

**Files:**
- Modify: `SessionChat.svelte` message loop
- Modify: `SessionChat.test.ts`, `SessionChat.focus.test.ts`
- Modify: `app.css` `.file-card`

**Interfaces:**
- For each message with `role === 'tool'` and a `changed_path` (field or JSON content via `changedPathsFromMessages` / per-message parse), render a button `.file-card` whose name is the basename; click → open session file tab (`source: 'session-file'`) and do not remount composer.
- Do not render raw tool JSON as `role=other` for those messages.
- After new `changed_path`, invalidate Files tree (call the same load Files uses, or bump a `filesEpoch` passed into the rail).

- [ ] **Step 1: Failing test**

```ts
messages include { role: 'tool', changed_path: 'dummy.md', content: '{"changed_path":"dummy.md"}' }
expect(screen.getByRole('button', { name: /dummy.md/i })).toBeInTheDocument()
```

Focus test: click card does not replace composer node.

- [ ] **Step 2: Run — FAIL**

- [ ] **Step 3: Implement** cards + skip dumping tool JSON. `mkdir` without a file path: no card.

- [ ] **Step 4: Tests PASS**

- [ ] **Step 5: Commit** `feat(web): show session file cards in chat`

---

### Task 8: Compound = Ship send

**Files:**
- Modify: `web/src/components/sessions/SessionChat.svelte` (`startCompound`)
- Modify: `SessionChat.test.ts` compound describe
- Modify: `ProjectHubPage.svelte` `tool_grants: { session_files: true }`
- Modify: `ProjectSessionsPage.svelte` remove checkbox; always `session_files: true`
- Modify: matching tests

**Interfaces:**
- `sendMessage(id, { content: COMPOUND_PROMPT, request_key })`
- Do not call `createCompound`
- Do not set `compoundProposal` from this button
- Export prompt constant from `web/src/lib/compoundPrompt.ts` so tests import the same string

- [ ] **Step 1: Failing tests**

```ts
await fireEvent.click(screen.getByRole('button', { name: 'Compound' }))
expect(api.sendMessage).toHaveBeenCalledWith('s1', {
  content: COMPOUND_PROMPT,
  request_key: 'rk-compound',
})
expect(api.createCompound).not.toHaveBeenCalled()
expect(screen.queryByText('Compound review')).not.toBeInTheDocument()
```

Hub test: create body `session_files: true` (or whatever create JSON the backend accepts — if backend still only knows `workspace_files`, send **both** `session_files: true` and keep decoding: treat either key as true, default true). **Backend create:** accept unknown extra keys? Current decode is strict. So create JSON stays `{ workspace_files: true }` **or** extend `session_handlers` to accept `session_files` and persist `{"session_files":true}`. Prefer persist `{"session_files":true}` and treat missing/false `workspace_files` as on (Task 1 already ignores the flag).

Sessions page: no checkbox named `/workspace files/i`.

- [ ] **Step 2: Run — FAIL** (still calls createCompound)

- [ ] **Step 3: Replace `startCompound` with send of `COMPOUND_PROMPT`. Remove review-card wiring from the button (component may remain unused). Create sessions with `session_files: true` if the handler allows it; otherwise `workspace_files: true` plus Task 1 always-on is enough — **do both**: handler default `session_files: true` / `workspace_files` ignored.

- [ ] **Step 4: `npm --prefix web test` + `rtk go test ./internal/httpapi/ -count=1` PASS. Focus test PASS.**

- [ ] **Step 5: Commit** `feat(web): Compound sends canned Ship prompt`

---

### Task 9: Browser vibe-pass (hard)

**Files:** none required unless bugs found

- [ ] **Step 1:** `make docker-dev` (or confirm `:8080` is this tree)
- [ ] **Step 2:** Open `http://localhost:8080/#/projects/eea7b476-e0b7-44d8-89e2-d3fafab87172` → **ivory bold**
- [ ] **Step 3:** Confirm rail **Files · Workspace · Config**. Compound sends the canned prompt (not a transcript). If the model writes a file: card in chat + Files row. ⋯ save → toast **Saved to Workspace.** Stay on Files. Workspace list has the copy.
- [ ] **Step 4:** Report URL + viewport + what you saw. Blocked ≠ passed.
- [ ] **Step 5:** Commit only if you had to fix UI: `fix(web): Files/Workspace vibe-pass`

---

## Spec coverage

| Spec | Task |
|------|------|
| Compound canned send, no POST /compound | 8 |
| Always-on session files (ivory bold) | 1, 8 |
| Remove grant checkbox | 8 |
| Rename off `workspace_*` public | 2, 3 |
| Files · Workspace · Config + localStorage migrate | 5 |
| ⋯ copy + toast + stay on Files | 4, 6 |
| In-chat file cards + Files sync | 7 |
| Vibe-pass ivory bold | 9 |
| Leave compound API in repo | (no delete task) |

## Placeholder scan

None. Prompt, routes, toast copy, tab ids, and test names are exact.
