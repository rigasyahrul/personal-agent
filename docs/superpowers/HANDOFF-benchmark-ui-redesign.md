# Handoff: Benchmark UI redesign execution

**Date:** 2026-08-21  
**Status:** Spec + plan committed; **implementation not started**  
**Coordinator prompt:** `docs/superpowers/PROMPT-benchmark-ui-redesign-coordinator.md`

## Authority

| Artifact | Path |
|----------|------|
| Design | `docs/superpowers/specs/2026-08-20-benchmark-ui-redesign-design.md` |
| Plan (12 tasks) | `docs/superpowers/plans/2026-08-20-benchmark-ui-redesign.md` |
| Lock | `docs/superpowers/plans/2026-08-20-benchmark-ui-redesign-lock.md` |
| Refs | repo root or `.amp/in/artifacts/`: `claude.png`, `claude-2.png`, `grok.png`, `grok-2.png`, `amp.png` |

## Execution model

- **Master/coordinator** owns board, merge, push, task ledger.
- **Workers:** `amp -m grok45 --no-archive-after-execute -x '…'` one at a time (or worktree-isolated).
- **Every task:** implement → tests → **new** consulting-grok-review thread → fix Critical+Important → ledger `Task N: consulting-grok-review PASS (T-…)` → then `Task N: complete` → FF-merge.
- Run `.agents/skills/subagent-driven-development/scripts/check-review-gate progress.md` before claiming plan done.
- Whole-branch consulting-grok-review before ship.
- Ship = push.

## Progress ledger (start empty)

Create `docs/superpowers/STATUS-benchmark-ui-redesign.md` on first coordinator turn.

## Do not

- Skip consulting-grok-review (“tests green = reviewed”)
- Use Task/OpenAI/oracle as Grok
- Use `amp -m grok45 -ox`
- Claim UI done without dist rebuild + cache-bust vibe-pass vs named refs
