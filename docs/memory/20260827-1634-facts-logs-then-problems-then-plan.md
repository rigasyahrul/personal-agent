# Facts and logs first, then problems, then a fix plan

**Date:** 2026-08-27  
**Tags:** agents, craft, browser, vibe-pass, ux

**Task:** After the `@` picker argument, user: every “do this / do that” must fact-find (logs/console), analyze, summarize problems, then a fix plan. Put it in AGENTS.md.

**Wrong / mistakes:**
- Jumping to explanations (`lsof` → SSH, “merged so it’s there”) instead of opening the named URL and reading console/network.
- Sounding sure without having used the UI or checked grants/logs.

**What worked:**
- CDP on the given hash URL; type `@`; screenshot; then API: orbit frost is `session_files: true`, picker still gates on `workspace_files`.

**Rule (next agent):**
User says do/check/fix → (1) facts (browser if UI + console/network/logs) (2) analyze (3) tell the user the problems (4) fix plan. Do not code or narrate as if you already used the app.

**Codified into:** `AGENTS.md` standing rule **Do / check / fix → facts, then problems, then plan.**

**Evidence:** User correction this thread.
