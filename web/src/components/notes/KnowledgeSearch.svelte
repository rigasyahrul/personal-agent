<!-- web/src/components/notes/KnowledgeSearch.svelte -->
<script lang="ts">
  import { api } from '../../lib/api'
  import type { KnowledgeSearchHit } from '../../lib/api'

  const DEBOUNCE_MS = 300

  let {
    projectId,
    onopen,
  }: {
    projectId: string
    onopen: (hit: KnowledgeSearchHit) => void
  } = $props()

  let query = $state('')
  let hits = $state<KnowledgeSearchHit[]>([])
  let error = $state('')
  let searched = $state(false)
  let seq = 0

  function label(hit: KnowledgeSearchHit): string {
    const title = hit.title.trim()
    return title || hit.path
  }

  async function runSearch(project: string, q: string) {
    const token = ++seq
    error = ''
    try {
      const next = await api.searchProject(project, q)
      if (token !== seq) return
      hits = next
      searched = true
    } catch (cause) {
      if (token !== seq) return
      hits = []
      searched = false
      error = cause instanceof Error ? cause.message : 'Could not search knowledge.'
    }
  }

  $effect(() => {
    const project = projectId
    const q = query.trim()
    if (!q) {
      seq += 1
      hits = []
      error = ''
      searched = false
      return
    }
    const timer = setTimeout(() => {
      void runSearch(project, q)
    }, DEBOUNCE_MS)
    return () => clearTimeout(timer)
  })
</script>

<section class="panel knowledge-search" aria-label="Search knowledge">
  <label class="knowledge-search__field">
    <span class="sr-only">Search knowledge</span>
    <input
      class="field-input"
      type="search"
      placeholder="Search knowledge"
      aria-label="Search knowledge"
      bind:value={query}
    />
  </label>

  {#if error}
    <p role="alert" class="alert alert--error">{error}</p>
  {:else if hits.length > 0}
    <ul class="knowledge-search__list">
      {#each hits as hit (`${hit.knowledge_id}:${hit.path}`)}
        <li>
          <button type="button" class="knowledge-search__hit" onclick={() => onopen(hit)}>
            <span class="knowledge-search__title">{label(hit)}</span>
            {#if hit.title.trim() && hit.path}
              <span class="knowledge-search__path">{hit.path}</span>
            {/if}
            {#if hit.snippet}
              <span class="knowledge-search__snippet">{hit.snippet}</span>
            {/if}
          </button>
        </li>
      {/each}
    </ul>
  {:else if searched}
    <p class="knowledge-search__empty">No matches.</p>
  {/if}
</section>
