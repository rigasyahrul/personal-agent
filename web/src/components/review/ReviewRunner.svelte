<!-- web/src/components/review/ReviewRunner.svelte -->
<script lang="ts">
  import Skeleton from '../Skeleton.svelte'
  import { api, APIError } from '../../lib/api'
  import type { ReviewItem, ReviewQueue } from '../../lib/api/types'
  import { routeToHash } from '../../lib/router'

  export type ReviewScope = 'all' | `project:${string}`

  type ProjectScopeOption = { scope: `project:${string}`; label: string }

  let {
    scope,
    showScopeChips = true,
    projectScopes = [] as ProjectScopeOption[],
    loadQueue,
    now = () => performance.now(),
    uuid = () => crypto.randomUUID(),
  }: {
    scope: ReviewScope | string
    showScopeChips?: boolean
    projectScopes?: ProjectScopeOption[]
    loadQueue?: () => Promise<ReviewQueue>
    now?: () => number
    uuid?: () => string
  } = $props()

  let loading = $state(true)
  let error = $state('')
  let actionError = $state('')
  let queue = $state<ReviewQueue>({ scope: 'all', caught_up: true, items: [] })
  let actionPending = $state(false)
  let revealed = $state<Record<string, boolean>>({})
  let shownAt = $state(0)
  let loadGeneration = 0

  const activeItem = $derived(queue.items[0] as ReviewItem | undefined)
  const caughtUpLabel = $derived(
    scope === 'all' ? 'Caught up in all projects.' : 'Caught up in this project.',
  )

  async function refresh() {
    const generation = ++loadGeneration
    loading = true
    error = ''
    try {
      const next = loadQueue
        ? await loadQueue()
        : await api.getReviewQueue(scope)
      if (generation !== loadGeneration) return
      queue = next
      revealed = {}
      shownAt = now()
      actionError = ''
    } catch (cause) {
      if (generation !== loadGeneration) return
      error = cause instanceof Error ? cause.message : 'Could not load review queue.'
    } finally {
      if (generation === loadGeneration) loading = false
    }
  }

  $effect(() => {
    void scope
    void loadQueue
    void refresh()
  })

  function chipActive(value: string): boolean {
    return value === scope
  }

  function selectScope(value: string) {
    if (value === scope) return
    // Preserve contract string `scope=` for Go web_test and hash routing.
    location.hash = routeToHash({ name: 'review', scope: value })
  }

  async function runAction(fn: () => Promise<unknown>) {
    if (actionPending || !activeItem) return
    actionPending = true
    actionError = ''
    try {
      await fn()
      await refresh()
    } catch (cause) {
      if (cause instanceof APIError && cause.status === 409) {
        await refresh()
        return
      }
      actionError = cause instanceof Error ? cause.message : 'Action failed.'
    } finally {
      actionPending = false
    }
  }

  function rate(rating: 'again' | 'hard' | 'good' | 'easy') {
    const item = activeItem
    if (!item) return
    const duration_ms = Math.max(0, Math.round(now() - shownAt))
    const payload = {
      rating,
      request_key: uuid(),
      row_version: item.row_version ?? 0,
      duration_ms,
    }
    return runAction(() => api.rateReviewItem(item.id, payload))
  }

  function suspend() {
    const item = activeItem
    if (!item) return
    return runAction(() => api.suspendReviewItem(item.id))
  }

  function noteHref(item: ReviewItem): string {
    const noteId = item.note_id != null ? String(item.note_id) : ''
    return `#/projects/${encodeURIComponent(item.project_id)}/notes/${encodeURIComponent(noteId)}`
  }

  const ratingLabels: Array<{ value: 'again' | 'hard' | 'good' | 'easy'; label: string }> = [
    { value: 'again', label: 'Again' },
    { value: 'hard', label: 'Hard' },
    { value: 'good', label: 'Good' },
    { value: 'easy', label: 'Easy' },
  ]
</script>

<div class="space-y-4" data-testid="review-runner">
  {#if showScopeChips}
    <nav class="flex flex-wrap gap-2" aria-label="Review scope">
      <button
        type="button"
        class="scope-chip"
        class:scope-chip--active={chipActive('all')}
        disabled={chipActive('all')}
        onclick={() => selectScope('all')}
      >All projects</button>
      {#each projectScopes as option (option.scope)}
        <button
          type="button"
          class="scope-chip"
          class:scope-chip--active={chipActive(option.scope)}
          disabled={chipActive(option.scope)}
          onclick={() => selectScope(option.scope)}
        >{option.label}</button>
      {/each}
    </nav>
  {/if}

  {#if error}
    <p role="alert" class="alert alert--error">{error}</p>
    <button type="button" class="btn btn--secondary" onclick={() => refresh()}>Retry</button>
  {:else if loading}
    <div class="space-y-3">
      <Skeleton class="h-32" />
      <Skeleton class="h-24" />
    </div>
  {:else if queue.caught_up || !activeItem}
    <p class="caught-up panel panel--dashed px-6 py-12 text-center text-slate-700">
      {caughtUpLabel}
    </p>
  {:else}
    <article class="review-card panel panel--pad">
      <h3 class="text-base font-semibold text-slate-950">{activeItem.prompt}</h3>

      {#if activeItem.kind === 'bite'}
        {#if revealed[activeItem.id]}
          <p class="answer mt-3 text-sm text-slate-700">{activeItem.answer ?? ''}</p>
        {:else}
          <button
            type="button"
            class="btn btn--secondary mt-3"
            onclick={() => {
              revealed = { ...revealed, [activeItem.id]: true }
            }}
          >Reveal answer</button>
        {/if}
      {:else if activeItem.note_id}
        <a class="btn btn--secondary mt-3" href={noteHref(activeItem)}>Open current note</a>
      {/if}

      <div class="ratings mt-5 flex flex-wrap gap-2">
        {#each ratingLabels as rating (rating.value)}
          <button
            type="button"
            class="btn btn--primary"
            disabled={actionPending}
            onclick={() => rate(rating.value)}
          >{rating.label}</button>
        {/each}
        <button
          type="button"
          class="btn btn--secondary"
          disabled={actionPending}
          onclick={() => suspend()}
        >Suspend</button>
      </div>

      {#if actionError}
        <p role="alert" class="mt-3 text-sm text-red-700">{actionError}</p>
      {/if}
    </article>
  {/if}
</div>
