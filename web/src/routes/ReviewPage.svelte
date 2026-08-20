<!-- web/src/routes/ReviewPage.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import ReviewRunner from '../components/review/ReviewRunner.svelte'
  import { api } from '../lib/api'
  import type { Project } from '../lib/api/types'
  import { routeToHash } from '../lib/router'

  let {
    query = new URLSearchParams(),
  }: {
    query?: URLSearchParams
  } = $props()

  let projectScopes = $state<Array<{ scope: `project:${string}`; label: string }>>([])

  const rawScope = $derived(query.get('scope'))
  const scope = $derived(rawScope && rawScope.length > 0 ? rawScope : 'all')

  onMount(() => {
    // Default missing scope to all via hash (legacy parity).
    if (rawScope === null || rawScope === '') {
      location.hash = routeToHash({ name: 'review', scope: 'all' })
    }
    void api
      .listProjects()
      .then((projects: Project[]) => {
        projectScopes = projects.map((project) => ({
          scope: `project:${project.id}` as const,
          label: project.name,
        }))
      })
      .catch(() => {
        projectScopes = []
      })
  })
</script>

<svelte:head><title>Review · Personal Agent</title></svelte:head>

<div class="space-y-6">
  <header>
    <p class="text-sm text-slate-500">Global desk</p>
    <h1 class="text-2xl font-semibold text-slate-950">Review</h1>
  </header>

  <ReviewRunner {scope} {projectScopes} showScopeChips={true} />
</div>
