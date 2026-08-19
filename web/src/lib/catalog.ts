// web/src/lib/catalog.ts
import type { Project } from './api/types'
import type { ShellContext } from './stores/shell-context'

export function isUnfiled(p: { vault_id?: string | null }): boolean {
  return !p.vault_id
}

export function filterProjectsByContext(
  projects: Array<{ vault_id?: string | null }>,
  ctx: ShellContext,
): typeof projects {
  return ctx.mode === 'global'
    ? projects.filter(isUnfiled)
    : projects.filter((project) => project.vault_id === ctx.vaultId)
}

export function filterByQuery<T extends { name: string }>(items: T[], query: string): T[] {
  const normalized = query.trim().toLocaleLowerCase()
  if (!normalized) return items
  return items.filter((item) => item.name.toLocaleLowerCase().includes(normalized))
}

export function filterReviewByProjectIds<T extends { project_id: string }>(
  items: T[],
  projectIds: Set<string>,
): T[] {
  return items.filter((item) => projectIds.has(item.project_id))
}

// Re-export Project for callers that want typed arrays
export type { Project }
