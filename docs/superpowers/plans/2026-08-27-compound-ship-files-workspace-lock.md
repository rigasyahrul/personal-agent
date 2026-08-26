# Lock: Compound as Ship + Files + Workspace

**Date:** 2026-08-27  
**Spec:** `docs/superpowers/specs/2026-08-27-compound-ship-files-workspace-design.md`  
**Assembled plan:** `docs/superpowers/plans/2026-08-27-compound-ship-files-workspace.md`

## Scope freeze

In:

- Compound button → `sendMessage` canned prompt (no `POST /compound`)
- Session file tools always on (ivory bold included); remove grant checkbox
- Rename session scratch off `workspace_*` (routes, types, grant, promote field, `source`)
- Rail **Files · Workspace · Config**; Files = session writes; Workspace = project notes
- In-chat file cards from `changed_path`
- ⋯ Save to Workspace? = copy + toast **Saved to Workspace.**

Out:

- Deleting compound proposal API / review card code
- `write_knowledge` / `AGENTS.md` / `memory/**`
- Workspace tree search redesign
- Toast queue / undo
- Standalone session route rail

## Authority

Spec wins on product names and copy. This lock wins on task split. Do not reintroduce `workspace_*` for session scratch.

## Draft files

Single assembled plan (no parallel drafts — Pi has no subagent tool this turn).
