# Personal Agent

Self-hosted, single-owner learning dashboard. Development requires Go 1.24+; deployment can use Docker Compose.

## Development

Run the test suite and start the server:

```sh
go test ./...
BOOTSTRAP_TOKEN='replace-with-at-least-32-random-characters' go run ./cmd/personal-agent
```

The app listens on port 8080. Runtime state defaults to `./data`; set `PA_DATA_DIR` to use another writable location. Bootstrap the owner once, then log in.

## Docker Compose

Copy the environment template, replace the bootstrap token, and start the app:

```sh
cp deploy/.env.example deploy/.env
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up --build
```

Compose persists application state in the `pa-data` volume mounted at `/data`. It passes credentials through environment variables and does not bake them into the image.

Model configuration uses `OPENAI_API_KEY`, optional `OPENAI_BASE_URL`, and `PA_MODELS`, a comma-separated list of `provider:model_id` pairs. Keep API keys and the bootstrap token out of source control.

For local plain HTTP, set `PA_SECURE_COOKIES=false`; this is unsafe on an untrusted network. For a real domain, set `PA_DOMAIN`, leave secure cookies enabled, and start the Caddy TLS reverse proxy:

```sh
docker compose --env-file deploy/.env -f deploy/docker-compose.yml --profile domain up --build
```
