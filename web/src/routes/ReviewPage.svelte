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

<div class="page-stack">
  <header class="page-header">
    <div><h1>Review</h1></div>
  </header>

  <ReviewRunner {scope} {projectScopes} showScopeChips={true} />
</div>
