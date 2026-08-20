import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { formatRelativeTime } from './format-relative-time'

describe('formatRelativeTime', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-20T12:00:00.000Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('returns null when iso is missing or invalid', () => {
    expect(formatRelativeTime(undefined)).toBeNull()
    expect(formatRelativeTime('')).toBeNull()
    expect(formatRelativeTime('not-a-date')).toBeNull()
  })

  it('formats just now, minutes, hours, and days ago', () => {
    expect(formatRelativeTime('2026-08-20T11:59:30.000Z')).toBe('just now')
    expect(formatRelativeTime('2026-08-20T11:55:00.000Z')).toBe('5m ago')
    expect(formatRelativeTime('2026-08-20T09:00:00.000Z')).toBe('3h ago')
    expect(formatRelativeTime('2026-08-18T12:00:00.000Z')).toBe('2d ago')
  })
})
