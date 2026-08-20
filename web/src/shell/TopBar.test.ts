// web/src/shell/TopBar.test.ts
import { cleanup, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';
import TopBar from './TopBar.svelte';

afterEach(cleanup);

describe('TopBar', () => {
  it('renders a quiet unknown health state without alarming copy', () => {
    render(TopBar, { props: { health: 'unknown' } });
    const pill = screen.getByTestId('health-pill');
    expect(pill).toHaveAttribute('data-tone', 'muted');
    expect(pill).toHaveTextContent('Storage');
    expect(pill.textContent).not.toMatch(/unavailable/i);
  });

  it('shows ready and warning labels when status is known', () => {
    const { unmount } = render(TopBar, { props: { health: 'ready' } });
    expect(screen.getByTestId('health-pill')).toHaveTextContent('Storage ready');
    expect(screen.getByTestId('health-pill')).toHaveAttribute('data-tone', 'ok');
    unmount();

    render(TopBar, { props: { health: 'error' } });
    expect(screen.getByTestId('health-pill')).toHaveTextContent('Storage issue');
    expect(screen.getByTestId('health-pill')).toHaveAttribute('data-tone', 'warn');
  });
});
