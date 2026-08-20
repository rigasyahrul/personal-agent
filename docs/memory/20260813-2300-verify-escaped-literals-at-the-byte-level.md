# Verify escaped literals at the byte level

**Date:** 2026-08-13  
**Tags:** review, go, fixtures


**Task:** Review of JSON-escaped tool output / Go raw strings.

**Wrong / mistakes:** A review treated JSON-escaped tool output as literal backslashes in Go raw strings, causing repeated fix rounds before the source bytes were checked.

**Rule (next agent):** When a finding depends on whether quotes are escaped, inspect the source directly with a fixed-string search or byte count before editing. Do not infer file contents from JSON-rendered diffs or tool results.

**Evidence:** Phase 3 Task 19 raw request fixture review.

---
