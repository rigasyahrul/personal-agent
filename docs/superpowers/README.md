# Superpowers artifacts

Design specs and implementation plans from Superpowers skills.

| Path | What |
|------|------|
| `specs/` | Design docs from brainstorming (`YYYY-MM-DD-*-design.md`) |
| `plans/` | Implementation plans from writing-plans (`YYYY-MM-DD-*.md`) |
| `plans/*-lock.md` | Optional shared lock when a plan is drafted in parallel |
| `plans/*-drafts/` | Optional phase drafts (scratch); final plan is the `*.md` without `-drafts` |
| `HANDOFF-*.md` | Thread handoffs |
| `HANDOFF-master-execution.md` | **Master coordinator (v1)** — spawn workers, board, ship gate |
| `STATUS-v1.md` | Execution board for v1 phases — **complete** on main |
| `HANDOFF-ui-svelte-redesign-master.md` | **Master coordinator (UI Svelte redesign)** |
| `STATUS-ui-svelte-redesign.md` | UI redesign board (phases A–F) — **complete** on main |
| `specs/2026-08-19-ui-svelte-redesign-design.md` | UI redesign design (historical + contracts) |
| `plans/2026-08-19-ui-svelte-redesign.md` | UI redesign plan (+ lock) |

**Owner-facing product docs** (routes, vaults, daily use) live under `docs/manual/` — keep those current when UI behavior ships; do not treat frozen specs as the only handbook.

**Lessons / compounding notes** are not here — index `docs/memory/lessons.md`, entries `docs/memory/YYYYMMDD-HHmm-*.md`.
