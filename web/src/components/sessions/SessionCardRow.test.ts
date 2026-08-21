import { cleanup, fireEvent, render, screen } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'
import SessionCardRow from './SessionCardRow.svelte'

afterEach(cleanup)

describe('SessionCardRow', () => {
  it('renders title and meta', () => {
    render(SessionCardRow, {
      props: { title: 'Plan sprint', meta: 'openai:gpt-4o' },
    })
    expect(screen.getByText('Plan sprint')).toBeInTheDocument()
    expect(screen.getByText('openai:gpt-4o')).toBeInTheDocument()
    expect(document.querySelector('.session-card')).toBeTruthy()
    expect(document.querySelector('.session-card__title')).toBeTruthy()
    expect(document.querySelector('.session-card__meta')).toBeTruthy()
  })

  it('renders as a button and fires onclick when no href', async () => {
    const onclick = vi.fn()
    render(SessionCardRow, {
      props: { title: 'Plan', meta: 'openai:gpt', onclick },
    })
    const btn = screen.getByRole('button', { name: /plan/i })
    await fireEvent.click(btn)
    expect(onclick).toHaveBeenCalledOnce()
  })

  it('renders as a link when href is set', () => {
    render(SessionCardRow, {
      props: {
        title: 'Plan',
        meta: 'openai:gpt',
        href: '#/projects/p1/sessions',
      },
    })
    const link = screen.getByRole('link', { name: /plan/i })
    expect(link).toHaveAttribute('href', '#/projects/p1/sessions')
    expect(link).toHaveClass('session-card')
  })

  it('list variant: chat icon, title, date right, menu with Rename and Delete', async () => {
    const onclick = vi.fn()
    const onrename = vi.fn()
    const ondelete = vi.fn()
    render(SessionCardRow, {
      props: {
        title: 'Test 1',
        meta: '',
        variant: 'list',
        dateLabel: 'May 30',
        onclick,
        onrename,
        ondelete,
      },
    })

    expect(document.querySelector('.session-row')).toBeTruthy()
    expect(document.querySelector('.session-row__icon')).toBeTruthy()
    expect(screen.getByText('Test 1')).toBeInTheDocument()
    expect(screen.getByText('May 30')).toBeInTheDocument()

    const menuBtn = screen.getByRole('button', { name: /session actions/i })
    await fireEvent.click(menuBtn)
    expect(screen.getByRole('menuitem', { name: /^rename$/i })).toBeInTheDocument()
    expect(screen.getByRole('menuitem', { name: /^delete$/i })).toBeInTheDocument()

    await fireEvent.click(screen.getByRole('menuitem', { name: /^rename$/i }))
    expect(onrename).toHaveBeenCalledOnce()
    expect(onclick).not.toHaveBeenCalled()

    await fireEvent.click(menuBtn)
    await fireEvent.click(screen.getByRole('menuitem', { name: /^delete$/i }))
    expect(ondelete).toHaveBeenCalledOnce()
  })
})
