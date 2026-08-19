---
name: compounding-engineering
description: Turn any completed task — feature, refactor, bugfix, research, or ops work — into compounding-engineering artifacts so the next task is easier to build. Use after finishing a piece of work, when the agent repeated a mistake, when a plan needed revision, or when the user asks to "codify learnings", "make the next feature easier", "compound this", "wrap this session", or "capture what we just learned". Codifies learnings into AGENTS.md standing rules, hooks, and tests, and always records a lesson under the project's docs/memory directory.
allowed-tools:
  - Read
  - Write
  - Edit
  - Glob
  - Grep
  - Bash
---

# Compound Engineering

Flip software's default of **diminishing returns** (each feature adds complexity, making the next harder) into **accelerating returns**: each task makes the next one easier by codifying its learnings back into reusable agent instructions. Applies to any all-around task — features, refactors, bugfixes, research, ops, onboarding — not just debugging.

> Source of truth for this skill: the pattern doc (@compounding-engineering-pattern.md) and its research report (@compounding-engineering-pattern-report.md). Locate them in the repo and read them if you need the full rationale, related patterns, or trade-offs.

## The loop

```
Do the task → Document the learnings → Codify into prompts/commands/hooks/tests
   → Record a memory entry in docs/memory/ → Next task uses that knowledge
   → Easier & faster → (repeat)
```

The codebase becomes increasingly **self-teaching**: a newcomer (human or agent) can be productive without knowing the whole codebase, because the accumulated memory carries the context.

## When to run this

Invoke after any unit of work that produced a non-obvious learning, especially when:

- The agent got something wrong initially and had to be corrected.
- The plan needed revision partway through.
- Testing surfaced an edge case that wasn't anticipated.
- You answered the same question (or hit the same trap) more than once.

If the work was purely mechanical and taught nothing reusable, say so and stop — don't manufacture artifacts, and don't write a memory entry.

## Step 1 — Extract the learnings

From the task context (`$ARGUMENTS` if provided, otherwise the current conversation's most recent work; if neither exists, ask the user to paste it), identify:

1. **What worked in the plan** and what needed adjustment.
2. **Issues discovered during testing** that weren't caught earlier.
3. **Common mistakes** the agent made — the corrections you had to give.
4. **Patterns and best practices** worth reusing.
5. **Repeated questions / environment traps** — anything asked or hit more than once.

Keep only what was *non-obvious*. Do not restate what the repo already records (code structure, git history, existing docs).

## Step 2 — Record a memory entry in `docs/memory/` (ALWAYS)

Every compounding run MUST update the project's lessons log. This is the durable, human-readable record of what was learned and why — Step 3 artifacts reference it, not the other way around.

**This repo's convention (overrides the generic multi-file diary):**

- **One stable file:** `docs/memory/lessons.md` only. Do **not** create `YYYY-MM-DD-lessons.md` per task/session.
- **Newest first:** prepend the new `###` section under `## Lessons (newest first)`.
- **Index:** update the topic → latest-lesson row in the file’s Index table for each touched tag.
- **Section template** (keep short and factual):

```markdown
### YYYY-MM-DD — <short title>

**Tags:** tag1, tag2

**Task:** <one line>

**Wrong / mistakes:** …
**What worked:** …
**Rule (next agent):** …
**Codified into:** <paths>
**Evidence:** <thread URL or commit>
```

- If an entry already covers the same learning, update or supersede it (note supersession) rather than duplicating.
- Standing rules that should load every session go in **`AGENTS.md`** (short bullets), with a pointer back to the lesson. Do **not** expect agents to re-read the full diary every session.

**Periodic synthesis:** when a pattern appears in **3 or more** independent lessons, promote it via the `synthesize-memory` skill into durable artifacts (standing bullets, hooks, tests). Diaries/lessons capture; synthesis compounds.

## Step 3 — Codify into the right artifact

Map each learning to the most durable form it fits. **Prefer executable over prose** — a check you can run compounds; a paragraph rots. Generate as many as the context genuinely supports:

| Learning type | Codify as |
|---|---|
| Global coding standard / convention / "always do X" | Short bullet in **`AGENTS.md`** Standing rules (+ lesson detail in `docs/memory/`) |
| A repeatable multi-step workflow | A **skill** under `.agents/skills/` (prefer over ad-hoc slash commands here) |
| Specialized validation or review expertise | A **skill** / subagent the main agent can invoke |
| A mistake that must never recur automatically | A **hook** (pre-commit, pre-run, CI step) or failing **test** that blocks it |
| A requirement or edge case | A **test** that encodes it and fails if violated |
| A working solution worth reusing verbatim | A **skill** or library helper in the codebase |

Before writing anything, read the project's actual paths and conventions (`AGENTS.md`, `Makefile`, existing `.agents/skills/`, `docs/memory/`). Keep artifacts minimal. Any script must exit non-zero on failure. Cross-link each artifact back to its lesson section so the rationale is one hop away. Do **not** invent `docs/solutions/` or parallel doc trees.

## Step 4 — Apply the compounding test

For each artifact ask:

> **"If a new teammate or agent hits this same situation next month, does this artifact let them skip the investigation entirely?"**

If the answer is still "no", you haven't compounded it — add or sharpen the missing artifact until the answer is "yes".

## Guard against the failure modes

Codification has real costs (see the trade-offs in the pattern doc). Actively avoid:

- **Over-specification** — too many rigid rules make agents inflexible. Codify principles and gates, not every micro-decision.
- **Prompt bloat** — `AGENTS.md` standing rules grow unbounded. Prefer a lesson section + skill/hook/test over another long paragraph; prune stale bullets when you touch them.
- **Stale knowledge** — codified rules that no longer hold are worse than none. When a learning contradicts an existing artifact, update or delete the old one rather than stacking a new rule on top. This applies to `docs/memory/` too: mark superseded entries instead of leaving contradictory records side by side.

## Output

1. A short plan: list the artifacts you'll create/update — starting with the lesson section in `docs/memory/lessons.md` — and the one-line reason each one compounds.
2. Prepend/update the lesson first (newest first + Index), then create or edit the other artifacts (`AGENTS.md` bullet, skill, test, hook), matching existing project conventions.
3. Finish with the single command (or one concrete next step) that proves the new artifacts work — e.g. run the new test or hook — and confirm the lesson path is `docs/memory/lessons.md`.
