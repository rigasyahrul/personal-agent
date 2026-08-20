# Frontend UI craft skill from live pass + standards

**Date:** 2026-08-20  
**Tags:** ui, skills, frontend, web, compounding

**Task:** Stop shipping AI-looking UI by codifying a project skill; learn from coding-standards Frontend domain + agentic patterns; ground red flags in a live Chrome pass.

**Wrong / mistakes:**
1. Treating “Svelte redesign + empty states + tests” as human-feeling UI — structure can be fine while chrome stays scaffold (`•` nav, orphan `‹`, desert layout, metric≡action cards).
2. Skill-shaped process alone without craft red flags → agents can run a loop and still ship generic indigo/slate soup.
3. Soft “prefer browser” gates → agents claim done from reading code; blocked app must not count as visual pass.
4. Chrome DevTools screenshot `filePath` into the repo can be denied (MCP workspace roots); a11y snapshot still works — don’t block the vibe-pass on PNG write failures.

**What worked:**
1. **Research first:** `ai-agent-coding-standards` Frontend domain + patterns (Visual AI, Rich feedback, Spec-as-test, Disposable scaffolding) before designing the skill.
2. **Brainstorm locks:** thin skill (C) = mandatory loop + red flags + light recipe; project-only; any FE touch; hard browser when reachable; general core + PA appendix.
3. **writing-skills shape:** trigger-only YAML description (no workflow summary); thin `SKILL.md` + `reference/craft.md`; RED baseline from live UI (`baseline-red.md`).
4. **Discovery:** AGENTS bootstrap line + installed-skills list + standing bullet so the skill is on the hot path without dumping craft into AGENTS.
5. **Live pass:** Chrome on `:8080` + DevTools CLI → concrete red flags that match product, not abstract taste essays.

**Rule (next agent):** Any visible UI work → load `frontend-ui-craft`. Open real UI when reachable (snapshot OK if screenshot path blocked). Do not claim done with unwaived red flags. Skill enforces quality bar; product IA stays in the UI redesign design spec. New sessions pick up the skill via AGENTS / `.agents/skills/`.

**Codified into:**
- `.agents/skills/frontend-ui-craft/SKILL.md`
- `.agents/skills/frontend-ui-craft/reference/craft.md`
- `.agents/skills/frontend-ui-craft/baseline-red.md`
- `AGENTS.md` (bootstrap priority, skills list, standing rule)
- `docs/superpowers/specs/2026-08-20-frontend-ui-craft-skill-design.md`

**Evidence:** Amp thread https://ampcode.com/threads/T-01a01e14-6165-71cb-bbca-e23ab2d2006a · commits `ce52948`, `9fde4ac`
