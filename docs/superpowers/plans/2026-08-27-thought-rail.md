# Thought chip + Thoughts rail Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Worker dispatch (this repo):** `pi -p --approve` in `.worktrees/<branch>` from **origin/main**. Review = new `pi -p --tools read,bash`. See `docs/superpowers/PROMPT-pi-master-coordinator.md`.
>
> **Checkout:** `make docker-dev` bind-mounts one tree on `/src`. Implement in **that** tree (`docker inspect` Mounts Source). Do not `find`/`grep` `$HOME` / `/Users/<name>`. Named refs: `docs/superpowers/specs/2026-08-27-thought-rail/*.png`.

**Goal:** Show a live per-run Thought chip in the Agent thread; click swaps the existing hub right rail to that run’s tool timeline.

**Architecture:** No new HTTP route. Parse `tool_calls_json` + `run_id` already on GET messages. Tick elapsed from `runs/current` `started_at`. Pure helper `web/src/lib/thoughts.ts` builds chips + rows. `SessionChat` renders chips and hides empty assistant/tool JSON. Hub stores `thoughtsRunId` and passes `ThoughtsView` into `ProjectRail`, which replaces Config/Files until X.

**Tech Stack:** Svelte 5 + TypeScript + Vitest + Testing Library. Existing Go JSON. Node `>=22 <23`.

**Spec:** `docs/superpowers/specs/2026-08-27-thought-rail-design.md`

## Global Constraints

- No new HTTP endpoint. Do not add `/runs/{id}/thoughts`.
- One agent. Do not copy Grok multi-agent avatars / Agent 2 / Agent 3 from ref 3.
- Thread: no empty assistant bubbles, no `ASSISTANT` other-rows, no raw tool JSON.
- File cards (`changed_path`) stay if already implemented; this plan does not add them.
- Poll must not remount the composer (`SessionChat.focus.test.ts`). Hub `{#key}` stays `activeSession.id` only — never `thoughtsRunId`.
- Rail swap is hub `ProjectRail` only. Do not add a second rail column or a session tab named Thoughts.
- Tokens in `web/src/app.css`: `.thought-chip`, `.thoughts-rail`, `.thought-row`. No `bg-indigo-600`.
- Dual-channel hide for Thoughts vs Config: CSS **and** `hidden={…}` (jsdom `toBeVisible`). Do not pair `grid-template-columns: 0 1fr` with `display: none`.
- Web tests: Node `>=22 <23`. If host `@rollup/rollup-darwin-*` is missing, run inside docker-dev: `docker exec pa-feat-personal-agent-1 sh -c 'cd /src/web && npx vitest run <file>'`.
- After UI: browser vibe-pass on `http://localhost:8080/#/projects/eea7b476-e0b7-44d8-89e2-d3fafab87172/sessions` (orbit frost) vs the three PNGs. `make docker-dev`. Blocked ≠ passed.
- Do not merge/push unless the user asks.
- Never scan `$HOME`.

## File map

| Path | Role |
|------|------|
| `web/src/lib/api/types.ts` | `ChatMessage.run_id`, `tool_calls_json`, `tool_call_id`; `RunStatus.id`, `started_at` |
| `web/src/lib/thoughts.ts` | Parse calls, verbs, args, elapsed, chip visibility, thread items, `ThoughtsView` |
| `web/src/lib/thoughts.test.ts` | Pure tests for the helper |
| `web/src/components/sessions/SessionChat.svelte` | Chips in thread; hide empty assistant/tool JSON; `onOpenThoughts`; bindable `thoughtsRunId` |
| `web/src/components/sessions/SessionChat.test.ts` | Chip, empty hide, no JSON dump |
| `web/src/components/ProjectRail.svelte` | Thoughts overlay: header + X + timeline |
| `web/src/components/ProjectRail.test.ts` | Swap / restore |
| `web/src/routes/ProjectHubPage.svelte` | `thoughtsRunId`, previous tab, pass `ThoughtsView` |
| `web/src/routes/ProjectHubPage.test.ts` | Click chip → Thoughts; X restores; composer identity |
| `web/src/app.css` | Chip + rail tokens |
| `web/src/styles-baseline.test.ts` | Assert new class names |

## Canonical contracts

### Stored `tool_calls_json` (real DB shape — not nested OpenAI `function`)

```json
[{"id":"call_1","name":"read_knowledge","arguments":"{\"path\":\"source/standing-rule.md\"}"}]
```

Parse as `{ id: string, name: string, arguments: string }[]`. Skip entries with no `name` and no arg.

### Types (exact)

```ts
export type ThoughtRow = {
  id: string
  verb: string
  arg: string
  status?: 'ok' | 'error' | 'pending'
  detail?: string
}

export type ThoughtsView = {
  runId: string
  elapsedSec: number
  live: boolean
  rows: ThoughtRow[]
}
```

`ChatMessage` adds optional `run_id?: string`, `tool_calls_json?: string`, `tool_call_id?: string`.  
`RunStatus` adds optional `id?: string`, `started_at?: string`.

### Verbs

| `name` | `verb` |
|--------|--------|
| `read_file`, `read_knowledge` | `Read` |
| `list_knowledge`, `list_dir` | `List` |
| `write_file`, `edit_file` | `Write` |
| `mkdir` | `Mkdir` |
| other `snake_case` | title-case words (`search_web` → `Search web`) |
| other | raw `name` |

**Arg:** `JSON.parse(arguments).path` if string, else first string value on that object, else `""`. Truncate to 48 chars + `…`.

**Status:** tool message `content` containing `"error"` → `error` + snippet ≤ 80 chars (strip JSON quotes if it is `{"error":"..."}`). No matching tool row while `live` → `pending`. Else `ok`.

### Chip rules

- Label: `Thought for ${n}s` (n integer ≥ 0).
- Live: `current.status` is `queued` or `running` **and** `current.id === runId` → `elapsedSec = floor((nowMs - Date.parse(started_at || firstMessage.created_at)) / 1000)`, min 0.
- Past: `elapsedSec = floor((last.created_at - first.created_at) / 1000)` for that `run_id`; if any tool call exists, min 1.
- Show while live always. After terminal: show if any tool call **or** elapsed ≥ 1. Instant no-tool → no chip.
- One chip per `run_id`, in the thread **after that run’s user message** (or before the run’s first message if there is no user row).

### Rail

`thoughts: ThoughtsView | null`. Non-null → hide icon toolbar tabs (Config/Files) **and** their panels (`hidden` + CSS); show `.thoughts-rail` with heading `Thoughts`, button `Close thoughts` (visible X), body rows or `Working…` if `rows.length === 0`. X calls `onCloseThoughts`. Does not write `pa.projectRail.tab`. Hub restores the previous tab on close.

### SessionChat props

```ts
thoughtsRunId?: string | null
onOpenThoughts?: (runId: string) => void
thoughtsView?: ThoughtsView | null  // unused if hub derives from messages; see Task 3
```

Hub holds `thoughtsRunId`. Chip click calls `onOpenThoughts(runId)` (switch run if already open). New send does **not** clear `thoughtsRunId`. Session close / `onclose` clears it.

---

### Task 1: `thoughts.ts` helper + API types

**Files:**
- Modify: `web/src/lib/api/types.ts`
- Create: `web/src/lib/thoughts.ts`
- Test: `web/src/lib/thoughts.test.ts`

**Interfaces:**
- Consumes: `ChatMessage`, `RunStatus` as extended below
- Produces: `ThoughtRow`, `ThoughtsView`, `parseToolCalls`, `toolVerb`, `toolArg`, `toolStatus`, `runElapsedSec`, `shouldShowChip`, `buildThoughtsView`, `chipInsertAfterSequence`

- [ ] **Step 1: Write the failing test**

Create `web/src/lib/thoughts.test.ts`:

```ts
import { describe, expect, it } from 'vitest'
import type { ChatMessage, RunStatus } from './api/types'
import {
  buildThoughtsView,
  chipInsertAfterSequence,
  parseToolCalls,
  runElapsedSec,
  shouldShowChip,
  toolArg,
  toolStatus,
  toolVerb,
} from './thoughts'

const calls = JSON.stringify([
  { id: 'c1', name: 'read_knowledge', arguments: '{"path":"source/standing-rule.md"}' },
])

const msgs: ChatMessage[] = [
  { sequence: 1, role: 'user', content: 'what is in @standing-rule.md ?', run_id: 'r1', created_at: '2026-08-27T00:00:00Z' },
  { sequence: 2, role: 'assistant', content: '', run_id: 'r1', tool_calls_json: calls, created_at: '2026-08-27T00:00:05Z' },
  { sequence: 3, role: 'tool', content: '{"path":"source/standing-rule.md","content":"# hi"}', run_id: 'r1', tool_call_id: 'c1', created_at: '2026-08-27T00:00:06Z' },
  { sequence: 4, role: 'assistant', content: 'Here is the content.', run_id: 'r1', created_at: '2026-08-27T00:00:10Z' },
]

describe('thoughts', () => {
  it('parses flat tool_calls_json and maps verb/arg', () => {
    expect(parseToolCalls(calls)).toEqual([
      { id: 'c1', name: 'read_knowledge', arguments: '{"path":"source/standing-rule.md"}' },
    ])
    expect(toolVerb('read_knowledge')).toBe('Read')
    expect(toolVerb('list_knowledge')).toBe('List')
    expect(toolArg('{"path":"source/standing-rule.md"}')).toBe('source/standing-rule.md')
    expect(toolArg('{"q":"hello world that is quite a long query for the rail"}').length).toBeLessThanOrEqual(49)
  })

  it('skips nameless empty calls', () => {
    expect(parseToolCalls(JSON.stringify([{ id: 'x', name: '', arguments: '' }]))).toEqual([])
  })

  it('statuses error vs ok vs pending', () => {
    expect(toolStatus('{"error":"workspace tool request rejected"}', false)).toEqual({
      status: 'error',
      detail: 'workspace tool request rejected',
    })
    expect(toolStatus('{"ok":true}', false).status).toBe('ok')
    expect(toolStatus(undefined, true).status).toBe('pending')
  })

  it('shows a live chip and past chip with tools; hides instant no-tool', () => {
    const live: RunStatus = { id: 'r1', status: 'running', started_at: '2026-08-27T00:00:00Z' }
    expect(shouldShowChip({ runId: 'r1', messages: msgs, current: live, nowMs: Date.parse('2026-08-27T00:00:32Z') })).toBe(true)
    expect(runElapsedSec({ runId: 'r1', messages: msgs, current: live, nowMs: Date.parse('2026-08-27T00:00:32Z') })).toBe(32)
    expect(shouldShowChip({ runId: 'r1', messages: msgs, current: null, nowMs: 0 })).toBe(true)
    expect(runElapsedSec({ runId: 'r1', messages: msgs, current: null, nowMs: 0 })).toBe(10)
    const instant: ChatMessage[] = [
      { sequence: 1, role: 'user', content: 'hi', run_id: 'r2', created_at: '2026-08-27T00:00:00Z' },
      { sequence: 2, role: 'assistant', content: 'hey', run_id: 'r2', created_at: '2026-08-27T00:00:00Z' },
    ]
    expect(shouldShowChip({ runId: 'r2', messages: instant, current: null, nowMs: 0 })).toBe(false)
  })

  it('builds one-run rows and insert-after user sequence', () => {
    const view = buildThoughtsView({ runId: 'r1', messages: msgs, current: null, nowMs: 0 })
    expect(view).toEqual({
      runId: 'r1',
      elapsedSec: 10,
      live: false,
      rows: [{ id: 'c1', verb: 'Read', arg: 'source/standing-rule.md', status: 'ok' }],
    })
    expect(chipInsertAfterSequence(msgs, 'r1')).toBe(1)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run src/lib/thoughts.test.ts` (from `web/`; docker exec if host rollup missing).

Expected: FAIL — module `./thoughts` not found.

- [ ] **Step 3: Extend types + minimal helper**

`web/src/lib/api/types.ts` — add to `ChatMessage`:

```ts
  run_id?: string
  tool_calls_json?: string
  tool_call_id?: string
```

and to `RunStatus`:

```ts
  id?: string
  started_at?: string
```

Implement `web/src/lib/thoughts.ts` to satisfy the tests (no extra exports required this task). `shouldShowChip` / `runElapsedSec` take the object shown in the test. Truncate args with a single `…` so length ≤ 49. Error snippet: `JSON.parse` content, use `.error` string when present.

- [ ] **Step 4: Run test to verify it passes**

Run: `npx vitest run src/lib/thoughts.test.ts`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/api/types.ts web/src/lib/thoughts.ts web/src/lib/thoughts.test.ts
git commit -m "feat(web): group session tool calls into a Thoughts view"
```

---

### Task 2: Thought chip in the thread

**Files:**
- Modify: `web/src/components/sessions/SessionChat.svelte`
- Modify: `web/src/components/sessions/SessionChat.test.ts`
- Modify: `web/src/app.css`

**Interfaces:**
- Consumes: `buildThoughtsView`, `shouldShowChip`, `chipInsertAfterSequence`, `runElapsedSec` from `web/src/lib/thoughts.ts`; `RunStatus.id` / `started_at`
- Produces: chip button `aria-label="Thought for Ns"`; prop `onOpenThoughts?: (runId: string) => void`; `messagesEqual` also compares `run_id` and `tool_calls_json`

- [ ] **Step 1: Write the failing test**

In `SessionChat.test.ts`, add (same `api` mock / `session` fixture as existing tests):

```ts
  it('hides empty assistant tool-call turns and shows a Thought chip', async () => {
    vi.mocked(api.listMessages).mockResolvedValue([
      { sequence: 1, role: 'user', content: 'Can you tell me, what is in the @standing-rule.md ?', run_id: 'r1', created_at: '2026-08-27T00:00:00Z' },
      {
        sequence: 2,
        role: 'assistant',
        content: '',
        run_id: 'r1',
        tool_calls_json: JSON.stringify([
          { id: 'c1', name: 'read_knowledge', arguments: '{"path":"source/standing-rule.md"}' },
        ]),
        created_at: '2026-08-27T00:00:05Z',
      },
      { sequence: 3, role: 'tool', content: '{"error":"workspace tool request rejected"}', run_id: 'r1', tool_call_id: 'c1', created_at: '2026-08-27T00:00:06Z' },
      { sequence: 4, role: 'assistant', content: 'Here is the content of @standing-rule.md.', run_id: 'r1', created_at: '2026-08-27T00:00:10Z' },
    ])
    const onOpenThoughts = vi.fn()
    render(SessionChat, { props: { session, projectId: 'p1', pollInterval: 60_000, onOpenThoughts } })
    expect(await screen.findByText('Can you tell me, what is in the @standing-rule.md ?')).toBeInTheDocument()
    expect(screen.getByText('Here is the content of @standing-rule.md.')).toBeInTheDocument()
    expect(document.querySelectorAll('li[data-role="assistant"]')).toHaveLength(1)
    expect(document.querySelectorAll('li[data-role="other"]')).toHaveLength(0)
    expect(screen.queryByText(/^ASSISTANT$/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/workspace tool request rejected/)).not.toBeInTheDocument()
    const chip = screen.getByRole('button', { name: /Thought for \d+s/ })
    expect(chip).toHaveClass('thought-chip')
    await fireEvent.click(chip)
    expect(onOpenThoughts).toHaveBeenCalledWith('r1')
  })

  it('does not show a Thought chip for an instant no-tool reply', async () => {
    vi.mocked(api.listMessages).mockResolvedValue([
      { sequence: 1, role: 'user', content: 'hi', run_id: 'r2', created_at: '2026-08-27T00:00:00Z' },
      { sequence: 2, role: 'assistant', content: 'hey', run_id: 'r2', created_at: '2026-08-27T00:00:00Z' },
    ])
    render(SessionChat, { props: { session, projectId: 'p1', pollInterval: 60_000 } })
    expect(await screen.findByText('hey')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Thought for/ })).toBeNull()
  })
```

If a test named `does not render empty assistant bubbles` already exists, **extend it** with the chip assertions instead of duplicating.

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run src/components/sessions/SessionChat.test.ts -t "Thought chip"`

Expected: FAIL — no button `Thought for`.

- [ ] **Step 3: Minimal SessionChat + chip CSS**

In the message `{#each}`:

1. Do **not** render assistant/model when `!(content ?? '').trim()`.
2. Do **not** render `role === 'tool'` unless existing file-card branch already handles `changed_path` (keep that branch; skip JSON else).
3. Delete the generic `message-row--other` fallback.
4. After each `user` row, if `chipInsertAfterSequence(messages, runId) === message.sequence` and `shouldShowChip(...)`, render:

```svelte
<li class="message message-row message-row--thought" data-role="thought">
  <button
    type="button"
    class="thought-chip"
    aria-label="Thought for {n}s"
    onclick={() => onOpenThoughts?.(runId)}
  >
    <svg class="thought-chip__bulb" viewBox="0 0 24 24" aria-hidden="true"><!-- bulb --></svg>
    <svg class="thought-chip__panel" viewBox="0 0 24 24" aria-hidden="true"><!-- chevron panel --></svg>
    <span>Thought for {n}s</span>
  </button>
</li>
```

`n` from `runElapsedSec` with `nowMs = Date.now()`. While `run?.status` is `queued`/`running`, `$effect` `setInterval` 1000ms to bump a `nowMs` state so the label ticks. Clear interval on destroy / idle.

`onOpenThoughts` optional prop next to `embeddedInHub`.

`messagesEqual`: also `a.run_id !== b.run_id || a.tool_calls_json !== b.tool_calls_json`.

CSS (muted like ref 1; hover swaps icons like ref 2):

```css
.thought-chip {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: #71717a;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
}
.thought-chip__bulb, .thought-chip__panel {
  width: 18px;
  height: 18px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.8;
}
.thought-chip__panel { display: none; }
.thought-chip:hover .thought-chip__bulb { display: none; }
.thought-chip:hover .thought-chip__panel { display: block; }
```

Bulb path: rounded bulb + base (match ref 1). Panel path: rounded rect + left chevron (match ref 2). No emoji.

- [ ] **Step 4: Run tests**

Run: `npx vitest run src/components/sessions/SessionChat.test.ts src/components/sessions/SessionChat.focus.test.ts`

Expected: PASS (focus composer still same node).

- [ ] **Step 5: Commit**

```bash
git add web/src/components/sessions/SessionChat.svelte web/src/components/sessions/SessionChat.test.ts web/src/app.css
git commit -m "feat(web): Thought chip for tool-call runs"
```

---

### Task 3: Thoughts rail swap

**Files:**
- Modify: `web/src/components/ProjectRail.svelte`
- Modify: `web/src/components/ProjectRail.test.ts`
- Modify: `web/src/routes/ProjectHubPage.svelte`
- Modify: `web/src/routes/ProjectHubPage.test.ts`
- Modify: `web/src/app.css`
- Modify: `web/src/styles-baseline.test.ts`

**Interfaces:**
- Consumes: `ThoughtsView` from `web/src/lib/thoughts.ts`
- Produces: `ProjectRail` props `thoughts?: ThoughtsView | null`, `onCloseThoughts?: () => void`. Hub: `thoughtsRunId`, `thoughtsPrevTab`, `openThoughts(runId)`, `closeThoughts()`. SessionChat `onOpenThoughts={openThoughts}`.

- [ ] **Step 1: Write the failing tests**

`ProjectRail.test.ts`:

```ts
  it('swaps Config/Files for Thoughts and restores on close', async () => {
    const onCloseThoughts = vi.fn()
    const thoughts = {
      runId: 'r1',
      elapsedSec: 32,
      live: false,
      rows: [{ id: 'c1', verb: 'Read', arg: 'source/standing-rule.md', status: 'ok' as const }],
    }
    render(ProjectRail, { props: { projectId: 'p1', tab: 'config', thoughts, onCloseThoughts } })
    expect(screen.getByRole('heading', { name: 'Thoughts' })).toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Config' })).not.toBeVisible()
    expect(screen.queryByRole('tab', { name: 'Files' })).not.toBeVisible()
    expect(screen.getByText('Read')).toBeInTheDocument()
    expect(screen.getByText('source/standing-rule.md')).toBeInTheDocument()
    await fireEvent.click(screen.getByRole('button', { name: 'Close thoughts' }))
    expect(onCloseThoughts).toHaveBeenCalled()
  })

  it('shows Working… when the run has no tool rows yet', () => {
    render(ProjectRail, {
      props: {
        projectId: 'p1',
        thoughts: { runId: 'r1', elapsedSec: 3, live: true, rows: [] },
      },
    })
    expect(screen.getByText('Working…')).toBeInTheDocument()
  })
```

`ProjectHubPage.test.ts` (use existing api mock / `showModal` polyfill `beforeEach`):

```ts
  it('opens Thoughts from the chip and restores Config on close without remounting composer', async () => {
    vi.mocked(api.listMessages).mockResolvedValue([
      { sequence: 1, role: 'user', content: 'what is in the file?', run_id: 'r1', created_at: '2026-08-27T00:00:00Z' },
      {
        sequence: 2,
        role: 'assistant',
        content: '',
        run_id: 'r1',
        tool_calls_json: JSON.stringify([{ id: 'c1', name: 'read_knowledge', arguments: '{"path":"source/standing-rule.md"}' }]),
        created_at: '2026-08-27T00:00:05Z',
      },
      { sequence: 3, role: 'tool', content: '{"path":"source/standing-rule.md"}', run_id: 'r1', tool_call_id: 'c1', created_at: '2026-08-27T00:00:06Z' },
      { sequence: 4, role: 'assistant', content: 'Here it is.', run_id: 'r1', created_at: '2026-08-27T00:00:10Z' },
    ])
    // open the existing session the same way other hub tests open a chat (click the session row)
    render(ProjectHubPage, { props: { projectId: 'p1' } })
    await fireEvent.click(await screen.findByRole('button', { name: /orbit frost|Test 1|ivory bold|Chat/i }))
    const composer = await screen.findByLabelText('Message')
    await fireEvent.click(await screen.findByRole('button', { name: /Thought for/ }))
    expect(await screen.findByRole('heading', { name: 'Thoughts' })).toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Config' })).not.toBeVisible()
    await fireEvent.click(screen.getByRole('button', { name: 'Close thoughts' }))
    expect(screen.getByRole('tab', { name: 'Config' })).toBeVisible()
    expect(screen.getByLabelText('Message')).toBe(composer)
  })
```

Wire the session-open click to **whatever selector existing hub tests already use** (copy that pattern; do not invent a new list API). If hub tests seed `listSessions` with `{ id: 's1', title: 'Chat', ... }`, click that title.

`styles-baseline.test.ts` — add `'.thought-chip'` and `'.thoughts-rail'` to the session token list.

- [ ] **Step 2: Run tests to verify they fail**

Run: `npx vitest run src/components/ProjectRail.test.ts src/routes/ProjectHubPage.test.ts src/styles-baseline.test.ts`

Expected: FAIL — no `thoughts` prop / no heading Thoughts / missing CSS tokens.

- [ ] **Step 3: Implement rail + hub + CSS**

`ProjectRail` new props:

```ts
thoughts?: ThoughtsView | null
onCloseThoughts?: () => void
```

When `thoughts` is truthy: `hidden` on the icon tablist and Config/Files (and Workspace if present) panels; render:

```svelte
<section class="thoughts-rail" data-thoughts="1">
  <header class="thoughts-rail__header">
    <h2 class="thoughts-rail__title">Thoughts</h2>
    <button type="button" class="thoughts-rail__close" aria-label="Close thoughts" onclick={() => onCloseThoughts?.()}>×</button>
  </header>
  {#if thoughts.rows.length === 0}
    <p class="thoughts-rail__empty">Working…</p>
  {:else}
    <ol class="thoughts-rail__list">
      {#each thoughts.rows as row (row.id)}
        <li class="thought-row" data-status={row.status}>
          <span class="thought-row__verb">{row.verb}</span>
          {#if row.arg}<span class="thought-row__arg">{row.arg}</span>{/if}
        </li>
      {/each}
    </ol>
  {/if}
</section>
```

No avatar cluster. `×` is the close control (`aria-label="Close thoughts"`).

Hub:

```ts
let thoughtsRunId = $state<string | null>(null)
let thoughtsPrevTab = $state<ProjectRailTab | null>(null)

function openThoughts(runId: string) {
  if (thoughtsRunId == null) thoughtsPrevTab = railTab
  thoughtsRunId = runId
}
function closeThoughts() {
  thoughtsRunId = null
  if (thoughtsPrevTab) {
    railTab = thoughtsPrevTab
    thoughtsPrevTab = null
  }
}
```

Clear `thoughtsRunId` in `closeSession`. Derive `thoughtsView` in `SessionChat` **or** in the hub by importing `buildThoughtsView` only if hub has messages — **do not duplicate poll**. Pass a callback from SessionChat:

Add bindable on SessionChat:

```ts
thoughtsRunId = $bindable<string | null>(null)
```

Chip `onclick` → `onOpenThoughts?.(runId)` **and** hub `openThoughts`. Hub:

```svelte
<SessionChat
  ...
  onOpenThoughts={openThoughts}
/>
<ProjectRail
  thoughts={thoughtsView}
  onCloseThoughts={closeThoughts}
  ...
/>
```

`thoughtsView` must update as SessionChat polls. Implement as SessionChat `$effect` calling optional `onThoughtsView?: (view: ThoughtsView | null) => void` whenever `thoughtsRunId` or messages/`run` change:

```ts
onThoughtsView?: (view: ThoughtsView | null) => void
```

Hub: `let thoughtsView = $state<ThoughtsView | null>(null)` and `onThoughtsView={(v) => (thoughtsView = v)}`. When `thoughtsRunId` is null, SessionChat calls `onThoughtsView(null)`.

Do not put `thoughtsRunId` on the `{#key}`.

CSS: `.thoughts-rail` full panel height, header 48px to match `.rail-iconbar`, title ~16px 600, close 28×28 muted, rows 14px, arg muted italic, list `align-content: start`, no indigo.

- [ ] **Step 4: Run tests**

Run: `npx vitest run src/components/ProjectRail.test.ts src/routes/ProjectHubPage.test.ts src/components/sessions/SessionChat.test.ts src/components/sessions/SessionChat.focus.test.ts src/styles-baseline.test.ts`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/components/ProjectRail.svelte web/src/components/ProjectRail.test.ts \
  web/src/routes/ProjectHubPage.svelte web/src/routes/ProjectHubPage.test.ts \
  web/src/components/sessions/SessionChat.svelte web/src/app.css web/src/styles-baseline.test.ts
git commit -m "feat(web): swap project rail to Thoughts for a run"
```

---

### Task 4: Browser vibe-pass vs named refs

**Files:** none required if Task 3 already matches. Fix CSS/markup only if the live DOM diverges from the refs.

**Interfaces:**
- Consumes: running `make docker-dev` on `:8080`, Chrome CDP `localhost:9222`
- Produces: completion report with URL, viewport, side-by-side vs each PNG

- [ ] **Step 1: Open the real session**

URL: `http://localhost:8080/#/projects/eea7b476-e0b7-44d8-89e2-d3fafab87172/sessions`  
Open **orbit frost**. Hard-refresh if HMR is stale.

If docker-dev `/src` is not this branch, serve this tree (`PA_ADDR=:18081` or restart compose from this checkout). Host `:8080` from another mount ≠ pass.

- [ ] **Step 2: Check rest chip vs `1-element-thought.png`**

Muted lightbulb + `Thought for Ns` under the user bubble, above the reply. No `ASSISTANT` labels. Final standing-rule reply still visible.

- [ ] **Step 3: Check hover vs `2-element-thought-hover.png`**

Hover chip → bulb hides, panel-chevron shows, same label.

- [ ] **Step 4: Check click vs `3-element-thought-clicked.png` (structure, not Grok agents)**

Click chip → right rail header **Thoughts** + X; Config/Files gone; timeline rows (Read / List / … + path). X restores Config (or previous tab). Composer still focused-capable (not remounted).

- [ ] **Step 5: Commit only if you changed CSS/markup to match**

```bash
git add web/src/app.css web/src/components/sessions/SessionChat.svelte web/src/components/ProjectRail.svelte
git commit -m "fix(web): Thought chip and Thoughts rail match named refs"
```

If no code change, do not empty-commit. Report URL + viewport + pass/fail per ref.

---

## Self-review

| Spec | Task |
|------|------|
| Chip rest/hover/live/frozen | 2, 4 |
| Per-run chip placement | 1, 2 |
| Instant no-tool → no chip | 1, 2 |
| Hide empty assistant + tool JSON | 2 |
| Rail swap B, X restores tab | 3 |
| Timeline that run only, live Working… | 1, 3 |
| No new endpoint; parse `tool_calls_json` | 1 |
| `RunStatus.id` + `started_at` | 1 |
| Composer not remounted | 2, 3 |
| File cards unchanged | 2 (keep existing branch) |
| Tokens + vibe-pass vs 3 PNGs | 3, 4 |
| No Grok multi-agent chrome | 3, 4 |
| Stay on old run if new send | 3 (`thoughtsRunId` not cleared on send) |
