<!-- web/src/components/sessions/SessionList.svelte -->
<script lang="ts">
  import type { Session } from '../../lib/api/types'
  import { formatRelativeTime } from '../../lib/format-relative-time'
  import SessionCardRow from './SessionCardRow.svelte'

  let {
    sessions,
    onopen,
  }: {
    sessions: Session[]
    onopen?: (session: Session) => void
  } = $props()

  function sessionMeta(session: Session): string {
    const model = `${session.provider}:${session.model_id}`
    const rel = formatRelativeTime(session.updated_at ?? session.created_at)
    return rel ? `${model} · ${rel}` : model
  }
</script>

{#if sessions.length === 0}
  <p class="text-sm text-slate-600">No sessions yet</p>
{:else}
  <ul class="flex flex-col gap-2">
    {#each sessions as session (session.id)}
      <li>
        <SessionCardRow
          title={session.title}
          meta={sessionMeta(session)}
          onclick={() => onopen?.(session)}
        />
      </li>
    {/each}
  </ul>
{/if}
