// web/src/shell/Sidebar.test.ts
import { cleanup, fireEvent, render, screen, within } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';
import Sidebar from './Sidebar.svelte';

afterEach(cleanup);

describe('Sidebar', () => {
  it('shows global navigation and persists collapse', async () => {
    localStorage.clear();
    render(Sidebar, { props: { context: { mode: 'global' }, route: { name: 'home' } } });
    for (const label of ['Home', 'Projects', 'Sessions', 'Vaults', 'Review', 'Settings']) {
      expect(screen.getByText(label)).toBeInTheDocument();
    }
    await fireEvent.click(screen.getByRole('button', { name: 'Collapse sidebar' }));
    expect(localStorage.getItem('pa.sidebarCollapsed')).toBe('true');
    expect(screen.getByTestId('sidebar')).toHaveAttribute('data-collapsed', 'true');
  });

  it('replaces global navigation in vault context', () => {
    render(Sidebar, {
      props: {
        context: { mode: 'vault', vaultId: 'v1', vaultName: 'HEALTH' },
        route: { name: 'vault-home', vaultId: 'v1' },
      },
    });
    expect(screen.getByText('HEALTH')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Leave vault' })).toHaveAttribute('href', '#/home');
    expect(screen.queryByText('Vaults')).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Projects' })).toHaveAttribute('href', '#/vaults/v1/projects');
  });

  it('uses real nav icons instead of bullet glyphs', () => {
    render(Sidebar, { props: { context: { mode: 'global' }, route: { name: 'home' } } });
    const nav = screen.getByRole('navigation', { name: 'Primary' });
    expect(within(nav).queryAllByText('•')).toHaveLength(0);
    expect(nav.querySelectorAll('svg[aria-hidden="true"]').length).toBeGreaterThanOrEqual(5);
  });

  it('exposes a labeled collapse control without orphan chevron text', () => {
    localStorage.clear();
    render(Sidebar, { props: { context: { mode: 'global' }, route: { name: 'home' } } });
    const collapse = screen.getByRole('button', { name: 'Collapse sidebar' });
    expect(collapse).toHaveAttribute('title', 'Collapse sidebar');
    expect(collapse.textContent?.replace(/\s/g, '')).not.toMatch(/^[‹›]$/);
    expect(collapse.querySelector('svg')).toBeTruthy();
  });

  it('explains disabled global Sessions for assistive tech and sighted users', () => {
    render(Sidebar, { props: { context: { mode: 'global' }, route: { name: 'home' } } });
    const sessions = screen.getByText('Sessions').closest('[aria-disabled="true"]');
    expect(sessions).toBeTruthy();
    expect(sessions).toHaveAttribute('title', 'Open a project to view its sessions');
    expect(sessions).toHaveAttribute(
      'aria-description',
      'Open a project to view its sessions',
    );
  });
});
