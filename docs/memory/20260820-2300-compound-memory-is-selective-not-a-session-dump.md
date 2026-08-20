# Compound memory is selective, not a session dump

**Date:** 2026-08-20  
**Tags:** memory, compounding, docs layout

**Task:** Stop compounding-engineering from bloating memory; restore pattern-native layout (index + timestamped entry files).

**Wrong / mistakes:**
1. Skill said record a memory entry **ALWAYS**, so agents logged full session narratives even when AGENTS/skill/test already held the rule.
2. Intermediate fix stuffed all bodies into one `lessons.md` — scannable index died under weight; not the original pattern (`YYYYMMDD-HHmm-slug.md`).

**What worked:**
1. **Codify first** (AGENTS, skills, hooks, tests).
2. **Selective evidence only** when rationale / multi-system / synthesis fodder is needed.
3. **Layout:** thin `docs/memory/lessons.md` (topic map + list) + detail in `docs/memory/YYYYMMDD-HHmm-<slug>.md`.

**Rule (next agent):** Compounding → durable artifact first. If evidence is needed: write one timestamped entry file, prepend one index bullet, refresh topic rows. Never paste full bodies into the index. “Codified into” must list durable paths.

**Codified into:**
- `.agents/skills/compounding-engineering/SKILL.md`
- `.agents/skills/synthesize-memory/SKILL.md`
- `AGENTS.md` (Compounding engineering section + Compound ≠ diary bullet)
- `docs/memory/lessons.md` (index only)
- Migrated historical sections into `docs/memory/YYYYMMDD-HHmm-*.md`

**Evidence:** Amp thread https://ampcode.com/threads/T-01a01d60-4a91-76a5-b6b3-5b8d6ac17de3

**Related:** Supersedes “always prepend into one lessons.md body” half of 20260819-2200-lessons-file-is-stable-lessons-md; keeps selective frequency. Aligns with pattern diary filename.
