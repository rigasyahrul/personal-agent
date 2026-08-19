# Plan lock: UI Svelte redesign

**Spec (source of truth):** `docs/superpowers/specs/2026-08-19-ui-svelte-redesign-design.md`  
**Assembled plan (target):** `docs/superpowers/plans/2026-08-19-ui-svelte-redesign.md`  
**Drafts dir:** `docs/superpowers/plans/2026-08-19-ui-svelte-redesign-drafts/`  
**Status:** assembled — awaiting user approval  

## Authority

1. Approved design spec wins over any draft snippet.  
2. **Canonical contracts** section in the assembled plan wins over stale task code.  
3. No application feature code until the assembled plan is user-approved and execution starts.  
4. Production compose stays image-baked; live UI = `make docker-dev` only.  

## Scope freeze (do not re-brainstorm)

- Svelte 5 + TS + Vite + Tailwind + Inter  
- Context sidebar: Global vs Vault  
- Home = dashboard; Projects + Vaults = searchable card grids  
- Vault name badge on vaulted projects; unfiled = no vault  
- Hash routes; existing Go API only  
- docker-dev: instant UI (Vite HMR preferred)  
- Session poll must not steal composer focus  

## Draft split

| Draft file | Owner content |
|------------|----------------|
| `00-header-canonical.md` | Plan header, global constraints, file map, canonical contracts |
| `01-tooling-docker.md` | Tasks: Vite scaffold, Makefile, Dockerfile prod/dev, HMR, deploy tests |
| `02-shell-auth.md` | Tasks: tokens, shell, context store, router, auth screens |
| `03-global-vault-grids.md` | Tasks: Home dashboard, Projects/Vaults grids, search, empty, create |
| `04-vault-context.md` | Tasks: enter/leave vault, vault-scoped pages, breadcrumbs |
| `05-project-surfaces.md` | Tasks: hub, notes, sessions (focus-safe), promote, workspace |
| `06-review-settings-harden.md` | Tasks: review, settings, remove vanilla, Go web tests, docs gate |

## Assembly rule

Concatenate drafts in order → one plan → self-review vs spec → commit plan only.
