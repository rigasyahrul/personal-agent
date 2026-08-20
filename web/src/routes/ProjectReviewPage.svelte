<!-- web/src/routes/ProjectReviewPage.svelte — Phase F surface; stub keeps project vault shell wiring -->
<script lang="ts">
    import Breadcrumbs from '../components/Breadcrumbs.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api } from '../lib/api'
  import type { Project } from '../lib/api/types'

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
      error = cause instanceof Error ? cause.message : 'Could not load project.'
      onProjectLoad?.(null)
    } finally {
      loading = false
    }
  }

  $effect(() => {
    void projectId
    void load()
  })
</script>

{#if loading}
  <Skeleton class="h-24" />
{:else if error}
  <p role="alert" class="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</p>
{:else if project}
  <div class="space-y-4">
    <Breadcrumbs {project} leaf="Review" />
    <h1 class="text-2xl font-semibold">Review</h1>
    <p class="text-sm text-slate-600">Project review arrives in a later phase.</p>
  </div>
{/if}
