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
