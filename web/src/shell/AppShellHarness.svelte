<!-- Test harness: provides a real Svelte 5 snippet child for AppShell. -->
<script lang="ts">
  import type { AppRoute } from '../lib/router';
  import type { ShellContext } from '../lib/stores/shell-context';
  import AppShell from './AppShell.svelte';

  let {
    context,
    route,
    health,
  }: {
    context: ShellContext;
    route: AppRoute;
    health: string;
  } = $props();
</script>

<AppShell {context} {route} {health}>
  <p>Dashboard</p>
</AppShell>
