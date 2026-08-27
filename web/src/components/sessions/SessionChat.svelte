<!-- web/src/components/sessions/SessionChat.svelte -->
<script lang="ts">
  import { onDestroy, onMount, untrack } from 'svelte'
  import { api } from '../../lib/api'
  import type {
    ChatMessage,
    CompoundItem,
    CompoundProposal,
    OperationStatus,
    Project,
    RunStatus,
    Session,
    WorkspaceEntry,
    WorkspaceFile,
  } from '../../lib/api/types'
  import {
    isTerminalBadge,
    loadOperationIds,
    saveOperationIds,
    workspaceEnabled,
  } from '../../lib/promote'
  import {
    clampMainPct,
    DEFAULT_MAIN_PCT,
    readFilesBarOpen,
    readFilesBarWidthPct,
    writeFilesBarOpen,
    writeFilesBarWidthPct,
  } from '../../lib/session-prefs'
  import { formatMessageDateTime, formatSessionDate } from '../../lib/format-session-date'
  import {
    buildThoughtsView,
    chipInsertAfterSequence,
    runElapsedSec,
    shouldShowChip,
    type ThoughtsView,
  } from '../../lib/thoughts'
  import Breadcrumbs from '../Breadcrumbs.svelte'
  import MarkdownView from '../markdown/MarkdownView.svelte'
  import { createSessionPoller } from './session-poller'
  import OperationBadges from './OperationBadges.svelte'
  import PromoteDialog from './PromoteDialog.svelte'
  import CompoundReviewCard from './CompoundReviewCard.svelte'
  import SessionFileTab from './SessionFileTab.svelte'
  import SessionFilesBar from './SessionFilesBar.svelte'
  import { changedPathsFromMessages } from '../../lib/workspace-tree'
  import {
    activeMention,
    insertMention,
    rankWorkspaceFiles,
    type RankedFile,
  } from '../../lib/mention-files'

  type TabId = 'agent' | `file:${string}`
  type FileSource = 'project-note' | 'workspace'
  type FileTabState = {
    path: string
    mode: 'preview' | 'source'
    source: FileSource
    noteId?: string
  }
  type OpenFileRequest = {
    path: string
    source: FileSource
    noteId?: string
  }

  const FILE_TAB_CAP = 8

  function basename(path: string): string {
    const parts = path.split('/')
    return parts[parts.length - 1] || path
  }

  function fileTabId(path: string): TabId {
    return `file:${path}`
  }

  let {
    session,
    projectId,
    project: projectProp = null,
    pollInterval = 1500,
    onclose,
    uuid = () => crypto.randomUUID(),
    storage = typeof localStorage !== 'undefined' ? localStorage : null,
    openPath = $bindable<string | null>(null),
    openFileRequest = $bindable<OpenFileRequest | null>(null),
    embeddedInHub = false,
    onOpenThoughts,
    thoughtsRunId = $bindable<string | null>(null),
    onThoughtsView,
  }: {
    session: Session
    projectId: string
    /** When provided (hub), used for header breadcrumbs without an extra fetch. */
    project?: Project | null
    pollInterval?: number
    onclose?: () => void
    uuid?: () => string
    storage?: Storage | null
    /** Hub convenience: open a workspace file tab; cleared after open. */
    openPath?: string | null
    /** Rich open request (project notes + workspace); cleared after open. */
    openFileRequest?: OpenFileRequest | null
    /** When true, hide internal Show files toggle / SessionFilesBar (hub ProjectRail owns files). */
    embeddedInHub?: boolean
    onOpenThoughts?: (runId: string) => void
    thoughtsRunId?: string | null
    onThoughtsView?: (view: ThoughtsView | null) => void
  } = $props()

  let loadedProject = $state<Project | null>(null)
  const breadcrumbProject = $derived(projectProp ?? loadedProject)

  let messages = $state<ChatMessage[]>([])
  let run = $state<RunStatus | null>(null)
  let draft = $state('')
  let composerEl: HTMLTextAreaElement | undefined = $state()
  let caret = $state(0)
  let mentionDismissed = $state(false)
  let treeEntries = $state<WorkspaceEntry[] | null>(null)
  let treeLoading = $state(false)
  let treeError = $state('')
  let treeLoadToken = 0
  let treeSignature = ''
  let mentionIndex = $state(0)
  let sending = $state(false)
  let sendingLock = false
  let error = $state('')
  let operationError = $state('')
  let pollFailed = $state(false)
  /** Sequence of assistant message last copied; drives brief "Copied" feedback. */
  let copiedSeq = $state<number | null>(null)
  let copiedClearTimer: ReturnType<typeof setTimeout> | null = null
  let nowMs = $state(Date.now())
  let generation = 0
  let destroyed = false
  let sendToken: object | null = null
  let operations = $state<string[]>([])
  let operationResults = $state(new Map<string, OperationStatus>())
  let retryingPending = $state(new Set<string>())
  let promoteOpen = $state(false)
  let promoteSource = $state<WorkspaceFile | null>(null)
  let compoundProposal = $state<CompoundProposal | null>(null)
  let compounding = $state(false)
  let deciding = $state(false)
  let compoundingLock = false
  let showWorkspace = $derived(workspaceEnabled(session))
  const mention = $derived(activeMention(draft, caret))
  const mentionActive = $derived(Boolean(showWorkspace && mention && !mentionDismissed))
  const mentionRows = $derived(
    mentionActive && treeEntries
      ? rankWorkspaceFiles(treeEntries, mention?.query ?? '')
      : [],
  )
  let activePath = $state<string | null>(null)
  let openFileTabs = $state<FileTabState[]>([])
  let activeTab = $state<TabId>('agent')
  /** Activation order among file tabs (most recent last). Agent never participates. */
  let fileTabLru = $state<string[]>([])

  let filesOpen = $state(false)
  let mainPct = $state(DEFAULT_MAIN_PCT)
  let splitEl: HTMLElement | undefined = $state()
  let dragging = false
  let isNarrow = $state(false)

  const NARROW_MQ = '(max-width: 1023px)'

  const alertText = $derived([operationError, error].filter(Boolean).join(' — '))
  const runLabel = $derived(run ? `Run: ${run.status}` : 'Idle')
  const sendDisabled = $derived(Boolean(sending || run))
  const runBusy = $derived(run?.status === 'queued' || run?.status === 'running')
  const compoundWait = $derived(Boolean(sending || runBusy))
  const compoundBusy = $derived(compounding || deciding)
  const compoundDisabled = $derived(
    compoundWait || compoundBusy || Boolean(compoundProposal),
  )
  const compoundTitle = $derived(compoundWait ? 'Wait for the current run' : undefined)
  const agentActive = $derived(activeTab === 'agent')
  const activeFileTab = $derived(
    activeTab.startsWith('file:')
      ? openFileTabs.find((t) => fileTabId(t.path) === activeTab) ?? null
      : null,
  )

  function summarizeRecentMessages(list: ChatMessage[]): string {
    const picked = [...list]
      .sort((a, b) => a.sequence - b.sequence)
      .filter((m) => m.role === 'user' || m.role === 'assistant' || m.role === 'model')
      .slice(-6)
    return picked
      .map((m) => {
        const role = m.role === 'model' ? 'assistant' : m.role
        const text = (m.content ?? '').replace(/\s+/g, ' ').trim().slice(0, 280)
        return `${role}: ${text}`
      })
      .join('\n')
  }

  function messagesEqual(left: ChatMessage[], right: ChatMessage[]): boolean {
    if (left === right) return true
    if (!left || !right || left.length !== right.length) return false
    for (let i = 0; i < left.length; i++) {
      const a = left[i]
      const b = right[i]
      if (
        a?.sequence !== b?.sequence ||
        a?.role !== b?.role ||
        a?.content !== b?.content ||
        a?.run_id !== b?.run_id ||
        a?.tool_calls_json !== b?.tool_calls_json
      ) {
        return false
      }
    }
    return true
  }

  $effect(() => {
    const runId = thoughtsRunId
    const msgs = messages
    const current = run
    const now = nowMs
    untrack(() => {
      if (!runId) {
        onThoughtsView?.(null)
        return
      }
      onThoughtsView?.(
        buildThoughtsView({
          runId,
          messages: msgs,
          current,
          nowMs: now,
        }),
      )
    })
  })

  function thoughtChipAfter(message: ChatMessage): { runId: string; n: number } | null {
    const runId = message.run_id
    if (!runId) return null
    if (chipInsertAfterSequence(messages, runId) !== message.sequence) return null
    const input = { runId, messages, current: run, nowMs }
    if (!shouldShowChip(input)) return null
    return { runId, n: runElapsedSec(input) }
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

  async function startCompound() {
    if (compoundingLock || compoundWait || compoundBusy || compoundProposal) return
    compoundingLock = true
    compounding = true
    error = ''
    pollFailed = false
    const id = session.id
    const gen = generation
    const key = uuid()
    try {
      const proposal = await api.createCompound(id, {
        request_key: key,
        user_context: summarizeRecentMessages(messages),
      })
      if (!destroyed && gen === generation && session.id === id && proposal?.status === 'pending') {
        compoundProposal = proposal
      }
    } catch (cause) {
      if (!destroyed && gen === generation && session.id === id) {
        error = cause instanceof Error ? cause.message : 'Compound failed'
        pollFailed = false
      }
    } finally {
      if (!destroyed && gen === generation && session.id === id) {
        compounding = false
        compoundingLock = false
      } else if (compoundingLock) {
        compoundingLock = false
      }
    }
  }

  async function onCompoundConfirm(decision: 'approve' | 'reject', items: CompoundItem[]) {
    const proposal = compoundProposal
    if (!proposal || deciding) return
    deciding = true
    error = ''
    const id = session.id
    const gen = generation
    const key = uuid()
    try {
      const got = await api.decideCompound(id, proposal.id, {
        request_key: key,
        decision,
        items,
      })
      if (!destroyed && gen === generation && session.id === id) {
        const finished = Boolean(got?.finished_at)
        const ok =
          got?.status === 'rejected' || (got?.status === 'approved' && finished)
        if (ok) {
          compoundProposal = null
        } else {
          error =
            got?.error?.trim() ||
            (got?.status === 'failed'
              ? 'Compound publish failed'
              : 'Compound is not finished yet')
        }
      }
    } catch (cause) {
      if (!destroyed && gen === generation && session.id === id) {
        error = cause instanceof Error ? cause.message : 'Decide failed'
      }
    } finally {
      if (!destroyed && gen === generation && session.id === id) {
        deciding = false
      }
    }
  }

  async function ensureTree() {
    const token = ++treeLoadToken
    treeLoading = true
    treeError = ''
    try {
      const tree = await api.workspaceTree(session.id)
      if (token !== treeLoadToken) return
      treeEntries = tree?.entries ?? []
    } catch {
      if (token !== treeLoadToken) return
      treeError = "Couldn't load files"
      treeEntries = []
    } finally {
      if (token === treeLoadToken) treeLoading = false
    }
  }

  $effect(() => {
    void session.id
    treeEntries = null
    treeError = ''
    treeSignature = ''
    treeLoadToken += 1
  })

  $effect(() => {
    if (!mentionActive || !showWorkspace) return
    const sig = [...changedPathsFromMessages(messages)].sort().join('|')
    if (treeEntries === null || (sig && sig !== treeSignature)) {
      if (sig) treeSignature = sig
      void ensureTree()
    }
  })

  function syncCaret(e: Event) {
    const el = e.currentTarget as HTMLTextAreaElement
    caret = el.selectionStart ?? el.value.length
  }

  $effect(() => {
    const token = mention ? `${mention.start}:${mention.query}` : ''
    void token
    mentionDismissed = false
  })

  $effect(() => {
    if (mentionIndex >= mentionRows.length) mentionIndex = 0
  })

  function pickMention(row: RankedFile) {
    const current = activeMention(draft, caret)
    if (!current) return
    const next = insertMention(draft, current, row.path)
    draft = next.text
    caret = next.cursor
    mentionDismissed = true
    requestAnimationFrame(() => {
      composerEl?.focus()
      composerEl?.setSelectionRange(next.cursor, next.cursor)
    })
  }

  /** Enter sends; Shift+Enter inserts a newline (same as hub composer). */
  function onComposerKeydown(e: KeyboardEvent) {
    if (e.isComposing) return
    if (mentionActive) {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        if (mentionRows.length) mentionIndex = (mentionIndex + 1) % mentionRows.length
        return
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        if (mentionRows.length) {
          mentionIndex = (mentionIndex - 1 + mentionRows.length) % mentionRows.length
        }
        return
      }
      if (e.key === 'Escape') {
        e.preventDefault()
        mentionDismissed = true
        return
      }
      if (e.key === 'Enter' || e.key === 'Tab') {
        e.preventDefault()
        const row = mentionRows[mentionIndex] ?? mentionRows[0]
        if (row) pickMention(row)
        return
      }
    }
    if (e.key !== 'Enter' || e.shiftKey) return
    e.preventDefault()
    if (sendDisabled || !draft.trim()) return
    const form = (e.currentTarget as HTMLTextAreaElement).form
    form?.requestSubmit()
  }

  async function copyAssistant(text: string, sequence: number) {
    try {
      await navigator.clipboard.writeText(text)
      if (destroyed) return
      copiedSeq = sequence
      if (copiedClearTimer) clearTimeout(copiedClearTimer)
      copiedClearTimer = setTimeout(() => {
        if (copiedSeq === sequence) copiedSeq = null
        copiedClearTimer = null
      }, 1500)
    } catch {
      /* clipboard may be unavailable; ignore */
    }
  }

  function openPromote(file: WorkspaceFile) {
    promoteSource = file
    promoteOpen = true
  }

  function bumpFileLru(path: string) {
    fileTabLru = [...fileTabLru.filter((p) => p !== path), path]
  }

  function openFile(
    path: string,
    meta?: { source?: FileSource; noteId?: string },
  ) {
    if (!path) return
    const source: FileSource = meta?.source ?? 'workspace'
    const noteId = source === 'project-note' ? meta?.noteId : undefined
    const existing = openFileTabs.find((t) => t.path === path)
    if (existing) {
      // Refresh source/note metadata if the same path is re-opened from the rail.
      openFileTabs = openFileTabs.map((t) =>
        t.path === path ? { ...t, source, noteId } : t,
      )
      activeTab = fileTabId(path)
      activePath = path
      bumpFileLru(path)
      return
    }
    let nextTabs = openFileTabs
    if (nextTabs.length >= FILE_TAB_CAP) {
      const lruPath = fileTabLru[0] ?? nextTabs[0]?.path
      if (lruPath) {
        nextTabs = nextTabs.filter((t) => t.path !== lruPath)
        fileTabLru = fileTabLru.filter((p) => p !== lruPath)
        if (activePath === lruPath) activePath = null
      }
    }
    openFileTabs = [...nextTabs, { path, mode: 'preview', source, noteId }]
    activeTab = fileTabId(path)
    activePath = path
    bumpFileLru(path)
  }

  function closeFile(path: string) {
    const wasActive = activeTab === fileTabId(path)
    openFileTabs = openFileTabs.filter((t) => t.path !== path)
    fileTabLru = fileTabLru.filter((p) => p !== path)
    if (activePath === path) activePath = null
    if (wasActive) {
      activeTab = 'agent'
    }
  }

  function selectAgentTab() {
    activeTab = 'agent'
  }

  function selectFileTab(path: string) {
    if (!openFileTabs.some((t) => t.path === path)) return
    activeTab = fileTabId(path)
    activePath = path
    bumpFileLru(path)
  }

  function setFileTabMode(path: string, mode: 'preview' | 'source') {
    openFileTabs = openFileTabs.map((t) => (t.path === path ? { ...t, mode } : t))
  }

  function openPromoteFromTab(file: WorkspaceFile) {
    // Promote is workspace-only; project notes must not open the dialog.
    if (activeFileTab?.source === 'project-note') return
    openPromote(file)
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

  function toggleFiles() {
    filesOpen = !filesOpen
    writeFilesBarOpen(storage, filesOpen)
  }

  function closeFiles() {
    if (!filesOpen) return
    filesOpen = false
    writeFilesBarOpen(storage, false)
  }

  function onHandlePointerDown(event: PointerEvent) {
    if (!splitEl) return
    dragging = true
    const target = event.currentTarget as HTMLElement
    target.setPointerCapture?.(event.pointerId)
    event.preventDefault()
  }

  function onHandlePointerMove(event: PointerEvent) {
    if (!dragging || !splitEl) return
    const rect = splitEl.getBoundingClientRect()
    if (rect.width <= 0) return
    const pct = ((event.clientX - rect.left) / rect.width) * 100
    mainPct = clampMainPct(Math.round(pct))
  }

  function onHandlePointerUp(event: PointerEvent) {
    if (!dragging) return
    dragging = false
    const target = event.currentTarget as HTMLElement
    target.releasePointerCapture?.(event.pointerId)
    writeFilesBarWidthPct(storage, mainPct)
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
    compoundProposal = null
    compounding = false
    deciding = false
    compoundingLock = false
    activePath = null
    openFileTabs = []
    activeTab = 'agent'
    fileTabLru = []
    operationResults = new Map()
    retryingPending = new Set()
    operations = loadOperationIds(value.id, storage)
    filesOpen = readFilesBarOpen(storage)
    mainPct = readFilesBarWidthPct(storage)
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

  $effect(() => {
    void projectId
    void projectProp
    if (projectProp) {
      loadedProject = projectProp
      return
    }
    let cancelled = false
    void api
      .getProject(projectId)
      .then((p) => {
        if (!cancelled) loadedProject = p
      })
      .catch(() => {
        if (!cancelled) loadedProject = null
      })
    return () => {
      cancelled = true
    }
  })

  onMount(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return
    const mql = window.matchMedia(NARROW_MQ)
    const apply = () => {
      isNarrow = mql.matches
    }
    apply()
    const onChange = () => apply()
    if (typeof mql.addEventListener === 'function') {
      mql.addEventListener('change', onChange)
      return () => mql.removeEventListener('change', onChange)
    }
    mql.addListener(onChange)
    return () => mql.removeListener(onChange)
  })

  $effect(() => {
    if (!isNarrow || !filesOpen || !showWorkspace || embeddedInHub) return
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        closeFiles()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  })

  $effect(() => {
    const live = run?.status === 'queued' || run?.status === 'running'
    if (!live) return
    nowMs = Date.now()
    const timer = setInterval(() => {
      nowMs = Date.now()
    }, 1000)
    return () => clearInterval(timer)
  })

  /** Hub ProjectRail drives file tabs via openFileRequest; clear after handling so the same path can re-open. */
  $effect(() => {
    const req = openFileRequest
    if (!req?.path) return
    openFile(req.path, { source: req.source, noteId: req.noteId })
    openFileRequest = null
  })

  /** Convenience: openPath means a workspace-source open (files bar / legacy hub bind). */
  $effect(() => {
    const path = openPath
    if (!path) return
    openFile(path, { source: 'workspace' })
    openPath = null
  })

  onDestroy(() => {
    destroyed = true
    startedFor = ''
    generation += 1
    poller.stop()
    promoteOpen = false
    if (copiedClearTimer) {
      clearTimeout(copiedClearTimer)
      copiedClearTimer = null
    }
  })
</script>

<div class="session-focus" data-files-open={filesOpen && showWorkspace ? '1' : '0'}>
  <header class="session-focus__header">
    <div class="session-focus__header-lead">
      {#if breadcrumbProject}
        <Breadcrumbs
          project={breadcrumbProject}
          leaf={session.title}
          onProjectClick={onclose}
        />
      {:else}
        <button type="button" class="link-accent" onclick={() => onclose?.()}>Back</button>
        <h2 class="session-focus__title">{session.title}</h2>
      {/if}
    </div>
    <div class="session-focus__header-meta">
      <span class="session-focus__model-quiet">{session.provider}:{session.model_id}</span>
      <p class="run-status session-focus__run" role="status" aria-live="polite">{runLabel}</p>
      {#if showWorkspace && !embeddedInHub}
        <button
          type="button"
          class="btn btn--secondary"
          aria-pressed={filesOpen}
          onclick={toggleFiles}
        >{filesOpen ? 'Hide files' : 'Show files'}</button>
      {/if}
    </div>
    <button
      type="button"
      class="btn btn--secondary"
      disabled={compoundDisabled}
      title={compoundTitle}
      aria-busy={compoundBusy}
      onclick={() => void startCompound()}
    >Compound</button>
  </header>

  <OperationBadges
    {operations}
    results={operationResults}
    retrying={retryingPending}
    onretry={(op) => void retryCards(op)}
  />

  {#if alertText}
    <p class="error alert alert--error" role="alert" data-chat-alert>{alertText}</p>
  {/if}

  <div class="session-tabs" role="tablist" aria-label="Session tabs">
    <button
      type="button"
      class="session-tab {agentActive ? 'session-tab--active' : ''}"
      role="tab"
      aria-selected={agentActive}
      onclick={selectAgentTab}
    >Agent</button>
    {#each openFileTabs as tab (tab.path)}
      {@const selected = activeTab === fileTabId(tab.path)}
      <button
        type="button"
        class="session-tab {selected ? 'session-tab--active' : ''}"
        role="tab"
        aria-selected={selected}
        title={tab.path}
        onclick={() => selectFileTab(tab.path)}
      >
        <span class="session-tab__label">{basename(tab.path)}</span>
        <span
          class="session-tab__close"
          role="button"
          tabindex="0"
          aria-label="Close {basename(tab.path)}"
          onclick={(event) => {
            event.stopPropagation()
            closeFile(tab.path)
          }}
          onkeydown={(event) => {
            if (event.key === 'Enter' || event.key === ' ') {
              event.preventDefault()
              event.stopPropagation()
              closeFile(tab.path)
            }
          }}
        >×</span>
      </button>
    {/each}
  </div>

  <div
    class="session-split"
    style="--session-main-pct: {mainPct}%"
    bind:this={splitEl}
  >
    <div class="session-split__main">
      {#if agentActive}
        <div class="session-chat-column">
          <ol class="messages message-thread session-focus__messages">
            {#each [...messages].sort((a, b) => a.sequence - b.sequence) as message (message.sequence)}
              {#if message.role === 'user'}
                {@const chip = thoughtChipAfter(message)}
                <li
                  class="message message-row message-row--user"
                  data-role="user"
                  data-raw-role={message.role}
                >
                  <div class="message-bubble message-bubble--user">
                    <p>{message.content}</p>
                  </div>
                </li>
                {#if chip}
                  <li class="message message-row message-row--thought" data-role="thought">
                    <button
                      type="button"
                      class="thought-chip"
                      aria-label="Thought for {chip.n}s"
                      onclick={() => onOpenThoughts?.(chip.runId)}
                    >
                      <svg class="thought-chip__bulb" viewBox="0 0 24 24" aria-hidden="true">
                        <path d="M15 14c.2-1 .7-1.7 1.5-2.5 1-.9 1.5-2.2 1.5-3.5A6 6 0 0 0 6 8c0 1 .2 2.2 1.3 3.5.7.7 1.3 1.5 1.5 2.5" />
                        <path d="M9 18h6" />
                        <path d="M10 22h4" />
                      </svg>
                      <svg class="thought-chip__panel" viewBox="0 0 24 24" aria-hidden="true">
                        <rect x="3" y="3" width="18" height="18" rx="2" />
                        <path d="M9 3v18" />
                        <path d="m16 15-3-3 3-3" />
                      </svg>
                      <span>Thought for {chip.n}s</span>
                    </button>
                  </li>
                {/if}
              {:else if (message.role === 'assistant' || message.role === 'model') && (message.content ?? '').trim()}
                {@const shortDate = formatSessionDate(message.created_at)}
                {@const fullDate = formatMessageDateTime(message.created_at)}
                <li
                  class="message message-row message-row--assistant"
                  data-role="assistant"
                  data-raw-role={message.role}
                >
                  <div class="message-assistant">
                    <div class="message-prose">
                      <MarkdownView source={message.content} />
                    </div>
                    <div class="message-assistant__footer">
                      <button
                        type="button"
                        class="message-copy"
                        aria-label={copiedSeq === message.sequence ? 'Copied' : 'Copy response'}
                        title={copiedSeq === message.sequence ? 'Copied' : 'Copy response'}
                        onclick={() => void copyAssistant(message.content, message.sequence)}
                      >
                        {#if copiedSeq === message.sequence}
                          <svg
                            class="message-copy__icon"
                            viewBox="0 0 24 24"
                            aria-hidden="true"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="2"
                            stroke-linecap="round"
                            stroke-linejoin="round"
                          >
                            <path d="M20 6 9 17l-5-5"></path>
                          </svg>
                        {:else}
                          <svg
                            class="message-copy__icon"
                            viewBox="0 0 24 24"
                            aria-hidden="true"
                            fill="none"
                            stroke="currentColor"
                            stroke-width="1.8"
                            stroke-linecap="round"
                            stroke-linejoin="round"
                          >
                            <rect x="9" y="9" width="11" height="11" rx="2"></rect>
                            <path d="M5 15V5a2 2 0 0 1 2-2h10"></path>
                          </svg>
                        {/if}
                      </button>
                      {#if shortDate && fullDate}
                        <time
                          class="message-assistant__date"
                          datetime={message.created_at}
                          data-tooltip={fullDate}
                          title={fullDate}
                        >{shortDate}</time>
                      {/if}
                    </div>
                  </div>
                </li>
              {/if}
            {/each}
          </ol>
        </div>
      {:else if activeFileTab}
        <SessionFileTab
          sessionId={session.id}
          path={activeFileTab.path}
          {projectId}
          source={activeFileTab.source}
          noteId={activeFileTab.noteId}
          mode={activeFileTab.mode}
          onmode={(m) => setFileTabMode(activeFileTab.path, m)}
          onpromote={openPromoteFromTab}
        />
      {/if}

      {#if compoundProposal}
        <div
          class="session-compound"
          hidden={!agentActive}
          inert={!agentActive ? true : undefined}
        >
          <CompoundReviewCard
            proposal={compoundProposal}
            busy={deciding}
            onconfirm={(decision, items) => void onCompoundConfirm(decision, items)}
          />
        </div>
      {/if}

      <!-- Composer ancestry is stable: never destroy/recreate this form on poll or tab switch. -->
      <form
        class="session-composer"
        class:session-composer--hidden={!agentActive}
        hidden={!agentActive}
        inert={!agentActive ? true : undefined}
        onsubmit={send}
      >
        <div class="session-composer__card">
          {#if mentionActive}
            <div class="session-composer__mentions" id="session-composer-mentions">
              {#if treeLoading && treeEntries === null}
                <p class="session-composer__mentions-status">Loading files…</p>
              {:else if treeError}
                <p class="session-composer__mentions-status">{treeError}</p>
              {:else if mentionRows.length === 0}
                <p class="session-composer__mentions-status">No matching files</p>
              {:else}
                <ul role="listbox" aria-label="Workspace files">
                  {#each mentionRows as row, i (row.path)}
                    <li
                      id={`mention-option-${i}`}
                      class="mention-option"
                      class:mention-option--active={i === mentionIndex}
                      role="option"
                      aria-selected={i === mentionIndex}
                      onmousedown={(event) => {
                        event.preventDefault()
                        pickMention(row)
                      }}
                    >
                      <span class="mention-option__name">{row.name}</span>
                      {#if row.parent}
                        <span class="mention-option__path">{row.parent}</span>
                      {/if}
                    </li>
                  {/each}
                </ul>
              {/if}
            </div>
          {/if}
          <textarea
            class="session-composer__input"
            name="message"
            aria-label="Message"
            placeholder="Reply…"
            required
            rows="2"
            bind:this={composerEl}
            bind:value={draft}
            autocomplete="off"
            aria-autocomplete="list"
            aria-expanded={mentionActive && mentionRows.length > 0}
            aria-controls={mentionActive ? 'session-composer-mentions' : undefined}
            aria-activedescendant={
              mentionActive && mentionRows.length
                ? `mention-option-${mentionIndex}`
                : undefined
            }
            oninput={syncCaret}
            onkeyup={syncCaret}
            onclick={syncCaret}
            onselect={syncCaret}
            onkeydown={onComposerKeydown}
          ></textarea>
          <div class="session-composer__toolbar">
            <span class="session-composer__model">{session.provider}:{session.model_id}</span>
            <button
              type="submit"
              class="session-composer__send btn btn--primary"
              disabled={sendDisabled}
              aria-label="Send"
            >Send</button>
          </div>
        </div>
      </form>
    </div>

    {#if filesOpen && showWorkspace && !embeddedInHub}
      {#if !isNarrow}
        <div
          class="session-split__handle"
          role="separator"
          aria-orientation="vertical"
          aria-label="Resize files pane"
          onpointerdown={onHandlePointerDown}
          onpointermove={onHandlePointerMove}
          onpointerup={onHandlePointerUp}
          onpointercancel={onHandlePointerUp}
        ></div>
      {/if}
      {#if isNarrow}
        <button
          type="button"
          class="session-files-backdrop"
          aria-label="Close files"
          onclick={closeFiles}
        ></button>
      {/if}
      <aside class="session-split__files {isNarrow ? 'session-files-drawer' : ''}">
        <SessionFilesBar
          sessionId={session.id}
          {messages}
          {activePath}
          onopen={openFile}
        />
      </aside>
    {/if}
  </div>
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
