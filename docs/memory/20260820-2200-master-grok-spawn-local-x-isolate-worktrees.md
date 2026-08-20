# Master Grok spawn: local `-x`, isolate worktrees

**Date:** 2026-08-20  
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
- `docs/memory/20260820-2200-master-grok-spawn-local-x-isolate-worktrees.md`
- `AGENTS.md` (standing bullets: grok spawn + master worktree + Node 22 web)
- `.agents/skills/consulting-grok-review/SKILL.md` (dispatch via grok45 thread, not Task; `-ox` fallback)
- `docs/superpowers/HANDOFF-ui-svelte-redesign-master.md` (spawn/isolation notes)

**Evidence:** Master thread https://ampcode.com/threads/T-01a01a38-e05a-74e6-9e81-dc97622bab29 ; board `docs/superpowers/STATUS-ui-svelte-redesign.md` (A–F done @ `468e571`); sample review https://ampcode.com/threads/T-01a01d20-827d-73cf-b4f0-764e69b778d5 ; plugin `.amp/plugins/grok-45-mode.ts`.

**Related:** supersedes the **dispatch command** half of “consulting-grok-review via Grok thread” (2026-08-19) regarding mandatory `-ox`; still forbids Task/OpenAI/oracle substitutes. Extends “Master merge: board tip can block pure FF” (2026-08-19) with cherry-pick-of-feature-only when worker board commits diverge.

---
