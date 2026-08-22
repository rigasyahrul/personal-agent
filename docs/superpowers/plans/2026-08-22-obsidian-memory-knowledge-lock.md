# Plan lock: Obsidian Memory + Knowledge

**Spec (source of truth):** `docs/superpowers/specs/2026-08-22-obsidian-memory-knowledge-design.md`  
**Assembled plan (target):** `docs/superpowers/plans/2026-08-22-obsidian-memory-knowledge.md`  
**Drafts dir:** `docs/superpowers/plans/2026-08-22-obsidian-memory-knowledge-drafts/`  
**Header / contracts (master-owned):** `docs/superpowers/plans/2026-08-22-obsidian-memory-knowledge-header.md`

## Scope freeze

In scope (slice 1, Approach 1 — files-first, SQLite index):

1. **P0** Layout seed + prompt load: project/global `SOUL.md` `SYSTEM.md` `AGENTS.md`; project/vault/global `memory/` + lessons index; `.agents/skills/compounding/SKILL.md` seed; session load rules (no fallback)
2. **P1** Explicit compound only: skill load, proposal API, in-session review card, publish to AGENTS/memory, timestamps `created_at` `decided_at` `finished_at`
3. **P2** Obsidian index: frontmatter, path wikilinks `[[path|title]]`, backlinks panel (source + memory + instructions)
4. **P3** Project-scoped FTS + UI + agent search tool

Out of scope: graph canvas, title-only wikilinks, vault/global search grants (document stubs only), vault SOUL/SYSTEM/AGENTS, auto-compound, compound inbox page, compound → `source/**`, external FS watcher writer, multi-user.

## Draft file list (disjoint task ranges)

| Draft | File | Tasks | Phase focus |
|-------|------|-------|-------------|
| A | `…-drafts/A-p0-layout-prompt.md` | 1–12 | Layout, seed, migrations skeleton, instruction read, prompt assembly |
| B | `…-drafts/B-p1-compound.md` | 20–35 | Compound skill, proposal store/API, publish, session UI card |
| C | `…-drafts/C-p2-obsidian-links.md` | 40–52 | Knowledge note index, frontmatter, wikilinks, backlinks API/UI |
| D | `…-drafts/D-p3-search.md` | 60–72 | FTS5, search API, UI, agent tool, final verify gates |

## Authority rules

1. **Spec wins** over drafts on any product conflict.
2. **Canonical contracts** in assembled plan header win over draft prose/snippets.
3. Drafts: checkbox steps, real repo paths, TDD (failing test → impl → pass → commit).
4. Worker dispatch for drafts **and** later implementation review: only  
   `amp -m grok45 --no-archive-after-execute -x '…'`  
   **Never** Task tool, OpenAI subagents, built-in oracle, or `amp -m grok45 -ox`.
5. **One long Grok worker at a time.** If no draft file on disk ~3–4 min or stream times out: **do not triple-retry** — master writes remaining drafts and assembles.
6. Go + SQLite + existing publish/fsroot patterns; web Node `>=22 <23`.
7. Do not break promote/direct/review/session-focus invariants.
8. Ship = push only when user asks to ship; docs commits may push when user allows.

## File map (expected — refine in header)

| Path | Role |
|------|------|
| `internal/layout/layout.go` | Roots for instructions, memory, `.agents` |
| `internal/db/migrations/00N_*.sql` | knowledge notes / links / FTS / compound tables |
| `internal/domain/` | CompoundProposal, knowledge note kinds |
| `internal/store/` | instructions, memory, compound, links, search |
| `internal/agent/` | prompt assembly, compounding skill load, search tool |
| `internal/agent/skills/compounding/SKILL.md` | Embedded default skill body |
| `internal/publish/` or new writer | Atomic write + reindex hooks for instructions/memory |
| `internal/httpapi/` | instruction, compound, backlinks, search handlers |
| `web/src/components/` | CompoundReviewCard, BacklinksPanel, search field, instruction editors |
| `web/src/components/ProjectRail.svelte` | Replace fake memory textarea |
| Tests | `*_test.go`, co-located `*.test.ts` |

## Parallel draft instructions

Each draft agent writes **ONLY** its draft file under `…-drafts/` with full Task N sections (steps with code, run commands, commits). No assembled plan. No product implementation. Read spec + lock + header contracts first.
