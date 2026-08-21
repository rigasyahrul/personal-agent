// web/src/lib/format-session-date.test.ts
import { describe, expect, it } from 'vitest'
import { formatSessionDate } from './format-session-date'

describe('formatSessionDate', () => {
  it('formats ISO as Mon D (e.g. May 30)', () => {
    expect(formatSessionDate('2026-05-30T12:00:00.000Z')).toMatch(/^May 30$/)
  })

  it('returns null for missing or invalid', () => {
    expect(formatSessionDate(undefined)).toBeNull()
    expect(formatSessionDate('')).toBeNull()
    expect(formatSessionDate('not-a-date')).toBeNull()
  })
})
