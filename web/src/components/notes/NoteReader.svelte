<!-- web/src/components/notes/NoteReader.svelte -->
<script lang="ts">
  import type { NoteDetail } from '../../lib/api/types'
  import type { NoteBacklink } from '../../lib/api/notes'
  import { api } from '../../lib/api'
  import { navigate, routeToHash } from '../../lib/router'
  import Skeleton from '../Skeleton.svelte'
  import BacklinksPanel from './BacklinksPanel.svelte'

  let {
    note,
    projectId,
    noteId,
  }: {
    note: NoteDetail
    projectId: string
    noteId: string
  } = $props()

  let backlinks = $state<NoteBacklink[]>([])
  let backlinksLoading = $state(true)
  let backlinksError = $state('')
  let backlinksSeq = 0

  async function loadBacklinks(project: string, id: string) {
    const seq = ++backlinksSeq
    backlinksLoading = true
    backlinksError = ''
    backlinks = []
    try {
      const items = await api.listProjectNoteBacklinks(project, id)
      if (seq !== backlinksSeq) return
      backlinks = items
    } catch (cause) {
      if (seq !== backlinksSeq) return
      backlinks = []
      backlinksError = cause instanceof Error ? cause.message : 'Could not load backlinks.'
    } finally {
      if (seq === backlinksSeq) backlinksLoading = false
    }
  }

  $effect(() => {
    void loadBacklinks(projectId, noteId)
  })

  function openBacklink(item: NoteBacklink) {
    if (item.kind === 'source' && item.sourceNoteId) {
      navigate(routeToHash({ name: 'note', projectId, noteId: item.sourceNoteId }))
    }
  }
</script>

<div class="note-reader">
  <article class="prose prose-slate max-w-none">
    <h3 class="mb-3 text-lg font-semibold text-slate-900">{note.relative_path}</h3>
    {#if note.rendered_html}
      <!-- Server-provided safe HTML only -->
      {@html note.rendered_html}
    {:else}
      <pre class="whitespace-pre-wrap rounded-md bg-slate-50 p-4 text-sm text-slate-800">{note.body}</pre>
    {/if}
  </article>

  {#if backlinksError}
    <p role="alert" class="alert alert--error">{backlinksError}</p>
  {:else if backlinksLoading}
    <div class="space-y-2" aria-busy="true" aria-label="Loading backlinks">
      <Skeleton class="h-6" />
      <Skeleton class="h-6" />
    </div>
  {:else}
    <BacklinksPanel items={backlinks} onopen={openBacklink} />
  {/if}
</div>
