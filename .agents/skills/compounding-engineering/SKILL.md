---
name: compounding-engineering
description: Turn any completed task — feature, refactor, bugfix, research, or ops work — into compounding-engineering artifacts so the next task is easier to build. Use after finishing a piece of work, when the agent repeated a mistake, when a plan needed revision, or when the user asks to "codify learnings", "make the next feature easier", "compound this", "wrap this session", or "capture what we just learned". Prefer durable codification (AGENTS.md, skills, hooks, tests); when evidence is needed, write docs/memory/YYYYMMDD-HHmm-slug.md and index it in docs/memory/lessons.md.
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
Do the task → Extract learnings → Codify into AGENTS.md / skills / hooks / tests
   → (only if needed) Entry file docs/memory/YYYYMMDD-HHmm-slug.md
   → Index row in docs/memory/lessons.md
   → Next task uses that knowledge → Easier & faster → (repeat)
```

**Compounding = durable behavior change**, not a longer diary. A standing bullet, skill edit, hook, or failing test that prevents the mistake again is the win. Memory entries are **optional evidence** for synthesis — not a mandatory session log.

The codebase becomes increasingly **self-teaching**: a newcomer (human or agent) can be productive without knowing the whole codebase, because the accumulated instructions and checks carry the context.

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

## Step 2 — Codify into the right artifact (primary)

Map each learning to the most durable form it fits. **Prefer executable over prose** — a check you can run compounds; a paragraph rots. Generate as many as the context genuinely supports:

| Learning type | Codify as |
|---|---|
| Global coding standard / convention / "always do X" | Short bullet in **`AGENTS.md`** Standing rules |
| A repeatable multi-step workflow | A **skill** under `.agents/skills/` (prefer over ad-hoc slash commands here) |
| Specialized validation or review expertise | A **skill** / subagent the main agent can invoke |
| A mistake that must never recur automatically | A **hook** (pre-commit, pre-run, CI step) or failing **test** that blocks it |
| A requirement or edge case | A **test** that encodes it and fails if violated |
| A working solution worth reusing verbatim | A **skill** or library helper in the codebase |

Before writing anything, read the project's actual paths and conventions (`AGENTS.md`, `Makefile`, existing `.agents/skills/`, `docs/memory/`). Keep artifacts minimal. Any script must exit non-zero on failure. Do **not** invent `docs/solutions/` or parallel doc trees.

**Done means the next agent hits the durable artifact first** — not that a diary page exists.

## Step 3 — Record evidence in `docs/memory/` (selective)

Memory is an **on-demand evidence corpus** for compound / synthesize / repeated-trap debugging. It is **not** required on every compounding run and must not become a full session transcript.

### Layout (this repo)

| Piece | Path | Role |
|-------|------|------|
| **Index (list only)** | `docs/memory/lessons.md` | Topic map + newest-first bullet list of links — keep thin |
| **Entry (detail)** | `docs/memory/YYYYMMDD-HHmm-<title-slug>.md` | Full Wrong / What worked / Rule / Codified / Evidence |

Matches the pattern’s timestamped diary file. **Do not** stuff full narratives into `lessons.md`.

### Write or update an entry **only when** at least one is true

- Non-obvious trap with **rationale** that will not fit a short AGENTS bullet.
- User/agent **correction** that needs an evidence trail for later `synthesize-memory` (3+ recurrence).
- Learning spans multiple systems and needs a **Wrong / What worked** narrative so the next agent skips the investigation.
- You are **superseding** an older entry (edit that file + fix index links; do not stack a duplicate).

### Skip memory when

- Nothing reusable was learned (mechanical work).
- The learning is fully captured by an **AGENTS bullet, skill, hook, or test** and adds no extra rationale worth keeping.
- The same learning is already in an entry — **update or supersede** that file and its index row instead of creating a near-duplicate.
- You would only be restating git history, the PR description, or the plan.

### How to add an entry

1. **Filename:** `docs/memory/YYYYMMDD-HHmm-<title-slug>.md`
   - `YYYYMMDD-HHmm` = local date/time when the lesson is written (not “session start”).
   - `title-slug` = lowercase kebab-case from the short title; max ~60 chars; ASCII only.
   - Example: `docs/memory/20260820-1430-master-grok-spawn-local-x.md`
2. **Entry body** (keep factual; prefer ≤40 lines; link thread/commit instead of pasting transcripts):

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

3. **Index** (`docs/memory/lessons.md`):
   - Prepend a bullet under **Entries (newest first)** linking to the new file.
   - Refresh **Topic → latest entry** rows for each touched tag.
   - Do **not** paste the full body into the index.

### Anti-bloat rules

- **Index stays thin** — list + topic table only. Detail lives in entry files.
- **No dump** — no full prompts, full diffs, or multi-phase play-by-plays in entries; link **Evidence**.
- **Codified into** must list the *durable* paths (AGENTS / skill / test / hook). Listing only a memory path means you have **not** compounded yet — go back to Step 2.
- **Session wrap:** if this session already wrote the entry while implementing, wrap = verify codification + attach evidence on that file — do **not** create a second entry for the same learning.
- Standing rules that should load every session go in **`AGENTS.md`** (short bullets), with an optional pointer to the entry file. Do **not** expect agents to re-read all entries every session.
- Do **not** create `docs/solutions/` or put plans under `docs/memory/`.

**Periodic synthesis:** when a pattern appears in **3 or more** independent entries, promote it via the `synthesize-memory` skill. Entries capture evidence; synthesis and Step 2 codify. Prune or move long-superseded files to `docs/memory/archive/` only when asked.

## Step 4 — Apply the compounding test

For each artifact ask:

> **"If a new teammate or agent hits this same situation next month, does this artifact let them skip the investigation entirely?"**

If the answer is still "no", you haven't compounded it — add or sharpen the missing **durable** artifact until the answer is "yes". A memory entry alone almost never passes this test.

## Guard against the failure modes

Codification has real costs (see the trade-offs in the pattern doc). Actively avoid:

- **Over-specification** — too many rigid rules make agents inflexible. Codify principles and gates, not every micro-decision.
- **Prompt bloat** — `AGENTS.md` standing rules grow unbounded. Prefer a tight bullet + skill/hook/test over another long paragraph; prune stale bullets when you touch them.
- **Diary bloat** — memory is not a changelog. If entry count grows fast, you are under-codifying or over-logging; favor Step 2 and skip or merge entries. Never re-inflate `lessons.md` with full bodies.
- **Stale knowledge** — codified rules that no longer hold are worse than none. When a learning contradicts an existing artifact, update or delete the old one rather than stacking a new rule on top. Same for entries: mark superseded (and fix index) instead of leaving contradictory files side by side.

## Output

1. A short plan: list the **durable** artifacts you'll create/update (AGENTS / skill / test / hook) and, only if Step 3 applies, the entry path + index update — one-line reason each compounds.
2. Codify first (Step 2). Write `docs/memory/YYYYMMDD-HHmm-slug.md` + thin index row only when Step 3 says so.
3. Finish with the single command (or one concrete next step) that proves the new durable artifact works — e.g. run the new test or hook. Confirm you did **not** write a memory entry solely to "have something in memory."
