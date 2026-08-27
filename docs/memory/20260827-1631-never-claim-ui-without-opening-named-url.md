# Never claim you used the UI without opening the named URL

**Date:** 2026-08-27  
**Tags:** craft, browser, vibe-pass, ux, frontend

**Task:** User asked to verify `@` file suggestions at `http://localhost:8080/#/projects/eea7b476-e0b7-44d8-89e2-d3fafab87172/sessions`. Agent argued from `lsof`/curl instead of driving the page.

**Wrong / mistakes:**
- Answered as if the running app had been used (“8080 is SSH, not this app”) without opening the URL.
- After curl showed Vite + `mention-files.ts`, still spoke as if that proved the composer UX.
- User had to say: open this link, type `@` yourself, never do this again.

**What worked:**
- Chrome CDP `:9222` → that exact hash URL.
- Typed `@` in hub “How can I help you today?” — no list.
- Opened **orbit frost**, typed `@` in Reply… — `aria-expanded=false`, no `.session-composer__mentions`. Screenshots under `.amp/in/artifacts/20260827-*-typed-at.png`.

**Rule (next agent):**
1. User names a URL or “open the browser” → open **that** URL in Chrome CDP, do the action, screenshot.
2. Never write as if you used the composer unless you did.
3. `lsof`, curl of HTML/Svelte, “merged on main”, “JS is served” ≠ vibe-pass.

**Codified into:** `AGENTS.md` standing rule **Named URL**; `.agents/skills/frontend-ui-craft/SKILL.md` HARD GATE step 5 + red flag.

**Evidence:** User correction this thread; artifacts `20260827-hub-typed-at.png`, `20260827-orbit-frost-typed-at.png`.
