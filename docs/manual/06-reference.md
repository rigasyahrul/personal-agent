# 06 — Reference

Quick tables and links. Prefer ops docs for long procedures.

## Environment variables

| Variable | Required | Default / notes |
|----------|----------|-----------------|
| `BOOTSTRAP_TOKEN` | First run | ≥32 random characters; one-time setup |
| `PA_DATA_DIR` | no | `./data` |
| `PA_ADDR` | no | `:8080` |
| `PA_SECURE_COOKIES` | no | Secure cookies on unless `false` (localhost HTTP only) |
| `PA_MODELS` | for chat UI | `provider:model_id,...` e.g. `openai:gpt-4o-mini` |
| `OPENAI_API_KEY` | for live chat/bites | |
| `OPENAI_BASE_URL` | no | OpenAI-compatible base URL |
| `PA_BACKUP_S3_BUCKET` / related S3 env | no | Enables remote backup upload; see backup-restore doc |
| `PA_DOMAIN` | domain profile | Used by Caddy compose profile |

Never commit real secrets. Use `deploy/.env` from `deploy/.env.example` for Compose.

## Useful Makefile targets

```sh
make test    # go test ./...
make lint    # go vet (needs Go 1.24+)
make run     # go run ./cmd/personal-agent
make build   # go build ./cmd/personal-agent
```

## Data layout (reminder)

```text
$PA_DATA_DIR/
  db/personal-agent.sqlite
  files/
  staging/
  backups/local/{run_id}/
```

## Limits you will hit in the UI

| Limit | Value |
|-------|--------|
| Markdown body | 1 MiB UTF-8 |
| Relative path | 512 bytes, depth 16, component 255 |
| Password (bootstrap) | min 12 characters |
| Promote/direct target | regular `.md` only; no overwrite |

## Compose one-liners

```sh
# Localhost
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up --build

# Domain TLS profile
docker compose --env-file deploy/.env -f deploy/docker-compose.yml --profile domain up --build
```

## Doc map

| Need | Go here |
|------|---------|
| This handbook TOC | [`README.md`](README.md) |
| Deploy / upgrade / TLS | [`../ops/deploy.md`](../ops/deploy.md) |
| Backup restore drill | [`../ops/backup-restore.md`](../ops/backup-restore.md) |
| Product design | [`../superpowers/specs/2026-08-12-personal-agent-design.md`](../superpowers/specs/2026-08-12-personal-agent-design.md) |
| Repo quick start | [`../../README.md`](../../README.md) |

## API

v1 is a browser app over `/api/v1` (plus `/health`). There is no separate public API handbook in this edition. Developers: routes live under `internal/httpapi/`; acceptance coverage in `internal/acceptance/`.
