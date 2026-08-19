<!-- web/src/shell/TopBar.svelte -->
<script lang="ts">let { health }: { health: string } = $props();</script>
<header class="topbar"><div class="topbar__spacer"></div><span class="health-pill">{health}</span></header>
