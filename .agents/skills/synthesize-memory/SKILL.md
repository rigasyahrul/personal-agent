---
name: synthesize-memory
description: Review accumulated lessons in docs/memory/ and promote recurring patterns into durable artifacts (AGENTS.md standing rules, skills, hooks, tests). Use when the user asks to "synthesize memory", "review learnings", "what patterns keep recurring", "consolidate docs/memory", or after many lesson sections have accumulated. This is the synthesis tier that pairs with compounding-engineering (which writes the entries); do NOT use this to capture learnings from a single task — use compounding-engineering for that.
allowed-tools:
  - Read
  - Write
  - Edit
  - Glob
  - Grep
  - Bash
---

# Synthesize Memory

Implements the synthesis tier of the *Memory Synthesis from Execution Logs* pattern. The `compounding-engineering` skill appends lesson sections under `docs/memory/*-lessons.md`; this skill periodically reads across those lessons, finds what recurs, and promotes it into durable artifacts. Lessons capture; synthesis compounds.

> Source of truth: @memory-synthesis-from-execution-logs.md and `../compounding-engineering/compounding-engineering-pattern.md`.

## When to run this

- The user explicitly asks to synthesize, consolidate, or review accumulated learnings.
- The lessons file has grown large (~10+ distinct `###` sections since last synthesis).
- Before a new teammate or agent onboards to the project (synthesis makes the memory scannable).

If there are fewer than ~5 independent lesson sections, say the corpus is too small for reliable pattern extraction (cold-start problem) and stop — do not force patterns out of coincidence.

## Step 1 — Load the lessons corpus

```bash
ls docs/memory/
# Primary: docs/memory/2026-08-12-lessons.md (one file; ### sections)
```

Read every `###` lesson section. Prefer the project's one-file format (`**Wrong**` / `**What worked**` / `**Rule**` / `**Codified into**`). If multi-file diary entries with `synthesized: false` frontmatter exist, include those too.

Also skim `AGENTS.md` standing rules — you need them in Step 3 to detect contradictions.

## Step 2 — Cluster and apply the evidence threshold

Group observations across entries by theme (tags and *Patterns discovered* sections are the primary signal; *Mistakes made* is the secondary one).

**A pattern is only promotable when it appears in 3 or more independent entries.** This is the evidence threshold from the source pattern — it separates genuine patterns from coincidence.

For each cluster, classify:

- **Promotable (3+ occurrences):** proceed to Step 3.
- **Watch list (2 occurrences):** do not promote; note it in the synthesis report so the next run checks it first.
- **One-off:** leave in the diary; it is already captured.

Beware false patterns: 3 entries that share a cause (e.g. the same underlying bug, since fixed) are one occurrence, not three. Check dates and context before counting.

## Step 3 — Promote into durable artifacts

For each promotable pattern, use the same artifact mapping as compounding-engineering (standard → `AGENTS.md` standing bullet; workflow → skill; must-never-recur → hook/test; reusable solution → skill/code). Prefer executable over prose.

Rules:

1. **Check for contradictions first.** If the new pattern contradicts an existing rule, hook, or previously synthesized promotion, update or delete the old artifact — never stack a contradictory rule on top. Note the supersession in the affected lesson.
2. **Match project conventions** — read `AGENTS.md`, `.agents/skills/`, and test layout before writing. No extra doc trees.
3. **Right abstraction level.** Promote the general rule, not the specific instance ("auth changes need a CORS review" — not "add CORS header to checkout endpoint"). If the entries only support the specific instance, it isn't a pattern yet.
4. **Cross-link.** Each promoted artifact cites the lesson section(s) that justify it, so the evidence trail survives.

## Step 4 — Mark and report

1. In the lessons file, mark promoted sections with a short note under the section (e.g. `**Synthesized:** YYYY-MM-DD → AGENTS.md / …`) so the next run does not re-promote blindly.
2. Append a short synthesis report section (`### YYYY-MM-DD — memory synthesis`) listing: sections reviewed, patterns promoted (with artifact paths), the watch list, and anything pruned or superseded.
3. Prune only if the user asks: move long-superseded material to `docs/memory/archive/` — never delete; the evidence trail matters.

## Guard against the failure modes

- **Over-promotion** — every promoted rule is permanent context cost. When in doubt, leave it on the watch list; the threshold exists to protect `AGENTS.md` from bloat.
- **False patterns** — coincidental correlation across entries is not causation. Verify independence before counting occurrences.
- **Stale promotions** — each run, spot-check 2–3 previously promoted rules against the current codebase; delete ones that no longer hold.

## Output

1. The synthesis report (entry created in Step 4) — this is the primary deliverable.
2. The list of created/updated artifacts.
3. Finish with the single command that proves a promoted artifact works — run the new hook, test, or slash command.
