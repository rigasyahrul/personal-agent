---
name: synthesize-memory
description: Review accumulated lessons in docs/memory/ and promote recurring patterns into durable artifacts (AGENTS.md standing rules, skills, hooks, tests). Use when the user asks to "synthesize memory", "review learnings", "what patterns keep recurring", "consolidate docs/memory", or after many lesson entries have accumulated. This is the synthesis tier that pairs with compounding-engineering (which codifies first and may leave selective entry files + index rows); do NOT use this to capture learnings from a single task — use compounding-engineering for that.
allowed-tools:
  - Read
  - Write
  - Edit
  - Glob
  - Grep
  - Bash
---

# Synthesize Memory

Implements the synthesis tier of the *Memory Synthesis from Execution Logs* pattern. The `compounding-engineering` skill **codifies first** (AGENTS / skills / hooks / tests) and only sometimes writes selective evidence as `docs/memory/YYYYMMDD-HHmm-*.md` plus a thin row in `docs/memory/lessons.md`; this skill periodically reads across those entries, finds what recurs, and promotes it into durable artifacts. Lessons are a selective evidence corpus — not a full session log. Synthesis compounds what still only lives as evidence.

> Source of truth: @memory-synthesis-from-execution-logs.md and `../compounding-engineering/compounding-engineering-pattern.md`.

## When to run this

- The user explicitly asks to synthesize, consolidate, or review accumulated learnings.
- The corpus has grown large (~10+ distinct entry files since last synthesis).
- Before a new teammate or agent onboards to the project (synthesis makes the memory scannable).

If there are fewer than ~5 independent entry files, say the corpus is too small for reliable pattern extraction (cold-start problem) and stop — do not force patterns out of coincidence.

## Step 1 — Load the lessons corpus

```bash
ls docs/memory/
# Index: docs/memory/lessons.md (topic map + entry list only)
# Entries: docs/memory/YYYYMMDD-HHmm-*.md
# Corpus may be sparse — compounding skips entries when durable artifacts already hold the rule.
```

1. Read **`docs/memory/lessons.md`** (topic table + entry list) to choose what to open.
2. Read each relevant **`docs/memory/YYYYMMDD-HHmm-*.md`** entry (`**Tags**` / `**Wrong**` / `**What worked**` / `**Rule**` / `**Codified into**`). Expect short files.
3. Also skim recently changed skills / `AGENTS.md` bullets — some patterns never hit the diary.

Also skim `AGENTS.md` standing rules — you need them in Step 3 to detect contradictions.

## Step 2 — Cluster and apply the evidence threshold

Group observations across entries by theme (tags and *Patterns discovered* / *What worked* are the primary signal; *Wrong / mistakes* is the secondary one).

**A pattern is only promotable when it appears in 3 or more independent entries.** This is the evidence threshold from the source pattern — it separates genuine patterns from coincidence.

For each cluster, classify:

- **Promotable (3+ occurrences):** proceed to Step 3.
- **Watch list (2 occurrences):** do not promote; note it in the synthesis report so the next run checks it first.
- **One-off:** leave in the diary; it is already captured.

Beware false patterns: 3 entries that share a cause (e.g. the same underlying bug, since fixed) are one occurrence, not three. Check dates and context before counting.

## Step 3 — Promote into durable artifacts

For each promotable pattern, use the same artifact mapping as compounding-engineering (standard → `AGENTS.md` standing bullet; workflow → skill; must-never-recur → hook/test; reusable solution → skill/code). Prefer executable over prose.

Rules:

1. **Check for contradictions first.** If the new pattern contradicts an existing rule, hook, or previously synthesized promotion, update or delete the old artifact — never stack a contradictory rule on top. Note the supersession in the affected entry file.
2. **Match project conventions** — read `AGENTS.md`, `.agents/skills/`, and test layout before writing. No extra doc trees.
3. **Right abstraction level.** Promote the general rule, not the specific instance ("auth changes need a CORS review" — not "add CORS header to checkout endpoint"). If the entries only support the specific instance, it isn't a pattern yet.
4. **Cross-link.** Each promoted artifact cites the entry file(s) that justify it, so the evidence trail survives.

## Step 4 — Mark and report

1. In each promoted **entry file**, mark with a short note (e.g. `**Synthesized:** YYYY-MM-DD → AGENTS.md / …`) so the next run does not re-promote blindly.
2. Write a short synthesis report as a **new entry file** `docs/memory/YYYYMMDD-HHmm-memory-synthesis.md` and prepend a list row in `docs/memory/lessons.md`. Report: entries reviewed, patterns promoted (with artifact paths), the watch list, and anything pruned or superseded. Refresh the topic index.
3. Prune only if the user asks: move long-superseded entry files to `docs/memory/archive/` — never delete; the evidence trail matters. Keep index links updated (or point at archive paths).

## Guard against the failure modes

- **Over-promotion** — every promoted rule is permanent context cost. When in doubt, leave it on the watch list; the threshold exists to protect `AGENTS.md` from bloat.
- **False patterns** — coincidental correlation across entries is not causation. Verify independence before counting occurrences.
- **Stale promotions** — each run, spot-check 2–3 previously promoted rules against the current codebase; delete ones that no longer hold.
- **Index bloat** — never paste full synthesis narratives into `lessons.md`; entry file + one list row only.

## Output

1. The synthesis report entry (Step 4) — this is the primary deliverable.
2. The list of created/updated artifacts.
3. Finish with the single command that proves a promoted artifact works — run the new hook, test, or slash command.
