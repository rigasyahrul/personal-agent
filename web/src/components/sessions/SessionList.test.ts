// web/src/components/sessions/SessionList.test.ts
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SessionList from './SessionList.svelte'

afterEach(() => {
  cleanup()
  vi.useRealTimers()
})

const baseSession = {
  id: 's1',
  title: 'Plan',
  status: 'idle',
  provider: 'openai',
  model_id: 'gpt',
}

describe('SessionList', () => {
  it('shows empty copy when there are no sessions', () => {
    render(SessionList, { props: { sessions: [] } })
    expect(screen.getByText(/no sessions yet/i)).toBeInTheDocument()
  })

  it('emits open when a session card is clicked', async () => {
    const onopen = vi.fn()
    render(SessionList, { props: { sessions: [baseSession], onopen } })
    await fireEvent.click(screen.getByRole('button', { name: /Plan/i }))
    expect(onopen).toHaveBeenCalledWith(baseSession)
  })

  it('meta omits time when no timestamp', () => {
    render(SessionList, { props: { sessions: [baseSession] } })
    const meta = document.querySelector('.session-card__meta')
    expect(meta?.textContent).toBe('openai:gpt')
    expect(meta?.textContent).not.toMatch(/ago|just now/)
  })

  it('meta includes relative time preferring updated_at over created_at', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-20T12:00:00.000Z'))
    render(SessionList, {
      props: {
        sessions: [
          {
            ...baseSession,
            created_at: '2026-08-18T12:00:00.000Z',
            updated_at: '2026-08-20T11:55:00.000Z',
          },
        ],
      },
    })
    const meta = document.querySelector('.session-card__meta')
    expect(meta?.textContent).toContain('openai:gpt')
    expect(meta?.textContent).toContain('5m ago')
  })

  it('renders session rows as session-card elements', () => {
    render(SessionList, { props: { sessions: [baseSession] } })
    expect(document.querySelectorAll('.session-card')).toHaveLength(1)
    expect(screen.getByText('Plan')).toBeInTheDocument()
  })
})
