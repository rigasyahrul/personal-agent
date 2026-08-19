// web/src/lib/stores/auth.ts
import { writable } from 'svelte/store';
import { APIError, get } from '../api/client';

export type AuthState =
  | { status: 'loading' }
  | { status: 'bootstrap' }
  | { status: 'login' }
  | { status: 'authenticated' }
  | { status: 'error'; message: string };
type Client = <T>(path: string) => Promise<T | null>;

export const authState = writable<AuthState>({ status: 'loading' });

export async function loadAuthState(client: Client = get): Promise<AuthState> {
  try {
    const setup = await client<{ bootstrapped: boolean }>('/api/v1/setup/status');
    if (!setup?.bootstrapped) return { status: 'bootstrap' };
    try {
      await client<{ owner: boolean }>('/api/v1/auth/me');
      return { status: 'authenticated' };
    } catch (error) {
      if (error instanceof APIError && error.status === 401) return { status: 'login' };
      throw error;
    }
  } catch (error) {
    return { status: 'error', message: error instanceof Error ? error.message : 'Could not start the app' };
  }
}

export async function refreshAuth(): Promise<void> {
  authState.set({ status: 'loading' });
  authState.set(await loadAuthState());
}
