<!-- web/src/App.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { parseRoute, type AppRoute } from './lib/router';
  import { api } from './lib/api/client';
  import type { Project, Vault } from './lib/api/types';
  import { loadAuthState, refreshAuth, type AuthState } from './lib/stores/auth';
  import { deriveShellContext, type VaultSummary } from './lib/stores/shell-context';
  import AppShell from './shell/AppShell.svelte';
  import BootstrapPage from './routes/auth/BootstrapPage.svelte';
  import LoginPage from './routes/auth/LoginPage.svelte';
  import HomePage from './routes/HomePage.svelte';
  import ProjectsPage from './routes/ProjectsPage.svelte';
  import VaultsPage from './routes/VaultsPage.svelte';
  import VaultHomePage from './routes/VaultHomePage.svelte';
  import VaultProjectsPage from './routes/VaultProjectsPage.svelte';
  import VaultSessionsPage from './routes/VaultSessionsPage.svelte';
  import VaultReviewPage from './routes/VaultReviewPage.svelte';
  import ProjectHubPage from './routes/ProjectHubPage.svelte';
  import NotesPage from './routes/NotesPage.svelte';
  import ProjectSessionsPage from './routes/ProjectSessionsPage.svelte';
  import ProjectReviewPage from './routes/ProjectReviewPage.svelte';
  import ReviewPage from './routes/ReviewPage.svelte';
  import SettingsPage from './routes/SettingsPage.svelte';

  let { authLoader = loadAuthState }: { authLoader?: typeof loadAuthState } = $props();
  let auth = $state<AuthState>({ status: 'loading' });
  let route = $state<AppRoute>(parseRoute(location.hash));
  let vaults = $state<VaultSummary[]>([]);
  let routeProject = $state<Project | null>(null);
  let health = $state<'unknown' | 'ready' | 'error'>('unknown');
  const context = $derived(deriveShellContext(route, vaults, routeProject ?? undefined));

  const vaultName = $derived(
    context.mode === 'vault' ? context.vaultName : 'Vault',
  );

  const projectRouteNames = new Set(['project', 'notes', 'note', 'sessions', 'project-review']);

  const reviewQuery = $derived(
    route.name === 'review'
      ? new URLSearchParams(route.scope ? `scope=${route.scope}` : '')
      : new URLSearchParams(),
  );

  onMount(() => {
    const updateRoute = () => {
      const next = parseRoute(location.hash);
      if (!projectRouteNames.has(next.name)) {
        routeProject = null;
      } else if (
        !routeProject ||
        !('projectId' in next) ||
        routeProject.id !== next.projectId
      ) {
        // Clear stale membership until the page loads the project.
        routeProject = null;
      }
      route = next;
    };
    addEventListener('hashchange', updateRoute);
    void authLoader().then((value) => {
      auth = value;
    });
    void api
      .get<Vault[]>('/api/v1/vaults')
      .then((listed) => {
        vaults = listed.map((vault) => ({ id: vault.id, name: vault.name }));
      })
      .catch(() => {
        /* shell falls back to generic vault name */
      });
    void api
      .get<{ ok?: boolean; storage_writable?: boolean }>('/health')
      .then((body) => {
        health = body?.storage_writable ? 'ready' : 'error';
      })
      .catch(() => {
        health = 'unknown';
      });
    return () => removeEventListener('hashchange', updateRoute);
  });

  async function completeAuth() {
    await refreshAuth();
    auth = await authLoader();
    if (!location.hash) location.hash = '#/home';
  }

  function handleProjectLoad(project: Project | null) {
    routeProject = project;
  }
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
  <AppShell {context} {route} {health}>
    {#if route.name === 'home'}
      <HomePage />
    {:else if route.name === 'projects'}
      <ProjectsPage />
    {:else if route.name === 'vaults'}
      <VaultsPage />
    {:else if route.name === 'vault-home'}
      <VaultHomePage vaultId={route.vaultId} {vaultName} />
    {:else if route.name === 'vault-projects'}
      <VaultProjectsPage vaultId={route.vaultId} {vaultName} />
    {:else if route.name === 'vault-sessions'}
      <VaultSessionsPage vaultId={route.vaultId} {vaultName} />
    {:else if route.name === 'vault-review'}
      <VaultReviewPage vaultId={route.vaultId} {vaultName} />
    {:else if route.name === 'project'}
      <ProjectHubPage projectId={route.projectId} onProjectLoad={handleProjectLoad} />
    {:else if route.name === 'notes' || route.name === 'note'}
      <NotesPage
        projectId={route.projectId}
        noteId={route.name === 'note' ? route.noteId : undefined}
        onProjectLoad={handleProjectLoad}
      />
    {:else if route.name === 'sessions'}
      <ProjectSessionsPage projectId={route.projectId} onProjectLoad={handleProjectLoad} />
    {:else if route.name === 'project-review'}
      <ProjectReviewPage projectId={route.projectId} onProjectLoad={handleProjectLoad} />
    {:else if route.name === 'review'}
      <ReviewPage query={reviewQuery} />
    {:else if route.name === 'settings'}
      <SettingsPage />
    {:else}
      <p>Route: {route.name}</p>
    {/if}
  </AppShell>
{/if}
