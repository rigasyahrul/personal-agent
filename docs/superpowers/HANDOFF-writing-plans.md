# Handoff: writing-plans (fresh thread)

**From:** Amp thread brainstorming Superpowers + personal-agent design  
**Date:** 2026-08-12  
**Goal of next thread:** Write the v1 **implementation plan** only (no feature code yet).

---

## Start prompt (paste into new thread)

```
Use Superpowers. Load using-superpowers, then writing-plans.

Approved design spec (source of truth):
docs/superpowers/specs/2026-08-12-personal-agent-design.md

Handoff context:
docs/superpowers/HANDOFF-writing-plans.md

Task: Write the implementation plan for personal-agent v1 thin vertical.
Save to: docs/superpowers/plans/2026-08-12-personal-agent-v1.md

Do NOT implement application code. Plan only. Follow writing-plans exactly
(TDD, bite-sized tasks, checkbox steps, commit per task). Assume greenfield
repo except orb setup, Grok plugin, Superpowers skills, and this spec.
```

---

## What is already done

| Item | Location |
|------|----------|
| Superpowers skills installed | `.agents/skills/*` |
| Session bootstrap | `AGENTS.md` |
| Approved design spec | `docs/superpowers/specs/2026-08-12-personal-agent-design.md` |
| Spec commit | `2db5455` (+ follow-up commits for skills/bootstrap/status if present) |
| Orb lifecycle | `.agents/setup` (Go), `.agents/resume` |
| Grok 4.5 mode plugin | `.amp/plugins/grok-45-mode.ts` |

**No application code yet** (no `go.mod`, no app packages). Greenfield build from the spec.

---

## Product summary (do not re-brainstorm)

Self-hosted **single-tenant** learning dashboard:

- **Browser** → Docker Compose on **one host** (laptop or VPS + domain)
- **SoT:** local data volume (SQLite + files). **S3 = optional backup only**
- **Project** ± optional **vault**; freeform `source/**` tree (folders + `.md`)
- **Session** = chat + **workspace directory** on disk (life of session)
- Tools opt-in; agent edits **workspace only**; **Promote** → copy to `source/` + optional review
- No DraftNote entity; publication state machine; path isolation; owner auth
- Review: whole | bites; append-only events; project or all-project scope
- v1 UI: Home → Project (Overview | Notes | Sessions | Review); project sessions only
- Schema ready for global/vault sessions; UI does not expose them yet

**Non-goals v1:** multi-tenant, live S3, note edit/delete UI, todos/Google, Amp required, external FS reconcile.

---

## Plan requirements (writing-plans)

1. Announce: using writing-plans.
2. Header block required by the skill (agentic workers, goal, architecture, tech stack, global constraints).
3. Map files/packages **before** tasks.
4. Bite-sized tasks: checkbox steps, TDD red/green, commit each task.
5. Plan path: `docs/superpowers/plans/YYYY-MM-DD-personal-agent-v1.md`
6. Prefer phasing aligned with spec §14:
   - Skeleton (Compose, SQLite, auth bootstrap, health, empty Home)
   - Projects + source tree
   - Sessions + chat + agent run
   - Workspace tools
   - Promote + review
   - Backup
   - Hardening / recovery tests
7. Each phase should leave **working, testable** software.
8. If the plan is huge, split into **sequenced plan docs** (plan A unlocks B) rather than one unreadable mega-plan — but keep one entry plan that links the rest.
9. After writing the plan: present it for **user review**; do not start `executing-plans` / implementation unless the user says go.

---

## Spec sections the planner must honor

Read the full spec. Especially:

- §3 Architecture & stack (Go, SQLite, Compose)
- §4 Filesystem layout (including `global/sessions`, `staging/`)
- §5 Data model & session scope invariants
- §6 Publication state machine (promote + direct create)
- §7 Review semantics
- §8 Screens (flattened nav)
- §9–13 Flows, concurrency, security, backup, acceptance tests
- §15 Open decisions → resolve in plan with concrete defaults where needed

---

## Out of scope for the planning thread

- Re-opening product brainstorming unless the spec is contradictory
- Implementing features
- Pushing to origin unless user asks
- Multi-vault UI, global chat UI, todos, Google

---

## Suggested first actions in the new thread

1. Skill: `using-superpowers` then `writing-plans`
2. Read the approved spec end-to-end
3. Read this handoff
4. Inspect repo root (what exists vs greenfield)
5. Write the plan file
6. Self-check against writing-plans checklist
7. Commit the plan
8. Ask user to review the plan before execution
