# Shell, Routing, API, and Auth Draft

> This is the Task 10–15 draft for assembly into the UI redesign implementation plan. Tasks 1–9 are reserved for tooling.

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
