// web/src/lib/stores/auth.test.ts
import { describe, expect, it, vi } from 'vitest';
import { APIError } from '../api/client';
import { loadAuthState } from './auth';

describe('loadAuthState', () => {
  it('requests setup first and stops at bootstrap', async () => {
    const client = vi.fn().mockResolvedValueOnce({ bootstrapped: false });
    await expect(loadAuthState(client)).resolves.toEqual({ status: 'bootstrap' });
    expect(client).toHaveBeenCalledTimes(1);
    expect(client).toHaveBeenCalledWith('/api/v1/setup/status');
  });

  it('loads the owner after setup', async () => {
    const client = vi.fn()
      .mockResolvedValueOnce({ bootstrapped: true })
      .mockResolvedValueOnce({ owner: true });
    await expect(loadAuthState(client)).resolves.toEqual({ status: 'authenticated' });
    expect(client).toHaveBeenNthCalledWith(2, '/api/v1/auth/me');
  });

  it('maps a 401 from auth/me to login', async () => {
    const client = vi.fn().mockResolvedValueOnce({ bootstrapped: true })
      .mockRejectedValueOnce(new APIError(401, 'unauthorized'));
    await expect(loadAuthState(client)).resolves.toEqual({ status: 'login' });
  });
});
