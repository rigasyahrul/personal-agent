---
name: compounding
description: >-
  Turn a finished unit of work into durable scope knowledge via a human-approved
  compound proposal. Codify first (AGENTS standing bullets); selective memory
  detail + thin lessons index only when evidence is needed. Never write disk
  directly — emit proposal JSON items only.
---

# Compounding (default skill)

Flip diminishing returns into accelerating ones: each task should make the next
easier by codifying learnings into **scope instructions and memory**, not by
appending a longer diary.

**Compound ≠ diary.** A standing bullet that prevents the mistake again is the
win. Full session transcripts, play-by-plays, and “what we did today” dumps are
out of scope. If nothing reusable was learned, emit **no items** (empty proposal
is valid) — do not manufacture artifacts.

You run only when the user explicitly triggers Compound. Tools are disabled for
this turn. Your entire output must be **proposal items JSON** matching the server
schema. Do **not** call write tools, edit `source/**`, or touch `.agents/**`.

## The loop

```
Finished work → Extract non-obvious learnings
  → Codify first into AGENTS.md (short standing bullets) when in scope
  → (only if needed) memory_detail evidence file
  → thin lessons_index_row linking that detail
  → Human approves → server publishes → next session loads them
```

## Scope write rules

| Session home | Allowed kinds / paths |
|--------------|------------------------|
| **project**  | `agents_patch` → `AGENTS.md`; `memory_detail` → `memory/YYYYMMDD-HHmm-slug.md`; `lessons_index_row` → `memory/lessons.md` |
| **global**   | same as project (global root) |
| **vault**    | **memory only** — `memory_detail` + `lessons_index_row`. **Never** `agents_patch` (no vault AGENTS). |

Forbidden always via compound:

- `source/**` (notes stay promote/direct/review)
- `.agents/**` (skill files are not compound targets)
- `SOUL.md` / `SYSTEM.md` (human direct only in slice 1)
- Path escape, absolute paths, `..` segments

## Proposal item schema (only these kinds)

Emit a JSON array of items. Each item:

```json
{
  "kind": "agents_patch | memory_detail | lessons_index_row",
  "path": "AGENTS.md | memory/YYYYMMDD-HHmm-slug.md | memory/lessons.md",
  "action": "create | update",
  "title": "optional short title",
  "content": "full markdown body after edit",
  "content_sha256": "hex sha256 of content"
}
```

Kinds only: **`agents_patch`**, **`memory_detail`**, **`lessons_index_row`**.
No other kinds. Prefer fewer high-quality items over many weak ones.

### `agents_patch` (project | global only)

- Path must be exactly `AGENTS.md`.
- **Codify first:** short standing-rule bullets agents will load every session.
- Prefer principles and gates over micro-scripts.
- **Preserve the Memory pointer.** AGENTS must keep (or seed) a clear pointer
  from a `## Memory` (or equivalent) section to `memory/lessons.md` so the next
  agent can open the lessons index. Never strip that pointer when patching.
- Do not paste full lesson bodies into AGENTS.

### `memory_detail` (selective)

- Path pattern: `memory/YYYYMMDD-HHmm-<slug>.md` (local timestamp + kebab slug).
- Write **only when** evidence is needed: non-obvious trap with rationale that
  will not fit a short AGENTS bullet; correction trail for later synthesis;
  multi-system Wrong / What worked narrative.
- Skip when the durable AGENTS bullet (or nothing reusable) already captures it.
- Keep factual and short. Link evidence (thread/commit) instead of pasting
  transcripts, full diffs, or multi-phase play-by-plays.
- Suggested sections: Task / Wrong / What worked / Rule / Codified into / Evidence.
- Use **path wikilinks** with optional title mask, e.g. `[[memory/lessons.md|Lessons]]`
  or `[[memory/20260822-1530-example-trap.md|Example trap]]`. Prefer path targets
  over bare titles so links resolve across Obsidian and the app.

### `lessons_index_row` (thin index)

- Path must be `memory/lessons.md`.
- **Thin rows only** — topic map + newest-first bullets linking detail paths.
- **Never** put full lesson bodies in the index.
- When any `memory_detail` is present, include a matching `lessons_index_row`
  that references that detail path (wikilink).
- Server merges/prepends by detail path; still propose a clear row for the new
  detail. Do not invent a second parallel index tree.

## Codify-first decision order

1. **AGENTS standing bullet** (project/global) if the learning is a durable rule.
2. Else if vault (or AGENTS already sufficient): skip agents_patch.
3. **memory_detail** only if selective evidence is warranted.
4. **lessons_index_row** whenever a new/updated detail is proposed.
5. If mechanical work taught nothing → **zero items**.

## Anti-patterns (reject yourself before emit)

- Compound as **diary** / session wrap dump / full chat transcript.
- Full bodies inside `lessons.md`.
- `agents_patch` on **vault** scope.
- Removing or emptying the AGENTS **Memory** → `memory/lessons.md` pointer.
- Writing or proposing paths under `source/**` or `.agents/**`.
- Duplicate near-identical details instead of superseding one path.
- Restating git history, PR text, or the plan with no new rule.

## Compounding test

For each item ask: *If a new agent hits this next month, does this let them skip
the investigation?* If no, sharpen the durable AGENTS bullet or drop the item.
A memory_detail alone almost never passes — pair with codify-first when scope allows.

## Output contract

Return **only** the JSON items array (or a single top-level object the host
unwraps to items — follow the compound system envelope if provided). No prose
before/after. No tool calls. No markdown fence unless the host envelope requires
one. Human review approves before any disk write.
