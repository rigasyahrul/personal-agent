# UI Svelte redesign — execution board

**Master owns this file.**  
**Plan:** docs/superpowers/plans/2026-08-19-ui-svelte-redesign.md  
**Handoff:** docs/superpowers/HANDOFF-ui-svelte-redesign-master.md  
**Rules:** Workers = Grok 4.5 only. Review = consulting-grok-review only.

| Phase | Tasks | Depends | Parallel-safe | Status | Branch | Worker thread | Review | Updated |
|-------|-------|---------|---------------|--------|--------|---------------|--------|---------|
| A Tooling + HMR | 1–8 | — | no | done | impl/ui-svelte-phase-A-tooling | [T-01a01a3a…](https://ampcode.com/threads/T-01a01a3a-4307-76ff-8bd6-65817aad385f) **grok45** | [T-01a01a42…](https://ampcode.com/threads/T-01a01a42-acdd-743a-a719-28f204dabc84) YES | 2026-08-19 |
| B Shell + auth | 10–15 | A | no | done | impl/ui-svelte-phase-B-shell-auth | [T-01a01a47…](https://ampcode.com/threads/T-01a01a47-f505-74fe-8f89-5a94ff94e556) **grok45** | [T-01a01a50…](https://ampcode.com/threads/T-01a01a50-27ca-77f9-a0d1-e3dbd04f8018) YES | 2026-08-19 |
| C Global grids | 20–25 | B | no | todo | | | | |
| D Vault context | 30–35 | C | no | todo | | | | |
| E Project surfaces | 40–46 | D | no | todo | | | | |
| F Review + harden | 50–55 | E | no | todo | | | | |

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

## Final ship gate (master)

- [ ] All phases `done` on board
- [ ] `go test ./...` green
- [ ] Frontend tests green
- [ ] `make docker-dev` HMR smoke (or documented manual check)
- [ ] Prod image build path serves `web/dist`
- [ ] `git push`; main not ahead of origin
- [ ] Optional: compound lesson if new traps appeared
