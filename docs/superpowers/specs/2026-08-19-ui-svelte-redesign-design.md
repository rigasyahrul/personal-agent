# Design: UI redesign — Svelte 5 + TypeScript + context shell

**Status:** Approved for planning (user sign-off 2026-08-19)  
**Date:** 2026-08-19  
**Stack decision:** Approach 2 — Svelte 5 + TypeScript + Vite + Tailwind  
**Visual language:** Clean dashboard (Vercel / shadcn-inspired), light-first, Inter  
**Backend:** Existing Go API and vault/project model; no product-feature expansion beyond exposing vault UI that the API already supports  

**Related:** `docs/superpowers/specs/2026-08-12-personal-agent-design.md` (domain), `docs/ops/deploy.md` (docker-dev), standing rule: polled SPA must not `innerHTML`-replace a focused composer.

---

## 1. Goals and non-goals

### Goals

1. **Design-first quality** — shell, tokens, and key screens feel like a modern single-owner dashboard before/during implementation.
2. **Replace vanilla JS/HTML UI** with a **Svelte 5 + TypeScript** SPA built by Vite, styled with Tailwind, using Inter.
3. **Smarter layout + UX polish** on the same product surface: empty states, skeletons, clearer errors, keyboard-visible focus, collapsible sidebar.
4. **First-class Vault UX** — create vaults, enter a vault and focus the shell on that vault, create projects inside a vault, vault name badges on project cards. Backend already has vaults; the gap is UI.
5. **Context-switching sidebar** — Global desk vs inside a named vault (e.g. HEALTH).
6. **Home = dashboard**; **Projects** and **Vaults** are their own nav destinations with **searchable card grids**.
7. **Instant feedback in Docker dev** — `make docker-dev` must reflect frontend edits immediately (HMR or equivalent), not only after image rebuild.

### Non-goals

- Multi-user, mobile apps, FTS, full filesystem “vault browser”
- Dark mode as a ship requirement (structure tokens so it can come later)
- Moving projects between vaults / renaming or deleting vaults (API is create+list)
- True `home=global` / `home=vault` agent sessions (schema-ready; out of scope)
- History API requirement — **hash routing stays**
- New review algorithms, promote rules, or auth model changes
- Putting live source mounts on **production** compose (prod stays image-baked)

---

## 2. Product model (vault vs unfiled)

| Term | Meaning |
|------|---------|
| **Vault** | Named container for projects (`GET/POST /api/v1/vaults`). |
| **Project in a vault** | `project.vault_id` set; immutable after create. |
| **Unfiled project** | `project.vault_id` empty/null — lives on the **global** desk, not inside any vault. |

UI copy: prefer plain language. Show a **vault name badge** on project cards when vaulted. Unfiled cards have **no** vault badge (do not invent a fake “Unfiled” vault). Empty lists always use a dedicated empty state + primary action.

---

## 3. App shell and navigation

### 3.1 Authenticated chrome

- **Left sidebar** + **top bar** + **content canvas**.
- **Bootstrap / login:** full-canvas auth card only (no sidebar).
- **Collapse:** user can collapse sidebar (icon rail preferred on desktop; full hide/drawer on small screens) for a wider session workspace. Persist collapsed vs expanded in `localStorage`.
- **Mobile (&lt; ~768px):** sidebar is a drawer; closed after navigate. Session chat + workspace stack vertically.

### 3.2 Two sidebar contexts

**Global context** (default after login; outside any vault)

| Item | Role |
|------|------|
| Home | Global **dashboard** (not the full project catalog) |
| Projects | **Unfiled** projects only — searchable **grid**; New project → `vault_id` null |
| Sessions | Sessions for unfiled projects only |
| Vaults | Searchable **grid** of vaults; enter a vault |
| Review | Review queue (default scope `all` so due work is never hidden globally) |
| Settings | App settings |

**Vault context** (after opening a vault, e.g. HEALTH)

Sidebar **replaces** global items:

| Item | Role |
|------|------|
| Header | Vault name + **Leave vault → Global** (and switcher if cheap) |
| Home | Vault **dashboard** (vault-scoped summary cards) |
| Projects | Projects **in this vault only** — searchable grid; New project locks `vault_id` |
| Sessions | Sessions for this vault’s projects only |
| Review | Due items for this vault’s projects (filter client-side by vault project IDs if API returns broader scope) |
| Settings | Same global settings |

### 3.3 Enter / leave vault

- Global **Vaults** → click card → vault context; URL reflects vault.
- **Leave vault** in sidebar header → global context.
- Opening a **vaulted** project deep link (`#/projects/:id/…`) **auto-enters** that vault context from `project.vault_id`.
- Opening an **unfiled** project keeps or returns to **global** context.
- Context is URL-driven so refresh preserves focus.

### 3.4 Routes (hash)

| Route | Screen |
|-------|--------|
| `#/home` | Global dashboard |
| `#/projects` | Global unfiled projects grid |
| `#/vaults` | Vaults grid |
| `#/vaults/:vaultId` | Vault dashboard (vault Home) |
| `#/vaults/:vaultId/projects` | Vault projects grid |
| `#/vaults/:vaultId/sessions` | Vault-scoped sessions entry (list/filter) |
| `#/vaults/:vaultId/review` | Vault-scoped review |
| `#/projects/:id` | Project hub (shell context from project.vault_id) |
| `#/projects/:id/notes` | Notes |
| `#/projects/:id/notes/:noteId` | Note |
| `#/projects/:id/sessions` | Project sessions |
| `#/projects/:id/review` | Project review |
| `#/review?scope=…` | Global review (existing) |
| `#/settings` or `#settings` | Settings |
| Auth screens | No hash chrome; same setup/login flows |

Legacy hashes used today should keep working or redirect to the table above.

---

## 4. Visual design

### 4.1 Tokens

- **Font:** **Inter** for v1 (load once; `system-ui` fallback).
- **Surfaces:** cool neutral canvas; white panels; sidebar slightly recessed.
- **Borders:** 1px subtle; structure from lines + spacing, not heavy shadows.
- **Accent:** single blue/indigo for primary actions, active nav, focus rings.
- **Semantic:** success / warning / danger as soft tints (status badges, run state, errors).
- **Density:** medium-dense dashboard (not Notion-airy, not admin-cramped).
- **Light-first;** avoid gradients, glass, multi-accent chrome.

### 4.2 Shared components

Buttons (primary / secondary / ghost / destructive), inputs, cards, badges/chips, sidebar nav row, dialog + backdrop, empty state, skeleton, inline alert, optional toast for async OK only, searchable toolbar above grids.

### 4.3 Key screens

**Home (global / vault)**  
Dashboard canvas: quick actions + summary cards from data we already have (due count, recent projects, storage health). Not the full catalog. Empty/zero states are friendly, not blank.

**Projects (global / vault)**  
Card **grid**, client-side **search by name**, richer card meta (notes / sessions / due, updated if available). **Vault name badge** when vaulted. Empty state + New project. Optional soft card tint in v1; heavier backgrounds later without nav changes.

**Vaults (global)**  
Card **grid**, **search by name**, project count on card. Empty state + New vault. Click → enter vault context.

**Project hub**  
Breadcrumbs: vault path when vaulted (`Vaults / {vault} / {project}`), else `Home / {project}` (or Projects). Actions: Notes · Sessions · Review.

**Notes**  
Two-pane desktop (tree + reader); stack on mobile. Empty states for no selection / empty tree. Restyle only; same APIs.

**Sessions**  
List + active session. Split chat | workspace. Sticky composer; **poll patches in place — never replace focused composer**. Promote dialog retained. Setup/grant as warning banner. Skeletons on load. Collapse sidebar for width.

**Review**  
Scope chips; queue card; ratings; suspend/retry; caught-up empty state. Vault context filters to that vault’s projects.

**Settings**  
Single-column sections on cards; backup list + backup now; inline errors/success.

**Bootstrap / Login**  
Centered card; clear labels and focus order.

### 4.4 Cross-cutting UX

| Item | Behavior |
|------|----------|
| Skeletons | Home, project/vault grids, session list, review card, notes |
| Empty states | Every list/grid and caught-up review |
| Errors | Inline near action; page-level only on hard fail |
| Keyboard | Visible focus; Esc closes dialogs |
| Health | Top bar storage pill |
| Search | Client-side filter on loaded Projects and Vaults lists (v1) |

---

## 5. Technical architecture

### 5.1 Stack

- **Svelte 5** + **TypeScript** + **Vite** + **Tailwind CSS**
- Thin hash router (custom or small lib) matching §3.4
- Typed API client over existing `/api/v1` (cookies, CSRF header on mutating methods, `APIError`)
- App stores: auth/session boot, **shell context** (global | vault), sidebar collapsed

### 5.2 Source layout (target)

Frontend lives under a Vite app root (exact folder name fixed at plan time), e.g. `web/` becomes the Vite project with `src/`:

```
src/
  shell/           # sidebar, top bar, collapse, context switch
  routes/          # pages
  lib/api/         # typed client
  lib/stores/      # context, ui prefs
  components/      # primitives
```

Production build output is static files Go already knows how to serve (see §5.4).

### 5.3 Data and filtering

- Reuse: `/api/v1/home`, `/vaults`, `/projects`, sessions, notes, review, backups, auth, setup.
- **No new backend required** for v1 UI redesign. Client filters:
  - Global projects: `!vault_id`
  - Vault projects: `vault_id === current`
  - Vault review: items whose `project_id` ∈ vault’s projects
  - Sessions lists scoped via project membership in context
- If a gap blocks vault-scoped sessions UX, flag during implementation; prefer client filter over schema change.

### 5.4 Go static serving

- **Production:** bake built assets into the image (today: `COPY web` in `deploy/Dockerfile`). Build step must emit the Vite production bundle into the path the binary serves (`http.Dir("web")` or updated equivalent).
- **Local `go run`:** serve either built assets or document a `make`/`npm run dev` proxy workflow; primary DX path is docker-dev (§6).

### 5.5 Session focus invariant

When polling session messages/status: **update the message list and status nodes in place**. Do not remount or rebuild the composer while it is focused. Carry forward regression coverage equivalent to current sessions focus tests.

---

## 6. Docker Compose dev — instant UI updates (required)

### Requirement

When the user runs **`make docker-dev`** (compose base + `docker-compose.dev.yml`), **edits to the Svelte/TS/CSS source must appear in the browser immediately** (HMR or reliable watch), without rebuilding the production image or recreating the container for ordinary UI edits.

### Constraints (standing)

- Production `docker-compose.yml` stays **image-baked** — no host source mounts on prod.
- Live loop = **override only** (`docker-compose.dev.yml`), repo mount `..:/src`, Go reload via **air**.
- Today air **excludes** `web/` and Go serves `http.Dir("web")` from the mount (refresh picks up static files). A Vite SPA changes that: source is compiled.

### Design choice (mandatory for implementation plan)

Implement **one** of the following so DX stays instant; prefer **A**:

**A — Vite dev server inside the dev container (recommended)**  
- Dev image includes Node.  
- Process supervisor or second process: Vite dev (HMR) + air for Go.  
- Browser hits API on `:8080`; Go **proxies** non-API routes to Vite in dev, **or** Vite proxies `/api` and `/health` to Go and the published port is Vite’s — pick one in the plan and document URLs.  
- HMR websocket must work through the published port.

**B — Vite build --watch writing into the directory Go serves**  
- Watcher writes hashed/static output to e.g. `web/dist` or `web/` serve root.  
- Go keeps `FileServer`; browser hard-refresh or light live-reload plugin.  
- Weaker than HMR but acceptable if A is blocked.

### Non-negotiables

- `make docker-dev` remains the documented one-command loop for API **and** UI.
- Docs (`docs/ops/deploy.md`) updated: how UI HMR/watch works, ports, and “prod has no live mounts.”
- Deploy tests continue to forbid live mounts on **production** compose.
- Claim “UI fix works on localhost:8080” only after verifying the process that listens and that served bytes match the edit (existing standing rule).

---

## 7. Testing strategy

| Layer | What |
|-------|------|
| Unit | Router/context helpers, search/filter, API error parsing |
| Component | Empty states, project card vault badge, sidebar global vs vault |
| Session | Focus regression: poll must not destroy focused composer |
| Go | Existing API/acceptance tests stay green; update static path assertions that pin old `web/js/...` strings |
| Deploy | Dockerfile.dev / compose.dev mention Node+Vite or watch as chosen; prod compose still baked |
| Manual | Collapse sidebar; create vault HEALTH; enter; create project; badge visible; search grids; leave vault; docker-dev HMR |

---

## 8. Implementation phasing (for writing-plans)

1. **Tooling** — Vite+Svelte+TS+Tailwind scaffold; Makefile targets; docker-dev instant UI path (§6); prod build copy path.  
2. **Shell** — tokens, Inter, sidebar contexts, collapse, top bar, auth screens.  
3. **Global** — Home dashboard, Projects grid+search+empty, Vaults grid+search+create+enter.  
4. **Vault context** — vault Home/Projects/Sessions/Review scoping; breadcrumbs; locked vault on create project.  
5. **Project surfaces** — hub, notes, sessions (focus-safe), promote dialog restyle.  
6. **Review + Settings** — restyle + vault filter; backup UX.  
7. **Hardening** — migrate/remove old vanilla assets; fix Go web tests; docs; verification on docker-dev and prod build.

---

## 9. Decisions log (brainstorming)

| Decision | Choice |
|----------|--------|
| Primary goal framing | Design quality first; then TS SPA (user path: visual confidence → stack B) |
| Framework | Svelte 5 + TS + Vite + Tailwind (Approach 2) |
| Look | Clean dashboard (Vercel/shadcn-like), not soft-Notion or dense-Linear-only |
| Scope | Same features + smarter layout + UX polish + Vault UI |
| Font | Inter from v1 |
| Home | Dashboard, not project list |
| Projects / Vaults | Own nav items; **card grids**; searchable |
| Unfiled | No vault; empty states; badge only when vaulted (vault **name**) |
| Sidebar | Context switch Global vs Vault; collapsible |
| Sessions in sidebar | Scoped lists via projects in context — not new session homes |
| Review global | `scope=all` by default |
| Routing | Hash |
| Docker dev | Instant UI updates required (§6) |

---

## 10. Success criteria

1. Owner can create a vault, enter it, create a project there, see the **vault name badge**, and work only with that vault’s projects/sessions/review from the sidebar.  
2. Global Projects shows only unfiled projects; empty and non-empty UX are clear.  
3. Authenticated pages share one shell; sidebar can collapse; auth pages have no shell.  
4. Session chat polling never steals composer focus.  
5. `make docker-dev`: UI source edit appears without image rebuild.  
6. Production image serves the Vite build; `go test ./...` and acceptance remain green.  
7. Visual bar: quiet surfaces, Inter, one accent, grids/search/empty/skeletons as specified.

---

## 11. Open points deferred to implementation plan (not product unknowns)

- Exact Vite app root path and serve directory name.  
- Compose process model detail for §6 option A vs B.  
- Whether vault Sessions page is a filtered multi-project list or requires picking a project first (prefer useful list if API allows).  
- shadcn-svelte vs hand-rolled primitives — either is fine if visual language matches §4.
