<!-- web/src/components/notes/NoteTree.svelte -->
<script lang="ts">
  import type { NoteTreeEntry } from '../../lib/api/types'

  let {
    projectId,
    entries,
    selectedNoteId,
    onselect,
  }: {
    projectId: string
    entries: NoteTreeEntry[]
    selectedNoteId?: string
    onselect?: (noteId: string) => void
  } = $props()
</script>

<ul role="tree" class="space-y-1 text-sm">
  {#each entries as entry (entry.path + (entry.note_id ?? ''))}
    {#if entry.kind === 'folder'}
      <li role="treeitem" class="font-medium text-slate-700" aria-expanded="true" aria-selected="false">
        📁 {entry.path}
      </li>
    {:else}
      <li role="treeitem" aria-selected={entry.note_id === selectedNoteId}>
        {#if entry.note_id}
          <a
            class="block rounded px-2 py-1 hover:bg-slate-100 {entry.note_id === selectedNoteId
              ? 'bg-indigo-50 font-medium text-indigo-800'
              : 'text-slate-800'}"
            href={`#/projects/${encodeURIComponent(projectId)}/notes/${encodeURIComponent(entry.note_id)}`}
            onclick={(event) => {
              if (onselect && entry.note_id) {
                event.preventDefault()
                onselect(entry.note_id)
              }
            }}
          >📄 {entry.path}</a>
        {:else}
          <span class="px-2 py-1 text-slate-600">📄 {entry.path}</span>
        {/if}
      </li>
    {/if}
  {/each}
</ul>
