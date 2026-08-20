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
  <ul class="divide-y divide-slate-100 rounded-xl border border-slate-200 bg-white">
    {#each sessions as session (session.id)}
      <li>
        <button
          type="button"
          class="flex w-full items-center justify-between gap-3 px-4 py-3 text-left hover:bg-slate-50"
          onclick={() => onopen?.(session)}
        >
          <span class="font-medium text-slate-900">{session.title}</span>
          <span class="shrink-0 rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
            {session.provider}:{session.model_id}
          </span>
        </button>
      </li>
    {/each}
  </ul>
{/if}
