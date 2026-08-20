<!-- web/src/routes/NotesPage.svelte — filled in Task 41 -->
<script lang="ts">
  import type { Project } from '../lib/api/types'

  let {
    projectId: _projectId,
    noteId: _noteId,
    onProjectLoad: _onProjectLoad,
  }: {
    projectId: string
    noteId?: string
    onProjectLoad?: (project: Project | null) => void
  } = $props()
</script>

<p>Notes</p>
