import { describe, expect, it } from 'vitest'
import type { ChatMessage, RunStatus } from './api/types'
import {
  buildThoughtsView,
  chipInsertAfterSequence,
  parseToolCalls,
  runElapsedSec,
  shouldShowChip,
  toolArg,
  toolStatus,
  toolVerb,
} from './thoughts'

const calls = JSON.stringify([
  { id: 'c1', name: 'read_knowledge', arguments: '{"path":"source/standing-rule.md"}' },
])

const msgs: ChatMessage[] = [
  { sequence: 1, role: 'user', content: 'what is in @standing-rule.md ?', run_id: 'r1', created_at: '2026-08-27T00:00:00Z' },
  { sequence: 2, role: 'assistant', content: '', run_id: 'r1', tool_calls_json: calls, created_at: '2026-08-27T00:00:05Z' },
  { sequence: 3, role: 'tool', content: '{"path":"source/standing-rule.md","content":"# hi"}', run_id: 'r1', tool_call_id: 'c1', created_at: '2026-08-27T00:00:06Z' },
  { sequence: 4, role: 'assistant', content: 'Here is the content.', run_id: 'r1', created_at: '2026-08-27T00:00:10Z' },
]

describe('thoughts', () => {
  it('parses flat tool_calls_json and maps verb/arg', () => {
    expect(parseToolCalls(calls)).toEqual([
      { id: 'c1', name: 'read_knowledge', arguments: '{"path":"source/standing-rule.md"}' },
    ])
    expect(toolVerb('read_knowledge')).toBe('Read')
    expect(toolVerb('list_knowledge')).toBe('List')
    expect(toolArg('{"path":"source/standing-rule.md"}')).toBe('source/standing-rule.md')
    expect(toolArg('{"q":"hello world that is quite a long query for the rail"}').length).toBeLessThanOrEqual(49)
  })

  it('skips nameless empty calls', () => {
    expect(parseToolCalls(JSON.stringify([{ id: 'x', name: '', arguments: '' }]))).toEqual([])
  })

  it('statuses error vs ok vs pending', () => {
    expect(toolStatus('{"error":"workspace tool request rejected"}', false)).toEqual({
      status: 'error',
      detail: 'workspace tool request rejected',
    })
    expect(toolStatus('{"ok":true}', false).status).toBe('ok')
    expect(toolStatus(undefined, true).status).toBe('pending')
  })

  it('shows a live chip and past chip with tools; hides instant no-tool', () => {
    const live: RunStatus = { id: 'r1', status: 'running', started_at: '2026-08-27T00:00:00Z' }
    expect(shouldShowChip({ runId: 'r1', messages: msgs, current: live, nowMs: Date.parse('2026-08-27T00:00:32Z') })).toBe(true)
    expect(runElapsedSec({ runId: 'r1', messages: msgs, current: live, nowMs: Date.parse('2026-08-27T00:00:32Z') })).toBe(32)
    expect(shouldShowChip({ runId: 'r1', messages: msgs, current: null, nowMs: 0 })).toBe(true)
    expect(runElapsedSec({ runId: 'r1', messages: msgs, current: null, nowMs: 0 })).toBe(10)
    const instant: ChatMessage[] = [
      { sequence: 1, role: 'user', content: 'hi', run_id: 'r2', created_at: '2026-08-27T00:00:00Z' },
      { sequence: 2, role: 'assistant', content: 'hey', run_id: 'r2', created_at: '2026-08-27T00:00:00Z' },
    ]
    expect(shouldShowChip({ runId: 'r2', messages: instant, current: null, nowMs: 0 })).toBe(false)
  })

  it('builds one-run rows and insert-after user sequence', () => {
    const view = buildThoughtsView({ runId: 'r1', messages: msgs, current: null, nowMs: 0 })
    expect(view).toEqual({
      runId: 'r1',
      elapsedSec: 10,
      live: false,
      rows: [{ id: 'c1', verb: 'Read', arg: 'source/standing-rule.md', status: 'ok' }],
    })
    expect(chipInsertAfterSequence(msgs, 'r1')).toBe(1)
  })
})
