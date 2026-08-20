# macOS local `make test` platform gaps

**Date:** 2026-08-19  
**Tags:** darwin, fs, shell, backup


**Task:** First local `make test` on darwin/arm64 after clone failed (build + deploy setup + backup).

**Wrong / mistakes:**
- Assumed Linux-only syscalls (`unix.Renameat2` / `RENAME_EXCHANGE`) in shared Go code.
- Assumed bash 4+ empty-array expansion under `set -u` (`"${privilege[@]}"` unbound on macOS bash 3.2).
- Rejected *any* ancestor symlink on `fsroot.Open` — breaks every `t.TempDir()` because `/var` → `/private/var` (and `/tmp` → `/private/tmp`).
- Sealed backup workdir read-only *before* `os.Rename` — Darwin returns EPERM; Linux often still renames same-parent dirs.

**What worked:**
1. `exchangeRename` behind build tags: darwin `RenameatxNp`+`RENAME_SWAP`, linux `Renameat2`+`RENAME_EXCHANGE`.
2. Bash empty-safe: `${privilege[@]+"${privilege[@]}"}`.
3. `filepath.EvalSymlinks` then `os.OpenRoot(resolved)`; keep Lstat/no-follow *inside* the root.
4. `sealTree` only after successful rename to final path.

**Rule (next agent):** Before claiming Linux-first FS/shell code is “done”, run `make test` on darwin/arm64 (or at least `go test` packages that touch unix rename, rooted paths under `t.TempDir()`, bash setup scripts, and directory chmod+rename). Prefer platform files over `// +build` hacks scattered in logic.

**Codified into:**
- `internal/fsroot/exchange_{darwin,linux,other}.go` + `Open` EvalSymlinks
- `.agents/setup` empty-array-safe privilege prefix
- `internal/backup/backup.go` seal-after-rename
- Tests: `TestOpenThroughSymlinkedAbsoluteParentPinsRealDirectory`, existing atomic-write / backup seal tests
- Standing bullet in `AGENTS.md`

**Evidence:** Amp thread https://ampcode.com/threads/T-01a018ac-cda1-71df-81d1-c74a1f411445

---
