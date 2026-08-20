<!-- web/src/components/sessions/SessionFilesBar.svelte -->
<script lang="ts">
  import { api } from '../../lib/api'
  import type { ChatMessage, WorkspaceEntry } from '../../lib/api/types'
  import {
    buildHierarchy,
    changedPathsFromMessages,
    filterEntriesByQuery,
    flattenTree,
  } from '../../lib/workspace-tree'
  import Skeleton from '../Skeleton.svelte'

  let {
    sessionId,
    messages = [],
    activePath = null,
    onopen,
  }: {
    sessionId: string
    messages?: ChatMessage[]
    activePath?: string | null
    onopen?: (path: string) => void
  } = $props()

  let entries = $state<WorkspaceEntry[]>([])
  let loading = $state(true)
  let error = $state('')
  let query = $state('')
  let loadToken = 0
  let lastToolSignature = ''

  const changed = $derived(changedPathsFromMessages(messages))
  const filtered = $derived(filterEntriesByQuery(entries, query))
  const rows = $derived(flattenTree(buildHierarchy(filtered)))

  async function refreshTree() {
    const token = ++loadToken
    loading = true
    error = ''
    try {
      const tree = await api.workspaceTree(sessionId)
      if (token !== loadToken) return
      entries = tree?.entries ?? []
    } catch (cause) {
      if (token !== loadToken) return
      error = cause instanceof Error ? cause.message : 'Unable to load workspace.'
      entries = []
    } finally {
      if (token === loadToken) loading = false
    }
  }

  $effect(() => {
    void sessionId
    lastToolSignature = ''
    void refreshTree()
  })

  // Refresh when tool messages introduce new changed paths.
  $effect(() => {
    const signature = [...changedPathsFromMessages(messages)].sort().join('|')
    if (signature && signature !== lastToolSignature) {
      lastToolSignature = signature
      void refreshTree()
    } else if (!signature) {
      lastToolSignature = ''
    }
  })

  function onRowClick(path: string, kind: string) {
    if (kind === 'directory') return
    onopen?.(path)
  }
</script>

<div class="session-files form-stack" aria-label="Session files" role="region">
  <label class="session-files__search field-label block text-sm">
    <span class="sr-only">Search files</span>
    <input
      class="field-input session-files__search-input"
      type="search"
      placeholder="Search files"
      aria-label="Search files"
      bind:value={query}
    />
  </label>

  {#if loading}
    <div class="space-y-2" aria-busy="true">
      <Skeleton class="h-6" />
      <Skeleton class="h-6" />
      <Skeleton class="h-6" />
    </div>
  {:else if error}
    <p role="alert" class="alert alert--error">{error || 'Unable to load workspace.'}</p>
  {:else if rows.length === 0}
    <p class="text-sm text-slate-500" style="margin:0">No files yet</p>
  {:else}
    <div class="workspace-tree session-files__tree max-h-full space-y-0.5 overflow-auto text-sm">
      {#each rows as row (row.path)}
        <button
          type="button"
          class="tree-item {row.path === activePath ? 'tree-item--active' : ''} {changed.has(row.path)
            ? 'bg-amber-50 tree-item--changed'
            : ''} {row.kind === 'directory' ? 'text-slate-500' : ''}"
          style="padding-left: {8 + row.depth * 12}px"
          disabled={row.kind === 'directory'}
          onclick={() => onRowClick(row.path, row.kind)}
        >{row.path}</button>
      {/each}
    </div>
  {/if}
</div>
