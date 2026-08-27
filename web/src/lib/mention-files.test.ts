import { describe, expect, it } from 'vitest'
import { activeMention, insertMention, rankWorkspaceFiles } from './mention-files'
import type { WorkspaceEntry } from './api/types'

describe('activeMention', () => {
  it('detects @ at start and after whitespace, including newlines', () => {
    expect(activeMention('@', 1)).toEqual({ start: 0, end: 1, query: '' })
    expect(activeMention('@stand', 6)).toEqual({ start: 0, end: 6, query: 'stand' })
    expect(activeMention('see @stand', 10)).toEqual({ start: 4, end: 10, query: 'stand' })
    expect(activeMention('see\n@x', 6)).toEqual({ start: 4, end: 6, query: 'x' })
  })

  it('uses the whole token when the cursor is in the middle', () => {
    const text = '@standing-rule.md'
    expect(activeMention(text, 7)).toEqual({ start: 0, end: 17, query: 'standing-rule.md' })
  })

  it('ignores foo@bar and a cursor not inside an @ token', () => {
    expect(activeMention('foo@bar', 7)).toBeNull()
    expect(activeMention('hello @x more', 5)).toBeNull()
    expect(activeMention('hello @x more', 13)).toBeNull()
  })
})

describe('rankWorkspaceFiles', () => {
  const entries: WorkspaceEntry[] = [
    { path: 'standing-rule.md', kind: 'file' },
    { path: 'notes/standing-rule.md', kind: 'file' },
    { path: 'notes', kind: 'directory' },
    { path: 'other.md', kind: 'file' },
    { path: 'notes/deep/alpha.md', kind: 'file' },
  ]

  it('drops directories and ranks basename starts-with before path substring', () => {
    const rows = rankWorkspaceFiles(entries, 'stand')
    expect(rows.map((r) => r.path)).toEqual(['notes/standing-rule.md', 'standing-rule.md'])
    expect(rows[0]).toEqual({
      path: 'notes/standing-rule.md',
      name: 'standing-rule.md',
      parent: 'notes/',
    })
    expect(rows[1]?.parent).toBe('')
  })

  it('matches path substrings and caps at 10', () => {
    expect(rankWorkspaceFiles(entries, 'notes').map((r) => r.path)).toEqual([
      'notes/deep/alpha.md',
      'notes/standing-rule.md',
    ])
    const many: WorkspaceEntry[] = Array.from({ length: 12 }, (_, i) => ({
      path: `f${String(i).padStart(2, '0')}.md`,
      kind: 'file' as const,
    }))
    expect(rankWorkspaceFiles(many, '').map((r) => r.path)).toHaveLength(10)
  })
})

describe('insertMention', () => {
  it('replaces the whole token and adds a trailing space', () => {
    const mention = activeMention('what is in @stand', 17)
    expect(mention).not.toBeNull()
    expect(insertMention('what is in @stand', mention!, 'notes/standing-rule.md')).toEqual({
      text: 'what is in @notes/standing-rule.md ',
      cursor: 35,
    })
  })

  it('preserves text after the token', () => {
    const text = 'see @stand please'
    const mention = activeMention(text, 10)
    expect(insertMention(text, mention!, 'a.md')).toEqual({
      text: 'see @a.md  please',
      cursor: 10,
    })
  })
})
