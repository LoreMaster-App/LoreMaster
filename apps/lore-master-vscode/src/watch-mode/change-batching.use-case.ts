/** Orders paths by code point, whatever the locale, as the engine does. */
function byCodePoint (a: string, b: string): number {
  if (a < b) {
    return -1
  }

  return a > b ? 1 : 0
}

/** The timer functions the batcher uses; a test supplies fakes. */
export interface Timers {
  setTimeout (callback: () => void, milliseconds: number): unknown
  clearTimeout (handle: unknown): void
  now (): number
}

const REAL_TIMERS: Timers = {
  setTimeout:   (callback, milliseconds) => setTimeout(callback, milliseconds),
  clearTimeout: handle => { clearTimeout(handle as NodeJS.Timeout) },
  now:          () => Date.now(),
}

export interface ChangeBatcherOptions {
  /** How long nothing may change before the pending changes are handled. */
  quietMs:      number
  /** The longest wait before a failed batch is tried again; the wait starts at `quietMs` and doubles. */
  maxBackoffMs: number
  /** Handles one batch (sorted, without repeats). Never runs twice at once. */
  run:          (changed: string[]) => Promise<void>
  /** Told when a batch failed and when it will be tried again. */
  onFailure?:   (error: unknown, retryInMs: number) => void
  timers?:      Timers
}

/**
 * Turns a stream of changed paths into batches: it waits until nothing has changed for
 * `quietMs`, runs the batch, and holds anything that changes meanwhile for the next one, so two
 * quick edits are one run and a run never overlaps another. A failed batch is kept (joined by
 * later changes) and tried again after a growing wait.
 */
export class ChangeBatcher {
  private readonly pending = new Set<string>()
  private readonly timers: Timers
  private timer:           unknown
  private running = false
  private failures = 0
  private retryAt = 0
  private disposed = false

  constructor (private readonly options: ChangeBatcherOptions) {
    this.timers = options.timers ?? REAL_TIMERS
  }

  private schedule (): void {
    if (this.running) {
      return
    }
    this.cancelTimer()
    const wait = Math.max(this.options.quietMs, this.retryAt - this.timers.now())
    this.timer = this.timers.setTimeout(() => { void this.flush() }, wait)
  }

  private cancelTimer (): void {
    if (this.timer === undefined) {
      return
    }

    this.timers.clearTimeout(this.timer)
    this.timer = undefined
  }

  private async flush (): Promise<void> {
    this.timer = undefined
    if (this.disposed || this.running || this.pending.size === 0) {
      return
    }
    const batch = [...this.pending].sort(byCodePoint)
    this.pending.clear()
    this.running = true
    try {
      await this.options.run(batch)
      this.failures = 0
      this.retryAt = 0
    } catch (error) {
      for (const path of batch) {
        this.pending.add(path)
      }
      this.failures++
      const wait = Math.min(this.options.quietMs * 2 ** (this.failures - 1), Math.max(this.options.maxBackoffMs, this.options.quietMs))
      this.retryAt = this.timers.now() + wait
      this.options.onFailure?.(error, wait)
    } finally {
      this.running = false
    }
    if (!this.disposed && this.pending.size > 0) {
      this.schedule()
    }
  }

  /** Notes that these paths changed. */
  add (paths: string[]): void {
    if (this.disposed || paths.length === 0) {
      return
    }
    for (const path of paths) {
      this.pending.add(path)
    }
    this.schedule()
  }

  /** Whether changes are waiting or a batch is running. */
  get busy (): boolean {
    return this.running || this.pending.size > 0
  }

  /** Stops: nothing more is run. */
  dispose (): void {
    this.disposed = true
    this.cancelTimer()
    this.pending.clear()
  }
}
