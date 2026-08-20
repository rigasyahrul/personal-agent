<!-- web/src/shell/Sidebar.svelte -->
<script lang="ts">
  import type { AppRoute } from '../lib/router';
  import { routeToHash } from '../lib/router';
  import type { ShellContext } from '../lib/stores/shell-context';
  import { iconForLabel, navIconPath, type NavIconName } from './nav-icons';
  import { readSidebarCollapsed, writeSidebarCollapsed } from './sidebar-state';

  let { context, route }: { context: ShellContext; route: AppRoute } = $props();
  let collapsed = $state(readSidebarCollapsed(localStorage));

  const sessionsHint = 'Open a project to view its sessions';

  const globalItems = [
    ['Home', routeToHash({ name: 'home' })],
    ['Projects', routeToHash({ name: 'projects' })],
    ['Sessions', ''],
    ['Vaults', routeToHash({ name: 'vaults' })],
    ['Review', routeToHash({ name: 'review', scope: 'all' })],
    ['Settings', routeToHash({ name: 'settings' })],
  ] as const;

  const vaultItems = $derived(
    context.mode === 'vault'
      ? ([
          ['Home', routeToHash({ name: 'vault-home', vaultId: context.vaultId })],
          ['Projects', routeToHash({ name: 'vault-projects', vaultId: context.vaultId })],
          ['Sessions', routeToHash({ name: 'vault-sessions', vaultId: context.vaultId })],
          ['Review', routeToHash({ name: 'vault-review', vaultId: context.vaultId })],
          ['Settings', routeToHash({ name: 'settings' })],
        ] as const)
      : [],
  );

  const items = $derived(context.mode === 'vault' ? vaultItems : globalItems);

  function toggle() {
    collapsed = !collapsed;
    writeSidebarCollapsed(localStorage, collapsed);
  }

  function iconName(label: string): NavIconName {
    return iconForLabel(label);
  }
</script>

<aside class="sidebar" data-testid="sidebar" data-collapsed={collapsed}>
  <div class="sidebar__brand">{collapsed ? 'PA' : 'Personal Agent'}</div>
  {#if context.mode === 'vault'}
    <div class="sidebar__context">
      <strong>{context.vaultName}</strong>
      <a href="#/home">Leave vault</a>
    </div>
  {/if}
  <nav aria-label="Primary">
    {#each items as item}
      {@const label = item[0]}
      {@const href = item[1]}
      {@const path = navIconPath(iconName(label))}
      {#if href}
        <a
          href={href}
          aria-current={href === routeToHash(route) ? 'page' : undefined}
          title={label}
        >
          <svg class="nav-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
            <path d={path} fill="currentColor" />
          </svg>
          <span class="sidebar__label">{label}</span>
        </a>
      {:else}
        <span
          class="sidebar__disabled"
          aria-disabled="true"
          title={sessionsHint}
          aria-description={sessionsHint}
        >
          <svg class="nav-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
            <path d={path} fill="currentColor" />
          </svg>
          <span class="sidebar__label">{label}</span>
        </span>
      {/if}
    {/each}
  </nav>
  <button
    type="button"
    class="sidebar__collapse"
    onclick={toggle}
    aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
    title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
  >
    <svg class="nav-icon" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path
        d={navIconPath(collapsed ? 'panel-left' : 'panel-left-close')}
        fill="currentColor"
      />
    </svg>
    <span class="sidebar__collapse-label">{collapsed ? 'Expand' : 'Collapse'}</span>
  </button>
</aside>
