// web/src/components/sessions/session-poller.ts
export function createSessionPoller<T>(
  load: () => Promise<T>,
  apply: (value: T) => void,
  intervalMs = 1500,
) {
  let active = false
  let queued = false
  let timer: ReturnType<typeof setInterval> | undefined
  let inFlight: Promise<void> | undefined

  async function poll() {
    queued = true
    if (active) return inFlight
    active = true
    inFlight = (async () => {
      try {
        while (queued) {
          queued = false
          apply(await load())
        }
      } finally {
        active = false
        inFlight = undefined
      }
    })()
    return inFlight
  }

  return {
    poll,
    start: () => {
      if (timer !== undefined) return
      void poll()
      timer = setInterval(() => {
        void poll()
      }, intervalMs)
    },
    stop: () => {
      if (timer !== undefined) clearInterval(timer)
      timer = undefined
      queued = false
    },
  }
}
