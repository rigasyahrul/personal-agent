// web/src/shell/AppShell.test.ts
import { cleanup, render, screen } from '@testing-library/svelte';
import { afterEach, expect, it } from 'vitest';
import AppShellHarness from './AppShellHarness.svelte';

afterEach(cleanup);

it('renders sidebar, top bar health, and content canvas', () => {
  render(AppShellHarness, {
    props: {
      context: { mode: 'global' },
      route: { name: 'home' },
      health: 'Storage ready',
    },
  });
  expect(screen.getByRole('navigation', { name: 'Primary' })).toBeInTheDocument();
  expect(screen.getByText('Storage ready')).toBeInTheDocument();
  expect(screen.getByRole('main')).toBeInTheDocument();
});
