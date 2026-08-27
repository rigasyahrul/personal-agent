<!-- web/src/components/ProjectRail.svelte -->
<script lang="ts">
  import { api } from '../lib/api'
  import type { NoteTreeEntry, WorkspaceEntry } from '../lib/api/types'
  import type { ProjectRailMode, ProjectRailTab } from '../lib/project-rail-prefs'
  import type { ThoughtsView } from '../lib/thoughts'
  import { buildHierarchy, flattenTree, type TreeNode } from '../lib/workspace-tree'
  import { railIconPath, type RailIconName } from './rail-icons'
  import InstructionEditor from './settings/InstructionEditor.svelte'
  import Skeleton from './Skeleton.svelte'

  export type OpenFileMeta = {
    source: 'project-note' | 'workspace'
    noteId?: string
  }

  let {
    projectId,
    sessionId = null,
    workspaceFilesEnabled = false,
    tab: controlledTab,
    mode,
    onTabChange,
    onModeChange,
    onOpenFile,
    thoughts = null,
    onCloseThoughts,
  }: {
    projectId: string
    sessionId?: string | null
    workspaceFilesEnabled?: boolean
    tab?: ProjectRailTab
    mode?: ProjectRailMode
    onTabChange?: (tab: ProjectRailTab) => void
    onModeChange?: (mode: ProjectRailMode) => void
    onOpenFile?: (path: string, meta?: OpenFileMeta) => void
    thoughts?: ThoughtsView | null
    onCloseThoughts?: () => void
  } = $props()

  type RailEntry = WorkspaceEntry & { note_id?: string }

  let localTab = $state<ProjectRailTab>('config')
  const activeTab = $derived(controlledTab ?? localTab)
  const activeMode = $derived(mode ?? 'open')
  const thoughtsOpen = $derived(Boolean(thoughts))

  let projectEntries = $state<RailEntry[]>([])
  let workspaceEntries = $state<WorkspaceEntry[]>([])
  /** path → note_id for project-note file rows (hierarchy drops note_id). */
  let noteIdByPath = $state<Record<string, string>>({})
  let loading = $state(false)
  let error = $state('')
  let loadToken = 0

  function noteKindToWorkspace(kind: NoteTreeEntry['kind']): WorkspaceEntry['kind'] {
    if (kind === 'folder' || kind === 'directory') return 'directory'
    return 'file'
  }

  function notesToWorkspace(entries: NoteTreeEntry[]): RailEntry[] {
    return entries.map((entry) => ({
      path: entry.path,
      kind: noteKindToWorkspace(entry.kind),
      note_id: entry.note_id,
    }))
  }

  async function loadFiles() {
    const token = ++loadToken
    loading = true
    error = ''
    try {
      const notes = await api.listProjectNotes(projectId)
      if (token !== loadToken) return
      const mapped = notesToWorkspace(notes ?? [])
      projectEntries = mapped
      const ids: Record<string, string> = {}
      for (const entry of mapped) {
        if (entry.kind === 'file' && entry.note_id) ids[entry.path] = entry.note_id
      }
      noteIdByPath = ids

      if (sessionId && workspaceFilesEnabled) {
        const tree = await api.workspaceTree(sessionId)
        if (token !== loadToken) return
        workspaceEntries = tree?.entries ?? []
      } else {
        workspaceEntries = []
      }
    } catch (cause) {
      if (token !== loadToken) return
      error = cause instanceof Error ? cause.message : 'Unable to load project files.'
      projectEntries = []
      workspaceEntries = []
      noteIdByPath = {}
    } finally {
      if (token === loadToken) loading = false
    }
  }

  $effect(() => {
    void projectId
    void sessionId
    void workspaceFilesEnabled
    if (activeTab === 'files') void loadFiles()
  })

  function selectTab(next: ProjectRailTab) {
    if (controlledTab === undefined) localTab = next
    onTabChange?.(next)
  }

  const projectRows = $derived(flattenTree(buildHierarchy(projectEntries)))
  const workspaceRows = $derived(flattenTree(buildHierarchy(workspaceEntries)))
  const hasAnyFiles = $derived(projectRows.length > 0 || workspaceRows.length > 0)

  function onProjectRowClick(path: string, kind: TreeNode['kind']) {
    if (kind === 'directory') return
    if (!sessionId) return
    const noteId = noteIdByPath[path]
    onOpenFile?.(path, {
      source: 'project-note',
      ...(noteId ? { noteId } : {}),
    })
  }

  function onWorkspaceRowClick(path: string, kind: TreeNode['kind']) {
    if (kind === 'directory') return
    if (!sessionId) return
    onOpenFile?.(path, { source: 'workspace' })
  }
</script>

{#snippet icon(name: RailIconName)}
  <svg
    viewBox="0 0 24 24"
    aria-hidden="true"
    fill="none"
    stroke="currentColor"
    stroke-width="1.8"
    stroke-linecap="round"
    stroke-linejoin="round"
  >
    <path d={railIconPath(name)}></path>
  </svg>
{/snippet}

<div
  class="project-rail"
  class:project-rail--collapsed={activeMode === 'collapsed'}
  data-rail-mode={activeMode}
  data-thoughts={thoughtsOpen ? '1' : undefined}
>
  {#if activeMode === 'collapsed'}
    <div class="rail-collapsed-chrome" data-testid="rail-collapsed-chrome">
      <button
        type="button"
        class="rail-icon"
        aria-label="Show canvas"
        title="Show canvas"
        onclick={() => onModeChange?.('open')}
      >{@render icon('show-canvas')}</button>
    </div>
  {:else}
    <div class="rail-iconbar" role="toolbar" aria-label="Project rail" hidden={thoughtsOpen}>
      <div class="rail-iconbar__group" role="tablist" aria-label="Project rail panels" hidden={thoughtsOpen}>
        <button
          type="button"
          role="tab"
          id="rail-tab-config"
          class="rail-icon {activeTab === 'config' ? 'rail-icon--active' : ''}"
          aria-label="Config"
          title="Config"
          aria-selected={activeTab === 'config'}
          aria-controls="rail-panel-config"
          tabindex={activeTab === 'config' ? 0 : -1}
          onclick={() => selectTab('config')}
        >{@render icon('config')}</button>
        <button
          type="button"
          role="tab"
          id="rail-tab-files"
          class="rail-icon {activeTab === 'files' ? 'rail-icon--active' : ''}"
          aria-label="Files"
          title="Files"
          aria-selected={activeTab === 'files'}
          aria-controls="rail-panel-files"
          tabindex={activeTab === 'files' ? 0 : -1}
          onclick={() => selectTab('files')}
        >{@render icon('files')}</button>
      </div>
      <div class="rail-iconbar__group">
        <button
          type="button"
          class="rail-icon"
          aria-label={activeMode === 'expanded' ? 'Exit expanded' : 'Expand workspace'}
          title={activeMode === 'expanded' ? 'Exit expanded' : 'Expand workspace'}
          aria-pressed={activeMode === 'expanded'}
          onclick={() => onModeChange?.(activeMode === 'expanded' ? 'open' : 'expanded')}
        >{@render icon('expand-workspace')}</button>
        <button
          type="button"
          class="rail-icon"
          aria-label="Collapse canvas"
          title="Collapse canvas"
          onclick={() => onModeChange?.('collapsed')}
        >{@render icon('collapse-canvas')}</button>
      </div>
    </div>

    {#if thoughts}
      <section class="thoughts-rail" data-thoughts="1">
        <header class="thoughts-rail__header">
          <h2 class="thoughts-rail__title">Thoughts</h2>
          <button
            type="button"
            class="thoughts-rail__close"
            aria-label="Close thoughts"
            onclick={() => onCloseThoughts?.()}
          >×</button>
        </header>
        {#if thoughts.rows.length === 0}
          <p class="thoughts-rail__empty">Working…</p>
        {:else}
          <ol class="thoughts-rail__list">
            {#each thoughts.rows as row (row.id)}
              <li class="thought-row" data-status={row.status}>
                <span class="thought-row__verb">{row.verb}</span>
                {#if row.arg}<span class="thought-row__arg">{row.arg}</span>{/if}
              </li>
            {/each}
          </ol>
        {/if}
      </section>
    {/if}

    {#if activeTab === 'config'}
      <div
        class="rail-panel rail-panel--config"
        role="tabpanel"
        id="rail-panel-config"
        aria-labelledby="rail-tab-config"
        hidden={thoughtsOpen}
      >
        <InstructionEditor scope="project" {projectId} variant="rail" />
      </div>
    {:else}
      <div
        class="rail-panel form-stack"
        role="tabpanel"
        id="rail-panel-files"
        aria-labelledby="rail-tab-files"
        hidden={thoughtsOpen}
      >
        {#if loading}
          <div class="space-y-2" aria-busy="true">
            <Skeleton class="h-6" />
            <Skeleton class="h-6" />
            <Skeleton class="h-6" />
          </div>
        {:else if error}
          <p role="alert" class="alert alert--error">{error}</p>
        {:else if !hasAnyFiles}
          <p class="text-sm text-slate-500" style="margin:0">No project files available.</p>
        {:else}
          {#if projectRows.length}
            <div class="workspace-tree space-y-0.5 text-sm" aria-label="Project notes">
              {#each projectRows as row (row.path)}
                <button
                  type="button"
                  class="tree-item {row.kind === 'directory' ? 'text-slate-500' : ''}"
                  style="padding-left: {8 + row.depth * 12}px"
                  disabled={row.kind === 'directory'}
                  onclick={() => onProjectRowClick(row.path, row.kind)}
                >{row.path}</button>
              {/each}
            </div>
          {/if}
          {#if workspaceRows.length}
            <div class="form-stack" style="gap:6px">
              <p class="text-sm font-medium text-slate-700" style="margin:0">Workspace</p>
              <div class="workspace-tree space-y-0.5 text-sm" aria-label="Workspace files">
                {#each workspaceRows as row (row.path)}
                  <button
                    type="button"
                    class="tree-item {row.kind === 'directory' ? 'text-slate-500' : ''}"
                    style="padding-left: {8 + row.depth * 12}px"
                    disabled={row.kind === 'directory'}
                    onclick={() => onWorkspaceRowClick(row.path, row.kind)}
                  >{row.path}</button>
                {/each}
              </div>
            </div>
          {/if}
        {/if}
      </div>
    {/if}
  {/if}
</div>
