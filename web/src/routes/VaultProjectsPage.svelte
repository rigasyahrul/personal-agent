<!-- web/src/routes/VaultProjectsPage.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import EmptyState from '../components/EmptyState.svelte'
  import ProjectCard from '../components/ProjectCard.svelte'
  import SearchField from '../components/SearchField.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api } from '../lib/api/client'
  import type { HomeResponse, Project } from '../lib/api/types'
  import { filterByQuery } from '../lib/catalog'
  import { navigate } from '../lib/router'
  import { createVaultProjectInput, filterVaultProjects } from '../lib/vault-scope'

  let {
    vaultId,
    vaultName = 'Vault',
  }: {
    vaultId: string
    vaultName?: string
  } = $props()

  let projects = $state<Project[]>([])
  let query = $state('')
  let loading = $state(true)
  let creating = $state(false)
  let saving = $state(false)
  let name = $state('')
  let error = $state('')

  let visible = $derived(filterByQuery(projects, query))

  onMount(async () => {
    try {
      const home = await api.get<HomeResponse>('/api/v1/home')
      projects = filterVaultProjects(home.projects, vaultId)
      if (typeof location !== 'undefined' && location.hash.includes('new=1')) {
        creating = true
      }
    } catch (e) {
      error = e instanceof Error ? e.message : 'Could not load projects.'
    } finally {
      loading = false
    }
  })

  async function createProject() {
    const payload = createVaultProjectInput(name, vaultId)
    if (!payload.name) return
    saving = true
    error = ''
    try {
      const project = await api.post<Project>('/api/v1/projects', payload)
      navigate(`#/projects/${encodeURIComponent(project.id)}`)
    } catch (e) {
      error = e instanceof Error ? e.message : 'Could not create project.'
    } finally {
      saving = false
    }
  }
</script>

<svelte:head><title>Projects · {vaultName}</title></svelte:head>

<div class="page-stack">
  <header class="page-header">
    <div>
      <p class="page-header__eyebrow">{vaultName}</p>
      <h1>Projects</h1>
    </div>
    <div class="page-header__actions">
      <button class="btn btn--primary" type="button" onclick={() => (creating = true)}>New project</button>
    </div>
  </header>

  <SearchField bind:value={query} label="Search projects" />

  {#if error}
    <p role="alert" class="alert alert--error">{error}</p>
  {/if}

  {#if creating}
    <form
      class="panel form-inline"
      onsubmit={(e) => {
        e.preventDefault()
        createProject()
      }}
    >
      <label>
        Vault
        <input class="field-input" value={vaultName} disabled aria-label="Vault" />
      </label>
      <label>
        Project name
        <input class="field-input" bind:value={name} aria-label="Project name" />
      </label>
      <button disabled={saving || !name.trim()} class="btn btn--primary" type="submit">Create project</button>
    </form>
  {/if}

  {#if loading}
    <div class="catalog-grid" aria-busy="true">
      <Skeleton class="h-28" />
      <Skeleton class="h-28" />
    </div>
  {:else if visible.length}
    <div class="catalog-grid">
      {#each visible as project (project.id)}
        <ProjectCard
          {project}
          onclick={() => navigate(`#/projects/${encodeURIComponent(project.id)}`)}
        />
      {/each}
    </div>
  {:else if query.trim()}
    <EmptyState
      title="No matching projects"
      description="Try a different project name."
      actionLabel="Clear search"
      onaction={() => (query = '')}
    />
  {:else}
    <EmptyState
      title="No projects in this vault yet"
      description="Create a project locked to this vault."
      actionLabel="New project"
      onaction={() => (creating = true)}
    />
  {/if}
</div>
