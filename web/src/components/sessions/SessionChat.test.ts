// web/src/components/sessions/SessionChat.test.ts
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SessionChat from './SessionChat.svelte'
import { api } from '../../lib/api'

vi.mock('../../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      listMessages: vi.fn(),
      currentRun: vi.fn(),
      sendMessage: vi.fn(),
      workspaceTree: vi.fn(),
      workspaceFile: vi.fn(),
      getProjectNote: vi.fn(),
      operationStatus: vi.fn(),
      promoteSession: vi.fn(),
      retryReviewPending: vi.fn(),
      getProject: vi.fn(),
    },
  }
})

const session = {
  id: 's1',
  title: 'Chat',
  status: 'idle',
  provider: 'openai',
  model_id: 'gpt',
}

const deferred = <T>() => {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((yes, no) => {
    resolve = yes
    reject = no
  })
  return { promise, resolve, reject }
}

afterEach(cleanup)

describe('SessionChat', () => {
  beforeEach(() => {
    vi.mocked(api.listMessages).mockReset().mockResolvedValue([
      { sequence: 1, role: 'user', content: 'cached' },
    ])
    vi.mocked(api.currentRun).mockReset().mockResolvedValue(null)
    vi.mocked(api.sendMessage).mockReset()
    vi.mocked(api.workspaceTree).mockReset().mockResolvedValue({ entries: [] })
    vi.mocked(api.getProject).mockReset().mockResolvedValue({
      id: 'p1',
      name: 'Sleep Protocol',
      vault_id: 'v1',
      vault_name: 'HEALTH',
      note_count: 0,
    })
  })

  it('shows breadcrumbs instead of Back + title when project is known', async () => {
    const onclose = vi.fn()
    render(SessionChat, {
      props: {
        session: { ...session, title: 'Test 1' },
        projectId: 'p1',
        project: {
          id: 'p1',
          name: 'Sleep Protocol',
          vault_id: 'v1',
          vault_name: 'HEALTH',
          note_count: 0,
        },
        onclose,
        pollInterval: 60_000,
      },
    })
    expect(await screen.findByRole('navigation', { name: 'Breadcrumb' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Sleep Protocol' })).toBeInTheDocument()
    expect(screen.getByText('Test 1')).toHaveAttribute('aria-current', 'page')
    expect(screen.queryByRole('button', { name: 'Back' })).toBeNull()
    expect(screen.queryByRole('heading', { name: 'Test 1' })).toBeNull()

    await fireEvent.click(screen.getByRole('link', { name: 'Sleep Protocol' }))
    expect(onclose).toHaveBeenCalledTimes(1)
  })

  it('posts once with one stable request_key and retains draft after failure', async () => {
    const gate = deferred<null>()
    vi.mocked(api.sendMessage).mockReturnValue(gate.promise)
    render(SessionChat, {
      props: {
        session,
        projectId: 'p1',
        pollInterval: 60_000,
        uuid: () => 'stable-key',
      },
    })
    const composer = await screen.findByLabelText('Message')
    await fireEvent.input(composer, { target: { value: 'draft' } })
    const form = composer.closest('form')!
    const first = fireEvent.submit(form)
    const second = fireEvent.submit(form)
    await first
    await second
    expect(api.sendMessage).toHaveBeenCalledTimes(1)
    expect(api.sendMessage).toHaveBeenCalledWith(session.id, {
      content: 'draft',
      request_key: 'stable-key',
    })
    gate.reject(new Error('AI unavailable'))
    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('AI unavailable'))
    expect(composer).toHaveValue('draft')
    expect(screen.getByText('cached')).toBeInTheDocument()
  })

  it('retains cached history when a poll fails', async () => {
    const { component } = render(SessionChat, {
      props: { session, projectId: 'p1', pollInterval: 60_000 },
    })
    expect(await screen.findByText('cached')).toBeInTheDocument()
    vi.mocked(api.listMessages).mockRejectedValueOnce(new Error('network down'))
    await (component as unknown as { poll: () => Promise<void> }).poll()
    expect(screen.getByText('cached')).toBeInTheDocument()
    expect(screen.getByRole('alert')).toHaveTextContent('network down')
  })

  it('suppresses duplicate submit while a send is in flight', async () => {
    const gate = deferred<null>()
    vi.mocked(api.sendMessage).mockReturnValue(gate.promise)
    render(SessionChat, {
      props: { session, projectId: 'p1', pollInterval: 60_000, uuid: () => 'one-key' },
    })
    const composer = await screen.findByLabelText('Message')
    await fireEvent.input(composer, { target: { value: 'keep draft' } })
    const form = composer.closest('form')!
    const first = fireEvent.submit(form)
    await first
    await fireEvent.submit(form)
    expect(api.sendMessage).toHaveBeenCalledTimes(1)
    gate.reject(new Error('send failed'))
    await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent('send failed'))
    expect(composer).toHaveValue('keep draft')
  })

  it('clears draft after a successful send', async () => {
    vi.mocked(api.sendMessage).mockResolvedValue(null)
    render(SessionChat, {
      props: { session, projectId: 'p1', pollInterval: 60_000, uuid: () => 'ok-key' },
    })
    const composer = await screen.findByLabelText('Message')
    await fireEvent.input(composer, { target: { value: 'hello' } })
    await fireEvent.submit(composer.closest('form')!)
    await waitFor(() => expect(composer).toHaveValue(''))
  })

  it('Enter sends message; Shift+Enter does not submit', async () => {
    vi.mocked(api.sendMessage).mockResolvedValue(null)
    render(SessionChat, {
      props: { session, projectId: 'p1', pollInterval: 60_000, uuid: () => 'enter-key' },
    })
    const composer = await screen.findByLabelText('Message')
    await fireEvent.input(composer, { target: { value: 'Hello from Enter' } })

    await fireEvent.keyDown(composer, { key: 'Enter', shiftKey: true })
    expect(api.sendMessage).not.toHaveBeenCalled()

    await fireEvent.keyDown(composer, { key: 'Enter', shiftKey: false })
    await waitFor(() => {
      expect(api.sendMessage).toHaveBeenCalledWith(
        session.id,
        expect.objectContaining({ content: 'Hello from Enter', request_key: 'enter-key' }),
      )
    })
  })

  it('renders assistant as bare prose without Assistant label', async () => {
    vi.mocked(api.listMessages).mockResolvedValue([
      { sequence: 1, role: 'user', content: 'hello from me' },
      { sequence: 2, role: 'assistant', content: 'reply from agent' },
    ])
    render(SessionChat, {
      props: { session, projectId: 'p1', pollInterval: 60_000 },
    })
    expect(await screen.findByText('hello from me')).toBeInTheDocument()
    expect(screen.getByText('reply from agent')).toBeInTheDocument()
    expect(screen.queryByText('Assistant')).not.toBeInTheDocument()

    const userRow = screen.getByText('hello from me').closest('li')
    const assistantRow = screen.getByText('reply from agent').closest('li')
    expect(userRow).toHaveAttribute('data-role', 'user')
    expect(assistantRow).toHaveAttribute('data-role', 'assistant')
    expect(userRow?.className).toMatch(/message-row--user/)
    expect(assistantRow?.className).toMatch(/message-row--assistant/)
    expect(userRow?.querySelector('.message-bubble--user')).toBeTruthy()
    expect(assistantRow?.querySelector('.message-bubble')).toBeNull()
    expect(assistantRow?.querySelector('.message-prose')).toBeTruthy()
  })

  it('copy control is an icon under assistant prose and copies plain text', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, { clipboard: { writeText } })

    const created = new Date(2026, 4, 30, 22, 36, 0)
    vi.mocked(api.listMessages).mockResolvedValue([
      { sequence: 1, role: 'user', content: 'hello' },
      {
        sequence: 2,
        role: 'assistant',
        content: 'Hi — how can I help you today?',
        created_at: created.toISOString(),
      },
    ])
    render(SessionChat, {
      props: { session, projectId: 'p1', pollInterval: 60_000 },
    })
    expect(await screen.findByText('Hi — how can I help you today?')).toBeInTheDocument()

    const row = screen.getByText('Hi — how can I help you today?').closest('li')
    const footer = row?.querySelector('.message-assistant__footer')
    expect(footer).toBeTruthy()

    const copyBtn = screen.getByRole('button', { name: 'Copy response' })
    expect(copyBtn).toHaveClass('message-copy')
    expect(copyBtn.querySelector('svg')).toBeTruthy()
    expect(copyBtn).not.toHaveTextContent(/^Copy$/)

    const dateEl = screen.getByText('May 30')
    expect(dateEl.tagName).toBe('TIME')
    expect(dateEl).toHaveAttribute('data-tooltip', 'May 30, 2026 10:36 PM')
    expect(dateEl).toHaveAttribute('title', 'May 30, 2026 10:36 PM')
    expect(dateEl).toHaveAttribute('datetime', created.toISOString())

    // Order: copy icon first, then date
    const footerKids = Array.from(footer?.children ?? [])
    expect(footerKids[0]).toBe(copyBtn)
    expect(footerKids[1]).toBe(dateEl)

    // Footer sits after prose in DOM order
    const prose = row?.querySelector('.message-prose')
    expect(prose && footer && (prose.compareDocumentPosition(footer) & Node.DOCUMENT_POSITION_FOLLOWING)).toBeTruthy()

    await fireEvent.click(copyBtn)
    expect(writeText).toHaveBeenCalledWith('Hi — how can I help you today?')
    expect(await screen.findByRole('button', { name: 'Copied' })).toBeInTheDocument()
  })

  it('omits assistant date when created_at is missing', async () => {
    vi.mocked(api.listMessages).mockResolvedValue([
      { sequence: 1, role: 'assistant', content: 'no stamp' },
    ])
    render(SessionChat, {
      props: { session, projectId: 'p1', pollInterval: 60_000 },
    })
    expect(await screen.findByText('no stamp')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Copy response' })).toBeInTheDocument()
    expect(document.querySelector('.message-assistant__date')).toBeNull()
  })

  it('composer has no visible Message label text node soup', async () => {
    render(SessionChat, {
      props: { session, projectId: 'p1', pollInterval: 60_000 },
    })
    const composer = await screen.findByLabelText('Message')
    expect(composer).toBeInTheDocument()
    expect(composer.tagName).toBe('TEXTAREA')
    expect(screen.queryByText('Message', { selector: 'span.font-medium' })).toBeNull()
    expect(composer.closest('form')?.className).toMatch(/session-composer/)
  })

  it('composer shows Reply… placeholder and Claude chat card chrome', async () => {
    render(SessionChat, {
      props: { session, projectId: 'p1', pollInterval: 60_000 },
    })
    const composer = await screen.findByLabelText('Message')
    expect(composer).toHaveAttribute('placeholder', 'Reply…')
    const form = composer.closest('form')
    expect(form?.className).toMatch(/session-composer/)
    expect(form?.querySelector('.session-composer__card')).toBeTruthy()
    expect(form?.querySelector('.session-composer__model')).toBeTruthy()
    expect(form?.querySelector('.session-composer__model')?.textContent).toMatch(
      /openai:gpt/,
    )
    const send = form?.querySelector('button[type="submit"]')
    expect(send).toBeTruthy()
    expect(send?.className).toMatch(/session-composer__send|btn--primary/)
  })

  const memStorage = () => {
    const m = new Map<string, string>()
    return {
      get length() {
        return m.size
      },
      clear: () => m.clear(),
      getItem: (k: string) => m.get(k) ?? null,
      setItem: (k: string, v: string) => {
        m.set(k, String(v))
      },
      removeItem: (k: string) => {
        m.delete(k)
      },
      key: () => null,
    } satisfies Storage
  }

  it('toggles files bar and persists pref', async () => {
    const mem = memStorage()
    const wsSession = {
      ...session,
      tool_grants: { workspace_files: true as const },
    }
    render(SessionChat, {
      props: {
        session: wsSession,
        projectId: 'p1',
        pollInterval: 60_000,
        storage: mem,
      },
    })
    const toggle = await screen.findByRole('button', { name: /show files/i })
    expect(toggle).toHaveAttribute('aria-pressed', 'false')
    await fireEvent.click(toggle)
    expect(mem.getItem('pa.session.filesBarOpen')).toBe('1')
    expect(screen.getByRole('button', { name: /hide files/i })).toHaveAttribute(
      'aria-pressed',
      'true',
    )
  })

  describe('narrow files drawer', () => {
    const wsSession = {
      ...session,
      tool_grants: { workspace_files: true as const },
    }

    type MediaListener = (event: MediaQueryListEvent) => void

    function mockMatchMedia(matches: boolean) {
      const listeners = new Set<MediaListener>()
      const mql = {
        matches,
        media: '(max-width: 1023px)',
        onchange: null as ((this: MediaQueryList, ev: MediaQueryListEvent) => void) | null,
        addEventListener: (_type: string, listener: EventListenerOrEventListenerObject) => {
          listeners.add(listener as MediaListener)
        },
        removeEventListener: (_type: string, listener: EventListenerOrEventListenerObject) => {
          listeners.delete(listener as MediaListener)
        },
        addListener: (listener: MediaListener) => {
          listeners.add(listener)
        },
        removeListener: (listener: MediaListener) => {
          listeners.delete(listener)
        },
        dispatchEvent: () => true,
      }
      const spy = vi.spyOn(window, 'matchMedia').mockImplementation((query: string) => {
        if (query.includes('1023')) return mql as unknown as MediaQueryList
        return {
          matches: false,
          media: query,
          onchange: null,
          addEventListener: () => {},
          removeEventListener: () => {},
          addListener: () => {},
          removeListener: () => {},
          dispatchEvent: () => true,
        } as unknown as MediaQueryList
      })
      return {
        mql,
        spy,
        setMatches(next: boolean) {
          mql.matches = next
          const event = { matches: next, media: mql.media } as MediaQueryListEvent
          for (const listener of listeners) listener(event)
          mql.onchange?.call(mql as unknown as MediaQueryList, event)
        },
      }
    }

    it('uses session-files-drawer + backdrop when narrow and files open', async () => {
      const media = mockMatchMedia(true)
      const mem = memStorage()
      mem.setItem('pa.session.filesBarOpen', '1')
      try {
        const { container } = render(SessionChat, {
          props: {
            session: wsSession,
            projectId: 'p1',
            pollInterval: 60_000,
            storage: mem,
          },
        })
        await screen.findByLabelText('Session files')
        const root = container.querySelector('.session-focus')
        expect(root).toHaveAttribute('data-files-open', '1')
        const files = container.querySelector('.session-split__files')
        expect(files?.classList.contains('session-files-drawer')).toBe(true)
        expect(screen.getByRole('button', { name: /close files/i })).toBeInTheDocument()
      } finally {
        media.spy.mockRestore()
      }
    })

    it('Escape closes drawer and writes open pref false', async () => {
      const media = mockMatchMedia(true)
      const mem = memStorage()
      mem.setItem('pa.session.filesBarOpen', '1')
      try {
        const { container } = render(SessionChat, {
          props: {
            session: wsSession,
            projectId: 'p1',
            pollInterval: 60_000,
            storage: mem,
          },
        })
        await screen.findByLabelText('Session files')
        await fireEvent.keyDown(window, { key: 'Escape' })
        await waitFor(() => {
          expect(container.querySelector('.session-split__files')).toBeNull()
        })
        expect(mem.getItem('pa.session.filesBarOpen')).toBe('0')
        expect(container.querySelector('.session-focus')).toHaveAttribute('data-files-open', '0')
      } finally {
        media.spy.mockRestore()
      }
    })

    it('backdrop click closes drawer and writes open pref false', async () => {
      const media = mockMatchMedia(true)
      const mem = memStorage()
      mem.setItem('pa.session.filesBarOpen', '1')
      try {
        const { container } = render(SessionChat, {
          props: {
            session: wsSession,
            projectId: 'p1',
            pollInterval: 60_000,
            storage: mem,
          },
        })
        await screen.findByLabelText('Session files')
        await fireEvent.click(screen.getByRole('button', { name: /close files/i }))
        await waitFor(() => {
          expect(container.querySelector('.session-split__files')).toBeNull()
        })
        expect(mem.getItem('pa.session.filesBarOpen')).toBe('0')
      } finally {
        media.spy.mockRestore()
      }
    })

    it('does not apply drawer class on wide viewport', async () => {
      const media = mockMatchMedia(false)
      const mem = memStorage()
      mem.setItem('pa.session.filesBarOpen', '1')
      try {
        const { container } = render(SessionChat, {
          props: {
            session: wsSession,
            projectId: 'p1',
            pollInterval: 60_000,
            storage: mem,
          },
        })
        await screen.findByLabelText('Session files')
        const files = container.querySelector('.session-split__files')
        expect(files?.classList.contains('session-files-drawer')).toBe(false)
        expect(screen.queryByRole('button', { name: /close files/i })).not.toBeInTheDocument()
      } finally {
        media.spy.mockRestore()
      }
    })
  })

  it('ignores stale poll results after session switch', async () => {
    const slow = deferred<Array<{ sequence: number; role: string; content: string }>>()
    let calls = 0
    vi.mocked(api.listMessages).mockImplementation(async (id: string) => {
      calls += 1
      if (id === 'old') return slow.promise as never
      return [{ sequence: 1, role: 'user', content: 'new history' }]
    })
    const first = render(SessionChat, {
      props: {
        session: { ...session, id: 'old', title: 'Old' },
        projectId: 'p1',
        pollInterval: 60_000,
      },
    })
    await waitFor(() => expect(calls).toBeGreaterThanOrEqual(1))
    first.unmount()
    render(SessionChat, {
      props: {
        session: { ...session, id: 'new', title: 'New' },
        projectId: 'p1',
        pollInterval: 60_000,
      },
    })
    expect(await screen.findByText('new history')).toBeInTheDocument()
    slow.resolve([{ sequence: 2, role: 'assistant', content: 'stale old' }])
    await waitFor(() => {
      expect(screen.queryByText('stale old')).not.toBeInTheDocument()
    })
    expect(screen.getByText('New')).toBeInTheDocument()
  })

  describe('file tabs', () => {
    const wsSession = {
      ...session,
      tool_grants: { workspace_files: true as const },
    }

    async function openFilesBarAndTree(paths: string[]) {
      const mem = memStorage()
      mem.setItem('pa.session.filesBarOpen', '1')
      vi.mocked(api.workspaceTree).mockResolvedValue({
        entries: paths.map((path) => ({ path, kind: 'file' as const })),
      })
      vi.mocked(api.workspaceFile).mockImplementation(async (_sid, path) => ({
        path,
        kind: 'file',
        content: path.endsWith('.md') ? `# ${path}` : `content:${path}`,
      }))
      render(SessionChat, {
        props: {
          session: wsSession,
          projectId: 'p1',
          pollInterval: 60_000,
          storage: mem,
        },
      })
      await screen.findByLabelText('Session files')
      for (const path of paths) {
        await screen.findByRole('button', { name: path })
      }
      return mem
    }

    it('opens file from bar and focuses that tab', async () => {
      await openFilesBarAndTree(['draft.md'])
      await fireEvent.click(screen.getByRole('button', { name: 'draft.md' }))
      const fileTab = await screen.findByRole('tab', { name: /draft\.md/i })
      expect(fileTab).toHaveAttribute('aria-selected', 'true')
      expect(fileTab.className).toMatch(/session-tab--active/)
      expect(screen.getByRole('tab', { name: /^Agent$/i })).toHaveAttribute('aria-selected', 'false')
      await waitFor(() => expect(api.workspaceFile).toHaveBeenCalledWith('s1', 'draft.md'))
      expect(await screen.findByRole('heading', { level: 1, name: 'draft.md' })).toBeInTheDocument()
      // Composer stays mounted while file tab is active
      expect(screen.getByLabelText('Message').closest('form')).toBeTruthy()
    })

    it('reuses the same path instead of duplicating tabs', async () => {
      await openFilesBarAndTree(['a.md', 'b.md'])
      await fireEvent.click(screen.getByRole('button', { name: 'a.md' }))
      await screen.findByRole('tab', { name: /a\.md/i })
      await fireEvent.click(screen.getByRole('button', { name: 'b.md' }))
      await screen.findByRole('tab', { name: /b\.md/i })
      await fireEvent.click(screen.getByRole('button', { name: 'a.md' }))
      const tabs = screen.getAllByRole('tab')
      // Agent + a + b only
      expect(tabs).toHaveLength(3)
      expect(tabs.filter((t) => (t.textContent ?? '').includes('a.md'))).toHaveLength(1)
      expect(screen.getByRole('tab', { name: /a\.md/i })).toHaveAttribute('aria-selected', 'true')
      expect(screen.getByRole('tab', { name: /b\.md/i })).toBeInTheDocument()
    })

    it('closes least-recently-activated file tab when opening a 9th path', async () => {
      const paths = Array.from({ length: 9 }, (_, i) => `f${i}.md`)
      await openFilesBarAndTree(paths)
      // Open f0..f7 first (8 tabs), activating in order so f0 is LRU
      for (let i = 0; i < 8; i++) {
        await fireEvent.click(screen.getByRole('button', { name: paths[i]! }))
        await screen.findByRole('tab', { name: new RegExp(paths[i]!.replace('.', '\\.')) })
      }
      // Open 9th → should close f0
      await fireEvent.click(screen.getByRole('button', { name: 'f8.md' }))
      await screen.findByRole('tab', { name: /f8\.md/i })
      expect(screen.queryByRole('tab', { name: /f0\.md/i })).not.toBeInTheDocument()
      // Still 8 file tabs + Agent
      const tabs = screen.getAllByRole('tab')
      expect(tabs).toHaveLength(9) // Agent + 8 files
      expect(screen.getByRole('tab', { name: /^Agent$/i })).toBeInTheDocument()
    })

    it('close button removes file tab and returns to Agent when it was active', async () => {
      await openFilesBarAndTree(['draft.md'])
      await fireEvent.click(screen.getByRole('button', { name: 'draft.md' }))
      const fileTab = await screen.findByRole('tab', { name: /draft\.md/i })
      const close = fileTab.querySelector('.session-tab__close') as HTMLButtonElement
      expect(close).toBeTruthy()
      await fireEvent.click(close)
      expect(screen.queryByRole('tab', { name: /draft\.md/i })).not.toBeInTheDocument()
      expect(screen.getByRole('tab', { name: /^Agent$/i })).toHaveAttribute('aria-selected', 'true')
      expect(screen.getByLabelText('Message')).toBeVisible()
    })

    it('keeps Agent draft when switching to file tab and back', async () => {
      await openFilesBarAndTree(['draft.md'])
      const composer = await screen.findByLabelText('Message')
      await fireEvent.input(composer, { target: { value: 'keep me' } })
      await fireEvent.click(screen.getByRole('button', { name: 'draft.md' }))
      await screen.findByRole('tab', { name: /draft\.md/i })
      await fireEvent.click(screen.getByRole('tab', { name: /^Agent$/i }))
      expect(screen.getByLabelText('Message')).toHaveValue('keep me')
    })

    it('shows Save to source from file tab for .md and opens PromoteDialog', async () => {
      // jsdom dialog polyfill (same as PromoteDialog.test.ts)
      if (!HTMLDialogElement.prototype.showModal) {
        HTMLDialogElement.prototype.showModal = function showModal() {
          this.setAttribute('open', '')
        }
        HTMLDialogElement.prototype.close = function close() {
          this.removeAttribute('open')
        }
      }
      await openFilesBarAndTree(['draft.md', 'raw.txt'])
      await fireEvent.click(screen.getByRole('button', { name: 'draft.md' }))
      const promote = await screen.findByRole('button', { name: 'Save to source' })
      await fireEvent.click(promote)
      expect(await screen.findByRole('heading', { name: 'Save to source' })).toBeInTheDocument()
    })

    it('opens file tab when openPath prop is set', async () => {
      vi.mocked(api.workspaceFile).mockImplementation(async (_sid, path) => ({
        path,
        kind: 'file',
        content: path.endsWith('.md') ? `# ${path}` : `content:${path}`,
      }))
      render(SessionChat, {
        props: {
          session: wsSession,
          projectId: 'p1',
          pollInterval: 60_000,
          openPath: 'notes/a.md',
        },
      })
      const fileTab = await screen.findByRole('tab', { name: /a\.md/i })
      expect(fileTab).toHaveAttribute('aria-selected', 'true')
      expect(fileTab).toHaveAttribute('title', 'notes/a.md')
      expect(screen.getByRole('tab', { name: /^Agent$/i })).toHaveAttribute('aria-selected', 'false')
      // Composer form stays mounted while file tab is active
      expect(screen.getByLabelText('Message').closest('form')).toBeTruthy()
      await waitFor(() => expect(api.workspaceFile).toHaveBeenCalledWith('s1', 'notes/a.md'))
    })

    it('openFileRequest with project-note loads via getProjectNote', async () => {
      vi.mocked(api.workspaceFile).mockClear()
      vi.mocked(api.getProjectNote).mockReset().mockResolvedValue({
        note_id: 'n-1',
        relative_path: 'notes/a.md',
        body: '# From note',
      })
      render(SessionChat, {
        props: {
          session: wsSession,
          projectId: 'p1',
          pollInterval: 60_000,
          openFileRequest: {
            path: 'notes/a.md',
            source: 'project-note',
            noteId: 'n-1',
          },
        },
      })
      const fileTab = await screen.findByRole('tab', { name: /a\.md/i })
      expect(fileTab).toHaveAttribute('aria-selected', 'true')
      await waitFor(() => expect(api.getProjectNote).toHaveBeenCalledWith('p1', 'n-1'))
      expect(api.workspaceFile).not.toHaveBeenCalled()
      expect(await screen.findByRole('heading', { level: 1, name: 'From note' })).toBeInTheDocument()
    })

    it('when embeddedInHub, does not show Show files toggle', async () => {
      render(SessionChat, {
        props: {
          session: wsSession,
          projectId: 'p1',
          pollInterval: 60_000,
          embeddedInHub: true,
        },
      })
      await screen.findByLabelText('Message')
      expect(screen.queryByRole('button', { name: /show files/i })).toBeNull()
      expect(screen.queryByRole('button', { name: /hide files/i })).toBeNull()
      expect(screen.queryByLabelText('Session files')).toBeNull()
    })
  })
})
