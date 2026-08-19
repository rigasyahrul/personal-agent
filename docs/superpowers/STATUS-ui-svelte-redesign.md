# UI Svelte redesign — execution board

**Master owns this file.**  
**Plan:** docs/superpowers/plans/2026-08-19-ui-svelte-redesign.md  
**Handoff:** docs/superpowers/HANDOFF-ui-svelte-redesign-master.md  
**Rules:** Workers = Grok 4.5 only. Review = consulting-grok-review only.

| Phase | Tasks | Depends | Parallel-safe | Status | Branch | Worker thread | Review | Updated |
|-------|-------|---------|---------------|--------|--------|---------------|--------|---------|
| A Tooling + HMR | 1–8 | — | no | running | impl/ui-svelte-phase-A-tooling | [T-01a01a3a…](https://ampcode.com/threads/T-01a01a3a-4307-76ff-8bd6-65817aad385f) **grok45** | | 2026-08-19 |
| B Shell + auth | 10–15 | A | no | todo | | | | |
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

## Active blockers
_None._

## Master thread
- URL: https://ampcode.com/threads/T-01a01a38-e05a-74e6-9e81-dc97622bab29

## Worker mode rule
Spawn workers with `amp -m grok45 -ox --no-archive-after-execute ...`. High-stakes review = new grok45 thread + `consulting-grok-review`.

## Accepted phases (on origin/main after push)

| Phase | HEAD | Notes |
|-------|------|-------|
| _(none yet)_ | | |

## Final ship gate (master)

- [ ] All phases `done` on board
- [ ] `go test ./...` green
- [ ] Frontend tests green
- [ ] `make docker-dev` HMR smoke (or documented manual check)
- [ ] Prod image build path serves `web/dist`
- [ ] `git push`; main not ahead of origin
- [ ] Optional: compound lesson if new traps appeared
