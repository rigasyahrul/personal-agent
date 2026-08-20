# Lessons learned (compounding engineering)

> **Hot path (every session):** `AGENTS.md` → Standing rules.  
> **This file:** long-term evidence diary — agents read it when compounding, synthesizing, or debugging a repeated trap. Do **not** dump the whole diary into every prompt.  
> Pattern: [Compounding Engineering](https://www.agentic-patterns.com/patterns/compounding-engineering-pattern/)

**Canonical path:** `docs/memory/lessons.md` (stable name; not a creation date).  
One file only under `docs/memory/`. No `solutions/`, no process/planning subfolders.  
Plans/specs/locks live under `docs/superpowers/` only.

## How memory works

| Layer | Path | When loaded |
|-------|------|-------------|
| Standing rules (policy) | `AGENTS.md` | Almost every session |
| Skills (workflows) | `.agents/skills/` | When the task matches |
| Lessons (evidence) | **this file** | Compound / synthesize / repeated trap |

**After non-trivial work or a user correction:** run `compounding-engineering` → prepend a lesson here (newest first) → optional short bullet in `AGENTS.md` if it must load every session.  
**When ~10+ new sections or a theme recurs 3+ times:** run `synthesize-memory`.  
**Same learning twice:** update or supersede; do not duplicate.

## Index (topic → latest lesson)

Scan this first. Jump to the matching `###` section below only when you need detail.

| Topic | Latest lesson |
|-------|----------------|
| multi-agent plans / writing-plans | 2026-08-19 — Big plans: lock + parallel drafts, don't solo-stall |
| skills / Skill tool | 2026-08-19 — Big plans: lock + parallel drafts, don't solo-stall |
| ui / product model | 2026-08-19 — Big plans: lock + parallel drafts, don't solo-stall |
| memory / docs layout | 2026-08-19 — Lessons file is stable `lessons.md` |
| ui / spa / focus | 2026-08-19 — Sessions chat focus + Docker live-reload |
| docker / dev loop | 2026-08-19 — Sessions chat focus + Docker live-reload |
| make / makefile | 2026-08-19 — Default `make` is help; build binary is gitignored |
| darwin / fs / shell | 2026-08-19 — macOS local `make test` platform gaps |
| git / merge | 2026-08-20 — Master Grok spawn: local -x, isolate worktrees |
| review / grok | 2026-08-20 — Master Grok spawn: local -x, isolate worktrees |
| ship / push | 2026-08-12 — Ship means push |
| multi-agent plans (authority) | 2026-08-12 — Multi-agent plans need one authority |
| master / spawn / grok45 | 2026-08-20 — Master Grok spawn: local -x, isolate worktrees |
| amp / orb-execute | 2026-08-20 — Master Grok spawn: local -x, isolate worktrees |
| node / web | 2026-08-20 — Master Grok spawn: local -x, isolate worktrees |
| review / escaped literals | 2026-08-13 — Verify escaped literals at the byte level |

## Where Superpowers artifacts go

| Artifact | Path |
|----------|------|
| Design specs | `docs/superpowers/specs/YYYY-MM-DD-*-design.md` |
| Implementation plans | `docs/superpowers/plans/YYYY-MM-DD-*.md` |
| Plan lock (optional) | `docs/superpowers/plans/YYYY-MM-DD-*-lock.md` |
| Plan phase drafts (optional) | `docs/superpowers/plans/YYYY-MM-DD-*-drafts/` |
| Handoffs | `docs/superpowers/HANDOFF-*.md` |
| **Lessons / compound notes** | **`docs/memory/lessons.md`** (this file) |
| Standing agent rules | **`AGENTS.md`** |

---

## Lessons (newest first)

### 2026-08-20 — Master Grok spawn: local `-x`, isolate worktrees

**Tags:** master, spawn, grok45, amp, orb-execute, git, merge, review, node, web

**Task:** UI Svelte redesign master execution (phases A–F): spawn Grok 4.5 workers + consulting-grok-review, merge/ship each phase.

**Wrong / mistakes:**
1. **`amp -m grok45 -ox` became invalid** mid-run (`Agent mode is invalid`) while the same mode still works **without** `-ox`. Early phases used `-ox` successfully; later CLI/plugin load path broke orb-execute for plugin agent modes. Master almost stalled on review dispatch.
2. **Local `amp -m grok45 -x` runs in the master's workspace** (not a separate orb checkout). Workers and master fought over the same tree (wrong branch, dirty board commits on worker branches, accidental board edits on worker CWD).
3. **Board-only commits on `main` after a worker branched** (or worker board commits forked from an older tip) made pure `git merge --ff-only` fail — same class as the earlier v1 board tip lesson; Phase D needed **feature-only cherry-pick**.
4. **Web tests require Node 22** (`engines: >=22 <23`). Orb default PATH often hits Node 20 first → Vitest fails on `styleText` / engine warnings. Must put Node 22 first (`/usr/bin/node` when v22, or `~/.local/node-v22…`).
5. **Master paused after Phase C** while waiting for the next loop tick instead of immediately verify → review → merge → next phase (user had to nudge).

**What worked:**
1. **Dispatch:** `amp -m grok45 --no-archive-after-execute -x "$(cat promptfile)"` (labels optional). Confirm thread via `amp threads list`. Prefer writing the worker/review prompt to a file.
2. **Fallback order:** try `amp -m grok45 -ox …` once; on `Agent mode is invalid` / plugin-mode failure, **immediately** fall back to local `-x` (no silent stall).
3. **Master isolation:** keep master's board/merge work in a **separate git worktree** (e.g. `/tmp/…-master-main` on `origin/main`). Never rely on the primary checkout while a local Grok worker is running.
4. **Merge:** prefer FF when worker is ancestor of main; else **cherry-pick feature commits only** (skip worker board noise) or merge commit; always re-run gates on the merge result.
5. **Review:** new Grok thread + filled `reviewer-prompt` / Oracle contract; poll `amp threads markdown T-…` until `## Verdict`; fix Critical/Important (e.g. Phase E hub `$effect` on `projectId`) before accept.
6. **Verify:** Node 22 first on `PATH` → `make web-test` / `make web-build` → `go test ./...` when Go touched; `web/dist` must exist for static tests (gitignored).

**Rule (next agent):**
- Spawn Grok workers/reviews with **`amp -m grok45 --no-archive-after-execute -x '…'`**. Treat **`-ox` as optional/best-effort** — if invalid, use local `-x` and isolate with worktrees.
- **Master never shares a dirty product checkout with a local worker.** Board + merge only from a clean main worktree.
- **Do not stop between phases** when the next phase is unblocked: verify → consulting-grok-review → merge/push board → dispatch next.
- **Node 22 on PATH** before any `make web-test` / Vitest claim.
- Merge policy: FF if possible; else cherry-pick **product** commits or merge commit — never force-push worker; never lose board history without intent.

**Codified into:**
- `docs/memory/lessons.md` (this section)
- `AGENTS.md` (standing bullets: grok spawn + master worktree + Node 22 web)
- `.agents/skills/consulting-grok-review/SKILL.md` (dispatch via grok45 thread, not Task; `-ox` fallback)
- `docs/superpowers/HANDOFF-ui-svelte-redesign-master.md` (spawn/isolation notes)

**Evidence:** Master thread https://ampcode.com/threads/T-01a01a38-e05a-74e6-9e81-dc97622bab29 ; board `docs/superpowers/STATUS-ui-svelte-redesign.md` (A–F done @ `468e571`); sample review https://ampcode.com/threads/T-01a01d20-827d-73cf-b4f0-764e69b778d5 ; plugin `.amp/plugins/grok-45-mode.ts`.

**Related:** supersedes the **dispatch command** half of “consulting-grok-review via Grok thread” (2026-08-19) regarding mandatory `-ox`; still forbids Task/OpenAI/oracle substitutes. Extends “Master merge: board tip can block pure FF” (2026-08-19) with cherry-pick-of-feature-only when worker board commits diverge.

---

### 2026-08-19 — Big plans: lock + parallel drafts, don't solo-stall

**Tags:** plans, multi-agent, writing-plans, skills, ui

**Task:** Brainstorm + write implementation plan for Svelte UI redesign (context shell, vault UX, docker-dev HMR). Session wrap via compounding-engineering.

**Wrong / mistakes:**
1. After user approved the design, started **writing-plans solo** and went quiet for a long stretch gathering context / drafting in one thread — user had to interrupt (“why not subagent delivery?”). The multi-agent plan lesson already required **lock + phase drafts + assemble**; delay was process failure, not missing knowledge.
2. Amp **Skill tool** returned `compounding-engineering` “not found” even though the skill lives under `.agents/skills/` and is listed in `AGENTS.md`. Must **Read SKILL.md and continue**, not stop.
3. Early product talk used “unfiled” without defining it; user confusion until explained as **no vault** (`vault_id` empty). Backend already had vaults; UI gap looked like “missing Vault.”

**What worked:**
1. **Brainstorming hard gate** — design approved before any app code; decisions frozen (Svelte 5, context sidebar Global vs Vault, Home=dashboard, searchable grids, Inter, docker-dev instant UI).
2. After correction: **lock file** → **parallel Task drafts** (phases 01–06) → **one assembled plan** with **Canonical contracts** header that wins over draft drift → commit.
3. Spec path: `docs/superpowers/specs/2026-08-19-ui-svelte-redesign-design.md`. Plan: `docs/superpowers/plans/2026-08-19-ui-svelte-redesign.md` (+ lock + `…-drafts/`).
4. Domain note for implementers: vault APIs exist (`GET/POST /api/v1/vaults`, project `vault_id` immutable); sessions remain **per-project**; vault Sessions/Review = client aggregate/filter.

**Rule (next agent):**
- When `writing-plans` scope is large (many phases / parallel-safe sections): **same turn** write the lock, dispatch **parallel draft agents**, then assemble. Do **not** solo-author a multi-thousand-line plan in silence.
- Canonical contracts section in the assembled plan beats conflicting task snippets.
- Skill tool miss → Read `.agents/skills/<name>/SKILL.md` (already standing).
- “Unfiled” = project with empty `vault_id`; say “no vault” in UI copy when clarity matters. Vault is first-class nav, not a buried optional select.

**Codified into:**
- `docs/memory/lessons.md` (this section)
- `AGENTS.md` (standing bullet: big writing-plans → lock + parallel drafts immediately)
- `.agents/skills/writing-plans/SKILL.md` (large-plan parallel draft gate)
- Spec + plan artifacts under `docs/superpowers/` (already committed)

**Evidence:** Amp thread https://ampcode.com/threads/T-01a019d4-7a0b-76bb-a368-c98695f346f8 ; commits `20b62cd` (spec), `d3a2d3a` (assembled plan)

**Related:** supersedes delay mode of “Multi-agent plans need one authority” (2026-08-12) — authority still required; **also** require parallel draft delivery so the human is not left waiting.

---

### 2026-08-19 — Lessons file is stable `lessons.md`

**Tags:** memory, docs, agents, compounding

**Task:** Make project memory scannable for AI agents; stop the filename date (`2026-08-12`) implying stale or per-day files. Session wrap via `compounding-engineering`.

**Wrong / mistakes:**
- Named the diary after the **creation** day while content kept growing across dates — agents misread freshness or opened/created a new dated file.
- Hardcoded `docs/memory/2026-08-12-lessons.md` in skills/AGENTS/handoffs.
- Template sat mid-file; order was oldest-first with no topic index.
- Assumed “every session updates the lessons file” improves output — only **standing rules** should load every session; the diary is on-demand evidence.
- Risk on wrap: re-prepending a second section for work that already wrote its lesson mid-session.

**What worked:**
1. Stable path `docs/memory/lessons.md`.
2. Newest-first sections + topic **Index** + fixed field template.
3. Hot path = `AGENTS.md`; this file = detail when compounding/synthesizing.
4. **Wrap hygiene:** when the lesson + AGENTS/skill pointers already landed in the same session, wrap = verify codification + attach commit/thread evidence; do **not** duplicate the `###` section.

**Rule (next agent):** Always use **`docs/memory/lessons.md`**. Prepend new lessons (newest first); refresh the Index row for touched topics. Do not create `YYYY-MM-DD-lessons.md` per session. Promote always-on rules to `AGENTS.md`; keep stories here. On “wrap / compound this session”: if a matching lesson already exists from the same work, **update that section** (evidence, supersession) instead of adding another.

**Codified into:**
- `docs/memory/lessons.md` (this file; replaced `2026-08-12-lessons.md`)
- `AGENTS.md` (memory layers + stable-path standing bullet)
- `.agents/skills/compounding-engineering/SKILL.md` (stable path, prepend, Index, Tags template)
- `.agents/skills/synthesize-memory/SKILL.md` (primary path + newest-first reports)
- `docs/superpowers/HANDOFF-master-execution.md`, `docs/superpowers/README.md`

**Evidence:** Amp thread https://ampcode.com/threads/T-01a0199e-6410-72f7-a6e9-1204cc4cad07 ; commit `4aa0ea2` (restructure + ship); wrap update this section 2026-08-19

---

### 2026-08-19 — Sessions chat focus + Docker live-reload

**Tags:** ui, spa, docker, make

**Task:** Fix message textarea losing focus ~500ms after click on sessions chat; make local Docker pick up API+web edits without rebuild thrash.

**Wrong / mistakes:**
1. Assumed “I edited `web/js/…` so localhost:8080 has the fix.” Port 8080 was **Docker** with a **baked** image (`COPY web /app/web`). Host file ≠ served file. First “fix” never reached the browser.
2. Assumed skipping unchanged poll re-renders was enough. Any poll that still called `root.innerHTML = …` (new messages, run status, send disabled) **destroyed** the focused textarea. User felt ~500ms loss = poll RTT, not the 1.5s timer.
3. Put a permanent `../web:/app/web` mount on **production** compose — wrong layer. Prod should stay image-baked; live mounts belong only in a **dev override**.
4. Told the user to pass two compose `-f` flags without a one-command path. Dev file is an override (ports/env/`pa-data` live in the base file) — document that, but expose **`make docker-dev`**.

**What worked:**
1. **Focus:** keep the composer node alive across polls — `patchChat()` updates `ol.messages`, `.run-status`, alert text, and Send `disabled` in place; full `renderChat()` only when the chat shell is missing or session switches. Restore focus/selection only if a full rebuild is unavoidable. Regression: `web/js/pages/sessions.test.js` (“polling does not steal message focus…”).
2. **Prove serve path before claiming UI fixed:** `curl -s http://127.0.0.1:8080/js/pages/sessions.js | wc -c` (or `rg patchChat`) vs host file; `docker exec … wc -c /app/web/…` or `/src/web/…`; `lsof -iTCP:8080 -sTCP:LISTEN`.
3. **Dev compose:** `deploy/docker-compose.dev.yml` overrides the same `personal-agent` service with `Dockerfile.dev` + `air` + `..:/src` + module/build caches. Base prod compose has **no** host source mounts (enforced in `deploy/deploy_test.go`).
4. **UX:** `make docker-dev` / `docker-dev-down` / `docker-dev-logs`; README + `docs/ops/deploy.md` explain override vs prod.

**Rule (next agent):**
- SPA pages that poll must **not** replace focused form controls via `innerHTML` on every tick. Prefer in-place DOM patches; gate full shell rebuilds; add a focus/selection regression test.
- Before claiming a frontend fix works against “localhost”, verify **which process owns the port** and that the **bytes served** include the change.
- Production Compose stays baked. Live API+web reload = **dev override** (`docker-compose.dev.yml` + `air`), not mounts on the prod file. Day-to-day command: **`make docker-dev`** (needs `deploy/.env`).
- New Makefile public targets: `##` help text, `.PHONY`, and the Development `print-help-section` list.

**Codified into:**
- `web/js/pages/sessions.js` (`patchChat` / `paintChat` / skip full rebuild)
- `web/js/pages/sessions.test.js` (focus + in-place update)
- `deploy/Dockerfile.dev`, `deploy/air.toml`, `deploy/docker-compose.dev.yml`
- `deploy/docker-compose.yml` (prod clean), `deploy/deploy_test.go` (prod forbids source mounts; dev requires `..:/src` + air)
- `Makefile` (`docker-dev*`), `README.md`, `docs/ops/deploy.md`
- `.gitignore` (`/tmp/` for air output)
- Standing bullets in `AGENTS.md`

**Evidence:** Amp thread https://ampcode.com/threads/T-01a01972-ac51-727e-be42-738fa7156a3c

---

### 2026-08-19 — Default `make` is help; build binary is gitignored

**Tags:** make, skills, gitignore

**Task:** Bare `make` should print a grouped command menu (RSSM-style), not run tests.

**Wrong / mistakes:**
- First Makefile target was `test`, so bare `make` ran the full suite — surprising for onboarding.
- `make build` writes `./personal-agent` at repo root; it was untracked and not in `.gitignore`.
- `compounding-engineering` (and `synthesize-memory`) live under `.agents/skills/` and are listed in `AGENTS.md`, but Amp’s **Skill tool registry may omit them** — `skill` call returns “not found”. Must Read the SKILL.md and follow it anyway.

**What worked:**
1. `.DEFAULT_GOAL := help` + each public target annotated `target: ## description`.
2. Hard-coded section headers (Common / Development) with portable awk via `$(call print-help-section,…)` — no heredoc-in-recipe, no GNU-only sed; works on Darwin.
3. New targets: add `##` line, list name in the right section’s `print-help-section` call, add to `.PHONY`.
4. When Skill tool misses a project skill: `Read .agents/skills/<name>/SKILL.md` and execute.

**Rule (next agent):**
- Bare `make` must stay help-only. Never put a heavy target first without `.DEFAULT_GOAL`.
- After `make build`, do not commit the binary; root binary name is gitignored.
- Project skills under `.agents/skills/` named in AGENTS.md are mandatory even if the Skill tool says not found — load from disk.

**Codified into:**
- `Makefile` (help default + `print-help-section`)
- `.gitignore` (`/personal-agent`)
- `AGENTS.md` standing bullets
- Spec/plan: `docs/superpowers/specs/2026-08-19-make-help-default-design.md`, `docs/superpowers/plans/2026-08-19-make-help-default.md`

**Evidence:** Amp thread https://ampcode.com/threads/T-01a01958-3f4e-720f-abc4-0c1e46b43665 ; commits `8f46359` / push on `main`

---

### 2026-08-19 — macOS local `make test` platform gaps

**Tags:** darwin, fs, shell, backup

**Task:** First local `make test` on darwin/arm64 after clone failed (build + deploy setup + backup).

**Wrong / mistakes:**
- Assumed Linux-only syscalls (`unix.Renameat2` / `RENAME_EXCHANGE`) in shared Go code.
- Assumed bash 4+ empty-array expansion under `set -u` (`"${privilege[@]}"` unbound on macOS bash 3.2).
- Rejected *any* ancestor symlink on `fsroot.Open` — breaks every `t.TempDir()` because `/var` → `/private/var` (and `/tmp` → `/private/tmp`).
- Sealed backup workdir read-only *before* `os.Rename` — Darwin returns EPERM; Linux often still renames same-parent dirs.

**What worked:**
1. `exchangeRename` behind build tags: darwin `RenameatxNp`+`RENAME_SWAP`, linux `Renameat2`+`RENAME_EXCHANGE`.
2. Bash empty-safe: `${privilege[@]+"${privilege[@]}"}`.
3. `filepath.EvalSymlinks` then `os.OpenRoot(resolved)`; keep Lstat/no-follow *inside* the root.
4. `sealTree` only after successful rename to final path.

**Rule (next agent):** Before claiming Linux-first FS/shell code is “done”, run `make test` on darwin/arm64 (or at least `go test` packages that touch unix rename, rooted paths under `t.TempDir()`, bash setup scripts, and directory chmod+rename). Prefer platform files over `// +build` hacks scattered in logic.

**Codified into:**
- `internal/fsroot/exchange_{darwin,linux,other}.go` + `Open` EvalSymlinks
- `.agents/setup` empty-array-safe privilege prefix
- `internal/backup/backup.go` seal-after-rename
- Tests: `TestOpenThroughSymlinkedAbsoluteParentPinsRealDirectory`, existing atomic-write / backup seal tests
- Standing bullet in `AGENTS.md`

**Evidence:** Amp thread https://ampcode.com/threads/T-01a018ac-cda1-71df-81d1-c74a1f411445

---

### 2026-08-19 — Master merge: board tip can block pure FF

**Tags:** git, merge, master

**Task:** Merge a phase worker branch after master advanced with board-only commits.

**Wrong / mistakes:** After Phase 7 worker branched from `99efb67`, master committed board-only `a3d8428` on `main`. `git merge --ff-only` failed even though histories only diverged on docs.

**What worked:** Merge commit (or rebase worker) instead of forcing FF.

**Rule (next agent):** Prefer merge commit (or rebase worker) when main has board-only commits after the worker base. Do not force-push worker. Still require green `go test ./...` on the merge result before ship.

**Codified into:** master execution practice / HANDOFF

**Evidence:** `origin/main...origin/impl/v1-p7-hardening` was 1 board commit vs 6 phase commits; merged as `3dec9b6`.

---

### 2026-08-19 — consulting-grok-review via Grok thread, not Task/OpenAI

**Tags:** review, grok

**Task:** High-stakes Oracle-shaped review without ChatGPT / OpenAI Task.

**Wrong / mistakes:** `consulting-grok-review` skill said dispatch Task/subagent; Task tool routes through OpenAI and fails when ChatGPT is unsubscribed. Worker stuck or fell back to self-review.

**What worked:** New Amp thread in grok45 mode with filled `reviewer-prompt` contract; poll until `## Verdict`.

**Rule (next agent):** For consulting-grok-review on this project: spawn a **new Amp thread in grok45 mode** with `amp -m grok45 --no-archive-after-execute -x '<prompt>'` (`-ox` best-effort only — see 2026-08-20 Master Grok spawn), poll `amp threads markdown T-…` until `## Verdict`, then act on Critical/Important. Do **not** use the Task tool or built-in oracle. Do **not** treat self-review as the gate.

**Superseded dispatch detail:** mandatory `-ox` replaced by 2026-08-20 lesson (local `-x` + worktree isolation).

**Codified into:** `AGENTS.md` standing bullet; `.agents/skills/consulting-grok-review/`

**Evidence:** Phase 6 review thread https://ampcode.com/threads/T-01a01801-6408-7553-a0d4-58af60b7885d

**Supersedes:** 2026-08-19 — Worker high-stakes review = consulting-grok-review, not built-in oracle (Task dispatch path obsolete).

---

### 2026-08-19 — Worker high-stakes review = consulting-grok-review, not built-in oracle

**Tags:** review, grok

**Task:** Phase 6 worker high-stakes review gate.

**Wrong / mistakes:** Phase 6 worker repeatedly called the built-in `oracle` tool; it hit OpenAI usage limits (user no longer subscribes to ChatGPT), then fell back to self-review instead of `consulting-grok-review`.

**Rule (next agent):** **Superseded** by “consulting-grok-review via Grok thread, not Task/OpenAI” above. Still true: do **not** use built-in `oracle`; do **not** substitute silent self-review. Dispatch path is Grok thread, not Task.

**Evidence:** Phase 6 Backup worker thread; user corrections 2026-08-19.

**Synthesized / superseded:** 2026-08-19 → later Grok-thread lesson + `AGENTS.md`

---

### 2026-08-13 — Verify escaped literals at the byte level

**Tags:** review, go, fixtures

**Task:** Review of JSON-escaped tool output / Go raw strings.

**Wrong / mistakes:** A review treated JSON-escaped tool output as literal backslashes in Go raw strings, causing repeated fix rounds before the source bytes were checked.

**Rule (next agent):** When a finding depends on whether quotes are escaped, inspect the source directly with a fixed-string search or byte count before editing. Do not infer file contents from JSON-rendered diffs or tool results.

**Evidence:** Phase 3 Task 19 raw request fixture review.

---

### 2026-08-12 — Keep docs simple

**Tags:** docs, memory

**Task:** Where plans vs lessons vs standing rules live.

**Wrong / mistakes:** Built `docs/solutions/{process,planning}/…` plus many memory files. Too many directories; plan drafts sat in the wrong place (`docs/memory` instead of Superpowers).

**Rule (next agent):**
- Lessons → **one** stable file: **`docs/memory/lessons.md`** (prepend here).
- Superpowers specs/plans/locks/drafts → **`docs/superpowers/` only**.
- Standing agent rules → short bullets in **`AGENTS.md`**.
Do not invent parallel “solutions/process/planning” trees unless the user asks.

**Codified into:** `AGENTS.md`, this file

---

### 2026-08-12 — Multi-agent plans need one authority

**Tags:** plans, multi-agent

**Task:** Large multi-phase plan with parallel drafts.

**Wrong / mistakes:** Parallel phase drafts disagreed (table names, DB paths, Runner types, backup shape). Oracle rejected until fixed.

**Rule (next agent):** For big plans: write a **lock** beside the plan (`docs/superpowers/plans/…-lock.md`), draft phases under `…-drafts/` if needed, assemble **one** plan under `docs/superpowers/plans/`. Put a **Canonical contracts** section in the final plan that wins over stale snippets. Run high-stakes review until **Approved**. Implementers follow the assembled plan + canonical section.

**This plan:** `docs/superpowers/plans/2026-08-12-personal-agent-v1.md`  
**Lock / drafts:** `…-v1-lock.md`, `…-v1-drafts/`

---

### 2026-08-12 — Ship means push

**Tags:** ship, git

**Task:** Mark v1 plan work as shipped.

**Wrong / mistakes:** Committed the v1 plan (`b3fa9b0`) and treated that as shipped. Did not `git push`. User saw `main` ahead of `origin/main`.

**Rule (next agent):** Ship / done / archive ⇒ commit → verify → **`git push origin HEAD`** → only then archive the thread. Local commit alone is not shipped. If remote is ahead: fetch + rebase; ask the user on real conflicts.

**Check:** `git status -sb` must not show `ahead of origin` after a ship request.

**Codified into:** `AGENTS.md` standing bullet

---

## Template for the next lesson

Prepend under **Lessons (newest first)**. Keep short and factual. Update the **Index** row for each touched topic.

```markdown
### YYYY-MM-DD — <short title>

**Tags:** tag1, tag2

**Task:** <one line>

**Wrong / mistakes:** …
**What worked:** …   # optional if thin
**Rule (next agent):** …   # imperative; one shot for the next agent
**Codified into:** <paths>
**Evidence:** <thread URL or commit>
```
