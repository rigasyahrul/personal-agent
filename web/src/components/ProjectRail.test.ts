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
      getProjectMemoryLessons: vi.fn(),
    },
  }
})

afterEach(cleanup)

describe('ProjectRail', () => {
  beforeEach(() => {
    vi.mocked(api.listProjectNotes).mockReset().mockResolvedValue([])
    vi.mocked(api.workspaceTree).mockReset().mockResolvedValue({ entries: [] })
    vi.mocked(api.getProjectMemoryLessons).mockReset().mockResolvedValue({ content: '' })
  })

  it('shows Memory and Files tabs without a Memory textarea', async () => {
    render(ProjectRail, { props: { projectId: 'p1' } })
    expect(screen.getByRole('tab', { name: 'Memory' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Files' })).toBeInTheDocument()
    expect(screen.queryByRole('textbox', { name: 'Memory' })).toBeNull()
    expect(screen.queryByRole('button', { name: /save/i })).toBeNull()
    expect(screen.queryByRole('button', { name: /compound/i })).toBeNull()
    expect(screen.getByRole('textbox', { name: /instructions/i })).toBeInTheDocument()
  })

  it('shows empty memory copy when lessons content is blank', async () => {
    vi.mocked(api.getProjectMemoryLessons).mockResolvedValue({ content: '  \n\n' })
    render(ProjectRail, { props: { projectId: 'p1' } })
    expect(await screen.findByText('No lessons yet.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Open memory' })).toBeNull()
  })

  it('previews the first non-empty lessons lines', async () => {
    const lines = Array.from({ length: 16 }, (_, i) => `- lesson ${i + 1}`)
    vi.mocked(api.getProjectMemoryLessons).mockResolvedValue({
      content: `\n# Lessons\n\n${lines.join('\n')}\n`,
    })
    render(ProjectRail, { props: { projectId: 'p1' } })
    const preview = await screen.findByTestId('memory-lessons-preview')
    expect(preview.textContent).toContain('# Lessons')
    expect(preview.textContent).toContain('- lesson 11')
    expect(preview.textContent).not.toContain('- lesson 12')
    expect(preview.textContent).not.toContain('- lesson 16')
  })

  it('opens memory/lessons.md as a project-note when a note_id exists', async () => {
    const onOpenFile = vi.fn()
    vi.mocked(api.getProjectMemoryLessons).mockResolvedValue({ content: '# Lessons\n' })
    vi.mocked(api.listProjectNotes).mockResolvedValue([
      { path: 'memory/lessons.md', kind: 'file', note_id: 'note-lessons' },
    ])
    render(ProjectRail, { props: { projectId: 'p1', sessionId: 's1', onOpenFile } })
    await fireEvent.click(await screen.findByRole('button', { name: 'Open memory' }))
    expect(onOpenFile).toHaveBeenCalledWith('memory/lessons.md', {
      source: 'project-note',
      noteId: 'note-lessons',
    })
  })

  it('opens memory/lessons.md without note_id when none exists', async () => {
    const onOpenFile = vi.fn()
    vi.mocked(api.getProjectMemoryLessons).mockResolvedValue({ content: '# Lessons\n' })
    render(ProjectRail, { props: { projectId: 'p1', sessionId: 's1', onOpenFile } })
    await fireEvent.click(await screen.findByRole('button', { name: 'Open memory' }))
    expect(onOpenFile).toHaveBeenCalledWith('memory/lessons.md', { source: 'project-note' })
  })

  it('shows memory load error without remounting the rail tabs', async () => {
    vi.mocked(api.getProjectMemoryLessons).mockRejectedValue(new Error('lessons down'))
    render(ProjectRail, { props: { projectId: 'p1' } })
    expect(await screen.findByRole('alert')).toHaveTextContent('lessons down')
    expect(screen.getByRole('tab', { name: 'Memory' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Files' })).toBeInTheDocument()
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
      { path: 'notes/a.md', kind: 'file', note_id: 'n-a' },
      { path: 'notes', kind: 'folder' },
    ])
    render(ProjectRail, { props: { projectId: 'p1', sessionId: 's1', onOpenFile } })
    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    const file = await screen.findByRole('button', { name: 'notes/a.md' })
    await fireEvent.click(file)
    expect(onOpenFile).toHaveBeenCalledWith('notes/a.md', {
      source: 'project-note',
      noteId: 'n-a',
    })
  })

  it('opens workspace rows with workspace source', async () => {
    const onOpenFile = vi.fn()
    vi.mocked(api.listProjectNotes).mockResolvedValue([])
    vi.mocked(api.workspaceTree).mockResolvedValue({
      entries: [{ path: 'scratch.txt', kind: 'file' }],
    })
    render(ProjectRail, {
      props: {
        projectId: 'p1',
        sessionId: 's1',
        workspaceFilesEnabled: true,
        onOpenFile,
      },
    })
    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    await fireEvent.click(await screen.findByRole('button', { name: 'scratch.txt' }))
    expect(onOpenFile).toHaveBeenCalledWith('scratch.txt', { source: 'workspace' })
  })

  it('does not call onOpenFile when there is no sessionId', async () => {
    const onOpenFile = vi.fn()
    vi.mocked(api.listProjectNotes).mockResolvedValue([
      { path: 'notes/a.md', kind: 'file', note_id: 'n-a' },
    ])
    render(ProjectRail, { props: { projectId: 'p1', onOpenFile } })
    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    await fireEvent.click(await screen.findByRole('button', { name: 'notes/a.md' }))
    expect(onOpenFile).not.toHaveBeenCalled()
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
