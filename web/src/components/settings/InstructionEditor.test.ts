// web/src/components/settings/InstructionEditor.test.ts
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import InstructionEditor from './InstructionEditor.svelte'
import { api } from '../../lib/api'

vi.mock('../../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../lib/api')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      getGlobalInstruction: vi.fn(),
      putGlobalInstruction: vi.fn(),
      getProjectInstruction: vi.fn(),
      putProjectInstruction: vi.fn(),
    },
  }
})

afterEach(cleanup)

describe('InstructionEditor', () => {
  beforeEach(() => {
    vi.mocked(api.getGlobalInstruction).mockReset().mockResolvedValue({ content: 'Be kind.\n' })
    vi.mocked(api.putGlobalInstruction).mockReset().mockResolvedValue({ content: 'Be kinder.\n' })
    vi.mocked(api.getProjectInstruction).mockReset().mockResolvedValue({ content: 'Ship it.\n' })
    vi.mocked(api.putProjectInstruction).mockReset().mockResolvedValue({ content: 'Ship sooner.\n' })
  })

  it('saves the current global file with PUT', async () => {
    render(InstructionEditor, { props: { scope: 'global' } })

    const textarea = await screen.findByRole('textbox', { name: 'SOUL' })
    expect(textarea).toHaveValue('Be kind.\n')
    expect(document.querySelector('.field-textarea')).toBeTruthy()
    expect(document.querySelector('.btn.btn--primary')).toBeTruthy()
    expect(document.querySelector('.bg-indigo-600')).toBeNull()

    await fireEvent.input(textarea, { target: { value: 'Be kinder.\n' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => {
      expect(api.putGlobalInstruction).toHaveBeenCalledWith('soul', 'Be kinder.\n')
    })
    expect(api.putProjectInstruction).not.toHaveBeenCalled()
  })

  it('saves the current project file with PUT', async () => {
    render(InstructionEditor, { props: { scope: 'project', projectId: 'p1' } })

    const textarea = await screen.findByRole('textbox', { name: 'SOUL' })
    expect(textarea).toHaveValue('Ship it.\n')

    await fireEvent.input(textarea, { target: { value: 'Ship sooner.\n' } })
    await fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => {
      expect(api.putProjectInstruction).toHaveBeenCalledWith('p1', 'soul', 'Ship sooner.\n')
    })
    expect(api.putGlobalInstruction).not.toHaveBeenCalled()
  })

  it('shows a distinct placeholder for each instruction file', async () => {
    vi.mocked(api.getProjectInstruction).mockResolvedValue({ content: '' })
    render(InstructionEditor, {
      props: { scope: 'project', projectId: 'p1', variant: 'rail' },
    })

    const soul = await screen.findByRole('textbox', { name: 'SOUL' })
    expect(soul).toHaveAttribute(
      'placeholder',
      'Identity, values, and voice for this scope.',
    )

    await fireEvent.click(screen.getByRole('tab', { name: 'SYSTEM' }))
    const system = await screen.findByRole('textbox', { name: 'SYSTEM' })
    expect(system).toHaveAttribute(
      'placeholder',
      'How the agent should behave in this scope.',
    )

    await fireEvent.click(screen.getByRole('tab', { name: 'AGENTS' }))
    const agents = await screen.findByRole('textbox', { name: 'AGENTS' })
    expect(agents).toHaveAttribute(
      'placeholder',
      'Standing rules. Keep them short. Leave ## Memory pointing at [[memory/lessons]].',
    )
  })

  it('uses rail chrome without a page panel card', async () => {
    render(InstructionEditor, {
      props: { scope: 'project', projectId: 'p1', variant: 'rail' },
    })

    await screen.findByRole('textbox', { name: 'SOUL' })
    const root = document.querySelector('.instruction-editor')
    expect(root).toHaveClass('instruction-editor--rail')
    expect(root).not.toHaveClass('panel')
    expect(root).not.toHaveClass('panel--pad')
    expect(screen.queryByRole('heading', { name: 'Instructions' })).toBeNull()
  })
})
