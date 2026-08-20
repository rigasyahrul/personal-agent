<!-- web/src/components/sessions/WorkspacePanel.svelte -->
<script lang="ts">
  import { api } from '../../lib/api'
  import type { ChatMessage, WorkspaceEntry, WorkspaceFile } from '../../lib/api/types'
  import { isPromotableWorkspaceFile } from '../../lib/promote'
  import Skeleton from '../Skeleton.svelte'

  let {
    sessionId,
    messages = [],
    onpromote,
  }: {
    sessionId: string
    messages?: ChatMessage[]
    onpromote?: (file: WorkspaceFile) => void
  } = $props()

  let entries = $state<WorkspaceEntry[]>([])
  let loading = $state(true)
  let error = $state('')
  let preview = $state('Select a file')
  let selected = $state<WorkspaceFile | null>(null)
  let loadToken = 0

  function changedPaths(list: ChatMessage[]): Set<string> {
    const paths = new Set<string>()
    for (const message of list) {
      if (message?.role !== 'tool') continue
      let path = message.changed_path
      if (!path && typeof message.content === 'string') {
        try {
          path = JSON.parse(message.content)?.changed_path
        } catch {
          path = ''
        }
      }
      if (typeof path === 'string' && path) paths.add(path)
    }
    return paths
  }

  async function refreshTree() {
    const token = ++loadToken
    loading = true
    error = ''
    try {
      const tree = await api.workspaceTree(sessionId)
      if (token !== loadToken) return
      entries = tree?.entries ?? []
      if (selected) {
        const same = entries.find(
          (entry) => entry.path === selected?.path && isPromotableWorkspaceFile(entry),
        )
        if (!same) selected = null
      }
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
    void refreshTree()
  })

  // Refresh when tool messages introduce new changed paths.
  let lastToolSignature = ''
  $effect(() => {
    const signature = [...changedPaths(messages)].sort().join('|')
    if (signature && signature !== lastToolSignature) {
      lastToolSignature = signature
      void refreshTree()
    } else if (!signature) {
      lastToolSignature = ''
    }
  })

  async function selectFile(entry: WorkspaceEntry) {
    if (entry.kind === 'directory') return
    try {
      const file = await api.workspaceFile(sessionId, entry.path)
      preview = file?.content ?? ''
      selected = { path: entry.path, kind: 'file', content: file?.content }
    } catch {
      preview = 'Unable to read file.'
      selected = null
    }
  }
</script>

<aside
  class="workspace-panel flex w-full flex-col rounded-xl border border-slate-200 bg-white p-4 md:w-80"
  aria-label="Workspace"
>
  <h2 class="mb-3 text-base font-semibold">Workspace files</h2>
  {#if loading}
    <div class="space-y-2" aria-busy="true">
      <Skeleton class="h-6" />
      <Skeleton class="h-6" />
    </div>
  {:else if error}
    <p role="alert" class="text-sm text-red-700">{error || 'Unable to load workspace.'}</p>
  {:else}
    {@const changed = changedPaths(messages)}
    <div class="workspace-tree mb-3 max-h-56 space-y-1 overflow-auto text-sm">
      {#each entries as entry (entry.path)}
        <button
          type="button"
          class="block w-full rounded px-2 py-1 text-left hover:bg-slate-100 {changed.has(entry.path)
            ? 'bg-amber-50'
            : ''} {entry.kind === 'directory' ? 'text-slate-500' : 'text-slate-800'}"
          disabled={entry.kind === 'directory'}
          onclick={() => void selectFile(entry)}
        >{entry.path}</button>
      {/each}
    </div>
    <pre class="workspace-preview max-h-48 overflow-auto rounded-md bg-slate-50 p-3 text-xs" aria-live="polite">{preview}</pre>
    {#if selected && isPromotableWorkspaceFile(selected)}
      <button
        type="button"
        class="mt-3 rounded-md border border-indigo-200 bg-indigo-50 px-3 py-1.5 text-sm font-medium text-indigo-800"
        data-promote=""
        onclick={() => selected && onpromote?.(selected)}
      >Save to source</button>
    {/if}
  {/if}
</aside>
