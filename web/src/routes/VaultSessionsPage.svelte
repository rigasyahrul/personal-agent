<!-- web/src/routes/VaultSessionsPage.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import EmptyState from '../components/EmptyState.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api } from '../lib/api/client'
  import type { HomeResponse, Project, Session } from '../lib/api/types'
  import { navigate } from '../lib/router'
  import { loadVaultSessions, type VaultSession } from '../lib/vault-sessions'

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
  let sessions = $state<VaultSession[]>([])
  let failures = $state<string[]>([])
  let selectedProjectId = $state('__all__')

  onMount(async () => {
    try {
      const result = await loadVaultSessions(vaultId, {
        listProjects: async () => {
          const home = await api.get<HomeResponse>('/api/v1/home')
          return home.projects
        },
        listProjectSessions: async (projectId: string) => {
          return api.get<Session[]>(`/api/v1/projects/${encodeURIComponent(projectId)}/sessions`)
        },
      })
      projects = result.projects
      sessions = result.sessions
      failures = result.failures
    } catch (cause) {
      error = cause instanceof Error ? cause.message : 'Could not load sessions.'
    } finally {
      loading = false
    }
  })

  let displaySessions = $derived(
    selectedProjectId === '__all__'
      ? sessions
      : sessions.filter((session) => session.project_id === selectedProjectId),
  )

  const newSessionHref = $derived(
    selectedProjectId !== '__all__'
      ? `#/projects/${encodeURIComponent(selectedProjectId)}/sessions`
      : projects[0]
        ? `#/projects/${encodeURIComponent(projects[0].id)}/sessions`
        : `#/vaults/${encodeURIComponent(vaultId)}/projects?new=1`,
  )
</script>

<svelte:head><title>Sessions · {vaultName}</title></svelte:head>

<div class="space-y-6">
  <header class="flex flex-wrap items-end justify-between gap-4">
    <div>
      <p class="text-sm text-slate-500">{vaultName}</p>
      <h1 class="text-2xl font-semibold">Sessions</h1>
    </div>
    {#if projects.length}
      <a
        class="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white"
        href={newSessionHref}
      >New session</a>
    {/if}
  </header>

  {#if error}
    <p role="alert" class="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</p>
  {/if}

  {#if failures.length}
    <p role="alert" class="rounded-md bg-amber-50 p-3 text-sm text-amber-900">
      Could not load sessions for: {failures.join(', ')}
    </p>
  {/if}

  {#if loading}
    <div class="space-y-3">
      <Skeleton class="h-12" />
      <Skeleton class="h-24" />
    </div>
  {:else if !projects.length}
    <EmptyState
      title="Create a project first"
      description="Sessions live on projects. Add a project to this vault, then start a session."
      actionLabel="New project"
      onaction={() => navigate(`#/vaults/${encodeURIComponent(vaultId)}/projects?new=1`)}
    />
  {:else}
    <label class="block max-w-xs">
      <span class="text-sm font-medium text-slate-700">Project</span>
      <select
        class="mt-1 w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm"
        bind:value={selectedProjectId}
        aria-label="Project"
      >
        <option value="__all__">All projects</option>
        {#each projects as project (project.id)}
          <option value={project.id}>{project.name}</option>
        {/each}
      </select>
    </label>

    {#if displaySessions.length}
      <ul class="divide-y divide-slate-200 rounded-xl border border-slate-200 bg-white">
        {#each displaySessions as session (session.id)}
          <li class="flex flex-wrap items-center justify-between gap-3 px-4 py-3">
            <div>
              <p class="font-medium text-slate-950">{session.title || 'Untitled session'}</p>
              <p class="text-sm text-slate-500">{session.project_name}</p>
            </div>
            <a
              class="text-sm font-medium text-indigo-700"
              href={`#/projects/${encodeURIComponent(session.project_id)}/sessions`}
            >Open</a>
          </li>
        {/each}
      </ul>
    {:else}
      <EmptyState
        title="No sessions yet"
        description="Start a session from a project in this vault."
        actionLabel="New session"
        onaction={() => navigate(newSessionHref)}
      />
    {/if}
  {/if}
</div>
