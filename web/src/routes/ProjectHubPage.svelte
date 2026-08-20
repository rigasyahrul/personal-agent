<!-- web/src/routes/ProjectHubPage.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
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

  onMount(() => {
    void load()
  })

  const notesHref = $derived(routeToHash({ name: 'notes', projectId }))
  const sessionsHref = $derived(routeToHash({ name: 'sessions', projectId }))
  const reviewHref = $derived(routeToHash({ name: 'project-review', projectId }))
</script>

{#if loading}
  <div class="space-y-4" aria-busy="true">
    <Skeleton class="h-6 w-48" />
    <Skeleton class="h-10 w-72" />
    <div class="grid gap-4 md:grid-cols-3">
      <Skeleton class="h-24" />
      <Skeleton class="h-24" />
      <Skeleton class="h-24" />
    </div>
  </div>
{:else if error}
  <div class="space-y-3">
    <p role="alert" class="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</p>
    <button
      type="button"
      class="rounded-md border border-slate-300 bg-white px-3 py-1.5 text-sm font-medium"
      onclick={() => void load()}
    >Retry</button>
  </div>
{:else if project}
  <div class="space-y-8">
    <div class="space-y-3">
      <Breadcrumbs {project} />
      <header class="flex flex-wrap items-center gap-3">
        <h1 class="text-2xl font-semibold text-slate-950">{project.name}</h1>
        {#if project.vault_name}
          <Badge text={project.vault_name} />
        {/if}
      </header>
    </div>

    <section class="grid gap-4 md:grid-cols-3" aria-label="Project metrics">
      <article class="rounded-xl border border-slate-200 bg-white p-5">
        <p class="text-sm text-slate-500">Notes</p>
        <p class="mt-2 text-2xl font-semibold">{project.note_count ?? 0}</p>
      </article>
      <article class="rounded-xl border border-slate-200 bg-white p-5">
        <p class="text-sm text-slate-500">Sessions</p>
        <p class="mt-2 text-2xl font-semibold">{project.session_count ?? 0}</p>
      </article>
      <article class="rounded-xl border border-slate-200 bg-white p-5">
        <p class="text-sm text-slate-500">Due</p>
        <p class="mt-2 text-2xl font-semibold">{project.due_count ?? 0}</p>
      </article>
    </section>

    <section class="grid gap-4 md:grid-cols-3" aria-label="Project surfaces">
      <a
        class="rounded-xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-indigo-300"
        href={notesHref}
      >
        <h2 class="text-lg font-semibold text-indigo-700">Notes</h2>
        <p class="mt-1 text-sm text-slate-600">Browse and read source notes.</p>
      </a>
      <a
        class="rounded-xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-indigo-300"
        href={sessionsHref}
      >
        <h2 class="text-lg font-semibold text-indigo-700">Sessions</h2>
        <p class="mt-1 text-sm text-slate-600">Chat with tools in this project.</p>
      </a>
      <a
        class="rounded-xl border border-slate-200 bg-white p-5 shadow-sm transition hover:border-indigo-300"
        href={reviewHref}
      >
        <h2 class="text-lg font-semibold text-indigo-700">Review</h2>
        <p class="mt-1 text-sm text-slate-600">Work through due cards.</p>
      </a>
    </section>
  </div>
{/if}
