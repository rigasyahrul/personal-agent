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

  it('defaults to Config with instructions and no Memory textarea', async () => {
    render(ProjectRail, { props: { projectId: 'p1' } })

    expect(screen.getByRole('tab', { name: 'Config' })).toHaveAttribute('aria-selected', 'true')
    expect(screen.getByRole('tab', { name: 'Files' })).toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Memory' })).not.toBeInTheDocument()
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
    expect(screen.getByRole('tab', { name: 'Config' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Files' })).toBeInTheDocument()
  })

  it('orders panel tabs before workspace controls in the icon bar', () => {
    render(ProjectRail, { props: { projectId: 'p1' } })

    const iconbar = screen.getByRole('toolbar', { name: 'Project rail' })
    const controls = Array.from(iconbar.querySelectorAll('button'))
    expect(controls.map((control) => control.getAttribute('aria-label'))).toEqual([
      'Config',
      'Files',
      'Expand workspace',
      'Collapse canvas',
    ])
  })

  it('uses a controlled tab and reports tab changes', async () => {
    const onTabChange = vi.fn()
    render(ProjectRail, { props: { projectId: 'p1', tab: 'config', onTabChange } })

    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    expect(onTabChange).toHaveBeenCalledWith('files')
    expect(screen.getByRole('tab', { name: 'Config' })).toHaveAttribute('aria-selected', 'true')
  })

  it('switches an uncontrolled rail to Files and shows empty copy', async () => {
    render(ProjectRail, { props: { projectId: 'p1' } })
    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    expect(await screen.findByText('No project files available.')).toBeInTheDocument()
  })

  it('lists project notes as files and opens one with project-note metadata', async () => {
    const onOpenFile = vi.fn()
    vi.mocked(api.listProjectNotes).mockResolvedValue([
      { path: 'notes/a.md', kind: 'file', note_id: 'n-a' },
      { path: 'notes', kind: 'folder' },
    ])
    render(ProjectRail, { props: { projectId: 'p1', sessionId: 's1', onOpenFile } })

    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    await fireEvent.click(await screen.findByRole('button', { name: 'notes/a.md' }))
    expect(onOpenFile).toHaveBeenCalledWith('notes/a.md', {
      source: 'project-note',
      noteId: 'n-a',
    })
  })

  it('opens workspace rows with workspace metadata', async () => {
    const onOpenFile = vi.fn()
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

  it('does not open a project note without a session', async () => {
    const onOpenFile = vi.fn()
    vi.mocked(api.listProjectNotes).mockResolvedValue([
      { path: 'notes/a.md', kind: 'file', note_id: 'n-a' },
    ])
    render(ProjectRail, { props: { projectId: 'p1', onOpenFile } })

    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    await fireEvent.click(await screen.findByRole('button', { name: 'notes/a.md' }))
    expect(onOpenFile).not.toHaveBeenCalled()
  })

  it('merges workspace files under Workspace when the session grant is enabled', async () => {
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
    await waitFor(() => expect(api.workspaceTree).toHaveBeenCalledWith('s1'))
    expect(await screen.findByText('Workspace')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'readme.md' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'scratch.txt' })).toBeInTheDocument()
  })

  it('does not load workspace files when the grant is disabled', async () => {
    render(ProjectRail, { props: { projectId: 'p1', sessionId: 's1' } })

    await fireEvent.click(screen.getByRole('tab', { name: 'Files' }))
    await screen.findByText('No project files available.')
    expect(api.workspaceTree).not.toHaveBeenCalled()
  })

  it('renders only Show canvas chrome when collapsed', () => {
    render(ProjectRail, { props: { projectId: 'p1', mode: 'collapsed' } })

    expect(screen.getAllByRole('button')).toHaveLength(1)
    expect(screen.getByRole('button', { name: 'Show canvas' })).toBeInTheDocument()
    expect(document.querySelector('.project-rail--collapsed')).toBeTruthy()
    expect(document.querySelector('.rail-collapsed-chrome')).toBeTruthy()
    expect(screen.queryByRole('tab', { name: 'Config' })).not.toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: 'Files' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Expand workspace' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Collapse canvas' })).not.toBeInTheDocument()
    expect(screen.queryByRole('textbox', { name: 'Instructions (system)' })).not.toBeInTheDocument()
    expect(screen.queryByRole('tabpanel')).not.toBeInTheDocument()
  })

  it('requests expanded mode from the open rail', async () => {
    const onModeChange = vi.fn()
    render(ProjectRail, { props: { projectId: 'p1', mode: 'open', onModeChange } })

    await fireEvent.click(screen.getByRole('button', { name: 'Expand workspace' }))
    expect(onModeChange).toHaveBeenCalledWith('expanded')
  })

  it('requests open mode from the expanded rail', async () => {
    const onModeChange = vi.fn()
    render(ProjectRail, { props: { projectId: 'p1', mode: 'expanded', onModeChange } })

    const exit = screen.getByRole('button', { name: 'Exit expanded' })
    expect(exit).toHaveAttribute('aria-pressed', 'true')
    await fireEvent.click(exit)
    expect(onModeChange).toHaveBeenCalledWith('open')
  })

  it('requests collapsed mode from Collapse canvas', async () => {
    const onModeChange = vi.fn()
    render(ProjectRail, { props: { projectId: 'p1', mode: 'open', onModeChange } })

    await fireEvent.click(screen.getByRole('button', { name: 'Collapse canvas' }))
    expect(onModeChange).toHaveBeenCalledWith('collapsed')
  })

  it('requests open mode from Show canvas', async () => {
    const onModeChange = vi.fn()
    render(ProjectRail, { props: { projectId: 'p1', mode: 'collapsed', onModeChange } })

    await fireEvent.click(screen.getByRole('button', { name: 'Show canvas' }))
    expect(onModeChange).toHaveBeenCalledWith('open')
  })
})
