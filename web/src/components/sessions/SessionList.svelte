<!-- web/src/components/sessions/SessionList.svelte -->
<script lang="ts">
  import type { Session } from '../../lib/api/types'

  let {
    sessions,
    onopen,
  }: {
    sessions: Session[]
    onopen?: (session: Session) => void
  } = $props()
</script>

{#if sessions.length === 0}
  <p class="text-sm text-slate-600">No sessions yet</p>
{:else}
  <ul class="list-panel">
    {#each sessions as session (session.id)}
      <li>
        <button type="button" class="list-row" onclick={() => onopen?.(session)}>
          <span class="font-medium text-slate-900">{session.title}</span>
          <span class="badge-chip" style="background:#f4f4f5;color:#52525b">
            {session.provider}:{session.model_id}
          </span>
        </button>
      </li>
    {/each}
  </ul>
{/if}
