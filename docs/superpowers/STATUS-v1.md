# personal-agent v1 — execution board

**Master owns this file.** Workers do not edit status except via report (master updates).  
**Plan:** `docs/superpowers/plans/2026-08-12-personal-agent-v1.md`  
**Handoff:** `docs/superpowers/HANDOFF-master-execution.md`

| Phase | Tasks | Depends | Parallel-safe | Status | Branch | Worker thread | Updated |
|-------|-------|---------|---------------|--------|--------|---------------|---------|
| 1 Skeleton | 1–8 | — | no | done | impl/v1-p1-skeleton | [T-019ff854…](https://ampcode.com/threads/T-019ff854-0162-7096-abde-b22632804e47) | 2026-08-13 |
| 2 Projects + source | 9–14 | 1 | no | done | impl/v1-p2-projects | [T-019ff890…](https://ampcode.com/threads/T-019ff890-519f-71ec-bea9-b3983d42fbcf) | 2026-08-13 |
| 3 Sessions + chat | 15–20 | 2 | no | done | impl/v1-p3-sessions | [T-019ff8ea…](https://ampcode.com/threads/T-019ff8ea-6a0b-77fe-91e5-09fbb64e6678) | 2026-08-13 |
| 4 Workspace tools | 21–24 | 3 | no | done | impl/v1-p4-tools | [T-019ff945…](https://ampcode.com/threads/T-019ff945-5949-7619-946a-e050c40d177f) | 2026-08-13 |
| 5 Promote + review | 25–32 | 4 | no | done | impl/v1-p5-promote-review | [T-019ff978…](https://ampcode.com/threads/T-019ff978-ac06-73fc-9f51-d4c8fcff2854) | 2026-08-13 |
| 6 Backup | 33–36 | 5 | no | done | impl/v1-p6-backup | [T-019ffad8…](https://ampcode.com/threads/T-019ffad8-acf3-712c-b091-a1b91d0ae257) **grok45** | 2026-08-19 |
| 7 Hardening | 37–42 | 6 | no | running | impl/v1-p7-hardening | [T-01a0181b…](https://ampcode.com/threads/T-01a0181b-8ad0-71bc-95d2-c8912ef20238) **grok45** | 2026-08-19 |

**Status:** `todo` | `running` | `review` | `done` | `blocked`

## Log

| When | Event |
|------|--------|
| 2026-08-13 | Phases 1–5 DONE and FF-merged to main. |
| 2026-08-13 | Phase 6 high-mode workers failed (ChatGPT limits). Root cause: use **grok45**. |
| 2026-08-13 | Phase 6 grok45 worker started: T-019ffad8… |
| 2026-08-19 | Phase 6 DONE @ `84ef54b` (9 commits). consulting-grok-review YES after ctx fix `57ae777`. Master `go test ./...` green. FF-merged to main. |
| 2026-08-19 | Dispatching Phase 7 Hardening (tasks 37–42) on **grok45**. |
| 2026-08-19 | Phase 7 worker: https://ampcode.com/threads/T-01a0181b-8ad0-71bc-95d2-c8912ef20238 |

## Active blockers

_None._

## Master thread

- URL: https://ampcode.com/threads/T-019ff850-de59-752f-b240-f3e790a566cc

## Worker mode rule

Spawn workers with `amp -m grok45 -ox ...` (plugin `.amp/plugins/grok-45-mode.ts`).  
High-stakes review = new grok45 thread + `consulting-grok-review` (not built-in oracle / not Task→OpenAI).

## Accepted phases (on origin/main)

| Phase | HEAD | Notes |
|-------|------|-------|
| 1 | `5b7a0ce` | Skeleton, auth, home shell, deploy |
| 2 | `0b79b00` | Layout, projects, notes, direct publish, UI |
| 3 | `5ac9dfc` | Sessions, runner, chat API/UI |
| 4 | `c8cddc6` | Rooted workspace tools, tool loop, API, UI panel |
| 5 | `85172f2` | Promote machine, SM-2, bites, review queue/UI |
| 6 | `84ef54b` | Backup barrier, local dir bundles, optional S3, settings UI, restore docs |

## Phase 6 acceptance (master)

- Worker: DONE, merge-ready (scoped Grok review YES)
- Branch: `impl/v1-p6-backup` @ `84ef54b`
- Master verify: `go test ./... -count=1` PASS; build OK
- Merged: FF `main` ← `origin/impl/v1-p6-backup`
- Note: do not wrap POST /api/v1/backups in Barrier.Mutate (deadlock with Snapshot)
