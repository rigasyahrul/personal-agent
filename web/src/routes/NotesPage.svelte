<!-- web/src/routes/NotesPage.svelte -->
<script lang="ts">
  import Breadcrumbs from '../components/Breadcrumbs.svelte'
  import NoteReader from '../components/notes/NoteReader.svelte'
  import NoteTree from '../components/notes/NoteTree.svelte'
  import Skeleton from '../components/Skeleton.svelte'
  import { api } from '../lib/api'
  import type { NoteDetail, NoteTreeEntry, Project } from '../lib/api/types'
  import { navigate, routeToHash } from '../lib/router'

  let {
    projectId,
    noteId,
    onProjectLoad,
  }: {
    projectId: string
    noteId?: string
    onProjectLoad?: (project: Project | null) => void
  } = $props()

  let project = $state<Project | null>(null)
  let entries = $state<NoteTreeEntry[]>([])
  let detail = $state<NoteDetail | null>(null)
  let treeLoading = $state(true)
  let detailLoading = $state(false)
  let treeError = $state('')
  let detailError = $state('')

  async function loadProjectAndTree() {
    treeLoading = true
    treeError = ''
    try {
      const [loadedProject, tree] = await Promise.all([
        api.getProject(projectId),
        api.listProjectNotes(projectId),
      ])
      project = loadedProject
      entries = tree ?? []
      onProjectLoad?.(loadedProject)
    } catch (cause) {
      project = null
      entries = []
      onProjectLoad?.(null)
      treeError = cause instanceof Error ? cause.message : 'Could not load notes.'
    } finally {
      treeLoading = false
    }
  }

  async function loadDetail(id: string) {
    detailLoading = true
    detailError = ''
    detail = null
    try {
      detail = await api.getProjectNote(projectId, id)
    } catch (cause) {
      detail = null
      detailError = cause instanceof Error ? cause.message : 'Could not load note.'
    } finally {
      detailLoading = false
    }
  }

  $effect(() => {
    // Track projectId so list reloads when the route project changes.
    void projectId
    void loadProjectAndTree()
  })

  $effect(() => {
    const id = noteId
    if (id) {
      void loadDetail(id)
    } else {
      detail = null
      detailError = ''
      detailLoading = false
    }
  })

  function selectNote(id: string) {
    navigate(routeToHash({ name: 'note', projectId, noteId: id }))
  }
</script>

{#if project}
  <div class="mb-4">
    <Breadcrumbs {project} leaf="Notes" />
  </div>
{/if}

{#if treeError}
  <p role="alert" class="mb-4 rounded-md bg-red-50 p-3 text-sm text-red-700">{treeError}</p>
{/if}

<div class="flex flex-col gap-4 md:flex-row md:items-start">
  <aside class="w-full shrink-0 rounded-xl border border-slate-200 bg-white p-4 md:w-72">
    <h1 class="mb-3 text-lg font-semibold">Notes</h1>
    {#if treeLoading}
      <div class="space-y-2" aria-busy="true">
        <Skeleton class="h-6" />
        <Skeleton class="h-6" />
        <Skeleton class="h-6" />
      </div>
    {:else if entries.length === 0}
      <p class="text-sm text-slate-600">No notes yet</p>
    {:else}
      <NoteTree
        {projectId}
        {entries}
        selectedNoteId={noteId}
        onselect={selectNote}
      />
    {/if}
  </aside>

  <section class="min-h-48 flex-1 rounded-xl border border-slate-200 bg-white p-4">
    {#if detailError}
      <p role="alert" class="rounded-md bg-red-50 p-3 text-sm text-red-700">{detailError}</p>
    {:else if detailLoading}
      <div class="space-y-3" aria-busy="true">
        <Skeleton class="h-8 w-1/2" />
        <Skeleton class="h-40" />
      </div>
    {:else if detail}
      <NoteReader note={detail} />
    {:else if entries.length > 0}
      <p class="text-sm text-slate-600">Select a note</p>
    {:else if !treeLoading}
      <p class="text-sm text-slate-500">Source notes will appear here.</p>
    {/if}
  </section>
</div>
