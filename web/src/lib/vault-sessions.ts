// web/src/lib/vault-sessions.ts
import type { Project, Session } from './api/types'
import { filterVaultProjects } from './vault-scope'

export type VaultSession = Session & { project_id: string; project_name: string }

export type VaultSessionsApi = {
  listProjects: () => Promise<Project[]>
  listProjectSessions: (projectId: string) => Promise<Session[]>
}

export async function loadVaultSessions(
  vaultId: string,
  api: VaultSessionsApi,
): Promise<{ projects: Project[]; sessions: VaultSession[]; failures: string[] }> {
  const all = await api.listProjects()
  const projects = filterVaultProjects(all, vaultId)
  const sessions: VaultSession[] = []
  const failures: string[] = []

  const settled = await Promise.allSettled(
    projects.map((project) => api.listProjectSessions(project.id)),
  )

  settled.forEach((result, index) => {
    const project = projects[index]
    if (result.status === 'rejected') {
      failures.push(project.name)
      return
    }
    sessions.push(
      ...result.value.map((session) => ({
        ...session,
        project_id: project.id,
        project_name: project.name,
      })),
    )
  })

  return { projects, sessions, failures }
}
