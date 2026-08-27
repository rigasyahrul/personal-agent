import type { WorkspaceEntry } from './api/types'

export type MentionRange = {
  start: number
  end: number
  query: string
}

export type RankedFile = {
  path: string
  name: string
  parent: string
}

const DEFAULT_LIMIT = 10
const WS = /[ \t\n]/

export function activeMention(text: string, cursor: number): MentionRange | null {
  if (cursor < 0 || cursor > text.length) return null
  let start = cursor
  while (start > 0 && !WS.test(text[start - 1]!)) start -= 1
  let end = cursor
  while (end < text.length && !WS.test(text[end]!)) end += 1
  const token = text.slice(start, end)
  if (!token.startsWith('@')) return null
  return { start, end, query: token.slice(1) }
}

function basename(path: string): string {
  const i = path.lastIndexOf('/')
  return i < 0 ? path : path.slice(i + 1)
}

function parentDir(path: string): string {
  const i = path.lastIndexOf('/')
  return i < 0 ? '' : path.slice(0, i + 1)
}

export function rankWorkspaceFiles(
  entries: WorkspaceEntry[],
  query: string,
  limit = DEFAULT_LIMIT,
): RankedFile[] {
  const q = query.trim().toLowerCase()
  const files = entries.filter((entry) => entry.kind === 'file')
  const matched = q
    ? files.filter((entry) => entry.path.toLowerCase().includes(q))
    : files
  const scored = matched.map((entry) => {
    const name = basename(entry.path)
    const n = name.toLowerCase()
    let rank = 3
    if (!q || n.startsWith(q)) rank = 1
    else if (n.includes(q)) rank = 2
    return { path: entry.path, name, parent: parentDir(entry.path), rank }
  })
  scored.sort((a, b) => a.rank - b.rank || a.path.localeCompare(b.path))
  return scored.slice(0, limit).map(({ rank: _rank, ...row }) => row)
}

export function insertMention(
  text: string,
  mention: MentionRange,
  path: string,
): { text: string; cursor: number } {
  const inserted = `@${path} `
  return {
    text: text.slice(0, mention.start) + inserted + text.slice(mention.end),
    cursor: mention.start + inserted.length,
  }
}
