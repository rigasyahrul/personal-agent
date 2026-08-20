<!-- web/src/routes/ProjectHubPage.svelte -->
<script lang="ts">
  import Badge from '../components/Badge.svelte'
  import Breadcrumbs from '../components/Breadcrumbs.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api } from '../lib/api'
  import type { Project } from '../lib/api/types'
  import { routeToHash } from '../lib/router'

  let {
    projectId,
    onProjectLoad,
  }: {
    projectId: string
    onProjectLoad?: (project: Project | null) => void
  } = $props()

  let project = $state<Project | null>(null)
  let loading = $state(true)
  let error = $state('')

  async function load() {
    loading = true
    error = ''
    try {
      project = await api.getProject(projectId)
      onProjectLoad?.(project)
    } catch (cause) {
      project = null
      onProjectLoad?.(null)
      error = cause instanceof Error ? cause.message : 'Could not load project.'
    } finally {
      loading = false
    }
  }

  $effect(() => {
    void projectId
    void load()
  })

  const notesHref = $derived(routeToHash({ name: 'notes', projectId }))
  const sessionsHref = $derived(routeToHash({ name: 'sessions', projectId }))
  const reviewHref = $derived(routeToHash({ name: 'project-review', projectId }))
</script>

{#if loading}
  <div class="page-stack" aria-busy="true">
    <Skeleton class="h-6 w-48" />
    <Skeleton class="h-10 w-72" />
    <div class="metric-strip">
      <Skeleton class="h-16" />
      <Skeleton class="h-16" />
      <Skeleton class="h-16" />
    </div>
  </div>
{:else if error}
  <div class="page-stack">
    <p role="alert" class="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</p>
    <div>
      <button type="button" class="btn btn--secondary" onclick={() => void load()}>Retry</button>
    </div>
  </div>
{:else if project}
  <div class="page-stack">
    <div class="space-y-3">
      <Breadcrumbs {project} />
      <header class="page-header" style="margin-bottom: 0">
        <div class="flex flex-wrap items-center gap-3">
          <h1>{project.name}</h1>
          {#if project.vault_name}
            <Badge text={project.vault_name} />
          {/if}
        </div>
      </header>
    </div>

    <section class="metric-strip" aria-label="Project metrics">
      <article class="metric-card" data-card="metric">
        <p class="metric-card__label">Notes</p>
        <p class="metric-card__value">{project.note_count ?? 0}</p>
      </article>
      <article class="metric-card" data-card="metric">
        <p class="metric-card__label">Sessions</p>
        <p class="metric-card__value">{project.session_count ?? 0}</p>
      </article>
      <article class="metric-card" data-card="metric">
        <p class="metric-card__label">Due</p>
        <p class="metric-card__value">{project.due_count ?? 0}</p>
      </article>
    </section>

    <section class="destination-grid" aria-label="Project surfaces">
      <a class="destination-card" data-card="destination" href={notesHref}>
        <h2 class="destination-card__title">Notes</h2>
        <p class="destination-card__body">Browse and read source notes.</p>
        <span class="destination-card__cta">Open →</span>
      </a>
      <a class="destination-card" data-card="destination" href={sessionsHref}>
        <h2 class="destination-card__title">Sessions</h2>
        <p class="destination-card__body">Chat with tools in this project.</p>
        <span class="destination-card__cta">Open →</span>
      </a>
      <a class="destination-card" data-card="destination" href={reviewHref}>
        <h2 class="destination-card__title">Review</h2>
        <p class="destination-card__body">Work through due cards.</p>
        <span class="destination-card__cta">Open →</span>
      </a>
    </section>
  </div>
{/if}
