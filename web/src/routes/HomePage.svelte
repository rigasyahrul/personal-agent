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
  let loading = $state(true), error = $state(''), dueCount = $state(0), projects = $state<Project[]>([])
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
</script>
<svelte:head><title>Home · Personal Agent</title></svelte:head>
<div class="space-y-8">
  <header><p class="text-sm text-slate-500">Global desk</p><h1 class="text-2xl font-semibold text-slate-950">Home</h1></header>
  <section aria-label="Quick actions" class="flex flex-wrap gap-3">
    <button class="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white" onclick={() => navigate('#/projects')}>New project</button>
    <button class="rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-medium" onclick={() => navigate('#/vaults')}>New vault</button>
  </section>
  {#if error}<p role="alert" class="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</p>{/if}
  {#if loading}
    <div class="grid gap-4 md:grid-cols-3"><Skeleton class="h-28" /><Skeleton class="h-28" /><Skeleton class="h-28" /></div>
  {:else}
    <section class="grid gap-4 md:grid-cols-2">
      <div class="rounded-xl border border-slate-200 bg-white p-5"><p class="text-sm text-slate-500">Review</p><p class="mt-2 text-xl font-semibold">{dueCount ? `${dueCount} ${dueCount === 1 ? 'item' : 'items'} due` : 'You’re all caught up'}</p></div>
      <div class="rounded-xl border border-slate-200 bg-white p-5"><p class="text-sm text-slate-500">Unfiled projects</p><p class="mt-2 text-xl font-semibold">{projects.length}</p></div>
    </section>
    <section class="space-y-4"><div class="flex items-center justify-between"><h2 class="text-lg font-semibold">Recent projects</h2><button class="text-sm font-medium text-indigo-700" onclick={() => navigate('#/projects')}>View all</button></div>
      {#if projects.length}<div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">{#each projects as project (project.id)}<ProjectCard {project} onclick={() => navigate(`#/projects/${encodeURIComponent(project.id)}`)} />{/each}</div>
      {:else}<EmptyState title="No unfiled projects yet" description="Create a project on your global desk, or organize work inside a vault." actionLabel="New project" onaction={() => navigate('#/projects')} />{/if}
    </section>
  {/if}
</div>
