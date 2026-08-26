# Docker --build context is host→daemon, not a registry pull

**Date:** 2026-08-26  
**Tags:** docker, dockerignore, colima, compose

**Task:** Explain why `make docker-dev-build` looked like it “pulled from the server”; add a root `.dockerignore`.

**Wrong / mistakes:** Treating `Sending build context to Docker daemon 132.8MB` as an image pull. Putting ignore rules under `deploy/` (Compose `context: ..` never reads that file). Rebuilding the image for ordinary Go/UI edits.

**What worked:** Repo-root `.dockerignore` excluding `.worktrees`, `web/node_modules`, `data`, `.git`, `deploy/.env`. Keep `cmd/`, `internal/` (embeds), `web/` source, `deploy/dev-entrypoint.sh`. Daemon here is Colima (Linux VM), so the tarball is a real host→VM copy.

**Rule (next agent):** Do not add `image:` / registry to `personal-agent` to “build locally.” It already builds from `deploy/Dockerfile.dev`. Shrink context; use `make docker-dev` unless the Dockerfile changed.

**Codified into:** `AGENTS.md` standing rule, `.dockerignore`, `docs/ops/deploy.md`, `deploy/deploy_test.go` (`TestRootDockerignoreShrinksComposeBuildContext`)
**Evidence:** local `make docker-dev-build` + `docker context show` → `colima`
