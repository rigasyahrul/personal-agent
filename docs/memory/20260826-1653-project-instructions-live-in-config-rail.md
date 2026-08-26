# Project SOUL / SYSTEM / AGENTS live in the Config rail

**Date:** 2026-08-26  
**Tags:** rail, hub, frontend, craft, vibe-pass, knowledge

**Task:** User saw SOUL / SYSTEM / AGENTS as a new card at the bottom of `#/projects/:id` while Config already existed.

**Wrong / mistakes:**
- Knowledge slice mounted `InstructionEditor` on the hub canvas (`panel` card) because the spec said “project and global settings/desk.”
- Config still had the 2026-08-21 placeholder: unsaved “Instructions (system)” + “Not saved yet.”
- Reading the old rail spec would put that fake field back.
- Vibe-pass attempt used desktop `screencapture`; user: use `.chrome.mcp.json`.

**What worked:**
- Config rail = `InstructionEditor` `variant="rail"` (tabs + GET/PUT + Save).
- Hub main = composer + sessions only; editor stays mounted while a session is open.
- Chrome DevTools on `:9222` (`.chrome.mcp.json`): `inMain: false`, `inRail: true`.

**Rule (next agent):**
- Project instruction files belong **only** in the Config rail. Settings keeps **global** files.
- Do not add a hub canvas instruction card. Do not restore the fake unsaved textarea. Do not put memory/lessons preview in Config.
- UI check: `.chrome.mcp.json` → `localhost:9222`. No desktop screen-record.

**Codified into:**
- `AGENTS.md` standing rule **Project rail chrome**
- `.agents/skills/frontend-ui-craft/SKILL.md` hub + browser appendix
- `ProjectHubPage.test.ts` / `ProjectRail.test.ts` / `InstructionEditor.test.ts`
- Spec supersession: `docs/superpowers/specs/2026-08-21-project-rail-icon-chrome-design.md` §5; `docs/superpowers/specs/2026-08-22-obsidian-memory-knowledge-design.md` §12

**Evidence:** Pi session `01a03d6b-2366-7cac-a294-841f459da701`; URL `http://localhost:8080/#/projects/eea7b476-e0b7-44d8-89e2-d3fafab87172`
