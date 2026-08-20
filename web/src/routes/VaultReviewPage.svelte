<!-- web/src/routes/VaultReviewPage.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api } from '../lib/api/client'
  import type { HomeResponse, ReviewItem, ReviewQueue } from '../lib/api/types'
  import { filterVaultProjects } from '../lib/vault-scope'
  import { filterQueueByProjectIds } from '../lib/review/vault-filter'

  let {
    vaultId,
    vaultName = 'Vault',
  }: {
    vaultId: string
    vaultName?: string
  } = $props()

  let loading = $state(true)
  let error = $state('')
  let queue = $state<ReviewQueue>({ scope: 'all', caught_up: true, items: [] })

  async function load() {
    loading = true
    error = ''
    try {
      const [home, full] = await Promise.all([
        api.get<HomeResponse>('/api/v1/home'),
        api.get<ReviewQueue>('/api/v1/review/queue?scope=all'),
      ])
      const projects = filterVaultProjects(home.projects, vaultId)
      const ids = new Set(projects.map((project) => project.id))
      queue = filterQueueByProjectIds(full, ids)
    } catch (cause) {
      error = cause instanceof Error ? cause.message : 'Could not load review queue.'
    } finally {
      loading = false
    }
  }

  onMount(() => {
    void load()
  })

  function itemLabel(item: ReviewItem): string {
    return item.prompt
  }
</script>

<svelte:head><title>Review · {vaultName}</title></svelte:head>

<div class="space-y-6">
  <header>
    <p class="text-sm text-slate-500">{vaultName}</p>
    <h1 class="text-2xl font-semibold">Review</h1>
  </header>

  {#if error}
    <p role="alert" class="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</p>
    <button
      type="button"
      class="rounded-md border border-slate-300 bg-white px-3 py-1.5 text-sm"
      onclick={() => load()}
    >Retry</button>
  {/if}

  {#if loading}
    <div class="space-y-3">
      <Skeleton class="h-32" />
      <Skeleton class="h-32" />
    </div>
  {:else if queue.caught_up || queue.items.length === 0}
    <p class="caught-up rounded-xl border border-dashed border-slate-300 bg-white px-6 py-12 text-center text-slate-700">
      Caught up in this vault.
    </p>
  {:else}
    <div class="space-y-4" data-testid="vault-review-runner">
      {#each queue.items as item (item.id)}
        <article class="review-card rounded-xl border border-slate-200 bg-white p-5">
          <h3 class="text-base font-semibold text-slate-950">{itemLabel(item)}</h3>
          {#if item.kind === 'bite' && item.answer}
            <p class="mt-3 text-sm text-slate-600">{item.answer}</p>
          {:else if item.note_id}
            <a
              class="mt-3 inline-block text-sm font-medium text-indigo-700"
              href={`#/projects/${encodeURIComponent(item.project_id)}/notes/${encodeURIComponent(String(item.note_id))}`}
            >Open current note</a>
          {/if}
        </article>
      {/each}
    </div>
  {/if}
</div>
