// web/src/components/notes/BacklinksPanel.test.ts
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'
import BacklinksPanel from './BacklinksPanel.svelte'

afterEach(cleanup)

const intro = {
  title: 'Intro',
  path: 'source/intro.md',
  knowledgeId: 'k-intro',
  kind: 'source',
}

const agents = {
  title: '',
  path: 'AGENTS.md',
  knowledgeId: 'k-agents',
}

describe('BacklinksPanel', () => {
  it('shows a quiet empty state when there are no backlinks', () => {
    render(BacklinksPanel, { props: { items: [], onopen: vi.fn() } })

    expect(screen.getByRole('heading', { name: 'Backlinks' })).toBeInTheDocument()
    expect(screen.getByText('No backlinks yet.')).toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
    expect(document.querySelector('.backlinks')).toBeTruthy()
  })

  it('lists titles and falls back to path when title is empty', () => {
    render(BacklinksPanel, {
      props: { items: [intro, agents], onopen: vi.fn() },
    })

    expect(screen.getByRole('button', { name: 'Intro' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'AGENTS.md' })).toBeInTheDocument()
    expect(screen.queryByText('No backlinks yet.')).not.toBeInTheDocument()
    expect(document.querySelectorAll('.backlinks__item')).toHaveLength(2)
  })

  it('opens the clicked backlink item', async () => {
    const onopen = vi.fn()
    render(BacklinksPanel, {
      props: { items: [intro, agents], onopen },
    })

    await fireEvent.click(screen.getByRole('button', { name: 'Intro' }))

    expect(onopen).toHaveBeenCalledOnce()
    expect(onopen).toHaveBeenCalledWith(intro)
  })
})
