# Lessons index (compounding engineering)

> **Hot path (every session):** `AGENTS.md` → Standing rules.  
> **Preferred durable form:** skills, hooks, tests — not memory entries.  
> **This file:** lightweight **index only** (list + topic map). Full write-ups live in sibling files.  
> Pattern: [Compounding Engineering](https://www.agentic-patterns.com/patterns/compounding-engineering-pattern/)

**Layout:**
- Index: `docs/memory/lessons.md` (this file)
- Entries: `docs/memory/YYYYMMDD-HHmm-<title-slug>.md` (detail; selective — not every session)
- Plans/specs/locks: `docs/superpowers/` only — never under `docs/memory/`
- No `docs/solutions/` trees

## How memory works

| Layer | Path | When loaded |
|-------|------|-------------|
| Standing rules (policy) | `AGENTS.md` | Almost every session |
| Skills / hooks / tests | `.agents/skills/`, repo checks | When the task matches / always as gates |
| Lessons index | **this file** | Scan topics / find entry paths |
| Lesson entries (evidence) | `docs/memory/YYYYMMDD-HHmm-*.md` | Only when rationale is needed |

**After non-trivial work or a user correction:** run `compounding-engineering` → **codify first** (AGENTS / skill / hook / test) → if evidence is needed, write **one** entry file + add a list row here. Skip memory when a durable artifact fully captures the rule.  
**When ~10+ new entries or a theme recurs 3+ times:** run `synthesize-memory`.  
**Same learning twice:** update or supersede the existing entry file; refresh the list/index row — do not duplicate.

## Topic → latest entry

Scan this first. Open the linked file only when you need detail.

| Topic | Latest lesson |
|-------|----------------|
| agents | [2026-08-19 — Lessons file is stable `lessons.md`](20260819-2200-lessons-file-is-stable-lessons-md.md) |
| amp | [2026-08-20 — Master Grok spawn: local `-x`, isolate worktrees](20260820-2200-master-grok-spawn-local-x-isolate-worktrees.md) |
| backup | [2026-08-19 — macOS local `make test` platform gaps](20260819-1900-macos-local-make-test-platform-gaps.md) |
| compounding | [2026-08-20 — Compound memory is selective, not a session dump](20260820-2300-compound-memory-is-selective-not-a-session-dump.md) |
| darwin | [2026-08-19 — macOS local `make test` platform gaps](20260819-1900-macos-local-make-test-platform-gaps.md) |
| docker | [2026-08-19 — Sessions chat focus + Docker live-reload](20260819-2100-sessions-chat-focus-docker-live-reload.md) |
| docs | [2026-08-19 — Lessons file is stable `lessons.md`](20260819-2200-lessons-file-is-stable-lessons-md.md) |
| docs layout | [2026-08-20 — Compound memory is selective, not a session dump](20260820-2300-compound-memory-is-selective-not-a-session-dump.md) |
| fixtures | [2026-08-13 — Verify escaped literals at the byte level](20260813-2300-verify-escaped-literals-at-the-byte-level.md) |
| craft | [2026-08-20 — UI craft tokens + Go dist cache-bust](20260820-0953-ui-craft-tokens-and-dist-cache-bust.md) |
| dist | [2026-08-20 — UI craft tokens + Go dist cache-bust](20260820-0953-ui-craft-tokens-and-dist-cache-bust.md) |
| frontend | [2026-08-20 — UI craft tokens + Go dist cache-bust](20260820-0953-ui-craft-tokens-and-dist-cache-bust.md) |
| fs | [2026-08-19 — macOS local `make test` platform gaps](20260819-1900-macos-local-make-test-platform-gaps.md) |
| git | [2026-08-20 — Master Grok spawn: local `-x`, isolate worktrees](20260820-2200-master-grok-spawn-local-x-isolate-worktrees.md) |
| gitignore | [2026-08-19 — Default `make` is help; build binary is gitignored](20260819-2000-default-make-is-help-build-binary-is-gitignored.md) |
| go | [2026-08-13 — Verify escaped literals at the byte level](20260813-2300-verify-escaped-literals-at-the-byte-level.md) |
| grok | [2026-08-19 — consulting-grok-review via Grok thread, not Task/OpenAI](20260819-1700-consulting-grok-review-via-grok-thread-not-task-openai.md) |
| grok45 | [2026-08-20 — Master Grok spawn: local `-x`, isolate worktrees](20260820-2200-master-grok-spawn-local-x-isolate-worktrees.md) |
| make | [2026-08-19 — Sessions chat focus + Docker live-reload](20260819-2100-sessions-chat-focus-docker-live-reload.md) |
| master | [2026-08-20 — Master Grok spawn: local `-x`, isolate worktrees](20260820-2200-master-grok-spawn-local-x-isolate-worktrees.md) |
| memory | [2026-08-20 — Compound memory is selective, not a session dump](20260820-2300-compound-memory-is-selective-not-a-session-dump.md) |
| merge | [2026-08-20 — Master Grok spawn: local `-x`, isolate worktrees](20260820-2200-master-grok-spawn-local-x-isolate-worktrees.md) |
| multi-agent | [2026-08-19 — Big plans: lock + parallel drafts, don't solo-stall](20260819-2300-big-plans-lock-parallel-drafts-don-t-solo-stall.md) |
| node | [2026-08-20 — Master Grok spawn: local `-x`, isolate worktrees](20260820-2200-master-grok-spawn-local-x-isolate-worktrees.md) |
| orb-execute | [2026-08-20 — Master Grok spawn: local `-x`, isolate worktrees](20260820-2200-master-grok-spawn-local-x-isolate-worktrees.md) |
| plans | [2026-08-19 — Big plans: lock + parallel drafts, don't solo-stall](20260819-2300-big-plans-lock-parallel-drafts-don-t-solo-stall.md) |
| review | [2026-08-20 — Master Grok spawn: local `-x`, isolate worktrees](20260820-2200-master-grok-spawn-local-x-isolate-worktrees.md) |
| shell | [2026-08-19 — macOS local `make test` platform gaps](20260819-1900-macos-local-make-test-platform-gaps.md) |
| ship | [2026-08-12 — Ship means push](20260812-2100-ship-means-push.md) |
| skills | [2026-08-20 — Frontend UI craft skill from live pass](20260820-1606-frontend-ui-craft-skill-from-live-pass.md) |
| spa | [2026-08-19 — Sessions chat focus + Docker live-reload](20260819-2100-sessions-chat-focus-docker-live-reload.md) |
| spawn | [2026-08-20 — Master Grok spawn: local `-x`, isolate worktrees](20260820-2200-master-grok-spawn-local-x-isolate-worktrees.md) |
| ui | [2026-08-20 — UI craft tokens + Go dist cache-bust](20260820-0953-ui-craft-tokens-and-dist-cache-bust.md) |
| web | [2026-08-20 — UI craft tokens + Go dist cache-bust](20260820-0953-ui-craft-tokens-and-dist-cache-bust.md) |
| writing-plans | [2026-08-19 — Big plans: lock + parallel drafts, don't solo-stall](20260819-2300-big-plans-lock-parallel-drafts-don-t-solo-stall.md) |

## Where Superpowers artifacts go

| Artifact | Path |
|----------|------|
| Design specs | `docs/superpowers/specs/YYYY-MM-DD-*-design.md` |
| Implementation plans | `docs/superpowers/plans/YYYY-MM-DD-*.md` |
| Plan lock / drafts | `docs/superpowers/plans/…-lock.md`, `…-drafts/` |
| Handoffs | `docs/superpowers/HANDOFF-*.md` |
| **Lessons index** | **`docs/memory/lessons.md`** (this file) |
| **Lesson entries** | **`docs/memory/YYYYMMDD-HHmm-<slug>.md`** |
| Standing agent rules | **`AGENTS.md`** |
| Skills / hooks / tests | Prefer these over new lesson entries |

---

## Entries (newest first)

- **2026-08-20** — [UI craft tokens + Go dist cache-bust](20260820-0953-ui-craft-tokens-and-dist-cache-bust.md)  
  Tags: frontend, ui, web, craft, dist, orb
- **2026-08-20** — [Frontend UI craft skill from live pass](20260820-1606-frontend-ui-craft-skill-from-live-pass.md)  
  Tags: ui, skills, frontend, web, compounding
- **2026-08-20** — [Compound memory is selective, not a session dump](20260820-2300-compound-memory-is-selective-not-a-session-dump.md)  
  Tags: memory, compounding, docs layout
- **2026-08-20** — [Master Grok spawn: local `-x`, isolate worktrees](20260820-2200-master-grok-spawn-local-x-isolate-worktrees.md)  
  Tags: master, spawn, grok45, amp, orb-execute, git, merge, review, node, web
- **2026-08-19** — [Big plans: lock + parallel drafts, don't solo-stall](20260819-2300-big-plans-lock-parallel-drafts-don-t-solo-stall.md)  
  Tags: plans, multi-agent, writing-plans, skills, ui
- **2026-08-19** — [Lessons file is stable `lessons.md`](20260819-2200-lessons-file-is-stable-lessons-md.md)  
  Tags: memory, docs, agents, compounding
- **2026-08-19** — [Sessions chat focus + Docker live-reload](20260819-2100-sessions-chat-focus-docker-live-reload.md)  
  Tags: ui, spa, docker, make
- **2026-08-19** — [Default `make` is help; build binary is gitignored](20260819-2000-default-make-is-help-build-binary-is-gitignored.md)  
  Tags: make, skills, gitignore
- **2026-08-19** — [macOS local `make test` platform gaps](20260819-1900-macos-local-make-test-platform-gaps.md)  
  Tags: darwin, fs, shell, backup
- **2026-08-19** — [Master merge: board tip can block pure FF](20260819-1800-master-merge-board-tip-can-block-pure-ff.md)  
  Tags: git, merge, master
- **2026-08-19** — [consulting-grok-review via Grok thread, not Task/OpenAI](20260819-1700-consulting-grok-review-via-grok-thread-not-task-openai.md)  
  Tags: review, grok
- **2026-08-19** — [Worker high-stakes review = consulting-grok-review, not built-in oracle](20260819-1600-worker-high-stakes-review-consulting-grok-review-not-built-in-oracle.md)  
  Tags: review, grok
- **2026-08-13** — [Verify escaped literals at the byte level](20260813-2300-verify-escaped-literals-at-the-byte-level.md)  
  Tags: review, go, fixtures
- **2026-08-12** — [Keep docs simple](20260812-2300-keep-docs-simple.md)  
  Tags: docs, memory
- **2026-08-12** — [Multi-agent plans need one authority](20260812-2200-multi-agent-plans-need-one-authority.md)  
  Tags: plans, multi-agent
- **2026-08-12** — [Ship means push](20260812-2100-ship-means-push.md)  
  Tags: ship, git

---

## Template for a new entry

**Only when** selective criteria in `compounding-engineering` say so. Codify durable artifacts first.

1. Create `docs/memory/YYYYMMDD-HHmm-<title-slug>.md` (local time; slug = lowercase kebab, ≤60 chars).
2. Prepend a bullet under **Entries** above + refresh **Topic → latest entry** rows for touched tags.

```markdown
# <short title>

**Date:** YYYY-MM-DD  
**Tags:** tag1, tag2

**Task:** <one line>

**Wrong / mistakes:** …
**What worked:** …
**Rule (next agent):** …
**Codified into:** <AGENTS.md / skill / test / hook paths — not only this file>
**Evidence:** <thread URL or commit>
```
