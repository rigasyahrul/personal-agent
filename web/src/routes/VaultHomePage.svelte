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

<div class="space-y-8">
  <header>
    <p class="text-sm text-slate-500">Vault</p>
    <h1 class="text-2xl font-semibold text-slate-950">{vaultName}</h1>
  </header>

  <section aria-label="Quick actions" class="flex flex-wrap gap-3">
    <a
      class="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white"
      href={newProjectHref}
    >New project</a>
    <a
      class="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-medium"
      href={projectsHref}
    >Projects</a>
    <a
      class="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-medium"
      href={sessionsHref}
    >Sessions</a>
    <a
      class="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-medium"
      href={reviewHref}
    >Review</a>
  </section>

  {#if error}
    <p role="alert" class="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</p>
  {/if}

  {#if loading}
    <div class="grid gap-4 md:grid-cols-3">
      <Skeleton class="h-28" />
      <Skeleton class="h-28" />
      <Skeleton class="h-28" />
    </div>
  {:else}
    <section class="grid gap-4 md:grid-cols-3">
      <div class="rounded-xl border border-slate-200 bg-white p-5">
        <p class="text-sm text-slate-500">Projects</p>
        <p class="mt-2 text-xl font-semibold">{projectLabel}</p>
      </div>
      <div class="rounded-xl border border-slate-200 bg-white p-5">
        <p class="text-sm text-slate-500">Review</p>
        <p class="mt-2 text-xl font-semibold">{dueLabel}</p>
      </div>
      <div class="rounded-xl border border-slate-200 bg-white p-5">
        <p class="text-sm text-slate-500">Notes / sessions</p>
        <p class="mt-2 text-xl font-semibold">{noteTotal} notes · {sessionTotal} sessions</p>
      </div>
    </section>

    <section class="space-y-4">
      <div class="flex items-center justify-between">
        <h2 class="text-lg font-semibold">Recent projects</h2>
        <button class="text-sm font-medium text-indigo-700" type="button" onclick={() => navigate(projectsHref)}>
          View all
        </button>
      </div>
      {#if projects.length}
        <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
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
