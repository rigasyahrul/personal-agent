// web/src/components/sessions/SessionList.test.ts
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'
import SessionList from './SessionList.svelte'

afterEach(cleanup)

const sessions = [
  {
    id: 's1',
    title: 'Plan',
    status: 'idle',
    provider: 'openai',
    model_id: 'gpt',
  },
]

describe('SessionList', () => {
  it('emits open when a session is clicked', async () => {
    const onopen = vi.fn()
    render(SessionList, { props: { sessions, onopen } })
    await fireEvent.click(screen.getByRole('button', { name: /Plan/i }))
    expect(onopen).toHaveBeenCalledWith(sessions[0])
  })

  it('shows empty copy when there are no sessions', () => {
    render(SessionList, { props: { sessions: [] } })
    expect(screen.getByText(/no sessions yet/i)).toBeInTheDocument()
  })
})
