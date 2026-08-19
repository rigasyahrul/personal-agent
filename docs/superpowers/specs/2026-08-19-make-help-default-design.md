# Design: Default `make` → help menu

**Date:** 2026-08-19  
**Status:** Approved  
**Scope:** Makefile UX only (no app/runtime changes)

## Goal

When a user runs bare `make` (no target), print a grouped command menu and exit successfully. Do **not** run tests or any other real task as the default.

Inspired by RSSM-style Makefiles: header, section banners, target + description columns.

## Non-goals

- Docker Compose / deploy make targets
- Colors, interactive selection, or paging
- External help scripts
- Changing behavior of existing named targets (`test`, `lint`, `fmt-check`, `run`, `build`)

## Decisions

| Decision | Choice |
|----------|--------|
| Bare `make` | Always help, exit 0 |
| Description source | Comment-driven: `target: ## description` on (or immediately above) each public target |
| Sections | Hard-coded section headers in the `help` recipe (not auto-discovered) |
| Implementation style | GNU Make + small awk/grep in `help` (Approach 1) |
| Docs | Optional one-line README note under Development |

## Behavior

1. `make` → same as `make help`
2. `make help` → print menu, exit 0
3. `make <target>` → existing behavior unchanged

### Example output

```text
 Choose a command run in personal-agent:

 --- Common ---
 help                           Show command help
 --- Development ---
 test                           Run all Go tests
 lint                           gofmt check + go vet
 fmt-check                      Fail if any .go file needs gofmt
 run                            Run the app (go run)
 build                          Build ./cmd/personal-agent
```

Column layout: target name left-aligned in a fixed width (~30), then description. Exact spacing may match common `awk` help patterns; readability over pixel-perfect RSSM clone.

## Makefile structure

### Default goal

```make
.DEFAULT_GOAL := help
```

### Phony targets

Include `help` in `.PHONY` alongside existing targets.

### Annotations

Every user-facing target gets a `##` description, e.g.:

```make
test: ## Run all Go tests
	go test ./...
```

`help` itself:

```make
help: ## Show command help
	@...
```

Internal/helper targets (if any later) omit `##` and do not appear in help.

### Sections (hard-coded)

Help recipe prints sections in this order:

1. **Common** — `help`
2. **Development** — `test`, `lint`, `fmt-check`, `run`, `build`

Implementation options (either is fine; prefer the simpler one that stays accurate):

- **A (preferred):** `help` recipe contains explicit section headers and, for each section, an awk/sed pass that extracts `##` lines for a fixed list of target names; or one pass over `$(MAKEFILE_LIST)` that only prints lines whose target is in that section’s list.
- **B:** Section banner comments in the Makefile (`##@ Common`) that the help script recognizes, with section order still defined by appearance / explicit list — only if A becomes awkward.

Do not require maintainers to remember a second file; descriptions live next to targets.

### Help recipe requirements

- Read from the current Makefile (`$(MAKEFILE_LIST)` / this file)
- Exit 0 on success
- No network, no Go build
- Works on macOS (BSD awk/sed) and Linux — avoid GNU-only sed extensions
- Suppress recipe echo (`@`)

## README

Under **Development**, note that `make` / `make help` lists targets. Keep existing `make test` / `make lint` / `make run` examples.

## Testing / verification

Manual only (Makefile-only change):

1. `make` prints the menu and does not run `go test`
2. `make help` matches `make`
3. `make test` still runs tests
4. `make build` / `make run` still work as before

No new Go tests required.

## Risks

| Risk | Mitigation |
|------|------------|
| Help drifts from real targets | Descriptions on the same line as the target rule; sections list target names explicitly so missing annotations are obvious |
| macOS awk differences | Keep the extractor trivial; verify on Darwin during implementation |
| Someone relies on bare `make` running tests | Document the change in README; this is intentional UX |

## Implementation outline

1. Set `.DEFAULT_GOAL := help`
2. Add `help` target with sectioned printer
3. Annotate all public targets with `## ...`
4. Update `.PHONY`
5. Touch README Development blurb
6. Verify with `make` and `make test`
