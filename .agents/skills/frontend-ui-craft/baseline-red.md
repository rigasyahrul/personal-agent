# RED baseline — frontend-ui-craft (pre-skill evidence)

**Date:** 2026-08-20  
**Method:** Live product inspection + prior agent-built UI (no `frontend-ui-craft` skill existed).  
**Scenario analogue:** "Polish shell/home so it feels less AI-generated; claim done when ready."

## Observed baseline behavior (failure)

Without this skill, agents and prior UI work produced:

1. **No forced browser craft gate** — structure/tests could pass while chrome stayed scaffolded.
2. **Scaffold nav** — `•` bullets as icons in `Sidebar.svelte`; collapse as raw `‹`/`›`.
3. **Default stack soup** — indigo primary, slate text, `rounded-xl` white cards, Inter, repeated page chrome.
4. **Desert layout** — Home/hub content top-heavy; large empty canvas; metric cards stretch wide.
5. **Weak hierarchy** — project hub metrics row and surface cards same visual weight.
6. **Disabled Sessions** in global nav with title-only hint, easy to miss.
7. **Completion risk** — "redesign shipped" from component completeness rather than vibe-pass against red flags.

## Rationalizations this skill must block

- "I read the Svelte; hierarchy is fine"
- "Empty states and skeletons exist, so UX is done"
- "Matches the redesign spec tokens, so it looks intentional"
- "Small leftover chrome can be a follow-up"

## GREEN expectation (with skill)

Agent loads `frontend-ui-craft`, opens real UI when reachable, lists red flags against shell/home, does not claim visual done while bullets/orphan collapse/unaddressed desert layout remain unless user waives.
