# Superpowers (obra/superpowers)

This project has [Superpowers](https://github.com/obra/superpowers) installed under `.agents/skills/`.

## Bootstrap (session start)

At the start of every conversation — before any clarifying questions, exploration, or action — load and follow the `using-superpowers` skill.

If there is even a small chance a Superpowers skill applies, invoke it with the Skill tool before proceeding. Process skills take priority:

- Creative / build work → `brainstorming` first (hard gate: no implementation until design is approved)
- Bugs / unexpected behavior → `systematic-debugging` first
- Implementing a planned task → `test-driven-development` (and related plan/execution skills)
- About to claim done → `verification-before-completion`

Announce which skill you are using, then follow it exactly.

## Installed Superpowers skills

- `using-superpowers` — mandatory skill-check bootstrap
- `brainstorming` — design before code
- `writing-plans` / `executing-plans` / `subagent-driven-development`
- `test-driven-development` / `systematic-debugging` / `verification-before-completion`
- `using-git-worktrees` / `finishing-a-development-branch`
- `requesting-code-review` / `receiving-code-review`
- `dispatching-parallel-agents` / `writing-skills`

## Notes for Amp orbs

- Skills live in `.agents/skills/` (project-local; survives orb recreation when committed).
- To refresh from upstream: clone `https://github.com/obra/superpowers` and copy `skills/*` into `.agents/skills/`.
- Optional: set `SUPERPOWERS_DISABLE_TELEMETRY=1` to disable brainstorming visual-companion telemetry.
