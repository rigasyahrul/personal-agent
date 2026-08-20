# Lock: Benchmark UI redesign design

**Date:** 2026-08-20  
**Status:** Design approved in thread; drafting spec via parallel sections  
**Thread:** T-01a01feb-df87-76bf-9b8c-a42ee477e118  

## Approved decisions (do not re-litigate)

| ID | Decision |
|----|----------|
| PATH | C: Phase A shell density fix first, then full benchmark redesign |
| APPROACH | Surface pack (not whole-app Claude clone; not CSS-only) |
| REFS | `claude.png`, `claude-2.png`, `grok.png`, `grok-2.png`, `amp.png` (repo root; move to artifacts in plan) |
| A | Fix sidebar nav stretch (~36–40px rows); tighter pad |
| HUB | Project hub = 2 content panes only (main + right rail) + app left nav |
| HUB_MAIN | “How can I help you today?” + large composer on top; session rows **below** prompt (Claude); no left session list; no New session button |
| START | Send on hub composer creates session + first message |
| RAIL | Default **open**; header tabs **Memory** \| **Files** |
| MEMORY | Design chrome now (Grok-2 fields); persist API later |
| FILES | Amp-style directory tree; click → main file tab |
| VAULT | Name-first project list (`claude-2`) |
| SESSION | Amp main tabs (Agent + file tabs, Preview\|Source); Grok rail continuous; dense bottom composer on Agent tab |
| COPY | Small copy icon on each assistant response container |
| MODALS | Create project/vault via modal (no inline form soup) |
| NON_GOALS | Global home full restyle; starred tabs; auto-collapse nav; fake Memory save success |

## Draft sections (parallel)

1. `…-drafts/01-goals-shell.md` — goals, non-goals, Phase A shell
2. `…-drafts/02-project-hub.md` — Section 2 hub
3. `…-drafts/03-vault-session-modals.md` — vault list, open session, modals, phases, acceptance
4. Assemble → `2026-08-20-benchmark-ui-redesign-design.md`
