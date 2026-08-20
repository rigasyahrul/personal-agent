<!-- web/src/shell/TopBar.svelte -->
<script lang="ts">
  export type HealthStatus = 'unknown' | 'ready' | 'error' | string;

  let { health }: { health: HealthStatus } = $props();

  const normalized = $derived.by(() => {
    const raw = (health ?? '').toString().trim().toLowerCase();
    if (!raw || raw === 'unknown' || raw.includes('unavailable')) return 'unknown' as const;
    if (raw === 'ready' || raw.includes('ready') || raw.includes('ok')) return 'ready' as const;
    if (raw === 'error' || raw.includes('issue') || raw.includes('fail') || raw.includes('error')) {
      return 'error' as const;
    }
    return 'unknown' as const;
  });

  const tone = $derived(
    normalized === 'ready' ? 'ok' : normalized === 'error' ? 'warn' : 'muted',
  );

  const label = $derived(
    normalized === 'ready'
      ? 'Storage ready'
      : normalized === 'error'
        ? 'Storage issue'
        : 'Storage',
  );

  const title = $derived(
    normalized === 'unknown' ? 'Storage status not checked yet' : label,
  );
</script>

<header class="topbar">
  <div class="topbar__spacer"></div>
  <span
    class="health-pill"
    data-testid="health-pill"
    data-tone={tone}
    title={title}
  >
    <span class="health-pill__dot" aria-hidden="true"></span>
    {label}
  </span>
</header>
