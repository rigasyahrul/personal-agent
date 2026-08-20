// web/src/components/sessions/OperationBadges.test.ts
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'
import OperationBadges from './OperationBadges.svelte'
import { BADGE_COPIES } from '../../lib/promote'

afterEach(cleanup)

describe('OperationBadges', () => {
  it('renders only the exact five safe badge copies', () => {
    for (const copy of BADGE_COPIES) {
      cleanup()
      render(OperationBadges, {
        props: {
          operations: ['op'],
          results: new Map([['op', { operation_id: 'op', badge: copy }]]),
        },
      })
      expect(screen.getByRole('status')).toHaveTextContent(copy)
    }
    cleanup()
    render(OperationBadges, {
      props: {
        operations: ['op'],
        results: new Map([['op', { operation_id: 'op', badge: '<img onerror=bad>' }]]),
      },
    })
    expect(screen.getByRole('status')).toHaveTextContent('Ready')
  })

  it('shows Promoting… until a result arrives and gates retry', async () => {
    const onretry = vi.fn()
    render(OperationBadges, {
      props: {
        operations: ['pending', 'failed'],
        results: new Map([
          [
            'failed',
            {
              operation_id: 'failed',
              badge: 'Cards failed — Retry cards',
              retry_cards: true,
              pending_id: 'p1',
            },
          ],
        ]),
        retrying: new Set(['p1']),
        onretry,
      },
    })
    expect(screen.getByText('Promoting…')).toBeInTheDocument()
    const button = screen.getByRole('button', { name: 'Retry cards' })
    expect(button).toBeDisabled()
  })

  it('invokes retry when enabled', async () => {
    const onretry = vi.fn()
    render(OperationBadges, {
      props: {
        operations: ['failed'],
        results: new Map([
          [
            'failed',
            {
              operation_id: 'failed',
              badge: 'Cards failed — Retry cards',
              retry_cards: true,
              pending_id: 'p1',
            },
          ],
        ]),
        onretry,
      },
    })
    await fireEvent.click(screen.getByRole('button', { name: 'Retry cards' }))
    expect(onretry).toHaveBeenCalledTimes(1)
  })
})
