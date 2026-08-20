# Worker high-stakes review = consulting-grok-review, not built-in oracle

**Date:** 2026-08-19  
**Tags:** review, grok


**Task:** Phase 6 worker high-stakes review gate.

**Wrong / mistakes:** Phase 6 worker repeatedly called the built-in `oracle` tool; it hit OpenAI usage limits (user no longer subscribes to ChatGPT), then fell back to self-review instead of `consulting-grok-review`.

**Rule (next agent):** **Superseded** by “consulting-grok-review via Grok thread, not Task/OpenAI” above. Still true: do **not** use built-in `oracle`; do **not** substitute silent self-review. Dispatch path is Grok thread, not Task.

**Evidence:** Phase 6 Backup worker thread; user corrections 2026-08-19.

**Synthesized / superseded:** 2026-08-19 → later Grok-thread lesson + `AGENTS.md`

---
