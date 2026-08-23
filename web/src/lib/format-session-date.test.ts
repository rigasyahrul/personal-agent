// web/src/lib/format-session-date.test.ts
import { describe, expect, it } from 'vitest'
import { formatMessageDateTime, formatSessionDate } from './format-session-date'

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

describe('formatMessageDateTime', () => {
  it('formats ISO as Mon D, YYYY h:mm AM/PM in local time', () => {
    // Fixed offset-free local construction via UTC components so the string is stable:
    // use a known local Date and round-trip through ISO.
    const local = new Date(2026, 4, 30, 22, 36, 0) // May 30, 2026 10:36 PM local
    const label = formatMessageDateTime(local.toISOString())
    expect(label).toBe('May 30, 2026 10:36 PM')
  })

  it('returns null for missing or invalid', () => {
    expect(formatMessageDateTime(undefined)).toBeNull()
    expect(formatMessageDateTime('')).toBeNull()
    expect(formatMessageDateTime('not-a-date')).toBeNull()
  })
})
