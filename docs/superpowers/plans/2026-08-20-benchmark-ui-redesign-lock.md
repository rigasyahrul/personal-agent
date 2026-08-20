# Lock: Benchmark UI redesign implementation plan

**Date:** 2026-08-20  
**Spec:** `docs/superpowers/specs/2026-08-20-benchmark-ui-redesign-design.md`  
**Status:** Planning — parallel drafts then assemble  

## Scope freeze

Implement Path C surface pack only:

| Phase | Tasks (range) | Draft file |
|-------|---------------|------------|
| Header + contracts + file map | (master) | assembled plan header |
| A Shell density | Task 1 | `…-drafts/A-shell.md` |
| B1 Modals | Tasks 2–3 | `…-drafts/B1-modals.md` |
| B2 Project hub | Tasks 4–6 | `…-drafts/B2-hub.md` |
| B3 Open session | Tasks 7–9 | `…-drafts/B3-session.md` |
| B4 Vault list | Task 10 | `…-drafts/B4-vault.md` |
| B5 Vibe-pass + skill | Tasks 11–12 | `…-drafts/B5-vibe-skill.md` |

## Authority

- Spec wins over session-focus design on rail/copy/composer/hub structure  
- TDD every task; Node `>=22 <23` for web tests  
- Rebuild `web/dist` + cache-bust before localhost vibe claims  
- Never remount focused composer on poll  
- Memory tab: chrome only, no fake save  
- consulting-grok-review per worker task before merge (repo standing rule)  

## Out of scope

Memory/soul API, global Home restyle, auto-collapse nav, dark mode, starred tabs  

## Assemble target

`docs/superpowers/plans/2026-08-20-benchmark-ui-redesign.md`
