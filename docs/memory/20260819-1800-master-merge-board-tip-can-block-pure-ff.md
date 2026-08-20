# Master merge: board tip can block pure FF

**Date:** 2026-08-19  
**Tags:** git, merge, master


**Task:** Merge a phase worker branch after master advanced with board-only commits.

**Wrong / mistakes:** After Phase 7 worker branched from `99efb67`, master committed board-only `a3d8428` on `main`. `git merge --ff-only` failed even though histories only diverged on docs.

**What worked:** Merge commit (or rebase worker) instead of forcing FF.

**Rule (next agent):** Prefer merge commit (or rebase worker) when main has board-only commits after the worker base. Do not force-push worker. Still require green `go test ./...` on the merge result before ship.

**Codified into:** master execution practice / HANDOFF

**Evidence:** `origin/main...origin/impl/v1-p7-hardening` was 1 board commit vs 6 phase commits; merged as `3dec9b6`.

---
