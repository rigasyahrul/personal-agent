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
<div class="page-stack">
  <header class="page-header">
    <div><h1>Projects</h1></div>
    <div class="page-header__actions">
      <button type="button" class="btn btn--primary" onclick={() => creating = true}>New project</button>
    </div>
  </header>
  <SearchField bind:value={query} label="Search projects" />
  {#if error}<p role="alert" class="alert alert--error">{error}</p>{/if}
  {#if creating}
    <form class="panel form-inline" onsubmit={(e) => { e.preventDefault(); createProject() }}>
      <label>
        Project name
        <input class="field-input" bind:value={name} aria-label="Project name" />
      </label>
      <button disabled={saving || !name.trim()} class="btn btn--primary" type="submit">Create project</button>
    </form>
  {/if}
  {#if loading}
    <div class="catalog-grid" aria-busy="true"><Skeleton class="h-28" /><Skeleton class="h-28" /></div>
  {:else if visible.length}
    <div class="catalog-grid">
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
