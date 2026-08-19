# Personal Agent

Self-hosted, single-owner learning dashboard. Promote session notes into a durable project library, review with SM-2-lite, and back up the data directory. Development requires **Go 1.24+**; deployment can use Docker Compose with optional Caddy TLS.

## Non-goals (v1)

- Multi-user / multi-tenant auth
- General shell or host filesystem access for the agent
- FTS search, vault browser, or mobile clients
- Automatic public internet exposure without HTTPS

## Prerequisites

- Go **1.24+** (`go version`)
- Optional: Docker Compose for packaged deployment
- Optional: OpenAI-compatible API key for chat and bite generation

## Development

`make` or `make help` lists common targets.

```sh
make test
make lint
make run
```

Or:

```sh
go test ./...
BOOTSTRAP_TOKEN='replace-with-at-least-32-random-characters' \
  PA_SECURE_COOKIES=false \
  PA_MODELS=openai:test \
  go run ./cmd/personal-agent
```

The app listens on **`:8080`**. Runtime state defaults to `./data` (`PA_DATA_DIR`).

### Amp orb

`.amp/services.yaml` starts the app for orb development:

```sh
amp orb services ensure
```

It sets `PA_DATA_DIR=.amp/state/personal-agent`, `PA_ADDR=:8080`, `PA_SECURE_COOKIES=false`, a dev `BOOTSTRAP_TOKEN`, and health-checks `/health`.

### First-run bootstrap

1. Open `http://127.0.0.1:8080` (or the orb portal URL).
2. Enter the `BOOTSTRAP_TOKEN` and an owner password (12+ characters).
3. Log in. Bootstrap is **one-time**; a second attempt returns `409 owner_exists`.

## Configuration

| Variable | Required | Notes |
|----------|----------|--------|
| `BOOTSTRAP_TOKEN` | yes (first run) | ≥32 random characters; never commit real values |
| `PA_DATA_DIR` | no | default `./data` |
| `PA_ADDR` | no | default `:8080` |
| `PA_SECURE_COOKIES` | no | default secure; set `false` only for plain HTTP localhost |
| `PA_MODELS` | no | `provider:model_id,...` e.g. `openai:gpt-4o-mini` |
| `OPENAI_API_KEY` | no | needed for live chat/bites |
| `OPENAI_BASE_URL` | no | OpenAI-compatible base URL |
| `PA_BACKUP_S3_BUCKET` | no | enables S3 directory upload after local bundle |

**Warning:** Domain deployment requires **HTTPS** and `PA_SECURE_COOKIES=true`. Never expose plain HTTP with secure cookies disabled on an untrusted network.

## Docker Compose

```sh
cp deploy/.env.example deploy/.env
# Set BOOTSTRAP_TOKEN. Keep PA_SECURE_COOKIES=false for local plain HTTP.
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up --build
```

Compose persists state in the `pa-data` volume (`PA_DATA_DIR` at `/data`). Configure models with `OPENAI_API_KEY`, optional `OPENAI_BASE_URL`, and `PA_MODELS`.

### Live-reload development (API + web)

Production compose bakes the binary and `web/` into the image. For day-to-day coding against one Docker service with hot reload:

```sh
# one-time: cp deploy/.env.example deploy/.env  (set BOOTSTRAP_TOKEN)
make docker-dev
```

Equivalent raw compose (dev file is an **override** — it still needs the base file for ports/env/data):

```sh
docker compose --env-file deploy/.env \
  -f deploy/docker-compose.yml -f deploy/docker-compose.dev.yml up --build
```

- Mounts the full repo at `/src` and runs [`air`](https://github.com/air-verse/air) (`deploy/air.toml`) so Go API changes rebuild/restart automatically.
- Serves `web/` from the mounted tree — hard-refresh the browser after UI edits (no image rebuild).
- Reuses the same `pa-data` volume and `.env` as production compose.
- Do **not** run plain production `up` and the dev override on `:8080` at the same time.
- Stop: `make docker-dev-down` · logs: `make docker-dev-logs`

For a real domain with Caddy TLS:

```sh
# PA_DOMAIN=agent.example.com
# PA_SECURE_COOKIES=true
docker compose --env-file deploy/.env -f deploy/docker-compose.yml --profile domain up --build
```

## Data layout

```
$PA_DATA_DIR/
  db/personal-agent.sqlite
  files/…          # project source + session workspaces
  staging/…        # publication staging
  backups/local/…  # directory bundles
```

## Docs

- **Owner handbook (start here):** [`docs/manual/README.md`](docs/manual/README.md)
- Design: [`docs/superpowers/specs/2026-08-12-personal-agent-design.md`](docs/superpowers/specs/2026-08-12-personal-agent-design.md)
- Deploy: [`docs/ops/deploy.md`](docs/ops/deploy.md)
- Backup / restore: [`docs/ops/backup-restore.md`](docs/ops/backup-restore.md)

## License

See [`LICENSE`](LICENSE).
