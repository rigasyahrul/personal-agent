// web/src/lib/workspace-tree.ts
import type { ChatMessage, WorkspaceEntry } from './api/types'

export type TreeNode = {
  path: string
  name: string
  kind: 'file' | 'directory'
  children: TreeNode[]
}

/** Case-insensitive substring filter on entry paths. */
export function filterEntriesByQuery(
  entries: WorkspaceEntry[],
  query: string,
): WorkspaceEntry[] {
  const q = query.trim().toLowerCase()
  if (!q) return entries
  return entries.filter((entry) => entry.path.toLowerCase().includes(q))
}

/** Collect changed_path values from tool-role messages (field or JSON content). */
export function changedPathsFromMessages(list: ChatMessage[]): Set<string> {
  const paths = new Set<string>()
  for (const message of list) {
    if (message?.role !== 'tool') continue
    let path = message.changed_path
    if (!path && typeof message.content === 'string') {
      try {
        path = JSON.parse(message.content)?.changed_path
      } catch {
        path = ''
      }
    }
    if (typeof path === 'string' && path) paths.add(path)
  }
  return paths
}

/**
 * Build a nested tree from flat workspace entries.
 * Path segments become directory nodes when needed.
 */
export function buildHierarchy(entries: WorkspaceEntry[]): TreeNode[] {
  type Mutable = {
    path: string
    name: string
    kind: 'file' | 'directory'
    children: Map<string, Mutable>
  }

  const root = new Map<string, Mutable>()

  function ensureDir(map: Map<string, Mutable>, path: string, name: string): Mutable {
    let node = map.get(name)
    if (!node) {
      node = { path, name, kind: 'directory', children: new Map() }
      map.set(name, node)
    } else if (node.kind === 'file') {
      // Prefer directory when both appear
      node.kind = 'directory'
    }
    return node
  }

  for (const entry of entries) {
    const parts = entry.path.split('/').filter(Boolean)
    if (!parts.length) continue
    let map = root
    let prefix = ''
    for (let i = 0; i < parts.length; i++) {
      const name = parts[i]!
      prefix = prefix ? `${prefix}/${name}` : name
      const isLeaf = i === parts.length - 1
      if (isLeaf) {
        const kind = entry.kind === 'directory' ? 'directory' : 'file'
        const existing = map.get(name)
        if (existing) {
          if (kind === 'directory') existing.kind = 'directory'
          else if (existing.kind !== 'directory') existing.kind = 'file'
        } else {
          map.set(name, {
            path: prefix,
            name,
            kind,
            children: new Map(),
          })
        }
      } else {
        const dir = ensureDir(map, prefix, name)
        map = dir.children
      }
    }
  }

  function freeze(map: Map<string, Mutable>): TreeNode[] {
    const nodes = [...map.values()].map((node) => ({
      path: node.path,
      name: node.name,
      kind: node.kind,
      children: freeze(node.children),
    }))
    nodes.sort((a, b) => {
      if (a.kind !== b.kind) return a.kind === 'directory' ? 1 : -1
      return a.path.localeCompare(b.path)
    })
    return nodes
  }

  return freeze(root)
}

/** Flatten tree back to display rows (depth-first), for simple list UIs. */
export function flattenTree(nodes: TreeNode[], depth = 0): Array<TreeNode & { depth: number }> {
  const rows: Array<TreeNode & { depth: number }> = []
  for (const node of nodes) {
    rows.push({ ...node, depth })
    if (node.children.length) rows.push(...flattenTree(node.children, depth + 1))
  }
  return rows
}
