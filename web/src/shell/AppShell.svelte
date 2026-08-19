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
