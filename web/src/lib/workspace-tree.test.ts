// web/src/lib/workspace-tree.test.ts
import { describe, expect, it } from 'vitest'
import {
  buildHierarchy,
  changedPathsFromMessages,
  filterEntriesByQuery,
  type TreeNode,
} from './workspace-tree'
import type { ChatMessage, WorkspaceEntry } from './api/types'

describe('buildHierarchy', () => {
  it('returns empty for empty entries', () => {
    expect(buildHierarchy([])).toEqual([])
  })

  it('keeps flat root files', () => {
    const entries: WorkspaceEntry[] = [
      { path: 'a.md', kind: 'file' },
      { path: 'b.txt', kind: 'file' },
    ]
    const tree = buildHierarchy(entries)
    expect(tree.map((n) => n.path)).toEqual(['a.md', 'b.txt'])
    expect(tree.every((n) => n.kind === 'file' && n.children.length === 0)).toBe(true)
  })

  it('nests path segments into directory nodes', () => {
    const entries: WorkspaceEntry[] = [
      { path: 'notes/raw.txt', kind: 'file' },
      { path: 'notes/deep/x.md', kind: 'file' },
      { path: 'draft.md', kind: 'file' },
    ]
    const tree = buildHierarchy(entries)
    expect(tree.map((n) => n.path)).toEqual(['draft.md', 'notes'])
    const notes = tree.find((n) => n.path === 'notes') as TreeNode
    expect(notes.kind).toBe('directory')
    expect(notes.children.map((c) => c.path).sort()).toEqual(['notes/deep', 'notes/raw.txt'])
    const deep = notes.children.find((c) => c.path === 'notes/deep') as TreeNode
    expect(deep.children.map((c) => c.path)).toEqual(['notes/deep/x.md'])
  })

  it('respects explicit directory entries without inventing file children', () => {
    const entries: WorkspaceEntry[] = [
      { path: 'empty-dir', kind: 'directory' },
      { path: 'empty-dir/later.txt', kind: 'file' },
    ]
    const tree = buildHierarchy(entries)
    const dir = tree.find((n) => n.path === 'empty-dir') as TreeNode
    expect(dir.kind).toBe('directory')
    expect(dir.children.map((c) => c.path)).toEqual(['empty-dir/later.txt'])
  })
})

describe('filterEntriesByQuery', () => {
  const entries: WorkspaceEntry[] = [
    { path: 'draft.md', kind: 'file' },
    { path: 'notes/Raw.txt', kind: 'file' },
    { path: 'notes', kind: 'directory' },
  ]

  it('returns all when query is blank', () => {
    expect(filterEntriesByQuery(entries, '')).toEqual(entries)
    expect(filterEntriesByQuery(entries, '   ')).toEqual(entries)
  })

  it('filters case-insensitively by path substring', () => {
    expect(filterEntriesByQuery(entries, 'raw').map((e) => e.path)).toEqual(['notes/Raw.txt'])
    expect(filterEntriesByQuery(entries, 'NOTES').map((e) => e.path).sort()).toEqual([
      'notes',
      'notes/Raw.txt',
    ])
  })
})

describe('changedPathsFromMessages', () => {
  it('collects only tool role paths from field or JSON content', () => {
    const messages: ChatMessage[] = [
      { sequence: 1, role: 'user', content: '', changed_path: 'ignored.txt' },
      { sequence: 2, role: 'tool', content: '', changed_path: 'draft.md' },
      { sequence: 3, role: 'tool', content: '{"changed_path":"notes/raw.txt"}' },
      { sequence: 4, role: 'tool', content: 'not-json' },
    ]
    expect([...changedPathsFromMessages(messages)].sort()).toEqual(['draft.md', 'notes/raw.txt'])
  })
})
