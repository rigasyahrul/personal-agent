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
