# Composer `@` file list is not a hidden grant

**Date:** 2026-08-27  
**Tags:** hub, session, craft, frontend

**Task:** `@` file suggestions in hub + orbit frost. User loved the ungated list.

**Wrong / mistakes:**
- v1 gated the picker on `workspaceEnabled` → `workspace_files === true`. No UI for that grant. Hub creates `workspace_files: false`. orbit frost is `session_files: true` only. `@` was a silent no-op.
- Hub start composer was scoped out; user types `@` there too.
- Tree API also 403’d on the old key.

**What worked:**
- Ungate the overlay. `session_files` counts as files-on. GET tree listing not behind a user grant. Tree fail → `listProjectNotes`. Same list on hub start.
- Prove on the named URL in Chrome: `@` shows `standing-rule.md`.

**Rule (next agent):** Do not hide composer file mentions behind a grant the user cannot see. Hub composer is in scope. Tests: SessionChat grant-off / `session_files`; hub `@stand` insert.

**Codified into:** `AGENTS.md` **Composer `@` is not a hidden grant**; `SessionChat.test.ts`; `ProjectHubPage.test.ts`; `workspaceEnabled` + `workspaceRoot` listing.

**Evidence:** User “really loves this”; CDP `20260827-fix2-orbit-at.png`.
