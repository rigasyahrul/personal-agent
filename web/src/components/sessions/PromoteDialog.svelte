<!-- web/src/components/sessions/PromoteDialog.svelte -->
<script lang="ts">
  import { onMount } from 'svelte'
  import { api } from '../../lib/api'
  import type { PromotePayload, WorkspaceFile } from '../../lib/api/types'
  import { nextPromoteAttempt, type PromoteAttempt } from '../../lib/promote'

  let {
    open = false,
    sessionId,
    projectId,
    source,
    onclose,
    onsuccess,
    uuid = () => crypto.randomUUID(),
  }: {
    open?: boolean
    sessionId: string
    projectId: string
    source: WorkspaceFile
    onclose?: () => void
    onsuccess?: (operationId: string) => void
    uuid?: () => string
  } = $props()

  let dialogEl = $state<HTMLDialogElement | null>(null)
  let targetPath = $state('')
  let reviewMode = $state<'none' | 'whole' | 'bites'>('none')
  let error = $state('')
  let saving = $state(false)
  let projectName = $state('')
  let attempt = $state<PromoteAttempt | null>(null)
  // Capture source/session at open — do not follow later selection changes.
  let capturedSource = $state<WorkspaceFile | null>(null)
  let capturedSessionId = $state('')

  $effect(() => {
    if (open) {
      const src = source
      const sid = sessionId
      const pid = projectId
      capturedSource = src
      capturedSessionId = sid
      targetPath = src.path
      reviewMode = 'none'
      error = ''
      saving = false
      attempt = null
      projectName = pid
      void api
        .getProject(pid)
        .then((project) => {
          projectName = project?.name || pid
        })
        .catch(() => {
          projectName = pid
        })
      queueMicrotask(() => dialogEl?.showModal())
    } else {
      try {
        if (dialogEl?.open) dialogEl.close()
      } catch {
        /* ignore */
      }
    }
  })

  onMount(() => {
    return () => {
      try {
        if (dialogEl?.open) dialogEl.close()
      } catch {
        /* ignore */
      }
    }
  })

  function close() {
    onclose?.()
  }

  async function submit(event: Event) {
    event.preventDefault()
    const trimmed = targetPath.trim()
    if (!trimmed || !trimmed.endsWith('.md')) {
      error = 'Target path must end in .md'
      return
    }
    if (!capturedSource || !capturedSessionId) return
    const payload: PromotePayload = {
      workspace_path: capturedSource.path,
      target_relative_path: trimmed,
      review_mode: reviewMode,
    }
    attempt = nextPromoteAttempt(attempt, payload, uuid)
    saving = true
    error = ''
    try {
      const result = await api.promoteSession(capturedSessionId, attempt.payload, attempt.key)
      if (!result?.operation_id) throw new Error('Promotion did not return an operation ID')
      attempt = null
      onsuccess?.(result.operation_id)
      close()
    } catch (cause) {
      error = cause instanceof Error ? cause.message : 'Promotion failed'
      saving = false
    }
  }
</script>

<dialog
  bind:this={dialogEl}
  class="promote-dialog panel w-full max-w-md p-0 shadow-xl backdrop:bg-slate-900/40"
  onclose={close}
>
  <form method="dialog" class="form-stack p-5" onsubmit={submit}>
    <h2 class="text-lg font-semibold" style="margin:0">Save to source</h2>
    <p class="text-sm text-slate-600" style="margin:0">Project: {projectName}</p>
    <label>
      Target path
      <input
        class="field-input"
        name="target_relative_path"
        required
        bind:value={targetPath}
      />
    </label>
    <fieldset class="space-y-1 text-sm">
      <legend class="font-medium">Review mode</legend>
      {#each ['none', 'whole', 'bites'] as mode (mode)}
        <label class="flex items-center gap-2" style="display:flex">
          <input type="radio" name="review_mode" value={mode} bind:group={reviewMode} />
          {mode}
        </label>
      {/each}
    </fieldset>
    {#if error}
      <p role="alert" class="alert alert--error">{error}</p>
    {/if}
    <div class="flex justify-end gap-2">
      <button type="button" class="btn btn--secondary" onclick={close}>Cancel</button>
      <button type="submit" class="btn btn--primary" disabled={saving}>Save</button>
    </div>
  </form>
</dialog>
