import type { ChatMessage, RunStatus } from './api/types'

export type ThoughtRow = {
  id: string
  verb: string
  arg: string
  status?: 'ok' | 'error' | 'pending'
  detail?: string
}

export type ThoughtsView = {
  runId: string
  elapsedSec: number
  live: boolean
  rows: ThoughtRow[]
}

type ToolCall = {
  id: string
  name: string
  arguments: string
}

type ThoughtInput = {
  runId: string
  messages: ChatMessage[]
  current: RunStatus | null
  nowMs: number
}

const VERBS: Record<string, string> = {
  read_file: 'Read',
  read_knowledge: 'Read',
  list_knowledge: 'List',
  list_dir: 'List',
  write_file: 'Write',
  edit_file: 'Write',
  mkdir: 'Mkdir',
}

export function parseToolCalls(json: string | undefined): ToolCall[] {
  if (!json) return []
  let parsed: unknown
  try {
    parsed = JSON.parse(json)
  } catch {
    return []
  }
  if (!Array.isArray(parsed)) return []
  const calls: ToolCall[] = []
  for (const item of parsed) {
    if (!item || typeof item !== 'object') continue
    const rec = item as Record<string, unknown>
    const id = typeof rec.id === 'string' ? rec.id : ''
    const name = typeof rec.name === 'string' ? rec.name : ''
    const args = typeof rec.arguments === 'string' ? rec.arguments : ''
    if (!name && !args) continue
    calls.push({ id, name, arguments: args })
  }
  return calls
}

export function toolVerb(name: string): string {
  if (VERBS[name]) return VERBS[name]
  if (name.includes('_')) {
    return name
      .split('_')
      .map((word, i) => (i === 0 ? capitalize(word) : word))
      .join(' ')
  }
  return name
}

export function toolArg(argumentsJson: string): string {
  let obj: unknown
  try {
    obj = JSON.parse(argumentsJson)
  } catch {
    return ''
  }
  if (!obj || typeof obj !== 'object') return ''
  const rec = obj as Record<string, unknown>
  if (typeof rec.path === 'string') return truncateArg(rec.path)
  for (const value of Object.values(rec)) {
    if (typeof value === 'string') return truncateArg(value)
  }
  return ''
}

export function toolStatus(
  content: string | undefined,
  live: boolean,
): { status: 'ok' | 'error' | 'pending'; detail?: string } {
  if (content) {
    try {
      const parsed = JSON.parse(content) as { error?: unknown }
      if (typeof parsed?.error === 'string') {
        return { status: 'error', detail: parsed.error.slice(0, 80) }
      }
    } catch {
      /* not JSON */
    }
    if (content.includes('error')) {
      return { status: 'error', detail: content.slice(0, 80) }
    }
    return { status: 'ok' }
  }
  if (live) return { status: 'pending' }
  return { status: 'ok' }
}

export function runElapsedSec(input: ThoughtInput): number {
  const runMsgs = messagesForRun(input.messages, input.runId)
  if (isLive(input.current, input.runId)) {
    const start = input.current?.started_at || runMsgs[0]?.created_at
    const startMs = start ? Date.parse(start) : input.nowMs
    if (!Number.isFinite(startMs)) return 0
    return Math.max(0, Math.floor((input.nowMs - startMs) / 1000))
  }
  const firstMs = Date.parse(runMsgs[0]?.created_at || '')
  const lastMs = Date.parse(runMsgs[runMsgs.length - 1]?.created_at || '')
  let elapsed = 0
  if (Number.isFinite(firstMs) && Number.isFinite(lastMs)) {
    elapsed = Math.floor((lastMs - firstMs) / 1000)
  }
  if (hasToolCalls(runMsgs) && elapsed < 1) elapsed = 1
  return Math.max(0, elapsed)
}

export function shouldShowChip(input: ThoughtInput): boolean {
  if (isLive(input.current, input.runId)) return true
  const runMsgs = messagesForRun(input.messages, input.runId)
  return hasToolCalls(runMsgs) || runElapsedSec(input) >= 1
}

export function buildThoughtsView(input: ThoughtInput): ThoughtsView {
  const live = isLive(input.current, input.runId)
  const runMsgs = messagesForRun(input.messages, input.runId)
  const rows: ThoughtRow[] = []
  for (const message of runMsgs) {
    for (const call of parseToolCalls(message.tool_calls_json)) {
      const result = runMsgs.find((m) => m.role === 'tool' && m.tool_call_id === call.id)
      const st = toolStatus(result?.content, live)
      const row: ThoughtRow = {
        id: call.id,
        verb: toolVerb(call.name),
        arg: toolArg(call.arguments),
        status: st.status,
      }
      if (st.detail) row.detail = st.detail
      rows.push(row)
    }
  }
  return {
    runId: input.runId,
    elapsedSec: runElapsedSec(input),
    live,
    rows,
  }
}

export function chipInsertAfterSequence(messages: ChatMessage[], runId: string): number {
  const runMsgs = messagesForRun(messages, runId)
  const user = runMsgs.find((m) => m.role === 'user')
  if (user) return user.sequence
  return runMsgs[0] ? runMsgs[0].sequence - 1 : 0
}

function isLive(current: RunStatus | null | undefined, runId: string): boolean {
  if (!current) return false
  return (current.status === 'queued' || current.status === 'running') && current.id === runId
}

function messagesForRun(messages: ChatMessage[], runId: string): ChatMessage[] {
  return messages.filter((m) => m.run_id === runId)
}

function hasToolCalls(messages: ChatMessage[]): boolean {
  return messages.some((m) => parseToolCalls(m.tool_calls_json).length > 0)
}

function truncateArg(value: string): string {
  if (value.length <= 48) return value
  return `${value.slice(0, 48)}…`
}

function capitalize(word: string): string {
  if (!word) return word
  return word[0].toUpperCase() + word.slice(1)
}
