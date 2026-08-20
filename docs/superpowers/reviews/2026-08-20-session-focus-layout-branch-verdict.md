## Verdict
YES — `impl/session-focus-layout` (`b355e55..c770f7e`) is ready to merge to main from a correctness/spec standpoint: composer focus holds under poll and file-tab switches, assistant is bare markdown (no bubble / no “Assistant” label), files bar + 8-tab LRU / promote-from-file-tab match the locked design, prefs clamp and keys are correct, markdown is XSS-sanitized, and there are **no Critical or Important residual bugs** on the session-focus surface under the inspected paths.

## Evidence
### Composer focus invariant
- `web/src/components/sessions/SessionChat.svelte:595–618` — sticky `<form>` is a **sibling after** the `{#if agentActive}` message list / `{#if activeFileTab}` body; never inside those branches. When a file tab is active: `hidden` + `inert` + `.session-composer--hidden` only.
- `web/src/app.css` (`.session-composer--hidden`) — `display: none !important` with explicit comment to keep the form in the DOM for focus/draft stability.
- `SessionChat.svelte:124–134, 172–173` — `applySnapshot` / exported `poll()` only patch `messages` / `run` (and poll error flags); no shell rebuild.
- `SessionChat.focus.test.ts:50–86` — after `poll()`, same textarea node identity, `document.activeElement`, value `"typing here"`, selection `[6,11]`, plus patched reply + run status.
- `SessionChat.test.ts:413–414, 462–469` — form remains mounted on file tab; draft survives file → Agent.
- Targeted run this review: `SessionChat.focus.test.ts` **PASS**.

### Assistant bare markdown (no label / no bubble)
- `SessionChat.svelte:559–568` — assistant → `.message-prose` + `MarkdownView`; no bubble wrapper; no role label.
- `SessionChat.svelte:549–557` — user → `.message-bubble--user` (end-aligned via `.message-row--user` CSS).
- `SessionChat.svelte:569–580` — tool/other muted meta; does not use “Assistant”.
- `rg Assistant SessionChat.svelte` → no matches.
- `SessionChat.test.ts:126–146` — asserts no text `"Assistant"`, no `.message-bubble` on assistant, `.message-prose` present, user bubble only.

### Files bar + tabs LRU / promote
- `SessionChat.svelte:36, 81–84, 278–304, 320–325` — `FILE_TAB_CAP = 8`; `fileTabLru` is file paths only (“Agent never participates”); reuse path; at cap evict `fileTabLru[0]`; `selectFileTab` bumps LRU (least-recently-**activated**).
- `SessionChat.svelte:500–507` — Agent tab always present, not closable, outside `openFileTabs`.
- `SessionChat.test.ts:432–448` — 9th path closes `f0.md`; tablist stays Agent + 8 files; path reuse has no duplicates.
- `SessionFilesBar.svelte` — tree + search only; directories disabled / no open; no preview `<pre>`; no promote CTA (`SessionFilesBar.test.ts:97–102`).
- `SessionFileTab.svelte:76–107` — Preview | Source (Preview default); md → `MarkdownView`; else monospace; “Save to source” only when promotable + loaded.
- `SessionChat.svelte:584–592, 654–666` — `onpromote={openPromote}` only on file tab → existing `PromoteDialog`.
- `promote.ts:47–53` + `SessionFileTab.svelte:44–47` (harden `c222cb6`) — missing/`''` kind + `.md` promotable; directories never; covered by `promote.test.ts` + `SessionFileTab.test.ts`.

### Prefs clamp
- `session-prefs.ts` — keys `pa.session.filesBarOpen` / `pa.session.filesBarWidthPct`; default open false (`!== '1'`); default main `70`; `clampMainPct` 50–85; garbage → 70; null storage safe.
- `SessionChat.svelte:341–374, 418–419` — toggle/close write open; drag move clamps; pointer-up writes width; `resetForSession` reads both.
- `session-prefs.test.ts` — defaults, 1/0, clamp, null storage **PASS** this review.

### XSS-safe markdown + Mermaid
- `render.ts` — `markdown-it` `{ html: false }` + link scheme allowlist (http/https/mailto + relative) + `rel=noopener noreferrer` + `DOMPurify.sanitize` (`USE_PROFILES.html`, `ADD_ATTR: ['target']` only).
- `render.test.ts` — headings/lists, strips `<script>`, fenced code, mermaid class marker **PASS**.
- `MarkdownView.svelte` — lazy dynamic `import('mermaid')`, `securityLevel: 'strict'`, failure → `pre.mermaid-fallback` with source (never blank); `MarkdownView.test.ts` **PASS**.

### Layout / lists / constraints
- Files default closed; toggle gated by `workspaceEnabled(session)` / `workspace_files` (`SessionChat.svelte:79, 479–486, 621`).
- Desktop split `--session-main-pct` + handle; narrow `<1024px` drawer + backdrop + Escape (`SessionChat.svelte:92, 432–458, 621–649`; tests cover Escape/backdrop/pref write).
- `ProjectSessionsPage.svelte:104–108` — `.content-canvas--session-focus` only when session open; hash routing unchanged.
- Vault + project lists use `SessionCardRow` / `.session-card*`; relative time only via `formatRelativeTime` when `created_at`/`updated_at` present.
- `git diff --name-only … | rg '\\.go$'` → **no Go files**; no API/auth product changes in range.
- Per-task board `reviews/BOARD.md`: Tasks 1–11 all PASS, Critical none, Important none.

## If wrong: failing sequence / invariant break
N/A — holds under inspected paths for the merge-gate contracts.

Counterfactuals that would have failed (and do not):
1. **Focus:** Poll or file-tab switch destroys/recreates `<form>` / textarea → focus test loses node identity or selection.
2. **Assistant:** Row still renders `"Assistant"` or `.message-bubble` around agent content.
3. **LRU:** Opening a 9th path closes Agent or a non-LRU file tab, or Agent counts toward cap.
4. **Promote:** “Save to source” remains on the files bar, or missing API `kind` hides promote on real `.md` workspace files.
5. **Prefs:** Width stored unclamped / open defaults true / wrong keys.
6. **XSS:** `{@html}` path allows raw `<script>` or executable handlers from assistant/file markdown.
7. **Scope creep:** Go handlers or auth/routing changed in this range.

## Smallest fix
none (for Critical/Important). Optional polish only — see Minor.

## Severity of residual issues
### Critical
- none

### Important
- none

### Minor
- **Agent message-list scroll** resets when switching to a file tab and back: messages live in `$state` but the `<ol class="session-focus__messages">` is under `{#if agentActive}` so the scroller DOM unmounts (`SessionChat.svelte:546–583`). Spec §4.2 asks scroll to survive file-tab switches; draft + composer mount (the standing hard rule) do survive. Not a data/security break; best fixed by keeping the message scroller mounted and hiding it (same pattern as the composer) if polish is desired later.
- **Inactive file-tab content** is not parent-cached: only the active path mounts `SessionFileTab`, which refetches on (re)mount. Per-tab **mode** is kept in `openFileTabs`; content/scroll for background file tabs is best-effort (spec allows best-effort scroll).
- **Files tree** is always fully flattened/expanded; directory rows are disabled rather than expand/collapse toggles (spec’s “notes-like expand/collapse” is only partially met; directories still do not open tabs).
- **Tab close control** is a `role="button"` `<span>` nested inside the tab `<button>` (`SessionChat.svelte:510–535`) — works with `stopPropagation` but is imperfect a11y (nested interactive roles).
- No SessionChat-level integration test that drag-end persists `pa.session.filesBarWidthPct` (module clamp + write path are covered; optional).
- Vault session cards still link to the project sessions **list** (`#/projects/{id}/sessions`), not a deep-linked session id — **same as pre-branch Open behavior**, not a regression; out of v1 if deep-link was never in scope.

## What would reverse this verdict
- A failing `SessionChat.focus.test.ts` (composer node/selection lost after poll) or moving the composer `<form>` inside `{#if agentActive}` / any poll-driven remount.
- Assistant rows showing text `"Assistant"` or a bubble wrapper again.
- File-tab cap ≠ 8, Agent counted in LRU/cap, or promote CTA returning on the files bar.
- `isPromotableWorkspaceFile` rejecting missing-kind `.md` (or allowing directories).
- Prefs keys/defaults/clamp diverging from `pa.session.filesBarOpen` / `filesBarWidthPct`, closed default, 70, 50–85.
- XSS proof: executable script/handler surviving `renderMarkdownToSafeHtml` into `{@html}`.
- Product Go/API/auth changes landing in this branch range.
- Elevating the message-list scroll unmount to a hard ship blocker (would require a product call; not warranted as Critical/Important on current standing rules).

## Explicitly out of scope
- Pure docs/memory compounding wording and AGENTS process bullets in the tip commits.
- Style nits / visual craft taste beyond token presence and structural contracts.
- Go backend, auth, hash-router redesign, `workspace_files` grant semantics beyond the existing client gate.
- Dark mode; Amp Changes/Portal/Terminal modes; edit-in-tab; keyboard tab-strip shortcuts beyond basic focusability.
- Full 205-test suite re-run in this review turn (parent reported 205 passed; this review re-ran the invariant-critical subset: focus, markdown render, MarkdownView, prefs, promote — all pass).
- Pre-existing vault→project list deep-link limitations unchanged by this branch.
- Whether inactive file-tab content caching or agent scroll restoration should be upgraded in a follow-up.
