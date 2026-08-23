import { describe, expect, it } from 'vitest'
import { compoundTimeToFinishMs } from './compoundMetrics'

describe('compoundTimeToFinishMs', () => {
  it('returns milliseconds between created_at and finished_at ISO timestamps', () => {
    expect(compoundTimeToFinishMs('2026-08-22T12:00:00Z', '2026-08-22T12:00:01Z')).toBe(1000)
    expect(
      compoundTimeToFinishMs('2026-08-22T12:00:00.000Z', '2026-08-22T12:00:00.250Z'),
    ).toBe(250)
    expect(
      compoundTimeToFinishMs('2026-08-22T12:00:00.123456789Z', '2026-08-22T12:00:01.123456789Z'),
    ).toBe(1000)
    expect(
      compoundTimeToFinishMs('2026-08-22T12:00:00+00:00', '2026-08-22T12:00:05+00:00'),
    ).toBe(5000)
  })

  it('returns null when either timestamp is missing or invalid', () => {
    expect(compoundTimeToFinishMs(undefined, '2026-08-22T12:00:00Z')).toBeNull()
    expect(compoundTimeToFinishMs('2026-08-22T12:00:00Z', undefined)).toBeNull()
    expect(compoundTimeToFinishMs()).toBeNull()
    expect(compoundTimeToFinishMs('', '2026-08-22T12:00:00Z')).toBeNull()
    expect(compoundTimeToFinishMs('2026-08-22T12:00:00Z', '')).toBeNull()
    expect(compoundTimeToFinishMs('not-a-date', '2026-08-22T12:00:00Z')).toBeNull()
    expect(compoundTimeToFinishMs('2026-08-22T12:00:00Z', 'nope')).toBeNull()
  })
})
