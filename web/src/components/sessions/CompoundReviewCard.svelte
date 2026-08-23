<!-- web/src/components/sessions/CompoundReviewCard.svelte -->
<script lang="ts">
  import type { CompoundItem, CompoundProposal } from '../../lib/api/types'

  let {
    proposal,
    onconfirm,
    oncancel,
    busy = false,
  }: {
    proposal: CompoundProposal
    onconfirm: (decision: 'approve' | 'reject', items: CompoundItem[]) => void
    oncancel?: () => void
    busy?: boolean
  } = $props()

  let syncedId = $state<string | null>(null)
  let items = $state<CompoundItem[]>([])

  $effect.pre(() => {
    const next = proposal
    if (syncedId === next.id) return
    syncedId = next.id
    items = next.items.map((item) => ({ ...item }))
  })

  function snapshot(): CompoundItem[] {
    return items.map((item) => ({ ...item }))
  }

  function confirm(decision: 'approve' | 'reject') {
    onconfirm(decision, snapshot())
  }
</script>

<section class="panel compound-card" aria-labelledby="compound-review-title">
  <header class="compound-card__head">
    <h2 id="compound-review-title" class="compound-card__title">Compound review</h2>
  </header>

  {#if items.length === 0}
    <p class="compound-card__empty">No items</p>
  {:else}
    <ul class="compound-card__items">
      {#each items as item, i (`${item.path}:${i}`)}
        <li class="compound-item">
          <div class="compound-item__meta">
            <span class="compound-item__kind">{item.kind}</span>
            <span class="compound-item__path">{item.path}</span>
          </div>
          <label class="compound-item__field">
            Content
            <textarea
              class="field-textarea"
              aria-label={`Content for ${item.path}`}
              bind:value={items[i].content}
              disabled={busy}
            ></textarea>
          </label>
        </li>
      {/each}
    </ul>
  {/if}

  <div class="compound-card__actions">
    {#if oncancel}
      <button type="button" class="btn btn--ghost" disabled={busy} onclick={() => oncancel()}>
        Cancel
      </button>
    {/if}
    <button type="button" class="btn btn--danger" disabled={busy} onclick={() => confirm('reject')}>
      Reject
    </button>
    <button type="button" class="btn btn--primary" disabled={busy} onclick={() => confirm('approve')}>
      Approve
    </button>
  </div>
</section>
