# Multi-agent plans need one authority

**Date:** 2026-08-12  
**Tags:** plans, multi-agent


**Task:** Large multi-phase plan with parallel drafts.

**Wrong / mistakes:** Parallel phase drafts disagreed (table names, DB paths, Runner types, backup shape). Oracle rejected until fixed.

**Rule (next agent):** For big plans: write a **lock** beside the plan (`docs/superpowers/plans/…-lock.md`), draft phases under `…-drafts/` if needed, assemble **one** plan under `docs/superpowers/plans/`. Put a **Canonical contracts** section in the final plan that wins over stale snippets. Run high-stakes review until **Approved**. Implementers follow the assembled plan + canonical section.

**This plan:** `docs/superpowers/plans/2026-08-12-personal-agent-v1.md`  
**Lock / drafts:** `…-v1-lock.md`, `…-v1-drafts/`

---
