<!-- web/src/components/sessions/WorkspacePanel.svelte -->
<!-- @deprecated Prefer SessionFilesBar. Thin reexport for back-compat; no preview/promote. -->
<script lang="ts">
  import type { ChatMessage } from '../../lib/api/types'
  import SessionFilesBar from './SessionFilesBar.svelte'

  let {
    sessionId,
    messages = [],
    activePath = null,
    onopen,
    onpromote: _onpromote,
  }: {
    sessionId: string
    messages?: ChatMessage[]
    activePath?: string | null
    onopen?: (path: string) => void
    /** @deprecated Promote moved to file tabs (Task 7). Ignored. */
    onpromote?: unknown
  } = $props()
</script>

<SessionFilesBar {sessionId} {messages} {activePath} {onopen} />
