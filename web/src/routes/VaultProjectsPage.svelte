<!-- web/src/routes/VaultProjectsPage.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import EmptyState from '../components/EmptyState.svelte'
  import Modal from '../components/Modal.svelte'
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
  let createError = $state('')

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

  function openCreate() {
    creating = true
    name = ''
    createError = ''
  }

  function closeCreate() {
    creating = false
    name = ''
    createError = ''
  }

  async function createProject() {
    const payload = createVaultProjectInput(name, vaultId)
    if (!payload.name) return
    saving = true
    createError = ''
    try {
      const project = await api.post<Project>('/api/v1/projects', payload)
      closeCreate()
      navigate(`#/projects/${encodeURIComponent(project.id)}`)
    } catch (e) {
      createError = e instanceof Error ? e.message : 'Could not create project.'
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
      <button class="btn btn--primary" type="button" onclick={openCreate}>New project</button>
    </div>
  </header>

  <SearchField bind:value={query} label="Search projects" />

  {#if error}
    <p role="alert" class="alert alert--error">{error}</p>
  {/if}

  <Modal open={creating} title="New project" onclose={closeCreate}>
    <form
      class="form-stack"
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
      {#if createError}
        <p role="alert" class="alert alert--error">{createError}</p>
      {/if}
      <div class="flex justify-end gap-2">
        <button type="button" class="btn btn--secondary" onclick={closeCreate}>Cancel</button>
        <button disabled={saving || !name.trim()} class="btn btn--primary" type="submit">Create project</button>
      </div>
    </form>
  </Modal>

  {#if loading}
    <ul class="name-list" aria-busy="true">
      <li><Skeleton class="h-11" /></li>
      <li><Skeleton class="h-11" /></li>
    </ul>
  {:else if visible.length}
    <ul class="name-list" role="list">
      {#each visible as project (project.id)}
        <li>
          <button
            type="button"
            class="name-row"
            onclick={() => navigate(`#/projects/${encodeURIComponent(project.id)}`)}
          >
            <span class="name-row__title">{project.name}</span>
            <span class="name-row__meta">
              {project.note_count}
              {project.note_count === 1 ? 'note' : 'notes'}
            </span>
            <span class="name-row__chevron" aria-hidden="true">→</span>
          </button>
        </li>
      {/each}
    </ul>
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
      onaction={openCreate}
    />
  {/if}
</div>
