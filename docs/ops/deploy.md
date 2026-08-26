# Deployment

## Localhost (Go binary)

Production UI is the Vite build under **`web/dist`**. Build it first (Node **22**), or use `make build` / `make run` which depend on `web-build`.

```sh
export PATH="/usr/bin:$PATH"   # ensure Node 22 if needed
make web-build                 # writes web/dist

export PA_DATA_DIR=./data
export PA_ADDR=:8080
export PA_SECURE_COOKIES=false
export BOOTSTRAP_TOKEN="$(openssl rand -hex 32)"
export PA_MODELS=openai:gpt-4o-mini
# optional:
# export OPENAI_API_KEY=...
# export OPENAI_BASE_URL=https://api.openai.com/v1

go run ./cmd/personal-agent
# or: make run / make build && ./personal-agent
```

Do **not** set `PA_UI_DEV_PROXY` for this path — that variable is only for the docker-dev / Vite HMR loop below.

Health: `curl -sf http://127.0.0.1:8080/health`

Bootstrap the owner **before** exposing the port beyond loopback. Complete setup at `/` once, then log in. After login you get the **Svelte** shell (sidebar: Home, Projects, Vaults, Review, Settings).

## Docker Compose (localhost)

Never commit `deploy/.env`. Create it from the template:

```sh
cp deploy/.env.example deploy/.env
# Edit BOOTSTRAP_TOKEN to a unique ≥32-character secret.
# Keep PA_SECURE_COOKIES=false for plain HTTP on 127.0.0.1 only.
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up --build
```

Compose publishes the app on `127.0.0.1:8080` and mounts the `pa-data` volume at `/data`.

### Live-reload development (Go + Vite HMR)

Production compose uses a baked image. Start the one-command API and UI loop:

```sh
make docker-dev
# open http://localhost:8080
# stop: make docker-dev-down
# first time / after Dockerfile.dev changes: make docker-dev-build
```

`make docker-dev` runs `up` only (no image rebuild). Use `make docker-dev-build` when the dev image itself must be rebuilt (`Dockerfile.dev`, base tools, etc.). Day-to-day Go/UI edits hot-reload via the mounted repo.

`Sending build context to Docker daemon` is the local repo uploaded to the Docker daemon (Colima on macOS), not a registry pull. Compose `build.context` is the repo root, so `.dockerignore` must live there — `deploy/.dockerignore` is ignored. Keep `.worktrees`, `web/node_modules`, `data`, and `deploy/.env` out of that tarball.

`deploy/docker-compose.dev.yml` mounts the repository at `/src` and starts `deploy/dev-entrypoint.sh`. The script runs Air for Go reloads and Vite for Svelte/TypeScript/CSS HMR. Go remains the only browser-facing server on port 8080: API and health requests terminate in Go, while non-API GETs are proxied because the override sets `PA_UI_DEV_PROXY=http://127.0.0.1:5173`.

Vite listens only inside the container on port 5173. Its client uses `host: localhost`, `clientPort: 8080`, and `path: /@vite-hmr`; Go carries `ws://localhost:8080/@vite-hmr`. Edits under `web/src/` update without rebuilding or recreating the container. Air continues to exclude `web/`, because Vite owns frontend watching.

Before claiming a localhost UI change works:

```sh
lsof -nP -iTCP:8080 -sTCP:LISTEN
curl -sf http://127.0.0.1:8080/src/App.svelte | grep 'Personal Agent'
```

Confirm `/@vite-hmr` is connected in DevTools → Network → WS. Production compose has no host source mounts: `deploy/Dockerfile` builds `web/dist` with Node 22 and copies only that output. Production Compose is image-baked and has **no live source mounts**; live repository mounts exist only in `docker-compose.dev.yml`. Never add `..:/src`, `../web:`, `PA_UI_DEV_PROXY`, or a published Vite port to `deploy/docker-compose.yml`.

### Persistent volume checks

- Confirm the volume is writable by the container user.
- After first start: `docker compose -f deploy/docker-compose.yml exec personal-agent ls -la /data` (service name may be `personal-agent` or as defined in the compose file).
- Ensure `db/`, `files/`, and `backups/` appear under `/data` after bootstrap.

## Domain + TLS (Caddy profile)

```sh
# In deploy/.env:
# PA_DOMAIN=agent.example.com
# PA_SECURE_COOKIES=true
# BOOTSTRAP_TOKEN=...

docker compose --env-file deploy/.env -f deploy/docker-compose.yml --profile domain up --build
```

Caddy terminates TLS for `PA_DOMAIN`. Set secure cookies **true** only when the browser reaches the app over HTTPS.

### Bootstrap-before-exposure

1. Start with compose bound to localhost (default).
2. Complete `/api/v1/setup/bootstrap` (UI setup) with `BOOTSTRAP_TOKEN`.
3. Verify login and `/health`.
4. Only then open the domain profile / firewall to the public internet.

## Health check

```sh
curl -sf http://127.0.0.1:8080/health
# or via domain:
curl -sf https://$PA_DOMAIN/health
```

## Upgrade sequence

1. **Backup** — Settings → Backup now, or wait for a successful scheduled run. Record `local_path` / object prefix and `manifest_hash`.
2. **Pull / build** — `git pull` (or ship a new image), then `docker compose ... build`.
3. **Stop writers** — `docker compose -f deploy/docker-compose.yml stop` (or stop the binary).
4. **Start** — migrations run on open; `docker compose ... up -d` (or restart the binary).
5. **Health** — `/health`, login, open a known note.

## Rollback boundaries

- Prefer restoring the last successful **directory bundle** into a fresh data directory (see `docs/ops/backup-restore.md`) rather than partially reverting files.
- Application binary rollback is independent of data: keep the previous image/binary until the new version passes health and a smoke review of notes/sessions.
- Do not restore `backups/` from a bundle as live state.

## Secret rotation

- **Bootstrap token:** only used until the owner exists; rotating after bootstrap has no effect on login. Generate a new token for a fresh install only.
- **Owner password:** change via a future settings flow is out of v1 scope; restore from backup if the password is lost and no session remains.
- **Session cookies:** logout invalidates the server-side `auth_sessions` row; clearing cookies forces re-login.
- **OpenAI / S3 credentials:** update env and restart; never bake secrets into the image. Do not commit `.env`.

## Compose reference commands

```sh
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up --build
docker compose --env-file deploy/.env -f deploy/docker-compose.yml --profile domain up --build
docker compose -f deploy/docker-compose.yml stop
docker compose -f deploy/docker-compose.yml start
docker compose -f deploy/docker-compose.yml logs -f
```
