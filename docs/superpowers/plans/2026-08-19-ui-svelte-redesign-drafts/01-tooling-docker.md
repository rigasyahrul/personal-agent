# UI Svelte Redesign: Tooling and Docker Draft

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish the Svelte 5 production build and a one-command Docker development loop with Vite HMR at `http://localhost:8080`.

**Architecture:** `web/` is the npm/Vite app and emits `web/dist/`; Go serves that directory normally. In `make docker-dev`, a Node 22 image runs Air and Vite, and Go proxies its non-API GET fallback to Vite. Vite's HMR client connects through Go on port 8080 at `/@vite-hmr`.

**Tech Stack:** Svelte 5, TypeScript, Vite, Tailwind CSS, Vitest, npm, Node 22 LTS, Go 1.24, Air, Docker Compose.

## Global Constraints

- Keep hash routing; this phase adds no History API fallback.
- Commit `web/package-lock.json`; use npm only.
- Production assets are exactly `web/dist/**`; Go defaults to `http.Dir("web/dist")`.
- Development keeps the existing `..:/src` override and `air.toml` continues to exclude `web/`.
- Production compose remains image-baked and contains no host source mount.
- Preserve `.DEFAULT_GOAL := help`; every public Make target has `##`, `.PHONY`, and a `print-help-section` entry.

This phase establishes only the build/runtime seam. Product screens replace the smoke component in later plan sections.

---

### Task 1: Scaffold Svelte 5, TypeScript, Vite, and Tailwind

**Files:**
- Move: `web/index.html`, `web/js/**`, `web/css/**` → `web-legacy/**`
- Create: `web/package.json`, `web/package-lock.json`, `web/index.html`, `web/tsconfig.json`, `web/vite.config.ts`
- Create: `web/src/vite-env.d.ts`, `web/src/main.ts`, `web/src/App.svelte`, `web/src/app.css`
- Test: `web/dist/index.html`

**Interfaces:**
- Consumes: Node `>=22 <23`; old vanilla roots `web/index.html`, `web/js/**`, `web/css/**`.
- Produces: npm scripts `dev`, `build`, `test`; source root `web/src/**`; output `web/dist/**`; HMR path `/@vite-hmr` via browser port `8080`.

- [ ] **Step 1: Run the failing scaffold check**

```bash
test -f web/package.json && test -f web/src/App.svelte && test -f web/vite.config.ts
```

Expected: FAIL because the Vite files do not exist.

- [ ] **Step 2: Move the vanilla UI aside**

```bash
mkdir web-legacy
git mv web/index.html web/js web/css web-legacy/
mkdir -p web/src
```

Expected: PASS; old assets are preserved under `web-legacy/` while later phases port their behavior.

- [ ] **Step 3: Create the scaffold**

`web/package.json`:

```json
{
  "name": "personal-agent-web",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "engines": { "node": ">=22 <23" },
  "scripts": {
    "dev": "vite --host 0.0.0.0 --port 5173",
    "build": "vite build",
    "test": "vitest run"
  },
  "devDependencies": {
    "@sveltejs/vite-plugin-svelte": "^6.1.0",
    "@tailwindcss/vite": "^4.1.0",
    "@testing-library/jest-dom": "^6.8.0",
    "@testing-library/svelte": "^5.2.8",
    "@types/node": "^22.0.0",
    "jsdom": "^26.1.0",
    "svelte": "^5.38.0",
    "tailwindcss": "^4.1.0",
    "typescript": "^5.9.0",
    "vite": "^7.1.0",
    "vitest": "^3.2.0"
  }
}
```

`web/index.html`:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Personal Agent</title>
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
```

`web/tsconfig.json`:

```json
{
  "compilerOptions": {
    "allowJs": true,
    "checkJs": true,
    "esModuleInterop": true,
    "forceConsistentCasingInFileNames": true,
    "isolatedModules": true,
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "skipLibCheck": true,
    "strict": true,
    "target": "ES2022",
    "types": ["vitest/globals", "@testing-library/jest-dom"]
  },
  "include": ["src/**/*.d.ts", "src/**/*.ts", "src/**/*.svelte", "vite.config.ts"]
}
```

`web/vite.config.ts`:

```ts
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [tailwindcss(), svelte()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    strictPort: true,
    hmr: { protocol: 'ws', host: 'localhost', clientPort: 8080, path: '/@vite-hmr' },
  },
  test: { environment: 'jsdom', setupFiles: ['./src/test/setup.ts'] },
})
```

`web/src/vite-env.d.ts`:

```ts
/// <reference types="vite/client" />
```

`web/src/main.ts`:

```ts
import { mount } from 'svelte'
import App from './App.svelte'
import './app.css'

mount(App, { target: document.getElementById('app')! })
```

`web/src/App.svelte`:

```svelte
<script lang="ts">
  const title: string = 'Personal Agent'
</script>

<main class="grid min-h-screen place-items-center bg-slate-50 text-slate-950">
  <h1 class="text-2xl font-semibold">{title}</h1>
</main>
```

`web/src/app.css`:

```css
@import "tailwindcss";

:root { font-family: Inter, system-ui, sans-serif; }
body { margin: 0; }
```

- [ ] **Step 4: Install and verify**

```bash
npm --prefix web install
npm --prefix web run build
test -f web/package-lock.json
test -f web/dist/index.html
```

Expected: PASS; Vite reports a successful build. Commit npm's generated lockfile without hand-editing it.

- [ ] **Step 5: Commit**

```bash
git add web web-legacy
git commit -m "build(web): scaffold Svelte Vite app"
```

---

### Task 2: Add the Web Unit-Test Contract and Smoke Test

**Files:**
- Create: `web/src/test/setup.ts`, `web/src/App.test.ts`
- Test: `web/src/App.test.ts`

**Interfaces:**
- Consumes: `App` from `web/src/App.svelte`; npm script `test`.
- Produces: test setup `web/src/test/setup.ts`; smoke contract that heading `Personal Agent` renders; command `npm --prefix web test`.

- [ ] **Step 1: Write a failing smoke test**

`web/src/test/setup.ts`:

```ts
import '@testing-library/jest-dom/vitest'
```

`web/src/App.test.ts`:

```ts
import { render, screen } from '@testing-library/svelte'
import { describe, expect, it } from 'vitest'
import App from './App.svelte'

describe('App', () => {
  it('renders the application banner', () => {
    render(App)
    expect(screen.getByRole('banner', { name: 'Personal Agent' })).toBeInTheDocument()
  })
})
```

- [ ] **Step 2: Verify failure**

```bash
npm --prefix web test -- --run src/App.test.ts
```

Expected: FAIL because `<main>` has no `banner` role or accessible name.

- [ ] **Step 3: Replace the test with the intended smoke contract**

```ts
import { render, screen } from '@testing-library/svelte'
import { describe, expect, it } from 'vitest'
import App from './App.svelte'

describe('App', () => {
  it('renders the application heading', () => {
    render(App)
    expect(screen.getByRole('heading', { name: 'Personal Agent' })).toBeInTheDocument()
  })
})
```

- [ ] **Step 4: Verify test and build**

```bash
npm --prefix web test
npm --prefix web run build
```

Expected: PASS; Vitest reports one passing test and Vite emits `web/dist/index.html`.

- [ ] **Step 5: Commit**

```bash
git add web/src/test web/src/App.test.ts
git commit -m "test(web): add Svelte smoke test"
```

---

### Task 3: Add Makefile Web Lifecycle Targets

**Files:**
- Modify/Test: `Makefile`

**Interfaces:**
- Consumes: npm scripts `build`, `test`; `web/package-lock.json`.
- Produces: `web-install`, `web-build`, `web-test`; release path `build: web-build`; Development help entries.

- [ ] **Step 1: Verify the target is absent**

```bash
make web-build
```

Expected: FAIL with `No rule to make target 'web-build'`.

- [ ] **Step 2: Add exact Makefile entries**

Use this `.PHONY` line:

```make
.PHONY: help test lint fmt-check run build web-install web-build web-test docker-dev docker-dev-down docker-dev-logs
```

Use this Development help call:

```make
	@$(call print-help-section,test lint fmt-check run build web-install web-build web-test docker-dev docker-dev-down docker-dev-logs)
```

Replace `build` and add:

```make
build: web-build ## Build web assets and ./cmd/personal-agent
	go build ./cmd/personal-agent

web-install: ## Install locked web dependencies
	npm --prefix web ci

web-build: web-install ## Build the production Svelte UI
	npm --prefix web run build

web-test: web-install ## Run web unit tests
	npm --prefix web test
```

- [ ] **Step 3: Verify lifecycle and help**

```bash
make web-test
make build
make help | grep -E 'web-install|web-build|web-test'
test -f web/dist/index.html
test -x personal-agent
```

Expected: PASS; all targets appear in help, and both artifacts build.

- [ ] **Step 4: Verify the default remains help**

```bash
make -s | grep 'Choose a command run in personal-agent'
```

Expected: PASS; bare `make` does not run build or tests.

- [ ] **Step 5: Commit**

```bash
git add Makefile
git commit -m "build: add web lifecycle targets"
```

---

### Task 4: Serve `web/dist` by Default

**Files:**
- Modify: `internal/app/app.go`, `internal/app/app_test.go`
- Modify: `internal/httpapi/static_test.go`, `internal/httpapi/web_test.go`
- Test: `internal/app/app_test.go`, `internal/httpapi/static_test.go`

**Interfaces:**
- Consumes: `web/dist/index.html`; injectable `Dependencies.Static http.FileSystem`.
- Produces: default `http.Dir("web/dist")`; legacy contract tests temporarily rooted at `web-legacy/js/**`.

- [ ] **Step 1: Write the failing application-path test**

Add to `internal/app/app_test.go` (and add `os`, `strings` imports):

```go
func TestDefaultStaticDirectoryIsViteDist(t *testing.T) {
	body, err := os.ReadFile("app.go")
	if err != nil { t.Fatal(err) }
	if !strings.Contains(string(body), `http.Dir("web/dist")`) {
		t.Fatal("default static directory must be web/dist")
	}
}
```

Replace `internal/httpapi/static_test.go`:

```go
package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestStaticShell(t *testing.T) {
	if _, err := os.Stat("../../web/dist/index.html"); err != nil {
		t.Fatal("run npm --prefix web run build first:", err)
	}
	h := http.FileServer(http.Dir("../../web/dist"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Personal Agent") || !strings.Contains(w.Body.String(), `type="module"`) {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}
```

- [ ] **Step 2: Verify failure**

```bash
npm --prefix web run build
go test ./internal/app ./internal/httpapi -run 'TestDefaultStaticDirectoryIsViteDist|TestStaticShell'
```

Expected: FAIL because `app.go` still contains `http.Dir("web")`.

- [ ] **Step 3: Change the default and preserve legacy checks**

In `internal/app/app.go` use:

```go
static = http.Dir("web/dist")
```

In `internal/httpapi/web_test.go`, replace the test map with:

```go
tests := map[string][]string{
	"../../web-legacy/js/pages/sessions.js":           {"Save to source", "target_relative_path", "review_mode", "operation_id"},
	"../../web-legacy/js/pages/review.js":             {"project:", "scope=", "caught_up", "row_version", "duration_ms"},
	"../../web-legacy/js/components/status-badges.js": {"Promoting…", "Promote failed — Retry", "Note saved; cards pending…", "Cards failed — Retry cards", "Ready"},
}
```

- [ ] **Step 4: Verify focused and full tests**

```bash
gofmt -w internal/app/app_test.go internal/httpapi/static_test.go internal/httpapi/web_test.go
go test ./internal/app ./internal/httpapi
go test ./...
```

Expected: PASS; old behavior remains pinned while Go serves only built assets by default.

- [ ] **Step 5: Commit**

```bash
git add internal/app/app.go internal/app/app_test.go internal/httpapi/static_test.go internal/httpapi/web_test.go
git commit -m "feat(http): serve Vite distribution by default"
```

---

### Task 5: Build Web Assets in the Production Image

**Files:**
- Modify: `deploy/Dockerfile`, `deploy/deploy_test.go`
- Test: `deploy/deploy_test.go`

**Interfaces:**
- Consumes: `node:22-alpine`, lockfile, npm `build`, static root `/app/web/dist`.
- Produces: stage `web-build`; exact runtime copy `/src/web/dist` → `/app/web/dist`; no frontend source in runtime.

- [ ] **Step 1: Add failing assertions**

Set the `Dockerfile` entry in `TestDeploymentFiles` to:

```go
"Dockerfile": {"node:22-alpine AS web-build", "npm ci", "npm run build", "golang:1.24", "CMD", "/app/web/dist"},
```

Add:

```go
func TestProductionImageCopiesOnlyBuiltWebAssets(t *testing.T) {
	dockerfile := readFile(t, "Dockerfile")
	if !strings.Contains(dockerfile, "COPY --from=web-build --chown=app:app /src/web/dist /app/web/dist") {
		t.Fatal("production image must copy Vite dist")
	}
	if strings.Contains(dockerfile, "COPY --chown=app:app web /app/web") {
		t.Fatal("production image must not copy web sources")
	}
}
```

- [ ] **Step 2: Verify failure**

```bash
go test ./deploy -run 'TestDeploymentFiles|TestProductionImageCopiesOnlyBuiltWebAssets'
```

Expected: FAIL because the current Dockerfile has no Node stage and copies all of `web`.

- [ ] **Step 3: Replace `deploy/Dockerfile`**

```dockerfile
FROM node:22-alpine AS web-build
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.24-alpine AS go-build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /personal-agent ./cmd/personal-agent

FROM alpine:3.22
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 app \
    && mkdir -p /data \
    && chown app:app /data
USER app
WORKDIR /app
COPY --from=go-build /personal-agent /usr/local/bin/personal-agent
COPY --from=web-build --chown=app:app /src/web/dist /app/web/dist
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:8080/health || exit 1
CMD ["personal-agent"]
```

- [ ] **Step 4: Verify test and image**

```bash
go test ./deploy -run 'TestDeploymentFiles|TestProductionImageCopiesOnlyBuiltWebAssets'
docker build -f deploy/Dockerfile -t personal-agent:web-tooling .
```

Expected: PASS; Docker completes Node, Go, and runtime stages.

- [ ] **Step 5: Commit**

```bash
git add deploy/Dockerfile deploy/deploy_test.go
git commit -m "build(docker): bake Vite assets into production image"
```

---

### Task 6: Add the Development UI Reverse Proxy

**Files:**
- Create: `internal/httpapi/ui_proxy.go`, `internal/httpapi/ui_proxy_test.go`
- Modify: `internal/httpapi/server.go`, `internal/app/app.go`
- Test: `internal/httpapi/ui_proxy_test.go`

**Interfaces:**
- Consumes: `PA_UI_DEV_PROXY`; upstream `http://127.0.0.1:5173`; `ServerDeps.Static`.
- Produces: `NewUIProxy(rawURL string) (http.Handler, error)`; `ServerDeps.UI http.Handler`; transparent HTTP/WebSocket UI fallback while API and health patterns remain Go-owned.

- [ ] **Step 1: Write the failing proxy tests**

`internal/httpapi/ui_proxy_test.go`:

```go
package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUIProxyForwardsRequestToVite(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/src/App.svelte" || r.URL.RawQuery != "direct=1" { t.Fatalf("unexpected URL %s", r.URL.String()) }
		w.Header().Set("X-Vite", "yes")
		_, _ = io.WriteString(w, "compiled svelte")
	}))
	defer upstream.Close()
	h, err := NewUIProxy(upstream.URL)
	if err != nil { t.Fatal(err) }
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/src/App.svelte?direct=1", nil))
	if w.Code != http.StatusOK || w.Header().Get("X-Vite") != "yes" || w.Body.String() != "compiled svelte" {
		t.Fatalf("code=%d header=%q body=%q", w.Code, w.Header().Get("X-Vite"), w.Body.String())
	}
}

func TestUIProxyRejectsInvalidURL(t *testing.T) {
	if _, err := NewUIProxy("://bad"); err == nil { t.Fatal("expected invalid URL error") }
}
```

- [ ] **Step 2: Verify failure**

```bash
go test ./internal/httpapi -run TestUIProxy
```

Expected: FAIL to compile with `undefined: NewUIProxy`.

- [ ] **Step 3: Implement and wire the proxy**

`internal/httpapi/ui_proxy.go`:

```go
package httpapi

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func NewUIProxy(rawURL string) (http.Handler, error) {
	target, err := url.ParseRequestURI(rawURL)
	if err != nil || target.Scheme == "" || target.Host == "" {
		return nil, fmt.Errorf("parse PA_UI_DEV_PROXY %q: %w", rawURL, err)
	}
	return httputil.NewSingleHostReverseProxy(target), nil
}
```

Add to `ServerDeps` in `server.go`:

```go
	UI http.Handler
```

Replace the direct `GET /` registration with:

```go
	ui := deps.UI
	if ui == nil { ui = http.FileServer(deps.Static) }
	mux.Handle("GET /", ui)
```

In `app.go`, import `os`, then after selecting `static` add:

```go
	var ui http.Handler
	if rawURL := os.Getenv("PA_UI_DEV_PROXY"); rawURL != "" {
		ui, err = httpapi.NewUIProxy(rawURL)
		if err != nil { _ = db.Close(); return nil, err }
	}
```

Pass the exact field in `ServerDeps`:

```go
			UI:                   ui,
```

- [ ] **Step 4: Verify focused and full tests**

```bash
gofmt -w internal/httpapi/ui_proxy.go internal/httpapi/ui_proxy_test.go internal/httpapi/server.go internal/app/app.go
go test ./internal/httpapi -run 'TestUIProxy|TestServer'
go test ./...
```

Expected: PASS. Go's specific API/health patterns win over `GET /`; `httputil.ReverseProxy` tunnels Vite's WebSocket upgrade.

- [ ] **Step 5: Commit**

```bash
git add internal/httpapi/ui_proxy.go internal/httpapi/ui_proxy_test.go internal/httpapi/server.go internal/app/app.go
git commit -m "feat(http): proxy UI requests to Vite in development"
```

---

### Task 7: Run Air and Vite Together in Docker Development

**Files:**
- Modify: `deploy/Dockerfile.dev`, `deploy/docker-compose.dev.yml`, `deploy/deploy_test.go`
- Create: `deploy/dev-entrypoint.sh`
- Test: `deploy/deploy_test.go`

**Interfaces:**
- Consumes: `/src` mount; `deploy/air.toml`; npm lockfile; `PA_UI_DEV_PROXY`.
- Produces: Node 22 + Go 1.24 image; `dev-entrypoint`; Vite at container `127.0.0.1:5173`; browser/HMR through `localhost:8080`.

- [ ] **Step 1: Add failing deployment checks**

Use these `TestDeploymentFiles` entries:

```go
"Dockerfile.dev":         {"node:22-alpine", "golang:1.24", "air", "dev-entrypoint.sh"},
"docker-compose.dev.yml": {"Dockerfile.dev", "..:/src", "go-mod-cache:", "PA_UI_DEV_PROXY: http://127.0.0.1:5173", "dev-entrypoint"},
"dev-entrypoint.sh":      {"npm --prefix web ci", "npm --prefix web run dev", "air -c deploy/air.toml", "trap cleanup"},
```

Add `PA_UI_DEV_PROXY: http://127.0.0.1:5173` and `dev-entrypoint` to `TestComposeDevOverrideMountsFullRepo`'s required strings.

- [ ] **Step 2: Verify failure**

```bash
go test ./deploy -run 'TestDeploymentFiles|TestComposeDevOverrideMountsFullRepo'
```

Expected: FAIL because the image has no Node runtime or process runner.

- [ ] **Step 3: Add the exact image, runner, and override**

`deploy/Dockerfile.dev`:

```dockerfile
FROM node:22-alpine

RUN apk add --no-cache go=~1.24 git ca-certificates curl \
    && GOBIN=/usr/local/bin go install github.com/air-verse/air@v1.61.7
COPY deploy/dev-entrypoint.sh /usr/local/bin/dev-entrypoint
RUN chmod +x /usr/local/bin/dev-entrypoint
WORKDIR /src
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=30s --retries=5 \
    CMD curl -sf http://127.0.0.1:8080/health >/dev/null || exit 1
CMD ["dev-entrypoint"]
```

`deploy/dev-entrypoint.sh`:

```sh
#!/bin/sh
set -eu

cleanup() {
  trap - INT TERM EXIT
  kill "${vite_pid:-}" "${air_pid:-}" 2>/dev/null || true
  wait "${vite_pid:-}" "${air_pid:-}" 2>/dev/null || true
}
trap cleanup INT TERM EXIT

npm --prefix web ci
npm --prefix web run dev &
vite_pid=$!
air -c deploy/air.toml &
air_pid=$!
while kill -0 "$vite_pid" 2>/dev/null && kill -0 "$air_pid" 2>/dev/null; do sleep 1; done
exit 1
```

In `docker-compose.dev.yml`, preserve all current volumes, replace `command`, environment, and stale comments with:

```yaml
    command: ["dev-entrypoint"]
    volumes:
      - pa-data:/data
      # Full repo: Air reloads Go; Vite reads web/src directly for HMR.
      - ..:/src
      - go-mod-cache:/go/pkg/mod
      - go-build-cache:/root/.cache/go-build
    environment:
      PA_DATA_DIR: /data
      PA_ADDR: :8080
      PA_UI_DEV_PROXY: http://127.0.0.1:5173
```

- [ ] **Step 4: Verify tests, Compose, and HMR**

```bash
chmod +x deploy/dev-entrypoint.sh
go test ./deploy -run 'TestDeploymentFiles|TestComposeDevOverrideMountsFullRepo|TestComposeDefaultsAreSafeForLocalHTTP'
docker compose --env-file deploy/.env -f deploy/docker-compose.yml -f deploy/docker-compose.dev.yml config | grep 'PA_UI_DEV_PROXY: http://127.0.0.1:5173'
make docker-dev
```

Expected: PASS; logs show Vite on 5173 and Air/Go on 8080. In another terminal:

```bash
curl -sf http://127.0.0.1:8080/ | grep 'src/main.ts'
curl -sf http://127.0.0.1:8080/src/App.svelte | grep 'Personal Agent'
```

Expected: PASS. Edit the heading in `web/src/App.svelte`: the open `http://localhost:8080` page updates without refresh and DevTools shows `ws://localhost:8080/@vite-hmr`. Restore the heading and run `make docker-dev-down`.

- [ ] **Step 5: Commit**

```bash
git add deploy/Dockerfile.dev deploy/docker-compose.dev.yml deploy/dev-entrypoint.sh deploy/deploy_test.go
git commit -m "build(docker): run Air and Vite with HMR"
```

---

### Task 8: Lock Deployment Invariants and Document HMR

**Files:**
- Modify: `deploy/deploy_test.go`, `docs/ops/deploy.md`
- Test: `deploy/deploy_test.go`

**Interfaces:**
- Consumes: `make docker-dev`; `PA_UI_DEV_PROXY=http://127.0.0.1:5173`; port `8080`; HMR path `/@vite-hmr`.
- Produces: tests forbidding development wiring in production compose; exact HMR operating guide.

- [ ] **Step 1: Add a failing docs/isolation test**

```go
func TestDockerDevHMRIsDocumentedAndProductionIsMountFree(t *testing.T) {
	docs := readFile(t, "../docs/ops/deploy.md")
	for _, required := range []string{
		"http://localhost:8080",
		"PA_UI_DEV_PROXY=http://127.0.0.1:5173",
		"ws://localhost:8080/@vite-hmr",
		"Production compose has no host source mounts",
	} {
		if !strings.Contains(docs, required) { t.Errorf("deploy docs missing %q", required) }
	}
	prod := readFile(t, "docker-compose.yml")
	for _, forbidden := range []string{"../web:", "..:/src", "PA_UI_DEV_PROXY", "5173:5173"} {
		if strings.Contains(prod, forbidden) { t.Errorf("production compose must not contain %q", forbidden) }
	}
}
```

- [ ] **Step 2: Verify failure**

```bash
go test ./deploy -run TestDockerDevHMRIsDocumentedAndProductionIsMountFree
```

Expected: FAIL because docs still describe hard-refreshing vanilla assets.

- [ ] **Step 3: Replace the live-reload section in `docs/ops/deploy.md`**

````markdown
### Live-reload development (Go + Vite HMR)

Production compose uses a baked image. Start the one-command API and UI loop:

```sh
make docker-dev
# open http://localhost:8080
# stop: make docker-dev-down
```

`deploy/docker-compose.dev.yml` mounts the repository at `/src` and starts `deploy/dev-entrypoint.sh`. The script runs Air for Go reloads and Vite for Svelte/TypeScript/CSS HMR. Go remains the only browser-facing server on port 8080: API and health requests terminate in Go, while non-API GETs are proxied because the override sets `PA_UI_DEV_PROXY=http://127.0.0.1:5173`.

Vite listens only inside the container on port 5173. Its client uses `host: localhost`, `clientPort: 8080`, and `path: /@vite-hmr`; Go carries `ws://localhost:8080/@vite-hmr`. Edits under `web/src/` update without rebuilding or recreating the container. Air continues to exclude `web/`, because Vite owns frontend watching.

Before claiming a localhost UI change works:

```sh
lsof -nP -iTCP:8080 -sTCP:LISTEN
curl -sf http://127.0.0.1:8080/src/App.svelte | grep 'Personal Agent'
```

Confirm `/@vite-hmr` is connected in DevTools → Network → WS. Production compose has no host source mounts: `deploy/Dockerfile` builds `web/dist` with Node 22 and copies only that output. Never add `..:/src`, `../web:`, `PA_UI_DEV_PROXY`, or a published Vite port to `deploy/docker-compose.yml`.
````

- [ ] **Step 4: Run complete tooling verification**

```bash
npm --prefix web test
npm --prefix web run build
go test ./deploy ./internal/app ./internal/httpapi
go test ./...
! docker compose --env-file deploy/.env -f deploy/docker-compose.yml config | grep -E '(\.\.:/src|PA_UI_DEV_PROXY|5173:5173)'
```

Expected: PASS; web test/build and all Go tests pass, and production Compose emits no development-only wiring.

- [ ] **Step 5: Commit**

```bash
git add deploy/deploy_test.go docs/ops/deploy.md
git commit -m "docs: document Docker Vite HMR workflow"
```
