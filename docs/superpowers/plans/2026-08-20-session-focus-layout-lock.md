# Plan lock: Session Focus Layout

**Spec (source of truth):** `docs/superpowers/specs/2026-08-20-session-focus-layout-design.md`  
**Assembled plan (target):** `docs/superpowers/plans/2026-08-20-session-focus-layout.md`  
**Drafts dir:** `docs/superpowers/plans/2026-08-20-session-focus-layout-drafts/`  

## Scope freeze

In scope (Approach 1 — session surface pack only):
1. Session focus shell: Agent + file tabs, files right bar (tree/search), 70/30 resizable split
2. Markdown + Mermaid for assistant messages and file Preview
3. Assistant: no bubble, no “Assistant” label; user keeps end-aligned bubble
4. Vault + project session list card rows
5. Composer focus/poll invariant preserved

Out of scope: global shell redesign, auto-collapse sidebar, notes redesign, new APIs, edit-in-tab, Amp Changes/Portal/Terminal modes, focus left history rail, dark mode.

## Draft file list (disjoint task ranges)

| Draft | File | Tasks | Owner focus |
|-------|------|-------|-------------|
| A | `…-drafts/A-foundation.md` | 1–8 | CSS tokens, localStorage prefs helpers, content-canvas focus modifier |
| B | `…-drafts/B-markdown.md` | 10–18 | markdown-it/marked + DOMPurify + mermaid, MarkdownView component + tests |
| C | `…-drafts/C-focus-shell.md` | 20–32 | SessionChat shell: tabs, split, files toggle, Agent tab layout wiring |
| D | `…-drafts/D-files-tabs.md` | 40–52 | Files bar tree/search; file tab Preview/Source/promote; WorkspacePanel evolution |
| E | `…-drafts/E-lists-harden.md` | 60–72 | Session card rows (vault+project); focus invariant; craft gates; final verify |

## Authority rules

1. **Spec wins** over drafts on any conflict.
2. **Canonical contracts** in the assembled plan header win over draft prose.
3. Drafts must use checkbox steps, real file paths under `web/`, TDD (failing test → impl → pass → commit).
4. Node `>=22 <23` on PATH for all web tests.
5. Do not touch Go API, auth, review algorithms.
6. Standing: poll never remounts focused composer.
7. Worker dispatch: `amp -m grok45 --no-archive-after-execute -x '…'` (not Task/OpenAI).
8. Ship = push after implementation phases; this lock/plan commit is docs only until user chooses execution.

## File map (expected)

| Path | Role |
|------|------|
| `web/src/app.css` | `.session-focus*`, tabs, split handle, files bar, `.message-prose`, `.session-card` |
| `web/src/lib/session-prefs.ts` | Read/write `pa.session.filesBarOpen`, `pa.session.filesBarWidthPct` |
| `web/src/components/markdown/MarkdownView.svelte` | Shared sanitized markdown + mermaid |
| `web/src/components/sessions/SessionChat.svelte` | Focus shell owner |
| `web/src/components/sessions/SessionFilesBar.svelte` | Tree + search (from WorkspacePanel) |
| `web/src/components/sessions/SessionFileTab.svelte` | Preview/Source + promote CTA |
| `web/src/components/sessions/SessionList.svelte` | Card rows |
| `web/src/components/sessions/SessionCardRow.svelte` | Shared card presentational |
| `web/src/routes/VaultSessionsPage.svelte` | Use cards |
| `web/src/routes/ProjectSessionsPage.svelte` | Use cards; focus entry unchanged |
| Tests | co-located `*.test.ts` + keep `SessionChat.focus.test.ts` green |

## Parallel draft instructions

Each draft agent writes ONLY its draft file with full Task N sections (steps with code, run commands, commits). No assembled plan. No UI implementation.
