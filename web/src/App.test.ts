// web/src/App.test.ts
import { cleanup, render, screen, waitFor } from '@testing-library/svelte';
import { afterEach, expect, it, vi } from 'vitest';
import App from './App.svelte';

afterEach(cleanup);

it('renders login without authenticated chrome', async () => {
  render(App, { props: { authLoader: vi.fn().mockResolvedValue({ status: 'login' }) } });
  await waitFor(() => expect(screen.getByRole('heading', { name: 'Sign in' })).toBeInTheDocument());
  expect(screen.queryByRole('navigation', { name: 'Primary' })).not.toBeInTheDocument();
});
