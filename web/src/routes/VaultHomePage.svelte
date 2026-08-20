<!-- web/src/routes/VaultHomePage.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import EmptyState from '../components/EmptyState.svelte'
  import ProjectCard from '../components/ProjectCard.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api } from '../lib/api/client'
  import type { HomeResponse, Project, ReviewQueue } from '../lib/api/types'
  import { filterVaultProjects } from '../lib/vault-scope'
  import { navigate, routeToHash } from '../lib/router'

  let {
    vaultId,
    vaultName = 'Vault',
  }: {
    vaultId: string
    vaultName?: string
  } = $props()

  let loading = $state(true)
  let error = $state('')
  let projects = $state<Project[]>([])
  let due = $state(0)
  let noteTotal = $state(0)
  let sessionTotal = $state(0)

  onMount(async () => {
    try {
      const [home, queue] = await Promise.all([
        api.get<HomeResponse>('/api/v1/home'),
        api.get<ReviewQueue>('/api/v1/review/queue?scope=all'),
      ])
      projects = filterVaultProjects(home.projects, vaultId)
      const ids = new Set(projects.map((project) => project.id))
      due = (queue.items ?? []).filter((item) => ids.has(item.project_id)).length
      noteTotal = projects.reduce((sum, project) => sum + (project.note_count ?? 0), 0)
      sessionTotal = projects.reduce((sum, project) => sum + (project.session_count ?? 0), 0)
    } catch (cause) {
      error = cause instanceof Error ? cause.message : 'Could not load vault dashboard.'
    } finally {
      loading = false
    }
  })

  const projectsHref = $derived(routeToHash({ name: 'vault-projects', vaultId }))
  const sessionsHref = $derived(routeToHash({ name: 'vault-sessions', vaultId }))
  const reviewHref = $derived(routeToHash({ name: 'vault-review', vaultId }))
  const newProjectHref = $derived(`${projectsHref}?new=1`)
  const projectLabel = $derived(`${projects.length} ${projects.length === 1 ? 'project' : 'projects'}`)
  const dueLabel = $derived(due === 1 ? '1 due' : `${due} due`)
</script>

<svelte:head><title>{vaultName} · Personal Agent</title></svelte:head>

<div class="page-stack">
  <header class="page-header">
    <div>
      <p class="page-header__eyebrow">Vault</p>
      <h1>{vaultName}</h1>
    </div>
    <div class="page-header__actions" aria-label="Quick actions">
      <a class="btn btn--primary" href={newProjectHref}>New project</a>
      <a class="btn btn--secondary" href={projectsHref}>Projects</a>
      <a class="btn btn--secondary" href={sessionsHref}>Sessions</a>
      <a class="btn btn--secondary" href={reviewHref}>Review</a>
    </div>
  </header>

  {#if error}
    <p role="alert" class="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</p>
  {/if}

  {#if loading}
    <div class="metric-strip" aria-busy="true">
      <Skeleton class="h-16" />
      <Skeleton class="h-16" />
      <Skeleton class="h-16" />
    </div>
  {:else}
    <section class="metric-strip" aria-label="Summary">
      <div class="metric-card" data-card="metric">
        <p class="metric-card__label">Projects</p>
        <p class="metric-card__value">{projectLabel}</p>
      </div>
      <div class="metric-card" data-card="metric">
        <p class="metric-card__label">Review</p>
        <p class="metric-card__value">{dueLabel}</p>
      </div>
      <div class="metric-card" data-card="metric">
        <p class="metric-card__label">Notes / sessions</p>
        <p class="metric-card__value">{noteTotal} notes · {sessionTotal} sessions</p>
      </div>
    </section>

    <section aria-label="Recent projects">
      <div class="section-head">
        <h2>Recent projects</h2>
        <button type="button" class="btn btn--ghost" onclick={() => navigate(projectsHref)}>
          View all
        </button>
      </div>
      {#if projects.length}
        <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {#each projects.slice(0, 6) as project (project.id)}
            <ProjectCard
              {project}
              onclick={() => navigate(`#/projects/${encodeURIComponent(project.id)}`)}
            />
          {/each}
        </div>
      {:else}
        <EmptyState
          title="No projects in this vault yet"
          description="Create a project here to keep related work together."
          actionLabel="New project"
          onaction={() => navigate(newProjectHref)}
        />
      {/if}
    </section>
  {/if}
</div>
