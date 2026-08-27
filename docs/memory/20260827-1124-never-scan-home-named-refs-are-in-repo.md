# Never scan $HOME; named mock refs live in the repo

**Date:** 2026-08-27  
**Tags:** agents, files, home, refs, craft

**Task:** User attached `1-element-thought.png` / `2-element-thought-hover.png` / `3-element-thought-clicked.png` as product mocks for a new Thought UI.

**Wrong / mistakes:** Agent ran `find` across `/Users/mac-103` (home) looking for those filenames and for a “thought” feature. Thought is a **new** element. Home scan is prohibited.

**What worked:** The three PNGs were already in the repo root. `Read` them. Do not search home or invent a code hunt for a feature that does not exist yet.

**Rule (next agent):** Never search `$HOME` / `~` / `/Users/<name>`. Stay in the repo or named worktree. Named `@file` refs → Read in cwd/repo. New-element mocks are design refs, not search targets.

**Codified into:** `AGENTS.md` standing rule **Never scan $HOME.**

**Evidence:** user correction this session (orbit frost Thought UI).
