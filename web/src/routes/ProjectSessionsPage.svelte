<!-- web/src/routes/ProjectSessionsPage.svelte — filled in Task 42 -->
<script lang="ts">
  import type { Project } from '../lib/api/types'

  let {
    projectId: _projectId,
    onProjectLoad: _onProjectLoad,
  }: {
    projectId: string
    onProjectLoad?: (project: Project | null) => void
  } = $props()
</script>

<p>Sessions</p>
