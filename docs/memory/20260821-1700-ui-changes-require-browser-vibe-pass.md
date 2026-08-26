# UI changes require a real browser vibe-pass

**Date:** 2026-08-21  
**Tags:** frontend, craft, ux, browser, docker-dev, vibe-pass

**Task:** Session wrap after rail/session UI work; user correction that agents coded UI blind.

**Wrong / mistakes:**
- Claiming spacing/alignment fixes from CSS + unit tests alone (collapsed rail padding, header height, bubble gap).
- User had to say “open the browser” and “I hate blind UI coding.”
- Soft craft language (“when possible”) let agents skip live verification.
- Confusing docker-dev (Vite HMR) with prod dist rebuild path delayed seeing real UI.

**What worked:**
- Opening real `:8080` (docker-dev), authenticating, measuring `getBoundingClientRect` gaps, saving screenshots under `.amp/in/artifacts/`.
- Browser metrics exposed wrong padding target (inner class vs `.project-workspace__rail`) and insufficient gap (~4.5px vs needed).
- Multi-viewport probe (1440 → 900) showed fixed 240+300 layout starving chat — responsive breakpoints needed.

**Rule (next agent):**
1. **Every UI edit → open browser before “done.”** URL + evidence required.
2. Green Vitest / reading CSS ≠ UX verified.
3. Prefer `make docker-dev` for instant UI; do not require dist rebuild on that path.
4. If blocked starting the app, say blocked — never fake a vibe-pass.
5. For layout complaints: measure live DOM, do not iterate CSS blindly.
6. **This repo (2026-08-26):** vibe-pass via `.chrome.mcp.json` → CDP on `localhost:9222`. Do **not** desktop screen-record.

**Codified into:**
- `AGENTS.md` standing rule **UI code without browser = not done (HARD)**
- `.agents/skills/frontend-ui-craft/SKILL.md` HARD GATE + `.chrome.mcp.json` / `:9222` appendix
- Existing baseline/layout tests remain secondary to browser evidence

**Evidence:**
- Thread: https://ampcode.com/threads/T-01a0231c-f9d8-731e-aad9-27af0ac11035  
- User: “whenever you code the UI open the browser”  
- Artifacts: `.amp/in/artifacts/collapsed-rail-vibe/`, `collapsed-rail-check.png`  
