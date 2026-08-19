# Review, Settings, and Frontend Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Finish global/project review and settings, remove the legacy frontend, align Go static contract tests with Vite output, document operations, and execute all release gates.

**Architecture:** One reusable `ReviewRunner` receives a scope (`all` or `project:{id}`), owns rating timing/version semantics, and is wrapped by route pages. Settings combines the existing settings and backup APIs. Once feature parity tests pass, the temporary `web-legacy` tree is deleted; Go serves/tests `web/dist`, while source-string contract tests point to stable Svelte/TypeScript files.

**Tech Stack:** Svelte 5, TypeScript, Vite, Tailwind CSS, Vitest, Go `net/http`, Docker Compose

## Global Constraints

- Global review defaults to `scope=all`; project review uses `scope=project:{id}`.
- Preserve `project:`, `scope=`, `caught_up`, `row_version`, and `duration_ms` in new Svelte/TypeScript sources for the Go contract test.
- Preserve promotion status strings listed in the redesign spec.
- Delete `web` legacy assets only after Svelte feature completeness is demonstrated.
- Update `internal/httpapi/web_test.go` and `internal/httpapi/static_test.go` for new sources and production `dist`.
- Update `README.md` frontend guidance and `docs/ops/deploy.md` HMR/production behavior.
- Final gate is `go test ./...`, frontend tests/build, and a manual `make docker-dev` HMR checklist; production compose must retain no live source mounts.

---

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
