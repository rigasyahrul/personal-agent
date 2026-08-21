// web/src/App.test.ts
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App.svelte'
import { api } from './lib/api/client'

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

const authenticated = { status: 'authenticated' as const }

afterEach(() => {
  cleanup()
  window.location.hash = ''
})

beforeEach(() => {
  if (!HTMLDialogElement.prototype.showModal) {
    HTMLDialogElement.prototype.showModal = function showModal() {
      this.setAttribute('open', '')
    }
  }
  if (!HTMLDialogElement.prototype.close) {
    HTMLDialogElement.prototype.close = function close() {
      this.removeAttribute('open')
    }
  }
  vi.mocked(api.get).mockImplementation(async (path: string) => {
    if (path === '/api/v1/vaults') return []
    if (path.startsWith('/api/v1/review/queue')) {
      return { scope: 'all', items: [], caught_up: true }
    }
    if (path === '/health') return { ok: true, storage_writable: true }
    return { generated_at: '', projects: [], due_count: 0 }
  })
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
        props: { authLoader: vi.fn().mockResolvedValue(authenticated) },
      })
      expect(await screen.findByRole('heading', { level: 1, name: heading })).toBeInTheDocument()
    })
  }
})

describe('legacy project sessions route', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input)
        if (path === '/api/v1/vaults') {
          return new Response(JSON.stringify([]), { status: 200 })
        }
        if (path === '/health') {
          return new Response(JSON.stringify({ ok: true, storage_writable: true }), { status: 200 })
        }
        if (path === '/api/v1/projects/p1') {
          return new Response(
            JSON.stringify({
              id: 'p1',
              name: 'Sleep Protocol',
              vault_id: 'v1',
              note_count: 0,
              session_count: 0,
              due_count: 0,
            }),
            { status: 200 },
          )
        }
        if (path === '/api/v1/projects/p1/sessions') {
          return new Response(JSON.stringify([]), { status: 200 })
        }
        if (path === '/api/v1/projects/p1/tree') {
          return new Response(JSON.stringify([]), { status: 200 })
        }
        if (path === '/api/v1/models') {
          return new Response(JSON.stringify({ models: [] }), { status: 200 })
        }
        return new Response(JSON.stringify({}), { status: 200 })
      }),
    )
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('renders ProjectHubPage for #/projects/:id/sessions (not legacy sessions page)', async () => {
    window.location.hash = '#/projects/p1/sessions'
    render(App, {
      props: { authLoader: vi.fn().mockResolvedValue(authenticated) },
    })

    expect(
      await screen.findByRole('heading', { name: 'Sleep Protocol' }),
    ).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /new session/i })).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Title')).not.toBeInTheDocument()
    expect(document.querySelector('.project-workspace')).toBeTruthy()
    expect(document.querySelector('.project-workspace__rail')).toBeTruthy()
    // Icon chrome: Config + Files only (no Memory rail control)
    expect(screen.getByRole('tab', { name: 'Config' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Files' })).toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Memory' })).not.toBeInTheDocument()
  })
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
        props: { authLoader: vi.fn().mockResolvedValue(authenticated) },
      })
      expect(await screen.findByRole('heading', { level: 1, name: heading })).toBeInTheDocument()
    })
  }

  it('shows the real vault name on vault project create (not generic Vault)', async () => {
    vi.mocked(api.get).mockImplementation(async (path: string) => {
      if (path === '/api/v1/vaults') {
        return [{ id: 'v-test', name: 'Test', created_at: '', updated_at: '' }]
      }
      if (path === '/health') return { ok: true, storage_writable: true }
      if (path.startsWith('/api/v1/review/queue')) {
        return { scope: 'all', items: [], caught_up: true }
      }
      return { generated_at: '', projects: [], due_count: 0 }
    })

    window.location.hash = '#/vaults/v-test/projects'
    render(App, { props: { authLoader: vi.fn().mockResolvedValue(authenticated) } })

    expect(await screen.findByRole('heading', { level: 1, name: 'Projects' })).toBeInTheDocument()
    // Sidebar context + page eyebrow must use the real name.
    expect(await screen.findAllByText('Test')).not.toHaveLength(0)
    await fireEvent.click(screen.getAllByRole('button', { name: /new project/i })[0])
    expect(screen.getByLabelText('Vault')).toHaveValue('Test')
  })

  it('refetches vaults when entering a newly created vault missing from the cache', async () => {
    let vaultListCalls = 0
    vi.mocked(api.get).mockImplementation(async (path: string) => {
      if (path === '/api/v1/vaults') {
        vaultListCalls += 1
        // First load (mount on home): empty. After navigate into new vault: includes Test.
        if (vaultListCalls === 1) return []
        return [{ id: 'v-new', name: 'Test', created_at: '', updated_at: '' }]
      }
      if (path === '/health') return { ok: true, storage_writable: true }
      if (path.startsWith('/api/v1/review/queue')) {
        return { scope: 'all', items: [], caught_up: true }
      }
      return { generated_at: '', projects: [], due_count: 0 }
    })

    window.location.hash = '#/home'
    render(App, { props: { authLoader: vi.fn().mockResolvedValue(authenticated) } })
    expect(await screen.findByRole('heading', { level: 1, name: 'Home' })).toBeInTheDocument()

    window.location.hash = '#/vaults/v-new/projects'
    window.dispatchEvent(new HashChangeEvent('hashchange'))

    // Wait until vault projects page is mounted (not Home's "New project").
    expect(await screen.findByRole('heading', { level: 1, name: 'Projects' })).toBeInTheDocument()
    await fireEvent.click(screen.getAllByRole('button', { name: /new project/i })[0])
    await waitFor(() => {
      expect(screen.getByLabelText('Vault')).toHaveValue('Test')
    })
    expect(vaultListCalls).toBeGreaterThanOrEqual(2)
  })
})
