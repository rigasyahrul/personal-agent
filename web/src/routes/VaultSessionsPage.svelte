<!-- web/src/routes/VaultSessionsPage.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import EmptyState from '../components/EmptyState.svelte'
  import SessionCardRow from '../components/sessions/SessionCardRow.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api } from '../lib/api/client'
  import type { HomeResponse, Project, Session } from '../lib/api/types'
  import { formatRelativeTime } from '../lib/format-relative-time'
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

  function sessionMeta(session: VaultSession): string {
    const model = `${session.provider}:${session.model_id}`
    const parts = [session.project_name, model]
    const rel = formatRelativeTime(session.updated_at ?? session.created_at)
    if (rel) parts.push(rel)
    return parts.join(' · ')
  }
</script>

<svelte:head><title>Sessions · {vaultName}</title></svelte:head>

<div class="page-stack">
  <header class="page-header">
    <div>
      <p class="page-header__eyebrow">{vaultName}</p>
      <h1>Sessions</h1>
    </div>
    {#if projects.length}
      <div class="page-header__actions">
        <a class="btn btn--primary" href={newSessionHref}>New session</a>
      </div>
    {/if}
  </header>

  {#if error}
    <p role="alert" class="alert alert--error">{error}</p>
  {/if}

  {#if failures.length}
    <p role="alert" class="alert alert--warn">
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
      <select class="field-select mt-1" bind:value={selectedProjectId} aria-label="Project">
        <option value="__all__">All projects</option>
        {#each projects as project (project.id)}
          <option value={project.id}>{project.name}</option>
        {/each}
      </select>
    </label>

    {#if displaySessions.length}
      <ul class="flex flex-col gap-2">
        {#each displaySessions as session (session.id)}
          <li>
            <SessionCardRow
              title={session.title || 'Untitled session'}
              meta={sessionMeta(session)}
              href={`#/projects/${encodeURIComponent(session.project_id)}/sessions`}
            />
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
