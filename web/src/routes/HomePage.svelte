<!-- web/src/routes/HomePage.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import EmptyState from '../components/EmptyState.svelte'
  import ProjectCard from '../components/ProjectCard.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api } from '../lib/api/client'
  import type { HomeResponse, Project } from '../lib/api/types'
  import { isUnfiled } from '../lib/catalog'
  import { navigate } from '../lib/router'

  let loading = $state(true)
  let error = $state('')
  let dueCount = $state(0)
  let projects = $state<Project[]>([])

  onMount(async () => {
    try {
      const data = await api.get<HomeResponse>('/api/v1/home')
      dueCount = data.due_count ?? 0
      projects = data.projects.filter(isUnfiled).slice(0, 6)
    } catch (cause) {
      error = cause instanceof Error ? cause.message : 'Could not load your dashboard.'
    } finally {
      loading = false
    }
  })

  const dueLabel = $derived(
    dueCount ? `${dueCount} ${dueCount === 1 ? 'item' : 'items'} due` : 'You’re all caught up',
  )
</script>

<svelte:head><title>Home · Personal Agent</title></svelte:head>

<div class="page-stack">
  <header class="page-header">
    <div>
      <h1>Home</h1>
    </div>
    <div class="page-header__actions" aria-label="Quick actions">
      <button type="button" class="btn btn--primary" onclick={() => navigate('#/projects')}>
        New project
      </button>
      <button type="button" class="btn btn--secondary" onclick={() => navigate('#/vaults')}>
        New vault
      </button>
    </div>
  </header>

  {#if error}
    <p role="alert" class="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</p>
  {/if}

  {#if loading}
    <div class="metric-strip" aria-busy="true">
      <Skeleton class="h-16" />
      <Skeleton class="h-16" />
    </div>
  {:else}
    <section class="metric-strip" aria-label="Summary">
      <div class="metric-card" data-card="metric">
        <p class="metric-card__label">Review</p>
        <p class="metric-card__value">{dueLabel}</p>
      </div>
      <div class="metric-card" data-card="metric">
        <p class="metric-card__label">Unfiled projects</p>
        <p class="metric-card__value">{projects.length}</p>
      </div>
    </section>

    <section aria-label="Recent projects">
      <div class="section-head">
        <h2>Recent projects</h2>
        <button type="button" class="btn btn--ghost" onclick={() => navigate('#/projects')}>
          View all
        </button>
      </div>
      {#if projects.length}
        <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {#each projects as project (project.id)}
            <ProjectCard
              {project}
              onclick={() => navigate(`#/projects/${encodeURIComponent(project.id)}`)}
            />
          {/each}
        </div>
      {:else}
        <EmptyState
          title="No unfiled projects yet"
          description="Create a project on your global desk, or organize work inside a vault."
          actionLabel="New project"
          onaction={() => navigate('#/projects')}
        />
      {/if}
    </section>
  {/if}
</div>
