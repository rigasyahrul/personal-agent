# Full-surface UI craft polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring every authenticated and auth screen to the same human craft bar as the polished shell/home (hierarchy, density, shared buttons/forms, no indigo-scaffold leftovers), without product-scope expansion.

**Architecture:** Extend `web/src/app.css` tokens (page header, buttons, panels, forms, chips, list rows) already started for shell/home. Restyle each route + shared component to consume those classes. Keep behavior and APIs identical. Spec-as-test per surface group. Browser vibe-pass every route before ship.

**Tech Stack:** Svelte 5 + TypeScript + Vite + Tailwind + Vitest + Testing Library; craft loop from `frontend-ui-craft`.

## Global Constraints

- Visual craft + interaction polish only — no new features, dark mode, IA rewrite, vault move/rename, backend API changes.
- Product look: clean light dashboard, Inter, single accent — `docs/superpowers/specs/2026-08-19-ui-svelte-redesign-design.md`.
- Polled sessions: never destroy focused composer on poll (`AGENTS.md`).
- Web tests: Node `>=22 <23` on PATH.
- Ship = commit + push; prove served bytes when claiming localhost fix.
- Shell/home/hub already polished in `a7901f8` — do not regress.

## Audit baseline (2026-08-20 live)

| Surface | Status | Red flags |
|---------|--------|-----------|
| Shell | Done | icons, collapse, health, sessions hint |
| Home / Vault home / Project hub | Done | metrics vs destinations, btn tokens |
| Projects / Vaults catalogs | **Needs polish** | “Global desk” eyebrow, raw `bg-indigo-600`, form soup |
| Vault projects | **Needs polish** | same as catalogs |
| Review (global/vault/project) + ReviewRunner | **Needs polish** | eyebrow, indigo chips/ratings, rounded-xl soup |
| Settings + BackupSection | **Needs polish** | “Account” eyebrow, indigo Backup now, panel soup |
| Project sessions + SessionList/Chat | **Needs polish** | create form default aesthetic, indigo Send |
| Vault sessions | **Needs polish** | indigo CTA, list row |
| Notes | **Needs polish** | panel soup, tree active indigo |
| Auth (login/bootstrap) | Light polish | already tokenized; align inputs/buttons if drift |
| Cards (Project/Vault), SearchField, Breadcrumbs, Badge | **Needs polish** | destination-card weight, focus rings |

## File map

| File | Responsibility |
|------|----------------|
| `web/src/app.css` | Shared panel, form, chip, list, link, page tokens |
| Catalog routes | ProjectsPage, VaultsPage, VaultProjectsPage |
| Review | ReviewPage, VaultReviewPage, ProjectReviewPage, ReviewRunner |
| Settings | SettingsPage, BackupSection |
| Sessions | ProjectSessionsPage, VaultSessionsPage, SessionList, SessionChat (chrome only) |
| Notes | NotesPage, NoteTree, NoteReader (chrome) |
| Primitives | ProjectCard, VaultCard, SearchField, Breadcrumbs, Badge |
| Auth | AuthCard / Login / Bootstrap if needed |
| Tests | Extend page tests with craft contracts (`btn--primary`, no Global desk where cut) |

---

### Task 1: Shared craft CSS extensions

**Files:**
- Modify: `web/src/app.css`
- Test: `web/src/styles-baseline.test.ts`

- [ ] **Step 1: Extend baseline test for new tokens**

Assert CSS contains `.panel`, `.form-field`, `.scope-chip`, `.list-panel`, `.link-accent`, `.field-input`.

- [ ] **Step 2: Implement tokens**

Add:
- `.panel` / `.panel--dashed` — white surface, border, radius-lg, padding
- `.field-input` / `.field-select` / `.field-textarea` — consistent controls
- `.form-inline` / `.form-stack` — create forms
- `.scope-chip` + `.scope-chip--active` — review chips
- `.list-panel` / `.list-row` — session lists
- `.link-accent` — breadcrumb / ghost text links
- `.alert` / `.alert--error` / `.alert--warn` — inline errors
- `.catalog-grid` — shared card grid gap

- [ ] **Step 3: Run tests + commit**

```bash
export PATH="$HOME/.local/node-v22/bin:$PATH"
cd web && npm test -- src/styles-baseline.test.ts
git add web/src/app.css web/src/styles-baseline.test.ts
git commit -m "style: extend shared craft tokens for full-surface polish"
```

---

### Task 2: Catalog surfaces (Projects, Vaults, Vault projects)

**Files:**
- Modify: `web/src/routes/ProjectsPage.svelte`, `VaultsPage.svelte`, `VaultProjectsPage.svelte`
- Modify: `web/src/components/ProjectCard.svelte`, `VaultCard.svelte`, `SearchField.svelte`
- Test: `ProjectsPage.test.ts`, `VaultsPage.test.ts`, `VaultProjectsPage.test.ts`, `catalog-components.test.ts`

**Craft contract:**
- Drop redundant “Global desk” eyebrow (vault name eyebrow OK on vault projects).
- Header uses `page-header` + `btn btn--primary`.
- Create form uses `panel form-inline` + field-input + primary submit.
- Cards use destination-card visual weight (border hover, not metric quiet).

- [ ] **Step 1: Failing craft tests** — assert no “Global desk”; primary button has `btn--primary`.
- [ ] **Step 2: Implement pages + cards + search**
- [ ] **Step 3: Tests green + commit**

---

### Task 3: Review surfaces

**Files:**
- Modify: `ReviewPage.svelte`, `VaultReviewPage.svelte`, `ProjectReviewPage.svelte`, `components/review/ReviewRunner.svelte`
- Test: `ReviewPages.test.ts`, `ReviewRunner.test.ts`, `VaultReviewPage.test.ts`

**Craft contract:**
- No “Global desk”; vault eyebrow OK.
- Scope chips use `.scope-chip` tokens (active = accent soft).
- Rating primary buttons `btn btn--primary`; suspend secondary.
- Caught-up uses panel--dashed; review card panel without heavy shadow.

- [ ] TDD craft assert + implement + commit

---

### Task 4: Settings + backup

**Files:**
- Modify: `SettingsPage.svelte`, `BackupSection.svelte`
- Test: `SettingsPage.test.ts`, `BackupSection.test.ts`

**Craft contract:**
- Drop “Account” eyebrow or replace with nothing.
- Panels use `.panel`; Backup now `btn--primary`; Retry `btn--secondary`.

---

### Task 5: Sessions surfaces (list + create + vault list)

**Files:**
- Modify: `ProjectSessionsPage.svelte`, `VaultSessionsPage.svelte`, `SessionList.svelte`
- Test: matching `*.test.ts`

**Craft contract:**
- page-header + primary New session.
- Create form panel + field tokens.
- Session list list-panel rows.
- Do **not** change poller/composer structure in SessionChat beyond button/class chrome.

---

### Task 6: Session chat chrome + promote dialog (safe)

**Files:**
- Modify: `SessionChat.svelte` (classes only on shell/composer button), `PromoteDialog.svelte`, `WorkspacePanel.svelte`, `OperationBadges.svelte`
- Test: `SessionChat.test.ts`, `SessionChat.focus.test.ts` must stay green

**Hard rule:** Keep composer form ancestry stable; only swap utility classes for `btn` / `field-textarea` / `panel`.

---

### Task 7: Notes two-pane

**Files:**
- Modify: `NotesPage.svelte`, `NoteTree.svelte`, `NoteReader.svelte` (if needed)
- Test: `NotesPage.test.ts`

**Craft contract:**
- Tree + reader panels; selected tree row accent-soft; empty copy quiet.

---

### Task 8: Breadcrumbs, Badge, Auth alignment

**Files:**
- Modify: `Breadcrumbs.svelte`, `Badge.svelte`, auth pages if needed
- Test: `Breadcrumbs.test.ts`, `AuthPages.test.ts`

---

### Task 9: Full vibe-pass + ship

- [ ] Rebuild `web/dist`; confirm served CSS/JS hashes include tokens.
- [ ] Browser walk: home, projects, vaults, review, settings; create vault+project if empty to hit hub/notes/sessions.
- [ ] `npm test` all green (Node 22).
- [ ] Commit remaining + **push**.
- [ ] Summarize before/after per surface group with portal URL.

## Out of scope

- Dark mode, mobile drawer redesign, new empty-state illustrations, backend health beyond existing `/health`, global Amp skill install.
