<!-- web/src/routes/ProjectHubPage.svelte -->
<script lang="ts">
  import Breadcrumbs from '../components/Breadcrumbs.svelte'
  import ProjectRail from '../components/ProjectRail.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import SessionCardRow from '../components/sessions/SessionCardRow.svelte'
  import SessionChat from '../components/sessions/SessionChat.svelte'
  import { api } from '../lib/api'
  import type { Project, Session } from '../lib/api/types'
  import { formatRelativeTime } from '../lib/format-relative-time'
  import { workspaceEnabled } from '../lib/promote'
  import { routeToHash } from '../lib/router'

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
  let draft = $state('')
  let starting = $state(false)
  let activeSession = $state<Session | null>(null)
  /** Rail → SessionChat file tab bridge (cleared by SessionChat after open). */
  let fileToOpen = $state<string | null>(null)

  async function load() {
    loading = true
    error = ''
    activeSession = null
    try {
      const [loadedProject, listed] = await Promise.all([
        api.getProject(projectId),
        api.listProjectSessions(projectId),
      ])
      project = loadedProject
      onProjectLoad?.(loadedProject)
      sessions = listed ?? []
    } catch (cause) {
      project = null
      onProjectLoad?.(null)
      sessions = []
      error = cause instanceof Error ? cause.message : 'Could not load project.'
    } finally {
      loading = false
    }
  }

  async function reloadSessions() {
    try {
      const listed = await api.listProjectSessions(projectId)
      sessions = listed ?? []
    } catch {
      /* keep existing list on soft reload failure */
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

  function sessionMeta(session: Session): string {
    const model = `${session.provider}:${session.model_id}`
    const rel = formatRelativeTime(session.updated_at ?? session.created_at)
    return rel ? `${model} · ${rel}` : model
  }

  function openSession(session: Session) {
    activeSession = session
  }

  function closeSession() {
    activeSession = null
    fileToOpen = null
    void reloadSessions()
  }

  async function startSession(e: Event) {
    e.preventDefault()
    const content = draft.trim()
    if (!content || starting) return
    starting = true
    error = ''
    try {
      const { models } = await api.listModels()
      if (!models?.length) {
        throw new Error('Configure a model in Settings before starting a session.')
      }
      const m = models[0]
      const session = await api.createProjectSession(projectId, {
        home: 'project',
        title: content.slice(0, 80) || 'Untitled',
        provider: m.provider,
        model_id: m.model_id,
        model_parameters: {},
        tool_grants: { workspace_files: false },
      })
      await api.sendMessage(session.id, { content, request_key: crypto.randomUUID() })
      draft = ''
      sessions = [session, ...sessions.filter((x) => x.id !== session.id)]
      activeSession = session
    } catch (cause) {
      error = cause instanceof Error ? cause.message : 'Could not start session.'
    } finally {
      starting = false
    }
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
  <div class="project-workspace">
    <div class="project-workspace__main">
      {#if activeSession}
        {#key activeSession.id}
          <SessionChat
            session={activeSession}
            {projectId}
            onclose={closeSession}
            embeddedInHub={true}
            bind:openPath={fileToOpen}
          />
        {/key}
      {:else}
        <header class="hub-header name-row">
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
          <h1 class="hub-start__title">How can I help you today?</h1>
          <form class="hub-composer" onsubmit={startSession}>
            <textarea
              class="field-textarea"
              bind:value={draft}
              aria-label="Message"
              rows="4"
            ></textarea>
            <div class="hub-composer__row">
              <button
                class="btn btn--primary"
                type="submit"
                disabled={starting || !draft.trim()}
              >Send</button>
            </div>
          </form>
          {#if error}
            <p role="alert" class="alert alert--error" style="margin-top:12px">{error}</p>
          {/if}
        </section>

        <section class="hub-session-list" aria-label="Sessions">
          {#each sessions as s (s.id)}
            <SessionCardRow
              title={s.title}
              meta={sessionMeta(s)}
              onclick={() => openSession(s)}
            />
          {/each}
          {#if !sessions.length}
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
        onOpenFile={(path) => {
          fileToOpen = path
        }}
      />
    </aside>
  </div>
{/if}
