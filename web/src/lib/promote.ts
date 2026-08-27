// web/src/lib/promote.ts
import type { PromotePayload } from './api/types'

export type PromoteAttempt = {
  fingerprint: string
  key: string
  payload: PromotePayload
}

export function nextPromoteAttempt(
  previous: PromoteAttempt | null | undefined,
  payload: PromotePayload,
  uuid: () => string,
): PromoteAttempt {
  const fingerprint = JSON.stringify(payload)
  if (previous?.fingerprint === fingerprint) return previous
  return { fingerprint, key: uuid(), payload }
}

export function operationStorageKey(sessionId: string): string {
  return `personal-agent:v1:promote-operations:${sessionId}`
}

export function loadOperationIds(sessionId: string, storage: Storage | null | undefined): string[] {
  if (!storage) return []
  try {
    const raw = JSON.parse(storage.getItem(operationStorageKey(sessionId)) || '[]')
    return Array.isArray(raw) ? raw.filter((item): item is string => typeof item === 'string') : []
  } catch {
    return []
  }
}

export function saveOperationIds(
  sessionId: string,
  operations: string[],
  storage: Storage | null | undefined,
): void {
  if (!storage) return
  try {
    storage.setItem(operationStorageKey(sessionId), JSON.stringify(operations))
  } catch {
    /* storage may be unavailable */
  }
}

export function isPromotableWorkspaceFile(entry: { kind?: string; path?: string } | null | undefined): boolean {
  if (!entry || typeof entry.path !== 'string' || !entry.path.endsWith('.md')) return false
  // Workspace file API may omit kind; treat missing kind as a regular file.
  // Directories are never promotable.
  if (entry.kind === 'directory' || entry.kind === 'folder') return false
  return entry.kind === undefined || entry.kind === '' || entry.kind === 'file'
}

export function workspaceEnabled(session: {
  tool_grants?: { workspace_files?: boolean; session_files?: boolean } | null
  tool_grants_json?: string | null
} | null | undefined): boolean {
  if (session?.tool_grants && typeof session.tool_grants === 'object') {
    return session.tool_grants.workspace_files === true || session.tool_grants.session_files === true
  }
  if (typeof session?.tool_grants_json !== 'string') return false
  try {
    const grants = JSON.parse(session.tool_grants_json)
    return grants?.workspace_files === true || grants?.session_files === true
  } catch {
    return false
  }
}

export const BADGE_COPIES = [
  'Promoting…',
  'Promote failed — Retry',
  'Note saved; cards pending…',
  'Cards failed — Retry cards',
  'Ready',
] as const

export function safeBadge(badge: string | undefined): string {
  return BADGE_COPIES.includes(badge as (typeof BADGE_COPIES)[number]) ? (badge as string) : 'Ready'
}

export function isTerminalBadge(badge: string | undefined): boolean {
  return (
    badge === 'Ready' ||
    badge === 'Promote failed — Retry' ||
    badge === 'Cards failed — Retry cards'
  )
}
