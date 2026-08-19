// web/src/App.test.ts
import { cleanup, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'
import App from './App.svelte'

vi.mock('./lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./lib/api/client')>()
  return {
    ...actual,
    api: {
      get: vi.fn().mockResolvedValue({ generated_at: '', projects: [], due_count: 0 }),
      post: vi.fn(),
    },
  }
})

afterEach(() => {
  cleanup()
  window.location.hash = ''
})

it('renders login without authenticated chrome', async () => {
  render(App, { props: { authLoader: vi.fn().mockResolvedValue({ status: 'login' }) } })
  await waitFor(() => expect(screen.getByRole('heading', { name: 'Sign in' })).toBeInTheDocument())
  expect(screen.queryByRole('navigation', { name: 'Primary' })).not.toBeInTheDocument()
})

describe('global catalog routes', () => {
  for (const [hash, heading] of [
    ['#/home', 'Home'],
    ['#/projects', 'Projects'],
    ['#/vaults', 'Vaults'],
  ] as const) {
    it(`renders ${hash}`, async () => {
      window.location.hash = hash
      render(App, {
        props: { authLoader: vi.fn().mockResolvedValue({ status: 'authenticated' }) },
      })
      expect(await screen.findByRole('heading', { level: 1, name: heading })).toBeInTheDocument()
    })
  }
})
