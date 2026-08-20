# Design: Frontend UI craft skill

**Status:** Implemented (skill landed 2026-08-20; user approved design then implementation)  
**Date:** 2026-08-20  
**Skill name:** `frontend-ui-craft`  
**Location:** `.agents/skills/frontend-ui-craft/` (project-only, `personal-agent`)  
**Shape:** Approach 2 — thin `SKILL.md` + `reference/craft.md`  
**Related:**  
- User pain: UI still feels non-human / AI-scaffolded  
- Product visual intent: `docs/superpowers/specs/2026-08-19-ui-svelte-redesign-design.md`  
- Process sources: `ai-agent-coding-standards` → Frontend & Web Automation; Hub standards  
- Patterns: Visual AI Multimodal Integration, Rich Feedback Loops, Spec-as-Test, Feature List as Immutable Contract, Disposable Scaffolding, Dev Tooling Assumptions Reset, Human-in-Loop  
- Skill authoring: Superpowers `writing-skills` / `building-skills` (TDD for skills)  
- Standing rules: polled SPA focus, prove served bytes, Docker dev vs prod, Node 22 for web tests (`AGENTS.md`)

---

## 1. Goals and non-goals

### Goals

1. **Force a human-feeling UI bar** on any frontend touch — not only “it compiles” or “markup looks fine.”
2. **Mandatory agent loop** for visible UI work: short screen spec → build cheaply → browser vibe-pass (when possible) → craft gate → spec-as-test → done.
3. **Anti-AI-slop craft** as red flags + light positive recipe (not a full design system).
4. **Project-local skill** tuned with a short personal-agent appendix; core habits stay general enough to promote later.
5. **Discoverable triggers** — load on any frontend/UI work; description is trigger-only (no workflow summary in YAML description).
6. **Test the skill** with RED baseline → GREEN with skill before treating it as done (writing-skills TDD).

### Non-goals (v1)

- Full design system, Figma library, or dark-mode spec
- Replacing `brainstorming`, `test-driven-development`, or product design specs
- Global Amp / user-skills install (project-only first)
- Mandating Playwright E2E for every one-line CSS tweak
- Redesigning the product inside the skill (skill enforces quality bar + process)
- CLI checklist scripts or hooks (optional later if agents still skip the gate)

---

## 2. Problem statement

Agents ship UI that is structurally correct but still reads as non-human:

- Scaffold chrome (bullet `•` nav, orphan `‹` controls)
- Default stack soup (indigo + slate + `rounded-xl` + Inter everywhere)
- Weak hierarchy (metric cards ≡ action cards; redundant eyebrows)
- Empty desert layout; content stuck top-left
- “Done” from reading Svelte/CSS without opening the real browser
- Occasional interaction bugs (e.g. polled chat stealing focus) when visual/process gates are skipped

Coding-standards Frontend domain already states the fix direction: **spec the screen → render + screenshot → vibe-user → spec-as-test**. This skill turns that into an invokable, testable Superpowers skill for this repo.

**Evidence (2026-08-20 live pass):** Chrome against `localhost:8080` after login — Home, Projects, Vaults, Settings, Review, Project hub, Sessions. Structure (shell, empty states, skeletons, focus-safe chat ancestry) is ahead of visual craft. Nav bullets, orphan collapse control, health pill chrome, and empty canvas were the clearest non-human cues.

---

## 3. Decisions locked with user

| Decision | Choice |
|----------|--------|
| Skill shape | **C** — thin mandatory loop + craft red flags + optional deeper reference |
| Location | **A** — project-only `.agents/skills/` |
| When to load | **A** — any frontend touch (escape: no rendered UI change) |
| Craft depth | **B** — red flags + light positive recipe |
| Browser gate | **B** — hard when app/browser reachable; else blocked, no fake pass |
| Generality | **C** — general core + short this-repo appendix |
| Success criteria | **C** — (1) opens real UI before done, (2) red flags fixed or explicitly waived |
| File layout | Thin `SKILL.md` + `reference/craft.md` |

---

## 4. Skill identity

### 4.1 Name and path

```
.agents/skills/frontend-ui-craft/
  SKILL.md
  reference/craft.md
```

- Directory name = frontmatter `name` = `frontend-ui-craft`
- Gerund/craft noun acceptable; name is searchable for frontend, UI, craft, vibe, screenshot, anti-slop

### 4.2 YAML description (triggers only)

Third person, starts with “Use when…”, **no workflow summary** (SDO rule — agents must not follow the description instead of the body):

```yaml
---
name: frontend-ui-craft
description: >
  Use when touching any frontend or UI work — screens, components, styles, layout,
  spacing, empty states, visual polish, “looks AI/generic”, or UI bugs — before
  implementing visible changes or before claiming UI done.
---
```

(Exact wording may be tightened at implementation for character limits; keep trigger-only.)

### 4.3 Relationship to other skills

| Skill | Relationship |
|-------|----------------|
| `brainstorming` | Still hard-gate for new product behavior/creative features before code |
| `test-driven-development` | Still required for behavior changes; this skill adds visual/vibe gates |
| `verification-before-completion` | Complements: UI “done” needs vibe-pass evidence, not only unit green |
| `systematic-debugging` | UI bugs: debug process first; this skill for craft + verify-in-browser |
| `a11y-debugging` / `chrome-devtools` | Tools used inside the vibe-pass; not replaced |
| Product design specs | Source of *what* to build; this skill is *how agents verify human-quality UI* |

---

## 5. Core principle

> **Code that compiles is not a UI. A human-feeling UI is specified, built cheaply, seen in a real browser, and fails a craft red-flag check before “done.”**

Supporting patterns (cite in skill body briefly, do not paste full pattern essays):

- Visual AI Multimodal Integration — screenshots/snapshots as ground truth  
- Rich Feedback Loops > Perfect Prompts — machine-readable failures after changes  
- Disposable Scaffolding — regenerate bad UI drafts from a sharper spec  
- Spec-as-Test / Feature List as Immutable Contract — observable acceptance steps  
- Dev Tooling Assumptions Reset — agent loop uses browser + HMR, not IDE-only eyeballing  

---

## 6. Mandatory loop (SKILL.md body)

For any change that affects what users see:

1. **Screen spec (short)** — goal, surfaces touched, states (empty / loading / error / populated), primary action, explicit out-of-scope. Freeze it (feature-list spirit; keep lightweight for small tasks).
2. **Build cheaply** — reuse tokens/components; if the draft is tangled or generic soup, regenerate from a clearer spec rather than endless patch nursing.
3. **Browser vibe-pass (hard when possible)**  
   - If app is reachable: open the real URL for the changed surface; a11y snapshot and/or screenshot; drive the path like a naive user (click primary actions, hit empty/error paths when relevant).  
   - If not reachable: state blocked; start the app or ask the user; **do not** claim visual done.  
   - Prefer Chrome DevTools MCP/CLI when configured (`.chrome.mcp.json`, debugging port).
4. **Craft gate** — run red flags (§7) and positive recipe (§8). Any unwaived red flag ⇒ not done.
5. **Spec-as-test** — add or update at least one automated check for the behavior/visual contract you changed (component/unit preferred; interaction tests when the risk is focus, keyboard, or multi-step). Personal-agent: preserve focus regression tests for polled session composers.
6. **Done means** — automated gates green **and** vibe-pass evidence (what was opened, what was checked) **and** no silent red flags.

### 6.1 Skip rule

Skip the browser vibe-pass **only** when the change cannot affect rendered UI (e.g. API client types only, pure test helpers with no component output). State the skip reason explicitly. When in doubt, do not skip.

### 6.2 Rationalizations to forbid (discipline)

| Excuse | Reality |
|--------|---------|
| “I read the Svelte/CSS; it looks fine” | Reading code is not seeing pixels or interaction. Open the browser. |
| “App isn’t running so I’ll skip” | Blocked ≠ passed. Start app or stop; don’t claim visual done. |
| “Just a small CSS tweak” | Small tweaks are how AI-slop accumulates. Still snapshot the surface. |
| “Craft is subjective” | Red flags are objective enough to block ship; waive only with user OK. |
| “Tests passed” | Unit green does not prove hierarchy, density, or nav chrome. |
| “I’ll polish in a follow-up” | Follow-up rarely comes; gate now or record an explicit waiver. |

---

## 7. Craft — red flags (STOP)

Do not ship (or claim done) while these remain, unless the user explicitly waives:

| Red flag | Why it feels non-human |
|----------|-------------------------|
| Bullet / `•` / emoji-as-icon navigation | Scaffold chrome, not product nav |
| Orphan controls (lone `‹`, unlabeled icon-only controls without name) | Unfinished shell |
| Default stack soup (indigo + slate + `rounded-xl` + Inter everywhere, no hierarchy) | Generic AI dashboard demo |
| Huge empty canvas; content stuck top-left with no grid/density intent | Layout not designed |
| Metric cards visually identical to action/nav cards | Weak hierarchy |
| Disabled nav items with no explanation | Broken mental model |
| Claiming done without opening the real UI (when reachable) | Guessing |
| Poll/timer full re-render or `innerHTML` replace destroying focused inputs | Hostile interaction (PA standing rule) |

---

## 8. Craft — positive recipe (light)

Not a full design system — default moves when building or polishing:

1. **Hierarchy** — one clear page H1; eyebrows only if they earn space; one obvious primary action.
2. **Density** — medium dashboard density; constrain content width/grid; avoid desert empty.
3. **Surfaces** — canvas vs panel vs sidebar from shared tokens; structure from border + spacing, not heavy shadows/gradients/glass.
4. **Actions** — primary / secondary / ghost (and destructive) are visually distinct on purpose.
5. **Nav** — consistent icons (or icon components), clear current-page state, collapse control aligned and accessible named.
6. **States** — empty, skeleton/loading, and inline error on lists/main surfaces you touch.
7. **Focus** — visible focus rings; do not trap or steal focus on poll.
8. **Copy** — plain product language; cut filler labels that do not help orientation.

Expanded examples and before/after notes live in `reference/craft.md` (load when polishing or when red flags fire).

---

## 9. Repo appendix (personal-agent)

Keep short in `SKILL.md`; detail can sit at end of `reference/craft.md` if needed.

- **Stack:** Svelte 5 + TypeScript + Vite + Tailwind; shared tokens/chrome in `web/src/app.css`
- **Routing:** hash router; shell context = global desk vs inside a vault
- **Sessions:** poll must patch DOM/state in place; never replace a focused composer control
- **Serve path:** before claiming a localhost fix, confirm which process owns the port and that served asset bytes include the edit
- **Docker:** `make docker-dev` for live API+web; production compose stays image-baked (no host source mounts)
- **Tests:** Node `>=22 <23` first on `PATH` for `make web-test` / Vitest
- **Browser:** Chrome DevTools via project `.chrome.mcp.json` (`--browser-url http://localhost:9222`) when available
- **Product visual intent:** clean dashboard (Vercel/shadcn-inspired), light-first, Inter — see UI redesign design spec; this skill does not redefine product IA

---

## 10. SKILL.md outline (implementation target)

Target: concise (prefer &lt;500 lines; aim much shorter). Suggested sections:

1. Frontmatter (`name`, `description`)
2. Overview + core principle (2–4 sentences)
3. When to use / when skip is allowed
4. Mandatory loop (numbered)
5. Red flags table
6. Positive recipe (bullets)
7. Browser vibe-pass rules (reachable vs blocked)
8. Rationalizations table
9. Personal-agent appendix
10. Link to `reference/craft.md` and related skills/specs
11. Done checklist (short)

`reference/craft.md`: expanded anti-slop examples, density/hierarchy notes, optional screenshot review prompts, PA-specific chrome notes.

---

## 11. Verification plan (skill TDD)

### RED — baseline without skill

Run a pressure scenario on a fresh agent context **without** the skill loaded, e.g.:

> Polish the Home/sidebar chrome so it feels less AI-generated. Fix nav affordances and empty layout. Claim done when ready.

Document verbatim whether the agent:

- Opens the real browser / takes snapshot
- Leaves bullet nav / orphan collapse
- Claims done from code only

### GREEN — with skill

Same scenario with `frontend-ui-craft` available and discoverable. Expect:

- Loads skill from trigger
- Opens UI (or explicit blocked)
- Hits red flags and addresses or waives
- Does not claim done on code-read alone

### Success criteria (user-locked)

1. Agent opens real UI before claiming done when reachable  
2. Obvious red flags are fixed or explicitly waived — not silently shipped  

Micro-test wording if needed: description must not summarize the full loop (SDO).

---

## 12. Implementation plan (after this spec is accepted)

1. Write failing baseline notes (RED) — can be a short doc under the skill dir or a lesson only if useful; prefer capturing rationalizations in the skill’s rationalization table  
2. Add `.agents/skills/frontend-ui-craft/SKILL.md` + `reference/craft.md`  
3. Optional one-line pointer in `AGENTS.md` standing rules (only if it earns hot-path space)  
4. GREEN re-test scenario  
5. Commit skill on a branch / main per normal git practice; push if shipping  

**Out of scope for the skill PR unless user asks:** actually redesigning Home/sidebar in product code (skill first; product polish can be a follow-up task using the skill).

---

## 13. Spec self-review

- [x] No unresolved placeholders (TODO/TBD/fill-in)  
- [x] No contradictions with locked user choices (C/A/A/B/B/C/C + approach 2)  
- [x] Scope clear: skill process + craft bar, not product redesign  
- [x] Success criteria match user definition  
- [x] Aligns with existing PA standing rules (focus, docker-dev, Node 22, serve path)  
- [x] Description rule documented (triggers only)  

---

## 14. Approval

User approved the design direction in-thread (2026-08-20) with “yes do that” after the full design presentation. This file is the durable spec for implementation.  
**Next:** user reviews this file → then implement skill via writing-skills RED→GREEN (not product UI redesign unless separately requested).
