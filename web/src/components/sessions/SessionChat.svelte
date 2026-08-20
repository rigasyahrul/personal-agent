<!-- web/src/components/sessions/SessionChat.svelte -->
<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import { api } from '../../lib/api'
  import type {
    ChatMessage,
    OperationStatus,
    RunStatus,
    Session,
    WorkspaceFile,
  } from '../../lib/api/types'
  import {
    isTerminalBadge,
    loadOperationIds,
    saveOperationIds,
    workspaceEnabled,
  } from '../../lib/promote'
  import { createSessionPoller } from './session-poller'
  import OperationBadges from './OperationBadges.svelte'
  import PromoteDialog from './PromoteDialog.svelte'
  import WorkspacePanel from './WorkspacePanel.svelte'

  let {
    session,
    projectId,
    pollInterval = 1500,
    onclose,
    uuid = () => crypto.randomUUID(),
    storage = typeof localStorage !== 'undefined' ? localStorage : null,
  }: {
    session: Session
    projectId: string
    pollInterval?: number
    onclose?: () => void
    uuid?: () => string
    storage?: Storage | null
  } = $props()

  let messages = $state<ChatMessage[]>([])
  let run = $state<RunStatus | null>(null)
  let draft = $state('')
  let sending = $state(false)
  let sendingLock = false
  let error = $state('')
  let operationError = $state('')
  let pollFailed = $state(false)
  let generation = 0
  let destroyed = false
  let sendToken: object | null = null
  let operations = $state<string[]>([])
  let operationResults = $state(new Map<string, OperationStatus>())
  let retryingPending = $state(new Set<string>())
  let promoteOpen = $state(false)
  let promoteSource = $state<WorkspaceFile | null>(null)
  let showWorkspace = $derived(workspaceEnabled(session))

  const alertText = $derived([operationError, error].filter(Boolean).join(' — '))
  const runLabel = $derived(run ? `Run: ${run.status}` : 'Idle')
  const sendDisabled = $derived(Boolean(sending || run))

  function messagesEqual(left: ChatMessage[], right: ChatMessage[]): boolean {
    if (left === right) return true
    if (!left || !right || left.length !== right.length) return false
    for (let i = 0; i < left.length; i++) {
      const a = left[i]
      const b = right[i]
      if (a?.sequence !== b?.sequence || a?.role !== b?.role || a?.content !== b?.content) return false
    }
    return true
  }

  async function loadSnapshot(): Promise<{ messages: ChatMessage[]; run: RunStatus | null }> {
    const id = session.id
    const [nextMessages, nextRun] = await Promise.all([
      api.listMessages(id),
      api.currentRun(id),
    ])
    return { messages: nextMessages ?? [], run: nextRun ?? null }
  }

  function applySnapshot(value: { messages: ChatMessage[]; run: RunStatus | null }, gen: number, id: string) {
    if (destroyed || gen !== generation || session.id !== id) return
    const nextList = value.messages
    const messagesChanged = !messagesEqual(messages, nextList)
    messages = nextList
    run = value.run
    if (pollFailed) error = ''
    pollFailed = false
    void pollOperations()
    return messagesChanged
  }

  let poller = createSessionPoller(
    async () => {
      const gen = generation
      const id = session.id
      try {
        const snapshot = await loadSnapshot()
        applySnapshot(snapshot, gen, id)
        return snapshot
      } catch (cause) {
        if (!destroyed && gen === generation && session.id === id) {
          error = cause instanceof Error ? cause.message : 'Poll failed'
          pollFailed = true
        }
        throw cause
      }
    },
    () => {
      /* apply already done inside load to retain history on failure */
    },
    1500,
  )

  async function poll() {
    const gen = generation
    const id = session.id
    try {
      const snapshot = await loadSnapshot()
      applySnapshot(snapshot, gen, id)
    } catch (cause) {
      if (!destroyed && gen === generation && session.id === id) {
        error = cause instanceof Error ? cause.message : 'Poll failed'
        pollFailed = true
      }
    }
  }

  // Expose poll for focus regression harness without remounting.
  export { poll }

  async function pollOperations() {
    const gen = generation
    const id = session.id
    const active = operations.filter((operationId) => {
      const value = operationResults.get(operationId)
      return !value || !isTerminalBadge(value.badge)
    })
    if (!active.length) return
    let failed = false
    let nextOperationError = ''
    await Promise.all(
      active.map(async (operationId) => {
        try {
          const value = await api.operationStatus(operationId)
          if (!destroyed && gen === generation && session.id === id) {
            const next = new Map(operationResults)
            next.set(operationId, value)
            operationResults = next
          }
        } catch (cause) {
          if (!destroyed && gen === generation && session.id === id) {
            nextOperationError = cause instanceof Error ? cause.message : 'operation failed'
            failed = true
          }
        }
      }),
    )
    if (!destroyed && gen === generation && session.id === id) {
      operationError = failed ? nextOperationError : ''
    }
  }

  async function retryCards(op: OperationStatus) {
    const pendingId = op.pending_id
    if (!op.retry_cards || !pendingId || retryingPending.has(pendingId)) return
    const gen = generation
    const id = session.id
    const nextRetry = new Set(retryingPending)
    nextRetry.add(pendingId)
    retryingPending = nextRetry
    try {
      await api.retryReviewPending(pendingId)
      if (!destroyed && gen === generation && session.id === id) {
        const next = new Map(operationResults)
        next.delete(op.operation_id)
        operationResults = next
      }
    } catch (cause) {
      if (!destroyed && gen === generation && session.id === id) {
        error = cause instanceof Error ? cause.message : 'Retry failed'
      }
    } finally {
      const cleared = new Set(retryingPending)
      cleared.delete(pendingId)
      retryingPending = cleared
      if (!destroyed && gen === generation && session.id === id) {
        await pollOperations()
      }
    }
  }

  async function send(event: Event) {
    event.preventDefault()
    // Synchronous lock: Svelte $state updates are not re-read mid-tick across double submits.
    if (sendingLock || sending || !session) return
    const content = draft
    if (!content.trim()) return
    sendingLock = true
    sending = true
    error = ''
    pollFailed = false
    const id = session.id
    const gen = generation
    const token = {}
    sendToken = token
    const key = uuid()
    try {
      await api.sendMessage(id, { content, request_key: key })
      if (sendToken === token && !destroyed && gen === generation && session.id === id) {
        draft = ''
      }
    } catch (cause) {
      if (sendToken === token && !destroyed && gen === generation && session.id === id) {
        error = cause instanceof Error ? cause.message : 'Send failed'
        pollFailed = false
      }
    } finally {
      if (sendToken === token && !destroyed && gen === generation && session.id === id) {
        sending = false
        sendingLock = false
        sendToken = null
        await poll()
      } else if (sendToken === token) {
        sendingLock = false
      }
    }
  }

  function openPromote(file: WorkspaceFile) {
    promoteSource = file
    promoteOpen = true
  }

  function onPromoteSuccess(operationId: string) {
    if (!operations.includes(operationId)) {
      operations = [...operations, operationId]
      saveOperationIds(session.id, operations, storage)
    }
    promoteOpen = false
    promoteSource = null
    void pollOperations()
  }

  function resetForSession(value: Session) {
    generation += 1
    poller.stop()
    poller = createSessionPoller(
      async () => {
        const gen = generation
        const id = session.id
        try {
          const snapshot = await loadSnapshot()
          applySnapshot(snapshot, gen, id)
          return snapshot
        } catch (cause) {
          if (!destroyed && gen === generation && session.id === id) {
            error = cause instanceof Error ? cause.message : 'Poll failed'
            pollFailed = true
          }
          throw cause
        }
      },
      () => {
        /* apply already done inside load */
      },
      pollInterval,
    )
    messages = []
    run = null
    draft = ''
    sending = false
    sendingLock = false
    error = ''
    operationError = ''
    pollFailed = false
    sendToken = null
    promoteOpen = false
    promoteSource = null
    operationResults = new Map()
    retryingPending = new Set()
    operations = loadOperationIds(value.id, storage)
    poller.start()
  }

  let startedFor = ''
  $effect(() => {
    const value = session
    if (startedFor === value.id && !destroyed) return
    destroyed = false
    startedFor = value.id
    resetForSession(value)
  })

  onDestroy(() => {
    destroyed = true
    startedFor = ''
    generation += 1
    poller.stop()
    promoteOpen = false
  })
</script>

<div class="session-layout">
  <section class="session-chat session-layout__chat panel form-stack">
    <div class="flex flex-wrap items-center gap-3">
      <button type="button" class="link-accent" onclick={() => onclose?.()}>Sessions</button>
      <h2 class="text-xl font-semibold" style="margin:0">{session.title}</h2>
      <span class="badge-chip" style="background:#f4f4f5;color:#52525b"
      >{session.provider}:{session.model_id}</span>
    </div>

    <OperationBadges
      {operations}
      results={operationResults}
      retrying={retryingPending}
      onretry={(op) => void retryCards(op)}
    />

    <ol class="messages max-h-[50vh] space-y-3 overflow-auto" style="list-style:none;margin:0;padding:0">
      {#each [...messages].sort((a, b) => a.sequence - b.sequence) as message (message.sequence)}
        <li class="message message-{message.role} message-bubble">
          <strong>{message.role}</strong>
          <p>{message.content}</p>
        </li>
      {/each}
    </ol>

    <p class="run-status text-sm text-slate-600" role="status" aria-live="polite" style="margin:0">{runLabel}</p>

    {#if alertText}
      <p class="error alert alert--error" role="alert" data-chat-alert>{alertText}</p>
    {/if}

    <!-- Composer ancestry is stable: never conditionally remount this form during polls. -->
    <form class="sticky bottom-0 space-y-2 border-t border-slate-100 bg-white pt-3" onsubmit={send}>
      <label class="block text-sm">
        <span class="font-medium">Message</span>
        <textarea
          class="field-textarea mt-1"
          name="message"
          required
          rows="3"
          bind:value={draft}
        ></textarea>
      </label>
      <button
        type="submit"
        class="btn btn--primary"
        disabled={sendDisabled}
      >Send</button>
    </form>
  </section>

  {#if showWorkspace}
    <WorkspacePanel sessionId={session.id} {messages} onpromote={openPromote} />
  {/if}
</div>

{#if promoteOpen && promoteSource}
  <PromoteDialog
    open={promoteOpen}
    sessionId={session.id}
    {projectId}
    source={promoteSource}
    {uuid}
    onclose={() => {
      promoteOpen = false
      promoteSource = null
    }}
    onsuccess={onPromoteSuccess}
  />
{/if}
