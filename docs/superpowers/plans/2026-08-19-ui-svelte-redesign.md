# UI Svelte Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the vanilla JS UI with a Svelte 5 + TypeScript dashboard that has a Global vs Vault context sidebar, searchable project/vault grids, and instant UI reload under `make docker-dev`.

**Architecture:** Vite builds `web/` → `web/dist` served by Go. In docker-dev, Air (Go) and Vite run together; Go proxies non-API UI to Vite with HMR on `:8080`. Hash router drives shell context (global | vault). Existing `/api/v1` only — client-side filters for unfiled/vault scope.

**Tech Stack:** Svelte 5, TypeScript, Vite, Tailwind CSS, Vitest, npm, Node 22 LTS, Go 1.24+, Air, Docker Compose.

**Spec:** `docs/superpowers/specs/2026-08-19-ui-svelte-redesign-design.md`  
**Lock:** `docs/superpowers/plans/2026-08-19-ui-svelte-redesign-lock.md`  
**Drafts (historical):** `docs/superpowers/plans/2026-08-19-ui-svelte-redesign-drafts/`

## Global Constraints

- Hash routing only (no History API requirement).
- npm + committed `web/package-lock.json`; Node **22** in Docker images.
- Production static root is **`web/dist`**; Go default `http.Dir("web/dist")`.
- Production compose stays **image-baked** (no `..:/src`, no `Dockerfile.dev` on prod file).
- Live loop is **`make docker-dev`** only; UI edits must HMR without image rebuild.
- `.DEFAULT_GOAL := help`; public Make targets need `##`, `.PHONY`, and `print-help-section` entry.
- Session chat polling must **never** replace a focused composer (patch messages/status in place).
- No new backend features required; vault APIs already exist. Project `vault_id` immutable.
- Unfiled project = empty/missing `vault_id`; vault badge shows **vault name** when set.
- Contract strings for promote/review/status badges must remain findable by Go `web_test` (update paths to new sources).
- Prefer TDD: failing test → implement → pass → commit per task.
- Do not put live source mounts on production compose.

## Canonical contracts

These names win over any conflicting snippet in task bodies.

### Shell context and routes

```ts
export type ShellContext =
  | { mode: 'global' }
  | { mode: 'vault'; vaultId: string; vaultName: string };

export type AppRoute =
  | { name: 'home' }
  | { name: 'projects' }
  | { name: 'vaults' }
  | { name: 'vault-home'; vaultId: string }
  | { name: 'vault-projects'; vaultId: string }
  | { name: 'vault-sessions'; vaultId: string }
  | { name: 'vault-review'; vaultId: string }
  | { name: 'project'; projectId: string }
  | { name: 'notes'; projectId: string }
  | { name: 'note'; projectId: string; noteId: string }
  | { name: 'sessions'; projectId: string }
  | { name: 'project-review'; projectId: string }
  | { name: 'review'; scope: string | null }
  | { name: 'settings' };

export function parseRoute(hash: string): AppRoute;
export function routeToHash(route: AppRoute): string;
```

Hash examples: `#/home`, `#/projects`, `#/vaults`, `#/vaults/{id}`, `#/vaults/{id}/projects`, `#/vaults/{id}/sessions`, `#/vaults/{id}/review`, `#/projects/{id}`, `#/projects/{id}/notes`, `#/projects/{id}/notes/{noteId}`, `#/projects/{id}/sessions`, `#/projects/{id}/review`, `#/review?scope=all`, `#/settings`.

### Filtering

```ts
export function isUnfiled(p: { vault_id?: string | null }): boolean;
// true when vault_id is null, undefined, or ""

export function filterProjectsByContext(
  projects: Array<{ vault_id?: string | null }>,
  ctx: ShellContext,
): typeof projects;

export function filterByQuery<T extends { name: string }>(items: T[], query: string): T[];
// case-insensitive substring on name; empty query returns all

export function filterReviewByProjectIds<T extends { project_id: string }>(
  items: T[],
  projectIds: Set<string>,
): T[];
```

### API client

- `class APIError extends Error { status: number }`
- `request(path, options)`, `get`, `api(path)` → `/api/v1` prefix
- Mutating methods send `X-CSRF-Token` from cookie `pa_csrf` (URL-decoded)
- Cookies: session `pa_session` (HttpOnly) — browser only

### Dev proxy

- Env `PA_UI_DEV_PROXY` (e.g. `http://127.0.0.1:5173`)
- When set: Go proxies non-`/api/` and non-`/health` GETs (and Vite HMR paths) to that origin
- Browser always uses host port **8080**
- Vite HMR path through proxy: `/@vite-hmr` (or as configured in task; keep one path)

### Sidebar

- Global nav: Home, Projects, Sessions (unfiled scope entry), Vaults, Review, Settings
- Vault nav: Leave control, Home, Projects, Sessions, Review, Settings
- Collapse: `localStorage['pa.sidebarCollapsed']` = `'true' | 'false'`

### Focus invariant

When polling session messages/run status: update message list and status nodes only. Do not remount the composer textarea while `document.activeElement` is the composer.

### Build layout

```text
web/                    # Vite app root (npm)
  package.json
  vite.config.ts
  index.html
  src/
    main.ts
    App.svelte
    app.css
    lib/api/
    lib/router/
    lib/stores/
    lib/filters/
    components/
    routes/ or pages/
  dist/                 # build output (gitignored)
web-legacy/             # temporary vanilla move; deleted in Task 52
```

Go: `internal/app` default static `web/dist`; optional dev UI reverse proxy module as planned in Tasks 4–6.

## File map (target)

| Path | Responsibility |
|------|----------------|
| `web/package.json` | npm scripts: dev, build, test |
| `web/vite.config.ts` | Svelte, Tailwind, Vitest, HMR |
| `web/src/lib/router/*` | parseRoute, routeToHash |
| `web/src/lib/stores/shell.ts` | context + collapsed |
| `web/src/lib/api/*` | typed fetch + CSRF |
| `web/src/lib/filters/*` | unfiled/vault/search/review filters |
| `web/src/components/*` | shell, cards, empty, skeleton, dialog |
| `web/src/routes/*` or `pages/*` | screen components |
| `internal/httpapi` or `internal/uidev` | static + optional PA_UI_DEV_PROXY |
| `deploy/Dockerfile` | Node build stage → copy dist |
| `deploy/Dockerfile.dev` | Go + Node 22 + air + entrypoint |
| `deploy/dev-entrypoint.sh` | air + vite |
| `deploy/docker-compose.dev.yml` | PA_UI_DEV_PROXY, mounts |
| `Makefile` | web-install, web-build, web-test, docker-dev |
| `docs/ops/deploy.md` | HMR / docker-dev UI loop |

## Task index

| Tasks | Phase |
|-------|--------|
| 1–8 | Tooling, static dir, Docker HMR |
| 10–15 | Router, shell, API client, auth, tokens |
| 20–25 | Home, Projects, Vaults grids |
| 30–35 | Vault context pages |
| 40–46 | Project hub, notes, sessions, promote |
| 50–55 | Review, settings, legacy removal, docs, gate |

---


## Phase: 01-tooling-docker

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


## Phase: 02-shell-auth

### Task 10: Add the typed hash router

**Files:**
- Create: `web/src/lib/router.ts`
- Create: `web/src/lib/router.test.ts`

**Interfaces:**
- Consumes: Browser hash strings from `window.location.hash`.
- Produces: `AppRoute`, `parseRoute(hash: string): AppRoute`, and `routeToHash(route: AppRoute): string` exactly as specified below.

- [ ] **Step 1: Write the failing route round-trip tests**

```ts
// web/src/lib/router.test.ts
import { describe, expect, it } from 'vitest';
import { parseRoute, routeToHash, type AppRoute } from './router';

const cases: Array<[string, AppRoute]> = [
  ['#/home', { name: 'home' }],
  ['#/projects', { name: 'projects' }],
  ['#/vaults', { name: 'vaults' }],
  ['#/vaults/health', { name: 'vault-home', vaultId: 'health' }],
  ['#/vaults/health/projects', { name: 'vault-projects', vaultId: 'health' }],
  ['#/vaults/health/sessions', { name: 'vault-sessions', vaultId: 'health' }],
  ['#/vaults/health/review', { name: 'vault-review', vaultId: 'health' }],
  ['#/projects/p1', { name: 'project', projectId: 'p1' }],
  ['#/projects/p1/notes', { name: 'notes', projectId: 'p1' }],
  ['#/projects/p1/notes/n1', { name: 'note', projectId: 'p1', noteId: 'n1' }],
  ['#/projects/p1/sessions', { name: 'sessions', projectId: 'p1' }],
  ['#/projects/p1/review', { name: 'project-review', projectId: 'p1' }],
  ['#/review', { name: 'review', scope: null }],
  ['#/review?scope=all', { name: 'review', scope: 'all' }],
  ['#/settings', { name: 'settings' }],
];

describe('hash router', () => {
  it.each(cases)('parses and serializes %s', (hash, route) => {
    expect(parseRoute(hash)).toEqual(route);
    expect(routeToHash(route)).toBe(hash);
  });

  it('decodes path and query values and encodes them on output', () => {
    const route: AppRoute = { name: 'note', projectId: 'project one', noteId: 'a/b' };
    expect(parseRoute(routeToHash(route))).toEqual(route);
    expect(parseRoute('#/review?scope=due%20today')).toEqual({ name: 'review', scope: 'due today' });
  });

  it.each(['', '#', '#/', '#/unknown', '#/vaults/v/unknown', '#settings'])(
    'falls back or supports a legacy hash: %s',
    (hash) => {
      expect(parseRoute(hash)).toEqual(hash === '#settings' ? { name: 'settings' } : { name: 'home' });
    },
  );
});
```

- [ ] **Step 2: Run the test and verify it fails because the router module is absent**

Run: `cd web && npm test -- --run src/lib/router.test.ts`  
Expected: FAIL with “Failed to resolve import './router'”.

- [ ] **Step 3: Implement the route union and pure parser/serializer**

```ts
// web/src/lib/router.ts
export type AppRoute =
  | { name: 'home' }
  | { name: 'projects' }
  | { name: 'vaults' }
  | { name: 'vault-home'; vaultId: string }
  | { name: 'vault-projects'; vaultId: string }
  | { name: 'vault-sessions'; vaultId: string }
  | { name: 'vault-review'; vaultId: string }
  | { name: 'project'; projectId: string }
  | { name: 'notes'; projectId: string }
  | { name: 'note'; projectId: string; noteId: string }
  | { name: 'sessions'; projectId: string }
  | { name: 'project-review'; projectId: string }
  | { name: 'review'; scope: string | null }
  | { name: 'settings' };

const part = (value: string) => decodeURIComponent(value);
const encoded = (value: string) => encodeURIComponent(value);

export function parseRoute(hash: string): AppRoute {
  if (hash === '#settings') return { name: 'settings' };
  const raw = hash.startsWith('#') ? hash.slice(1) : hash;
  const [pathname, query = ''] = raw.split('?', 2);
  const segments = pathname.split('/').filter(Boolean).map(part);

  if (segments[0] === 'home' && segments.length === 1) return { name: 'home' };
  if (segments[0] === 'projects' && segments.length === 1) return { name: 'projects' };
  if (segments[0] === 'vaults' && segments.length === 1) return { name: 'vaults' };
  if (segments[0] === 'settings' && segments.length === 1) return { name: 'settings' };
  if (segments[0] === 'review' && segments.length === 1) {
    return { name: 'review', scope: new URLSearchParams(query).get('scope') };
  }
  if (segments[0] === 'vaults' && segments[1] && segments.length === 2) {
    return { name: 'vault-home', vaultId: segments[1] };
  }
  if (segments[0] === 'vaults' && segments[1] && segments.length === 3) {
    const name = { projects: 'vault-projects', sessions: 'vault-sessions', review: 'vault-review' }[
      segments[2]
    ] as 'vault-projects' | 'vault-sessions' | 'vault-review' | undefined;
    if (name) return { name, vaultId: segments[1] };
  }
  if (segments[0] === 'projects' && segments[1]) {
    const projectId = segments[1];
    if (segments.length === 2) return { name: 'project', projectId };
    if (segments[2] === 'notes' && segments.length === 3) return { name: 'notes', projectId };
    if (segments[2] === 'notes' && segments[3] && segments.length === 4) {
      return { name: 'note', projectId, noteId: segments[3] };
    }
    if (segments[2] === 'sessions' && segments.length === 3) return { name: 'sessions', projectId };
    if (segments[2] === 'review' && segments.length === 3) return { name: 'project-review', projectId };
  }
  return { name: 'home' };
}

export function routeToHash(route: AppRoute): string {
  switch (route.name) {
    case 'home': return '#/home';
    case 'projects': return '#/projects';
    case 'vaults': return '#/vaults';
    case 'vault-home': return `#/vaults/${encoded(route.vaultId)}`;
    case 'vault-projects': return `#/vaults/${encoded(route.vaultId)}/projects`;
    case 'vault-sessions': return `#/vaults/${encoded(route.vaultId)}/sessions`;
    case 'vault-review': return `#/vaults/${encoded(route.vaultId)}/review`;
    case 'project': return `#/projects/${encoded(route.projectId)}`;
    case 'notes': return `#/projects/${encoded(route.projectId)}/notes`;
    case 'note': return `#/projects/${encoded(route.projectId)}/notes/${encoded(route.noteId)}`;
    case 'sessions': return `#/projects/${encoded(route.projectId)}/sessions`;
    case 'project-review': return `#/projects/${encoded(route.projectId)}/review`;
    case 'review': return route.scope === null ? '#/review' : `#/review?scope=${encoded(route.scope)}`;
    case 'settings': return '#/settings';
  }
}
```

- [ ] **Step 4: Run the focused tests**

Run: `cd web && npm test -- --run src/lib/router.test.ts`  
Expected: PASS (all route cases).

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/router.ts web/src/lib/router.test.ts
git commit -m "feat(web): add typed hash router"
```

### Task 11: Derive shell context from routes and vault data

**Files:**
- Create: `web/src/lib/stores/shell-context.ts`
- Create: `web/src/lib/stores/shell-context.test.ts`

**Interfaces:**
- Consumes: `AppRoute` from Task 10 and loaded `VaultSummary[]`; project deep links may supply the loaded project's `vault_id`.
- Produces: `ShellContext`, `findVaultName(vaults, vaultId)`, `deriveShellContext(route, vaults, project?)`, and writable `shellContext`.

- [ ] **Step 1: Write failing helper and store tests**

```ts
// web/src/lib/stores/shell-context.test.ts
import { get } from 'svelte/store';
import { describe, expect, it } from 'vitest';
import { deriveShellContext, findVaultName, shellContext } from './shell-context';

const vaults = [{ id: 'v1', name: 'HEALTH' }, { id: 'v2', name: 'WORK' }];

describe('shell context', () => {
  it('looks up a vault name without mutating input', () => {
    expect(findVaultName(vaults, 'v1')).toBe('HEALTH');
    expect(findVaultName(vaults, 'missing')).toBeNull();
  });

  it('derives vault context from every vault route', () => {
    for (const name of ['vault-home', 'vault-projects', 'vault-sessions', 'vault-review'] as const) {
      expect(deriveShellContext({ name, vaultId: 'v1' }, vaults)).toEqual({
        mode: 'vault', vaultId: 'v1', vaultName: 'HEALTH',
      });
    }
  });

  it('uses project membership for project deep links', () => {
    expect(deriveShellContext({ name: 'project', projectId: 'p1' }, vaults, { vault_id: 'v2' }))
      .toEqual({ mode: 'vault', vaultId: 'v2', vaultName: 'WORK' });
    expect(deriveShellContext({ name: 'notes', projectId: 'p2' }, vaults, { vault_id: null }))
      .toEqual({ mode: 'global' });
  });

  it('falls back safely when vault data is unavailable', () => {
    expect(deriveShellContext({ name: 'vault-home', vaultId: 'missing' }, vaults))
      .toEqual({ mode: 'vault', vaultId: 'missing', vaultName: 'Vault' });
    shellContext.set({ mode: 'global' });
    expect(get(shellContext)).toEqual({ mode: 'global' });
  });
});
```

- [ ] **Step 2: Run the test and verify the missing module failure**

Run: `cd web && npm test -- --run src/lib/stores/shell-context.test.ts`  
Expected: FAIL resolving `./shell-context`.

- [ ] **Step 3: Implement pure context derivation and the store**

```ts
// web/src/lib/stores/shell-context.ts
import { writable } from 'svelte/store';
import type { AppRoute } from '../router';

export type ShellContext =
  | { mode: 'global' }
  | { mode: 'vault'; vaultId: string; vaultName: string };

export type VaultSummary = { id: string; name: string };
export type ProjectMembership = { vault_id?: string | null };

export const shellContext = writable<ShellContext>({ mode: 'global' });

export function findVaultName(vaults: readonly VaultSummary[], vaultId: string): string | null {
  return vaults.find((vault) => vault.id === vaultId)?.name ?? null;
}

export function deriveShellContext(
  route: AppRoute,
  vaults: readonly VaultSummary[],
  project?: ProjectMembership,
): ShellContext {
  if (route.name.startsWith('vault-')) {
    return { mode: 'vault', vaultId: route.vaultId, vaultName: findVaultName(vaults, route.vaultId) ?? 'Vault' };
  }
  if (['project', 'notes', 'note', 'sessions', 'project-review'].includes(route.name) && project?.vault_id) {
    return {
      mode: 'vault',
      vaultId: project.vault_id,
      vaultName: findVaultName(vaults, project.vault_id) ?? 'Vault',
    };
  }
  return { mode: 'global' };
}
```

- [ ] **Step 4: Run focused tests**

Run: `cd web && npm test -- --run src/lib/stores/shell-context.test.ts`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/stores/shell-context.ts web/src/lib/stores/shell-context.test.ts
git commit -m "feat(web): derive URL-driven shell context"
```

### Task 12: Port the API client to TypeScript

**Files:**
- Create: `web/src/lib/api/client.ts`
- Create: `web/src/lib/api/client.test.ts`

**Interfaces:**
- Consumes: Same-origin `/api/v1` endpoints, `document.cookie`, and standard `fetch`.
- Produces: `APIError`, `request<T>(path, options?)`, `api<T>(path, options?)`, `get<T>(path)`, and `mutate<T>(path, method, body)`; mutating requests send decoded `pa_csrf` as `X-CSRF-Token`.

- [ ] **Step 1: Write failing API client tests**

```ts
// web/src/lib/api/client.test.ts
import { afterEach, describe, expect, it, vi } from 'vitest';
import { APIError, request } from './client';

afterEach(() => vi.unstubAllGlobals());

describe('request', () => {
  it('parses API errors into APIError', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ code: 'bad_request', message: 'Choose another name' }),
      { status: 400, headers: { 'Content-Type': 'application/json' } },
    )));
    await expect(request('/api/v1/vaults')).rejects.toEqual(
      expect.objectContaining<Partial<APIError>>({ status: 400, message: 'Choose another name' }),
    );
  });

  it('adds JSON and CSRF headers to POST requests', async () => {
    document.cookie = 'pa_csrf=token%2Fvalue; path=/';
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal('fetch', fetchMock);
    await request('/api/v1/auth/logout', { method: 'POST', body: {} });
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/auth/logout', expect.objectContaining({
      method: 'POST',
      body: '{}',
      headers: expect.objectContaining({
        Accept: 'application/json',
        'Content-Type': 'application/json',
        'X-CSRF-Token': 'token/value',
      }),
    }));
  });

  it('does not attach CSRF to GET and returns null for 204', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal('fetch', fetchMock);
    await expect(request('/health')).resolves.toBeNull();
    expect(fetchMock.mock.calls[0][1].headers).not.toHaveProperty('X-CSRF-Token');
  });
});
```

- [ ] **Step 2: Verify the client tests fail**

Run: `cd web && npm test -- --run src/lib/api/client.test.ts`  
Expected: FAIL resolving `./client`.

- [ ] **Step 3: Implement the typed client**

```ts
// web/src/lib/api/client.ts
export class APIError extends Error {
  constructor(public readonly status: number, message: string) {
    super(message);
    this.name = 'APIError';
  }
}

export type RequestOptions = Omit<RequestInit, 'body'> & { body?: unknown };

function cookie(name: string): string | undefined {
  const value = document.cookie.split('; ').find((entry) => entry.startsWith(`${name}=`));
  return value?.slice(name.length + 1);
}

export async function request<T = unknown>(path: string, options: RequestOptions = {}): Promise<T | null> {
  const method = (options.method ?? 'GET').toUpperCase();
  const headers = new Headers(options.headers);
  headers.set('Accept', 'application/json');
  let body: BodyInit | undefined;
  if (options.body !== undefined) {
    headers.set('Content-Type', 'application/json');
    body = typeof options.body === 'string' ? options.body : JSON.stringify(options.body);
  }
  if (!['GET', 'HEAD'].includes(method)) {
    const csrf = cookie('pa_csrf');
    if (csrf) headers.set('X-CSRF-Token', decodeURIComponent(csrf));
  }
  const response = await fetch(path, { ...options, method, headers: Object.fromEntries(headers), body });
  const text = response.status === 204 ? '' : await response.text();
  if (!response.ok) {
    let message = text.trim();
    try {
      const data = JSON.parse(text) as { message?: string; code?: string; error?: string };
      message = data.message ?? data.code ?? data.error?.replaceAll('_', ' ') ?? message;
    } catch { /* retain plain-text response */ }
    throw new APIError(response.status, message || `Request failed (${response.status})`);
  }
  return text.trim() ? JSON.parse(text) as T : null;
}

export const get = <T>(path: string) => request<T>(path);
export const mutate = <T>(path: string, method: string, body: unknown) => request<T>(path, { method, body });
export const api = <T>(path: string, options?: RequestOptions) => request<T>(`/api/v1${path}`, options);
```

- [ ] **Step 4: Run API client tests**

Run: `cd web && npm test -- --run src/lib/api/client.test.ts`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/api/client.ts web/src/lib/api/client.test.ts
git commit -m "feat(web): port API client to TypeScript"
```

### Task 13: Build the context-aware application shell

**Files:**
- Create: `web/src/shell/sidebar-state.ts`
- Create: `web/src/shell/Sidebar.svelte`
- Create: `web/src/shell/TopBar.svelte`
- Create: `web/src/shell/AppShell.svelte`
- Create: `web/src/shell/Sidebar.test.ts`
- Create: `web/src/shell/AppShell.test.ts`

**Interfaces:**
- Consumes: `ShellContext` from Task 11, `AppRoute`/`routeToHash` from Task 10, storage health text, and `localStorage['pa.sidebarCollapsed']`.
- Produces: `Sidebar`, `TopBar`, and `AppShell`; `readSidebarCollapsed(storage)` and `writeSidebarCollapsed(storage, value)`. The global Sessions row is present but disabled until a canonical global-sessions route exists; do not add a route outside the canonical `AppRoute` union.

- [ ] **Step 1: Write failing shell tests**

```ts
// web/src/shell/Sidebar.test.ts
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';
import Sidebar from './Sidebar.svelte';

afterEach(cleanup);

describe('Sidebar', () => {
  it('shows global navigation and persists collapse', async () => {
    localStorage.clear();
    render(Sidebar, { context: { mode: 'global' }, route: { name: 'home' } });
    for (const label of ['Home', 'Projects', 'Sessions', 'Vaults', 'Review', 'Settings']) {
      expect(screen.getByText(label)).toBeInTheDocument();
    }
    await fireEvent.click(screen.getByRole('button', { name: 'Collapse sidebar' }));
    expect(localStorage.getItem('pa.sidebarCollapsed')).toBe('true');
    expect(screen.getByTestId('sidebar')).toHaveAttribute('data-collapsed', 'true');
  });

  it('replaces global navigation in vault context', () => {
    render(Sidebar, {
      context: { mode: 'vault', vaultId: 'v1', vaultName: 'HEALTH' },
      route: { name: 'vault-home', vaultId: 'v1' },
    });
    expect(screen.getByText('HEALTH')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Leave vault' })).toHaveAttribute('href', '#/home');
    expect(screen.queryByText('Vaults')).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Projects' })).toHaveAttribute('href', '#/vaults/v1/projects');
  });
});
```

```ts
// web/src/shell/AppShell.test.ts
import { render, screen } from '@testing-library/svelte';
import { expect, it } from 'vitest';
import AppShell from './AppShell.svelte';

it('renders sidebar, top bar health, and content canvas', () => {
  render(AppShell, {
    context: { mode: 'global' }, route: { name: 'home' }, health: 'Storage ready',
    children: (() => ({ render: () => '<p>Dashboard</p>' })) as never,
  });
  expect(screen.getByRole('navigation', { name: 'Primary' })).toBeInTheDocument();
  expect(screen.getByText('Storage ready')).toBeInTheDocument();
  expect(screen.getByRole('main')).toBeInTheDocument();
});
```

- [ ] **Step 2: Run tests and verify missing-component failures**

Run: `cd web && npm test -- --run src/shell/Sidebar.test.ts src/shell/AppShell.test.ts`  
Expected: FAIL resolving `Sidebar.svelte` and `AppShell.svelte`.

- [ ] **Step 3: Implement persistence helpers**

```ts
// web/src/shell/sidebar-state.ts
export const SIDEBAR_COLLAPSED_KEY = 'pa.sidebarCollapsed';
export const readSidebarCollapsed = (storage: Storage) => storage.getItem(SIDEBAR_COLLAPSED_KEY) === 'true';
export const writeSidebarCollapsed = (storage: Storage, value: boolean) =>
  storage.setItem(SIDEBAR_COLLAPSED_KEY, String(value));
```

- [ ] **Step 4: Implement Sidebar**

```svelte
<!-- web/src/shell/Sidebar.svelte -->
<script lang="ts">
  import type { AppRoute } from '../lib/router';
  import { routeToHash } from '../lib/router';
  import type { ShellContext } from '../lib/stores/shell-context';
  import { readSidebarCollapsed, writeSidebarCollapsed } from './sidebar-state';

  let { context, route }: { context: ShellContext; route: AppRoute } = $props();
  let collapsed = $state(readSidebarCollapsed(localStorage));
  const globalItems = [
    ['Home', routeToHash({ name: 'home' })],
    ['Projects', routeToHash({ name: 'projects' })],
    ['Sessions', ''],
    ['Vaults', routeToHash({ name: 'vaults' })],
    ['Review', routeToHash({ name: 'review', scope: 'all' })],
    ['Settings', routeToHash({ name: 'settings' })],
  ] as const;
  const vaultItems = $derived(context.mode === 'vault' ? [
    ['Home', routeToHash({ name: 'vault-home', vaultId: context.vaultId })],
    ['Projects', routeToHash({ name: 'vault-projects', vaultId: context.vaultId })],
    ['Sessions', routeToHash({ name: 'vault-sessions', vaultId: context.vaultId })],
    ['Review', routeToHash({ name: 'vault-review', vaultId: context.vaultId })],
    ['Settings', routeToHash({ name: 'settings' })],
  ] as const : []);
  const items = $derived(context.mode === 'vault' ? vaultItems : globalItems);
  function toggle() {
    collapsed = !collapsed;
    writeSidebarCollapsed(localStorage, collapsed);
  }
</script>

<aside class="sidebar" data-testid="sidebar" data-collapsed={collapsed}>
  <div class="sidebar__brand">{collapsed ? 'PA' : 'Personal Agent'}</div>
  {#if context.mode === 'vault'}
    <div class="sidebar__context"><strong>{context.vaultName}</strong><a href="#/home">Leave vault</a></div>
  {/if}
  <nav aria-label="Primary">
    {#each items as item}
      {#if item[1]}
        <a href={item[1]} aria-current={item[1] === routeToHash(route) ? 'page' : undefined} title={item[0]}>
          <span aria-hidden="true">•</span><span class="sidebar__label">{item[0]}</span>
        </a>
      {:else}
        <span class="sidebar__disabled" aria-disabled="true" title="Choose a project to view sessions">
          <span aria-hidden="true">•</span><span class="sidebar__label">{item[0]}</span>
        </span>
      {/if}
    {/each}
  </nav>
  <button type="button" onclick={toggle} aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}>
    {collapsed ? '›' : '‹'}
  </button>
</aside>
```

- [ ] **Step 5: Implement TopBar and AppShell**

```svelte
<!-- web/src/shell/TopBar.svelte -->
<script lang="ts">let { health }: { health: string } = $props();</script>
<header class="topbar"><div class="topbar__spacer"></div><span class="health-pill">{health}</span></header>
```

```svelte
<!-- web/src/shell/AppShell.svelte -->
<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { AppRoute } from '../lib/router';
  import type { ShellContext } from '../lib/stores/shell-context';
  import Sidebar from './Sidebar.svelte';
  import TopBar from './TopBar.svelte';
  let { context, route, health, children }: {
    context: ShellContext; route: AppRoute; health: string; children: Snippet;
  } = $props();
</script>
<div class="app-shell">
  <Sidebar {context} {route} />
  <div class="app-shell__body"><TopBar {health} /><main class="content-canvas">{@render children()}</main></div>
</div>
```

- [ ] **Step 6: Run shell component tests**

Run: `cd web && npm test -- --run src/shell/Sidebar.test.ts src/shell/AppShell.test.ts`  
Expected: PASS. If the scaffold's Svelte Testing Library cannot construct a snippet prop, replace only the `AppShell.test.ts` fixture with the repository's established wrapper-component pattern; do not weaken the three assertions.

- [ ] **Step 7: Commit**

```bash
git add web/src/shell
git commit -m "feat(web): add context-aware application shell"
```

### Task 14: Bootstrap authentication and keep auth pages outside the shell

**Files:**
- Create: `web/src/lib/stores/auth.ts`
- Create: `web/src/lib/stores/auth.test.ts`
- Create: `web/src/routes/auth/AuthCard.svelte`
- Create: `web/src/routes/auth/BootstrapPage.svelte`
- Create: `web/src/routes/auth/LoginPage.svelte`
- Create: `web/src/routes/auth/AuthPages.test.ts`
- Modify: `web/src/App.svelte`
- Create: `web/src/App.test.ts`

**Interfaces:**
- Consumes: `request`/`APIError` from Task 12; boot calls `GET /api/v1/setup/status`, then (only when bootstrapped) `GET /api/v1/auth/me`.
- Produces: `AuthState = loading | bootstrap | login | authenticated | error`, `loadAuthState(client)`, accessible bootstrap/login pages, and `App.svelte` boot rendering with no `AppShell` around auth states.

- [ ] **Step 1: Write failing auth-state tests**

```ts
// web/src/lib/stores/auth.test.ts
import { describe, expect, it, vi } from 'vitest';
import { APIError } from '../api/client';
import { loadAuthState } from './auth';

describe('loadAuthState', () => {
  it('requests setup first and stops at bootstrap', async () => {
    const client = vi.fn().mockResolvedValueOnce({ bootstrapped: false });
    await expect(loadAuthState(client)).resolves.toEqual({ status: 'bootstrap' });
    expect(client).toHaveBeenCalledTimes(1);
    expect(client).toHaveBeenCalledWith('/api/v1/setup/status');
  });

  it('loads the owner after setup', async () => {
    const client = vi.fn()
      .mockResolvedValueOnce({ bootstrapped: true })
      .mockResolvedValueOnce({ owner: true });
    await expect(loadAuthState(client)).resolves.toEqual({ status: 'authenticated' });
    expect(client).toHaveBeenNthCalledWith(2, '/api/v1/auth/me');
  });

  it('maps a 401 from auth/me to login', async () => {
    const client = vi.fn().mockResolvedValueOnce({ bootstrapped: true })
      .mockRejectedValueOnce(new APIError(401, 'unauthorized'));
    await expect(loadAuthState(client)).resolves.toEqual({ status: 'login' });
  });
});
```

- [ ] **Step 2: Run and verify the auth-state test fails**

Run: `cd web && npm test -- --run src/lib/stores/auth.test.ts`  
Expected: FAIL resolving `./auth`.

- [ ] **Step 3: Implement authentication boot state**

```ts
// web/src/lib/stores/auth.ts
import { writable } from 'svelte/store';
import { APIError, get } from '../api/client';

export type AuthState =
  | { status: 'loading' }
  | { status: 'bootstrap' }
  | { status: 'login' }
  | { status: 'authenticated' }
  | { status: 'error'; message: string };
type Client = <T>(path: string) => Promise<T | null>;

export const authState = writable<AuthState>({ status: 'loading' });

export async function loadAuthState(client: Client = get): Promise<AuthState> {
  try {
    const setup = await client<{ bootstrapped: boolean }>('/api/v1/setup/status');
    if (!setup?.bootstrapped) return { status: 'bootstrap' };
    try {
      await client<{ owner: boolean }>('/api/v1/auth/me');
      return { status: 'authenticated' };
    } catch (error) {
      if (error instanceof APIError && error.status === 401) return { status: 'login' };
      throw error;
    }
  } catch (error) {
    return { status: 'error', message: error instanceof Error ? error.message : 'Could not start the app' };
  }
}

export async function refreshAuth(): Promise<void> {
  authState.set({ status: 'loading' });
  authState.set(await loadAuthState());
}
```

- [ ] **Step 4: Write failing auth-page component tests**

```ts
// web/src/routes/auth/AuthPages.test.ts
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import BootstrapPage from './BootstrapPage.svelte';
import LoginPage from './LoginPage.svelte';

afterEach(cleanup);

describe('auth pages', () => {
  it('submits bootstrap token and a 12+ character password', async () => {
    const submit = vi.fn().mockResolvedValue(null);
    render(BootstrapPage, { submit, onComplete: vi.fn() });
    await fireEvent.input(screen.getByLabelText('Bootstrap token'), { target: { value: 'token' } });
    await fireEvent.input(screen.getByLabelText('Password'), { target: { value: 'long-enough-password' } });
    await fireEvent.submit(screen.getByRole('form', { name: 'Set up owner account' }));
    expect(submit).toHaveBeenCalledWith('/api/v1/setup/bootstrap', 'POST', {
      token: 'token', password: 'long-enough-password',
    });
  });

  it('shows login errors beside the form', async () => {
    render(LoginPage, { submit: vi.fn().mockRejectedValue(new Error('Incorrect password')), onComplete: vi.fn() });
    await fireEvent.input(screen.getByLabelText('Password'), { target: { value: 'wrong' } });
    await fireEvent.submit(screen.getByRole('form', { name: 'Sign in' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Incorrect password');
  });
});
```

- [ ] **Step 5: Implement the shared auth card and pages**

```svelte
<!-- web/src/routes/auth/AuthCard.svelte -->
<script lang="ts">import type { Snippet } from 'svelte'; let { children }: { children: Snippet } = $props();</script>
<main class="auth-canvas"><section class="auth-card">{@render children()}</section></main>
```

```svelte
<!-- web/src/routes/auth/BootstrapPage.svelte -->
<script lang="ts">
  import { mutate } from '../../lib/api/client'; import AuthCard from './AuthCard.svelte';
  let { submit = mutate, onComplete }: { submit?: typeof mutate; onComplete: () => void | Promise<void> } = $props();
  let token = $state(''), password = $state(''), error = $state(''), pending = $state(false);
  async function handleSubmit() { pending = true; error = ''; try {
    await submit('/api/v1/setup/bootstrap', 'POST', { token, password }); await onComplete();
  } catch (cause) { error = cause instanceof Error ? cause.message : 'Setup failed'; pending = false; } }
</script>
<AuthCard><h1>Set up your owner account</h1><form aria-label="Set up owner account" onsubmit={(event) => { event.preventDefault(); void handleSubmit(); }}>
  <label>Bootstrap token<input bind:value={token} required autocomplete="off" /></label>
  <label>Password<input bind:value={password} type="password" minlength="12" required autocomplete="new-password" /></label>
  {#if error}<p role="alert">{error}</p>{/if}<button disabled={pending}>Continue</button>
</form></AuthCard>
```

```svelte
<!-- web/src/routes/auth/LoginPage.svelte -->
<script lang="ts">
  import { mutate } from '../../lib/api/client'; import AuthCard from './AuthCard.svelte';
  let { submit = mutate, onComplete }: { submit?: typeof mutate; onComplete: () => void | Promise<void> } = $props();
  let password = $state(''), error = $state(''), pending = $state(false);
  async function handleSubmit() { pending = true; error = ''; try {
    await submit('/api/v1/auth/login', 'POST', { password }); await onComplete();
  } catch (cause) { error = cause instanceof Error ? cause.message : 'Sign in failed'; pending = false; } }
</script>
<AuthCard><h1>Sign in</h1><form aria-label="Sign in" onsubmit={(event) => { event.preventDefault(); void handleSubmit(); }}>
  <label>Password<input bind:value={password} type="password" required autocomplete="current-password" /></label>
  {#if error}<p role="alert">{error}</p>{/if}<button disabled={pending}>Continue</button>
</form></AuthCard>
```

- [ ] **Step 6: Wire App.svelte boot and add a shell-exclusion regression test**

```ts
// web/src/App.test.ts
import { render, screen, waitFor } from '@testing-library/svelte';
import { expect, it, vi } from 'vitest';
import App from './App.svelte';

it('renders login without authenticated chrome', async () => {
  render(App, { authLoader: vi.fn().mockResolvedValue({ status: 'login' }) });
  await waitFor(() => expect(screen.getByRole('heading', { name: 'Sign in' })).toBeInTheDocument());
  expect(screen.queryByRole('navigation', { name: 'Primary' })).not.toBeInTheDocument();
});
```

```svelte
<!-- web/src/App.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { parseRoute, type AppRoute } from './lib/router';
  import { loadAuthState, refreshAuth, type AuthState } from './lib/stores/auth';
  import { deriveShellContext, type VaultSummary } from './lib/stores/shell-context';
  import AppShell from './shell/AppShell.svelte';
  import BootstrapPage from './routes/auth/BootstrapPage.svelte';
  import LoginPage from './routes/auth/LoginPage.svelte';
  let { authLoader = loadAuthState }: { authLoader?: typeof loadAuthState } = $props();
  let auth = $state<AuthState>({ status: 'loading' });
  let route = $state<AppRoute>(parseRoute(location.hash));
  let vaults = $state<VaultSummary[]>([]);
  const context = $derived(deriveShellContext(route, vaults));
  onMount(() => {
    const updateRoute = () => route = parseRoute(location.hash);
    addEventListener('hashchange', updateRoute);
    void authLoader().then((value) => auth = value);
    return () => removeEventListener('hashchange', updateRoute);
  });
  async function completeAuth() { await refreshAuth(); auth = await authLoader(); if (!location.hash) location.hash = '#/home'; }
</script>
{#if auth.status === 'loading'}
  <main class="auth-canvas" aria-busy="true">Starting…</main>
{:else if auth.status === 'bootstrap'}
  <BootstrapPage onComplete={completeAuth} />
{:else if auth.status === 'login'}
  <LoginPage onComplete={completeAuth} />
{:else if auth.status === 'error'}
  <main class="auth-canvas"><p role="alert">{auth.message}</p></main>
{:else}
  <AppShell {context} {route} health="Storage status unavailable"><p>Route: {route.name}</p></AppShell>
{/if}
```

- [ ] **Step 7: Run all auth and App tests**

Run: `cd web && npm test -- --run src/lib/stores/auth.test.ts src/routes/auth/AuthPages.test.ts src/App.test.ts`  
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add web/src/lib/stores/auth.ts web/src/lib/stores/auth.test.ts web/src/routes/auth web/src/App.svelte web/src/App.test.ts
git commit -m "feat(web): add auth bootstrap and login flow"
```

### Task 15: Establish visual tokens and the responsive layout baseline

**Files:**
- Modify: `web/src/app.css`
- Modify: `web/src/app.html`
- Modify: `web/tailwind.config.ts`
- Create: `web/src/styles-baseline.test.ts`

**Interfaces:**
- Consumes: `@fontsource/inter` installed by the tooling tasks and class names emitted by Tasks 13–14.
- Produces: light-first zinc neutral tokens, one blue accent, Inter typography, shell/auth surfaces, visible focus rings, desktop collapsed rail, and a `<768px` sidebar/content baseline.

- [ ] **Step 1: Write a failing static baseline test**

```ts
// web/src/styles-baseline.test.ts
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const css = readFileSync(new URL('./app.css', import.meta.url), 'utf8');
const html = readFileSync(new URL('./app.html', import.meta.url), 'utf8');

describe('visual baseline', () => {
  it('loads Inter once and declares required theme surfaces', () => {
    expect(css).toContain("@import '@fontsource/inter/variable.css'");
    for (const token of ['--canvas', '--panel', '--sidebar', '--border', '--accent', '--danger']) {
      expect(css).toContain(token);
    }
    expect(html).toContain('<meta name="viewport" content="width=device-width, initial-scale=1" />');
  });

  it('has focus and mobile shell rules', () => {
    expect(css).toContain(':focus-visible');
    expect(css).toContain('@media (max-width: 767px)');
    expect(css).not.toMatch(/linear-gradient|radial-gradient|backdrop-filter/);
  });
});
```

- [ ] **Step 2: Run and verify the baseline test fails**

Run: `cd web && npm test -- --run src/styles-baseline.test.ts`  
Expected: FAIL because Inter and the required tokens are absent.

- [ ] **Step 3: Configure the Tailwind theme**

```ts
// web/tailwind.config.ts
import type { Config } from 'tailwindcss';
export default {
  content: ['./src/**/*.{html,js,svelte,ts}', './src/app.html'],
  theme: {
    extend: {
      fontFamily: { sans: ['Inter Variable', 'Inter', 'system-ui', 'sans-serif'] },
      colors: { accent: { DEFAULT: '#2563eb', foreground: '#ffffff' } },
    },
  },
  plugins: [],
} satisfies Config;
```

- [ ] **Step 4: Replace app.css with the token and layout baseline**

```css
/* web/src/app.css */
@import '@fontsource/inter/variable.css';
@tailwind base;
@tailwind components;
@tailwind utilities;

:root {
  font-family: 'Inter Variable', Inter, system-ui, sans-serif;
  color: #18181b;
  background: #f4f4f5;
  --canvas: #f4f4f5;
  --panel: #ffffff;
  --sidebar: #fafafa;
  --border: #e4e4e7;
  --muted: #71717a;
  --accent: #2563eb;
  --accent-soft: #eff6ff;
  --success: #15803d;
  --warning: #a16207;
  --danger: #b91c1c;
}
* { box-sizing: border-box; }
html, body, #app { min-height: 100%; margin: 0; }
body { min-width: 320px; background: var(--canvas); }
button, input { font: inherit; }
a { color: inherit; text-decoration: none; }
:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }

.app-shell { display: grid; grid-template-columns: auto minmax(0, 1fr); min-height: 100vh; }
.app-shell__body { min-width: 0; }
.sidebar { width: 240px; padding: 16px 12px; background: var(--sidebar); border-right: 1px solid var(--border); }
.sidebar[data-collapsed='true'] { width: 64px; }
.sidebar[data-collapsed='true'] .sidebar__label, .sidebar[data-collapsed='true'] .sidebar__context { display: none; }
.sidebar__brand { height: 40px; font-weight: 650; }
.sidebar nav { display: grid; gap: 4px; margin: 12px 0; }
.sidebar nav a, .sidebar__disabled { display: flex; gap: 10px; align-items: center; min-height: 40px; padding: 8px 10px; border-radius: 6px; }
.sidebar nav a[aria-current='page'] { color: var(--accent); background: var(--accent-soft); }
.sidebar__disabled { color: var(--muted); cursor: not-allowed; }
.sidebar__context { display: grid; gap: 4px; padding: 12px 10px; border: 1px solid var(--border); border-radius: 8px; background: var(--panel); }
.topbar { height: 56px; display: flex; align-items: center; border-bottom: 1px solid var(--border); background: var(--panel); padding: 0 24px; }
.topbar__spacer { flex: 1; }
.health-pill { border: 1px solid var(--border); border-radius: 999px; padding: 4px 9px; color: var(--muted); font-size: 12px; }
.content-canvas { width: min(100%, 1440px); margin: 0 auto; padding: 24px; }
.auth-canvas { min-height: 100vh; display: grid; place-items: center; padding: 24px; background: var(--canvas); }
.auth-card { width: min(100%, 420px); padding: 28px; border: 1px solid var(--border); border-radius: 10px; background: var(--panel); }
.auth-card form, .auth-card label { display: grid; gap: 8px; }
.auth-card form { gap: 16px; }
.auth-card input { width: 100%; min-height: 40px; padding: 8px 10px; border: 1px solid var(--border); border-radius: 6px; }
.auth-card button { min-height: 40px; border: 0; border-radius: 6px; color: white; background: var(--accent); }
.auth-card [role='alert'] { color: var(--danger); }

@media (max-width: 767px) {
  .app-shell { display: block; }
  .sidebar { position: fixed; inset: 0 auto 0 0; z-index: 20; width: min(84vw, 280px); transform: translateX(-100%); }
  .sidebar[data-mobile-open='true'] { transform: translateX(0); }
  .content-canvas { padding: 16px; }
  .topbar { padding: 0 16px; }
}
```

- [ ] **Step 5: Set the app document baseline**

```html
<!-- web/src/app.html -->
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="theme-color" content="#f4f4f5" />
    <title>Personal Agent</title>
  </head>
  <body data-sveltekit-preload-data="hover">
    <div id="app"></div>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
```

- [ ] **Step 6: Run the baseline test and production build**

Run: `cd web && npm test -- --run src/styles-baseline.test.ts && npm run build`  
Expected: PASS; Vite production build exits 0 with no missing font or CSS imports.

- [ ] **Step 7: Commit**

```bash
git add web/src/app.css web/src/app.html web/tailwind.config.ts web/src/styles-baseline.test.ts
git commit -m "style(web): establish dashboard tokens and layout"
```


## Phase: 03-global-vault-grids

### Task 20: Pure project context and search helpers

**Files:**
- Create: `web/src/lib/catalog.ts`
- Test: `web/src/lib/catalog.test.ts`

**Interfaces:**
- Consumes: `Project` and `ShellContext` contracts above.
- Produces: `isUnfiled(p)`, `filterProjectsByContext(projects, ctx)`, and `filterByQuery(items, query)` with the exact signatures below.

- [ ] **Step 1: Write the failing unit tests**

```ts
// web/src/lib/catalog.test.ts
import { describe, expect, it } from 'vitest'
import type { Project } from './api/types'
import { filterByQuery, filterProjectsByContext, isUnfiled } from './catalog'

const projects: Project[] = [
  { id: 'p0', name: 'Inbox', note_count: 0 },
  { id: 'p1', name: 'Loose notes', vault_id: '', note_count: 2 },
  { id: 'p2', name: 'Training Plan', vault_id: 'health', vault_name: 'HEALTH', note_count: 4 },
  { id: 'p3', name: 'Budget', vault_id: 'finance', vault_name: 'FINANCE', note_count: 1 },
]

describe('catalog helpers', () => {
  it('treats missing, null, and empty vault IDs as unfiled', () => {
    expect(isUnfiled({})).toBe(true)
    expect(isUnfiled({ vault_id: null })).toBe(true)
    expect(isUnfiled({ vault_id: '' })).toBe(true)
    expect(isUnfiled({ vault_id: 'health' })).toBe(false)
  })

  it('returns only unfiled projects in global context', () => {
    expect(filterProjectsByContext(projects, { kind: 'global' }).map((p) => p.id)).toEqual(['p0', 'p1'])
  })

  it('returns only exact vault matches in vault context', () => {
    expect(filterProjectsByContext(projects, { kind: 'vault', vaultId: 'health', vaultName: 'HEALTH' }).map((p) => p.id)).toEqual(['p2'])
  })

  it('trims and matches names case-insensitively without mutating input', () => {
    const result = filterByQuery(projects, '  PLAN ')
    expect(result.map((p) => p.id)).toEqual(['p2'])
    expect(projects).toHaveLength(4)
    expect(filterByQuery(projects, '   ')).toEqual(projects)
  })
})
```

- [ ] **Step 2: Run the focused test and verify RED**

Run: `cd web && npm test -- --run src/lib/catalog.test.ts`

Expected: FAIL because `./catalog` does not exist.

- [ ] **Step 3: Add the minimal pure implementation**

```ts
// web/src/lib/catalog.ts
import type { Project } from './api/types'
import type { ShellContext } from './stores/shell-context'

export function isUnfiled(p: { vault_id?: string | null }): boolean {
  return !p.vault_id
}

export function filterProjectsByContext(projects: Project[], ctx: ShellContext): Project[] {
  return ctx.kind === 'global'
    ? projects.filter(isUnfiled)
    : projects.filter((project) => project.vault_id === ctx.vaultId)
}

export function filterByQuery<T extends { name: string }>(items: T[], query: string): T[] {
  const normalized = query.trim().toLocaleLowerCase()
  if (!normalized) return items
  return items.filter((item) => item.name.toLocaleLowerCase().includes(normalized))
}
```

- [ ] **Step 4: Run the focused test and typecheck**

Run: `cd web && npm test -- --run src/lib/catalog.test.ts && npm run check`

Expected: PASS; Svelte/TypeScript check exits zero.

- [ ] **Step 5: Commit Task 20**

```bash
git add web/src/lib/catalog.ts web/src/lib/catalog.test.ts
git commit -m "feat(ui): add catalog filtering helpers"
```

---

### Task 21: Catalog UI primitives

**Files:**
- Create: `web/src/components/EmptyState.svelte`
- Create: `web/src/components/Skeleton.svelte`
- Create: `web/src/components/SearchField.svelte`
- Create: `web/src/components/Badge.svelte`
- Create: `web/src/components/ProjectCard.svelte`
- Create: `web/src/components/VaultCard.svelte`
- Test: `web/src/components/catalog-components.test.ts`

**Interfaces:**
- Consumes: `Project`, `Vault`, Svelte callback props, and Tailwind tokens established by shell work.
- Produces: `EmptyState { title, description, actionLabel, onaction }`, `Skeleton { class? }`, bindable `SearchField { value, label? }`, `Badge { text }`, `ProjectCard { project, onclick }`, and `VaultCard { vault, projectCount, onclick }`.

- [ ] **Step 1: Write failing component tests**

```ts
// web/src/components/catalog-components.test.ts
import { fireEvent, render, screen } from '@testing-library/svelte'
import { describe, expect, it, vi } from 'vitest'
import EmptyState from './EmptyState.svelte'
import ProjectCard from './ProjectCard.svelte'
import SearchField from './SearchField.svelte'
import VaultCard from './VaultCard.svelte'

describe('catalog components', () => {
  it('renders an actionable empty state', async () => {
    const onaction = vi.fn()
    render(EmptyState, { title: 'No projects yet', description: 'Create your first project.', actionLabel: 'New project', onaction })
    await fireEvent.click(screen.getByRole('button', { name: 'New project' }))
    expect(onaction).toHaveBeenCalledOnce()
  })

  it('labels and updates search input', async () => {
    render(SearchField, { value: '', label: 'Search vaults' })
    await fireEvent.input(screen.getByRole('searchbox', { name: 'Search vaults' }), { target: { value: 'health' } })
    expect(screen.getByRole<HTMLInputElement>('searchbox').value).toBe('health')
  })

  it('shows vault name and project metrics on a vaulted project', () => {
    render(ProjectCard, { project: { id: 'p1', name: 'Training', vault_id: 'v1', vault_name: 'HEALTH', note_count: 3, session_count: 2, due_count: 1 }, onclick: vi.fn() })
    expect(screen.getByText('HEALTH')).toBeInTheDocument()
    expect(screen.getByText('3 notes')).toBeInTheDocument()
    expect(screen.getByText('2 sessions')).toBeInTheDocument()
    expect(screen.getByText('1 due')).toBeInTheDocument()
  })

  it('does not invent a badge for an unfiled project', () => {
    render(ProjectCard, { project: { id: 'p0', name: 'Inbox', vault_id: null, note_count: 0 }, onclick: vi.fn() })
    expect(screen.queryByText('Unfiled')).not.toBeInTheDocument()
  })

  it('renders a vault card as a named button with project count', async () => {
    const onclick = vi.fn()
    render(VaultCard, { vault: { id: 'v1', name: 'HEALTH', created_at: '', updated_at: '' }, projectCount: 4, onclick })
    await fireEvent.click(screen.getByRole('button', { name: /enter health vault/i }))
    expect(screen.getByText('4 projects')).toBeInTheDocument()
    expect(onclick).toHaveBeenCalledOnce()
  })
})
```

- [ ] **Step 2: Run the component test and verify RED**

Run: `cd web && npm test -- --run src/components/catalog-components.test.ts`

Expected: FAIL because the six components do not exist.

- [ ] **Step 3: Implement the primitives**

```svelte
<!-- web/src/components/EmptyState.svelte -->
<script lang="ts">
  let { title, description, actionLabel, onaction }: { title: string; description: string; actionLabel: string; onaction: () => void } = $props()
</script>
<section class="rounded-xl border border-dashed border-slate-300 bg-white px-6 py-12 text-center">
  <h2 class="text-base font-semibold text-slate-950">{title}</h2>
  <p class="mx-auto mt-2 max-w-md text-sm text-slate-600">{description}</p>
  <button class="mt-5 rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:ring-offset-2" type="button" onclick={onaction}>{actionLabel}</button>
</section>
```

```svelte
<!-- web/src/components/Skeleton.svelte -->
<script lang="ts">
  let { class: className = '' }: { class?: string } = $props()
</script>
<div aria-hidden="true" class={`animate-pulse rounded-lg bg-slate-200 ${className}`}></div>
```

```svelte
<!-- web/src/components/SearchField.svelte -->
<script lang="ts">
  let { value = $bindable(''), label = 'Search' }: { value?: string; label?: string } = $props()
</script>
<label class="relative block w-full sm:max-w-xs">
  <span class="sr-only">{label}</span>
  <input type="search" {value} oninput={(event) => value = event.currentTarget.value} aria-label={label} placeholder={label} class="w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200" />
</label>
```

```svelte
<!-- web/src/components/Badge.svelte -->
<script lang="ts">let { text }: { text: string } = $props()</script>
<span class="inline-flex rounded-full bg-indigo-50 px-2 py-0.5 text-xs font-medium text-indigo-700">{text}</span>
```

```svelte
<!-- web/src/components/ProjectCard.svelte -->
<script lang="ts">
  import type { Project } from '../lib/api/types'
  import Badge from './Badge.svelte'
  let { project, onclick }: { project: Project; onclick: () => void } = $props()
</script>
<button type="button" {onclick} aria-label={`Open ${project.name}`} class="w-full rounded-xl border border-slate-200 bg-white p-5 text-left hover:border-slate-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500">
  <div class="flex items-start justify-between gap-3"><h2 class="font-semibold text-slate-950">{project.name}</h2>{#if project.vault_id && project.vault_name}<Badge text={project.vault_name} />{/if}</div>
  <div class="mt-5 flex flex-wrap gap-3 text-xs text-slate-500">
    <span>{project.note_count} {project.note_count === 1 ? 'note' : 'notes'}</span>
    {#if project.session_count !== undefined}<span>{project.session_count} {project.session_count === 1 ? 'session' : 'sessions'}</span>{/if}
    {#if project.due_count !== undefined}<span>{project.due_count} due</span>{/if}
  </div>
</button>
```

```svelte
<!-- web/src/components/VaultCard.svelte -->
<script lang="ts">
  import type { Vault } from '../lib/api/types'
  let { vault, projectCount, onclick }: { vault: Vault; projectCount: number; onclick: () => void } = $props()
</script>
<button type="button" {onclick} aria-label={`Enter ${vault.name} vault`} class="w-full rounded-xl border border-slate-200 bg-white p-5 text-left hover:border-slate-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500">
  <h2 class="font-semibold text-slate-950">{vault.name}</h2>
  <p class="mt-5 text-xs text-slate-500">{projectCount} {projectCount === 1 ? 'project' : 'projects'}</p>
</button>
```

- [ ] **Step 4: Run tests and checks**

Run: `cd web && npm test -- --run src/components/catalog-components.test.ts && npm run check`

Expected: PASS with no accessibility or TypeScript diagnostics.

- [ ] **Step 5: Commit Task 21**

```bash
git add web/src/components
git commit -m "feat(ui): add catalog card primitives"
```

---

### Task 22: Global Home dashboard

**Files:**
- Create: `web/src/routes/HomePage.svelte`
- Test: `web/src/routes/HomePage.test.ts`

**Interfaces:**
- Consumes: `api.get<HomeResponse>('/api/v1/home')`, `isUnfiled`, `ProjectCard`, `EmptyState`, `Skeleton`, and `navigate`.
- Produces: a route component with no props; quick actions navigate to `#/projects` and `#/vaults`, and recent cards navigate to `#/projects/:id`.

- [ ] **Step 1: Write the failing route test**

```ts
// web/src/routes/HomePage.test.ts
import { render, screen, waitFor } from '@testing-library/svelte'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import HomePage from './HomePage.svelte'
import { api } from '../lib/api/client'

vi.mock('../lib/api/client', () => ({ api: { get: vi.fn() } }))

describe('HomePage', () => {
  beforeEach(() => vi.mocked(api.get).mockReset())

  it('shows due summary and only recent unfiled projects', async () => {
    vi.mocked(api.get).mockResolvedValue({ generated_at: '2026-08-19T00:00:00Z', due_count: 3, projects: [
      { id: 'loose', name: 'Inbox', note_count: 1 },
      { id: 'vaulted', name: 'Training', vault_id: 'health', vault_name: 'HEALTH', note_count: 2 },
    ] })
    render(HomePage)
    expect(await screen.findByText('3 items due')).toBeInTheDocument()
    expect(screen.getByText('Inbox')).toBeInTheDocument()
    expect(screen.queryByText('Training')).not.toBeInTheDocument()
  })

  it('is friendly when no projects or reviews are due', async () => {
    vi.mocked(api.get).mockResolvedValue({ generated_at: '2026-08-19T00:00:00Z', projects: [] })
    render(HomePage)
    await waitFor(() => expect(screen.getByText('You’re all caught up')).toBeInTheDocument())
    expect(screen.getByText('No unfiled projects yet')).toBeInTheDocument()
  })
})
```

- [ ] **Step 2: Run the test and verify RED**

Run: `cd web && npm test -- --run src/routes/HomePage.test.ts`

Expected: FAIL because `HomePage.svelte` does not exist.

- [ ] **Step 3: Implement the dashboard**

```svelte
<!-- web/src/routes/HomePage.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import EmptyState from '../components/EmptyState.svelte'
  import ProjectCard from '../components/ProjectCard.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api } from '../lib/api/client'
  import type { HomeResponse, Project } from '../lib/api/types'
  import { isUnfiled } from '../lib/catalog'
  import { navigate } from '../lib/router'
  let loading = $state(true), error = $state(''), dueCount = $state(0), projects = $state<Project[]>([])
  onMount(async () => {
    try { const data = await api.get<HomeResponse>('/api/v1/home'); dueCount = data.due_count ?? 0; projects = data.projects.filter(isUnfiled).slice(0, 6) }
    catch (cause) { error = cause instanceof Error ? cause.message : 'Could not load your dashboard.' }
    finally { loading = false }
  })
</script>
<svelte:head><title>Home · Personal Agent</title></svelte:head>
<div class="space-y-8">
  <header><p class="text-sm text-slate-500">Global desk</p><h1 class="text-2xl font-semibold text-slate-950">Home</h1></header>
  <section aria-label="Quick actions" class="flex flex-wrap gap-3">
    <button class="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white" onclick={() => navigate('#/projects')}>New project</button>
    <button class="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-medium" onclick={() => navigate('#/vaults')}>New vault</button>
  </section>
  {#if error}<p role="alert" class="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</p>{/if}
  {#if loading}
    <div class="grid gap-4 md:grid-cols-3"><Skeleton class="h-28" /><Skeleton class="h-28" /><Skeleton class="h-28" /></div>
  {:else}
    <section class="grid gap-4 md:grid-cols-2">
      <div class="rounded-xl border border-slate-200 bg-white p-5"><p class="text-sm text-slate-500">Review</p><p class="mt-2 text-xl font-semibold">{dueCount ? `${dueCount} ${dueCount === 1 ? 'item' : 'items'} due` : 'You’re all caught up'}</p></div>
      <div class="rounded-xl border border-slate-200 bg-white p-5"><p class="text-sm text-slate-500">Unfiled projects</p><p class="mt-2 text-xl font-semibold">{projects.length}</p></div>
    </section>
    <section class="space-y-4"><div class="flex items-center justify-between"><h2 class="text-lg font-semibold">Recent projects</h2><button class="text-sm font-medium text-indigo-700" onclick={() => navigate('#/projects')}>View all</button></div>
      {#if projects.length}<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">{#each projects as project (project.id)}<ProjectCard {project} onclick={() => navigate(`#/projects/${encodeURIComponent(project.id)}`)} />{/each}</div>
      {:else}<EmptyState title="No unfiled projects yet" description="Create a project on your global desk, or organize work inside a vault." actionLabel="New project" onaction={() => navigate('#/projects')} />{/if}
    </section>
  {/if}
</div>
```

- [ ] **Step 4: Run focused tests and checks**

Run: `cd web && npm test -- --run src/routes/HomePage.test.ts && npm run check`

Expected: PASS.

- [ ] **Step 5: Commit Task 22**

```bash
git add web/src/routes/HomePage.svelte web/src/routes/HomePage.test.ts
git commit -m "feat(ui): add global home dashboard"
```

---

### Task 23: Global unfiled Projects grid

**Files:**
- Create: `web/src/routes/ProjectsPage.svelte`
- Test: `web/src/routes/ProjectsPage.test.ts`

**Interfaces:**
- Consumes: `/api/v1/home`, `POST /api/v1/projects` with `{ name, vault_id: null }`, catalog helpers/primitives, and `navigate`.
- Produces: global projects route with client search and an inline create form; successful creation navigates to `#/projects/:id`.

- [ ] **Step 1: Write failing behavior tests**

```ts
// web/src/routes/ProjectsPage.test.ts
import { fireEvent, render, screen } from '@testing-library/svelte'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ProjectsPage from './ProjectsPage.svelte'
import { api } from '../lib/api/client'
vi.mock('../lib/api/client', () => ({ api: { get: vi.fn(), post: vi.fn() } }))

describe('ProjectsPage', () => {
  beforeEach(() => vi.clearAllMocks())
  it('shows only searched unfiled projects', async () => {
    vi.mocked(api.get).mockResolvedValue({ generated_at: '', projects: [
      { id: 'a', name: 'Alpha', note_count: 0 }, { id: 'b', name: 'Beta', vault_id: 'v1', vault_name: 'WORK', note_count: 0 },
    ] })
    render(ProjectsPage)
    expect(await screen.findByText('Alpha')).toBeInTheDocument()
    expect(screen.queryByText('Beta')).not.toBeInTheDocument()
    await fireEvent.input(screen.getByRole('searchbox'), { target: { value: 'none' } })
    expect(screen.getByText('No matching projects')).toBeInTheDocument()
  })
  it('creates an unfiled project', async () => {
    vi.mocked(api.get).mockResolvedValue({ generated_at: '', projects: [] })
    vi.mocked(api.post).mockResolvedValue({ id: 'new', name: 'Inbox', vault_id: null, note_count: 0 })
    render(ProjectsPage)
    await fireEvent.click(await screen.findByRole('button', { name: 'New project' }))
    await fireEvent.input(screen.getByLabelText('Project name'), { target: { value: 'Inbox' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Create project' }))
    expect(api.post).toHaveBeenCalledWith('/api/v1/projects', { name: 'Inbox', vault_id: null })
  })
})
```

- [ ] **Step 2: Run the test and verify RED**

Run: `cd web && npm test -- --run src/routes/ProjectsPage.test.ts`

Expected: FAIL because the route component is absent.

- [ ] **Step 3: Implement the grid, search, empty states, and create form**

```svelte
<!-- web/src/routes/ProjectsPage.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'; import EmptyState from '../components/EmptyState.svelte'; import ProjectCard from '../components/ProjectCard.svelte'; import SearchField from '../components/SearchField.svelte'; import Skeleton from '../components/Skeleton.svelte'
  import { api } from '../lib/api/client'; import type { HomeResponse, Project } from '../lib/api/types'; import { filterByQuery, isUnfiled } from '../lib/catalog'; import { navigate } from '../lib/router'
  let projects = $state<Project[]>([]), query = $state(''), loading = $state(true), creating = $state(false), saving = $state(false), name = $state(''), error = $state('')
  let visible = $derived(filterByQuery(projects, query))
  onMount(async () => { try { projects = (await api.get<HomeResponse>('/api/v1/home')).projects.filter(isUnfiled) } catch (e) { error = e instanceof Error ? e.message : 'Could not load projects.' } finally { loading = false } })
  async function createProject() { const clean = name.trim(); if (!clean) return; saving = true; error = ''; try { const project = await api.post<Project>('/api/v1/projects', { name: clean, vault_id: null }); navigate(`#/projects/${encodeURIComponent(project.id)}`) } catch (e) { error = e instanceof Error ? e.message : 'Could not create project.' } finally { saving = false } }
</script>
<svelte:head><title>Projects · Personal Agent</title></svelte:head>
<div class="space-y-6"><header class="flex flex-wrap items-end justify-between gap-4"><div><p class="text-sm text-slate-500">Global desk</p><h1 class="text-2xl font-semibold">Projects</h1></div><button class="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white" onclick={() => creating = true}>New project</button></header>
  <SearchField bind:value={query} label="Search projects" />
  {#if error}<p role="alert" class="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</p>{/if}
  {#if creating}<form class="flex max-w-lg gap-2 rounded-xl border bg-white p-4" onsubmit={(e) => { e.preventDefault(); createProject() }}><label class="flex-1"><span class="text-sm font-medium">Project name</span><input class="mt-1 w-full rounded-md border px-3 py-2" bind:value={name} /></label><button disabled={saving || !name.trim()} class="self-end rounded-md bg-indigo-600 px-4 py-2 text-sm text-white">Create project</button></form>{/if}
  {#if loading}<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3"><Skeleton class="h-32" /><Skeleton class="h-32" /></div>
  {:else if visible.length}<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">{#each visible as project (project.id)}<ProjectCard {project} onclick={() => navigate(`#/projects/${encodeURIComponent(project.id)}`)} />{/each}</div>
  {:else if query.trim()}<EmptyState title="No matching projects" description="Try a different project name." actionLabel="Clear search" onaction={() => query = ''} />
  {:else}<EmptyState title="No unfiled projects yet" description="Create your first project on the global desk." actionLabel="New project" onaction={() => creating = true} />{/if}
</div>
```

- [ ] **Step 4: Run tests and checks**

Run: `cd web && npm test -- --run src/routes/ProjectsPage.test.ts && npm run check`

Expected: PASS.

- [ ] **Step 5: Commit Task 23**

```bash
git add web/src/routes/ProjectsPage.svelte web/src/routes/ProjectsPage.test.ts
git commit -m "feat(ui): add unfiled projects grid"
```

---

### Task 24: Vaults grid, creation, and enter navigation

**Files:**
- Create: `web/src/routes/VaultsPage.svelte`
- Test: `web/src/routes/VaultsPage.test.ts`

**Interfaces:**
- Consumes: `GET/POST /api/v1/vaults`, `/api/v1/home` for project counts, catalog primitives, and `navigate`.
- Produces: vault catalog route; card click and successful create both navigate to `#/vaults/:vaultId`.

- [ ] **Step 1: Write failing route tests**

```ts
// web/src/routes/VaultsPage.test.ts
import { fireEvent, render, screen } from '@testing-library/svelte'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import VaultsPage from './VaultsPage.svelte'; import { api } from '../lib/api/client'; import { navigate } from '../lib/router'
vi.mock('../lib/api/client', () => ({ api: { get: vi.fn(), post: vi.fn() } })); vi.mock('../lib/router', () => ({ navigate: vi.fn() }))
describe('VaultsPage', () => {
  beforeEach(() => vi.clearAllMocks())
  it('searches vaults, shows project count, and enters a vault', async () => {
    vi.mocked(api.get).mockImplementation(async (path) => path === '/api/v1/vaults' ? [{ id: 'v1', name: 'HEALTH', created_at: '', updated_at: '' }] : { projects: [{ id: 'p1', name: 'Training', vault_id: 'v1', note_count: 0 }], generated_at: '' })
    render(VaultsPage); expect(await screen.findByText('1 project')).toBeInTheDocument()
    await fireEvent.click(screen.getByRole('button', { name: /enter health vault/i })); expect(navigate).toHaveBeenCalledWith('#/vaults/v1')
  })
  it('creates and enters a vault', async () => {
    vi.mocked(api.get).mockResolvedValueOnce([]).mockResolvedValueOnce({ projects: [], generated_at: '' })
    vi.mocked(api.post).mockResolvedValue({ id: 'v2', name: 'WORK', created_at: '', updated_at: '' })
    render(VaultsPage); await fireEvent.click(await screen.findByRole('button', { name: 'New vault' })); await fireEvent.input(screen.getByLabelText('Vault name'), { target: { value: 'WORK' } }); await fireEvent.click(screen.getByRole('button', { name: 'Create vault' }))
    expect(api.post).toHaveBeenCalledWith('/api/v1/vaults', { name: 'WORK' }); expect(navigate).toHaveBeenCalledWith('#/vaults/v2')
  })
})
```

- [ ] **Step 2: Run the test and verify RED**

Run: `cd web && npm test -- --run src/routes/VaultsPage.test.ts`

Expected: FAIL because `VaultsPage.svelte` is absent.

- [ ] **Step 3: Implement vault loading, search, counts, create, and enter**

```svelte
<!-- web/src/routes/VaultsPage.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'; import EmptyState from '../components/EmptyState.svelte'; import SearchField from '../components/SearchField.svelte'; import Skeleton from '../components/Skeleton.svelte'; import VaultCard from '../components/VaultCard.svelte'
  import { api } from '../lib/api/client'; import type { HomeResponse, Vault } from '../lib/api/types'; import { filterByQuery } from '../lib/catalog'; import { navigate } from '../lib/router'
  let vaults = $state<Vault[]>([]), counts = $state<Record<string, number>>({}), query = $state(''), loading = $state(true), creating = $state(false), saving = $state(false), name = $state(''), error = $state('')
  let visible = $derived(filterByQuery(vaults, query))
  onMount(async () => { try { const [listed, home] = await Promise.all([api.get<Vault[]>('/api/v1/vaults'), api.get<HomeResponse>('/api/v1/home')]); vaults = listed; counts = home.projects.reduce<Record<string, number>>((all, p) => { if (p.vault_id) all[p.vault_id] = (all[p.vault_id] ?? 0) + 1; return all }, {}) } catch (e) { error = e instanceof Error ? e.message : 'Could not load vaults.' } finally { loading = false } })
  async function createVault() { const clean = name.trim(); if (!clean) return; saving = true; error = ''; try { const vault = await api.post<Vault>('/api/v1/vaults', { name: clean }); navigate(`#/vaults/${encodeURIComponent(vault.id)}`) } catch (e) { error = e instanceof Error ? e.message : 'Could not create vault.' } finally { saving = false } }
</script>
<svelte:head><title>Vaults · Personal Agent</title></svelte:head>
<div class="space-y-6"><header class="flex flex-wrap items-end justify-between gap-4"><div><p class="text-sm text-slate-500">Global desk</p><h1 class="text-2xl font-semibold">Vaults</h1></div><button class="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white" onclick={() => creating = true}>New vault</button></header>
  <SearchField bind:value={query} label="Search vaults" />{#if error}<p role="alert" class="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</p>{/if}
  {#if creating}<form class="flex max-w-lg gap-2 rounded-xl border bg-white p-4" onsubmit={(e) => { e.preventDefault(); createVault() }}><label class="flex-1"><span class="text-sm font-medium">Vault name</span><input class="mt-1 w-full rounded-md border px-3 py-2" bind:value={name} /></label><button disabled={saving || !name.trim()} class="self-end rounded-md bg-indigo-600 px-4 py-2 text-sm text-white">Create vault</button></form>{/if}
  {#if loading}<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3"><Skeleton class="h-28" /><Skeleton class="h-28" /></div>
  {:else if visible.length}<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">{#each visible as vault (vault.id)}<VaultCard {vault} projectCount={counts[vault.id] ?? 0} onclick={() => navigate(`#/vaults/${encodeURIComponent(vault.id)}`)} />{/each}</div>
  {:else if query.trim()}<EmptyState title="No matching vaults" description="Try a different vault name." actionLabel="Clear search" onaction={() => query = ''} />
  {:else}<EmptyState title="No vaults yet" description="Create a vault to organize related projects." actionLabel="New vault" onaction={() => creating = true} />{/if}
</div>
```

- [ ] **Step 4: Run tests and checks**

Run: `cd web && npm test -- --run src/routes/VaultsPage.test.ts && npm run check`

Expected: PASS.

- [ ] **Step 5: Commit Task 24**

```bash
git add web/src/routes/VaultsPage.svelte web/src/routes/VaultsPage.test.ts
git commit -m "feat(ui): add searchable vaults grid"
```

---

### Task 25: Register global routes in the app router

**Files:**
- Modify: `web/src/lib/router.ts`
- Modify: `web/src/App.svelte`
- Test: `web/src/App.test.ts`

**Interfaces:**
- Consumes: existing `parseHash()`/route union and authenticated shell from the shell/router task, plus `HomePage`, `ProjectsPage`, and `VaultsPage` from Tasks 22–24.
- Produces: exact matches for `#/home`, `#/projects`, and `#/vaults`; unknown hashes retain the router’s existing fallback behavior. Parameterized `#/vaults/:vaultId` remains owned by the vault-context task.

- [ ] **Step 1: Extend the router integration test first**

```ts
// Add to web/src/App.test.ts
import { render, screen } from '@testing-library/svelte'
import { afterEach, describe, expect, it } from 'vitest'
import App from './App.svelte'

describe('global catalog routes', () => {
  afterEach(() => { window.location.hash = '' })
  for (const [hash, heading] of [['#/home', 'Home'], ['#/projects', 'Projects'], ['#/vaults', 'Vaults']] as const) {
    it(`renders ${hash}`, () => {
      window.location.hash = hash
      render(App)
      expect(screen.getByRole('heading', { level: 1, name: heading })).toBeInTheDocument()
    })
  }
})
```

- [ ] **Step 2: Run the integration test and verify RED**

Run: `cd web && npm test -- --run src/App.test.ts`

Expected: FAIL because at least one global route does not render its new page.

- [ ] **Step 3: Add route variants and page selection**

Add these exact variants to the existing router union and exact-hash branch in `web/src/lib/router.ts` (preserve all routes implemented by other tasks):

```ts
export type GlobalRoute =
  | { name: 'home' }
  | { name: 'projects' }
  | { name: 'vaults' }

export function parseGlobalHash(hash: string): GlobalRoute | null {
  const path = hash.replace(/^#/, '').replace(/\/$/, '') || '/home'
  if (path === '/home') return { name: 'home' }
  if (path === '/projects') return { name: 'projects' }
  if (path === '/vaults') return { name: 'vaults' }
  return null
}
```

Call `parseGlobalHash(hash)` from the existing `parseHash` before its unknown-route fallback, but after more-specific project/vault parameter routes. Then add imports and these branches to the existing authenticated content switch in `web/src/App.svelte` without replacing the shell:

```svelte
<script lang="ts">
  import HomePage from './routes/HomePage.svelte'
  import ProjectsPage from './routes/ProjectsPage.svelte'
  import VaultsPage from './routes/VaultsPage.svelte'
  // Keep the shell, auth bootstrap, current route state, and other page imports already present.
</script>

{#if route.name === 'home'}
  <HomePage />
{:else if route.name === 'projects'}
  <ProjectsPage />
{:else if route.name === 'vaults'}
  <VaultsPage />
{:else}
  <!-- Keep the existing branches for every other route here. -->
{/if}
```

The comment in the snippet marks existing code that must remain; do not paste it as a runtime fallback or delete other route branches.

- [ ] **Step 4: Run the focused suite, complete frontend suite, and check**

Run: `cd web && npm test -- --run src/App.test.ts src/routes/HomePage.test.ts src/routes/ProjectsPage.test.ts src/routes/VaultsPage.test.ts && npm test -- --run && npm run check`

Expected: all tests PASS and check exits zero.

- [ ] **Step 5: Commit Task 25**

```bash
git add web/src/lib/router.ts web/src/App.svelte web/src/App.test.ts
git commit -m "feat(ui): wire global dashboard routes"
```


## Phase: 04-vault-context

### Task 30: Resolve URL-Driven Vault Shell Context

**Files:**
- Create: `web/src/lib/stores/shell-context.ts`
- Create: `web/src/lib/stores/shell-context.test.ts`
- Modify: `web/src/App.svelte`
- Modify: `web/src/shell/Sidebar.svelte`
- Test: `web/src/shell/Sidebar.test.ts`

**Interfaces:**
- Consumes: `Route` from `web/src/lib/router.ts`, `Project`/`Vault` and `api.getProject(id)`/`api.listVaults()` from `web/src/lib/api`.
- Produces: `type ShellContext = { kind: 'global' } | { kind: 'vault'; vault: Vault }` and `resolveShellContext(route, deps): Promise<ShellContext>`.

- [ ] **Step 1: Write failing context and sidebar tests**

```ts
it.each(['vault-home', 'vault-projects', 'vault-sessions', 'vault-review'])('uses route vault for %s', async name => {
  const context = await resolveShellContext({ name, vaultId: 'v1' } as Route, deps)
  expect(context).toEqual({ kind: 'vault', vault: { id: 'v1', name: 'HEALTH' } })
})

it('derives a project deep-link context from project.vault_id', async () => {
  deps.getProject.mockResolvedValue({ id: 'p1', name: 'Sleep', vault_id: 'v1', vault_name: 'HEALTH' })
  expect(await resolveShellContext({ name: 'project-notes', projectId: 'p1' }, deps))
    .toEqual({ kind: 'vault', vault: { id: 'v1', name: 'HEALTH' } })
})

it('renders replacement vault navigation and leaves to global home', async () => {
  render(Sidebar, { context: { kind: 'vault', vault: { id: 'v1', name: 'HEALTH' } }, route })
  expect(screen.getByText('HEALTH')).toBeVisible()
  expect(screen.queryByText('Vaults')).not.toBeInTheDocument()
  expect(screen.getByRole('link', { name: /leave vault/i })).toHaveAttribute('href', '#/home')
})
```

- [ ] **Step 2: Run tests and confirm the new module/behavior is absent**

Run: `rtk npm run test -- --run web/src/lib/stores/shell-context.test.ts web/src/shell/Sidebar.test.ts`
Expected: FAIL because `resolveShellContext` and vault-mode navigation do not exist.

- [ ] **Step 3: Implement resolution and replacement navigation**

```ts
export async function resolveShellContext(route: Route, deps: ContextDeps): Promise<ShellContext> {
  if ('vaultId' in route) {
    const vault = (await deps.listVaults()).find(candidate => candidate.id === route.vaultId)
    if (!vault) throw new Error('Vault not found')
    return { kind: 'vault', vault }
  }
  if ('projectId' in route) {
    const project = await deps.getProject(route.projectId)
    if (project.vault_id) return { kind: 'vault', vault: { id: project.vault_id, name: project.vault_name } }
  }
  return { kind: 'global' }
}
```

In `App.svelte`, resolve context after every parsed route, discard stale async resolutions with a generation counter, show the existing shell skeleton while resolving, and pass the result to `Sidebar`. In vault mode render only Home (`#/vaults/{id}`), Projects, Sessions, Review, Settings, plus `Leave vault` → `#/home`; close the mobile drawer after navigation.

- [ ] **Step 4: Run focused tests**

Run: `rtk npm run test -- --run web/src/lib/stores/shell-context.test.ts web/src/shell/Sidebar.test.ts`
Expected: PASS, including vaulted and unfiled project deep links.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/lib/stores/shell-context.ts web/src/lib/stores/shell-context.test.ts web/src/App.svelte web/src/shell/Sidebar.svelte web/src/shell/Sidebar.test.ts
rtk git commit -m "feat(web): derive vault shell context from routes"
```

### Task 31: Build the Vault Home Dashboard

**Files:**
- Create: `web/src/routes/VaultHomePage.svelte`
- Create: `web/src/routes/VaultHomePage.test.ts`
- Modify: `web/src/lib/router.ts`
- Modify: `web/src/App.svelte`

**Interfaces:**
- Consumes: `vaultId: string`, `api.listProjects()`, and `api.getReviewQueue('all')`.
- Produces: route `{ name: 'vault-home'; vaultId: string }` for `#/vaults/:vaultId` and dashboard links to vault projects/sessions/review.

- [ ] **Step 1: Write failing dashboard test**

```ts
it('shows only vault summary data and useful actions', async () => {
  api.listProjects.mockResolvedValue([vaultProject, unfiledProject])
  api.getReviewQueue.mockResolvedValue({ items: [{ id: 'r1', project_id: 'p-v' }], caught_up: false })
  render(VaultHomePage, { vaultId: 'v1' })
  expect(await screen.findByText('1 project')).toBeVisible()
  expect(screen.getByText('1 due')).toBeVisible()
  expect(screen.queryByText(unfiledProject.name)).not.toBeInTheDocument()
  expect(screen.getByRole('link', { name: /new project/i })).toHaveAttribute('href', '#/vaults/v1/projects?new=1')
})
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/routes/VaultHomePage.test.ts`
Expected: FAIL because the route and page are missing.

- [ ] **Step 3: Implement the scoped dashboard**

Load projects and queue concurrently, calculate project/session/note/due totals from only matching projects/items, and render quick actions, recent project cards, skeletons, a no-project empty state, and an inline retry alert. Do not render the full catalog.

```ts
const [allProjects, queue] = await Promise.all([api.listProjects(), api.getReviewQueue('all')])
projects = allProjects.filter(project => project.vault_id === vaultId)
const ids = new Set(projects.map(project => project.id))
due = queue.items.filter(item => ids.has(item.project_id)).length
```

- [ ] **Step 4: Verify green**

Run: `rtk npm run test -- --run web/src/routes/VaultHomePage.test.ts web/src/lib/router.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/routes/VaultHomePage.svelte web/src/routes/VaultHomePage.test.ts web/src/lib/router.ts web/src/App.svelte
rtk git commit -m "feat(web): add vault dashboard"
```

### Task 32: Add Vault Projects and Locked Project Creation

**Files:**
- Create: `web/src/lib/vault-scope.ts`
- Create: `web/src/lib/vault-scope.test.ts`
- Create: `web/src/routes/VaultProjectsPage.svelte`
- Create: `web/src/routes/VaultProjectsPage.test.ts`
- Modify: `web/src/lib/router.ts`
- Modify: `web/src/App.svelte`

**Interfaces:**
- Produces: `filterVaultProjects(projects: Project[], vaultId: string): Project[]` and `createVaultProjectInput(name: string, vaultId: string): { name: string; vault_id: string }`.
- Consumes: shared `ProjectGrid`, `ProjectCreateDialog`, and `api.createProject` established by the global projects plan.

- [ ] **Step 1: Write failing helper and component tests**

```ts
expect(filterVaultProjects([vaultProject, otherVaultProject, unfiledProject], 'v1')).toEqual([vaultProject])
expect(createVaultProjectInput(' Sleep ', 'v1')).toEqual({ name: 'Sleep', vault_id: 'v1' })

it('locks the vault and submits it even when the dialog is reopened', async () => {
  render(VaultProjectsPage, { vaultId: 'v1', vaultName: 'HEALTH' })
  await user.click(await screen.findByRole('button', { name: /new project/i }))
  expect(screen.getByLabelText('Vault')).toHaveValue('HEALTH')
  expect(screen.getByLabelText('Vault')).toBeDisabled()
  await user.type(screen.getByLabelText('Project name'), 'Sleep')
  await user.click(screen.getByRole('button', { name: 'Create project' }))
  expect(api.createProject).toHaveBeenCalledWith({ name: 'Sleep', vault_id: 'v1' })
})
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/lib/vault-scope.test.ts web/src/routes/VaultProjectsPage.test.ts`
Expected: FAIL because vault filtering and locked creation are absent.

- [ ] **Step 3: Implement grid, search, empty state, and locked create**

Use `#/vaults/:vaultId/projects`, filter before client-side name search, preserve the vault ID in component state as an immutable prop (never derive it from form input), and navigate a created project to `#/projects/{encodedId}`. Show no fake “Unfiled” badge.

```ts
export const filterVaultProjects = (projects: Project[], vaultId: string) =>
  projects.filter(project => project.vault_id === vaultId)

export function createVaultProjectInput(name: string, vaultId: string) {
  return { name: name.trim(), vault_id: vaultId }
}
```

- [ ] **Step 4: Verify green**

Run: `rtk npm run test -- --run web/src/lib/vault-scope.test.ts web/src/routes/VaultProjectsPage.test.ts web/src/lib/router.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/lib/vault-scope.ts web/src/lib/vault-scope.test.ts web/src/routes/VaultProjectsPage.svelte web/src/routes/VaultProjectsPage.test.ts web/src/lib/router.ts web/src/App.svelte
rtk git commit -m "feat(web): add locked vault project creation"
```

### Task 33: Aggregate Vault Sessions Through Project Endpoints

**Files:**
- Create: `web/src/lib/vault-sessions.ts`
- Create: `web/src/lib/vault-sessions.test.ts`
- Create: `web/src/routes/VaultSessionsPage.svelte`
- Create: `web/src/routes/VaultSessionsPage.test.ts`
- Modify: `web/src/lib/api/types.ts`
- Modify: `web/src/lib/api/index.ts`
- Modify: `web/src/lib/router.ts`
- Modify: `web/src/App.svelte`

**Interfaces:**
- Produces: `type VaultSession = Session & { project_id: string; project_name: string }` and `loadVaultSessions(vaultId, api): Promise<{ projects: Project[]; sessions: VaultSession[]; failures: string[] }>`.
- Consumes only `listProjects()` and `listProjectSessions(projectId)` (`GET /api/v1/projects/{id}/sessions`).

- [ ] **Step 1: Write failing aggregation tests**

```ts
it('calls sessions once per vault project and annotates results', async () => {
  api.listProjects.mockResolvedValue([projectA, projectB, unfiledProject])
  api.listProjectSessions.mockImplementation(async id => id === 'a' ? [sessionA] : [sessionB])
  const result = await loadVaultSessions('v1', api)
  expect(api.listProjectSessions.mock.calls.map(([id]) => id).sort()).toEqual(['a', 'b'])
  expect(result.sessions).toEqual([
    { ...sessionA, project_id: 'a', project_name: projectA.name },
    { ...sessionB, project_id: 'b', project_name: projectB.name },
  ])
})

it('keeps successful projects and reports a partial failure', async () => {
  api.listProjectSessions.mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce([sessionB])
  expect((await loadVaultSessions('v1', api)).failures).toEqual([projectA.name])
})
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/lib/vault-sessions.test.ts web/src/routes/VaultSessionsPage.test.ts`
Expected: FAIL because no aggregate loader/page exists.

- [ ] **Step 3: Implement bounded client-side aggregation and picker**

Use `Promise.allSettled` over vault projects. Render a project picker plus a combined, project-labelled session list; selecting “New session” navigates to that project's `#/projects/:id/sessions`. If there are no vault projects, show “Create a project first”; if projects have no sessions, show a session empty state. A partial request failure must retain successful rows and show an inline warning listing failed project names.

```ts
const settled = await Promise.allSettled(projects.map(project => api.listProjectSessions(project.id)))
settled.forEach((result, index) => {
  const project = projects[index]
  if (result.status === 'rejected') failures.push(project.name)
  else sessions.push(...result.value.map(session => ({ ...session, project_id: project.id, project_name: project.name })))
})
```

- [ ] **Step 4: Verify green and endpoint discipline**

Run: `rtk npm run test -- --run web/src/lib/vault-sessions.test.ts web/src/routes/VaultSessionsPage.test.ts`
Expected: PASS; request assertions contain no `/vaults/{id}/sessions` API call.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/lib/vault-sessions.ts web/src/lib/vault-sessions.test.ts web/src/routes/VaultSessionsPage.svelte web/src/routes/VaultSessionsPage.test.ts web/src/lib/api/types.ts web/src/lib/api/index.ts web/src/lib/router.ts web/src/App.svelte
rtk git commit -m "feat(web): aggregate sessions across vault projects"
```

### Task 34: Filter the Vault Review Queue Client-Side

**Files:**
- Create: `web/src/lib/review/vault-filter.ts`
- Create: `web/src/lib/review/vault-filter.test.ts`
- Create: `web/src/routes/VaultReviewPage.svelte`
- Create: `web/src/routes/VaultReviewPage.test.ts`
- Modify: `web/src/lib/router.ts`
- Modify: `web/src/App.svelte`

**Interfaces:**
- Produces: `filterQueueByProjectIds(queue: ReviewQueue, projectIds: ReadonlySet<string>): ReviewQueue`.
- Consumes: `api.getReviewQueue('all')`, vault projects, and the shared `ReviewRunner` from Task 50.

- [ ] **Step 1: Write failing filter/page tests**

```ts
expect(filterQueueByProjectIds(
  { items: [vaultItem, otherItem], caught_up: false }, new Set(['p-v'])
)).toEqual({ items: [vaultItem], caught_up: false })

it('requests all and passes only vault items to the runner', async () => {
  render(VaultReviewPage, { vaultId: 'v1' })
  expect(api.getReviewQueue).toHaveBeenCalledWith('all')
  expect(await screen.findByText(vaultItem.prompt)).toBeVisible()
  expect(screen.queryByText(otherItem.prompt)).not.toBeInTheDocument()
})
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/lib/review/vault-filter.test.ts web/src/routes/VaultReviewPage.test.ts`
Expected: FAIL because the filter/page are absent.

- [ ] **Step 3: Implement filtering without falsifying server state**

Filter `items` by `project_id`; set `caught_up` to `true` when the filtered items are empty so the vault page has a caught-up state even if other projects are due globally. Refresh projects and `scope=all` after rate/suspend/retry callbacks.

```ts
export function filterQueueByProjectIds(queue: ReviewQueue, ids: ReadonlySet<string>): ReviewQueue {
  const items = queue.items.filter(item => ids.has(item.project_id))
  return { ...queue, items, caught_up: items.length === 0 }
}
```

- [ ] **Step 4: Verify green**

Run: `rtk npm run test -- --run web/src/lib/review/vault-filter.test.ts web/src/routes/VaultReviewPage.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/lib/review/vault-filter.ts web/src/lib/review/vault-filter.test.ts web/src/routes/VaultReviewPage.svelte web/src/routes/VaultReviewPage.test.ts web/src/lib/router.ts web/src/App.svelte
rtk git commit -m "feat(web): scope review queue to vault projects"
```

### Task 35: Add Context-Aware Breadcrumbs

**Files:**
- Create: `web/src/components/Breadcrumbs.svelte`
- Create: `web/src/components/Breadcrumbs.test.ts`
- Modify: `web/src/routes/ProjectHubPage.svelte`
- Modify: `web/src/routes/NotesPage.svelte`
- Modify: `web/src/routes/ProjectSessionsPage.svelte`
- Modify: `web/src/routes/ProjectReviewPage.svelte`

**Interfaces:**
- Consumes: `project: Project` and optional `leaf: string`.
- Produces: accessible `<nav aria-label="Breadcrumb">`; vaulted path `Vaults / {vault} / {project} [/ leaf]`, unfiled path `Projects / {project} [/ leaf]`.

- [ ] **Step 1: Write failing breadcrumb tests**

```ts
it('links a vaulted project back through its vault', () => {
  render(Breadcrumbs, { project: vaultedProject, leaf: 'Sessions' })
  expect(screen.getByRole('link', { name: 'Vaults' })).toHaveAttribute('href', '#/vaults')
  expect(screen.getByRole('link', { name: 'HEALTH' })).toHaveAttribute('href', '#/vaults/v1')
  expect(screen.getByRole('link', { name: 'Sleep' })).toHaveAttribute('href', '#/projects/p1')
  expect(screen.getByText('Sessions')).toHaveAttribute('aria-current', 'page')
})
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/components/Breadcrumbs.test.ts`
Expected: FAIL because the component is missing.

- [ ] **Step 3: Implement and place breadcrumbs on every project surface**

Encode all IDs with `encodeURIComponent`, use an ordered list with separators hidden from assistive technology, and truncate long names visually without removing their full accessible text. Ensure project pages use the already-loaded `Project`, avoiding duplicate project requests.

- [ ] **Step 4: Verify focused and vault suites**

Run: `rtk npm run test -- --run web/src/components/Breadcrumbs.test.ts web/src/routes/VaultHomePage.test.ts web/src/routes/VaultProjectsPage.test.ts web/src/routes/VaultSessionsPage.test.ts web/src/routes/VaultReviewPage.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/components/Breadcrumbs.svelte web/src/components/Breadcrumbs.test.ts web/src/routes/ProjectHubPage.svelte web/src/routes/NotesPage.svelte web/src/routes/ProjectSessionsPage.svelte web/src/routes/ProjectReviewPage.svelte
rtk git commit -m "feat(web): add vault-aware project breadcrumbs"
```


## Phase: 05-project-surfaces

### Task 40: Build the Project Hub

**Files:**
- Create: `web/src/routes/ProjectHubPage.svelte`
- Create: `web/src/routes/ProjectHubPage.test.ts`
- Modify: `web/src/lib/router.ts`
- Modify: `web/src/App.svelte`

**Interfaces:**
- Consumes: `api.getProject(projectId)`, `Breadcrumbs`, and route shell context.
- Produces: `#/projects/:id` hub with Notes, Sessions, and Review links.

- [ ] **Step 1: Write the failing hub test**

```ts
it('renders project metrics and links without a second catalog', async () => {
  api.getProject.mockResolvedValue({ ...project, note_count: 3, session_count: 2, due_count: 1 })
  render(ProjectHubPage, { projectId: 'p1' })
  expect(await screen.findByRole('heading', { name: project.name })).toBeVisible()
  expect(screen.getByRole('link', { name: /notes/i })).toHaveAttribute('href', '#/projects/p1/notes')
  expect(screen.getByRole('link', { name: /sessions/i })).toHaveAttribute('href', '#/projects/p1/sessions')
  expect(screen.getByRole('link', { name: /review/i })).toHaveAttribute('href', '#/projects/p1/review')
})
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/routes/ProjectHubPage.test.ts`
Expected: FAIL because the hub is absent.

- [ ] **Step 3: Implement the hub**

Render breadcrumbs, title, optional vault badge, count cards, three action cards, a skeleton, and retryable inline hard-load error. Route links must encode the project ID.

- [ ] **Step 4: Verify green**

Run: `rtk npm run test -- --run web/src/routes/ProjectHubPage.test.ts web/src/lib/router.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/routes/ProjectHubPage.svelte web/src/routes/ProjectHubPage.test.ts web/src/lib/router.ts web/src/App.svelte
rtk git commit -m "feat(web): add project hub"
```

### Task 41: Restyle Notes as a Responsive Two-Pane Surface

**Files:**
- Create: `web/src/routes/NotesPage.svelte`
- Create: `web/src/routes/NotesPage.test.ts`
- Create: `web/src/components/notes/NoteTree.svelte`
- Create: `web/src/components/notes/NoteReader.svelte`
- Modify: `web/src/lib/api/index.ts`
- Modify: `web/src/lib/api/types.ts`

**Interfaces:**
- Consumes existing note tree/detail endpoints through `listProjectNotes(projectId)` and `getProjectNote(projectId, noteId)` matching the legacy client paths.
- Produces: selected note URL `#/projects/:id/notes/:noteId`; `NoteTree` emits `select(noteId: string)`.

- [ ] **Step 1: Write failing page tests**

```ts
it('shows tree and selected note in two panes', async () => {
  render(NotesPage, { projectId: 'p1', noteId: 'n1' })
  expect(await screen.findByRole('tree')).toBeVisible()
  expect(await screen.findByRole('article')).toHaveTextContent('Rendered note')
})

it('distinguishes an empty tree from no selection', async () => {
  api.listProjectNotes.mockResolvedValueOnce([])
  const { rerender } = render(NotesPage, { projectId: 'p1' })
  expect(await screen.findByText(/no notes yet/i)).toBeVisible()
  api.listProjectNotes.mockResolvedValueOnce([note])
  await rerender({ projectId: 'p1' })
  expect(await screen.findByText(/select a note/i)).toBeVisible()
})
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/routes/NotesPage.test.ts`
Expected: FAIL because Svelte note components are absent.

- [ ] **Step 3: Implement the same API behavior with new layout**

Use a fixed-width tree and flexible reader above 768px; stack them on mobile. Preserve server-provided safe rendering semantics from the old page (plain text remains text; only render HTML if the existing endpoint explicitly returns rendered HTML). Show tree/reader skeletons independently and keep tree visible if detail loading fails.

- [ ] **Step 4: Verify green**

Run: `rtk npm run test -- --run web/src/routes/NotesPage.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/routes/NotesPage.svelte web/src/routes/NotesPage.test.ts web/src/components/notes/NoteTree.svelte web/src/components/notes/NoteReader.svelte web/src/lib/api/index.ts web/src/lib/api/types.ts
rtk git commit -m "feat(web): rebuild notes as two-pane reader"
```

### Task 42 (40a): Build Project Session List and Creation

**Files:**
- Create: `web/src/routes/ProjectSessionsPage.svelte`
- Create: `web/src/routes/ProjectSessionsPage.test.ts`
- Create: `web/src/components/sessions/SessionList.svelte`
- Create: `web/src/components/sessions/SessionList.test.ts`
- Modify: `web/src/lib/api/index.ts`
- Modify: `web/src/lib/api/types.ts`

**Interfaces:**
- Produces: `listProjectSessions(projectId)`, `createProjectSession(projectId, input)`, `SessionList` event `open(session: Session)`.
- Request: `{ home: 'project', title, provider, model_id, model_parameters: {}, tool_grants: { workspace_files: boolean } }`.

- [ ] **Step 1: Write failing list/create tests**

```ts
it('lists and creates only through the project endpoint', async () => {
  render(ProjectSessionsPage, { projectId: 'p1' })
  await user.type(await screen.findByLabelText('Title'), 'Plan')
  await user.selectOptions(screen.getByLabelText('Model'), 'openai\u0000gpt')
  await user.click(screen.getByRole('button', { name: 'New session' }))
  expect(api.createProjectSession).toHaveBeenCalledWith('p1', {
    home: 'project', title: 'Plan', provider: 'openai', model_id: 'gpt',
    model_parameters: {}, tool_grants: { workspace_files: false },
  })
})

it('shows setup guidance rather than a form when models are empty', async () => {
  api.listModels.mockResolvedValue({ models: [] })
  render(ProjectSessionsPage, { projectId: 'p1' })
  expect(await screen.findByText(/configure a model/i)).toBeVisible()
  expect(screen.queryByRole('button', { name: 'New session' })).not.toBeInTheDocument()
})
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/components/sessions/SessionList.test.ts web/src/routes/ProjectSessionsPage.test.ts`
Expected: FAIL because list/create components are missing.

- [ ] **Step 3: Implement list, skeleton, empty/setup states, and create form**

Load models and project sessions concurrently. Keep creation errors inline, disable duplicate submit, expose workspace-files as an explicit grant checkbox, and open the created session only after a successful response.

- [ ] **Step 4: Verify green**

Run: `rtk npm run test -- --run web/src/components/sessions/SessionList.test.ts web/src/routes/ProjectSessionsPage.test.ts`
Expected: PASS with requests only under `/api/v1/projects/p1/sessions`.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/routes/ProjectSessionsPage.svelte web/src/routes/ProjectSessionsPage.test.ts web/src/components/sessions/SessionList.svelte web/src/components/sessions/SessionList.test.ts web/src/lib/api/index.ts web/src/lib/api/types.ts
rtk git commit -m "feat(web): add project session list and creation"
```

### Task 43 (40c): Lock the Composer Focus Invariant with a Failing Poll Test

**Files:**
- Create: `web/src/components/sessions/session-poller.ts`
- Create: `web/src/components/sessions/session-poller.test.ts`
- Create: `web/src/components/sessions/SessionChat.focus.test.ts`

**Interfaces:**
- Produces test contract for `SessionChat` from Task 44 and `createSessionPoller(load, apply, intervalMs)` with `start()`, `poll()`, and `stop()`.
- The test requires the exact same `<textarea>` DOM node, value, focus, and selection after unchanged and changed polls.

- [ ] **Step 1: Write the regression test before chat implementation**

```ts
it('patches messages and run state without replacing the focused composer', async () => {
  render(SessionChat, { session, projectId: 'p1', pollInterval: 60_000 })
  const composer = await screen.findByLabelText('Message') as HTMLTextAreaElement
  await user.click(composer)
  await user.type(composer, 'typing here')
  composer.setSelectionRange(6, 11)

  await pollHarness.resolve({ messages: [...initialMessages, reply], run: { status: 'running' } })

  expect(screen.getByLabelText('Message')).toBe(composer)
  expect(document.activeElement).toBe(composer)
  expect(composer.value).toBe('typing here')
  expect([composer.selectionStart, composer.selectionEnd]).toEqual([6, 11])
  expect(screen.getByText(reply.content)).toBeVisible()
  expect(screen.getByRole('status')).toHaveTextContent('Run: running')
})
```

- [ ] **Step 2: Run and preserve the intentional failure**

Run: `rtk npm run test -- --run web/src/components/sessions/SessionChat.focus.test.ts`
Expected: FAIL because `SessionChat.svelte` does not exist. Do not weaken identity/focus/selection assertions.

- [ ] **Step 3: Implement only the serialized/coalesced polling controller**

```ts
export function createSessionPoller<T>(load: () => Promise<T>, apply: (value: T) => void, intervalMs = 1500) {
  let active = false, queued = false, timer: ReturnType<typeof setInterval> | undefined
  async function poll() {
    queued = true
    if (active) return
    active = true
    try { while (queued) { queued = false; apply(await load()) } }
    finally { active = false }
  }
  return { poll, start: () => { void poll(); timer ??= setInterval(() => void poll(), intervalMs) }, stop: () => { if (timer) clearInterval(timer); timer = undefined; queued = false } }
}
```

- [ ] **Step 4: Test controller separately while focus test remains red**

Run: `rtk npm run test -- --run web/src/components/sessions/session-poller.test.ts web/src/components/sessions/SessionChat.focus.test.ts`
Expected: poller tests PASS and focus test FAIL only because chat is not implemented.

- [ ] **Step 5: Commit the red regression contract**

```bash
rtk git add web/src/components/sessions/session-poller.ts web/src/components/sessions/session-poller.test.ts web/src/components/sessions/SessionChat.focus.test.ts
rtk git commit -m "test(web): lock session composer focus during polls"
```

### Task 44 (40b): Implement the Stable Session Chat Shell

**Files:**
- Create: `web/src/components/sessions/SessionChat.svelte`
- Create: `web/src/components/sessions/SessionChat.test.ts`
- Modify: `web/src/components/sessions/SessionChat.focus.test.ts`
- Modify: `web/src/routes/ProjectSessionsPage.svelte`
- Modify: `web/src/lib/api/index.ts`

**Interfaces:**
- Consumes: `session`, `projectId`, `createSessionPoller`; APIs `listMessages`, `currentRun`, `sendMessage`.
- Produces stable chat shell with message list, run status, inline alert, sticky composer, and `close` callback.

- [ ] **Step 1: Add failing send/race/error tests**

Port cases from `web/js/pages/sessions.test.js`: one stable `request_key`, duplicate-submit suppression, failed-send draft retention, cached-history retention on poll failure, stale old-session result rejection, one timer across overlapping opens, and timer cleanup on destroy.

```ts
expect(api.sendMessage).toHaveBeenCalledTimes(1)
expect(api.sendMessage).toHaveBeenCalledWith(session.id, { content: 'draft', request_key: 'stable-key' })
expect(composer).toHaveValue('draft') // failed send
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/components/sessions/SessionChat.test.ts web/src/components/sessions/SessionChat.focus.test.ts`
Expected: FAIL against the missing chat.

- [ ] **Step 3: Implement without conditional/remounting composer ancestry**

Keep one keyed chat shell per `session.id`; messages and run are rune state updates. Do not key the message list/run status together with the form, do not conditionally recreate the form during polling, and clear the textarea only after this session's successful send. Guard every async apply with a generation/session ID.

```svelte
<section class="session-chat">
  <ol class="messages">{#each messages as message (message.sequence)}<MessageRow {message} />{/each}</ol>
  <p class="run-status" role="status" aria-live="polite">{run ? `Run: ${run.status}` : 'Idle'}</p>
  <InlineAlert message={error} />
  <form class="sticky bottom-0" onsubmit={send}>
    <label>Message<textarea bind:this={composer} bind:value={draft} required></textarea></label>
    <Button disabled={sending || !!run}>Send</Button>
  </form>
</section>
```

- [ ] **Step 4: Verify all chat behavior including node identity**

Run: `rtk npm run test -- --run web/src/components/sessions/SessionChat.test.ts web/src/components/sessions/SessionChat.focus.test.ts`
Expected: PASS; the focus test confirms the textarea object is unchanged after a changed poll.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/components/sessions/SessionChat.svelte web/src/components/sessions/SessionChat.test.ts web/src/components/sessions/SessionChat.focus.test.ts web/src/routes/ProjectSessionsPage.svelte web/src/lib/api/index.ts
rtk git commit -m "feat(web): add focus-safe session chat"
```

### Task 45: Add the Session Workspace Panel

**Files:**
- Create: `web/src/components/sessions/WorkspacePanel.svelte`
- Create: `web/src/components/sessions/WorkspacePanel.test.ts`
- Modify: `web/src/components/sessions/SessionChat.svelte`
- Modify: `web/src/lib/api/index.ts`

**Interfaces:**
- Consumes: session persisted `tool_grants`/`tool_grants_json`, `workspaceTree(sessionId)`, `workspaceFile(sessionId, path)`.
- Produces: selected promotable file callback `onpromote(file: WorkspaceFile)`; panel appears only when `workspace_files === true`.

- [ ] **Step 1: Write failing grant/tree/refresh tests**

```ts
it.each([{ grants: '{bad', visible: false }, { grants: '{"workspace_files":false}', visible: false }, { grants: '{"workspace_files":true}', visible: true }])(
  'gates workspace from persisted grants', async ({ grants, visible }) => {
    render(SessionChat, { session: { ...session, tool_grants_json: grants }, projectId: 'p1' })
    if (visible) expect(await screen.findByRole('complementary', { name: 'Workspace' })).toBeVisible()
    else expect(screen.queryByRole('complementary', { name: 'Workspace' })).not.toBeInTheDocument()
  },
)
it('refreshes tree after a newly polled tool message changes a path', async () => {
  expect(api.workspaceTree).toHaveBeenCalledTimes(2)
})
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/components/sessions/WorkspacePanel.test.ts`
Expected: FAIL because panel is absent.

- [ ] **Step 3: Implement responsive panel and safe file selection**

Split chat/workspace on desktop and stack on mobile. Parse malformed grant JSON as disabled. Render tree skeleton/error independently; fetch selected file content; offer `Save to source` only for regular lowercase `.md` files. A message refresh must update panel content without remounting the chat composer.

- [ ] **Step 4: Verify workspace and focus together**

Run: `rtk npm run test -- --run web/src/components/sessions/WorkspacePanel.test.ts web/src/components/sessions/SessionChat.focus.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/components/sessions/WorkspacePanel.svelte web/src/components/sessions/WorkspacePanel.test.ts web/src/components/sessions/SessionChat.svelte web/src/lib/api/index.ts
rtk git commit -m "feat(web): add session workspace panel"
```

### Task 46: Add Promotion Dialog and Operation Status Badges

**Files:**
- Create: `web/src/components/sessions/PromoteDialog.svelte`
- Create: `web/src/components/sessions/PromoteDialog.test.ts`
- Create: `web/src/components/sessions/OperationBadges.svelte`
- Create: `web/src/components/sessions/OperationBadges.test.ts`
- Create: `web/src/lib/promote.ts`
- Create: `web/src/lib/promote.test.ts`
- Modify: `web/src/components/sessions/SessionChat.svelte`
- Modify: `web/src/lib/api/index.ts`

**Interfaces:**
- Produces: `nextPromoteAttempt(previous, payload, uuid): { payload; key }`; POST `/api/v1/sessions/{id}/promote` with `Idempotency-Key`; persisted operation IDs key `personal-agent:v1:promote-operations:{sessionId}`.
- Consumes operation GET `/api/v1/operations/{operationId}` and retry POST `/api/v1/review/pending/{pendingId}/retry`.

- [ ] **Step 1: Write failing idempotency, lifecycle, and badge tests**

```ts
expect(nextPromoteAttempt(first, samePayload, uuid).key).toBe(first.key)
expect(nextPromoteAttempt(first, changedPayload, uuid).key).not.toBe(first.key)
expect(api.promoteSession).toHaveBeenCalledWith('s1', {
  workspace_path: 'draft.md', target_relative_path: 'notes/draft.md', review_mode: 'bites',
}, expect.any(String))
```

Test cancel/Escape/native close/back/session-switch cleanup, captured source file, `.md` validation, dialog state surviving polls, operation polling coalescing, retry-card deduplication, and all five exact badge strings.

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/lib/promote.test.ts web/src/components/sessions/PromoteDialog.test.ts web/src/components/sessions/OperationBadges.test.ts`
Expected: FAIL because promotion UI/helpers are absent.

- [ ] **Step 3: Implement promotion and resilient operation polling**

The dialog heading is exactly `Save to source`; field names are exactly `target_relative_path` and `review_mode`; require lowercase regular Markdown target/source rules already enforced by legacy tests. Capture the selected source and session at open. On success require `operation_id`, persist it, close, and poll. Keep dialog outside frequently-updated message markup. Disable retry while pending and retain ordinary errors independently from operation errors.

```ts
export const promoteSession = (sessionId: string, payload: PromotePayload, key: string) =>
  request<{ operation_id: string }>(`/api/v1/sessions/${encodeURIComponent(sessionId)}/promote`, {
    method: 'POST', body: payload, headers: { 'Idempotency-Key': key },
  })
```

- [ ] **Step 4: Run full session surface suite**

Run: `rtk npm run test -- --run web/src/components/sessions web/src/lib/promote.test.ts web/src/routes/ProjectSessionsPage.test.ts`
Expected: PASS, including focus while badges/workspace update.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/components/sessions/PromoteDialog.svelte web/src/components/sessions/PromoteDialog.test.ts web/src/components/sessions/OperationBadges.svelte web/src/components/sessions/OperationBadges.test.ts web/src/lib/promote.ts web/src/lib/promote.test.ts web/src/components/sessions/SessionChat.svelte web/src/lib/api/index.ts
rtk git commit -m "feat(web): add idempotent session promotion"
```


## Phase: 06-review-settings-harden

### Task 50: Build Shared Global and Project Review

**Files:**
- Create: `web/src/components/review/ReviewRunner.svelte`
- Create: `web/src/components/review/ReviewRunner.test.ts`
- Create: `web/src/routes/ReviewPage.svelte`
- Create: `web/src/routes/ProjectReviewPage.svelte`
- Create: `web/src/routes/ReviewPages.test.ts`
- Modify: `web/src/lib/api/index.ts`
- Modify: `web/src/lib/api/types.ts`
- Modify: `web/src/lib/router.ts`
- Modify: `web/src/App.svelte`

**Interfaces:**
- Consumes: `getReviewQueue(scope)`, `rateReviewItem(id, { rating, request_key, row_version, duration_ms })`, `suspendReviewItem(id)`, `retryReviewPending(id)`.
- Produces: `ReviewRunner` prop `scope: 'all' | `project:${string}``; global route query `#/review?scope=all`; project route `#/projects/:id/review`.

- [ ] **Step 1: Write failing scope and interaction tests**

```ts
it('defaults global review to all', async () => {
  render(ReviewPage, { query: new URLSearchParams() })
  expect(api.getReviewQueue).toHaveBeenCalledWith('all')
})

it('uses project scope and sends concurrency/timing fields', async () => {
  render(ProjectReviewPage, { projectId: 'p1' })
  expect(api.getReviewQueue).toHaveBeenCalledWith('project:p1')
  await user.click(await screen.findByRole('button', { name: 'Good' }))
  expect(api.rateReviewItem).toHaveBeenCalledWith('r1', expect.objectContaining({
    rating: 'good', request_key: expect.any(String), row_version: 4, duration_ms: expect.any(Number),
  }))
})
```

Also test scope chips reflected as `scope=` in hash, caught-up empty state from `caught_up`, retry, suspend, loading skeleton, inline failure, duplicate rating suppression, and 409 refresh.

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/components/review/ReviewRunner.test.ts web/src/routes/ReviewPages.test.ts`
Expected: FAIL because review surfaces are missing.

- [ ] **Step 3: Implement the shared runner and wrappers**

Record `performance.now()` when a card becomes active and calculate non-negative integer `duration_ms` at rating. Preserve the card on request failure; on success/409 reload queue. Render chips for All and available `project:` scopes on global review. Project wrapper passes exactly `project:${projectId}` and breadcrumbs.

```ts
const payload = {
  rating,
  request_key: crypto.randomUUID(),
  row_version: item.row_version,
  duration_ms: Math.max(0, Math.round(performance.now() - shownAt)),
}
```

- [ ] **Step 4: Verify green**

Run: `rtk npm run test -- --run web/src/components/review/ReviewRunner.test.ts web/src/routes/ReviewPages.test.ts web/src/routes/VaultReviewPage.test.ts`
Expected: PASS for global, project, and Task 34's vault wrapper.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/components/review/ReviewRunner.svelte web/src/components/review/ReviewRunner.test.ts web/src/routes/ReviewPage.svelte web/src/routes/ProjectReviewPage.svelte web/src/routes/ReviewPages.test.ts web/src/lib/api/index.ts web/src/lib/api/types.ts web/src/lib/router.ts web/src/App.svelte
rtk git commit -m "feat(web): rebuild global and project review"
```

### Task 51: Rebuild Settings and Backup UX

**Files:**
- Create: `web/src/routes/SettingsPage.svelte`
- Create: `web/src/routes/SettingsPage.test.ts`
- Create: `web/src/components/settings/BackupSection.svelte`
- Create: `web/src/components/settings/BackupSection.test.ts`
- Modify: `web/src/lib/api/index.ts`
- Modify: `web/src/lib/api/types.ts`

**Interfaces:**
- Consumes: `GET/PUT /api/v1/settings`, `GET/POST /api/v1/backups`.
- Produces single-column settings cards; backup schedule `off | daily`; backup list/status and “Backup now.”

- [ ] **Step 1: Write failing settings and backup tests**

```ts
it('saves schedule while preserving the complete settings payload', async () => {
  render(SettingsPage)
  await user.selectOptions(await screen.findByLabelText('Schedule'), 'daily')
  expect(api.updateSettings).toHaveBeenCalledWith({
    timezone: 'Asia/Jakarta', default_provider: 'openai', default_model_id: 'gpt', backup_schedule: 'daily',
  })
  expect(await screen.findByText('Schedule saved.')).toBeVisible()
})

it('runs backup, refreshes history, and reports completion', async () => {
  await user.click(await screen.findByRole('button', { name: 'Backup now' }))
  expect(api.createBackup).toHaveBeenCalledTimes(1)
  expect(api.listBackups).toHaveBeenCalledTimes(2)
  expect(await screen.findByText('Backup completed.')).toBeVisible()
})
```

- [ ] **Step 2: Verify red**

Run: `rtk npm run test -- --run web/src/routes/SettingsPage.test.ts web/src/components/settings/BackupSection.test.ts`
Expected: FAIL because settings components are absent.

- [ ] **Step 3: Implement settings cards and resilient backup state**

Render timezone/model values, remote sink status, schedule, last success, newer last failure, and backup history. Disable only the active action. Keep loaded settings/history on failures and put errors/success next to their action with `aria-live="polite"`; a hard initial failure gets a page retry.

- [ ] **Step 4: Verify green**

Run: `rtk npm run test -- --run web/src/routes/SettingsPage.test.ts web/src/components/settings/BackupSection.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
rtk git add web/src/routes/SettingsPage.svelte web/src/routes/SettingsPage.test.ts web/src/components/settings/BackupSection.svelte web/src/components/settings/BackupSection.test.ts web/src/lib/api/index.ts web/src/lib/api/types.ts
rtk git commit -m "feat(web): rebuild settings and backups"
```

### Task 52: Prove Feature Parity and Remove Legacy Web Assets

**Files:**
- Delete: `web-legacy/`
- Modify: `web/index.html` (remove any remaining legacy script/style references; retain only the Vite entry during development)
- Modify: `web/package.json`

**Interfaces:**
- Consumes: all frontend component/unit tests from Tasks 10–51.
- Produces: one Svelte source tree and one generated `web/dist` production artifact; no `web-legacy` fallback.

- [ ] **Step 1: Add a failing no-legacy assertion**

Add a frontend script:

```json
"check:no-legacy": "test ! -d ../web-legacy && ! grep -R 'web-legacy' src index.html"
```

Run: `rtk npm run check:no-legacy`
Expected: FAIL while `web-legacy` exists.

- [ ] **Step 2: Run the complete frontend suite before deletion**

Run: `rtk npm test -- --run`
Expected: PASS, proving auth, shell, global, vault, project, sessions focus, review, and settings behavior exists in Svelte.

- [ ] **Step 3: Delete only legacy assets and remove references**

Delete `web-legacy`; do not delete `web/src`, package files, Vite config, or production output configuration. Ensure `web/index.html` contains the Svelte entry `<script type="module" src="/src/main.ts"></script>` and no old paths.

- [ ] **Step 4: Verify source/build contain no legacy dependency**

Run: `rtk npm run check:no-legacy && rtk npm run build`
Expected: PASS and `web/dist/index.html` plus hashed assets are generated.

- [ ] **Step 5: Commit**

```bash
rtk git add -A web-legacy web/index.html web/package.json
rtk git commit -m "chore(web): remove legacy frontend"
```

### Task 53: Point Go Static and Contract Tests at Vite Output

**Files:**
- Modify: `internal/httpapi/static_test.go`
- Modify: `internal/httpapi/web_test.go`
- Modify: `internal/httpapi/server.go` if an earlier tooling task has not already changed the production root to `web/dist`
- Modify: `deploy/Dockerfile` if an earlier tooling task has not already copied the built `web/dist`

**Interfaces:**
- Production static root: `web/dist` (repository-relative; tests use `../../web/dist`).
- Contract source paths: promotion component/helper, review component/API helper, and operation badge component created in Tasks 46 and 50.

- [ ] **Step 1: Update tests first and run them red against old assumptions**

```go
func TestStaticShell(t *testing.T) {
    h := http.FileServer(http.Dir("../../web/dist"))
    r := httptest.NewRequest(http.MethodGet, "/", nil)
    w := httptest.NewRecorder()
    h.ServeHTTP(w, r)
    if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Personal Agent") || !strings.Contains(w.Body.String(), `type="module"`) {
        t.Fatalf("%d %s", w.Code, w.Body.String())
    }
}
```

Change `web_test.go` path map to:

```go
tests := map[string][]string{
  "../../web/src/components/sessions/PromoteDialog.svelte": {"Save to source", "target_relative_path", "review_mode"},
  "../../web/src/lib/api/index.ts": {"operation_id", "scope="},
  "../../web/src/components/review/ReviewRunner.svelte": {"project:", "caught_up", "row_version", "duration_ms"},
  "../../web/src/components/sessions/OperationBadges.svelte": {"Promoting…", "Promote failed — Retry", "Note saved; cards pending…", "Cards failed — Retry cards", "Ready"},
}
```

Run: `rtk go test ./internal/httpapi -run 'TestStaticShell|TestWebContains' -count=1`
Expected: FAIL until the production build exists/serve root is aligned.

- [ ] **Step 2: Build the frontend production artifact**

Run: `rtk npm run build`
Expected: PASS and `web/dist/index.html` exists.

- [ ] **Step 3: Align production serving/copy paths**

Set the production file server to the built dist directory according to the tooling plan's configurable static-root mechanism. Ensure the Docker build runs frontend build before the final image and copies `web/dist`, not raw `web/src`. Do not add a source mount to production compose.

- [ ] **Step 4: Verify Go contracts and production compose policy**

Run: `rtk go test ./internal/httpapi -run 'TestStaticShell|TestWebContains' -count=1 && rtk grep 'volumes:' deploy/docker-compose.yml`
Expected: Go tests PASS; production compose contains no live `web`, repository, or source mount.

- [ ] **Step 5: Commit**

```bash
rtk git add internal/httpapi/static_test.go internal/httpapi/web_test.go internal/httpapi/server.go deploy/Dockerfile web/dist
rtk git commit -m "test(web): validate Vite production assets"
```

### Task 54: Document Frontend Development and Deployment

**Files:**
- Modify: `README.md`
- Modify: `docs/ops/deploy.md`

**Interfaces:**
- Documents the exact tooling decision established by the scaffold plan: `make docker-dev`, browser URL, Vite HMR path, production build command/output, and production no-live-mount rule.

- [ ] **Step 1: Write a documentation assertion checklist**

Run:

```bash
rtk grep 'make docker-dev|localhost:8080|HMR|web/dist|production.*mount' README.md docs/ops/deploy.md
```

Expected: FAIL to find one or more required statements before edits.

- [ ] **Step 2: Update README frontend section**

State: prerequisites; `make docker-dev` as one-command API+UI loop; open `http://localhost:8080`; ordinary `web/src` edits update by Vite HMR without rebuilding; frontend test/build commands run from `web`; `web/dist` is generated production output.

- [ ] **Step 3: Update deployment operations guide**

Document the chosen dev process/proxy and HMR websocket, how to inspect the `:8080` listener and verify served bytes, production image frontend build/copy, and this exact warning: “Production Compose is image-baked and has no live source mounts; live repository mounts exist only in `docker-compose.dev.yml`."

- [ ] **Step 4: Verify required documentation terms**

Run: `rtk grep 'make docker-dev|localhost:8080|HMR|web/dist|no live source mounts|docker-compose.dev.yml' README.md docs/ops/deploy.md`
Expected: all terms appear in appropriate sections.

- [ ] **Step 5: Commit**

```bash
rtk git add README.md docs/ops/deploy.md
rtk git commit -m "docs: explain Svelte development and deployment"
```

### Task 55: Verification-Before-Completion Release Gate

**Files:**
- Modify only if verification exposes a defect: the smallest relevant source/test/doc file from Tasks 10–54.

**Interfaces:**
- Consumes the complete repository and running Docker dev environment.
- Produces fresh command evidence and a completed manual checklist; this task does not waive failures.

- [ ] **Step 1: Load the required verification skill and inspect the diff**

Invoke `verification-before-completion`, then run:

```bash
rtk git status --short
rtk git diff --check
rtk grep 'web-legacy' web internal/httpapi README.md docs deploy
```

Expected: no whitespace errors and no stale runtime/test/doc references to deleted legacy assets.

- [ ] **Step 2: Run backend and frontend automated gates from fresh builds**

```bash
rtk go test ./...
rtk npm test -- --run
rtk npm run build
```

Run frontend commands with `workdir=web`. Expected: all commands exit 0; `web/dist/index.html` exists.

- [ ] **Step 3: Verify production and development composition**

```bash
rtk docker compose -f deploy/docker-compose.yml config
rtk docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.dev.yml config
rtk make docker-dev
```

Expected: both configs validate; production config has no repository/web source mount; development services become healthy without rebuilding for an ordinary UI edit.

- [ ] **Step 4: Execute and record the manual browser checklist**

At `http://localhost:8080`, verify each item:

```text
[ ] Edit a visible web/src string; HMR updates it without image rebuild/container recreation.
[ ] Create vault HEALTH, enter #/vaults/:id, and observe vault-only sidebar.
[ ] Create a project; request has locked vault_id and project deep link retains vault context.
[ ] Leave vault; URL is #/home and global sidebar returns.
[ ] Search global projects/vaults and see dedicated empty states.
[ ] Open a vault session page; sessions are aggregated only through project endpoints.
[ ] Type in a session composer through multiple polls; value, caret, and focus remain.
[ ] Open workspace promotion; submit uses Idempotency-Key and status reaches Ready/retry state.
[ ] Global review requests scope=all; project review requests scope=project:<id>; vault queue excludes other projects.
[ ] Run Backup now and observe refreshed success/history or an inline actionable failure.
[ ] Collapse sidebar and verify mobile drawer/session workspace behavior.
```

Also run `rtk lsof -nP -iTCP:8080 -sTCP:LISTEN` and `rtk curl http://localhost:8080/`; confirm the listener is the intended dev stack and served HTML references Vite assets/HMR.

- [ ] **Step 5: Stop dev services, rerun affected tests after any fix, and commit verification repairs**

```bash
rtk docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.dev.yml down
rtk git status --short
```

If verification required changes, rerun the failing command plus `rtk go test ./...` and `rtk npm test -- --run`, then commit only those repairs:

```bash
rtk git add web/src internal/httpapi deploy README.md docs/ops/deploy.md
rtk git commit -m "fix(web): resolve release verification findings"
```

Expected: dev services stop cleanly; no untracked temporary files; all final evidence is green. If no repair was needed, do not create an empty commit.

