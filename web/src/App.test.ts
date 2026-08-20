// web/src/App.test.ts
import { cleanup, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'
import App from './App.svelte'

vi.mock('./lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./lib/api/client')>()
  return {
    ...actual,
    api: {
      get: vi.fn().mockImplementation(async (path: string) => {
        if (path === '/api/v1/vaults') return []
        if (path.startsWith('/api/v1/review/queue')) {
          return { scope: 'all', items: [], caught_up: true }
        }
        return { generated_at: '', projects: [], due_count: 0 }
      }),
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

describe('vault context routes', () => {
  for (const [hash, heading] of [
    ['#/vaults/v1', 'Vault'],
    ['#/vaults/v1/projects', 'Projects'],
    ['#/vaults/v1/sessions', 'Sessions'],
    ['#/vaults/v1/review', 'Review'],
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
