# 02 — Install and first run

Get a process listening, bootstrap the owner once, and log in.

For domain TLS, upgrades, and volume checks, prefer the full procedure in [`docs/ops/deploy.md`](../ops/deploy.md). This chapter is the short path.

## Prerequisites

- **Go 1.24+** for local binary (`go version`)
- **Node 22** LTS if you build or test the UI yourself (`node -v` → `v22.x`) — required for `make web-build` / `make web-test`
- Optional: **Docker Compose** for packaged deploy and `make docker-dev`
- Optional: **OpenAI-compatible API key** if you want live chat / bites

## Option A — Local Go binary

Production-style static UI: build the Vite app, then run Go (serves **`web/dist`**).

```sh
# From the repo root
export PATH="/usr/bin:$PATH"   # ensure Node 22 if your default is older
make web-build                 # writes web/dist (needs Node 22 + npm)

export PA_DATA_DIR=./data
export PA_ADDR=:8080
export PA_SECURE_COOKIES=false
export BOOTSTRAP_TOKEN="$(openssl rand -hex 32)"
export PA_MODELS=openai:gpt-4o-mini
# optional live model:
# export OPENAI_API_KEY=...
# export OPENAI_BASE_URL=https://api.openai.com/v1

go run ./cmd/personal-agent
# or: make run / make build && ./personal-agent
```

`make build` also runs `web-build` first, then builds the Go binary.

Check:

```sh
curl -sf http://127.0.0.1:8080/health
# expect storage_writable true
```

Open `http://127.0.0.1:8080`.

**Keep the bootstrap token.** You need it only once, but you cannot recover it from the app if you lose it before bootstrap.

## Option B — Docker Compose (localhost)

```sh
cp deploy/.env.example deploy/.env
# Edit BOOTSTRAP_TOKEN (≥32 random characters).
# Keep PA_SECURE_COOKIES=false for plain HTTP on 127.0.0.1.
# Set OPENAI_API_KEY / PA_MODELS if you want chat.

docker compose --env-file deploy/.env -f deploy/docker-compose.yml up --build
```

App: `http://127.0.0.1:8080`. Data volume: `pa-data` → `/data` in the container.  
The production image builds `web/dist` with Node 22 and copies only that output (no host source mounts).

## Option C — Live UI development (`make docker-dev`)

For coding on API **and** Svelte with HMR:

```sh
cp deploy/.env.example deploy/.env   # one-time; set BOOTSTRAP_TOKEN
make docker-dev
# open http://localhost:8080
# stop: make docker-dev-down
```

Go remains the only browser-facing server on **:8080**. Non-API GETs are proxied to Vite (`PA_UI_DEV_PROXY`). Edit `web/src/` and use HMR. Details: [`docs/ops/deploy.md`](../ops/deploy.md).

## First-run UI

1. If no owner exists, you see **Set up your owner account** (no app chrome).
2. Enter **Bootstrap token** (same as `BOOTSTRAP_TOKEN`) and a **password (12+ characters)**.
3. Continue → **Sign in** with that password.
4. The **sidebar** appears (Home, Projects, Vaults, Review, Settings). The top bar should show storage **ready**.

Bootstrap is **one-time**. A second attempt fails with conflict (`owner_exists` / HTTP 409).

## Models before chat

Sessions need at least one configured model:

```sh
export PA_MODELS=openai:gpt-4o-mini
# restart the process after changing env
```

Without models, the Sessions page tells you to configure models (env-driven list). Live replies also need `OPENAI_API_KEY` (and optional `OPENAI_BASE_URL`).

## Domain + HTTPS (short)

1. Run on localhost first; complete bootstrap.
2. Set `PA_DOMAIN`, `PA_SECURE_COOKIES=true`, then:

```sh
docker compose --env-file deploy/.env -f deploy/docker-compose.yml --profile domain up --build
```

3. Only then expose the host. Details: [`docs/ops/deploy.md`](../ops/deploy.md).

**Never** put plain HTTP with `PA_SECURE_COOKIES=false` on an untrusted network.

## Amp orb (developers)

```sh
amp orb services ensure
```

Uses `.amp/services.yaml` (dev data under `.amp/state/personal-agent`, secure cookies off, health on `/health`). Use the **portal URL**, not raw localhost from outside the orb.

For UI HMR inside an orb, prefer `make docker-dev` when Docker is available, or build `web/dist` before `go run`.

## Checklist

- [ ] `/health` reports writable storage
- [ ] Bootstrap completed once
- [ ] Login works; sidebar visible
- [ ] `PA_MODELS` set if you need sessions
- [ ] Token and password stored offline
- [ ] (Local binary) `web/dist` built with Node 22

## Next

→ [03 — Daily use](03-daily-use.md)
