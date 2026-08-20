<!-- web/src/routes/ProjectReviewPage.svelte -->
<script lang="ts">
  import Breadcrumbs from '../components/Breadcrumbs.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import ReviewRunner from '../components/review/ReviewRunner.svelte'
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

  const scope = $derived(`project:${projectId}`)

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

<svelte:head><title>Review · Personal Agent</title></svelte:head>

{#if loading}
  <Skeleton class="h-24" />
{:else if error}
  <p role="alert" class="alert alert--error">{error}</p>
{:else if project}
  <div class="page-stack">
    <Breadcrumbs {project} leaf="Review" />
    <header class="page-header" style="margin-bottom: 0">
      <div><h1>Review</h1></div>
    </header>
    <ReviewRunner
      {scope}
      showScopeChips={true}
      projectScopes={[{ scope: `project:${projectId}`, label: 'This project' }]}
    />
  </div>
{/if}
