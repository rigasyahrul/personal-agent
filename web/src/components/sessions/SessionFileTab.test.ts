// web/src/components/sessions/SessionFileTab.test.ts
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SessionFileTab from './SessionFileTab.svelte'
import { api } from '../../lib/api'

vi.mock('../../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      workspaceFile: vi.fn(),
    },
  }
})

afterEach(cleanup)

describe('SessionFileTab', () => {
  beforeEach(() => {
    vi.mocked(api.workspaceFile).mockReset()
  })

  it('loads file and shows Preview markdown for .md by default', async () => {
    vi.mocked(api.workspaceFile).mockResolvedValue({
      path: 'notes/draft.md',
      kind: 'file',
      content: '# Hello Tab\n\nBody text.',
    })
    render(SessionFileTab, {
      props: { sessionId: 's1', path: 'notes/draft.md', projectId: 'p1' },
    })
    await waitFor(() => expect(api.workspaceFile).toHaveBeenCalledWith('s1', 'notes/draft.md'))
    expect(await screen.findByRole('heading', { level: 1, name: 'Hello Tab' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Preview' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Source' })).toHaveAttribute('aria-pressed', 'false')
  })

  it('switches to Source monospace and back to Preview', async () => {
    vi.mocked(api.workspaceFile).mockResolvedValue({
      path: 'draft.md',
      kind: 'file',
      content: '# Title\n\npara',
    })
    render(SessionFileTab, {
      props: { sessionId: 's1', path: 'draft.md', projectId: 'p1' },
    })
    await screen.findByRole('heading', { level: 1, name: 'Title' })
    await fireEvent.click(screen.getByRole('button', { name: 'Source' }))
    expect(screen.getByRole('button', { name: 'Source' })).toHaveAttribute('aria-pressed', 'true')
    const pre = document.querySelector('pre.workspace-preview')
    expect(pre).toBeTruthy()
    expect(pre).toHaveTextContent('# Title')
    expect(screen.queryByRole('heading', { level: 1, name: 'Title' })).not.toBeInTheDocument()
    await fireEvent.click(screen.getByRole('button', { name: 'Preview' }))
    expect(await screen.findByRole('heading', { level: 1, name: 'Title' })).toBeInTheDocument()
  })

  it('shows Save to source for promotable .md and calls onpromote', async () => {
    const file = { path: 'draft.md', kind: 'file', content: '# hi' }
    vi.mocked(api.workspaceFile).mockResolvedValue(file)
    const onpromote = vi.fn()
    render(SessionFileTab, {
      props: { sessionId: 's1', path: 'draft.md', projectId: 'p1', onpromote },
    })
    const promote = await screen.findByRole('button', { name: 'Save to source' })
    await fireEvent.click(promote)
    expect(onpromote).toHaveBeenCalledWith(expect.objectContaining({ path: 'draft.md', kind: 'file' }))
  })

  it('shows Save to source when API omits kind on .md files', async () => {
    // Matches real GET /workspace/file payload shape (path + content only).
    vi.mocked(api.workspaceFile).mockResolvedValue({
      path: 'notes/outline.md',
      content: '# Outline',
    } as Awaited<ReturnType<typeof api.workspaceFile>>)
    render(SessionFileTab, {
      props: { sessionId: 's1', path: 'notes/outline.md', projectId: 'p1' },
    })
    expect(await screen.findByRole('button', { name: 'Save to source' })).toBeInTheDocument()
  })

  it('hides Save to source for non-markdown files', async () => {
    vi.mocked(api.workspaceFile).mockResolvedValue({
      path: 'raw.txt',
      kind: 'file',
      content: 'plain text',
    })
    render(SessionFileTab, {
      props: { sessionId: 's1', path: 'raw.txt', projectId: 'p1' },
    })
    await screen.findByText('plain text')
    expect(screen.queryByRole('button', { name: 'Save to source' })).not.toBeInTheDocument()
    // non-md preview is monospace pre
    expect(document.querySelector('pre.workspace-preview')).toHaveTextContent('plain text')
  })

  it('shows loading then inline error on failure', async () => {
    vi.mocked(api.workspaceFile).mockRejectedValue(new Error('file boom'))
    render(SessionFileTab, {
      props: { sessionId: 's1', path: 'missing.md', projectId: 'p1' },
    })
    expect(await screen.findByRole('alert')).toHaveTextContent('file boom')
  })

  it('respects controlled mode prop and notifies onmode', async () => {
    vi.mocked(api.workspaceFile).mockResolvedValue({
      path: 'a.md',
      kind: 'file',
      content: '# A',
    })
    const onmode = vi.fn()
    render(SessionFileTab, {
      props: {
        sessionId: 's1',
        path: 'a.md',
        projectId: 'p1',
        mode: 'source',
        onmode,
      },
    })
    await waitFor(() => expect(document.querySelector('pre.workspace-preview')).toBeTruthy())
    expect(screen.getByRole('button', { name: 'Source' })).toHaveAttribute('aria-pressed', 'true')
    await fireEvent.click(screen.getByRole('button', { name: 'Preview' }))
    expect(onmode).toHaveBeenCalledWith('preview')
  })
})
