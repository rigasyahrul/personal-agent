// web/src/lib/stores/shell-context.ts
import { writable } from 'svelte/store';
import type { AppRoute } from '../router';

export type ShellContext =
  | { mode: 'global' }
  | { mode: 'vault'; vaultId: string; vaultName: string };

export type VaultSummary = { id: string; name: string };
export type ProjectMembership = { vault_id?: string | null };

export const shellContext = writable<ShellContext>({ mode: 'global' });

export function findVaultName(vaults: readonly VaultSummary[], vaultId: string): string | null {
  return vaults.find((vault) => vault.id === vaultId)?.name ?? null;
}

export function deriveShellContext(
  route: AppRoute,
  vaults: readonly VaultSummary[],
  project?: ProjectMembership,
): ShellContext {
  if (route.name.startsWith('vault-')) {
    return { mode: 'vault', vaultId: route.vaultId, vaultName: findVaultName(vaults, route.vaultId) ?? 'Vault' };
  }
  if (['project', 'notes', 'note', 'sessions', 'project-review'].includes(route.name) && project?.vault_id) {
    return {
      mode: 'vault',
      vaultId: project.vault_id,
      vaultName: findVaultName(vaults, project.vault_id) ?? 'Vault',
    };
  }
  return { mode: 'global' };
}
