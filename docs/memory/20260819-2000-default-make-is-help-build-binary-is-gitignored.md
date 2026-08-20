# Default `make` is help; build binary is gitignored

**Date:** 2026-08-19  
**Tags:** make, skills, gitignore


**Task:** Bare `make` should print a grouped command menu (RSSM-style), not run tests.

**Wrong / mistakes:**
- First Makefile target was `test`, so bare `make` ran the full suite — surprising for onboarding.
- `make build` writes `./personal-agent` at repo root; it was untracked and not in `.gitignore`.
- `compounding-engineering` (and `synthesize-memory`) live under `.agents/skills/` and are listed in `AGENTS.md`, but Amp’s **Skill tool registry may omit them** — `skill` call returns “not found”. Must Read the SKILL.md and follow it anyway.

**What worked:**
1. `.DEFAULT_GOAL := help` + each public target annotated `target: ## description`.
2. Hard-coded section headers (Common / Development) with portable awk via `$(call print-help-section,…)` — no heredoc-in-recipe, no GNU-only sed; works on Darwin.
3. New targets: add `##` line, list name in the right section’s `print-help-section` call, add to `.PHONY`.
4. When Skill tool misses a project skill: `Read .agents/skills/<name>/SKILL.md` and execute.

**Rule (next agent):**
- Bare `make` must stay help-only. Never put a heavy target first without `.DEFAULT_GOAL`.
- After `make build`, do not commit the binary; root binary name is gitignored.
- Project skills under `.agents/skills/` named in AGENTS.md are mandatory even if the Skill tool says not found — load from disk.

**Codified into:**
- `Makefile` (help default + `print-help-section`)
- `.gitignore` (`/personal-agent`)
- `AGENTS.md` standing bullets
- Spec/plan: `docs/superpowers/specs/2026-08-19-make-help-default-design.md`, `docs/superpowers/plans/2026-08-19-make-help-default.md`

**Evidence:** Amp thread https://ampcode.com/threads/T-01a01958-3f4e-720f-abc4-0c1e46b43665 ; commits `8f46359` / push on `main`

---
