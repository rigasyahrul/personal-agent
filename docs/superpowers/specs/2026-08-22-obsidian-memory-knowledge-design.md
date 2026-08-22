# Obsidian-like Knowledge + Compounding Memory — Design Spec

**Status:** Approved (2026-08-22) — plan at `docs/superpowers/plans/2026-08-22-obsidian-memory-knowledge.md`  
**Date:** 2026-08-22  
**Repo:** `personal-agent`  
**Priority order:** **B → C → A** (compounding memory → Obsidian shape → scoped search)  
**Architecture approach:** Files-first, SQLite as index (Approach 1)

---

## 1. Purpose

Extend the working vault → project → session product into an **Obsidian-shaped knowledge system** with **Superpowers-style compounding memory**, so:

1. Agents get smarter over time via human-gated lessons (not session dumps).
2. Notes on disk are portable Markdown with frontmatter, path wikilinks, and backlinks.
3. Humans and agents can **search within a project** (vault/global search later).

This builds on v1 (`source/**`, promote/direct, review) without replacing it. `memory/` stops being an empty reserved dir; root instruction files and per-scope compounding skills become first-class.

---

## 2. Goals and non-goals

### Goals (slice 1)

1. **Compounding memory** at project, vault, and global scopes (thin index + selective detail).
2. **AGENTS.md / SOUL.md / SYSTEM.md** at **project** and **global** roots; load rules with **no silent fallback**.
3. **Per-scope compounding skill** at `.agents/skills/compounding/SKILL.md`, seeded on create; loaded **only** on explicit Compound.
4. **Human-gated compound proposals** in-session (approve / edit / reject) before any memory/AGENTS write.
5. **Obsidian-compatible** Markdown: YAML frontmatter, `[[path|title]]` wikilinks, **backlinks panel** (no graph canvas yet).
6. Links and backlinks across **project `source/` + `memory/` + root instruction files**.
7. **Project-scoped search** (UI + agent tool) over that corpus via SQLite FTS.
8. Files remain the system of record; SQLite indexes paths, hashes, frontmatter, edges, FTS.

### Non-goals (slice 1)

- Full knowledge-graph canvas UI
- Title-only wikilink resolution
- Vault/global full-text search grants (spec reserved only)
- Vault-level SOUL / SYSTEM / AGENTS
- Project ← global fallback for SOUL / SYSTEM / AGENTS / memory
- Auto-compound after agent runs
- Separate compound proposal inbox page
- Agent free-write to AGENTS/memory without approve
- Compound writes into `source/**` (library stays promote/direct)
- Live external FS watcher as writer; multi-device Obsidian sync
- Searching `.agents/**` or session workspaces as knowledge
- Multi-user / sharing

---

## 3. Priority and phasing

| Priority | Pillar | Slice 1 |
|----------|--------|---------|
| **B** | Compounding memory + instruction load | **Implement** |
| **C** | Obsidian shape (frontmatter, path wikilinks, backlinks) | **Implement** |
| **A** | Scoped search | **Project-only implement**; vault/global grants **spec only** |

### Implementation phases

| Phase | Delivers |
|-------|----------|
| **P0** | Layout seed: root files, memory trees, `.agents/skills/compounding`; prompt load per home |
| **P1** | Compound: skill load, proposal API, in-session review card, publish + timestamps |
| **P2** | Note index: frontmatter, wikilink edges, backlinks panel (source + memory + instructions) |
| **P3** | Project FTS + UI + agent `search_project` tool |
| **Later** | Vault/global search grants; graph canvas; synthesize-memory; compound inbox; SOUL/SYSTEM via compound |

---

## 4. Filesystem layout

```
files/global/
  SOUL.md
  SYSTEM.md
  AGENTS.md
  .agents/skills/compounding/SKILL.md
  memory/
    lessons.md
    YYYYMMDD-HHmm-slug.md
  projects/{project_id}/          # unfiled projects
  sessions/{session_id}/

files/vaults/{vault_id}/
  .agents/skills/compounding/SKILL.md
  memory/
    lessons.md
    YYYYMMDD-HHmm-slug.md
  projects/{project_id}/
    SOUL.md
    SYSTEM.md
    AGENTS.md
    .agents/skills/compounding/SKILL.md
    source/**
    memory/
      lessons.md
      YYYYMMDD-HHmm-slug.md
    sessions/{session_id}/
  sessions/{session_id}/
```

### Layout rules

1. **IDs in path segments** for vault/project/session dirs (unchanged). Display names never move trees.
2. **`source/**`** — freeform library (existing promote/direct).
3. **`memory/**`** — compounding evidence only; not a second freeform dump of chat.
4. **Root instruction files** — `SOUL.md`, `SYSTEM.md`, `AGENTS.md` at project root and global root only.
5. **`.agents/`** — agent machinery (skills). Not part of knowledge graph / FTS corpus.
6. **Legacy `soul/` directory** under projects (v1 `EnsureProjectDirs`) — keep for backward compat; **product surface is root `SOUL.md`**, not `soul/**`. Do not promote into `soul/` or `memory/` as source paths.
7. **Export** — copy project tree excluding `sessions/`; remains Obsidian-openable.

### Seed on create

| Event | Seed |
|-------|------|
| App init / global ensure | Global `SOUL.md`, `SYSTEM.md`, `AGENTS.md` (empty or minimal stubs), `memory/lessons.md` (empty index scaffold), `.agents/skills/compounding/SKILL.md` from app default |
| Vault create | Vault `memory/lessons.md`, vault `.agents/skills/compounding/SKILL.md` |
| Project create | Project root three instruction files, `memory/lessons.md`, project `.agents/skills/compounding/SKILL.md`; existing `source/`, `memory/`, `soul/` dirs |

Empty instruction file = **omit layer** at load time (not an error).

---

## 5. File roles

| Path | Role | Writers (slice 1) |
|------|------|-------------------|
| `SOUL.md` | Identity / values / voice for scope | Human direct edit API/UI |
| `SYSTEM.md` | User system instruction for scope | Human direct edit API/UI |
| `AGENTS.md` | Standing agent rules (hot path) + **Memory → lessons pointer** | Human + **compound approve** patch |
| `memory/lessons.md` | Thin index: title + path link + one-line summary | Compound approve (+ human edit) |
| `memory/YYYYMMDD-HHmm-slug.md` | Selective detail evidence | Compound approve |
| `source/**` | Library notes | Promote + direct create (existing) |
| `.agents/skills/compounding/SKILL.md` | How to compound (per-scope override) | Seeded default; human may edit later |

### AGENTS ↔ memory communication

```
Durable lesson
  → 1. Codify short rule into AGENTS.md (preferred)
  → 2. If evidence needed: memory/YYYYMMDD-HHmm-slug.md
  → 3. Thin row in memory/lessons.md → detail path
```

- **AGENTS does not paste full lesson bodies.**
- **Every non-empty AGENTS.md** includes a stable **Memory** section pointing at `[[memory/lessons|lessons.md]]` so agents know lessons may already exist.
- On first create / compound that touches AGENTS: **seed Memory section if missing**; compound **must not strip** it.
- Detail frontmatter may include `codified_into: [AGENTS.md]`.
- Wikilinks are path-based with optional title mask: `[[memory/20260820-ship|Ship means push]]`.

### Compounding is a skill, not AGENTS prose

- Long “how to compound” procedure lives **only** in the compounding skill.
- AGENTS keeps standing rules + Memory pointer only.
- Skill loaded **only** on explicit user Compound action (UI and/or chat intent).

---

## 6. Session load rules (no silent widening)

| Session `home` | SOUL / SYSTEM / AGENTS | Memory auto-load (thin index) | Compound skill path |
|----------------|------------------------|-------------------------------|---------------------|
| **project** | **That project only**; empty = omit; **no global fallback** | **Project `memory/lessons.md` only** (not vault, not global) | Project `.agents/skills/compounding/SKILL.md` |
| **vault** | **Global** only; empty = omit | **Vault `memory/lessons.md` only** | Vault skill |
| **global** | **Global** only; empty = omit | **Global `memory/lessons.md` only** | Global skill |

Vault has **no** vault-level SOUL/SYSTEM/AGENTS.

### Prompt assembly order

1. **App runtime wrapper** (not user-editable): tools, safety, session home/ids, path roots, “compound only on explicit user action”, tool grants.
2. Scope `SYSTEM.md` (if non-empty)
3. Scope `SOUL.md` (if non-empty)
4. Scope `AGENTS.md` (if non-empty)
5. Scope `memory/lessons.md` thin index (if present)
6. Conversation messages + tool results

**Never** bulk-inject all `memory/*.md` detail bodies or entire `source/**`.

### Budgets

- Per-file and total character caps for injected markdown.
- Prefer keep order under pressure: AGENTS > SYSTEM > SOUL > lessons (truncate lower-priority with explicit “truncated” note in wrapper).
- Exact numeric caps: implementation plan.

### Missing skill file

If scoped skill missing/unreadable: use **embedded app default** for that Compound run; optionally re-seed on next ensure. Compound must not hard-fail solely because the override file is missing.

---

## 7. Compound proposals

### Trigger

- **Explicit only:** UI **Compound** control and/or user chat intent handled as compound.
- **No** auto-propose after every run.

### Flow

1. User triggers Compound.
2. App loads scoped compounding skill into that turn.
3. Model emits a **structured proposal** (not raw FS writes).
4. Server validates → status `pending`.
5. In-session **review card**: per-item approve / edit / reject.
6. On confirm: publish approved items → disk + index; set timestamps.

### Proposal model

```
CompoundProposal
  id
  session_id
  scope: project | vault | global
  status: pending | approved | rejected | failed
  created_at      # proposal created / pending
  decided_at?     # user approved or rejected
  finished_at?    # publish fully done or terminal failure (primary latency metric: time-to-finish)
  error?
  items[]:
    kind: agents_patch | memory_detail | lessons_index_row
    path              # relative to scope root
    action: create | update
    title?
    content_full | unified_diff
    content_sha256
```

**Slice 1 kinds:** `agents_patch`, `memory_detail`, `lessons_index_row` only.  
SOUL/SYSTEM are **not** compound targets in slice 1 (human direct only).

### Write scope by session

| Session | May write |
|---------|-----------|
| project | That project: `AGENTS.md`, `memory/**` |
| vault | That vault: `memory/**` only (no AGENTS at vault) |
| global | Global: `AGENTS.md`, `memory/**` |

### Validation (reject before pending or at approve)

1. Paths under allowed roots only; no escape; no `.agents/` writes via compound; **no `source/**` via compound**.
2. `memory_detail` path pattern `memory/YYYYMMDD-HHmm-<slug>.md` (server may assign timestamp).
3. If detail item present → matching `lessons_index_row` required.
4. AGENTS-only short rule without detail → OK.
5. Must not remove Memory → lessons pointer from AGENTS.
6. Size limits; reject obvious session-transcript dumps (max bytes + skill rules).
7. Index row target must match detail path when both present.

### Publish

Reuse promote/direct spirit: freeze → staging → atomic write → DB finalize → reindex (frontmatter, links, FTS).  
Idempotent approve via `request_key`.  
API remains **sole writer**.

### Metrics

- Store `created_at`, `decided_at`, `finished_at`.
- Primary derived metric: **time-to-finish** = `finished_at - created_at` (not a stored column).

---

## 8. Obsidian shape (notes, links, backlinks)

### Corpus kinds (project)

| Kind | Path | Backlinks + FTS |
|------|------|-----------------|
| Library | `source/**/*.md` | Yes |
| Memory detail | `memory/YYYYMMDD-HHmm-*.md` | Yes |
| Memory index | `memory/lessons.md` | Yes |
| Instructions | root `AGENTS.md`, `SOUL.md`, `SYSTEM.md` | Yes |
| Skills | `.agents/**` | **No** |

### Frontmatter (recommended)

```yaml
---
title: Hub startSession soft-fail
date: 2026-08-20
tags: [hub, sessions]
codified_into: [AGENTS.md]
---
```

Missing `title` → filename stem for display.

### Wikilinks

- **Resolve by relative path** under scope root (project root for project notes).
- Optional display alias: `[[memory/20260820-ship|Ship means push]]`.
- `.md` extension optional in link text.
- **No** title-only resolution in slice 1.
- Cross-project / cross-vault resolution: out of scope (unresolved edge OK).

### Link index

On successful publish of any indexed path:

1. Parse `[[target|alias?]]`
2. Normalize to logical relative path
3. Upsert edges `from_note_id → to_path` (+ `to_note_id` when exists)
4. Keep unresolved path edges for later fix-up

### Backlinks panel

- When viewing a note: list inbound links **within the same scope** (project slice 1).
- Show display title, path, optional snippet line.
- Click opens target.
- Empty: “No backlinks yet.”
- Graph canvas: **later** (edges already stored).

---

## 9. Project search (slice 1)

**Scope:** current project only.  
**Corpus:** `source/**/*.md` + `memory/**/*.md` + root `AGENTS.md` / `SOUL.md` / `SYSTEM.md`.  
**Excluded:** `.agents/**`, `sessions/**`, other projects, vault/global trees.

| Surface | Behavior |
|---------|----------|
| UI | Search box → ranked hits (title, path, snippet) |
| Agent tool | `search_project` (final name in plan) — query → paths + snippets |
| Backend | SQLite FTS5 (or equivalent) maintained on publish |

**Ranking (v1):** title/path match > body; weak `updated_at` tie-break.  
**Limits:** max results, max snippet length.

### Deferred (spec only)

Session-start optional grants for vault-wide / global search — default **off**; not implemented in slice 1.

---

## 10. Data model additions (conceptual)

Extend beyond v1 `notes` (source-only) with a separate **`knowledge_notes`** index (stable independent ids) for source mirrors, memory, and instruction files.

**Path namespaces (locked after review):**

| Store | Path meaning | Example |
|-------|--------------|---------|
| v1 `notes.relative_path` | Source-relative (unchanged) | `articles/intro.md` |
| `knowledge_notes.relative_path` | Scope-root-relative | `source/articles/intro.md` |

Source mirror: `knowledge.relative_path = "source/" + notes.relative_path`. Optional `source_note_id` FK. Never assume `notes.id == knowledge_notes.id`.

New:

- `compound_proposals` (items JSON + status machine; validate on create **and** decide **and** publish)
- `note_links` (from/to **knowledge** ids only)
- FTS virtual table on `knowledge_notes.id`
- Partial unique indexes for project/vault/global paths (SQLite NULL-safe)

Exact DDL: implementation plan Canonical contracts. Preserve promote/direct/review contracts for v1 `notes`.
---

## 11. API surface (conceptual)

| Area | Endpoints (all under **`/api/v1`**) |
|------|-------------------------------------|
| Instructions | GET/PUT project + global SOUL/SYSTEM/AGENTS |
| Knowledge | GET tree/read by **scope-root** path |
| Compound | POST start; GET proposal; POST decide (revalidate final items) |
| Backlinks | GET `knowledge/backlinks?path=` or `knowledge_id=`; optional notes.id convenience resolve |
| Search | GET project search?q= (returns knowledge_id + path) |
| Existing | promote/direct/notes/sessions unchanged in spirit |

Auth: owner session + CSRF on mutations (existing). Knowledge path validation is separate from promote `ValidateRelPath`.

**Knowledge FS:** do not open `memory/**` through stock `fsroot`+`ValidateRelPath` (reserved components). Use MemoryDir/SourceDir sub-roots or a knowledge opener; never loosen promote reserved `memory`/`soul`. Compound scope/ids come only from the session row. Migrator must apply `002` (not hard-code `001` only).

---

## 12. UI surface (slice 1)

1. **Project hub / notes:** browse source + memory; open note with **backlinks** panel; project search field.
2. **Instruction editors** (minimal): view/edit SOUL, SYSTEM, AGENTS for project and global settings/desk.
3. **Session:** Compound control; proposal review card; agent tools for read/search within grants.
4. **Memory rail:** replace non-persistent textarea with real scoped memory summary or link into memory tree (no fake local-only state).
5. Tokens-first UI (`app.css`); frontend-ui-craft for visible surfaces.

---

## 13. Agent tools (slice 1, project home)

| Tool | Purpose |
|------|---------|
| Read knowledge file by relative path | `source/`, `memory/`, root instructions |
| List/tree knowledge roots | Navigate library + memory |
| `search_project` | FTS over project corpus |
| Existing workspace tools | Unchanged; workspace ≠ knowledge |
| Compound propose | Only when user explicitly triggered compound turn |

Writes to knowledge still via API publish paths / compound approve — not unrestricted tool write to AGENTS/memory.

---

## 14. Default compounding skill (requirements)

Shipped embedded default, copied to each scope path on create. Must specify:

1. Codify first into AGENTS (short bullets).
2. Selective detail files only when evidence needed.
3. Thin `lessons.md` row only (never full bodies in index).
4. Preserve/seed Memory → lessons pointer in AGENTS.
5. Path wikilinks with optional title mask.
6. Output **proposal items only** matching server schema.
7. Compound ≠ diary / no full transcripts.
8. Vault compound: memory only; no fake vault AGENTS.

Per-scope file may diverge after seed.

---

## 15. Testing focus

- Load isolation: empty project instructions ≠ global bleed; project memory ≠ vault memory.
- Skill seed on global/vault/project create; override used when present; missing → embedded default.
- Compound validation: diary dump reject; detail requires index row; Memory pointer preserved; path escape reject.
- Timestamps: `finished_at` set on terminal success/failure after publish attempt.
- Wikilink parse + backlinks within project.
- FTS returns only current project corpus.
- UI: proposal card approve path; backlinks panel; search box; no persistent-fake memory textarea.
- Existing promote/direct/review/session focus tests stay green.

---

## 16. Success criteria

1. Explicit Compound + approve updates project (or vault/global) memory/AGENTS on disk; next session loads them.
2. AGENTS points at `memory/lessons.md`; agent can open a detail by path.
3. Opening a note shows backlinks from other in-scope notes.
4. Project search finds content in source + memory for human and agent.
5. Files remain Obsidian-openable (md + frontmatter + `[[path|title]]`).
6. No auto-compound; no cross-scope instruction fallback.

---

## 17. Relation to prior design

Supersedes the “memory/ reserved empty” and “identity TBD” rows of `2026-08-12-personal-agent-design.md` for product direction, without invalidating v1 source/promote/review.  
v1 non-goal “full-text search across the library” is **narrowed**: project FTS is now in slice 1; vault/global search remains deferred.

---

## 18. Open implementation choices (plan may fix)

**Already locked in plan Canonical contracts (do not reopen):** dual table (`knowledge_notes` + v1 `notes`); path namespaces + `source/` mirror; `/api/v1`; compound validate on create+decide+publish; FTS5 in-process.

Still free within those locks:

- Whether instruction PUT reuses publish machine or a thinner atomic write helper sharing reindex hooks.
- Chat-intent detector vs UI-only Compound trigger for P1 (items-POST path required either way).
- Exact numeric prompt byte caps (defaults in plan).

Spec wins on product behavior; plan Canonical contracts win on engineering locks.
