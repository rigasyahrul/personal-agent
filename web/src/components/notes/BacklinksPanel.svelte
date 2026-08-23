<!-- web/src/components/notes/BacklinksPanel.svelte -->
<script lang="ts">
  export type BacklinkItem = {
    title: string
    path: string
    knowledgeId: string
    kind?: string
    sourceNoteId?: string
  }

  let {
    items,
    onopen,
  }: {
    items: BacklinkItem[]
    onopen: (item: BacklinkItem) => void
  } = $props()

  function label(item: BacklinkItem): string {
    const title = item.title.trim()
    return title || item.path
  }
</script>

<section class="panel backlinks" aria-labelledby="backlinks-heading">
  <h2 id="backlinks-heading" class="backlinks__heading">Backlinks</h2>
  {#if items.length === 0}
    <p class="backlinks__empty">No backlinks yet.</p>
  {:else}
    <ul class="backlinks__list">
      {#each items as item (`${item.knowledgeId}:${item.path}`)}
        <li>
          <button type="button" class="backlinks__item" onclick={() => onopen(item)}>
            {label(item)}
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</section>
