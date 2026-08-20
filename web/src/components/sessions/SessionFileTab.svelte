<!-- web/src/components/sessions/SessionFileTab.svelte -->
<script lang="ts">
  import { api } from '../../lib/api'
  import type { WorkspaceFile } from '../../lib/api/types'
  import { isPromotableWorkspaceFile } from '../../lib/promote'
  import MarkdownView from '../markdown/MarkdownView.svelte'
  import Skeleton from '../Skeleton.svelte'

  let {
    sessionId,
    path,
    projectId: _projectId,
    mode: modeProp,
    onmode,
    onpromote,
  }: {
    sessionId: string
    path: string
    projectId: string
    mode?: 'preview' | 'source'
    onmode?: (mode: 'preview' | 'source') => void
    onpromote?: (file: WorkspaceFile) => void
  } = $props()

  let file = $state<WorkspaceFile | null>(null)
  let loading = $state(true)
  let error = $state('')
  let localMode = $state<'preview' | 'source'>('preview')
  let loadToken = 0

  const mode = $derived(modeProp ?? localMode)
  const promotable = $derived(isPromotableWorkspaceFile(file ?? { kind: 'file', path }))
  const isMarkdown = $derived(typeof path === 'string' && path.endsWith('.md'))
  const content = $derived(file?.content ?? '')

  async function load() {
    const token = ++loadToken
    loading = true
    error = ''
    file = null
    try {
      const next = await api.workspaceFile(sessionId, path)
      if (token !== loadToken) return
      file = next ?? { path, kind: 'file', content: '' }
    } catch (cause) {
      if (token !== loadToken) return
      error = cause instanceof Error ? cause.message : 'Unable to load file.'
      file = null
    } finally {
      if (token === loadToken) loading = false
    }
  }

  $effect(() => {
    void sessionId
    void path
    void load()
  })

  function setMode(next: 'preview' | 'source') {
    if (modeProp === undefined) localMode = next
    onmode?.(next)
  }

  function promote() {
    if (!file || !promotable) return
    onpromote?.(file)
  }
</script>

<div class="session-file-tab" data-path={path}>
  <div class="session-file-tab__toolbar">
    <div class="flex flex-wrap items-center gap-2" role="group" aria-label="View mode">
      <button
        type="button"
        class="btn btn--ghost"
        aria-pressed={mode === 'preview'}
        onclick={() => setMode('preview')}
      >Preview</button>
      <button
        type="button"
        class="btn btn--ghost"
        aria-pressed={mode === 'source'}
        onclick={() => setMode('source')}
      >Source</button>
    </div>
    {#if promotable && file && !loading && !error}
      <button type="button" class="btn btn--secondary" onclick={promote}>Save to source</button>
    {/if}
  </div>

  {#if loading}
    <div class="space-y-2" aria-busy="true">
      <Skeleton class="h-6" />
      <Skeleton class="h-24" />
    </div>
  {:else if error}
    <p role="alert" class="alert alert--error">{error}</p>
  {:else if mode === 'source' || !isMarkdown}
    <pre class="workspace-preview text-sm" style="margin:0; white-space:pre-wrap; font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace; overflow:auto">{content}</pre>
  {:else}
    <div class="message-prose session-file-tab__preview">
      <MarkdownView source={content} />
    </div>
  {/if}
</div>
