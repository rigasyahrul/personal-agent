# Benchmark UI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Worker dispatch (this repo):** After each task: `consulting-grok-review` via **new** `amp -m grok45 --no-archive-after-execute -x '…'` (never `-ox`, never Task/OpenAI self-review). Isolate product work in git worktrees when using local `-x`.
>
> **Spec:** `docs/superpowers/specs/2026-08-20-benchmark-ui-redesign-design.md`  
> **Lock:** `docs/superpowers/plans/2026-08-20-benchmark-ui-redesign-lock.md`

**Goal:** Make the authenticated UI structurally match the Claude / Grok / Amp benchmark screenshots: compact shell nav, Claude-style project hub (prompt + sessions below + right rail), Amp open-session tabs with bottom composer and assistant copy, Grok Memory|Files rail, vault name-first list, and modal creates.

**Architecture:** Path C surface pack. Phase A is a pure CSS density fix for stretched sidebar nav. Phase B adds a shared `<dialog>` Modal, rewrites ProjectHubPage into a two-pane workspace (main + default-open ProjectRail), embeds SessionChat in the hub main canvas, migrates vault projects to name-first rows, then hardens `frontend-ui-craft` with a benchmark fidelity gate and full vibe-pass. Memory/soul persist API is out of scope (chrome only).

**Tech Stack:** Svelte 5 + TypeScript + Vite + Tailwind + Vitest + Testing Library; tokens in `web/src/app.css`; Go serves `web/dist` on `:8080`.

## Global Constraints

- Spec wins over `2026-08-20-session-focus-layout-design.md` on rail default-open, Memory|Files tabs, assistant copy, hub structure, and bottom-composer gates.
- Node `>=22 <23` on `PATH` before any `npm test` / `make web-test`.
- Rebuild `web/dist` + cache-bust (`?v=<ts>#/route`) before claiming localhost vibe-pass (Go serves dist, not Vite HMR).
- Polled session UI: never remount/replace a focused composer (`SessionChat.focus.test.ts` is a hard gate).
- Memory tab: design fields only — no enabled save, toast, or “saved” claim without API.
- Creates: modals only for New project / New vault — no inline `form-inline` soup.
- App left nav: user collapse only — never auto-collapse on hub/session entry.
- Tokens first in `web/src/app.css` (`btn--*`, `panel`, `field-*`, new `.modal`, hub/rail classes) before one-off utilities.
- Every worker task: consulting-grok-review PASS before FF-merge (repo standing rule).
- Do not commit large benchmark PNGs unless user explicitly asks; refs live at repo root / `.amp/in/artifacts/` during vibe-pass.
- Ship = push when user allows; commit alone is not ship.

## File map

| Path | Responsibility |
|------|----------------|
| `web/src/app.css` | Shell density, modal, hub two-pane, rail tabs, session composer/copy tokens |
| `web/src/styles-baseline.test.ts` | CSS contract assertions for density/modal/hub/rail/session tokens |
| `web/src/shell/Sidebar.svelte` | Unchanged markup preferred; density via CSS |
| `web/src/shell/Sidebar.test.ts` | Existing nav behavior stays green |
| `web/src/components/Modal.svelte` | Shared native `<dialog>` primitive |
| `web/src/components/Modal.test.ts` | Open/close/Esc/title/children |
| `web/src/components/ProjectRail.svelte` | Right rail: Memory \| Files tabs; Memory chrome; Files tree |
| `web/src/components/ProjectRail.test.ts` | Tab switch; non-persistent memory; tree render |
| `web/src/routes/ProjectHubPage.svelte` | Claude hub: prompt + composer + sessions below + rail; embed SessionChat |
| `web/src/routes/ProjectHubPage.test.ts` | Hub craft + start-session + open-row contracts |
| `web/src/routes/ProjectSessionsPage.svelte` | Legacy: redirect/alias to hub |
| `web/src/routes/ProjectsPage.svelte` | New project → Modal |
| `web/src/routes/VaultsPage.svelte` | New vault → Modal |
| `web/src/routes/VaultProjectsPage.svelte` | Name-first list + Modal create |
| `web/src/components/sessions/SessionChat.svelte` | Continuous rail; dense bottom composer; assistant copy; files → tabs |
| `web/src/components/sessions/SessionChat.test.ts` | Tabs, copy, composer chrome |
| `web/src/components/sessions/SessionChat.focus.test.ts` | Must stay green |
| `web/src/App.svelte` | sessions route → hub if needed |
| `web/src/lib/api/index.ts` | Existing: createProjectSession, sendMessage, listProjectSessions, listModels, listProjectNotes, workspaceTree |
| `web/src/lib/workspace-tree.ts` | Existing hierarchy builder for Files tree |
| `.agents/skills/frontend-ui-craft/SKILL.md` | Benchmark fidelity gate |
| `.agents/skills/frontend-ui-craft/reference/craft.md` | Benchmark vibe-pass detail |

## Canonical contracts

### CSS (Phase A)

```css
.sidebar { width: 240px; padding: 12px 10px; /* was 16px 12px */ }
.sidebar nav {
  display: grid;
  gap: 2px;
  margin: 12px 0;
  flex: 1;
  align-content: start; /* CRITICAL: prevents ~147px stretched rows */
}
.sidebar nav a,
.sidebar__disabled { min-height: 40px; /* 36–40 allowed */ }
.sidebar__collapse { margin-top: auto; }
```

### Modal

```ts
// web/src/components/Modal.svelte
let {
  open = false,
  title,
  onclose,
  children,
}: {
  open?: boolean
  title: string
  onclose: () => void
  children: import('svelte').Snippet
} = $props()
// native <dialog class="modal">; $effect open → showModal() / close()
```

### Hub start session

```ts
// ProjectHubPage — on Send (non-empty draft):
const models = await api.listModels()
const m = models.models[0]
const session = await api.createProjectSession(projectId, {
  home: 'project',
  title: draft.trim().slice(0, 80) || 'Untitled',
  provider: m.provider,
  model_id: m.model_id,
  model_parameters: {},
  tool_grants: { workspace_files: false },
})
await api.sendMessage(session.id, { content: draft.trim(), request_key: crypto.randomUUID() })
// then set activeSession = session (main shows SessionChat)
```

### ProjectRail

```ts
// tabs: 'memory' | 'files' — default 'memory' or last local preference
// Memory fields: bind local state only; label non-persistent; no save button that claims success
// Files: buildHierarchy from notes tree and/or workspaceTree(sessionId) when session + workspace_files
// onOpenFile?: (path: string) => void  // hub may no-op; session opens file tab
```

### SessionChat (supersession)

- Right rail default **open** (not default-closed files bar).
- Agent tab: sticky **bottom** composer (no `Message` label soup).
- Each assistant message container: focusable copy control → `navigator.clipboard.writeText(plain)`.
- File tab active: hide composer without destroying form ancestry.
- Poll: patch messages/run only; composer form stays mounted.

### Legacy route

```ts
// Prefer: route name 'sessions' renders ProjectHubPage with same projectId
// or replace hash to #/projects/:id
```

### Benchmark acceptance (Task 12)

| Ref | Check |
|-----|--------|
| Shell | nav item ≤ 44px @ 1440×900 |
| claude.png | hub prompt top + session rows below; no metric destination grid |
| claude-2.png | vault name-first rows; modal create |
| grok / grok-2 | rail default open; Memory \| Files |
| amp.png | Agent + file tabs; bottom composer; assistant copy |

---

## Tasks

<!-- DRAFTS ASSEMBLED BELOW -->
