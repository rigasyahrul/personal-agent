# Composer `@` Workspace File Mentions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Worker dispatch (this repo):** `pi -p --approve` in `.worktrees/<branch>` from origin/main. Review = new `pi -p --tools read,bash`. See `docs/superpowers/PROMPT-pi-master-coordinator.md`.

**Goal:** When the user types `@` in the open-session Reply… composer, show a VS Code-style list of session workspace files and insert `@relative/path ` without sending.

**Architecture:** Pure helpers parse the `@` token, rank files, and splice the replacement. `SessionChat` keeps the existing `<textarea>` and mounts a listbox overlay inside `.session-composer__card`. File list comes from `GET /api/v1/sessions/{id}/workspace/tree`. Send payload is unchanged; the agent uses `read_file`.

**Tech Stack:** Svelte 5 + TypeScript + Vite + Vitest + Testing Library; tokens in `web/src/app.css`.

**Spec:** `docs/superpowers/specs/2026-08-27-composer-at-file-mentions-design.md`

## Global Constraints

- Spec wins. Filename-first list; insert `@relative/path` plus one trailing space.
- Session Reply… composer only. Not the hub start box. Not project notes / knowledge / `memory/**`.
- Keep the `<textarea>`. No contenteditable. Never remount `.session-composer` (existing `SessionChat.focus.test.ts` is a gate).
- Current API: `api.workspaceTree` + `workspaceEnabled(session)` (`tool_grants.workspace_files`). Do not invent a parallel client. If `/files/tree` already merged, use that same tree the Files rail uses.
- Grant off → `@` is plain text (no fetch, no overlay).
- No new HTTP API. No attaching file bytes on send.
- Node `>=22 <23` on `PATH` before `npm --prefix web test` / `make web-test`.
- Tokens in `app.css`. No `bg-indigo-600`, emoji bullets, or a second search field.
- After UI: browser vibe-pass on a real session composer. Blocked ≠ passed.
- Do not merge/push unless the user asks.

## File map

| Path | Role |
|------|------|
| `web/src/lib/mention-files.ts` | Pure mention parse / rank / insert |
| `web/src/lib/mention-files.test.ts` | Unit tests for those helpers |
| `web/src/app.css` | Overlay + row tokens |
| `web/src/styles-baseline.test.ts` | Token contract for mention overlay |
| `web/src/components/sessions/SessionChat.svelte` | Overlay, fetch, keyboard, a11y |
| `web/src/components/sessions/SessionChat.test.ts` | Composer mention behavior |
| `web/src/components/sessions/SessionChat.focus.test.ts` | Must still pass (no extra work unless broken) |

## Canonical contracts

```ts
// web/src/lib/mention-files.ts
import type { WorkspaceEntry } from './api/types'

export type MentionRange = {
  start: number // index of '@'
  end: number   // exclusive end of the '@'+non-whitespace run
  query: string // characters after '@'
}

export type RankedFile = {
  path: string  // workspace-relative, e.g. 'notes/standing-rule.md'
  name: string  // basename, e.g. 'standing-rule.md'
  parent: string // directory prefix with trailing slash, or '' for root files
}

export function activeMention(text: string, cursor: number): MentionRange | null
export function rankWorkspaceFiles(
  entries: WorkspaceEntry[],
  query: string,
  limit?: number, // default 10
): RankedFile[]
export function insertMention(
  text: string,
  mention: MentionRange,
  path: string,
): { text: string; cursor: number }
```

**Product copy (exact):** `Loading files…` / `No matching files` / `Couldn't load files`

**a11y:** listbox accessible name `Workspace files`. Textarea `aria-label` stays `Message`.

**Rank:** files only (`kind === 'file'`). Case-insensitive. (1) basename starts with query (2) basename contains (3) path contains (4) `path.localeCompare`. Empty query → all files, still ranked, still capped at 10.

---

### Task 1: Mention helpers (parse, rank, insert)

**Files:**
- Create: `web/src/lib/mention-files.ts`
- Create: `web/src/lib/mention-files.test.ts`

**Interfaces:**
- Consumes: `WorkspaceEntry` from `web/src/lib/api/types.ts`.
- Produces: `MentionRange`, `RankedFile`, `activeMention`, `rankWorkspaceFiles`, `insertMention` with the signatures in Canonical contracts.

- [ ] **Step 1: Write the failing helper tests**

Create `web/src/lib/mention-files.test.ts`:

```ts
import { describe, expect, it } from 'vitest'
import { activeMention, insertMention, rankWorkspaceFiles } from './mention-files'
import type { WorkspaceEntry } from './api/types'

describe('activeMention', () => {
  it('detects @ at start and after whitespace, including newlines', () => {
    expect(activeMention('@', 1)).toEqual({ start: 0, end: 1, query: '' })
    expect(activeMention('@stand', 6)).toEqual({ start: 0, end: 6, query: 'stand' })
    expect(activeMention('see @stand', 10)).toEqual({ start: 4, end: 10, query: 'stand' })
    expect(activeMention('see\n@x', 6)).toEqual({ start: 4, end: 6, query: 'x' })
  })

  it('uses the whole token when the cursor is in the middle', () => {
    const text = '@standing-rule.md'
    expect(activeMention(text, 7)).toEqual({ start: 0, end: 17, query: 'standing-rule.md' })
  })

  it('ignores foo@bar and a cursor not inside an @ token', () => {
    expect(activeMention('foo@bar', 7)).toBeNull()
    expect(activeMention('hello @x more', 5)).toBeNull()
    expect(activeMention('hello @x more', 13)).toBeNull()
  })
})

describe('rankWorkspaceFiles', () => {
  const entries: WorkspaceEntry[] = [
    { path: 'standing-rule.md', kind: 'file' },
    { path: 'notes/standing-rule.md', kind: 'file' },
    { path: 'notes', kind: 'directory' },
    { path: 'other.md', kind: 'file' },
    { path: 'notes/deep/alpha.md', kind: 'file' },
  ]

  it('drops directories and ranks basename starts-with before path substring', () => {
    const rows = rankWorkspaceFiles(entries, 'stand')
    expect(rows.map((r) => r.path)).toEqual(['notes/standing-rule.md', 'standing-rule.md'])
    expect(rows[0]).toEqual({
      path: 'notes/standing-rule.md',
      name: 'standing-rule.md',
      parent: 'notes/',
    })
    expect(rows[1]?.parent).toBe('')
  })

  it('matches path substrings and caps at 10', () => {
    expect(rankWorkspaceFiles(entries, 'notes').map((r) => r.path)).toEqual([
      'notes/deep/alpha.md',
      'notes/standing-rule.md',
    ])
    const many: WorkspaceEntry[] = Array.from({ length: 12 }, (_, i) => ({
      path: `f${String(i).padStart(2, '0')}.md`,
      kind: 'file' as const,
    }))
    expect(rankWorkspaceFiles(many, '').map((r) => r.path)).toHaveLength(10)
  })
})

describe('insertMention', () => {
  it('replaces the whole token and adds a trailing space', () => {
    const mention = activeMention('what is in @stand', 17)
    expect(mention).not.toBeNull()
    expect(insertMention('what is in @stand', mention!, 'notes/standing-rule.md')).toEqual({
      text: 'what is in @notes/standing-rule.md ',
      cursor: 35,
    })
  })

  it('preserves text after the token', () => {
    const text = 'see @stand please'
    const mention = activeMention(text, 10)
    expect(insertMention(text, mention!, 'a.md')).toEqual({
      text: 'see @a.md  please',
      cursor: 10,
    })
  })
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
node -v
# must print v22.x
npm --prefix web test -- src/lib/mention-files.test.ts
```

Expected: FAIL (module `./mention-files` not found, or exports missing).

- [ ] **Step 3: Implement helpers**

Create `web/src/lib/mention-files.ts`:

```ts
import type { WorkspaceEntry } from './api/types'

export type MentionRange = {
  start: number
  end: number
  query: string
}

export type RankedFile = {
  path: string
  name: string
  parent: string
}

const DEFAULT_LIMIT = 10
const WS = /[ \t\n]/

export function activeMention(text: string, cursor: number): MentionRange | null {
  if (cursor < 0 || cursor > text.length) return null
  let start = cursor
  while (start > 0 && !WS.test(text[start - 1]!)) start -= 1
  let end = cursor
  while (end < text.length && !WS.test(text[end]!)) end += 1
  const token = text.slice(start, end)
  if (!token.startsWith('@')) return null
  return { start, end, query: token.slice(1) }
}

function basename(path: string): string {
  const i = path.lastIndexOf('/')
  return i < 0 ? path : path.slice(i + 1)
}

function parentDir(path: string): string {
  const i = path.lastIndexOf('/')
  return i < 0 ? '' : path.slice(0, i + 1)
}

export function rankWorkspaceFiles(
  entries: WorkspaceEntry[],
  query: string,
  limit = DEFAULT_LIMIT,
): RankedFile[] {
  const q = query.trim().toLowerCase()
  const files = entries.filter((entry) => entry.kind === 'file')
  const matched = q
    ? files.filter((entry) => entry.path.toLowerCase().includes(q))
    : files
  const scored = matched.map((entry) => {
    const name = basename(entry.path)
    const n = name.toLowerCase()
    let rank = 3
    if (!q || n.startsWith(q)) rank = 1
    else if (n.includes(q)) rank = 2
    return { path: entry.path, name, parent: parentDir(entry.path), rank }
  })
  scored.sort((a, b) => a.rank - b.rank || a.path.localeCompare(b.path))
  return scored.slice(0, limit).map(({ rank: _rank, ...row }) => row)
}

export function insertMention(
  text: string,
  mention: MentionRange,
  path: string,
): { text: string; cursor: number } {
  const inserted = `@${path} `
  return {
    text: text.slice(0, mention.start) + inserted + text.slice(mention.end),
    cursor: mention.start + inserted.length,
  }
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
npm --prefix web test -- src/lib/mention-files.test.ts
```

Expected: PASS. If `insertMention` cursor assertion fails, compute `cursor` as `mention.start + inserted.length` (for `'what is in @notes/standing-rule.md '` that is 35). If rank order fails, `localeCompare` on full path must put `notes/standing-rule.md` before `standing-rule.md`.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/mention-files.ts web/src/lib/mention-files.test.ts
git commit -m "$(cat <<'EOF'
feat(web): add @ workspace file mention helpers

Parse the composer @ token, rank session files filename-first,
and insert @relative/path with a trailing space.
EOF
)"
```

---

### Task 2: Mention overlay tokens

**Files:**
- Modify: `web/src/app.css` (`.session-composer__card` near line 1496; add mention rules after `.session-composer--hidden`)
- Modify: `web/src/styles-baseline.test.ts` (`declares session-focus layout tokens`)

**Interfaces:**
- Consumes: existing `--panel`, `--border`, `--muted`, `--accent`, `--accent-soft`, `--fg`, `--radius-sm`, `--radius-lg`.
- Produces: classes `.session-composer__mentions`, `.session-composer__mentions-status`, `.mention-option`, `.mention-option--active`, `.mention-option__name`, `.mention-option__path`. Card is `position: relative` so the list can sit above it.

- [ ] **Step 1: Write the failing token tests**

In `web/src/styles-baseline.test.ts`, add these class names to the existing `declares session-focus layout tokens` array:

```ts
'.session-composer__mentions',
'.session-composer__mentions-status',
'.mention-option',
'.mention-option--active',
'.mention-option__name',
'.mention-option__path',
```

Add a new test in the same file:

```ts
  it('mention overlay uses theme tokens and editor-row density', () => {
    expect(css).toMatch(/\.session-composer__card\s*\{[^}]*position:\s*relative/s)
    expect(css).toMatch(/\.session-composer__mentions\s*\{[^}]*position:\s*absolute/s)
    expect(css).toMatch(/\.session-composer__mentions\s*\{[^}]*bottom:\s*calc\(100%/s)
    expect(css).toMatch(/\.mention-option\s*\{[^}]*min-height:\s*36px/s)
    expect(css).toMatch(/\.mention-option--active\s*\{[^}]*background:\s*var\(--accent-soft\)/s)
    const blocks =
      css.match(
        /\.(session-composer__mentions[\w-]*|mention-option[\w-]*)\s*\{[^}]*\}/g,
      ) ?? []
    expect(blocks.length).toBeGreaterThan(3)
    expect(blocks.filter((block) => /indigo/i.test(block))).toEqual([])
  })
```

- [ ] **Step 2: Run test to verify it fails**

```bash
npm --prefix web test -- src/styles-baseline.test.ts
```

Expected: FAIL — strings not found in `app.css`.

- [ ] **Step 3: Add CSS**

On `.session-composer__card`, add `position: relative;` (keep existing width/grid/padding/border/radius/background/shadow).

After `.session-composer--hidden`, append:

```css
.session-composer__mentions {
  position: absolute;
  left: 0;
  right: 0;
  bottom: calc(100% + 8px);
  z-index: 3;
  margin: 0;
  padding: 6px;
  max-height: 280px;
  overflow: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--panel);
  box-shadow: 0 1px 2px rgb(24 24 27 / 0.04), 0 4px 16px rgb(24 24 27 / 0.06);
}
.session-composer__mentions ul {
  list-style: none;
  margin: 0;
  padding: 0;
}
.session-composer__mentions-status {
  margin: 0;
  padding: 10px 12px;
  font-size: 13px;
  color: var(--muted);
}
.mention-option {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
  min-height: 36px;
  padding: 8px 10px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  text-align: left;
  font: inherit;
  color: inherit;
  cursor: pointer;
}
.mention-option--active,
.mention-option:hover {
  background: var(--accent-soft);
}
.mention-option__name {
  font-size: 13px;
  font-weight: 550;
  color: var(--fg);
}
.mention-option--active .mention-option__name,
.mention-option:hover .mention-option__name {
  color: var(--accent);
}
.mention-option__path {
  font-size: 12px;
  color: var(--muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
```

Do not add indigo, gradients, or emoji.

- [ ] **Step 4: Run tests to verify they pass**

```bash
npm --prefix web test -- src/styles-baseline.test.ts
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/app.css web/src/styles-baseline.test.ts
git commit -m "$(cat <<'EOF'
feat(web): add session composer @ mention overlay tokens

List sits above the composer card with 36px filename rows
and accent-soft highlight. No indigo soup.
EOF
)"
```

---

### Task 3: Wire the picker into SessionChat

**Files:**
- Modify: `web/src/components/sessions/SessionChat.svelte`
- Modify: `web/src/components/sessions/SessionChat.test.ts`
- Verify: `web/src/components/sessions/SessionChat.focus.test.ts` (must stay green)

**Interfaces:**
- Consumes: `activeMention`, `rankWorkspaceFiles`, `insertMention` from Task 1; overlay classes from Task 2; `api.workspaceTree`; `workspaceEnabled` (already imported as `showWorkspace`); `changedPathsFromMessages` from `web/src/lib/workspace-tree.ts`.
- Produces: listbox `Workspace files` when an active mention exists and workspace files are granted; insert on Enter/Tab/click; Enter with list closed still sends.

- [ ] **Step 1: Write the failing SessionChat tests**

Append inside the existing `describe('SessionChat', …)` in `web/src/components/sessions/SessionChat.test.ts` (same mock/`beforeEach` as the rest of the file). Use `pollInterval: 60_000` like the other tests.

```ts
  const granted = { ...session, tool_grants: { workspace_files: true as const } }
  const treeEntries = [
    { path: 'standing-rule.md', kind: 'file' as const },
    { path: 'notes/standing-rule.md', kind: 'file' as const },
    { path: 'notes', kind: 'directory' as const },
    { path: 'other.md', kind: 'file' as const },
  ]

  async function typeMention(composer: HTMLTextAreaElement, value: string) {
    await fireEvent.input(composer, { target: { value } })
    composer.setSelectionRange(value.length, value.length)
    await fireEvent.keyUp(composer)
  }

  it('opens a filename-first workspace list on @ when granted', async () => {
    vi.mocked(api.workspaceTree).mockResolvedValue({ entries: treeEntries })
    render(SessionChat, { props: { session: granted, projectId: 'p1', pollInterval: 60_000 } })
    const composer = (await screen.findByLabelText('Message')) as HTMLTextAreaElement
    await typeMention(composer, '@')
    const list = await screen.findByRole('listbox', { name: 'Workspace files' })
    expect(list).toBeInTheDocument()
    expect(screen.getAllByText('standing-rule.md').length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText('notes/')).toBeInTheDocument()
    expect(screen.queryByRole('option', { name: 'notes' })).toBeNull()
    expect(composer).toHaveAttribute('aria-expanded', 'true')
  })

  it('does not open on @ when workspace files are not granted', async () => {
    render(SessionChat, { props: { session, projectId: 'p1', pollInterval: 60_000 } })
    const composer = (await screen.findByLabelText('Message')) as HTMLTextAreaElement
    await typeMention(composer, '@')
    expect(screen.queryByRole('listbox', { name: 'Workspace files' })).toBeNull()
    expect(api.workspaceTree).not.toHaveBeenCalled()
  })

  it('does not open for foo@bar', async () => {
    vi.mocked(api.workspaceTree).mockResolvedValue({ entries: treeEntries })
    render(SessionChat, { props: { session: granted, projectId: 'p1', pollInterval: 60_000 } })
    const composer = (await screen.findByLabelText('Message')) as HTMLTextAreaElement
    await typeMention(composer, 'foo@bar')
    expect(screen.queryByRole('listbox', { name: 'Workspace files' })).toBeNull()
  })

  it('filters as the query grows', async () => {
    vi.mocked(api.workspaceTree).mockResolvedValue({ entries: treeEntries })
    render(SessionChat, { props: { session: granted, projectId: 'p1', pollInterval: 60_000 } })
    const composer = (await screen.findByLabelText('Message')) as HTMLTextAreaElement
    await typeMention(composer, '@other')
    await screen.findByRole('listbox', { name: 'Workspace files' })
    expect(screen.getByRole('option', { name: /other\.md/ })).toBeInTheDocument()
    expect(screen.queryByText('standing-rule.md')).toBeNull()
  })

  it('Enter on an open list inserts @path and does not send', async () => {
    vi.mocked(api.sendMessage).mockResolvedValue(null)
    vi.mocked(api.workspaceTree).mockResolvedValue({ entries: treeEntries })
    render(SessionChat, {
      props: { session: granted, projectId: 'p1', pollInterval: 60_000, uuid: () => 'mention-key' },
    })
    const composer = (await screen.findByLabelText('Message')) as HTMLTextAreaElement
    await typeMention(composer, '@stand')
    await screen.findByRole('listbox', { name: 'Workspace files' })
    await fireEvent.keyDown(composer, { key: 'Enter', shiftKey: false })
    expect(api.sendMessage).not.toHaveBeenCalled()
    expect(composer).toHaveValue('@notes/standing-rule.md ')
    expect(screen.queryByRole('listbox', { name: 'Workspace files' })).toBeNull()
  })

  it('Enter with the list closed still sends', async () => {
    vi.mocked(api.sendMessage).mockResolvedValue(null)
    render(SessionChat, {
      props: { session: granted, projectId: 'p1', pollInterval: 60_000, uuid: () => 'send-key' },
    })
    const composer = (await screen.findByLabelText('Message')) as HTMLTextAreaElement
    await fireEvent.input(composer, { target: { value: 'Hello from Enter' } })
    await fireEvent.keyDown(composer, { key: 'Enter', shiftKey: false })
    await waitFor(() => {
      expect(api.sendMessage).toHaveBeenCalledWith(
        granted.id,
        expect.objectContaining({ content: 'Hello from Enter', request_key: 'send-key' }),
      )
    })
  })

  it('Escape closes the list and keeps the typed query', async () => {
    vi.mocked(api.workspaceTree).mockResolvedValue({ entries: treeEntries })
    render(SessionChat, { props: { session: granted, projectId: 'p1', pollInterval: 60_000 } })
    const composer = (await screen.findByLabelText('Message')) as HTMLTextAreaElement
    await typeMention(composer, '@stand')
    await screen.findByRole('listbox', { name: 'Workspace files' })
    await fireEvent.keyDown(composer, { key: 'Escape' })
    expect(screen.queryByRole('listbox', { name: 'Workspace files' })).toBeNull()
    expect(composer).toHaveValue('@stand')
  })

  it('Enter on an open empty list does not send', async () => {
    vi.mocked(api.sendMessage).mockResolvedValue(null)
    vi.mocked(api.workspaceTree).mockResolvedValue({ entries: treeEntries })
    render(SessionChat, {
      props: { session: granted, projectId: 'p1', pollInterval: 60_000, uuid: () => 'empty-key' },
    })
    const composer = (await screen.findByLabelText('Message')) as HTMLTextAreaElement
    await typeMention(composer, '@zzzz-nope')
    expect(await screen.findByText('No matching files')).toBeInTheDocument()
    await fireEvent.keyDown(composer, { key: 'Enter', shiftKey: false })
    expect(api.sendMessage).not.toHaveBeenCalled()
    expect(composer).toHaveValue('@zzzz-nope')
  })

  it('shows a tree error in the overlay and keeps the textarea node', async () => {
    vi.mocked(api.workspaceTree).mockRejectedValue(new Error('boom'))
    render(SessionChat, { props: { session: granted, projectId: 'p1', pollInterval: 60_000 } })
    const composer = (await screen.findByLabelText('Message')) as HTMLTextAreaElement
    await typeMention(composer, '@')
    expect(await screen.findByText("Couldn't load files")).toBeInTheDocument()
    expect(screen.getByLabelText('Message')).toBe(composer)
    await fireEvent.keyDown(composer, { key: 'Enter', shiftKey: false })
    expect(api.sendMessage).not.toHaveBeenCalled()
  })
```

Option accessible names: if the option’s accessible name is only the filename (name + parent may concatenate), use `getByText('notes/')` and `getAllByText('standing-rule.md')` rather than a brittle `name:` regex. For `@other`, `getByText('other.md')` is enough if `getByRole('option', { name: /other\.md/ })` fails.

- [ ] **Step 2: Run tests to verify they fail**

```bash
npm --prefix web test -- src/components/sessions/SessionChat.test.ts
```

Expected: FAIL — no listbox / values unchanged.

- [ ] **Step 3: Wire SessionChat**

Import:

```ts
  import {
    changedPathsFromMessages,
  } from '../../lib/workspace-tree'
  import type { WorkspaceEntry } from '../../lib/api/types'
  import {
    activeMention,
    insertMention,
    rankWorkspaceFiles,
    type RankedFile,
  } from '../../lib/mention-files'
```

(`WorkspaceFile` is already imported from types; add `WorkspaceEntry` to that import instead of a second types import.)

State (next to `draft`):

```ts
  let composerEl: HTMLTextAreaElement | undefined = $state()
  let caret = $state(0)
  let mentionDismissed = $state(false)
  let treeEntries = $state<WorkspaceEntry[] | null>(null)
  let treeLoading = $state(false)
  let treeError = $state('')
  let treeLoadToken = 0
  let treeSignature = ''
  let mentionIndex = $state(0)
```

Derived:

```ts
  const mention = $derived(activeMention(draft, caret))
  const mentionActive = $derived(Boolean(showWorkspace && mention && !mentionDismissed))
  const mentionRows = $derived(
    mentionActive && treeEntries
      ? rankWorkspaceFiles(treeEntries, mention?.query ?? '')
      : [],
  )
```

Load tree when the picker should open. Reset cache when `session.id` changes. Refresh when tool `changed_path` signature changes (same idea as `SessionFilesBar`):

```ts
  async function ensureTree() {
    const token = ++treeLoadToken
    treeLoading = true
    treeError = ''
    try {
      const tree = await api.workspaceTree(session.id)
      if (token !== treeLoadToken) return
      treeEntries = tree?.entries ?? []
    } catch {
      if (token !== treeLoadToken) return
      treeError = "Couldn't load files"
      treeEntries = []
    } finally {
      if (token === treeLoadToken) treeLoading = false
    }
  }

  $effect(() => {
    void session.id
    treeEntries = null
    treeError = ''
    treeSignature = ''
    treeLoadToken += 1
  })

  $effect(() => {
    if (!mentionActive || !showWorkspace) return
    const sig = [...changedPathsFromMessages(messages)].sort().join('|')
    if (treeEntries === null || (sig && sig !== treeSignature)) {
      if (sig) treeSignature = sig
      void ensureTree()
    }
  })
```

Sync caret and clear Escape-dismiss when the token changes:

```ts
  function syncCaret(e: Event) {
    const el = e.currentTarget as HTMLTextAreaElement
    caret = el.selectionStart ?? el.value.length
  }

  $effect(() => {
    const token = mention ? `${mention.start}:${mention.query}` : ''
    void token
    mentionDismissed = false
  })
```

The dismiss-reset effect must not fight Escape: reset only when `mention.start` / `query` changes, not on every caret tick inside the same token. If a naive effect reopens the list on Escape, gate it on `mention?.start` + `mention?.query` as above (caret movement inside `@stand` does not change those).

`pickMention` + keyboard — **replace** `onComposerKeydown` with:

```ts
  function pickMention(row: RankedFile) {
    const current = activeMention(draft, caret)
    if (!current) return
    const next = insertMention(draft, current, row.path)
    draft = next.text
    caret = next.cursor
    mentionDismissed = true
    requestAnimationFrame(() => {
      composerEl?.focus()
      composerEl?.setSelectionRange(next.cursor, next.cursor)
    })
  }

  function onComposerKeydown(e: KeyboardEvent) {
    if (e.isComposing) return
    if (mentionActive) {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        if (mentionRows.length) mentionIndex = (mentionIndex + 1) % mentionRows.length
        return
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        if (mentionRows.length) {
          mentionIndex = (mentionIndex - 1 + mentionRows.length) % mentionRows.length
        }
        return
      }
      if (e.key === 'Escape') {
        e.preventDefault()
        mentionDismissed = true
        return
      }
      if (e.key === 'Enter' || e.key === 'Tab') {
        e.preventDefault()
        const row = mentionRows[mentionIndex] ?? mentionRows[0]
        if (row) pickMention(row)
        return
      }
    }
    if (e.key !== 'Enter' || e.shiftKey) return
    e.preventDefault()
    if (sendDisabled || !draft.trim()) return
    const form = (e.currentTarget as HTMLTextAreaElement).form
    form?.requestSubmit()
  }
```

Clamp highlight when the filtered list shrinks:

```ts
  $effect(() => {
    if (mentionIndex >= mentionRows.length) mentionIndex = 0
  })
```

Markup: keep the same `<form class="session-composer">`. Do not wrap it in `{#if}`. Update the card + textarea:

```svelte
        <div class="session-composer__card">
          {#if mentionActive}
            <div class="session-composer__mentions" id="session-composer-mentions">
              {#if treeLoading && treeEntries === null}
                <p class="session-composer__mentions-status">Loading files…</p>
              {:else if treeError}
                <p class="session-composer__mentions-status">{treeError}</p>
              {:else if mentionRows.length === 0}
                <p class="session-composer__mentions-status">No matching files</p>
              {:else}
                <ul role="listbox" aria-label="Workspace files">
                  {#each mentionRows as row, i (row.path)}
                    <li
                      id={`mention-option-${i}`}
                      class="mention-option"
                      class:mention-option--active={i === mentionIndex}
                      role="option"
                      aria-selected={i === mentionIndex}
                      onmousedown={(event) => {
                        event.preventDefault()
                        pickMention(row)
                      }}
                    >
                      <span class="mention-option__name">{row.name}</span>
                      {#if row.parent}
                        <span class="mention-option__path">{row.parent}</span>
                      {/if}
                    </li>
                  {/each}
                </ul>
              {/if}
            </div>
          {/if}
          <textarea
            class="session-composer__input"
            name="message"
            aria-label="Message"
            placeholder="Reply…"
            required
            rows="2"
            bind:this={composerEl}
            bind:value={draft}
            autocomplete="off"
            aria-autocomplete="list"
            aria-expanded={mentionActive && mentionRows.length > 0}
            aria-controls={mentionActive ? 'session-composer-mentions' : undefined}
            aria-activedescendant={
              mentionActive && mentionRows.length
                ? `mention-option-${mentionIndex}`
                : undefined
            }
            oninput={syncCaret}
            onkeyup={syncCaret}
            onclick={syncCaret}
            onselect={syncCaret}
            onkeydown={onComposerKeydown}
          ></textarea>
```

`aria-expanded="true"` in the open-list test: keep `aria-expanded={mentionActive && mentionRows.length > 0}` so it is true only for a populated listbox (the test that looks for `listbox` / `aria-expanded`). Empty/error/loading use the status `<p>`, not a listbox.

Do not add a `.mention-list` class; Task 2 already sets `.session-composer__mentions ul { list-style: none }`.

- [ ] **Step 4: Run SessionChat tests (including focus)**

```bash
npm --prefix web test -- src/components/sessions/SessionChat.test.ts src/components/sessions/SessionChat.focus.test.ts src/lib/mention-files.test.ts src/styles-baseline.test.ts
```

Expected: PASS.

If Enter-insert fails because caret never entered the token, `syncCaret` must run on `input` **and** `keyup`; `typeMention` in tests fires both.

If Escape immediately reopens, the dismiss-reset effect is too broad — key off `mention.start` + `mention.query` only.

If the focus test fails, the `<form class="session-composer">` was destroyed/recreated. Put the overlay **inside** the card; do not `{#if agentActive}` the form.

- [ ] **Step 5: Browser vibe-pass (HARD)**

Start or use `make docker-dev`. Open the real session composer (example: `http://localhost:8080/#/projects/eea7b476-e0b7-44d8-89e2-d3fafab87172/sessions` → orbit frost, or any granted session).

Check:

1. Type `@` → list above the composer, filenames first.
2. Type more characters → list filters.
3. Arrow + Enter inserts `@path ` and does not send.
4. Escape leaves the query.
5. Enter with no `@` still sends.
6. Focus stays in the textarea while the list is open.

Report URL + viewport + what you saw. If the app is down: start it or mark **blocked**. Do not claim UI done from CSS/unit tests.

Worktree checkout: do **not** treat laptop `:8080` as this tree (`docker-dev` mounts the laptop repo). Serve this tree (`PA_ADDR=:1808x`) or run vibe-pass only when this code is what `:8080` serves.

- [ ] **Step 6: Commit**

```bash
git add web/src/components/sessions/SessionChat.svelte web/src/components/sessions/SessionChat.test.ts web/src/app.css
git commit -m "$(cat <<'EOF'
feat(web): suggest workspace files when typing @ in chat

Overlay a listbox on the Reply composer, insert @relative/path,
and keep Enter-to-send when the list is closed.
EOF
)"
```

---

## Spec coverage (self-review)

| Spec section | Task |
|--------------|------|
| Filename-first list, dim parent | 1 + 3 |
| Insert `@relative/path ` | 1 + 3 |
| Keyboard arrows / Enter / Tab / Escape; Enter closed still sends | 3 |
| `foo@bar` ignored | 1 + 3 |
| Grant off | 3 |
| Empty / loading / error copy | 3 |
| Tree = `workspaceTree`, files only, cap 10 | 1 + 3 |
| No new API / no attach | all (non-goals) |
| Tokens + no indigo | 2 |
| Focus / form ancestry | 3 |
| Vibe-pass | 3 |
| Hub start / chips / notes | out of scope |

No TBD. Signatures in Task 3 match Task 1 (`MentionRange`, `RankedFile`, `insertMention` returns `{ text, cursor }`).
