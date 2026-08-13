---
name: consulting-grok-review
description: "Use when you need a high-stakes second-opinion code or design review shaped like Amp Oracle—especially if Oracle/ChatGPT is unavailable or usage-limited, before merge/ship, after a hard debugging diagnosis, or when choosing between plausible alternatives with real downside."
---

# Consulting Grok Review (Oracle-shaped)

**Core idea:** Get a **read-only, high-impact second opinion** with the same *input discipline* and *feedback shape* as Amp’s Oracle—without depending on the built-in `oracle` tool or ChatGPT-backed routing.

This skill does **not** invent a new model API. It forces the parent agent to:
1. Frame the question the way Oracle expects.
2. Dispatch an isolated reviewer (Task/subagent) with a fixed prompt.
3. Accept only Oracle-grade output (verdict + evidence + decision impact).
4. Act on Critical/Important findings before claiming done.

Evaluation of model quality vs real Oracle is **out of scope here**—run that later deliberately.

## When to use

- Built-in `oracle` is unavailable, failing, or blocked by provider/subscription limits.
- You need a **second pair of eyes** on a concrete high-impact question, not a vibe check.
- Before accepting a phase/branch merge, shipping, or locking a multi-agent plan.
- After you already gathered evidence and still have **one unresolved judgment**.

## When NOT to use

- Routine self-review of a tiny diff you can settle yourself.
- Broad “find anything wrong” fishing with no decision at stake.
- Asking whether work is “ready” without a specific risk or invariant.
- Codebase search (use finder/rg) or implementing the fix (do it yourself after the review).
- Replacing `requesting-code-review` for every task—this is for **Oracle-tier** calls only.

## Relationship to other skills

| Skill | Role |
|-------|------|
| Built-in `oracle` tool | Prefer when available and healthy. |
| `requesting-code-review` | Default per-task / phase diff review. |
| **This skill** | Oracle-shaped second opinion when stakes are high or Oracle is blocked. |
| `receiving-code-review` | How to handle findings (no performative agreement). |

If both this skill and `requesting-code-review` could apply, use **this skill** only when the question is a single unresolved high-impact judgment; otherwise use the standard code-review skill.

---

## Oracle input contract (required)

Before dispatching, the parent agent must write a **single focused task** that includes all of:

1. **Unresolved question** — one decision, invariant, or failure mode (not a laundry list).
2. **What you already checked** — commands, files, tests, ruled-out theories.
3. **Why the answer changes the decision** — what you will do differently for each outcome.
4. **Intended behavior** — product/spec truth the implementation must match.
5. **Settled constraints** — non-negotiables already decided (canonical contracts, no redesign, etc.).
6. **Scope boundary** — `@`-mention the few critical files; name what to ignore.
7. **Evidence request** — exact quotes, paths, line refs, failing sequences if any.

### Good task shape (copy and fill)

```text
UNRESOLVED QUESTION:
<one sentence>

ALREADY CHECKED:
- <command or inspection → result>
- <ruled out: …>

DECISION IMPACT:
- If YES/A: I will …
- If NO/B: I will …

INTENDED BEHAVIOR:
<spec/plan truth>

SETTLED CONSTRAINTS:
- <canonical X wins over snippet Y>
- Do not redesign product.

SCOPE (read these first):
- @path/to/file1
- @path/to/file2
- git range: <base>..<head>   # if reviewing a diff

IGNORE:
- style nits unrelated to the question
- <out of scope subsystems>

RETURN:
- Verdict on the question
- Evidence (file:line or exact sequence)
- Smallest fix if broken
- What would reverse the verdict
```

### Bad tasks (reject and rewrite)

- “Review everything and tell me if it’s good.”
- “Any concerns?” with no decision.
- Multiple unrelated questions in one call.
- No files, no intended behavior, no “already checked.”

---

## Dispatch procedure

### 1. Prefer real Oracle if usable

If the built-in `oracle` tool is available and not erroring on provider limits, **use it** with the same task text. Stop here.

### 2. Package context (parent does this)

```bash
# If reviewing a branch/phase:
git fetch origin 2>/dev/null || true
BASE=<merge-base or origin/main or task base sha>
HEAD=$(git rev-parse HEAD)
git diff --stat "$BASE"...HEAD
# Optional package for large diffs:
# .agents/skills/subagent-driven-development/scripts/review-package PLAN BASE HEAD
```

Collect:
- Plan/spec paths (if any)
- Exact SHAs
- The filled Oracle task block above

### 3. Dispatch isolated reviewer

Use the **Task** tool (or equivalent general-purpose subagent) with:

- `description`: short label, e.g. `Grok-style oracle review: <topic>`
- `prompt`: full contents of [reviewer-prompt.md](reviewer-prompt.md) with placeholders replaced

**Hard rules for the dispatch:**

- Reviewer is **read-only** on the product checkout (no commits, no “helpful” refactors).
- Reviewer must **not** see the parent’s full chat history—only the packaged task.
- One question per dispatch. Split independent questions into parallel Tasks only if truly independent.

### 4. Consume feedback (Oracle output contract)

Accept the result only if it contains:

| Field | Required |
|-------|----------|
| **Verdict** | Direct answer to the unresolved question |
| **Evidence** | file:line and/or concrete sequence |
| **Impact** | What breaks if ignored |
| **Smallest fix** | If broken; else “none” |
| **Reversal condition** | What evidence would flip the verdict |
| **Out of scope** | Explicitly ignored items (optional but good) |

Severity for any findings attached to the verdict:

- **Critical** — must fix before proceed/merge
- **Important** — should fix before proceed/merge
- **Minor** — note only; do not block unless user says so

### 5. Parent adjudication

1. Fix **Critical** and **Important** (or push back with technical counter-evidence).
2. Do **not** re-ask the same question with a vaguer prompt hoping for approval.
3. One scoped re-review only for the claimed fix range, same input contract.
4. If the reviewer is wrong, document why with code/tests; do not cargo-cult.

---

## Feedback shape (what “good” looks like)

Reviewer returns exactly this structure (see template for full text):

```markdown
## Verdict
<Direct answer to the unresolved question>

## Evidence
- `path:line` — …

## If wrong: failing sequence / invariant break
…

## Smallest fix
…

## Severity of residual issues
- Critical: …
- Important: …
- Minor: …

## What would reverse this verdict
…

## Explicitly out of scope
…
```

**Forbidden reviewer behavior:**

- “LGTM” without evidence
- Rewriting the product/plan
- Fixing code in the review turn
- Expanding into unrelated nits as blockers
- Asking the parent to restate the whole thread

---

## Calibration (Oracle-like restraint)

Use this skill **sparingly**. Good triggers:

- Cross-file concurrency / durability invariant
- Security boundary (auth, path rooting, CSRF)
- Plan lock vs implementation conflict
- Merge gate for a whole phase with known hard edges

Bad triggers:

- Every single commit
- Pure formatting
- Questions answerable by one test run

---

## Placeholders quick reference

| Placeholder | Fill with |
|-------------|-----------|
| `{UNRESOLVED_QUESTION}` | One decision sentence |
| `{ALREADY_CHECKED}` | Bullets of evidence already gathered |
| `{DECISION_IMPACT}` | Branching consequences |
| `{INTENDED_BEHAVIOR}` | Spec/plan truth |
| `{SETTLED_CONSTRAINTS}` | Non-negotiables |
| `{SCOPE_FILES}` | @paths and/or git range |
| `{IGNORE}` | Out-of-scope list |
| `{BASE_SHA}` / `{HEAD_SHA}` | Diff bounds when reviewing commits |

Full prompt: [reviewer-prompt.md](reviewer-prompt.md)

---

## Common mistakes

| Mistake | Fix |
|---------|-----|
| Treating this as free Oracle replacement for all reviews | Keep using `requesting-code-review` for routine task gates |
| Vague task → vague praise | Rewrite until the question is falsifiable |
| Parent implements while “reviewing” | Reviewer stays read-only; parent fixes after |
| Multiple questions in one call | Split or pick the one that blocks the decision |
| Ignoring Important findings | Same bar as Oracle/plan approval: no silent proceed |

---

## Later evaluation (do not do in the critical path)

When deliberately comparing to real Oracle:

1. Same packaged task → Oracle and this skill.
2. Score: correct invariant catch, false positives, actionability, token/latency cost.
3. Record lessons in `docs/memory/` only if the comparison changes process rules.

Do **not** block shipping on that experiment.
