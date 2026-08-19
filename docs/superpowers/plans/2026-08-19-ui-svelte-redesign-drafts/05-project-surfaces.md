# Project Surfaces Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild project hub, notes, sessions, workspace, and promotion as polished Svelte surfaces without changing API behavior or losing session composer focus.

**Architecture:** Project routes share one loaded `Project` and breadcrumb context. Session list and chat are separate components; a dedicated polling controller serializes/coalesces requests and mutates message/run state without changing the keyed chat shell or composer node. Workspace and promote are children of the stable chat shell, and promotion operations persist per session.

**Tech Stack:** Svelte 5, TypeScript, Vite, Tailwind CSS, Vitest, Testing Library

## Global Constraints

- Session list/create API is only `GET/POST /api/v1/projects/{id}/sessions`.
- Poll messages/status in place; never replace or remount a focused composer.
- Port the behavioral coverage from `web/js/pages/sessions.js` and `web/js/pages/sessions.test.js` before deleting legacy code.
- Promotion POST must send `Idempotency-Key`; unchanged retry reuses the key and changed payload gets a new key.
- Preserve exact contract strings: `Save to source`, `target_relative_path`, `review_mode`, `operation_id`, `Promoting…`, `Promote failed — Retry`, `Note saved; cards pending…`, `Cards failed — Retry cards`, `Ready`.
- Keep the same notes, message, run, workspace, operation, and retry APIs; no backend expansion.

---

### Task 40: Build the Project Hub

**Files:**
- Create: `web/src/routes/ProjectHubPage.svelte`
- Create: `web/src/routes/ProjectHubPage.test.ts`
- Modify: `web/src/lib/router.ts`
- Modify: `web/src/App.svelte`

**Interfaces:**
- Consumes: `api.getProject(projectId)`, `Breadcrumbs`, and route shell context.
- Produces: `#/projects/:id` hub with Notes, Sessions, and Review links.

- [ ] **Step 1: Write the failing hub test**

```ts
it('renders project metrics and links without a second catalog', async () => {
  api.getProject.mockResolvedValue({ ...project, note_count: 3, session_count: 2, due_count: 1 })
  render(ProjectHubPage, { projectId: 'p1' })
  expect(await screen.findByRole('heading', { name: project.name })).toBeVisible()
  expect(screen.getByRole('link', { name: /notes/i })).toHaveAttribute('href', '#/projects/p1/notes')
  expect(screen.getByRole('link', { name: /sessions/i })).toHaveAttribute('href', '#/projects/p1/sessions')
  expect(screen.getByRole('link', { name: /review/i })).toHaveAttribute('href', '#/projects/p1/review')
})
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/routes/ProjectHubPage.test.ts`
Expected: FAIL because the hub is absent.

- [ ] **Step 3: Implement the hub**

Render breadcrumbs, title, optional vault badge, count cards, three action cards, a skeleton, and retryable inline hard-load error. Route links must encode the project ID.

- [ ] **Step 4: Verify green**

Run: `rtk npm run test -- --run web/src/routes/ProjectHubPage.test.ts web/src/lib/router.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/routes/ProjectHubPage.svelte web/src/routes/ProjectHubPage.test.ts web/src/lib/router.ts web/src/App.svelte
rtk git commit -m "feat(web): add project hub"
```

### Task 41: Restyle Notes as a Responsive Two-Pane Surface

**Files:**
- Create: `web/src/routes/NotesPage.svelte`
- Create: `web/src/routes/NotesPage.test.ts`
- Create: `web/src/components/notes/NoteTree.svelte`
- Create: `web/src/components/notes/NoteReader.svelte`
- Modify: `web/src/lib/api/index.ts`
- Modify: `web/src/lib/api/types.ts`

**Interfaces:**
- Consumes existing note tree/detail endpoints through `listProjectNotes(projectId)` and `getProjectNote(projectId, noteId)` matching the legacy client paths.
- Produces: selected note URL `#/projects/:id/notes/:noteId`; `NoteTree` emits `select(noteId: string)`.

- [ ] **Step 1: Write failing page tests**

```ts
it('shows tree and selected note in two panes', async () => {
  render(NotesPage, { projectId: 'p1', noteId: 'n1' })
  expect(await screen.findByRole('tree')).toBeVisible()
  expect(await screen.findByRole('article')).toHaveTextContent('Rendered note')
})

it('distinguishes an empty tree from no selection', async () => {
  api.listProjectNotes.mockResolvedValueOnce([])
  const { rerender } = render(NotesPage, { projectId: 'p1' })
  expect(await screen.findByText(/no notes yet/i)).toBeVisible()
  api.listProjectNotes.mockResolvedValueOnce([note])
  await rerender({ projectId: 'p1' })
  expect(await screen.findByText(/select a note/i)).toBeVisible()
})
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/routes/NotesPage.test.ts`
Expected: FAIL because Svelte note components are absent.

- [ ] **Step 3: Implement the same API behavior with new layout**

Use a fixed-width tree and flexible reader above 768px; stack them on mobile. Preserve server-provided safe rendering semantics from the old page (plain text remains text; only render HTML if the existing endpoint explicitly returns rendered HTML). Show tree/reader skeletons independently and keep tree visible if detail loading fails.

- [ ] **Step 4: Verify green**

Run: `rtk npm run test -- --run web/src/routes/NotesPage.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/routes/NotesPage.svelte web/src/routes/NotesPage.test.ts web/src/components/notes/NoteTree.svelte web/src/components/notes/NoteReader.svelte web/src/lib/api/index.ts web/src/lib/api/types.ts
rtk git commit -m "feat(web): rebuild notes as two-pane reader"
```

### Task 42 (40a): Build Project Session List and Creation

**Files:**
- Create: `web/src/routes/ProjectSessionsPage.svelte`
- Create: `web/src/routes/ProjectSessionsPage.test.ts`
- Create: `web/src/components/sessions/SessionList.svelte`
- Create: `web/src/components/sessions/SessionList.test.ts`
- Modify: `web/src/lib/api/index.ts`
- Modify: `web/src/lib/api/types.ts`

**Interfaces:**
- Produces: `listProjectSessions(projectId)`, `createProjectSession(projectId, input)`, `SessionList` event `open(session: Session)`.
- Request: `{ home: 'project', title, provider, model_id, model_parameters: {}, tool_grants: { workspace_files: boolean } }`.

- [ ] **Step 1: Write failing list/create tests**

```ts
it('lists and creates only through the project endpoint', async () => {
  render(ProjectSessionsPage, { projectId: 'p1' })
  await user.type(await screen.findByLabelText('Title'), 'Plan')
  await user.selectOptions(screen.getByLabelText('Model'), 'openai\u0000gpt')
  await user.click(screen.getByRole('button', { name: 'New session' }))
  expect(api.createProjectSession).toHaveBeenCalledWith('p1', {
    home: 'project', title: 'Plan', provider: 'openai', model_id: 'gpt',
    model_parameters: {}, tool_grants: { workspace_files: false },
  })
})

it('shows setup guidance rather than a form when models are empty', async () => {
  api.listModels.mockResolvedValue({ models: [] })
  render(ProjectSessionsPage, { projectId: 'p1' })
  expect(await screen.findByText(/configure a model/i)).toBeVisible()
  expect(screen.queryByRole('button', { name: 'New session' })).not.toBeInTheDocument()
})
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/components/sessions/SessionList.test.ts web/src/routes/ProjectSessionsPage.test.ts`
Expected: FAIL because list/create components are missing.

- [ ] **Step 3: Implement list, skeleton, empty/setup states, and create form**

Load models and project sessions concurrently. Keep creation errors inline, disable duplicate submit, expose workspace-files as an explicit grant checkbox, and open the created session only after a successful response.

- [ ] **Step 4: Verify green**

Run: `rtk npm run test -- --run web/src/components/sessions/SessionList.test.ts web/src/routes/ProjectSessionsPage.test.ts`
Expected: PASS with requests only under `/api/v1/projects/p1/sessions`.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/routes/ProjectSessionsPage.svelte web/src/routes/ProjectSessionsPage.test.ts web/src/components/sessions/SessionList.svelte web/src/components/sessions/SessionList.test.ts web/src/lib/api/index.ts web/src/lib/api/types.ts
rtk git commit -m "feat(web): add project session list and creation"
```

### Task 43 (40c): Lock the Composer Focus Invariant with a Failing Poll Test

**Files:**
- Create: `web/src/components/sessions/session-poller.ts`
- Create: `web/src/components/sessions/session-poller.test.ts`
- Create: `web/src/components/sessions/SessionChat.focus.test.ts`

**Interfaces:**
- Produces test contract for `SessionChat` from Task 44 and `createSessionPoller(load, apply, intervalMs)` with `start()`, `poll()`, and `stop()`.
- The test requires the exact same `<textarea>` DOM node, value, focus, and selection after unchanged and changed polls.

- [ ] **Step 1: Write the regression test before chat implementation**

```ts
it('patches messages and run state without replacing the focused composer', async () => {
  render(SessionChat, { session, projectId: 'p1', pollInterval: 60_000 })
  const composer = await screen.findByLabelText('Message') as HTMLTextAreaElement
  await user.click(composer)
  await user.type(composer, 'typing here')
  composer.setSelectionRange(6, 11)

  await pollHarness.resolve({ messages: [...initialMessages, reply], run: { status: 'running' } })

  expect(screen.getByLabelText('Message')).toBe(composer)
  expect(document.activeElement).toBe(composer)
  expect(composer.value).toBe('typing here')
  expect([composer.selectionStart, composer.selectionEnd]).toEqual([6, 11])
  expect(screen.getByText(reply.content)).toBeVisible()
  expect(screen.getByRole('status')).toHaveTextContent('Run: running')
})
```

- [ ] **Step 2: Run and preserve the intentional failure**

Run: `rtk npm run test -- --run web/src/components/sessions/SessionChat.focus.test.ts`
Expected: FAIL because `SessionChat.svelte` does not exist. Do not weaken identity/focus/selection assertions.

- [ ] **Step 3: Implement only the serialized/coalesced polling controller**

```ts
export function createSessionPoller<T>(load: () => Promise<T>, apply: (value: T) => void, intervalMs = 1500) {
  let active = false, queued = false, timer: ReturnType<typeof setInterval> | undefined
  async function poll() {
    queued = true
    if (active) return
    active = true
    try { while (queued) { queued = false; apply(await load()) } }
    finally { active = false }
  }
  return { poll, start: () => { void poll(); timer ??= setInterval(() => void poll(), intervalMs) }, stop: () => { if (timer) clearInterval(timer); timer = undefined; queued = false } }
}
```

- [ ] **Step 4: Test controller separately while focus test remains red**

Run: `rtk npm run test -- --run web/src/components/sessions/session-poller.test.ts web/src/components/sessions/SessionChat.focus.test.ts`
Expected: poller tests PASS and focus test FAIL only because chat is not implemented.

- [ ] **Step 5: Commit the red regression contract**

```bash
rtk git add web/src/components/sessions/session-poller.ts web/src/components/sessions/session-poller.test.ts web/src/components/sessions/SessionChat.focus.test.ts
rtk git commit -m "test(web): lock session composer focus during polls"
```

### Task 44 (40b): Implement the Stable Session Chat Shell

**Files:**
- Create: `web/src/components/sessions/SessionChat.svelte`
- Create: `web/src/components/sessions/SessionChat.test.ts`
- Modify: `web/src/components/sessions/SessionChat.focus.test.ts`
- Modify: `web/src/routes/ProjectSessionsPage.svelte`
- Modify: `web/src/lib/api/index.ts`

**Interfaces:**
- Consumes: `session`, `projectId`, `createSessionPoller`; APIs `listMessages`, `currentRun`, `sendMessage`.
- Produces stable chat shell with message list, run status, inline alert, sticky composer, and `close` callback.

- [ ] **Step 1: Add failing send/race/error tests**

Port cases from `web/js/pages/sessions.test.js`: one stable `request_key`, duplicate-submit suppression, failed-send draft retention, cached-history retention on poll failure, stale old-session result rejection, one timer across overlapping opens, and timer cleanup on destroy.

```ts
expect(api.sendMessage).toHaveBeenCalledTimes(1)
expect(api.sendMessage).toHaveBeenCalledWith(session.id, { content: 'draft', request_key: 'stable-key' })
expect(composer).toHaveValue('draft') // failed send
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/components/sessions/SessionChat.test.ts web/src/components/sessions/SessionChat.focus.test.ts`
Expected: FAIL against the missing chat.

- [ ] **Step 3: Implement without conditional/remounting composer ancestry**

Keep one keyed chat shell per `session.id`; messages and run are rune state updates. Do not key the message list/run status together with the form, do not conditionally recreate the form during polling, and clear the textarea only after this session's successful send. Guard every async apply with a generation/session ID.

```svelte
<section class="session-chat">
  <ol class="messages">{#each messages as message (message.sequence)}<MessageRow {message} />{/each}</ol>
  <p class="run-status" role="status" aria-live="polite">{run ? `Run: ${run.status}` : 'Idle'}</p>
  <InlineAlert message={error} />
  <form class="sticky bottom-0" onsubmit={send}>
    <label>Message<textarea bind:this={composer} bind:value={draft} required></textarea></label>
    <Button disabled={sending || !!run}>Send</Button>
  </form>
</section>
```

- [ ] **Step 4: Verify all chat behavior including node identity**

Run: `rtk npm run test -- --run web/src/components/sessions/SessionChat.test.ts web/src/components/sessions/SessionChat.focus.test.ts`
Expected: PASS; the focus test confirms the textarea object is unchanged after a changed poll.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/components/sessions/SessionChat.svelte web/src/components/sessions/SessionChat.test.ts web/src/components/sessions/SessionChat.focus.test.ts web/src/routes/ProjectSessionsPage.svelte web/src/lib/api/index.ts
rtk git commit -m "feat(web): add focus-safe session chat"
```

### Task 45: Add the Session Workspace Panel

**Files:**
- Create: `web/src/components/sessions/WorkspacePanel.svelte`
- Create: `web/src/components/sessions/WorkspacePanel.test.ts`
- Modify: `web/src/components/sessions/SessionChat.svelte`
- Modify: `web/src/lib/api/index.ts`

**Interfaces:**
- Consumes: session persisted `tool_grants`/`tool_grants_json`, `workspaceTree(sessionId)`, `workspaceFile(sessionId, path)`.
- Produces: selected promotable file callback `onpromote(file: WorkspaceFile)`; panel appears only when `workspace_files === true`.

- [ ] **Step 1: Write failing grant/tree/refresh tests**

```ts
it.each([{ grants: '{bad', visible: false }, { grants: '{"workspace_files":false}', visible: false }, { grants: '{"workspace_files":true}', visible: true }])(
  'gates workspace from persisted grants', async ({ grants, visible }) => {
    render(SessionChat, { session: { ...session, tool_grants_json: grants }, projectId: 'p1' })
    if (visible) expect(await screen.findByRole('complementary', { name: 'Workspace' })).toBeVisible()
    else expect(screen.queryByRole('complementary', { name: 'Workspace' })).not.toBeInTheDocument()
  },
)
it('refreshes tree after a newly polled tool message changes a path', async () => {
  expect(api.workspaceTree).toHaveBeenCalledTimes(2)
})
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/components/sessions/WorkspacePanel.test.ts`
Expected: FAIL because panel is absent.

- [ ] **Step 3: Implement responsive panel and safe file selection**

Split chat/workspace on desktop and stack on mobile. Parse malformed grant JSON as disabled. Render tree skeleton/error independently; fetch selected file content; offer `Save to source` only for regular lowercase `.md` files. A message refresh must update panel content without remounting the chat composer.

- [ ] **Step 4: Verify workspace and focus together**

Run: `rtk npm run test -- --run web/src/components/sessions/WorkspacePanel.test.ts web/src/components/sessions/SessionChat.focus.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/components/sessions/WorkspacePanel.svelte web/src/components/sessions/WorkspacePanel.test.ts web/src/components/sessions/SessionChat.svelte web/src/lib/api/index.ts
rtk git commit -m "feat(web): add session workspace panel"
```

### Task 46: Add Promotion Dialog and Operation Status Badges

**Files:**
- Create: `web/src/components/sessions/PromoteDialog.svelte`
- Create: `web/src/components/sessions/PromoteDialog.test.ts`
- Create: `web/src/components/sessions/OperationBadges.svelte`
- Create: `web/src/components/sessions/OperationBadges.test.ts`
- Create: `web/src/lib/promote.ts`
- Create: `web/src/lib/promote.test.ts`
- Modify: `web/src/components/sessions/SessionChat.svelte`
- Modify: `web/src/lib/api/index.ts`

**Interfaces:**
- Produces: `nextPromoteAttempt(previous, payload, uuid): { payload; key }`; POST `/api/v1/sessions/{id}/promote` with `Idempotency-Key`; persisted operation IDs key `personal-agent:v1:promote-operations:{sessionId}`.
- Consumes operation GET `/api/v1/operations/{operationId}` and retry POST `/api/v1/review/pending/{pendingId}/retry`.

- [ ] **Step 1: Write failing idempotency, lifecycle, and badge tests**

```ts
expect(nextPromoteAttempt(first, samePayload, uuid).key).toBe(first.key)
expect(nextPromoteAttempt(first, changedPayload, uuid).key).not.toBe(first.key)
expect(api.promoteSession).toHaveBeenCalledWith('s1', {
  workspace_path: 'draft.md', target_relative_path: 'notes/draft.md', review_mode: 'bites',
}, expect.any(String))
```

Test cancel/Escape/native close/back/session-switch cleanup, captured source file, `.md` validation, dialog state surviving polls, operation polling coalescing, retry-card deduplication, and all five exact badge strings.

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/lib/promote.test.ts web/src/components/sessions/PromoteDialog.test.ts web/src/components/sessions/OperationBadges.test.ts`
Expected: FAIL because promotion UI/helpers are absent.

- [ ] **Step 3: Implement promotion and resilient operation polling**

The dialog heading is exactly `Save to source`; field names are exactly `target_relative_path` and `review_mode`; require lowercase regular Markdown target/source rules already enforced by legacy tests. Capture the selected source and session at open. On success require `operation_id`, persist it, close, and poll. Keep dialog outside frequently-updated message markup. Disable retry while pending and retain ordinary errors independently from operation errors.

```ts
export const promoteSession = (sessionId: string, payload: PromotePayload, key: string) =>
  request<{ operation_id: string }>(`/api/v1/sessions/${encodeURIComponent(sessionId)}/promote`, {
    method: 'POST', body: payload, headers: { 'Idempotency-Key': key },
  })
```

- [ ] **Step 4: Run full session surface suite**

Run: `rtk npm run test -- --run web/src/components/sessions web/src/lib/promote.test.ts web/src/routes/ProjectSessionsPage.test.ts`
Expected: PASS, including focus while badges/workspace update.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/components/sessions/PromoteDialog.svelte web/src/components/sessions/PromoteDialog.test.ts web/src/components/sessions/OperationBadges.svelte web/src/components/sessions/OperationBadges.test.ts web/src/lib/promote.ts web/src/lib/promote.test.ts web/src/components/sessions/SessionChat.svelte web/src/lib/api/index.ts
rtk git commit -m "feat(web): add idempotent session promotion"
```
