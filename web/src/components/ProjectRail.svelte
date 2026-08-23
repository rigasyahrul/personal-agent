<!-- web/src/components/ProjectRail.svelte -->
<script lang="ts">
  import { api } from '../lib/api'
  import type { NoteTreeEntry, WorkspaceEntry } from '../lib/api/types'
  import { buildHierarchy, flattenTree, type TreeNode } from '../lib/workspace-tree'
  import Skeleton from './Skeleton.svelte'

  export type OpenFileMeta = {
    source: 'project-note' | 'workspace'
    noteId?: string
  }

  let {
    projectId,
    sessionId = null,
    workspaceFilesEnabled = false,
    onOpenFile,
  }: {
    projectId: string
    sessionId?: string | null
    workspaceFilesEnabled?: boolean
    onOpenFile?: (path: string, meta?: OpenFileMeta) => void
  } = $props()

  type RailTab = 'memory' | 'files'
  type RailEntry = WorkspaceEntry & { note_id?: string }

  const MEMORY_PREVIEW_LINES = 12

  let tab = $state<RailTab>('memory')
  let instructions = $state('')

  let projectEntries = $state<RailEntry[]>([])
  let workspaceEntries = $state<WorkspaceEntry[]>([])
  /** path → note_id for project-note file rows (hierarchy drops note_id). */
  let noteIdByPath = $state<Record<string, string>>({})
  let loading = $state(false)
  let error = $state('')
  let loadToken = 0

  let lessonsContent = $state('')
  let memoryLoading = $state(false)
  let memoryError = $state('')
  let memoryLoadToken = 0

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

  function firstNonEmptyLines(content: string, limit = MEMORY_PREVIEW_LINES): string {
    return content
      .split('\n')
      .filter((line) => line.trim() !== '')
      .slice(0, limit)
      .join('\n')
  }

  async function loadLessons() {
    const token = ++memoryLoadToken
    memoryLoading = true
    memoryError = ''
    try {
      const next = await api.getProjectMemoryLessons(projectId)
      if (token !== memoryLoadToken) return
      lessonsContent = next?.content ?? ''
    } catch (cause) {
      if (token !== memoryLoadToken) return
      memoryError = cause instanceof Error ? cause.message : 'Unable to load memory.'
      lessonsContent = ''
    } finally {
      if (token === memoryLoadToken) memoryLoading = false
    }
  }

  $effect(() => {
    void projectId
    void sessionId
    void workspaceFilesEnabled
    if (tab === 'files') {
      void loadFiles()
    }
    if (tab === 'memory') {
      void loadLessons()
    }
  })

  function selectTab(next: RailTab) {
    tab = next
  }

  const projectRows = $derived(flattenTree(buildHierarchy(projectEntries)))
  const workspaceRows = $derived(flattenTree(buildHierarchy(workspaceEntries)))
  const hasAnyFiles = $derived(projectRows.length > 0 || workspaceRows.length > 0)
  const lessonsPreview = $derived(firstNonEmptyLines(lessonsContent))
  const canOpenMemory = $derived(Boolean(sessionId && onOpenFile))

  async function openMemory() {
    if (!sessionId || !onOpenFile) return
    let noteId: string | undefined
    try {
      const notes = await api.listProjectNotes(projectId)
      const hit = (notes ?? []).find((entry) => entry.path === 'memory/lessons.md' && entry.note_id)
      if (hit?.note_id) noteId = hit.note_id
    } catch {
      // P2 will hook knowledge/read; still open by path.
    }
    onOpenFile('memory/lessons.md', {
      source: 'project-note',
      ...(noteId ? { noteId } : {}),
    })
  }

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

<div class="project-rail">
  <div class="rail-tabs" role="tablist" aria-label="Project rail">
    <button
      type="button"
      role="tab"
      id="rail-tab-memory"
      class="rail-tab {tab === 'memory' ? 'rail-tab--active' : ''}"
      aria-selected={tab === 'memory'}
      aria-controls="rail-panel-memory"
      tabindex={tab === 'memory' ? 0 : -1}
      onclick={() => selectTab('memory')}
    >Memory</button>
    <button
      type="button"
      role="tab"
      id="rail-tab-files"
      class="rail-tab {tab === 'files' ? 'rail-tab--active' : ''}"
      aria-selected={tab === 'files'}
      aria-controls="rail-panel-files"
      tabindex={tab === 'files' ? 0 : -1}
      onclick={() => selectTab('files')}
    >Files</button>
  </div>

  {#if tab === 'memory'}
    <div class="rail-panel form-stack" role="tabpanel" id="rail-panel-memory">
      {#if memoryLoading}
        <div class="space-y-2" aria-busy="true">
          <Skeleton class="h-6" />
          <Skeleton class="h-6" />
          <Skeleton class="h-6" />
        </div>
      {:else if memoryError}
        <p role="alert" class="alert alert--error">{memoryError}</p>
      {:else if !lessonsPreview}
        <p class="text-sm text-muted" style="margin:0">No lessons yet.</p>
      {:else}
        <pre
          class="text-sm rail-memory-preview"
          data-testid="memory-lessons-preview"
        >{lessonsPreview}</pre>
      {/if}
      {#if canOpenMemory && !memoryLoading}
        <button type="button" class="btn btn--ghost" onclick={() => void openMemory()}>
          Open memory
        </button>
      {/if}
      <label class="block text-sm" for="rail-instructions">
        Instructions (system)
        <textarea
          id="rail-instructions"
          class="field-textarea mt-1"
          aria-label="Instructions (system)"
          bind:value={instructions}
          rows="6"
        ></textarea>
      </label>
      <p class="text-sm text-muted" style="margin:0">
        Not saved yet — persistence coming later.
      </p>
    </div>
  {:else}
    <div class="rail-panel form-stack" role="tabpanel" id="rail-panel-files">
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
          <div class="workspace-tree space-y-0.5 overflow-auto text-sm" aria-label="Project notes">
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
            <div class="workspace-tree space-y-0.5 overflow-auto text-sm" aria-label="Workspace files">
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
</div>
