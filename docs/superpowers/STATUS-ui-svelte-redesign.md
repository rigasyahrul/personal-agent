# UI Svelte redesign — execution board

**Master owns this file.**  
**Plan:** docs/superpowers/plans/2026-08-19-ui-svelte-redesign.md  
**Handoff:** docs/superpowers/HANDOFF-ui-svelte-redesign-master.md  
**Rules:** Workers = Grok 4.5 only. Review = consulting-grok-review only.

| Phase | Tasks | Depends | Parallel-safe | Status | Branch | Worker thread | Review | Updated |
|-------|-------|---------|---------------|--------|--------|---------------|--------|---------|
| A Tooling + HMR | 1–8 | — | no | done | impl/ui-svelte-phase-A-tooling | [T-01a01a3a…](https://ampcode.com/threads/T-01a01a3a-4307-76ff-8bd6-65817aad385f) **grok45** | [T-01a01a42…](https://ampcode.com/threads/T-01a01a42-acdd-743a-a719-28f204dabc84) YES | 2026-08-19 |
| B Shell + auth | 10–15 | A | no | done | impl/ui-svelte-phase-B-shell-auth | [T-01a01a47…](https://ampcode.com/threads/T-01a01a47-f505-74fe-8f89-5a94ff94e556) **grok45** | [T-01a01a50…](https://ampcode.com/threads/T-01a01a50-27ca-77f9-a0d1-e3dbd04f8018) YES | 2026-08-19 |
| C Global grids | 20–25 | B | no | done | impl/ui-svelte-phase-C-global-grids | [T-01a01a54…](https://ampcode.com/threads/T-01a01a54-0d40-769f-ad58-587595e51efa) **grok45** | [T-01a01cee…](https://ampcode.com/threads/T-01a01cee-528a-7621-8b07-472ae3f98b1d) YES | 2026-08-20 |
| D Vault context | 30–35 | C | no | done | impl/ui-svelte-phase-D-vault-context | [T-01a01cf1…](https://ampcode.com/threads/T-01a01cf1-a728-7684-8005-157a66eb7428) **grok45** | [T-01a01cf8…](https://ampcode.com/threads/T-01a01cf8-bd16-73e3-a3e3-8269decbb40b) YES | 2026-08-20 |
| E Project surfaces | 40–46 | D | no | done | impl/ui-svelte-phase-E-project-surfaces | [T-01a01cfc…](https://ampcode.com/threads/T-01a01cfc-6430-76ad-ad45-be0cefbafa63) **grok45** | [T-01a01d09…](https://ampcode.com/threads/T-01a01d09-bcb7-75a8-bda0-b77afd3f0894) YES | 2026-08-20 |
| F Review + harden | 50–55 | E | no | done | impl/ui-svelte-phase-F-review-harden | [T-01a01d10…](https://ampcode.com/threads/T-01a01d10-8b16-76da-8bc7-93edb154ed30) **grok45** | [T-01a01d20…](https://ampcode.com/threads/T-01a01d20-827d-73cf-b4f0-764e69b778d5) YES | 2026-08-20 |
| F | `78f134f` | Review/settings; remove web-legacy; Go contracts on Vite; docs |

**Status:** todo | running | review | done | blocked

## Log
| When | Event |
|------|--------|
| 2026-08-19 | Master started (thread T-01a01a38-e05a-74e6-9e81-dc97622bab29). Baseline origin/main @ `8911240` (handoff); plan/spec/compound on main (`20b62cd`, `d3a2d3a`, `d962894`). Board created. |
| 2026-08-19 | Phase A dispatched → grok45 worker https://ampcode.com/threads/T-01a01a3a-4307-76ff-8bd6-65817aad385f branch `impl/ui-svelte-phase-A-tooling` (Tasks 1–8). |
| 2026-08-19 | Phase A worker DONE @ `fab962f` — tasks 1–8, branch pushed. |
| 2026-08-19 | Master verify: make web-test + go test ./... green (Node 22). |
| 2026-08-19 | consulting-grok-review YES (no Critical/Important): https://ampcode.com/threads/T-01a01a42-acdd-743a-a719-28f204dabc84 |
| 2026-08-19 | Phase A FF-merged to main @ `fab962f`. |
| 2026-08-19 | Phase B dispatched → grok45 worker https://ampcode.com/threads/T-01a01a47-f505-74fe-8f89-5a94ff94e556 branch `impl/ui-svelte-phase-B-shell-auth` (Tasks 10–15). |
| 2026-08-19 | Phase B worker DONE @ `00bbb46` — tasks 10–15, 40 web tests, branch pushed. |
| 2026-08-19 | Master verify B: make web-test 40/40 + web-build + go packages green (Node 22). |
| 2026-08-19 | consulting-grok-review B YES (no Critical/Important): https://ampcode.com/threads/T-01a01a50-27ca-77f9-a0d1-e3dbd04f8018 |
| 2026-08-19 | Phase B FF-merged to main @ `00bbb46`. |
| 2026-08-19 | Phase C dispatched → grok45 worker https://ampcode.com/threads/T-01a01a54-0d40-769f-ad58-587595e51efa branch `impl/ui-svelte-phase-C-global-grids` (Tasks 20–25). |
| 2026-08-20 | Phase C worker DONE @ `3480acb` — tasks 20–25, 59 web tests, branch pushed. |
| 2026-08-20 | Master verify C: make web-test 59/59 + web-build green (Node 22). |
| 2026-08-20 | consulting-grok-review C YES (no Critical/Important): https://ampcode.com/threads/T-01a01cee-528a-7621-8b07-472ae3f98b1d |
| 2026-08-20 | Phase C FF-merged to main @ `3480acb`. Note: `amp -m grok45 -ox` invalid after CLI update; local `amp -m grok45 -x` works for review/workers. |

| 2026-08-20 | Phase D dispatched → grok45 worker https://ampcode.com/threads/T-01a01cf1-a728-7684-8005-157a66eb7428 branch `impl/ui-svelte-phase-D-vault-context` (Tasks 30–35). |
| 2026-08-20 | Phase D worker DONE @ `3916b92` — tasks 30–35, 79 web tests, branch pushed. |
| 2026-08-20 | Master verify D: 79/79 web tests + build green. consulting-grok-review YES: https://ampcode.com/threads/T-01a01cf8-bd16-73e3-a3e3-8269decbb40b |
| 2026-08-20 | Phase D cherry-picked to main (worker board commit e33daae diverged from master board tip 14dee31; feature-only pick). |
| 2026-08-20 | Phase E dispatched → grok45 worker https://ampcode.com/threads/T-01a01cfc-6430-76ad-ad45-be0cefbafa63 branch `impl/ui-svelte-phase-E-project-surfaces` (Tasks 40–46). Focus invariant required. |
| 2026-08-20 | Phase E worker DONE @ `3c49617` — tasks 40–46, 112 web tests incl. SessionChat.focus, branch pushed. |
| 2026-08-20 | Master verify E: 112/112 + build green. consulting-grok-review YES on focus (Important: hub projectId $effect — fixed on main). https://ampcode.com/threads/T-01a01d09-bcb7-75a8-bda0-b77afd3f0894 |
| 2026-08-20 | Phase E FF-merged to main @ `3c49617` + projectId reactivity fix. |
| 2026-08-20 | Phase F dispatched → grok45 worker https://ampcode.com/threads/T-01a01d10-8b16-76da-8bc7-93edb154ed30 branch `impl/ui-svelte-phase-F-review-harden` (Tasks 50–55). |
| 2026-08-20 | Phase F worker DONE @ `78f134f` — tasks 50–55, 137 web tests, go test ./... green, web-legacy removed, branch pushed. |
| 2026-08-20 | Master verify F: web-test 137/137 + web-build + go test ./... green; prod compose clean. consulting-grok-review YES: https://ampcode.com/threads/T-01a01d20-827d-73cf-b4f0-764e69b778d5 |
| 2026-08-20 | Phase F merged to main @ `78f134f`. **UI Svelte redesign phases A–F complete.** Residual: live `make docker-dev` HMR smoke needs Docker host. |

## Active blockers
_None._

## Master thread
- URL: https://ampcode.com/threads/T-01a01a38-e05a-74e6-9e81-dc97622bab29

## Worker mode rule
Spawn workers with `amp -m grok45 -ox --no-archive-after-execute ...`. High-stakes review = new grok45 thread + `consulting-grok-review`.

## Accepted phases (on origin/main after push)

| Phase | HEAD | Notes |
|-------|------|-------|
| A | `fab962f` | Tooling + docker HMR; Svelte scaffold; PA_UI_DEV_PROXY |
| B | `00bbb46` | Router, shell context, API CSRF, AppShell, auth outside shell, tokens |
| C | `3480acb` | Catalog filters, Home/Projects/Vaults grids, global route wiring |
| D | `3916b92` | Vault pages + locked vault_id create; client sessions/review aggregates; breadcrumbs |
| E | `3c49617`+fix | Project hub/notes/sessions/chat/promote; focus-safe poll; projectId $effect fix |

## Final ship gate (master)

- [x] All phases `done` on board
- [x] `go test ./...` green
- [x] Frontend tests green
- [ ] `make docker-dev` HMR smoke (or documented manual check)
- [x] Prod image build path serves `web/dist`
- [x] `git push`; main not ahead of origin
- [ ] Optional: compound lesson if new traps appeared
