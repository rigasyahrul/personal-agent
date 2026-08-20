// web/src/lib/promote.test.ts
import { describe, expect, it } from 'vitest'
import {
  isPromotableWorkspaceFile,
  nextPromoteAttempt,
  safeBadge,
  workspaceEnabled,
} from './promote'

describe('promote helpers', () => {
  it('reuses the idempotency key for an unchanged payload', () => {
    let n = 0
    const uuid = () => `key-${++n}`
    const payload = {
      workspace_path: 'draft.md',
      target_relative_path: 'notes/draft.md',
      review_mode: 'bites',
    }
    const first = nextPromoteAttempt(null, payload, uuid)
    const same = nextPromoteAttempt(first, payload, uuid)
    expect(same.key).toBe(first.key)
    const changed = nextPromoteAttempt(first, { ...payload, review_mode: 'whole' }, uuid)
    expect(changed.key).not.toBe(first.key)
  })

  it('gates workspace from grants object and json', () => {
    expect(workspaceEnabled({ tool_grants: { workspace_files: true } })).toBe(true)
    expect(workspaceEnabled({ tool_grants_json: '{"workspace_files":true}' })).toBe(true)
    expect(workspaceEnabled({ tool_grants_json: '{"workspace_files":false}' })).toBe(false)
    expect(workspaceEnabled({ tool_grants_json: '{bad' })).toBe(false)
  })

  it('only treats regular .md files as promotable', () => {
    expect(isPromotableWorkspaceFile({ kind: 'file', path: 'a.md' })).toBe(true)
    expect(isPromotableWorkspaceFile({ kind: 'file', path: 'a.txt' })).toBe(false)
    expect(isPromotableWorkspaceFile({ kind: 'directory', path: 'a.md' })).toBe(false)
    // Real workspace/file responses often omit kind — still promotable when path ends in .md
    expect(isPromotableWorkspaceFile({ path: 'notes/outline.md' })).toBe(true)
    expect(isPromotableWorkspaceFile({ path: 'raw.txt' })).toBe(false)
  })

  it('clamps unknown badges to Ready', () => {
    expect(safeBadge('Ready')).toBe('Ready')
    expect(safeBadge('Promoting…')).toBe('Promoting…')
    expect(safeBadge('<img onerror=bad>')).toBe('Ready')
  })
})
