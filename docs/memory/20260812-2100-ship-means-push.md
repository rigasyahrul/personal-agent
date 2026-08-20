# Ship means push

**Date:** 2026-08-12  
**Tags:** ship, git


**Task:** Mark v1 plan work as shipped.

**Wrong / mistakes:** Committed the v1 plan (`b3fa9b0`) and treated that as shipped. Did not `git push`. User saw `main` ahead of `origin/main`.

**Rule (next agent):** Ship / done / archive ⇒ commit → verify → **`git push origin HEAD`** → only then archive the thread. Local commit alone is not shipped. If remote is ahead: fetch + rebase; ask the user on real conflicts.

**Check:** `git status -sb` must not show `ahead of origin` after a ship request.

**Codified into:** `AGENTS.md` standing bullet

---
