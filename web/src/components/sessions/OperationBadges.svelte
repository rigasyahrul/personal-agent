<!-- web/src/components/sessions/OperationBadges.svelte -->
<script lang="ts">
  import type { OperationStatus } from '../../lib/api/types'
  import { safeBadge } from '../../lib/promote'

  let {
    operations,
    results,
    retrying = new Set<string>(),
    onretry,
  }: {
    operations: string[]
    results: Map<string, OperationStatus>
    retrying?: Set<string>
    onretry?: (operation: OperationStatus) => void
  } = $props()
</script>

<div class="flex flex-wrap gap-2" data-operation-statuses>
  {#each operations as id (id)}
    {@const op = results.get(id)}
    {#if !op}
      <div role="status" class="rounded-full bg-amber-50 px-2 py-1 text-xs text-amber-800">Promoting…</div>
    {:else}
      {@const label = safeBadge(op.badge)}
      <div role="status" class="inline-flex items-center gap-2 rounded-full bg-slate-100 px-2 py-1 text-xs text-slate-800">
        <span>{label}</span>
        {#if op.retry_cards === true && op.pending_id}
          <button
            type="button"
            class="font-medium text-indigo-700 disabled:opacity-50"
            disabled={retrying.has(op.pending_id)}
            onclick={() => onretry?.(op)}
          >Retry cards</button>
        {/if}
      </div>
    {/if}
  {/each}
</div>
