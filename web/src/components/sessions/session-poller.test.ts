// web/src/components/sessions/session-poller.test.ts
import { describe, expect, it, vi } from 'vitest'
import { createSessionPoller } from './session-poller'

const deferred = <T>() => {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((yes, no) => {
    resolve = yes
    reject = no
  })
  return { promise, resolve, reject }
}

describe('createSessionPoller', () => {
  it('serializes and coalesces overlapping polls', async () => {
    const pending: Array<ReturnType<typeof deferred<number>>> = []
    const applied: number[] = []
    const poller = createSessionPoller(
      () => {
        const item = deferred<number>()
        pending.push(item)
        return item.promise
      },
      (value) => {
        applied.push(value)
      },
      60_000,
    )

    const first = poller.poll()
    const second = poller.poll()
    expect(pending).toHaveLength(1)
    pending[0].resolve(1)
    await Promise.resolve()
    await Promise.resolve()
    expect(pending.length).toBe(2)
    pending[1].resolve(2)
    await Promise.all([first, second])
    expect(applied).toEqual([1, 2])
  })

  it('start installs one interval and stop clears it', async () => {
    vi.useFakeTimers()
    const load = vi.fn().mockResolvedValue('ok')
    const apply = vi.fn()
    const poller = createSessionPoller(load, apply, 1000)
    poller.start()
    poller.start()
    await Promise.resolve()
    expect(load).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(1000)
    expect(load).toHaveBeenCalledTimes(2)
    poller.stop()
    await vi.advanceTimersByTimeAsync(5000)
    expect(load).toHaveBeenCalledTimes(2)
    vi.useRealTimers()
  })
})
