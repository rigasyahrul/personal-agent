# HANDOFF: Session Focus Layout

**Date:** 2026-08-20  
**Prior master thread:** https://ampcode.com/threads/T-01a01eac-2694-7622-9396-d70e0086d08b  
**Resume master thread:** https://ampcode.com/threads/T-01a01f00-30b2-748a-8a68-c625574d5a35  
**Status:** Design + plan restored on fresh orb; implementation continuing. **Do not merge or push** unless user says so.  
**Reason for handoff:** User raising/restarting specification server for orbs — start a **new** thread from this file.

---

## Goal (product)

Re-layout **sessions focus only** (Approach 1):

1. **Amp-style main tabs:** always-on **Agent** + file tabs opened from tree  
2. **Files right bar:** searchable tree only (no embedded preview); toggle from session header; default **closed**  
3. When open: default **70% main / 30% files**, resizable, clamp main **50–85%**, prefs in localStorage  
4. File tab body: **Preview** (default) / **Source**; promote CTA on file tab when promotable  
5. **Assistant:** no bubble, no “Assistant” label — bare **Markdown + Mermaid**  
6. **User:** end-aligned bubble  
7. **Vault + project session lists:** card rows (Claude Projects–inspired); desk list only (no focus left history rail)  
8. App shell / left nav **unchanged**; collapse stays user-controlled (no auto-collapse)  
9. Composer poll invariant: never remount focused composer  

**Out of scope:** global shell redesign, notes page redesign, new APIs, edit-in-tab, Amp Changes/Portal/Terminal modes, dark mode, focus-mode session history rail.

---

## Authority docs (read these first)

| Doc | Path |
|-----|------|
| Spec (approved) | `docs/superpowers/specs/2026-08-20-session-focus-layout-design.md` |
| Plan lock | `docs/superpowers/plans/2026-08-20-session-focus-layout-lock.md` |
| Implementation plan | `docs/superpowers/plans/2026-08-20-session-focus-layout.md` |
| UI shell baseline | `docs/superpowers/specs/2026-08-19-ui-svelte-redesign-design.md` |
| Standing rules | `AGENTS.md` |

Visual refs (if still on disk):  
`.amp/in/artifacts/ref-amp-tabs.png`, `ref-ss.png`, `ref-grok-rightbar.png` (or `/tmp/ref-*.png`).

---

## Git state (as of resume on fresh orb)

Prior orb lost unpushed commits. Restored on this orb from prior-thread artifacts, then re-committed.

**Feature branch:** `impl/session-focus-layout` under `.worktrees/session-focus-layout`  
**User constraint:** do **not** merge or push until explicitly asked.

---

## Implementation progress

| Task | Status |
|------|--------|
| 1 Session prefs | Restored from prior Task 1 worker (was `88e6dc9`) |
| 2 CSS tokens | Not started |
| 3 Markdown render helper | Not started |
| 4 MarkdownView + Mermaid | Not started |
| 5 SessionChat focus shell | Not started |
| 6 SessionFilesBar | Not started |
| 7 File tabs + promote move | Not started |
| 8 Split drag + narrow drawer | Not started |
| 9 Session card rows | Not started |
| 10 Vault + project list pages | Not started |
| 11 Harden + vibe-pass | Not started |

---

## Worker spawn rules (critical)

1. **Grok workers:** `amp -m grok45 --no-archive-after-execute -x '…'`  
2. **Do NOT use `-ox` / `--orb-execute` with `grok45`** — fails with `Agent mode is invalid`.  
3. **Default mode + `-ox` works** but is **not Grok** — user asked for Grok + worktrees.  
4. **One git worktree per worker**; merge product commits into `impl/session-focus-layout` only after verify; master board/merge from clean tree.  
5. **No Task/OpenAI subagents** as Grok substitute.  
6. **Node 22:** `export PATH="$HOME/.local/node-v22/bin:$PATH"` (or install Node 22 under `~/.local/node-v22`).  
7. Long parallel Grok streams may **timeout** — prefer **one worker at a time**, shorter prompts, commit often.  
8. High-stakes review: new grok45 thread + `consulting-grok-review` skill contract.  
9. Skills: `using-superpowers` → `subagent-driven-development` / `test-driven-development` / `frontend-ui-craft` / `verification-before-completion` as appropriate.

---

## Resume recipe

```bash
cd /home/user/workspace/repo
export PATH="$HOME/.local/node-v22/bin:$PATH"

# Feature worktree
cd .worktrees/session-focus-layout   # branch impl/session-focus-layout

# Continue plan tasks 2→11 with grok45 -x + new worktree per task
# Plan: docs/superpowers/plans/2026-08-20-session-focus-layout.md
# Spec: docs/superpowers/specs/2026-08-20-session-focus-layout-design.md
```

After UI changes: `npm --prefix web test`, keep `SessionChat.focus.test.ts` green; before claiming UI done rebuild `web/dist` + browser vibe-pass with cache-bust.

---

## Success when

Plan acceptance checklist in `docs/superpowers/plans/2026-08-20-session-focus-layout.md` is green; user then decides merge/push.
