# Default `make` Help Menu Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bare `make` prints a grouped, comment-driven command menu and exits 0 instead of running tests.

**Architecture:** Set `.DEFAULT_GOAL := help`. Annotate each public target with `## description`. The `help` recipe prints hard-coded section headers (Common, Development) and extracts matching target descriptions from `$(MAKEFILE_LIST)` via portable awk.

**Tech Stack:** GNU Make, POSIX/BSD-friendly awk, existing Makefile + README.

**Spec:** `docs/superpowers/specs/2026-08-19-make-help-default-design.md`

## Global Constraints

- Bare `make` and `make help` must exit 0 and must not run `go test` or any other real target
- Descriptions live on the target line as `target: ## description` (comment-driven)
- Sections are hard-coded: **Common** (`help`), **Development** (`test`, `lint`, `fmt-check`, `run`, `build`)
- Help extractor must work on macOS (Darwin/BSD awk) and Linux — no GNU-only sed
- Do not change behavior of named targets `test`, `lint`, `fmt-check`, `run`, `build`
- No new Go tests; verification is manual shell checks
- No Docker/deploy targets, colors, or external help scripts

## File map

| File | Role |
|------|------|
| `Makefile` | Default goal, `help` target, `##` annotations, `.PHONY` |
| `README.md` | One-line note that `make` / `make help` lists targets |

---

### Task 1: Makefile help menu + README blurb

**Files:**
- Modify: `Makefile` (full file rewrite of current 17-line Makefile)
- Modify: `README.md` (Development section only)

**Interfaces:**
- Consumes: existing target recipes unchanged (`go test ./...`, `go vet`, gofmt check, `go run`, `go build`)
- Produces: `.DEFAULT_GOAL := help`; public targets `help`, `test`, `lint`, `fmt-check`, `run`, `build` each with `##` description; sectioned help output

- [ ] **Step 1: Replace `Makefile` with annotated targets and sectioned help**

Write the entire `Makefile` as follows (tabs before recipe lines are required):

```makefile
.DEFAULT_GOAL := help

.PHONY: help test lint fmt-check run build

help: ## Show command help
	@echo ""
	@echo " Choose a command run in personal-agent:"
	@echo ""
	@echo " --- Common ---"
	@awk -v targets=" help " -f- $(MAKEFILE_LIST) <<'EOF'
/^[a-zA-Z0-9_.-]+:.*##/ {
  split($$1, a, ":")
  name = a[1]
  if (index(targets, " " name " ") == 0) next
  desc = $$0
  sub(/^[^#]*##[[:space:]]*/, "", desc)
  printf " %-30s %s\n", name, desc
}
EOF
	@echo " --- Development ---"
	@awk -v targets=" test lint fmt-check run build " -f- $(MAKEFILE_LIST) <<'EOF'
/^[a-zA-Z0-9_.-]+:.*##/ {
  split($$1, a, ":")
  name = a[1]
  if (index(targets, " " name " ") == 0) next
  desc = $$0
  sub(/^[^#]*##[[:space:]]*/, "", desc)
  printf " %-30s %s\n", name, desc
}
EOF

test: ## Run all Go tests
	go test ./...

lint: fmt-check ## gofmt check + go vet
	go vet ./...

fmt-check: ## Fail if any .go file needs gofmt
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './.git/*'))" || \
	  (echo "Go files need gofmt"; gofmt -l $$(find . -name '*.go' -not -path './.git/*'); exit 1)

run: ## Run the app (go run)
	go run ./cmd/personal-agent

build: ## Build ./cmd/personal-agent
	go build ./cmd/personal-agent
```

**Important implementation notes:**

1. **Heredoc + make:** Inline `<<'EOF'` inside a Make recipe is fragile (Make may not pass it the way a shell script does when recipes are separate lines). Prefer the **single portable awk one-liner pattern** below if the heredoc form misbehaves when you run `make`. Use this equivalent recipe instead (recommended default for the implementer):

```makefile
.DEFAULT_GOAL := help

.PHONY: help test lint fmt-check run build

# Print targets whose names appear in TARGETS (space-padded list), with ## descriptions.
# Usage: $(call print-help-section, target1 target2 ...)
define print-help-section
awk -v targets=" $(1) " ' \
  /^[a-zA-Z0-9_.-]+:.*##/ { \
    split($$1, a, ":"); \
    name = a[1]; \
    if (index(targets, " " name " ") == 0) next; \
    desc = $$0; \
    sub(/^[^#]*##[ \t]*/, "", desc); \
    printf " %-30s %s\n", name, desc; \
  }' $(MAKEFILE_LIST)
endef

help: ## Show command help
	@echo ""
	@echo " Choose a command run in personal-agent:"
	@echo ""
	@echo " --- Common ---"
	@$(call print-help-section,help)
	@echo " --- Development ---"
	@$(call print-help-section,test lint fmt-check run build)

test: ## Run all Go tests
	go test ./...

lint: fmt-check ## gofmt check + go vet
	go vet ./...

fmt-check: ## Fail if any .go file needs gofmt
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './.git/*'))" || \
	  (echo "Go files need gofmt"; gofmt -l $$(find . -name '*.go' -not -path './.git/*'); exit 1)

run: ## Run the app (go run)
	go run ./cmd/personal-agent

build: ## Build ./cmd/personal-agent
	go build ./cmd/personal-agent
```

2. **Target order in output:** awk prints in Makefile file order. Keep Development targets in the Makefile in the order: `test`, `lint`, `fmt-check`, `run`, `build` so the menu matches the spec example. `help` is listed under Common and also defined first among annotated rules (aside from the `define`).

3. **`lint` line:** Keep prerequisite `fmt-check` and put `##` after prerequisites: `lint: fmt-check ## gofmt check + go vet`. The awk pattern `.*##` still matches.

4. **Tabs:** Recipe lines must start with a real tab character, not spaces.

- [ ] **Step 2: Update README Development section**

In `README.md`, change the Development block from:

```markdown
## Development

```sh
make test
make lint
make run
```
```

to:

```markdown
## Development

`make` or `make help` lists common targets.

```sh
make test
make lint
make run
```
```

Do not remove the longer `go test` / `go run` env example that follows.

- [ ] **Step 3: Verify bare `make` prints help and does not run tests**

Run:

```bash
make
```

Expected (exit 0): output contains:

- `Choose a command run in personal-agent:`
- `--- Common ---`
- `help` with description `Show command help`
- `--- Development ---`
- `test`, `lint`, `fmt-check`, `run`, `build` each with a non-empty description

Also confirm the command returns quickly (no full `go test ./...` run). Timing should be near-instant; if you see many `ok github.com/...` lines, default goal is wrong.

Run:

```bash
make help
```

Expected: same menu as bare `make` (compare with `diff <(make) <(make help)` — empty diff).

- [ ] **Step 4: Verify named targets still work**

Run:

```bash
make test
```

Expected: `go test ./...` runs; packages pass (or same result as before this change).

Run:

```bash
make build
```

Expected: builds `personal-agent` binary (or succeeds with `go build ./cmd/personal-agent`); exit 0.

Optional smoke:

```bash
make -n run
make -n lint
```

Expected: prints the would-be recipes without error (`go run ./cmd/personal-agent`, and lint’s fmt-check + vet chain).

- [ ] **Step 5: Commit**

```bash
git add Makefile README.md
git commit -m "$(cat <<'EOF'
feat: default make target prints grouped help menu

Bare make no longer runs tests. Comment-driven ## descriptions
and hard-coded Common/Development sections match the approved UX.
EOF
)"
```

---

## Self-review (plan vs spec)

| Spec requirement | Task |
|------------------|------|
| Bare `make` → help, exit 0 | Task 1 Steps 1, 3 |
| `make help` same | Task 1 Step 3 |
| Comment-driven `##` | Task 1 Step 1 |
| Hard-coded Common + Development sections | Task 1 Step 1 |
| Existing targets unchanged | Task 1 Steps 1, 4 |
| macOS-safe awk | `print-help-section` define; no GNU sed |
| README note | Task 1 Step 2 |
| Manual verification | Task 1 Steps 3–4 |
| No Docker/colors/external scripts | Not in plan |

No placeholders remain. Single task is appropriate: one coherent Makefile+README deliverable.
