// web/src/lib/session-title.test.ts
import { describe, expect, it } from 'vitest'
import { randomSessionTitle } from './session-title'

describe('randomSessionTitle', () => {
  it('returns two lowercase words separated by a space', () => {
    const title = randomSessionTitle()
    expect(title).toMatch(/^[a-z]+ [a-z]+$/)
    const [a, b] = title.split(' ')
    expect(a!.length).toBeGreaterThan(0)
    expect(b!.length).toBeGreaterThan(0)
  })

  it('does not use the first-message slice pattern', () => {
    const title = randomSessionTitle()
    expect(title).not.toMatch(/Plan the week/)
    expect(title.length).toBeLessThan(40)
  })

  it('produces varied titles across calls', () => {
    const set = new Set(Array.from({ length: 20 }, () => randomSessionTitle()))
    expect(set.size).toBeGreaterThan(1)
  })
})
