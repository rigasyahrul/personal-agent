.DEFAULT_GOAL := help

.PHONY: help test lint fmt-check run build web-install web-build web-test docker-dev docker-dev-build docker-dev-down docker-dev-logs

COMPOSE_ENV := --env-file deploy/.env
COMPOSE_PROD := -f deploy/docker-compose.yml
COMPOSE_DEV := $(COMPOSE_PROD) -f deploy/docker-compose.dev.yml

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
	@$(call print-help-section,test lint fmt-check run build web-install web-build web-test docker-dev docker-dev-build docker-dev-down docker-dev-logs)

test: ## Run all Go tests
	go test ./...

lint: fmt-check ## gofmt check + go vet
	go vet ./...

fmt-check: ## Fail if any .go file needs gofmt
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './.git/*'))" || \
	  (echo "Go files need gofmt"; gofmt -l $$(find . -name '*.go' -not -path './.git/*'); exit 1)

run: ## Run the app (go run)
	go run ./cmd/personal-agent

build: web-build ## Build web assets and ./cmd/personal-agent
	go build ./cmd/personal-agent

web-install: ## Install locked web dependencies
	npm --prefix web ci

web-build: web-install ## Build the production Svelte UI
	npm --prefix web run build

web-test: web-install ## Run web unit tests
	npm --prefix web test

docker-dev: ## Live-reload Docker (API+web); needs deploy/.env
	@test -f deploy/.env || (echo "Create deploy/.env first: cp deploy/.env.example deploy/.env"; exit 1)
	docker compose $(COMPOSE_ENV) $(COMPOSE_DEV) up

docker-dev-build: ## Rebuild dev image, then live-reload up; needs deploy/.env
	@test -f deploy/.env || (echo "Create deploy/.env first: cp deploy/.env.example deploy/.env"; exit 1)
	docker compose $(COMPOSE_ENV) $(COMPOSE_DEV) up --build

docker-dev-down: ## Stop live-reload Docker stack
	docker compose $(COMPOSE_ENV) $(COMPOSE_DEV) down

docker-dev-logs: ## Tail live-reload Docker logs
	docker compose $(COMPOSE_ENV) $(COMPOSE_DEV) logs -f personal-agent
