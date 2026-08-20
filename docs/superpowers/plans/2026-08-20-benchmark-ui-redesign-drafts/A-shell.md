# Phase A — Shell nav density (draft)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Worker dispatch (this repo):** `amp -m grok45 --no-archive-after-execute -x '…'` — not Task/OpenAI. Isolate with git worktrees when using local `-x`.
>
> **Assembly:** This draft is Task 1 only. Master assembles into `docs/superpowers/plans/2026-08-20-benchmark-ui-redesign.md`.

**Goal:** Fix stretched primary-nav rows so shell density matches benchmark acceptance A (rows 36–40px, never fill leftover sidebar height).

**Spec:** `docs/superpowers/specs/2026-08-20-benchmark-ui-redesign-design.md` §4 Phase A  
**Lock:** `docs/superpowers/plans/2026-08-20-benchmark-ui-redesign-lock.md`

**Root cause:** `.sidebar nav { flex: 1; display: grid }` receives remaining sidebar height; default grid `align-content: stretch` / row stretch makes each nav item ~147px tall when few items exist.

**Fix (CSS only):** Keep collapse-at-bottom layout (`nav` still `flex: 1` so `.sidebar__collapse` stays at bottom via `margin-top: auto` on the collapse control). Stop row stretch with `align-content: start` on `.sidebar nav`. Keep nav item `min-height` in **36–40px**. Compact sidebar chrome: width **220–240px**, padding **~12×10**.

**Tech / gates:**
- Node `>=22 <23` for web tests
- Tokens first in `web/src/app.css` — no Svelte markup change required for this task
- Do not auto-collapse nav; do not change IA or routes
- No product surfaces beyond shell density

---

### Task 1: Shell nav density fix

**Files:**
- Modify: `web/src/app.css` (`.sidebar`, `.sidebar nav`, nav row rules only as needed)
- Modify: `web/src/styles-baseline.test.ts` (density contract assertions)
- Modify: `web/src/shell/Sidebar.test.ts` only if a behavior assertion is needed; prefer CSS contract in baseline — existing Sidebar tests must stay green
- Test: `web/src/styles-baseline.test.ts`
- Test: `web/src/shell/Sidebar.test.ts`

**Interfaces / contracts (CSS source contracts, string-asserted):**
- `.sidebar` expanded width is `220px`–`240px` inclusive (keep `240px` or tighten within range; collapsed stays `64px`)
- `.sidebar` padding is compact **~12×10** (e.g. `padding: 12px 10px`) — not the current `16px 12px`
- `.sidebar nav` keeps `display: grid` and `flex: 1` (collapse control remains bottom-pinned)
- `.sidebar nav` includes `align-content: start` (or equivalent that packs rows to the start and prevents stretch distribution)
- `.sidebar nav` must **not** use stretch-distributing content alignment (`align-content: stretch` / omit that causes stretch)
- `.sidebar nav a` and `.sidebar__disabled` keep `min-height` in **36px–40px** (current `40px` is valid)
- Brand stays compact; collapse control stays at bottom (`margin-top: auto` on `.sidebar__collapse` unchanged)

**Acceptance A (spec):**
- At 1440×900, every primary nav item height ≤ 44px
- No row/gap visually consumes unused sidebar height

**Out of scope for this task:** hub, rail, modals, session chrome, `web/dist` rebuild (no vibe-pass gate on other surfaces here)

---

- [ ] **Step 1: Write the failing density contract tests**

Add a focused describe (or `it`) to `web/src/styles-baseline.test.ts` that locks the Phase A CSS contracts. Keep all existing baseline tests intact.

```ts
// web/src/styles-baseline.test.ts — add inside describe('visual baseline', …)
// (file already loads `const css = readFileSync(join(here, 'app.css'), 'utf8')`)

it('packs sidebar nav rows without stretch (benchmark Phase A density)', () => {
  // Expanded sidebar chrome: 220–240px width, ~12×10 padding
  expect(css).toMatch(/\.sidebar\s*\{[^}]*width:\s*(220|240|230|225|235)px/s);
  expect(css).toMatch(/\.sidebar\s*\{[^}]*padding:\s*12px\s+10px/s);

  // Nav grows to push collapse down, but rows pack to start (no stretch fill)
  const navBlock = css.match(/\.sidebar nav\s*\{[^}]*\}/);
  expect(navBlock?.[0]).toBeTruthy();
  expect(navBlock![0]).toMatch(/flex:\s*1/);
  expect(navBlock![0]).toMatch(/display:\s*grid/);
  expect(navBlock![0]).toMatch(/align-content:\s*start/);

  // Row min-height stays compact (36–40px)
  expect(css).toMatch(
    /\.sidebar nav a,\s*\n\s*\.sidebar__disabled\s*\{[^}]*min-height:\s*(36|37|38|39|40)px/s,
  );
});
```

Optional hardening in the same test (include if the implementer wants a negative guard):

```ts
// Still inside the same it(…)
// Default grid stretch is the bug; explicit stretch must not reappear on nav
expect(navBlock![0]).not.toMatch(/align-content:\s*stretch/);
```

Do **not** delete or weaken existing Sidebar behavior tests. Confirm `web/src/shell/Sidebar.test.ts` still covers:

- global labels + collapse persistence
- vault context swap
- real SVG icons (no bullet glyphs)
- labeled collapse control
- disabled global Sessions assistive text

No new Sidebar.svelte markup is required for density; CSS-only fix is sufficient.

- [ ] **Step 2: Run tests and verify the new density test fails**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
cd web && npm test -- src/styles-baseline.test.ts src/shell/Sidebar.test.ts
```

**Expected:** FAIL on the new baseline assertion(s). Current `app.css` has:

```css
.sidebar {
  width: 240px;
  /* … */
  padding: 16px 12px; /* too tall vertical pad vs ~12×10 */
}
.sidebar nav { display: grid; gap: 2px; margin: 12px 0; flex: 1; }
/* missing align-content: start → rows stretch to ~147px in tall sidebars */
```

Failure modes (any is fine):

- missing `align-content: start` on `.sidebar nav`
- padding still `16px 12px` (not `12px 10px`)

Existing `Sidebar.test.ts` cases should still PASS (behavior unchanged).

- [ ] **Step 3: Fix sidebar CSS density (minimal change)**

In `web/src/app.css`, update only the shell density rules. Canonical target:

```css
.sidebar {
  width: 240px; /* within 220–240; 240px OK */
  display: flex;
  flex-direction: column;
  padding: 12px 10px; /* was 16px 12px */
  background: var(--sidebar);
  border-right: 1px solid var(--border);
}
.sidebar[data-collapsed='true'] { width: 64px; }

/* … brand / collapsed label rules unchanged … */

.sidebar nav {
  display: grid;
  gap: 2px;
  margin: 12px 0;
  flex: 1;                 /* keep: absorbs free space so collapse stays bottom */
  align-content: start;    /* fix: pack rows; do not stretch items to fill height */
}
.sidebar nav a,
.sidebar__disabled {
  display: flex;
  gap: 10px;
  align-items: center;
  min-height: 40px; /* 36–40 allowed; keep 40 */
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  color: #3f3f46;
}

/* … hover / current / disabled / context unchanged … */

.sidebar__collapse {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 40px;
  margin-top: auto; /* keep bottom pin */
  padding: 8px 10px;
  /* … rest unchanged … */
}
```

**Do not:**

- remove `flex: 1` from `.sidebar nav` (would risk collapse floating up unless another spacer is introduced)
- change nav item markup in `Sidebar.svelte`
- touch mobile `@media` sidebar rules except if a conflict appears (collapsed width / off-canvas transform stay as-is)
- introduce one-off indigo/scaffold classes

**Why this works:** `flex: 1` still gives the nav the free column height; `align-content: start` packs grid tracks to the top so leftover height is empty space *below the last row*, not distributed into each row. Collapse stays bottom via flex column + `margin-top: auto` on `.sidebar__collapse`.

- [ ] **Step 4: Run the same tests and verify they pass**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
cd web && npm test -- src/styles-baseline.test.ts src/shell/Sidebar.test.ts
```

**Expected:** PASS — density contract green; all existing Sidebar tests green.

If baseline fails on the width regex, adjust the test or the CSS so width remains in `220|225|230|235|240`. Prefer keeping `width: 240px` and matching the test to it.

- [ ] **Step 5: Commit**

```bash
git add web/src/app.css web/src/styles-baseline.test.ts web/src/shell/Sidebar.test.ts
git commit -m "$(cat <<'MSG'
fix(web): pack sidebar nav rows for benchmark density

Stop grid stretch on .sidebar nav (align-content: start), tighten
sidebar padding to 12×10, keep 36–40px row min-height and bottom collapse.
MSG
)"
```

Only stage files actually touched. If `Sidebar.test.ts` was not modified, omit it from `git add`.

---

### Task 1 done criteria

- [x] Failing density test written first, then CSS fix, then green
- [x] `.sidebar nav` has `align-content: start` and still `flex: 1` + `display: grid`
- [x] Nav row `min-height` ∈ 36–40px
- [x] Expanded sidebar width ∈ 220–240px; padding `12px 10px`
- [x] Collapse control remains at bottom; no IA/route changes
- [x] `styles-baseline.test.ts` + `Sidebar.test.ts` pass under Node 22
- [x] Commit created

**Manual spot-check (not a separate task):** at ~1440×900, primary nav items read as single dense rows (≤44px), unused sidebar height is empty gap above the collapse control — not fat stretched links.
