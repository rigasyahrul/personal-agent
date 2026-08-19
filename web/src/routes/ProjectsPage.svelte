<!-- web/src/routes/ProjectsPage.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import EmptyState from '../components/EmptyState.svelte'
  import ProjectCard from '../components/ProjectCard.svelte'
  import SearchField from '../components/SearchField.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api } from '../lib/api/client'
  import type { HomeResponse, Project } from '../lib/api/types'
  import { filterByQuery, isUnfiled } from '../lib/catalog'
  import { navigate } from '../lib/router'
  let projects = $state<Project[]>([]), query = $state(''), loading = $state(true), creating = $state(false), saving = $state(false), name = $state(''), error = $state('')
  let visible = $derived(filterByQuery(projects, query))
  onMount(async () => {
    try {
      projects = (await api.get<HomeResponse>('/api/v1/home')).projects.filter(isUnfiled)
    } catch (e) {
      error = e instanceof Error ? e.message : 'Could not load projects.'
    } finally {
      loading = false
    }
  })
  async function createProject() {
    const clean = name.trim()
    if (!clean) return
    saving = true
    error = ''
    try {
      const project = await api.post<Project>('/api/v1/projects', { name: clean, vault_id: null })
      navigate(`#/projects/${encodeURIComponent(project.id)}`)
    } catch (e) {
      error = e instanceof Error ? e.message : 'Could not create project.'
    } finally {
      saving = false
    }
  }
</script>
<svelte:head><title>Projects · Personal Agent</title></svelte:head>
<div class="space-y-6">
  <header class="flex flex-wrap items-end justify-between gap-4">
    <div><p class="text-sm text-slate-500">Global desk</p><h1 class="text-2xl font-semibold">Projects</h1></div>
    <button class="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white" onclick={() => creating = true}>New project</button>
  </header>
  <SearchField bind:value={query} label="Search projects" />
  {#if error}<p role="alert" class="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</p>{/if}
  {#if creating}
    <form class="flex max-w-lg gap-2 rounded-xl border bg-white p-4" onsubmit={(e) => { e.preventDefault(); createProject() }}>
      <label class="flex-1">
        <span class="text-sm font-medium">Project name</span>
        <input class="mt-1 w-full rounded-md border px-3 py-2" bind:value={name} />
      </label>
      <button disabled={saving || !name.trim()} class="self-end rounded-md bg-indigo-600 px-4 py-2 text-sm text-white" type="submit">Create project</button>
    </form>
  {/if}
  {#if loading}
    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3"><Skeleton class="h-32" /><Skeleton class="h-32" /></div>
  {:else if visible.length}
    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {#each visible as project (project.id)}
        <ProjectCard {project} onclick={() => navigate(`#/projects/${encodeURIComponent(project.id)}`)} />
      {/each}
    </div>
  {:else if query.trim()}
    <EmptyState title="No matching projects" description="Try a different project name." actionLabel="Clear search" onaction={() => query = ''} />
  {:else}
    <EmptyState title="No unfiled projects yet" description="Create your first project on the global desk." actionLabel="New project" onaction={() => creating = true} />
  {/if}
</div>
