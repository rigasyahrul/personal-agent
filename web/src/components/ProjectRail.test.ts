// web/src/components/ProjectRail.test.ts
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import ProjectRail from './ProjectRail.svelte'
import { api } from '../lib/api'

vi.mock('../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      listProjectNotes: vi.fn(),
      workspaceTree: vi.fn(),
    },
  }
})

afterEach(cleanup)

describe('ProjectRail', () => {
  beforeEach(() => {
    vi.mocked(api.listProjectNotes).mockReset().mockResolvedValue([])
    vi.mocked(api.workspaceTree).mockReset().mockResolvedValue({ entries: [] })
  })

  it('shows Memory and Files tabs; Memory has non-persistent helper', async () => {
    vi.mocked(api.listProjectNotes).mockResolvedValue([])
    render(ProjectRail, { props: { projectId: 'p1' } })
    expect(screen.getByRole('tab', { name: 'Memory' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Files' })).toBeInTheDocument()
    // Prefer textbox role so the Memory tab name cannot collide with the field label.
    expect(screen.getByRole('textbox', { name: 'Memory' })).toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: /instructions/i })).toBeInTheDocument()
    expect(screen.getByText(/not saved yet/i)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /save/i })).toBeNull()
  })

  it('switches to Files and shows empty copy', async () => {
    vi.mocked(api.listProjectNotes).mockResolvedValue([])
    render(ProjectRail, { props: { projectId: 'p1' } })
    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    expect(await screen.findByText(/no project files available/i)).toBeInTheDocument()
  })

  it('lists project notes as files and opens on click', async () => {
    const onOpenFile = vi.fn()
    vi.mocked(api.listProjectNotes).mockResolvedValue([
      { path: 'notes/a.md', kind: 'file' },
      { path: 'notes', kind: 'folder' },
    ])
    render(ProjectRail, { props: { projectId: 'p1', onOpenFile } })
    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    const file = await screen.findByRole('button', { name: 'notes/a.md' })
    await fireEvent.click(file)
    expect(onOpenFile).toHaveBeenCalledWith('notes/a.md')
  })

  it('merges workspace tree under Workspace when session + grant', async () => {
    vi.mocked(api.listProjectNotes).mockResolvedValue([{ path: 'readme.md', kind: 'file' }])
    vi.mocked(api.workspaceTree).mockResolvedValue({
      entries: [{ path: 'scratch.txt', kind: 'file' }],
    })
    render(ProjectRail, {
      props: {
        projectId: 'p1',
        sessionId: 's1',
        workspaceFilesEnabled: true,
      },
    })
    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    await waitFor(() => {
      expect(api.workspaceTree).toHaveBeenCalledWith('s1')
    })
    expect(await screen.findByText('Workspace')).toBeInTheDocument()
    expect(await screen.findByRole('button', { name: 'scratch.txt' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'readme.md' })).toBeInTheDocument()
  })
})
