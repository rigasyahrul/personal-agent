<!-- web/src/components/ProjectRail.svelte -->
<script lang="ts">
  import { api } from '../lib/api'
  import type { NoteTreeEntry, WorkspaceEntry } from '../lib/api/types'
  import { buildHierarchy, flattenTree, type TreeNode } from '../lib/workspace-tree'
  import Skeleton from './Skeleton.svelte'

  let {
    projectId,
    sessionId = null,
    workspaceFilesEnabled = false,
    onOpenFile,
  }: {
    projectId: string
    sessionId?: string | null
    workspaceFilesEnabled?: boolean
    onOpenFile?: (path: string) => void
  } = $props()

  type RailTab = 'memory' | 'files'

  let tab = $state<RailTab>('memory')
  let memory = $state('')
  let instructions = $state('')

  let projectEntries = $state<WorkspaceEntry[]>([])
  let workspaceEntries = $state<WorkspaceEntry[]>([])
  let loading = $state(false)
  let error = $state('')
  let loadToken = 0

  function noteKindToWorkspace(kind: NoteTreeEntry['kind']): WorkspaceEntry['kind'] {
    if (kind === 'folder' || kind === 'directory') return 'directory'
    return 'file'
  }

  function notesToWorkspace(entries: NoteTreeEntry[]): WorkspaceEntry[] {
    return entries.map((entry) => ({
      path: entry.path,
      kind: noteKindToWorkspace(entry.kind),
    }))
  }

  async function loadFiles() {
    const token = ++loadToken
    loading = true
    error = ''
    try {
      const notes = await api.listProjectNotes(projectId)
      if (token !== loadToken) return
      projectEntries = notesToWorkspace(notes ?? [])

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
    } finally {
      if (token === loadToken) loading = false
    }
  }

  $effect(() => {
    void projectId
    void sessionId
    void workspaceFilesEnabled
    if (tab === 'files') {
      void loadFiles()
    }
  })

  function selectTab(next: RailTab) {
    tab = next
  }

  const projectRows = $derived(flattenTree(buildHierarchy(projectEntries)))
  const workspaceRows = $derived(flattenTree(buildHierarchy(workspaceEntries)))
  const hasAnyFiles = $derived(projectRows.length > 0 || workspaceRows.length > 0)

  function onRowClick(path: string, kind: TreeNode['kind']) {
    if (kind === 'directory') return
    onOpenFile?.(path)
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
      <label class="block text-sm" for="rail-memory">
        Memory
        <textarea
          id="rail-memory"
          class="field-textarea mt-1"
          aria-label="Memory"
          bind:value={memory}
          rows="6"
        ></textarea>
      </label>
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
      <p class="text-sm text-slate-500" style="margin:0">
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
                onclick={() => onRowClick(row.path, row.kind)}
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
                  onclick={() => onRowClick(row.path, row.kind)}
                >{row.path}</button>
              {/each}
            </div>
          </div>
        {/if}
      {/if}
    </div>
  {/if}
</div>
