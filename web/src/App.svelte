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
    const updateRoute = () => {
      route = parseRoute(location.hash);
    };
    addEventListener('hashchange', updateRoute);
    void authLoader().then((value) => {
      auth = value;
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
    <p>Route: {route.name}</p>
  </AppShell>
{/if}
