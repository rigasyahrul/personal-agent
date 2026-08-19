// web/src/lib/stores/shell-context.test.ts
import { get } from 'svelte/store';
import { describe, expect, it } from 'vitest';
import { deriveShellContext, findVaultName, shellContext } from './shell-context';

const vaults = [{ id: 'v1', name: 'HEALTH' }, { id: 'v2', name: 'WORK' }];

describe('shell context', () => {
  it('looks up a vault name without mutating input', () => {
    expect(findVaultName(vaults, 'v1')).toBe('HEALTH');
    expect(findVaultName(vaults, 'missing')).toBeNull();
  });

  it('derives vault context from every vault route', () => {
    for (const name of ['vault-home', 'vault-projects', 'vault-sessions', 'vault-review'] as const) {
      expect(deriveShellContext({ name, vaultId: 'v1' }, vaults)).toEqual({
        mode: 'vault', vaultId: 'v1', vaultName: 'HEALTH',
      });
    }
  });

  it('uses project membership for project deep links', () => {
    expect(deriveShellContext({ name: 'project', projectId: 'p1' }, vaults, { vault_id: 'v2' }))
      .toEqual({ mode: 'vault', vaultId: 'v2', vaultName: 'WORK' });
    expect(deriveShellContext({ name: 'notes', projectId: 'p2' }, vaults, { vault_id: null }))
      .toEqual({ mode: 'global' });
  });

  it('falls back safely when vault data is unavailable', () => {
    expect(deriveShellContext({ name: 'vault-home', vaultId: 'missing' }, vaults))
      .toEqual({ mode: 'vault', vaultId: 'missing', vaultName: 'Vault' });
    shellContext.set({ mode: 'global' });
    expect(get(shellContext)).toEqual({ mode: 'global' });
  });
});
