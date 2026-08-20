// web/src/lib/vault-scope.ts
import type { Project } from './api/types'

export function filterVaultProjects(projects: Project[], vaultId: string): Project[] {
  return projects.filter((project) => project.vault_id === vaultId)
}

export function createVaultProjectInput(name: string, vaultId: string): { name: string; vault_id: string } {
  return { name: name.trim(), vault_id: vaultId }
}
