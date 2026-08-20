<!-- web/src/App.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { parseRoute, type AppRoute } from './lib/router';
  import { api } from './lib/api/client';
  import type { Vault } from './lib/api/types';
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

  let { authLoader = loadAuthState }: { authLoader?: typeof loadAuthState } = $props();
  let auth = $state<AuthState>({ status: 'loading' });
  let route = $state<AppRoute>(parseRoute(location.hash));
  let vaults = $state<VaultSummary[]>([]);
  const context = $derived(deriveShellContext(route, vaults));

  const vaultName = $derived(
    context.mode === 'vault' ? context.vaultName : 'Vault',
  );

  onMount(() => {
    const updateRoute = () => {
      route = parseRoute(location.hash);
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
    return () => removeEventListener('hashchange', updateRoute);
  });

  async function completeAuth() {
    await refreshAuth();
    auth = await authLoader();
    if (!location.hash) location.hash = '#/home';
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
  <AppShell {context} {route} health="Storage status unavailable">
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
    {:else}
      <p>Route: {route.name}</p>
    {/if}
  </AppShell>
{/if}
