# Hub start soft-fail, sessions list soft-fail, rail notes load

**Date:** 2026-08-20  
**Tags:** hub, session, rail, review, sdd, frontend, jsdom, grok45

**Task:** Master SDD execution of benchmark UI redesign (Tasks 1–12) on `impl/benchmark-ui-redesign`.

**Wrong / mistakes:**
1. **startSession** treated create+send as one success path: `sendMessage` failure left the new session off the list and closed → user retry created a second session (orphans). Whole-branch review Important.
2. **load** used `Promise.all(getProject, listSessions)` so a sessions API blip blanked the whole hub (no composer). Spec §5 wants list error + composer usable.
3. **Rail Files** opened project notes via `openPath` → `SessionFileTab` → `workspaceFile` only. Hub defaults `workspace_files: false` → 403. Task 7 review FAIL until `source`/`noteId` + `getProjectNote`.
4. **jsdom Modal:** suite “passed” while App tests threw uncaught `showModal is not a function` (errors ≠ fail without harden/polyfill).
5. **Grok workers** often returned `Error: terminated` mid-task; worktree had partial files — continue thread, don’t triple-spawn.
6. **`.name-row`** redefined as list-row button while hub header still used `name-row` as a flex helper → would restyle hub chrome.

**What worked:**
- Per-task consulting-grok-review + worktree workers + FF-merge; `check-review-gate` on progress file.
- Fix: surface session immediately after create; nested try on send; split load; rail meta + SessionFileTab dual load path; Modal try/typeof + App.test polyfill; continue interrupted Grok once; strip stray token classes.
- Tests as gates: hub soft-fail / no-orphan; SessionFileTab note path; full suite 229; portal vibe-pass structural.

**Rule (next agent):**
- Hub create path and list load must match AGENTS hub bullets (not atomic create∧send; sessions soft-fail).
- Rail file open must distinguish note vs workspace APIs.
- Dialog tests always polyfill; Modal never throws uncaught.
- Terminated Grok → continue once from worktree state.

**Codified into:**
- `AGENTS.md` standing rules (hub start, hub load, rail notes, Modal jsdom, Grok terminated, token collision)
- `.agents/skills/frontend-ui-craft/SKILL.md` personal-agent appendix
- Tests: `web/src/routes/ProjectHubPage.test.ts`, `SessionFileTab.test.ts`, `Modal.svelte` / `App.test.ts`

**Evidence:** coordinator thread T-01a02022-ba82-72da-aa1b-101905a3ada7; branch review T-01a02074-8bd1-7719-8c52-13432e9a871a; fix commit `0e8c63d`; ship branch `impl/benchmark-ui-redesign`
