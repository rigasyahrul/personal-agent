<!-- web/src/routes/ProjectSessionsPage.svelte -->
<script lang="ts">
  import Breadcrumbs from '../components/Breadcrumbs.svelte'
  import SessionChat from '../components/sessions/SessionChat.svelte'
  import SessionList from '../components/sessions/SessionList.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api } from '../lib/api'
  import type { ModelOption, Project, Session } from '../lib/api/types'

  let {
    projectId,
    onProjectLoad,
  }: {
    projectId: string
    onProjectLoad?: (project: Project | null) => void
  } = $props()

  let project = $state<Project | null>(null)
  let models = $state<ModelOption[]>([])
  let sessions = $state<Session[]>([])
  let loading = $state(true)
  let error = $state('')
  let createError = $state('')
  let creating = $state(false)
  let title = $state('')
  let modelValue = $state('')
  let workspaceFiles = $state(false)
  let activeSession = $state<Session | null>(null)

  $effect(() => {
    void projectId
    void load()
  })

  async function load() {
    loading = true
    error = ''
    activeSession = null
    try {
      const [loadedProject, modelResponse, listed] = await Promise.all([
        api.getProject(projectId),
        api.listModels(),
        api.listProjectSessions(projectId),
      ])
      project = loadedProject
      onProjectLoad?.(loadedProject)
      models = modelResponse?.models ?? []
      sessions = listed ?? []
      if (models.length && !modelValue) {
        modelValue = `${models[0].provider}\u0000${models[0].model_id}`
      }
    } catch (cause) {
      project = null
      onProjectLoad?.(null)
      error = cause instanceof Error ? cause.message : 'Could not load sessions.'
    } finally {
      loading = false
    }
  }

  async function createSession(event: Event) {
    event.preventDefault()
    if (creating || !models.length) return
    const [provider, model_id] = modelValue.split('\u0000')
    if (!provider || !model_id) return
    creating = true
    createError = ''
    try {
      const created = await api.createProjectSession(projectId, {
        home: 'project',
        title: title.trim() || 'Untitled',
        provider,
        model_id,
        model_parameters: {},
        tool_grants: { workspace_files: workspaceFiles },
      })
      title = ''
      workspaceFiles = false
      activeSession = created
      sessions = [created, ...sessions.filter((item) => item.id !== created.id)]
    } catch (cause) {
      createError = cause instanceof Error ? cause.message : 'Could not create session.'
    } finally {
      creating = false
    }
  }

  function openSession(session: Session) {
    activeSession = session
  }

  function closeSession() {
    activeSession = null
    void load()
  }
</script>

{#if project}
  <div class="mb-4">
    <Breadcrumbs {project} leaf="Sessions" />
  </div>
{/if}

{#if activeSession}
  <div class="content-canvas--session-focus">
    {#key activeSession.id}
      <SessionChat session={activeSession} {projectId} onclose={closeSession} />
    {/key}
  </div>
{:else}
  <div class="page-stack">
    <header class="page-header">
      <div><h1>Sessions</h1></div>
    </header>

    {#if error}
      <p role="alert" class="alert alert--error">{error}</p>
      <button type="button" class="btn btn--secondary" onclick={() => void load()}>Retry</button>
    {:else if loading}
      <div class="space-y-3" aria-busy="true">
        <Skeleton class="h-24" />
        <Skeleton class="h-16" />
      </div>
    {:else}
      {#if models.length}
        <form class="panel form-stack max-w-xl" onsubmit={createSession}>
          <label>
            Title
            <input
              class="field-input"
              name="title"
              maxlength="200"
              required
              bind:value={title}
            />
          </label>
          <label>
            Model
            <select class="field-select" name="model" required bind:value={modelValue}>
              {#each models as model (model.provider + model.model_id)}
                <option value={`${model.provider}\u0000${model.model_id}`}>
                  {model.provider}:{model.model_id}
                </option>
              {/each}
            </select>
          </label>
          <label class="flex items-center gap-2 text-sm font-medium text-slate-700" style="display:flex">
            <input type="checkbox" name="workspace_files" bind:checked={workspaceFiles} />
            Allow workspace files
          </label>
          {#if createError}
            <p role="alert" class="text-sm text-red-700">{createError}</p>
          {/if}
          <div>
            <button type="submit" class="btn btn--primary" disabled={creating}>New session</button>
          </div>
        </form>
      {:else}
        <section class="panel panel--dashed">
          <p style="margin:0">Configure a model before creating a session.</p>
          <a class="link-accent mt-3 inline-block" href="#/settings">Open settings</a>
        </section>
      {/if}

      <SessionList {sessions} onopen={openSession} />
    {/if}
  </div>
{/if}
