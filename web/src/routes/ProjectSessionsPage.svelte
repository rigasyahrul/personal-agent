<!-- web/src/routes/ProjectSessionsPage.svelte -->
<script lang="ts">
  import Breadcrumbs from '../components/Breadcrumbs.svelte'
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
  <section class="space-y-3 rounded-xl border border-slate-200 bg-white p-4" data-session-open={activeSession.id}>
    <button type="button" class="text-sm font-medium text-indigo-700" onclick={closeSession}>Sessions</button>
    <h2 class="text-xl font-semibold">{activeSession.title}</h2>
    <p class="text-sm text-slate-600">{activeSession.provider}:{activeSession.model_id}</p>
    <!-- SessionChat mounts here in Task 44 -->
  </section>
{:else}
  <div class="space-y-6">
    <header>
      <h1 class="text-2xl font-semibold">Sessions</h1>
    </header>

    {#if error}
      <p role="alert" class="rounded-md bg-red-50 p-3 text-sm text-red-700">{error}</p>
      <button type="button" class="rounded-md border px-3 py-1.5 text-sm" onclick={() => void load()}>Retry</button>
    {:else if loading}
      <div class="space-y-3" aria-busy="true">
        <Skeleton class="h-24" />
        <Skeleton class="h-16" />
      </div>
    {:else}
      {#if models.length}
        <form
          class="grid max-w-xl gap-3 rounded-xl border border-slate-200 bg-white p-4"
          onsubmit={createSession}
        >
          <label class="block text-sm">
            <span class="font-medium">Title</span>
            <input
              class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
              name="title"
              maxlength="200"
              required
              bind:value={title}
            />
          </label>
          <label class="block text-sm">
            <span class="font-medium">Model</span>
            <select
              class="mt-1 w-full rounded-md border border-slate-300 px-3 py-2"
              name="model"
              required
              bind:value={modelValue}
            >
              {#each models as model (model.provider + model.model_id)}
                <option value={`${model.provider}\u0000${model.model_id}`}>
                  {model.provider}:{model.model_id}
                </option>
              {/each}
            </select>
          </label>
          <label class="flex items-center gap-2 text-sm">
            <input type="checkbox" name="workspace_files" bind:checked={workspaceFiles} />
            Allow workspace files
          </label>
          {#if createError}
            <p role="alert" class="text-sm text-red-700">{createError}</p>
          {/if}
          <button
            type="submit"
            class="rounded-md bg-indigo-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
            disabled={creating}
          >New session</button>
        </form>
      {:else}
        <section class="rounded-xl border border-dashed border-slate-300 bg-white p-6">
          <p>Configure a model before creating a session.</p>
          <a class="mt-3 inline-block text-sm font-medium text-indigo-700" href="#/settings">Open settings</a>
        </section>
      {/if}

      <SessionList {sessions} onopen={openSession} />
    {/if}
  </div>
{/if}
