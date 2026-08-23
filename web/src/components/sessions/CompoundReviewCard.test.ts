// web/src/components/sessions/CompoundReviewCard.test.ts
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'
import CompoundReviewCard from './CompoundReviewCard.svelte'
import type { CompoundItem, CompoundProposal } from '../../lib/api/types'

afterEach(cleanup)

function sampleItem(overrides: Partial<CompoundItem> = {}): CompoundItem {
  return {
    kind: 'agents_patch',
    path: 'AGENTS.md',
    action: 'upsert',
    title: 'Memory pointer',
    content: 'original rule',
    content_sha256: 'abc123',
    ...overrides,
  }
}

function sampleProposal(overrides: Partial<CompoundProposal> = {}): CompoundProposal {
  return {
    id: 'prop-1',
    status: 'pending',
    created_at: '2026-08-22T00:00:00Z',
    items: [
      sampleItem(),
      sampleItem({
        kind: 'memory_detail',
        path: 'memory/20260822-1200-slug.md',
        action: 'create',
        title: 'Lesson',
        content: 'lesson body',
        content_sha256: 'def456',
      }),
    ],
    ...overrides,
  }
}

describe('CompoundReviewCard', () => {
  it('renders item kinds and paths', () => {
    render(CompoundReviewCard, {
      props: {
        proposal: sampleProposal(),
        onconfirm: vi.fn(),
      },
    })

    expect(screen.getByText('Compound review')).toBeInTheDocument()
    expect(screen.getByText('AGENTS.md')).toBeInTheDocument()
    expect(screen.getByText('memory/20260822-1200-slug.md')).toBeInTheDocument()
    expect(screen.getByText('agents_patch')).toBeInTheDocument()
    expect(screen.getByText('memory_detail')).toBeInTheDocument()
    expect(document.querySelector('.compound-card')).toBeTruthy()
    expect(document.querySelectorAll('.compound-item')).toHaveLength(2)
  })

  it('approve callback receives locally edited item content, not the original snapshot', async () => {
    const proposal = sampleProposal()
    const onconfirm = vi.fn()
    render(CompoundReviewCard, {
      props: { proposal, onconfirm },
    })

    const textarea = screen.getByLabelText('Content for AGENTS.md') as HTMLTextAreaElement
    expect(textarea).toHaveValue('original rule')
    await fireEvent.input(textarea, { target: { value: 'edited standing rule' } })

    await fireEvent.click(screen.getByRole('button', { name: 'Approve' }))

    expect(onconfirm).toHaveBeenCalledOnce()
    expect(onconfirm).toHaveBeenCalledWith('approve', [
      expect.objectContaining({
        kind: 'agents_patch',
        path: 'AGENTS.md',
        content: 'edited standing rule',
      }),
      expect.objectContaining({
        path: 'memory/20260822-1200-slug.md',
        content: 'lesson body',
      }),
    ])
    expect(proposal.items[0].content).toBe('original rule')
  })

  it('reject callback receives current local items', async () => {
    const onconfirm = vi.fn()
    render(CompoundReviewCard, {
      props: { proposal: sampleProposal(), onconfirm },
    })

    await fireEvent.click(screen.getByRole('button', { name: 'Reject' }))

    expect(onconfirm).toHaveBeenCalledOnce()
    expect(onconfirm).toHaveBeenCalledWith(
      'reject',
      expect.arrayContaining([
        expect.objectContaining({ path: 'AGENTS.md', content: 'original rule' }),
      ]),
    )
  })

  it('busy disables approve, reject, cancel, and item editors', () => {
    render(CompoundReviewCard, {
      props: {
        proposal: sampleProposal(),
        onconfirm: vi.fn(),
        oncancel: vi.fn(),
        busy: true,
      },
    })

    expect(screen.getByRole('button', { name: 'Approve' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Reject' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeDisabled()
    expect(screen.getByLabelText('Content for AGENTS.md')).toBeDisabled()
    expect(screen.getByLabelText('Content for memory/20260822-1200-slug.md')).toBeDisabled()
  })

  it('shows empty chrome with reject still available and no placeholder items', () => {
    render(CompoundReviewCard, {
      props: {
        proposal: sampleProposal({ items: [] }),
        onconfirm: vi.fn(),
      },
    })

    expect(screen.getByText('Compound review')).toBeInTheDocument()
    expect(screen.getByText('No items')).toBeInTheDocument()
    expect(document.querySelector('.compound-item')).toBeNull()
    expect(screen.queryByRole('textbox')).toBeNull()
    expect(screen.getByRole('button', { name: 'Reject' })).toBeEnabled()
    expect(screen.queryByRole('button', { name: 'Cancel' })).toBeNull()
  })

  it('calls oncancel only when provided', async () => {
    const oncancel = vi.fn()
    const { unmount } = render(CompoundReviewCard, {
      props: {
        proposal: sampleProposal(),
        onconfirm: vi.fn(),
        oncancel,
      },
    })

    await fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(oncancel).toHaveBeenCalledOnce()
    unmount()
  })
})
