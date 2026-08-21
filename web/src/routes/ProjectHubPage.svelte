<!-- web/src/routes/ProjectHubPage.svelte -->
<script lang="ts">
  import Breadcrumbs from '../components/Breadcrumbs.svelte'
  import ProjectRail from '../components/ProjectRail.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import SessionCardRow from '../components/sessions/SessionCardRow.svelte'
  import SessionChat from '../components/sessions/SessionChat.svelte'
  import { api } from '../lib/api'
  import type { ModelOption, Project, Session } from '../lib/api/types'
  import { formatSessionDate } from '../lib/format-session-date'
  import { workspaceEnabled } from '../lib/promote'
  import {
    readProjectRailMode,
    readProjectRailTab,
    writeProjectRailMode,
    writeProjectRailTab,
    type ProjectRailMode,
    type ProjectRailTab,
  } from '../lib/project-rail-prefs'
  import { routeToHash } from '../lib/router'
  import { randomSessionTitle } from '../lib/session-title'

  let {
    projectId,
    onProjectLoad,
  }: {
    projectId: string
    onProjectLoad?: (project: Project | null) => void
  } = $props()

  let project = $state<Project | null>(null)
  let sessions = $state<Session[]>([])
  let loading = $state(true)
  let error = $state('')
  let sessionsError = $state('')
  let draft = $state('')
  let starting = $state(false)
  let activeSession = $state<Session | null>(null)
  /** Default model for hub start (same chip as open-session composer). */
  let defaultModel = $state<ModelOption | null>(null)
  let railMode = $state<ProjectRailMode>(readProjectRailMode(localStorage))
  let railTab = $state<ProjectRailTab>(readProjectRailTab(localStorage))
  /** Rail → SessionChat file tab bridge (cleared by SessionChat after open). */
  let openFileRequest = $state<{
    path: string
    source: 'project-note' | 'workspace'
    noteId?: string
  } | null>(null)

  async function loadDefaultModel() {
    try {
      const { models } = await api.listModels()
      defaultModel = models?.[0] ?? null
    } catch {
      defaultModel = null
    }
  }

  async function load() {
    loading = true
    error = ''
    sessionsError = ''
    activeSession = null
    try {
      const loadedProject = await api.getProject(projectId)
      project = loadedProject
      onProjectLoad?.(loadedProject)
    } catch (cause) {
      project = null
      onProjectLoad?.(null)
      sessions = []
      error = cause instanceof Error ? cause.message : 'Could not load project.'
      loading = false
      return
    }

    void loadDefaultModel()

    try {
      const listed = await api.listProjectSessions(projectId)
      sessions = listed ?? []
      sessionsError = ''
    } catch (cause) {
      sessions = []
      sessionsError =
        cause instanceof Error ? cause.message : 'Could not load sessions.'
    } finally {
      loading = false
    }
  }

  async function reloadSessions() {
    try {
      const listed = await api.listProjectSessions(projectId)
      sessions = listed ?? []
      sessionsError = ''
    } catch (cause) {
      sessionsError =
        cause instanceof Error ? cause.message : 'Could not load sessions.'
    }
  }

  $effect(() => {
    void projectId
    void load()
  })

  const notesHref = $derived(routeToHash({ name: 'notes', projectId }))
  const reviewHref = $derived(routeToHash({ name: 'project-review', projectId }))

  const workspaceFilesEnabled = $derived(
    activeSession ? workspaceEnabled(activeSession) : false,
  )

  function openSession(session: Session) {
    activeSession = session
  }

  function closeSession() {
    activeSession = null
    openFileRequest = null
    void reloadSessions()
  }

  async function renameSession(session: Session) {
    const next = window.prompt('Rename session', session.title)
    if (next == null) return
    const title = next.trim()
    if (!title || title === session.title) return
    try {
      const updated = await api.renameSession(session.id, title)
      sessions = sessions.map((s) =>
        s.id === session.id ? { ...s, title: updated?.title ?? title } : s,
      )
    } catch (cause) {
      error = cause instanceof Error ? cause.message : 'Could not rename session.'
    }
  }

  async function deleteSession(session: Session) {
    if (!window.confirm(`Delete “${session.title}”? This cannot be undone.`)) return
    try {
      await api.deleteSession(session.id)
      sessions = sessions.filter((s) => s.id !== session.id)
      if (activeSession?.id === session.id) {
        activeSession = null
        openFileRequest = null
      }
    } catch (cause) {
      error = cause instanceof Error ? cause.message : 'Could not delete session.'
    }
  }

  const hubModelLabel = $derived(
    defaultModel ? `${defaultModel.provider}:${defaultModel.model_id}` : '',
  )

  async function startSession(e: Event) {
    e.preventDefault()
    const content = draft.trim()
    if (!content || starting) return
    starting = true
    error = ''
    try {
      let m = defaultModel
      if (!m) {
        const { models } = await api.listModels()
        m = models?.[0] ?? null
        defaultModel = m
      }
      if (!m) {
        throw new Error('Configure a model in Settings before starting a session.')
      }
      const session = await api.createProjectSession(projectId, {
        home: 'project',
        title: randomSessionTitle(),
        provider: m.provider,
        model_id: m.model_id,
        model_parameters: {},
        tool_grants: { workspace_files: false },
      })
      // Always surface the created session so a failed first send cannot orphan it
      // or cause a second create on retry.
      sessions = [session, ...sessions.filter((x) => x.id !== session.id)]
      activeSession = session
      try {
        await api.sendMessage(session.id, { content, request_key: crypto.randomUUID() })
        draft = ''
      } catch (sendCause) {
        error =
          sendCause instanceof Error ? sendCause.message : 'Could not send first message.'
        // Keep draft so user can resend from the open chat composer.
      }
    } catch (cause) {
      error = cause instanceof Error ? cause.message : 'Could not start session.'
    } finally {
      starting = false
    }
  }

  /** Enter submits (create session); Shift+Enter inserts a newline. */
  function onComposerKeydown(e: KeyboardEvent) {
    if (e.key !== 'Enter' || e.shiftKey) return
    if (e.isComposing) return
    e.preventDefault()
    const form = (e.currentTarget as HTMLTextAreaElement).form
    if (!form || starting || !draft.trim()) return
    form.requestSubmit()
  }
</script>

{#if loading}
  <div class="page-stack" aria-busy="true">
    <Skeleton class="h-6 w-48" />
    <Skeleton class="h-10 w-72" />
    <Skeleton class="h-24" />
  </div>
{:else if error && !project}
  <div class="page-stack">
    <p role="alert" class="alert alert--error">{error}</p>
    <div>
      <button type="button" class="btn btn--secondary" onclick={() => void load()}>Retry</button>
    </div>
  </div>
{:else if project}
  <div class="project-workspace" data-rail={railMode}>
    <div class="project-workspace__main" hidden={railMode === 'expanded'}>
      {#if activeSession}
        {#key activeSession.id}
          <SessionChat
            session={activeSession}
            {projectId}
            onclose={closeSession}
            embeddedInHub={true}
            bind:openFileRequest
          />
        {/key}
      {:else}
        <header class="hub-header">
          <div class="hub-header__lead">
            <Breadcrumbs {project} />
            <h1 class="hub-header__title">{project.name}</h1>
          </div>
          <nav class="hub-header__links" aria-label="Project links">
            <a class="link-accent" href={notesHref}>Notes</a>
            <a class="link-accent" href={reviewHref}>Review</a>
          </nav>
        </header>

        <section class="hub-start">
          <form class="hub-composer" onsubmit={startSession}>
            <div class="session-composer__card hub-composer__card">
              <textarea
                class="session-composer__input"
                bind:value={draft}
                aria-label="Message"
                placeholder="How can I help you today?"
                rows="3"
                onkeydown={onComposerKeydown}
              ></textarea>
              <div class="session-composer__toolbar">
                {#if hubModelLabel}
                  <span class="session-composer__model">{hubModelLabel}</span>
                {:else}
                  <span class="session-composer__model session-composer__model--muted"
                  >No model</span>
                {/if}
                <button
                  type="submit"
                  class="session-composer__send btn btn--primary"
                  disabled={starting || !draft.trim()}
                  aria-label="Send"
                >Send</button>
              </div>
            </div>
          </form>
          {#if error}
            <p role="alert" class="alert alert--error" style="margin-top:12px">{error}</p>
          {/if}
        </section>

        <section class="hub-session-list" aria-label="Sessions">
          {#if sessionsError}
            <p role="alert" class="alert alert--error">{sessionsError}</p>
            <div style="margin-bottom:8px">
              <button
                type="button"
                class="btn btn--secondary"
                onclick={() => void reloadSessions()}
              >Retry sessions</button>
            </div>
          {/if}
          {#if sessions.length > 0}
            <h2 class="hub-session-list__label">Recent</h2>
          {/if}
          {#each sessions as s (s.id)}
            <SessionCardRow
              variant="list"
              title={s.title}
              dateLabel={formatSessionDate(s.created_at) ?? ''}
              onclick={() => openSession(s)}
              onrename={() => void renameSession(s)}
              ondelete={() => void deleteSession(s)}
            />
          {/each}
          {#if !sessions.length && !sessionsError}
            <p class="text-sm text-muted" style="margin:0">
              No sessions yet. Send a message above to start one.
            </p>
          {/if}
        </section>
      {/if}
    </div>
    <aside class="project-workspace__rail">
      <ProjectRail
        {projectId}
        sessionId={activeSession?.id}
        workspaceFilesEnabled={workspaceFilesEnabled}
        tab={railTab}
        mode={railMode}
        onTabChange={(tab) => {
          writeProjectRailTab(localStorage, tab)
          railTab = tab
        }}
        onModeChange={(mode) => {
          writeProjectRailMode(localStorage, mode)
          railMode = mode
        }}
        onOpenFile={(path, meta) => {
          if (!activeSession) return
          openFileRequest = {
            path,
            source: meta?.source ?? 'workspace',
            noteId: meta?.noteId,
          }
        }}
      />
    </aside>
  </div>
{/if}
