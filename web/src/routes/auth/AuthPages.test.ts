// web/src/routes/auth/AuthPages.test.ts
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import BootstrapPage from './BootstrapPage.svelte';
import LoginPage from './LoginPage.svelte';

afterEach(cleanup);

describe('auth pages', () => {
  it('submits bootstrap token and a 12+ character password', async () => {
    const submit = vi.fn().mockResolvedValue(null);
    render(BootstrapPage, { props: { submit, onComplete: vi.fn() } });
    await fireEvent.input(screen.getByLabelText('Bootstrap token'), { target: { value: 'token' } });
    await fireEvent.input(screen.getByLabelText('Password'), { target: { value: 'long-enough-password' } });
    await fireEvent.submit(screen.getByRole('form', { name: 'Set up owner account' }));
    expect(submit).toHaveBeenCalledWith('/api/v1/setup/bootstrap', 'POST', {
      token: 'token', password: 'long-enough-password',
    });
  });

  it('shows login errors beside the form', async () => {
    render(LoginPage, {
      props: {
        submit: vi.fn().mockRejectedValue(new Error('Incorrect password')),
        onComplete: vi.fn(),
      },
    });
    await fireEvent.input(screen.getByLabelText('Password'), { target: { value: 'wrong' } });
    await fireEvent.submit(screen.getByRole('form', { name: 'Sign in' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('Incorrect password');
  });
});
